package storage_test

import (
	"testing"
	"time"

	"github.com/monitoring-engine/monitoring-tool/internal/config"
	"github.com/monitoring-engine/monitoring-tool/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestHealthCheck(t *testing.T) {
	t.Run("should return nil for healthy database", func(t *testing.T) {
		db := setupTestDB(t)
		err := storage.HealthCheck(db)
		assert.NoError(t, err)
	})

	t.Run("should return error for nil database", func(t *testing.T) {
		err := storage.HealthCheck(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database instance is nil")
	})

	t.Run("should return error for closed database", func(t *testing.T) {
		db := setupTestDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)

		// Close the database
		sqlDB.Close()

		// Health check should fail
		err = storage.HealthCheck(db)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database ping failed")
	})

	t.Run("should use context with timeout", func(t *testing.T) {
		db := setupTestDB(t)

		// Multiple health checks should complete quickly
		start := time.Now()
		for i := 0; i < 5; i++ {
			err := storage.HealthCheck(db)
			assert.NoError(t, err)
		}
		duration := time.Since(start)

		// Should complete well within 10 seconds (2 second timeout per check)
		assert.Less(t, duration, 10*time.Second)
	})
}

func TestClose(t *testing.T) {
	t.Run("should close database successfully", func(t *testing.T) {
		db := setupTestDB(t)

		// Close should not panic
		storage.Close(db)

		// Verify database is closed
		sqlDB, err := db.DB()
		require.NoError(t, err)

		// Ping should fail after close
		err = sqlDB.Ping()
		assert.Error(t, err)
	})

	t.Run("should handle nil database gracefully", func(t *testing.T) {
		// Should not panic
		storage.Close(nil)
	})

	t.Run("should be idempotent", func(t *testing.T) {
		db := setupTestDB(t)

		// Close multiple times should not panic
		storage.Close(db)
		storage.Close(db)
		storage.Close(db)
	})
}

func TestPostgresConfig_Methods(t *testing.T) {
	t.Run("should return correct DSN", func(t *testing.T) {
		cfg := config.PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "testuser",
			Password: "testpass",
			Database: "testdb",
			SSLMode:  "disable",
		}

		dsn := cfg.GetDSN()
		expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
		assert.Equal(t, expected, dsn)
	})

	t.Run("should return default max connections", func(t *testing.T) {
		cfg := config.PostgresConfig{}
		assert.Equal(t, 25, cfg.MaxConnections())
	})

	t.Run("should return default max idle connections", func(t *testing.T) {
		cfg := config.PostgresConfig{}
		assert.Equal(t, 5, cfg.MaxIdleConnections())
	})

	t.Run("should return default connection lifetime", func(t *testing.T) {
		cfg := config.PostgresConfig{}
		assert.Equal(t, 5*time.Minute, cfg.ConnectionLifetime())
	})
}

