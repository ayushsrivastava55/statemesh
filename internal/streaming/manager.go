package streaming

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/cosmos/state-mesh/internal/config"
	"github.com/cosmos/state-mesh/pkg/types"
	"go.uber.org/zap"
)

// Manager handles streaming operations
type Manager struct {
	producer *kafka.Producer
	topic    string
	logger   *zap.Logger
}

// NewManager creates a new streaming manager
func NewManager(cfg config.StreamingConfig, logger *zap.Logger) (*Manager, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("streaming is disabled")
	}

	// Configure Kafka producer
	configMap := &kafka.ConfigMap{
		"bootstrap.servers": cfg.Kafka.Brokers[0], // Use first broker for simplicity
		"client.id":         "state-mesh-producer",
		"acks":             "all",
		"retries":          3,
		"batch.size":       16384,
		"linger.ms":        10,
		"compression.type": "snappy",
	}

	producer, err := kafka.NewProducer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &Manager{
		producer: producer,
		topic:    cfg.Kafka.Topic,
		logger:   logger.Named("streaming"),
	}, nil
}

// Close closes the streaming manager
func (m *Manager) Close() error {
	if m.producer != nil {
		m.producer.Close()
	}
	return nil
}

// PublishStateChange publishes a state change event
func (m *Manager) PublishStateChange(ctx context.Context, change *types.StateChange) error {
	data, err := json.Marshal(change)
	if err != nil {
		return fmt.Errorf("failed to marshal state change: %w", err)
	}

	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &m.topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(fmt.Sprintf("%s:%s", change.ChainName, change.StoreKey)),
		Value: data,
		Headers: []kafka.Header{
			{Key: "chain", Value: []byte(change.ChainName)},
			{Key: "store", Value: []byte(change.StoreKey)},
			{Key: "height", Value: []byte(fmt.Sprintf("%d", change.Height))},
		},
	}

	deliveryChan := make(chan kafka.Event)
	err = m.producer.Produce(message, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery confirmation
	select {
	case e := <-deliveryChan:
		if msg, ok := e.(*kafka.Message); ok {
			if msg.TopicPartition.Error != nil {
				return fmt.Errorf("delivery failed: %w", msg.TopicPartition.Error)
			}
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// PublishBalanceEvent publishes a balance change event
func (m *Manager) PublishBalanceEvent(ctx context.Context, event *types.BalanceEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal balance event: %w", err)
	}

	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &m.topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(fmt.Sprintf("%s:balance:%s:%s", event.ChainName, event.Address, event.Denom)),
		Value: data,
		Headers: []kafka.Header{
			{Key: "chain", Value: []byte(event.ChainName)},
			{Key: "type", Value: []byte("balance")},
			{Key: "address", Value: []byte(event.Address)},
			{Key: "denom", Value: []byte(event.Denom)},
		},
	}

	return m.produceMessage(ctx, message)
}

// PublishDelegationEvent publishes a delegation change event
func (m *Manager) PublishDelegationEvent(ctx context.Context, event *types.DelegationEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal delegation event: %w", err)
	}

	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &m.topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(fmt.Sprintf("%s:delegation:%s:%s", event.ChainName, event.DelegatorAddress, event.ValidatorAddress)),
		Value: data,
		Headers: []kafka.Header{
			{Key: "chain", Value: []byte(event.ChainName)},
			{Key: "type", Value: []byte("delegation")},
			{Key: "delegator", Value: []byte(event.DelegatorAddress)},
			{Key: "validator", Value: []byte(event.ValidatorAddress)},
		},
	}

	return m.produceMessage(ctx, message)
}

// produceMessage is a helper method to produce a message with delivery confirmation
func (m *Manager) produceMessage(ctx context.Context, message *kafka.Message) error {
	deliveryChan := make(chan kafka.Event)
	err := m.producer.Produce(message, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery confirmation
	select {
	case e := <-deliveryChan:
		if msg, ok := e.(*kafka.Message); ok {
			if msg.TopicPartition.Error != nil {
				return fmt.Errorf("delivery failed: %w", msg.TopicPartition.Error)
			}
			m.logger.Debug("Message delivered",
				zap.String("topic", *msg.TopicPartition.Topic),
				zap.Int32("partition", msg.TopicPartition.Partition),
				zap.Int64("offset", int64(msg.TopicPartition.Offset)))
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// Flush flushes any pending messages
func (m *Manager) Flush(timeoutMs int) error {
	remaining := m.producer.Flush(timeoutMs)
	if remaining > 0 {
		return fmt.Errorf("failed to flush %d messages within timeout", remaining)
	}
	return nil
}
