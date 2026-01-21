package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDB(t *testing.T) {
	// This test requires a running PostgreSQL instance
	// Skip if DATABASE_URL is not set
	config := Config{
		Host:            "localhost",
		Port:            5432,
		User:            "mangahub",
		Password:        "mangahub_dev_password",
		Database:        "mangahub_dev",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		Timeout:         10 * time.Second,
	}

	db, err := NewDB(config)
	if err != nil {
		t.Skipf("Skipping test: PostgreSQL not available: %v", err)
		return
	}
	defer db.Close()

	// Test connection is alive
	ctx := context.Background()
	err = db.HealthCheck(ctx)
	require.NoError(t, err)

	// Test stats
	stats := db.Stats()
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 5)
}

func TestHealthCheck(t *testing.T) {
	config := Config{
		Host:            "localhost",
		Port:            5432,
		User:            "mangahub",
		Password:        "mangahub_dev_password",
		Database:        "mangahub_dev",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		Timeout:         10 * time.Second,
	}

	db, err := NewDB(config)
	if err != nil {
		t.Skipf("Skipping test: PostgreSQL not available: %v", err)
		return
	}
	defer db.Close()

	ctx := context.Background()
	err = db.HealthCheck(ctx)
	assert.NoError(t, err)

	// Test with cancelled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err = db.HealthCheck(cancelCtx)
	assert.Error(t, err)
}