func TestHealthCheck_EdgeCases(t *testing.T) {
	t.Run("should handle rapid health checks", func(t *testing.T) {
		db := setupTestDB(t)

		// Perform many rapid health checks
		for i := 0; i < 50; i++ {
			err := storage.HealthCheck(db)
			assert.NoError(t, err)
		}
	})

	t.Run("should handle concurrent health checks", func(t *testing.T) {
		db := setupTestDB(t)

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				err := storage.HealthCheck(db)
				assert.NoError(t, err)
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestClose_EdgeCases(t *testing.T) {
	t.Run("should handle closing already closed database", func(t *testing.T) {
		db := setupTestDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)

		// Close manually first
		sqlDB.Close()

		// Close via storage.Close should not panic
		storage.Close(db)
	})

	t.Run("should handle concurrent close calls", func(t *testing.T) {
		db := setupTestDB(t)

		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func() {
				storage.Close(db)
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 5; i++ {
			<-done
		}
	})
}

func TestHealthCheck_Timeout(t *testing.T) {
	t.Run("should timeout after 2 seconds", func(t *testing.T) {
		db := setupTestDB(t)

		start := time.Now()
		err := storage.HealthCheck(db)
		duration := time.Since(start)

		assert.NoError(t, err)
		// Should complete quickly for a working database
		assert.Less(t, duration, 2*time.Second)
	})
}

func TestPostgresConfig_ConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   config.PostgresConfig
		expected string
	}{
		{
			name: "standard config",
			config: config.PostgresConfig{
				Host:     "db.example.com",
				Port:     5432,
				User:     "admin",
				Password: "secret",
				Database: "mydb",
				SSLMode:  "require",
			},
			expected: "host=db.example.com port=5432 user=admin password=secret dbname=mydb sslmode=require",
		},
		{
			name: "localhost config",
			config: config.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "postgres",
				Database: "postgres",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable",
		},
		{
			name: "custom port",
			config: config.PostgresConfig{
				Host:     "localhost",
				Port:     5433,
				User:     "user",
				Password: "pass",
				Database: "db",
				SSLMode:  "prefer",
			},
			expected: "host=localhost port=5433 user=user password=pass dbname=db sslmode=prefer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.GetDSN()
			assert.Equal(t, tt.expected, dsn)
		})
	}
}

func TestDatabase_Lifecycle(t *testing.T) {
	t.Run("should handle full lifecycle", func(t *testing.T) {
		// Create database
		db := setupTestDB(t)
		assert.NotNil(t, db)

		// Health check should pass
		err := storage.HealthCheck(db)
		assert.NoError(t, err)

		// Close database
		storage.Close(db)

		// Health check should fail after close
		err = storage.HealthCheck(db)
		assert.Error(t, err)
	})
}

func TestHealthCheck_ContextTimeout(t *testing.T) {
	t.Run("should respect context timeout", func(t *testing.T) {
		db := setupTestDB(t)

		// Health check uses 2 second timeout internally
		start := time.Now()
		err := storage.HealthCheck(db)
		duration := time.Since(start)

		assert.NoError(t, err)
		// Should not wait longer than necessary
		assert.Less(t, duration, 3*time.Second)
	})
}

func TestClose_SafetyChecks(t *testing.T) {
	t.Run("should safely handle partially initialized db", func(t *testing.T) {
		db := setupTestDB(t)

		// Close immediately
		storage.Close(db)

		// Verify closed
		sqlDB, err := db.DB()
		require.NoError(t, err)
		err = sqlDB.Ping()
		assert.Error(t, err)
	})
}

func TestHealthCheck_MultipleSequentialChecks(t *testing.T) {
	t.Run("should handle multiple sequential health checks", func(t *testing.T) {
		db := setupTestDB(t)

		// Perform sequential health checks
		for i := 0; i < 20; i++ {
			err := storage.HealthCheck(db)
			assert.NoError(t, err, "Health check %d failed", i)
		}
	})
}

func TestPostgresConfig_PoolSettings(t *testing.T) {
	t.Run("should have consistent pool settings", func(t *testing.T) {
		cfg := config.PostgresConfig{}

		maxConns := cfg.MaxConnections()
		maxIdle := cfg.MaxIdleConnections()
		lifetime := cfg.ConnectionLifetime()

		// Idle connections should be less than max connections
		assert.Less(t, maxIdle, maxConns)

		// Lifetime should be reasonable
		assert.Greater(t, lifetime, time.Duration(0))
		assert.LessOrEqual(t, lifetime, 1*time.Hour)
	})
}

func TestClose_WithActiveConnections(t *testing.T) {
	t.Run("should close database with active connections", func(t *testing.T) {
		db := setupTestDB(t)

		// Perform some operations to create active connections
		for i := 0; i < 5; i++ {
			err := storage.HealthCheck(db)
			assert.NoError(t, err)
		}

		// Close should still work
		storage.Close(db)

		// Verify closed
		sqlDB, err := db.DB()
		require.NoError(t, err)
		err = sqlDB.Ping()
		assert.Error(t, err)
	})
}
