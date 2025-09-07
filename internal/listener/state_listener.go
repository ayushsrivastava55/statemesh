package listener

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cosmos/state-mesh/internal/config"
	"github.com/cosmos/state-mesh/internal/storage"
	"github.com/cosmos/state-mesh/internal/streaming"
	"github.com/cosmos/state-mesh/pkg/types"
	"go.uber.org/zap"
)

// StateListener implements ADR-038 State Listening for real-time state ingestion
type StateListener struct {
	cfg       config.Config
	storage   *storage.Manager
	streaming *streaming.Manager
	logger    *zap.Logger
	
	// State change channels
	stateChanges chan *StateChange
	
	// Worker management
	workers    map[string]*ListenerWorker
	workersMux sync.RWMutex
	
	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// StateChange represents a state change event from ADR-038
type StateChange struct {
	ChainName string
	StoreKey  string
	Key       []byte
	Value     []byte
	Delete    bool
	Height    int64
	Timestamp time.Time
}

// ListenerWorker handles state changes for a specific chain
type ListenerWorker struct {
	chainName string
	cfg       config.ChainConfig
	storage   *storage.Manager
	streaming *streaming.Manager
	logger    *zap.Logger
	
	// State change processing
	changes chan *StateChange
	
	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewStateListener creates a new state listener
func NewStateListener(cfg config.Config, storage *storage.Manager, streaming *streaming.Manager, logger *zap.Logger) *StateListener {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &StateListener{
		cfg:          cfg,
		storage:      storage,
		streaming:    streaming,
		logger:       logger.Named("state_listener"),
		stateChanges: make(chan *StateChange, 10000), // Buffer for high throughput
		workers:      make(map[string]*ListenerWorker),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start starts the state listener
func (sl *StateListener) Start(ctx context.Context) error {
	sl.logger.Info("Starting State Listener")
	
	// Start workers for each enabled chain
	for _, chain := range sl.cfg.Chains {
		if !chain.Enabled {
			continue
		}
		
		worker := sl.createWorker(chain)
		sl.workersMux.Lock()
		sl.workers[chain.Name] = worker
		sl.workersMux.Unlock()
		
		sl.wg.Add(1)
		go func(w *ListenerWorker) {
			defer sl.wg.Done()
			if err := w.start(sl.ctx); err != nil {
				sl.logger.Error("Worker failed",
					zap.String("chain", w.chainName),
					zap.Error(err))
			}
		}(worker)
	}
	
	// Start main state change processor
	sl.wg.Add(1)
	go func() {
		defer sl.wg.Done()
		sl.processStateChanges()
	}()
	
	sl.logger.Info("State Listener started")
	return nil
}

// Stop stops the state listener
func (sl *StateListener) Stop() error {
	sl.logger.Info("Stopping State Listener")
	
	sl.cancel()
	sl.wg.Wait()
	
	// Close channels
	close(sl.stateChanges)
	
	sl.logger.Info("State Listener stopped")
	return nil
}

// OnStateChange handles incoming state changes from ADR-038
func (sl *StateListener) OnStateChange(chainName, storeKey string, key, value []byte, delete bool, height int64) {
	change := &StateChange{
		ChainName: chainName,
		StoreKey:  storeKey,
		Key:       key,
		Value:     value,
		Delete:    delete,
		Height:    height,
		Timestamp: time.Now(),
	}
	
	select {
	case sl.stateChanges <- change:
		// Successfully queued
	default:
		// Channel full, log warning
		sl.logger.Warn("State change channel full, dropping change",
			zap.String("chain", chainName),
			zap.String("store", storeKey),
			zap.Int64("height", height))
	}
}

// processStateChanges processes incoming state changes
func (sl *StateListener) processStateChanges() {
	sl.logger.Info("Starting state change processor")
	
	for {
		select {
		case <-sl.ctx.Done():
			sl.logger.Info("State change processor stopping")
			return
		case change := <-sl.stateChanges:
			if change == nil {
				continue
			}
			
			// Route to appropriate worker
			sl.workersMux.RLock()
			worker, exists := sl.workers[change.ChainName]
			sl.workersMux.RUnlock()
			
			if !exists {
				sl.logger.Warn("No worker for chain",
					zap.String("chain", change.ChainName))
				continue
			}
			
			// Send to worker
			select {
			case worker.changes <- change:
				// Successfully routed
			default:
				sl.logger.Warn("Worker channel full",
					zap.String("chain", change.ChainName))
			}
		}
	}
}

// createWorker creates a new listener worker for a chain
func (sl *StateListener) createWorker(chainCfg config.ChainConfig) *ListenerWorker {
	ctx, cancel := context.WithCancel(sl.ctx)
	
	return &ListenerWorker{
		chainName: chainCfg.Name,
		cfg:       chainCfg,
		storage:   sl.storage,
		streaming: sl.streaming,
		logger:    sl.logger.Named(chainCfg.Name),
		changes:   make(chan *StateChange, 1000),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// start starts the listener worker
func (lw *ListenerWorker) start(ctx context.Context) error {
	lw.logger.Info("Starting listener worker")
	
	for {
		select {
		case <-ctx.Done():
			lw.logger.Info("Listener worker stopping")
			return nil
		case change := <-lw.changes:
			if change == nil {
				continue
			}
			
			if err := lw.processStateChange(change); err != nil {
				lw.logger.Error("Failed to process state change",
					zap.String("store", change.StoreKey),
					zap.Int64("height", change.Height),
					zap.Error(err))
			}
		}
	}
}

// processStateChange processes a single state change
func (lw *ListenerWorker) processStateChange(change *StateChange) error {
	lw.logger.Debug("Processing state change",
		zap.String("store", change.StoreKey),
		zap.Int("key_len", len(change.Key)),
		zap.Int("value_len", len(change.Value)),
		zap.Bool("delete", change.Delete),
		zap.Int64("height", change.Height))
	
	// Parse the state change based on store key
	switch change.StoreKey {
	case "bank":
		return lw.processBankStateChange(change)
	case "staking":
		return lw.processStakingStateChange(change)
	case "distribution":
		return lw.processDistributionStateChange(change)
	case "gov":
		return lw.processGovernanceStateChange(change)
	case "mint":
		return lw.processMintStateChange(change)
	case "slashing":
		return lw.processSlashingStateChange(change)
	default:
		lw.logger.Debug("Unknown store key", zap.String("store", change.StoreKey))
		return nil
	}
}

// processBankStateChange processes bank module state changes
func (lw *ListenerWorker) processBankStateChange(change *StateChange) error {
	// Parse bank state change
	// Key format: balances/{address}/{denom} or supply/{denom}
	keyStr := string(change.Key)
	
	if len(keyStr) > 9 && keyStr[:9] == "balances/" {
		// Balance change
		return lw.processBalanceChange(change, keyStr[9:])
	} else if len(keyStr) > 7 && keyStr[:7] == "supply/" {
		// Supply change
		return lw.processSupplyChange(change, keyStr[7:])
	}
	
	return nil
}

// processBalanceChange processes balance changes
func (lw *ListenerWorker) processBalanceChange(change *StateChange, keyRemainder string) error {
	// Parse address and denom from key
	// Format: {address}/{denom}
	parts := []string{} // TODO: Parse key properly
	if len(parts) < 2 {
		return fmt.Errorf("invalid balance key format")
	}
	
	address := parts[0]
	denom := parts[1]
	
	// Parse amount from value
	amount := string(change.Value)
	if change.Delete {
		amount = "0"
	}
	
	// Create balance event
	balanceEvent := types.BalanceEvent{
		ChainName: change.ChainName,
		Address:   address,
		Denom:     denom,
		Amount:    amount,
		EventType: "balance_change",
		Height:    change.Height,
		Timestamp: change.Timestamp,
	}
	
	// Store in database
	balance := types.Balance{
		ChainName: change.ChainName,
		Address:   address,
		Denom:     denom,
		Amount:    amount,
		Height:    change.Height,
		UpdatedAt: change.Timestamp,
	}
	
	if err := lw.storage.Postgres().UpsertBalance(context.Background(), balance); err != nil {
		return fmt.Errorf("failed to upsert balance: %w", err)
	}
	
	// Stream event
	if lw.streaming != nil {
		if err := lw.streaming.PublishBalanceEvent(balanceEvent); err != nil {
			lw.logger.Warn("Failed to publish balance event", zap.Error(err))
		}
	}
	
	// Store in ClickHouse for analytics
	if lw.storage.ClickHouse() != nil {
		if err := lw.storage.ClickHouse().InsertBalanceEvent(context.Background(), balanceEvent); err != nil {
			lw.logger.Warn("Failed to insert balance event to ClickHouse", zap.Error(err))
		}
	}
	
	return nil
}

// processSupplyChange processes supply changes
func (lw *ListenerWorker) processSupplyChange(change *StateChange, denom string) error {
	// TODO: Implement supply change processing
	lw.logger.Debug("Supply change detected",
		zap.String("denom", denom),
		zap.Int64("height", change.Height))
	return nil
}

// processStakingStateChange processes staking module state changes
func (lw *ListenerWorker) processStakingStateChange(change *StateChange) error {
	// Parse staking state change
	// Key formats: validators/{validator}, delegations/{delegator}/{validator}, etc.
	keyStr := string(change.Key)
	
	if len(keyStr) > 11 && keyStr[:11] == "validators/" {
		// Validator change
		return lw.processValidatorChange(change, keyStr[11:])
	} else if len(keyStr) > 12 && keyStr[:12] == "delegations/" {
		// Delegation change
		return lw.processDelegationChange(change, keyStr[12:])
	}
	
	return nil
}

// processValidatorChange processes validator changes
func (lw *ListenerWorker) processValidatorChange(change *StateChange, validatorAddr string) error {
	// TODO: Parse validator data from protobuf value
	lw.logger.Debug("Validator change detected",
		zap.String("validator", validatorAddr),
		zap.Int64("height", change.Height))
	return nil
}

// processDelegationChange processes delegation changes
func (lw *ListenerWorker) processDelegationChange(change *StateChange, keyRemainder string) error {
	// TODO: Parse delegation data from protobuf value
	lw.logger.Debug("Delegation change detected",
		zap.String("key", keyRemainder),
		zap.Int64("height", change.Height))
	return nil
}

// processDistributionStateChange processes distribution module state changes
func (lw *ListenerWorker) processDistributionStateChange(change *StateChange) error {
	// TODO: Implement distribution state change processing
	lw.logger.Debug("Distribution state change",
		zap.String("key", string(change.Key)),
		zap.Int64("height", change.Height))
	return nil
}

// processGovernanceStateChange processes governance module state changes
func (lw *ListenerWorker) processGovernanceStateChange(change *StateChange) error {
	// TODO: Implement governance state change processing
	lw.logger.Debug("Governance state change",
		zap.String("key", string(change.Key)),
		zap.Int64("height", change.Height))
	return nil
}

// processMintStateChange processes mint module state changes
func (lw *ListenerWorker) processMintStateChange(change *StateChange) error {
	// TODO: Implement mint state change processing
	lw.logger.Debug("Mint state change",
		zap.String("key", string(change.Key)),
		zap.Int64("height", change.Height))
	return nil
}

// processSlashingStateChange processes slashing module state changes
func (lw *ListenerWorker) processSlashingStateChange(change *StateChange) error {
	// TODO: Implement slashing state change processing
	lw.logger.Debug("Slashing state change",
		zap.String("key", string(change.Key)),
		zap.Int64("height", change.Height))
	return nil
}
