package api

import (
	"net/http"
	"strconv"

	"github.com/cosmos/state-mesh/pkg/types"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// getAccountBalances handles GET /api/v1/accounts/:address/balances
func (s *Server) getAccountBalances(c *gin.Context) {
	address := c.Param("address")
	chainName := c.Query("chain")

	if chainName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "chain parameter is required",
		})
		return
	}

	balances, err := s.storage.Postgres().GetBalances(c.Request.Context(), chainName, address)
	if err != nil {
		s.logger.Error("Failed to get balances", 
			zap.String("address", address),
			zap.String("chain", chainName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get balances",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chain":    chainName,
		"address":  address,
		"balances": balances,
	})
}

// getAccountDelegations handles GET /api/v1/accounts/:address/delegations
func (s *Server) getAccountDelegations(c *gin.Context) {
	address := c.Param("address")
	chainName := c.Query("chain")

	if chainName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "chain parameter is required",
		})
		return
	}

	delegations, err := s.storage.Postgres().GetDelegations(c.Request.Context(), chainName, address)
	if err != nil {
		s.logger.Error("Failed to get delegations",
			zap.String("address", address),
			zap.String("chain", chainName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get delegations",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chain":       chainName,
		"address":     address,
		"delegations": delegations,
	})
}

// getAccountState handles GET /api/v1/accounts/:address/state
func (s *Server) getAccountState(c *gin.Context) {
	address := c.Param("address")
	chainName := c.Query("chain")

	if chainName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "chain parameter is required",
		})
		return
	}

	// Get balances
	balances, err := s.storage.Postgres().GetBalances(c.Request.Context(), chainName, address)
	if err != nil {
		s.logger.Error("Failed to get balances for account state",
			zap.String("address", address),
			zap.String("chain", chainName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get account state",
		})
		return
	}

	// Get delegations
	delegations, err := s.storage.Postgres().GetDelegations(c.Request.Context(), chainName, address)
	if err != nil {
		s.logger.Error("Failed to get delegations for account state",
			zap.String("address", address),
			zap.String("chain", chainName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get account state",
		})
		return
	}

	// Create unified account state
	accountState := types.AccountState{
		ChainName:   chainName,
		Address:     address,
		Balances:    balances,
		Delegations: delegations,
		// TODO: Add unbonding, redelegations, rewards when implemented
	}

	c.JSON(http.StatusOK, accountState)
}

// getChains handles GET /api/v1/chains
func (s *Server) getChains(c *gin.Context) {
	// For now, return hardcoded chain info
	// In a real implementation, this would come from the database
	chains := []types.ChainInfo{
		{
			Name:    "cosmoshub",
			ChainID: "cosmoshub-4",
			Status:  "active",
		},
		{
			Name:    "osmosis",
			ChainID: "osmosis-1",
			Status:  "active",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"chains": chains,
	})
}

// getValidators handles GET /api/v1/chains/:chain/validators
func (s *Server) getValidators(c *gin.Context) {
	chainName := c.Param("chain")

	validators, err := s.storage.Postgres().GetValidators(c.Request.Context(), chainName)
	if err != nil {
		s.logger.Error("Failed to get validators",
			zap.String("chain", chainName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get validators",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chain":      chainName,
		"validators": validators,
	})
}

// getChainStats handles GET /api/v1/chains/:chain/stats
func (s *Server) getChainStats(c *gin.Context) {
	chainName := c.Param("chain")

	// Try to get stats from ClickHouse if available
	if s.storage.ClickHouse() != nil {
		stats, err := s.storage.ClickHouse().GetChainStats(c.Request.Context(), chainName)
		if err == nil {
			c.JSON(http.StatusOK, stats)
			return
		}
		s.logger.Warn("Failed to get chain stats from ClickHouse, falling back",
			zap.String("chain", chainName),
			zap.Error(err))
	}

	// Fallback: basic stats from PostgreSQL
	validators, err := s.storage.Postgres().GetValidators(c.Request.Context(), chainName)
	if err != nil {
		s.logger.Error("Failed to get validators for stats",
			zap.String("chain", chainName),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get chain stats",
		})
		return
	}

	stats := types.ChainStats{
		ChainName:       chainName,
		TotalValidators: int64(len(validators)),
		// TODO: Calculate other stats from PostgreSQL
	}

	c.JSON(http.StatusOK, stats)
}

// getCrossChainAccount handles GET /api/v1/cross-chain/accounts/:address
func (s *Server) getCrossChainAccount(c *gin.Context) {
	address := c.Param("address")

	// For now, return a placeholder response
	// In a real implementation, this would aggregate data across all chains
	crossChainState := types.CrossChainAccountState{
		Address: address,
		Chains:  make(map[string]types.AccountState),
		Totals: types.CrossChainTotals{
			TotalBalance:   make(map[string]string),
			TotalDelegated: make(map[string]string),
			TotalUnbonding: make(map[string]string),
			TotalRewards:   make(map[string]string),
		},
	}

	// TODO: Implement cross-chain aggregation logic

	c.JSON(http.StatusOK, crossChainState)
}

// getCrossChainValidators handles GET /api/v1/cross-chain/validators
func (s *Server) getCrossChainValidators(c *gin.Context) {
	chains := c.QueryArray("chains")
	if len(chains) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "at least one chain must be specified",
		})
		return
	}

	allValidators := make(map[string][]types.Validator)

	for _, chainName := range chains {
		validators, err := s.storage.Postgres().GetValidators(c.Request.Context(), chainName)
		if err != nil {
			s.logger.Error("Failed to get validators for cross-chain query",
				zap.String("chain", chainName),
				zap.Error(err))
			continue
		}
		allValidators[chainName] = validators
	}

	c.JSON(http.StatusOK, gin.H{
		"validators": allValidators,
	})
}

// getProposals handles GET /api/v1/governance/proposals
func (s *Server) getProposals(c *gin.Context) {
	chainName := c.Query("chain")
	if chainName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "chain parameter is required",
		})
		return
	}

	// TODO: Implement proposal queries
	c.JSON(http.StatusOK, gin.H{
		"chain":     chainName,
		"proposals": []types.Proposal{},
	})
}

// getProposal handles GET /api/v1/governance/proposals/:id
func (s *Server) getProposal(c *gin.Context) {
	chainName := c.Query("chain")
	if chainName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "chain parameter is required",
		})
		return
	}

	proposalIDStr := c.Param("id")
	proposalID, err := strconv.ParseUint(proposalIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid proposal ID",
		})
		return
	}

	// TODO: Implement single proposal query
	c.JSON(http.StatusOK, gin.H{
		"chain":       chainName,
		"proposal_id": proposalID,
		"proposal":    nil,
	})
}

// getProposalVotes handles GET /api/v1/governance/proposals/:id/votes
func (s *Server) getProposalVotes(c *gin.Context) {
	chainName := c.Query("chain")
	if chainName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "chain parameter is required",
		})
		return
	}

	proposalIDStr := c.Param("id")
	proposalID, err := strconv.ParseUint(proposalIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid proposal ID",
		})
		return
	}

	// TODO: Implement proposal votes query
	c.JSON(http.StatusOK, gin.H{
		"chain":       chainName,
		"proposal_id": proposalID,
		"votes":       []types.Vote{},
	})
}
