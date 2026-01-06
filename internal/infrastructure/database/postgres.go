package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/bimakw/chain-indexer/internal/config"
)

// PostgresDB wraps the sqlx database connection
type PostgresDB struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewPostgresDB creates a new PostgreSQL connection
func NewPostgresDB(cfg config.DatabaseConfig, logger *zap.Logger) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Connected to PostgreSQL",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
	)

	return &PostgresDB{
		db:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// DB returns the underlying sqlx.DB
func (p *PostgresDB) DB() *sqlx.DB {
	return p.db
}

// HealthCheck performs a health check on the database
func (p *PostgresDB) HealthCheck(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
