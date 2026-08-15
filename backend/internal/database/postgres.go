package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect establishes a connection to the PostgreSQL database using the provided database URL and returns a connection pool.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	// Set connection pool settings
	config.MaxConns = 10               // Adjust as needed
	config.MinConns = 2                // the minimum number of connections to maintain in the pool
	config.MaxConnLifetime = time.Hour // the maximum amount of time a connection can be reused before being closed and replaced with a new connection

	// Create a new connection pool and handle any errors that occur during the creation of the pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	// Ping the database to ensure it's reachable
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil // Return the connection pool to the caller
}
