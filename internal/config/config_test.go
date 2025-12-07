package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/monitoring-engine/monitoring-tool/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("should load valid config file", func(t *testing.T) {
		// Create a temporary config file
		configContent := `
server:
  port: 8080
  read_timeout: 30
  write_timeout: 30

postgres:
  auto_migrate: true
  host: localhost
  port: 5432
  user: postgres
  password: secret
  database: testdb
  sslmode: disable

kubernetes:
  in_cluster: false
  config_path: ~/.kube/config
  metrics_interval: 30

logging:
  level: info
  format: json
  output: stdout

email:
  enabled: false
  smtp_host: smtp.gmail.com
  smtp_port: 587
  username: test@example.com
  password: password
  from: alerts@example.com
  to:
    - admin@example.com

alert_rules:
  pod_restart_threshold: 3
  pod_cpu_threshold: 80
  pod_memory_threshold: 85
  node_cpu_threshold: 70
  node_memory_threshold: 75
  metrics_check_interval: 60
`
		tmpFile := filepath.Join(t.TempDir(), "config.yaml")
		err := os.WriteFile(tmpFile, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := config.Load(tmpFile)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// Verify server config
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, 30, cfg.Server.ReadTimeout)
		assert.Equal(t, 30, cfg.Server.WriteTimeout)

		// Verify postgres config
		assert.Equal(t, "localhost", cfg.Postgres.Host)
		assert.Equal(t, 5432, cfg.Postgres.Port)
		assert.Equal(t, "postgres", cfg.Postgres.User)
		assert.Equal(t, "testdb", cfg.Postgres.Database)

		// Verify kubernetes config
		assert.False(t, cfg.Kubernetes.InCluster)
		assert.Equal(t, 30, cfg.Kubernetes.MetricsInterval)

		// Verify alert rules
		assert.Equal(t, 3, cfg.AlertRules.PodRestartThreshold)
		assert.Equal(t, 80, cfg.AlertRules.PodCPUThreshold)
		assert.Equal(t, 85, cfg.AlertRules.PodMemoryThreshold)

		// Verify computed percentages
		assert.Equal(t, 80.0, cfg.AlertRules.PodCPUPercent)
		assert.Equal(t, 85.0, cfg.AlertRules.PodMemoryPercent)
		assert.Equal(t, 70.0, cfg.AlertRules.NodeCPUPercent)
		assert.Equal(t, 75.0, cfg.AlertRules.NodeMemoryPercent)
	})

	t.Run("should apply default timeouts", func(t *testing.T) {
		configContent := `
server:
  port: 8080

postgres:
  host: localhost
  port: 5432
  user: postgres
  password: secret
  database: testdb
  sslmode: disable
`
		tmpFile := filepath.Join(t.TempDir(), "config.yaml")
		err := os.WriteFile(tmpFile, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := config.Load(tmpFile)
		assert.NoError(t, err)

		// Should use default timeouts
		assert.Equal(t, 15, cfg.Server.ReadTimeout)
		assert.Equal(t, 15, cfg.Server.WriteTimeout)
	})

	t.Run("should override postgres password from env", func(t *testing.T) {
		configContent := `
postgres:
  host: localhost
  port: 5432
  user: postgres
  password: old_password
  database: testdb
  sslmode: disable
`
		tmpFile := filepath.Join(t.TempDir(), "config.yaml")
		err := os.WriteFile(tmpFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Set environment variable
		os.Setenv("POSTGRES_PASSWORD", "new_password")
		defer os.Unsetenv("POSTGRES_PASSWORD")

		cfg, err := config.Load(tmpFile)
		assert.NoError(t, err)
		assert.Equal(t, "new_password", cfg.Postgres.Password)
	})

	t.Run("should expand environment variables", func(t *testing.T) {
		os.Setenv("DB_HOST", "myhost")
		os.Setenv("DB_PORT", "5433")
		defer os.Unsetenv("DB_HOST")
		defer os.Unsetenv("DB_PORT")

		configContent := `
postgres:
  host: ${DB_HOST}
  port: ${DB_PORT}
  user: postgres
  password: secret
  database: testdb
  sslmode: disable
`
		tmpFile := filepath.Join(t.TempDir(), "config.yaml")
		err := os.WriteFile(tmpFile, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := config.Load(tmpFile)
		assert.NoError(t, err)
		assert.Equal(t, "myhost", cfg.Postgres.Host)
		assert.Equal(t, 5433, cfg.Postgres.Port)
	})

	t.Run("should return error for missing file", func(t *testing.T) {
		cfg, err := config.Load("/nonexistent/config.yaml")
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("should return error for invalid yaml", func(t *testing.T) {
		configContent := `
invalid yaml content
  this is not: valid
    - yaml
`
		tmpFile := filepath.Join(t.TempDir(), "config.yaml")
		err := os.WriteFile(tmpFile, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := config.Load(tmpFile)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to parse config")
	})
}

func TestPostgresConfig_ConnectionString(t *testing.T) {
	t.Run("should generate correct connection string", func(t *testing.T) {
		pg := config.PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "testuser",
			Password: "testpass",
			Database: "testdb",
			SSLMode:  "disable",
		}

		connStr := pg.ConnectionString()
		assert.Equal(t, "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable", connStr)
	})

	t.Run("should handle special characters", func(t *testing.T) {
		pg := config.PostgresConfig{
			Host:     "db.example.com",
			Port:     5432,
			User:     "user@example",
			Password: "p@ssw0rd!",
			Database: "my-db",
			SSLMode:  "require",
		}

		connStr := pg.ConnectionString()
		assert.Contains(t, connStr, "host=db.example.com")
		assert.Contains(t, connStr, "user=user@example")
		assert.Contains(t, connStr, "password=p@ssw0rd!")
	})
}

func TestPostgresConfig_GetDSN(t *testing.T) {
	t.Run("should return DSN", func(t *testing.T) {
		pg := config.PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "secret",
			Database: "mydb",
			SSLMode:  "disable",
		}

		dsn := pg.GetDSN()
		assert.Equal(t, pg.ConnectionString(), dsn)
	})
}

func TestPostgresConfig_MaxConnections(t *testing.T) {
	t.Run("should return default max connections", func(t *testing.T) {
		pg := config.PostgresConfig{}
		assert.Equal(t, 25, pg.MaxConnections())
	})
}

func TestPostgresConfig_MaxIdleConnections(t *testing.T) {
	t.Run("should return default max idle connections", func(t *testing.T) {
		pg := config.PostgresConfig{}
		assert.Equal(t, 5, pg.MaxIdleConnections())
	})
}

func TestPostgresConfig_ConnectionLifetime(t *testing.T) {
	t.Run("should return default connection lifetime", func(t *testing.T) {
		pg := config.PostgresConfig{}
		assert.Equal(t, 5*time.Minute, pg.ConnectionLifetime())
	})
}

func TestPostgresConfig_MigrationSourceURL(t *testing.T) {
	t.Run("should return migration source URL", func(t *testing.T) {
		pg := config.PostgresConfig{}
		assert.Equal(t, "file://internal/storage/migrations", pg.MigrationSourceURL())
	})
}

func TestPostgresConfig_MigrationDatabaseURL(t *testing.T) {
	t.Run("should generate correct migration URL", func(t *testing.T) {
		pg := config.PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "secret",
			Database: "mydb",
			SSLMode:  "disable",
		}

		url := pg.MigrationDatabaseURL()
		expected := "postgres://postgres:secret@localhost:5432/mydb?sslmode=disable"
		assert.Equal(t, expected, url)
	})

	t.Run("should escape @ in password", func(t *testing.T) {
		pg := config.PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "pass@word",
			Database: "db",
			SSLMode:  "disable",
		}

		url := pg.MigrationDatabaseURL()
		assert.Contains(t, url, "pass%40word")
		assert.NotContains(t, url, "pass@word@")
	})
}

func TestGlobalConfig(t *testing.T) {
	t.Run("should set and get global config", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Port: 9090,
			},
		}

		config.SetGlobalConfig(cfg)
		retrieved := config.Get()

		assert.NotNil(t, retrieved)
		assert.Equal(t, 9090, retrieved.Server.Port)
	})

	t.Run("should return nil before setting", func(t *testing.T) {
		// Reset global config
		config.SetGlobalConfig(nil)

		retrieved := config.Get()
		assert.Nil(t, retrieved)
	})
}
