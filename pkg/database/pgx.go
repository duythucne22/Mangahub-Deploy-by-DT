package database

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

// NewPGXPool creates a pgx connection pool (matches repository layer)
func NewPGXPool(config Config) (*pgxpool.Pool, error) {
    if config.Timeout == 0 {
        config.Timeout = 5 * time.Second
    }
    if config.SSLMode == "" {
        config.SSLMode = "disable"
    }

    dsn := fmt.Sprintf(
        "postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=%d",
        config.User,
        config.Password,
        config.Host,
        config.Port,
        config.Database,
        config.SSLMode,
        int(config.Timeout.Seconds()),
    )

    poolConfig, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to parse pgx config: %w", err)
    }

    if config.MaxOpenConns > 0 {
        poolConfig.MaxConns = int32(config.MaxOpenConns)
    }
    if config.MaxIdleConns > 0 {
        poolConfig.MinConns = int32(config.MaxIdleConns)
    }
    if config.ConnMaxLifetime > 0 {
        poolConfig.MaxConnLifetime = config.ConnMaxLifetime
    }
    if config.ConnMaxIdleTime > 0 {
        poolConfig.MaxConnIdleTime = config.ConnMaxIdleTime
    }

    ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
    defer cancel()

    pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create pgx pool: %w", err)
    }

    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("failed to ping pgx pool: %w", err)
    }

    return pool, nil
}