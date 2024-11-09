package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config holds all configuration settings for the application
type Config struct {
	Environment string         `mapstructure:"environment"`
	LogLevel    string         `mapstructure:"log_level"`
	Database    DatabaseConfig `mapstructure:"database"`
	P2P         P2PConfig      `mapstructure:"p2p"`
	Scripts     ScriptConfig   `mapstructure:"scripts"`
	Scheduler   SchedConfig    `mapstructure:"scheduler"`
	Security    SecurityConfig `mapstructure:"security"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	URL      string        `mapstructure:"url"`
	MaxConns int           `mapstructure:"max_conns"`
	Timeout  time.Duration `mapstructure:"timeout"`
	SSLMode  string        `mapstructure:"ssl_mode"`
}

// P2PConfig holds P2P network related configuration
type P2PConfig struct {
	Port             int           `mapstructure:"port"`
	BootstrapPeers   []string      `mapstructure:"bootstrap_peers"`
	MaxPeers         int           `mapstructure:"max_peers"`
	MinPeers         int           `mapstructure:"min_peers"`
	PeerTimeout      time.Duration `mapstructure:"peer_timeout"`
	MinVoters        int           `mapstructure:"min_voters"`
	ValidationQuorum float64       `mapstructure:"validation_quorum"`
	VotingTimeout    time.Duration `mapstructure:"voting_timeout"`
	Topics           []string      `mapstructure:"topics"`
	Security         *SecurityConfig
}

// ScriptConfig holds script execution related configuration
type ScriptConfig struct {
	ScriptDir   string        `mapstructure:"script_dir"`
	PythonPath  string        `mapstructure:"python_path"`
	MaxExecTime time.Duration `mapstructure:"max_exec_time"`
	MaxMemoryMB int           `mapstructure:"max_memory_mb"`
	AllowedPkgs []string      `mapstructure:"allowed_packages"`
}

// SchedConfig holds scheduler related configuration
type SchedConfig struct {
	MaxConcurrent   int           `mapstructure:"max_concurrent"`
	DefaultInterval time.Duration `mapstructure:"default_interval"`
	RetryAttempts   int           `mapstructure:"retry_attempts"`
	RetryDelay      time.Duration `mapstructure:"retry_delay"`
}

// SecurityConfig holds security related configuration
type SecurityConfig struct {
	MinReputationScore float64       `mapstructure:"min_reputation_score"`
	MaxPenalty         float64       `mapstructure:"max_penalty"`
	KeyFile            string        `mapstructure:"key_file"`
	AuthorityNodes     []string      `mapstructure:"authority_nodes"`
	TokenExpiry        time.Duration `mapstructure:"token_expiry"`
}

// Load reads the configuration file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set default configuration values
	setDefaults(v)

	// Read the config file
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, will rely on defaults and env vars
	}

	// Override with environment variables
	v.SetEnvPrefix("P2P")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Parse the configuration
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default values for all configuration options
func setDefaults(v *viper.Viper) {
	// General defaults
	v.SetDefault("environment", "development")
	v.SetDefault("log_level", "info")

	// P2P defaults
	v.SetDefault("p2p.port", 9000)
	v.SetDefault("p2p.max_peers", 50)
	v.SetDefault("p2p.min_peers", 3)
	v.SetDefault("p2p.peer_timeout", "30s")
	v.SetDefault("p2p.validation_quorum", 0.66)
	v.SetDefault("p2p.topics", []string{"market_data", "validation", "scripts"})

	// Script defaults
	v.SetDefault("scripts.script_dir", "scripts")
	v.SetDefault("scripts.max_exec_time", "5m")
	v.SetDefault("scripts.max_memory_mb", 512)
	v.SetDefault("scripts.allowed_packages", []string{
		"pandas",
		"numpy",
		"scikit-learn",
		"requests",
	})

	// Scheduler defaults
	v.SetDefault("scheduler.max_concurrent", 10)
	v.SetDefault("scheduler.default_interval", "5m")
	v.SetDefault("scheduler.retry_attempts", 3)
	v.SetDefault("scheduler.retry_delay", "1m")

	// Security defaults
	v.SetDefault("security.min_reputation_score", 0.5)
	v.SetDefault("security.max_penalty", 1.0)
	v.SetDefault("security.token_expiry", "24h")

	// Database defaults
	v.SetDefault("database.max_conns", 10)
	v.SetDefault("database.timeout", "30s")
	v.SetDefault("database.ssl_mode", "disable")
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate Database configuration
	if err := c.validateDatabase(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}

	// Validate P2P configuration
	if err := c.validateP2P(); err != nil {
		return fmt.Errorf("p2p config: %w", err)
	}

	// Validate Scripts configuration
	if err := c.validateScripts(); err != nil {
		return fmt.Errorf("scripts config: %w", err)
	}

	// Validate Scheduler configuration
	if err := c.validateScheduler(); err != nil {
		return fmt.Errorf("scheduler config: %w", err)
	}

	// Validate Security configuration
	if err := c.validateSecurity(); err != nil {
		return fmt.Errorf("security config: %w", err)
	}

	return nil
}

func (c *Config) validateDatabase() error {
	if c.Database.URL == "" {
		return fmt.Errorf("database URL cannot be empty")
	}
	if c.Database.MaxConns <= 0 {
		return fmt.Errorf("max_conns must be positive")
	}
	if c.Database.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

func (c *Config) validateP2P() error {
	if c.P2P.Port <= 0 || c.P2P.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", c.P2P.Port)
	}

	if c.P2P.MaxPeers < c.P2P.MinPeers {
		return fmt.Errorf("max_peers (%d) cannot be less than min_peers (%d)",
			c.P2P.MaxPeers, c.P2P.MinPeers)
	}

	if c.P2P.ValidationQuorum <= 0 || c.P2P.ValidationQuorum > 1 {
		return fmt.Errorf("validation_quorum must be between 0 and 1")
	}

	return nil
}

func (c *Config) validateScripts() error {
	if c.Scripts.ScriptDir == "" {
		return fmt.Errorf("script_dir cannot be empty")
	}

	// Ensure script directory exists
	if _, err := os.Stat(c.Scripts.ScriptDir); os.IsNotExist(err) {
		// Try to create the directory
		if err := os.MkdirAll(c.Scripts.ScriptDir, 0755); err != nil {
			return fmt.Errorf("failed to create script directory: %w", err)
		}
	}

	if c.Scripts.MaxMemoryMB <= 0 {
		return fmt.Errorf("max_memory_mb must be positive")
	}

	return nil
}

func (c *Config) validateScheduler() error {
	if c.Scheduler.MaxConcurrent <= 0 {
		return fmt.Errorf("max_concurrent must be positive")
	}

	if c.Scheduler.RetryAttempts < 0 {
		return fmt.Errorf("retry_attempts cannot be negative")
	}

	return nil
}

func (c *Config) validateSecurity() error {
	if c.Security.MinReputationScore < 0 || c.Security.MinReputationScore > 1 {
		return fmt.Errorf("min_reputation_score must be between 0 and 1")
	}

	if c.Security.MaxPenalty <= 0 || c.Security.MaxPenalty > 1 {
		return fmt.Errorf("max_penalty must be between 0 and 1")
	}

	if c.Security.KeyFile != "" {
		if !filepath.IsAbs(c.Security.KeyFile) {
			c.Security.KeyFile = filepath.Clean(c.Security.KeyFile)
		}
	}

	return nil
}

// GetLogLevel returns a zap log level based on the configured string
func (c *Config) GetLogLevel() zap.AtomicLevel {
	level := zap.NewAtomicLevel()
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		level.SetLevel(zap.DebugLevel)
	case "info":
		level.SetLevel(zap.InfoLevel)
	case "warn":
		level.SetLevel(zap.WarnLevel)
	case "error":
		level.SetLevel(zap.ErrorLevel)
	default:
		level.SetLevel(zap.InfoLevel)
	}
	return level
}

// IsDevelopment returns true if the environment is set to development
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == "development"
}
