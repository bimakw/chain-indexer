package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the application
type Config struct {
	// Ethereum node configuration
	Ethereum EthereumConfig

	// Database configuration
	Database DatabaseConfig

	// Redis configuration
	Redis RedisConfig

	// API server configuration
	API APIConfig

	// Indexer configuration
	Indexer IndexerConfig

	// Logging configuration
	Log LogConfig
}

// EthereumConfig holds Ethereum node connection settings
type EthereumConfig struct {
	RPCURL          string        `envconfig:"ETH_RPC_URL" default:"http://localhost:8545"`
	ChainID         int64         `envconfig:"ETH_CHAIN_ID" default:"1"`
	RequestTimeout  time.Duration `envconfig:"ETH_REQUEST_TIMEOUT" default:"30s"`
	MaxRetries      int           `envconfig:"ETH_MAX_RETRIES" default:"3"`
	RetryDelay      time.Duration `envconfig:"ETH_RETRY_DELAY" default:"1s"`
}

// DatabaseConfig holds PostgreSQL connection settings
type DatabaseConfig struct {
	Host            string        `envconfig:"DB_HOST" default:"localhost"`
	Port            int           `envconfig:"DB_PORT" default:"5432"`
	User            string        `envconfig:"DB_USER" default:"indexer"`
	Password        string        `envconfig:"DB_PASSWORD" default:"indexer"`
	Name            string        `envconfig:"DB_NAME" default:"chain_indexer"`
	SSLMode         string        `envconfig:"DB_SSL_MODE" default:"disable"`
	MaxOpenConns    int           `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"DB_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m"`
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Host     string `envconfig:"REDIS_HOST" default:"localhost"`
	Port     int    `envconfig:"REDIS_PORT" default:"6379"`
	Password string `envconfig:"REDIS_PASSWORD" default:""`
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

// APIConfig holds API server settings
type APIConfig struct {
	Host            string        `envconfig:"API_HOST" default:"0.0.0.0"`
	Port            int           `envconfig:"API_PORT" default:"8081"`
	ReadTimeout     time.Duration `envconfig:"API_READ_TIMEOUT" default:"10s"`
	WriteTimeout    time.Duration `envconfig:"API_WRITE_TIMEOUT" default:"10s"`
	ShutdownTimeout time.Duration `envconfig:"API_SHUTDOWN_TIMEOUT" default:"30s"`
	RateLimitRPS    int           `envconfig:"API_RATE_LIMIT_RPS" default:"100"`
	CacheTTL        time.Duration `envconfig:"API_CACHE_TTL" default:"30s"`
}

// IndexerConfig holds indexer-specific settings
type IndexerConfig struct {
	MetricsPort       int           `envconfig:"INDEXER_METRICS_PORT" default:"8080"`
	BatchSize         int           `envconfig:"INDEXER_BATCH_SIZE" default:"100"`
	BlockConfirmations int          `envconfig:"INDEXER_BLOCK_CONFIRMATIONS" default:"12"`
	PollInterval      time.Duration `envconfig:"INDEXER_POLL_INTERVAL" default:"12s"`
	BackfillBatchSize int           `envconfig:"INDEXER_BACKFILL_BATCH_SIZE" default:"1000"`
	WorkerCount       int           `envconfig:"INDEXER_WORKER_COUNT" default:"4"`

	// Tokens to index (comma-separated addresses)
	TokenAddresses []string `envconfig:"INDEXER_TOKEN_ADDRESSES" default:"0xdAC17F958D2ee523a2206206994597C13D831ec7,0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"`
}

// LogConfig holds logging settings
type LogConfig struct {
	Level  string `envconfig:"LOG_LEVEL" default:"info"`
	Format string `envconfig:"LOG_FORMAT" default:"json"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// DSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + string(rune(c.Port)) +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.Name +
		" sslmode=" + c.SSLMode
}
