package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config represents the entire application configuration
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Postgres     PostgresConfig     `yaml:"postgres"`
	Kubernetes   KubernetesConfig   `yaml:"kubernetes"`
	Logging      LoggingConfig      `yaml:"logging"`
	Email        EmailConfig        `yaml:"email"`
	AlertRules   AlertRulesConfig   `yaml:"alert_rules"`
}

type ServerConfig struct {
	ReadTimeout  int `yaml:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout"`
	Port int `yaml:"port"`
}

type PostgresConfig struct {
	AutoMigrate bool `yaml:"auto_migrate"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"sslmode"`
}

type KubernetesConfig struct {
	InCluster       bool   `yaml:"in_cluster"`
	ConfigPath      string `yaml:"config_path"`
	MetricsInterval int    `yaml:"metrics_interval"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

type EmailConfig struct {
	Enabled  bool     `yaml:"enabled"`
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
}

type AlertRulesConfig struct {
	PodRestartThreshold   int     `yaml:"pod_restart_threshold"`
	PodCPUThreshold       int     `yaml:"pod_cpu_threshold"`
	PodMemoryThreshold    int     `yaml:"pod_memory_threshold"`
	NodeCPUThreshold      int     `yaml:"node_cpu_threshold"`
	NodeMemoryThreshold   int     `yaml:"node_memory_threshold"`
	MetricsCheckInterval  int     `yaml:"metrics_check_interval"`
	PodCPUPercent         float64 `yaml:"-"` // Computed from PodCPUThreshold
	PodMemoryPercent      float64 `yaml:"-"` // Computed from PodMemoryThreshold
	NodeCPUPercent        float64 `yaml:"-"` // Computed from NodeCPUThreshold
	NodeMemoryPercent     float64 `yaml:"-"` // Computed from NodeMemoryThreshold
}

// overrideFromEnv overrides config values with environment variables
func overrideFromEnv(cfg *Config) {
	// Server configuration
	if port := os.Getenv("SERVER_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Server.Port)
	}
	if readTimeout := os.Getenv("SERVER_READ_TIMEOUT"); readTimeout != "" {
		fmt.Sscanf(readTimeout, "%d", &cfg.Server.ReadTimeout)
	}
	if writeTimeout := os.Getenv("SERVER_WRITE_TIMEOUT"); writeTimeout != "" {
		fmt.Sscanf(writeTimeout, "%d", &cfg.Server.WriteTimeout)
	}

	// PostgreSQL configuration
	if host := os.Getenv("POSTGRES_HOST"); host != "" {
		cfg.Postgres.Host = host
	}
	if port := os.Getenv("POSTGRES_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Postgres.Port)
	}
	if user := os.Getenv("POSTGRES_USER"); user != "" {
		cfg.Postgres.User = user
	}
	if pass := os.Getenv("POSTGRES_PASSWORD"); pass != "" {
		cfg.Postgres.Password = pass
	}
	if db := os.Getenv("POSTGRES_DB"); db != "" {
		cfg.Postgres.Database = db
	}
	if sslmode := os.Getenv("POSTGRES_SSLMODE"); sslmode != "" {
		cfg.Postgres.SSLMode = sslmode
	}
	if autoMigrate := os.Getenv("POSTGRES_AUTO_MIGRATE"); autoMigrate != "" {
		cfg.Postgres.AutoMigrate = strings.ToLower(autoMigrate) == "true"
	}

	// Kubernetes configuration
	if inCluster := os.Getenv("K8S_IN_CLUSTER"); inCluster != "" {
		cfg.Kubernetes.InCluster = strings.ToLower(inCluster) == "true"
	}
	if configPath := os.Getenv("KUBECONFIG"); configPath != "" {
		cfg.Kubernetes.ConfigPath = configPath
	}
	if metricsInterval := os.Getenv("K8S_METRICS_INTERVAL"); metricsInterval != "" {
		fmt.Sscanf(metricsInterval, "%d", &cfg.Kubernetes.MetricsInterval)
	}

	// Logging configuration
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Logging.Level = level
	}
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		cfg.Logging.Format = format
	}
	if output := os.Getenv("LOG_OUTPUT"); output != "" {
		cfg.Logging.Output = output
	}

	// Email configuration
	if enabled := os.Getenv("EMAIL_ENABLED"); enabled != "" {
		cfg.Email.Enabled = strings.ToLower(enabled) == "true"
	}
	if host := os.Getenv("SMTP_HOST"); host != "" {
		cfg.Email.SMTPHost = host
	}
	if port := os.Getenv("SMTP_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Email.SMTPPort)
	}
	if from := os.Getenv("SMTP_FROM"); from != "" {
		cfg.Email.From = from
	}
	if username := os.Getenv("SMTP_USERNAME"); username != "" {
		cfg.Email.Username = username
	}
	if password := os.Getenv("SMTP_PASSWORD"); password != "" {
		cfg.Email.Password = password
	}
	if to := os.Getenv("SMTP_TO"); to != "" {
		cfg.Email.To = strings.Split(to, ",")
	}

	// Alert rules configuration
	if podRestartThreshold := os.Getenv("ALERT_POD_RESTART_THRESHOLD"); podRestartThreshold != "" {
		fmt.Sscanf(podRestartThreshold, "%d", &cfg.AlertRules.PodRestartThreshold)
	}
	if podCPUThreshold := os.Getenv("ALERT_POD_CPU_THRESHOLD"); podCPUThreshold != "" {
		fmt.Sscanf(podCPUThreshold, "%d", &cfg.AlertRules.PodCPUThreshold)
	}
	if podMemThreshold := os.Getenv("ALERT_POD_MEMORY_THRESHOLD"); podMemThreshold != "" {
		fmt.Sscanf(podMemThreshold, "%d", &cfg.AlertRules.PodMemoryThreshold)
	}
	if nodeCPUThreshold := os.Getenv("ALERT_NODE_CPU_THRESHOLD"); nodeCPUThreshold != "" {
		fmt.Sscanf(nodeCPUThreshold, "%d", &cfg.AlertRules.NodeCPUThreshold)
	}
	if nodeMemThreshold := os.Getenv("ALERT_NODE_MEMORY_THRESHOLD"); nodeMemThreshold != "" {
		fmt.Sscanf(nodeMemThreshold, "%d", &cfg.AlertRules.NodeMemoryThreshold)
	}
	if metricsCheckInterval := os.Getenv("ALERT_METRICS_CHECK_INTERVAL"); metricsCheckInterval != "" {
		fmt.Sscanf(metricsCheckInterval, "%d", &cfg.AlertRules.MetricsCheckInterval)
	}
}

// Load reads and parses the config file
func Load(path string) (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Override with environment variables if present (env vars take priority)
	overrideFromEnv(&cfg)

	// Compute percent fields
	cfg.AlertRules.PodCPUPercent = float64(cfg.AlertRules.PodCPUThreshold)
	cfg.AlertRules.PodMemoryPercent = float64(cfg.AlertRules.PodMemoryThreshold)
	cfg.AlertRules.NodeCPUPercent = float64(cfg.AlertRules.NodeCPUThreshold)
	cfg.AlertRules.NodeMemoryPercent = float64(cfg.AlertRules.NodeMemoryThreshold)
	
	// Copy to Alerts alias
	
	
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 15
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 15
	}

	return &cfg, nil
}

// ConnectionString returns the PostgreSQL connection string
func (p PostgresConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.Database, p.SSLMode)
}

// GetDSN returns the data source name for PostgreSQL
func (p PostgresConfig) GetDSN() string {
	return p.ConnectionString()
}

// MaxConnections returns max connections (default 25)
func (p PostgresConfig) MaxConnections() int {
	return 25
}

// MaxIdleConnections returns max idle connections (default 5)
func (p PostgresConfig) MaxIdleConnections() int {
	return 5
}

// ConnectionLifetime returns connection lifetime in minutes (default 5)
func (p PostgresConfig) ConnectionLifetime() time.Duration {
	return 5 * time.Minute
}

// MigrationSourceURL returns the migration source URL
func (p PostgresConfig) MigrationSourceURL() string {
	return "file://internal/storage/migrations"
}

// MigrationDatabaseURL returns the database URL for migrations
func (p PostgresConfig) MigrationDatabaseURL() string {
	password := strings.ReplaceAll(p.Password, "@", "%40")
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, password, p.Host, p.Port, p.Database, p.SSLMode)
}

// Global config instance (for backwards compatibility)
var globalConfig *Config

// Get returns the global config instance
func Get() *Config {
	return globalConfig
}

// SetGlobalConfig sets the global config
func SetGlobalConfig(cfg *Config) {
	globalConfig = cfg
}
