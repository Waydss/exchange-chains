// Package database provides PostgreSQL infrastructure.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	connectTimeout    = 10 * time.Second
	maxConnections    = 10
	minConnections    = 1
	maxConnectionLife = 30 * time.Minute
	maxConnectionIdle = 5 * time.Minute
	healthCheckPeriod = time.Minute
)

// Connect creates and verifies a PostgreSQL connection pool.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL configuration: %w", err)
	}

	poolConfig.MaxConns = maxConnections
	poolConfig.MinConns = minConnections
	poolConfig.MaxConnLifetime = maxConnectionLife
	poolConfig.MaxConnIdleTime = maxConnectionIdle
	poolConfig.HealthCheckPeriod = healthCheckPeriod

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return pool, nil
}
