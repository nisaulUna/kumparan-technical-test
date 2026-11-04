package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(databaseURL string) (*pgxpool.Pool, error) {
	// Parse database configuration
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse database config: %w", err,
		)
	}

	// Configure connection pool
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 15 * time.Minute
	config.HealthCheckPeriod = time.Minute

	// Create connection pool
	db, err := pgxpool.NewWithConfig(
		context.Background(),
		config,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create connection pool: %w", err,
		)
	}

	// Verify database connection
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf(
			"failed to connect to PostgreSQL: %w", err,
		)
	}

	return db, nil
}
