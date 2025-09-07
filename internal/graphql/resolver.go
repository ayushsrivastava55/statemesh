package graphql

import (
	"github.com/cosmos/state-mesh/internal/storage"
	"go.uber.org/zap"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{
	storage *storage.Manager
	logger  *zap.Logger
}

// NewResolver creates a new GraphQL resolver with dependencies
func NewResolver(storage *storage.Manager, logger *zap.Logger) *Resolver {
	return &Resolver{
		storage: storage,
		logger:  logger,
	}
}
