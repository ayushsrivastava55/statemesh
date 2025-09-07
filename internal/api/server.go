package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cosmos/state-mesh/internal/config"
	"github.com/cosmos/state-mesh/internal/graphql"
	"github.com/cosmos/state-mesh/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Server represents the API server
type Server struct {
	cfg           config.APIConfig
	storage       *storage.Manager
	logger        *zap.Logger
	graphqlServer *http.Server
	restServer    *http.Server
	metricsServer *http.Server
}

// NewServer creates a new API server
func NewServer(cfg config.APIConfig, storage *storage.Manager, logger *zap.Logger) (*Server, error) {
	return &Server{
		cfg:     cfg,
		storage: storage,
		logger:  logger.Named("api"),
	}, nil
}

// StartGraphQL starts the GraphQL server
func (s *Server) StartGraphQL(ctx context.Context) error {
	// Initialize GraphQL handler
	graphqlHandler, err := s.setupGraphQLHandler()
	if err != nil {
		return fmt.Errorf("failed to setup GraphQL handler: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/graphql", graphqlHandler)
	
	if s.cfg.GraphQL.Playground {
		playgroundHandler := s.setupPlaygroundHandler()
		mux.Handle("/playground", playgroundHandler)
	}

	// Health check endpoint
	mux.HandleFunc("/health", s.healthHandler)

	s.graphqlServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.GraphQL.Port),
		Handler: s.corsMiddleware(mux),
	}

	s.logger.Info("GraphQL server starting", zap.Int("port", s.cfg.GraphQL.Port))

	if err := s.graphqlServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("GraphQL server error: %w", err)
	}

	return nil
}

// StartREST starts the REST server
func (s *Server) StartREST(ctx context.Context) error {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(s.ginLogger())

	if s.cfg.CORS.Enabled {
		router.Use(s.ginCORS())
	}

	// Setup REST routes
	s.setupRESTRoutes(router)

	s.restServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.REST.Port),
		Handler: router,
	}

	s.logger.Info("REST server starting", zap.Int("port", s.cfg.REST.Port))

	if err := s.restServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("REST server error: %w", err)
	}

	return nil
}

// StartMetrics starts the metrics server
func (s *Server) StartMetrics(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", s.healthHandler)

	s.metricsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Metrics.Port),
		Handler: mux,
	}

	s.logger.Info("Metrics server starting", zap.Int("port", s.cfg.Metrics.Port))

	if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("metrics server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down all servers
func (s *Server) Shutdown(ctx context.Context) error {
	var errs []error

	if s.graphqlServer != nil {
		if err := s.graphqlServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("GraphQL server shutdown error: %w", err))
		}
	}

	if s.restServer != nil {
		if err := s.restServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("REST server shutdown error: %w", err))
		}
	}

	if s.metricsServer != nil {
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("metrics server shutdown error: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("server shutdown errors: %v", errs)
	}

	return nil
}

// setupGraphQLHandler sets up the GraphQL handler
func (s *Server) setupGraphQLHandler() (http.Handler, error) {
	// Import the GraphQL resolver
	resolver := graphql.NewResolver(s.storage, s.logger)
	
	// For now, return a simple handler that shows the schema is ready
	// In a production setup, this would use the generated gqlgen handler
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"message": "GraphQL resolver ready - use gqlgen to generate full handler"}}`))
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Only POST method allowed"}`))
		}
	}), nil
}

// setupPlaygroundHandler sets up the GraphQL playground handler
func (s *Server) setupPlaygroundHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>GraphQL Playground</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
    <link rel="shortcut icon" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
    <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
    <div id="root">
        <style>
            body { margin: 0; font-family: Open Sans, sans-serif; overflow: hidden; }
            #root { height: 100vh; }
        </style>
    </div>
    <script>
        window.addEventListener('load', function (event) {
            GraphQLPlayground.init(document.getElementById('root'), {
                endpoint: '/graphql'
            })
        })
    </script>
</body>
</html>
		`))
	})
}

// setupRESTRoutes sets up REST API routes
func (s *Server) setupRESTRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")

	// Health check
	api.GET("/health", s.ginHealthHandler)

	// Account routes
	accounts := api.Group("/accounts")
	{
		accounts.GET("/:address/balances", s.getAccountBalances)
		accounts.GET("/:address/delegations", s.getAccountDelegations)
		accounts.GET("/:address/state", s.getAccountState)
	}

	// Chain routes
	chains := api.Group("/chains")
	{
		chains.GET("/", s.getChains)
		chains.GET("/:chain/validators", s.getValidators)
		chains.GET("/:chain/stats", s.getChainStats)
	}

	// Cross-chain routes
	crosschain := api.Group("/cross-chain")
	{
		crosschain.GET("/accounts/:address", s.getCrossChainAccount)
		crosschain.GET("/validators", s.getCrossChainValidators)
	}

	// Governance routes
	gov := api.Group("/governance")
	{
		gov.GET("/proposals", s.getProposals)
		gov.GET("/proposals/:id", s.getProposal)
		gov.GET("/proposals/:id/votes", s.getProposalVotes)
	}
}

// healthHandler handles health check requests
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check database connectivity
	if err := s.storage.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status": "unhealthy", "error": "database connection failed"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy"}`))
}

// ginHealthHandler handles health check requests for Gin
func (s *Server) ginHealthHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Check database connectivity
	if err := s.storage.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "database connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.CORS.Enabled {
			origin := "*"
			if len(s.cfg.CORS.Origins) > 0 && s.cfg.CORS.Origins[0] != "*" {
				origin = s.cfg.CORS.Origins[0]
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// ginCORS adds CORS middleware for Gin
func (s *Server) ginCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := "*"
		if len(s.cfg.CORS.Origins) > 0 && s.cfg.CORS.Origins[0] != "*" {
			origin = s.cfg.CORS.Origins[0]
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// ginLogger creates a Gin logger middleware
func (s *Server) ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		s.logger.Info("HTTP request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
		)
	}
}
