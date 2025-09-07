package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete application configuration
type Config struct {
	Chains    []ChainConfig    `mapstructure:"chains"`
	Database  DatabaseConfig   `mapstructure:"database"`
	Streaming StreamingConfig  `mapstructure:"streaming"`
	API       APIConfig        `mapstructure:"api"`
	Ingester  IngesterConfig   `mapstructure:"ingester"`
	Log       LogConfig        `mapstructure:"log"`
}

// ChainConfig represents configuration for a single Cosmos SDK chain
type ChainConfig struct {
	Name         string   `mapstructure:"name"`
	ChainID      string   `mapstructure:"chain_id"`
	GRPCEndpoint string   `mapstructure:"grpc_endpoint"`
	RESTEndpoint string   `mapstructure:"rest_endpoint"`
	Modules      []string `mapstructure:"modules"`
	Enabled      bool     `mapstructure:"enabled"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Postgres   PostgresConfig   `mapstructure:"postgres"`
	ClickHouse ClickHouseConfig `mapstructure:"clickhouse"`
}

// PostgresConfig represents PostgreSQL configuration
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"ssl_mode"`
	MaxConns int    `mapstructure:"max_conns"`
	MinConns int    `mapstructure:"min_conns"`
}

// ClickHouseConfig represents ClickHouse configuration
type ClickHouseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Enabled  bool   `mapstructure:"enabled"`
}

// StreamingConfig represents streaming configuration
type StreamingConfig struct {
	Enabled bool        `mapstructure:"enabled"`
	Kafka   KafkaConfig `mapstructure:"kafka"`
}

// KafkaConfig represents Kafka configuration
type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Topic   string   `mapstructure:"topic"`
}

// APIConfig represents API server configuration
type APIConfig struct {
	GraphQL GraphQLConfig `mapstructure:"graphql"`
	REST    RESTConfig    `mapstructure:"rest"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	CORS    CORSConfig    `mapstructure:"cors"`
}

// GraphQLConfig represents GraphQL server configuration
type GraphQLConfig struct {
	Port       int  `mapstructure:"port"`
	Playground bool `mapstructure:"playground"`
}

// RESTConfig represents REST server configuration
type RESTConfig struct {
	Port int `mapstructure:"port"`
}

// MetricsConfig represents metrics server configuration
type MetricsConfig struct {
	Port int `mapstructure:"port"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	Origins []string `mapstructure:"origins"`
}

// IngesterConfig represents ingester configuration
type IngesterConfig struct {
	BatchSize     int           `mapstructure:"batch_size"`
	FlushInterval time.Duration `mapstructure:"flush_interval"`
	Workers       int           `mapstructure:"workers"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Set defaults
	setDefaults()

	// Unmarshal configuration
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate chains
	if len(c.Chains) == 0 {
		return fmt.Errorf("at least one chain must be configured")
	}

	for i, chain := range c.Chains {
		if chain.Name == "" {
			return fmt.Errorf("chain[%d]: name is required", i)
		}
		if chain.GRPCEndpoint == "" {
			return fmt.Errorf("chain[%d]: grpc_endpoint is required", i)
		}
		if len(chain.Modules) == 0 {
			return fmt.Errorf("chain[%d]: at least one module must be specified", i)
		}
	}

	// Validate database
	if c.Database.Postgres.Host == "" {
		return fmt.Errorf("postgres host is required")
	}
	if c.Database.Postgres.Database == "" {
		return fmt.Errorf("postgres database is required")
	}

	// Validate API ports
	if c.API.GraphQL.Port <= 0 || c.API.GraphQL.Port > 65535 {
		return fmt.Errorf("invalid GraphQL port: %d", c.API.GraphQL.Port)
	}
	if c.API.REST.Port <= 0 || c.API.REST.Port > 65535 {
		return fmt.Errorf("invalid REST port: %d", c.API.REST.Port)
	}
	if c.API.Metrics.Port <= 0 || c.API.Metrics.Port > 65535 {
		return fmt.Errorf("invalid metrics port: %d", c.API.Metrics.Port)
	}

	// Validate streaming if enabled
	if c.Streaming.Enabled {
		if len(c.Streaming.Kafka.Brokers) == 0 {
			return fmt.Errorf("kafka brokers are required when streaming is enabled")
		}
		if c.Streaming.Kafka.Topic == "" {
			return fmt.Errorf("kafka topic is required when streaming is enabled")
		}
	}

	return nil
}

// GetPostgresURL returns the PostgreSQL connection URL
func (c *Config) GetPostgresURL() string {
	sslMode := c.Database.Postgres.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.Postgres.User,
		c.Database.Postgres.Password,
		c.Database.Postgres.Host,
		c.Database.Postgres.Port,
		c.Database.Postgres.Database,
		sslMode,
	)
}

// GetClickHouseURL returns the ClickHouse connection URL
func (c *Config) GetClickHouseURL() string {
	return fmt.Sprintf("tcp://%s:%d/%s?username=%s&password=%s",
		c.Database.ClickHouse.Host,
		c.Database.ClickHouse.Port,
		c.Database.ClickHouse.Database,
		c.Database.ClickHouse.User,
		c.Database.ClickHouse.Password,
	)
}

// setDefaults sets default configuration values
func setDefaults() {
	// Chain defaults
	viper.SetDefault("chains", []ChainConfig{
		{
			Name:         "cosmoshub",
			ChainID:      "cosmoshub-4",
			GRPCEndpoint: "localhost:9090",
			RESTEndpoint: "localhost:1317",
			Modules:      []string{"bank", "staking", "distribution", "gov"},
			Enabled:      true,
		},
	})

	// Database defaults
	viper.SetDefault("database.postgres.host", "localhost")
	viper.SetDefault("database.postgres.port", 5432)
	viper.SetDefault("database.postgres.database", "statemesh")
	viper.SetDefault("database.postgres.user", "postgres")
	viper.SetDefault("database.postgres.password", "password")
	viper.SetDefault("database.postgres.ssl_mode", "disable")
	viper.SetDefault("database.postgres.max_conns", 20)
	viper.SetDefault("database.postgres.min_conns", 5)

	viper.SetDefault("database.clickhouse.host", "localhost")
	viper.SetDefault("database.clickhouse.port", 9000)
	viper.SetDefault("database.clickhouse.database", "statemesh_analytics")
	viper.SetDefault("database.clickhouse.user", "default")
	viper.SetDefault("database.clickhouse.password", "")
	viper.SetDefault("database.clickhouse.enabled", true)

	// Streaming defaults
	viper.SetDefault("streaming.enabled", false)
	viper.SetDefault("streaming.kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("streaming.kafka.topic", "cosmos-state-changes")

	// API defaults
	viper.SetDefault("api.graphql.port", 8080)
	viper.SetDefault("api.graphql.playground", true)
	viper.SetDefault("api.rest.port", 8081)
	viper.SetDefault("api.metrics.port", 9090)
	viper.SetDefault("api.cors.enabled", true)
	viper.SetDefault("api.cors.origins", []string{"*"})

	// Ingester defaults
	viper.SetDefault("ingester.batch_size", 1000)
	viper.SetDefault("ingester.flush_interval", "5s")
	viper.SetDefault("ingester.workers", 4)

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "console")
}
