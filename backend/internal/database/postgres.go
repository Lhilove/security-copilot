package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
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

// Migrate runs all pending database migrations from the provided migrations directory.
func Migrate(databaseURL string, migrationsPath string) error {
	// Open the migrations directory as an fs.FS
	dir := os.DirFS(migrationsPath)

	driver, err := iofs.New(dir, ".")
	if err != nil {
		return fmt.Errorf("create migrations source: %w", err)
	}

	migrateURL := "pgx5://" + strings.TrimPrefix(
		strings.TrimPrefix(databaseURL, "postgres://"),
		"postgresql://",
	)

	m, err := migrate.NewWithSourceInstance("iofs", driver, migrateURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
