// Package redisclient creates and verifies the application's Redis client.
package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config contains the Redis connection settings required by the client.
type Config struct {
	Addr         string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Connection owns the Redis client and exposes readiness-friendly operations.
type Connection struct {
	client *redis.Client
}

// Connect creates one Redis client and verifies the connection with PING.
func Connect(ctx context.Context, cfg Config) (*Connection, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	pingContext, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
	defer cancel()
	if err := client.Ping(pingContext).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping Redis: %w", err)
	}

	return &Connection{client: client}, nil
}

// Client returns the single underlying Redis client for dependency injection.
func (c *Connection) Client() *redis.Client {
	return c.client
}

// Ping verifies Redis availability for application readiness.
func (c *Connection) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping Redis: %w", err)
	}
	return nil
}

// Close releases Redis client resources.
func (c *Connection) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("close Redis client: %w", err)
	}
	return nil
}
