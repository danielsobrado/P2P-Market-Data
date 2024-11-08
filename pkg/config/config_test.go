 
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	configContent := []byte(`
		environment: production
		log_level: debug
		p2p:
		port: 9001
		max_peers: 100
		min_peers: 10
		peer_timeout: 1m
		validation_quorum: 0.75
		scripts:
		script_dir: test_scripts
		max_exec_time: 10m
		max_memory_mb: 1024
		scheduler:
		max_concurrent: 5
		default_interval: 1m
		retry_attempts: 3
		security:
		min_reputation_score: 0.6
		max_penalty: 0.8
		token_expiry: 12h
		`)

	err := os.WriteFile(configPath, configContent, 0644)
	require.NoError(t, err)

	// Test successful config loading
	t.Run("LoadValidConfig", func(t *testing.T) {
		cfg, err := Load(configPath)
		require.NoError(t, err)
		assert.NotNil(t, cfg)

		// Verify loaded values
		assert.Equal(t, "production", cfg.Environment)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, 9001, cfg.P2P.Port)
		assert.Equal(t, 100, cfg.P2P.MaxPeers)
		assert.Equal(t, time.Minute, cfg.P2P.PeerTimeout)
	})

	// Test environment variable override
	t.Run("EnvironmentOverride", func(t *testing.T) {
		os.Setenv("P2P_LOG_LEVEL", "error")
		defer os.Unsetenv("P2P_LOG_LEVEL")

		cfg, err := Load(configPath)
		require.NoError(t, err)
		assert.Equal(t, "error", cfg.LogLevel)
	})

	// Test invalid config file
	t.Run("InvalidConfig", func(t *testing.T) {
		invalidPath := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(invalidPath, []byte("invalid: [yaml: syntax"), 0644)
		require.NoError(t, err)

		cfg, err := Load(invalidPath)
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	// Test missing config file falls back to defaults
	t.Run("DefaultValues", func(t *testing.T) {
		cfg, err := Load("nonexistent.yaml")
		require.NoError(t, err)
		assert.NotNil(t, cfg)

		// Check default values
		assert.Equal(t, "development", cfg.Environment)
		assert.Equal(t, "info", cfg.LogLevel)
		assert.Equal(t, 9000, cfg.P2P.Port)
	})
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		modifyConfig func(*Config)
		wantErr     bool
		errSubstr   string
	}{
		{
			name: "ValidConfig",
			modifyConfig: func(c *Config) {},
			wantErr: false,
		},
		{
			name: "InvalidPort",
			modifyConfig: func(c *Config) {
				c.P2P.Port = -1
			},
			wantErr: true,
			errSubstr: "invalid port number",
		},
		{
			name: "InvalidPeerCount",
			modifyConfig: func(c *Config) {
				c.P2P.MaxPeers = 5
				c.P2P.MinPeers = 10
			},
			wantErr: true,
			errSubstr: "max_peers",
		},
		{
			name: "InvalidQuorum",
			modifyConfig: func(c *Config) {
				c.P2P.ValidationQuorum = 1.5
			},
			wantErr: true,
			errSubstr: "validation_quorum",
		},
		{
			name: "InvalidMemory",
			modifyConfig: func(c *Config) {
				c.Scripts.MaxMemoryMB = -1
			},
			wantErr: true,
			errSubstr: "max_memory_mb",
		},
		{
			name: "InvalidReputationScore",
			modifyConfig: func(c *Config) {
				c.Security.MinReputationScore = 1.5
			},
			wantErr: true,
			errSubstr: "min_reputation_score",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Environment: "test",
				LogLevel: "info",
				P2P: P2PConfig{
					Port: 9000,
					MaxPeers: 50,
					MinPeers: 3,
					ValidationQuorum: 0.66,
				},
				Scripts: ScriptConfig{
					ScriptDir: "scripts",
					MaxMemoryMB: 512,
				},
				Security: SecurityConfig{
					MinReputationScore: 0.5,
					MaxPenalty: 0.8,
				},
			}

			tt.modifyConfig(cfg)
			err := cfg.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		wantLevel string
	}{
		{
			name:      "Debug",
			logLevel:  "debug",
			wantLevel: "debug",
		},
		{
			name:      "Info",
			logLevel:  "info",
			wantLevel: "info",
		},
		{
			name:      "Warn",
			logLevel:  "warn",
			wantLevel: "warn",
		},
		{
			name:      "Error",
			logLevel:  "error",
			wantLevel: "error",
		},
		{
			name:      "Invalid",
			logLevel:  "invalid",
			wantLevel: "info", // defaults to info
		},
		{
			name:      "Empty",
			logLevel:  "",
			wantLevel: "info", // defaults to info
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{LogLevel: tt.logLevel}
			level := cfg.GetLogLevel()
			assert.Equal(t, tt.wantLevel, level.String())
		})
	}
}

func TestIsDevelopment(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        bool
	}{
		{
			name:        "Development",
			environment: "development",
			want:        true,
		},
		{
			name:        "Development Uppercase",
			environment: "DEVELOPMENT",
			want:        true,
		},
		{
			name:        "Production",
			environment: "production",
			want:        false,
		},
		{
			name:        "Staging",
			environment: "staging",
			want:        false,
		},
		{
			name:        "Empty",
			environment: "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Environment: tt.environment}
			assert.Equal(t, tt.want, cfg.IsDevelopment())
		})
	}
}

func TestConfigFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	t.Run("CreateScriptDir", func(t *testing.T) {
		cfg := &Config{
			Scripts: ScriptConfig{
				ScriptDir: filepath.Join(tmpDir, "scripts"),
			},
		}

		err := cfg.validateScripts()
		require.NoError(t, err)

		// Verify directory was created
		info, err := os.Stat(cfg.Scripts.ScriptDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("AbsoluteKeyPath", func(t *testing.T) {
		cfg := &Config{
			Security: SecurityConfig{
				KeyFile: "keys/private.key",
			},
		}

		err := cfg.validateSecurity()
		require.NoError(t, err)

		// Verify path was cleaned
		assert.Equal(t, filepath.Clean("keys/private.key"), cfg.Security.KeyFile)
	})
}

func TestLoadWithEnvVars(t *testing.T) {
	// Setup environment variables
	envVars := map[string]string{
		"P2P_ENVIRONMENT":                    "production",
		"P2P_LOG_LEVEL":                      "debug",
		"P2P_P2P_PORT":                       "9002",
		"P2P_P2P_MAX_PEERS":                  "150",
		"P2P_SCRIPTS_MAX_MEMORY_MB":          "2048",
		"P2P_SECURITY_MIN_REPUTATION_SCORE":  "0.7",
	}

	// Set environment variables
	for k, v := range envVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	// Create minimal config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := []byte(`
		environment: development
		log_level: info
		`)

	err := os.WriteFile(configPath, configContent, 0644)
	require.NoError(t, err)

	// Load config
	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Verify environment variables took precedence
	assert.Equal(t, "production", cfg.Environment)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 9002, cfg.P2P.Port)
	assert.Equal(t, 150, cfg.P2P.MaxPeers)
	assert.Equal(t, 2048, cfg.Scripts.MaxMemoryMB)
	assert.Equal(t, 0.7, cfg.Security.MinReputationScore)
}

func TestValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		modifyConfig func(*Config)
		wantErr     bool
		errSubstr   string
	}{
		{
			name: "ZeroValues",
			modifyConfig: func(c *Config) {
				c.P2P.Port = 0
				c.P2P.MaxPeers = 0
				c.Scripts.MaxMemoryMB = 0
			},
			wantErr:   true,
			errSubstr: "must be positive",
		},
		{
			name: "EmptyRequiredStrings",
			modifyConfig: func(c *Config) {
				c.Scripts.ScriptDir = ""
			},
			wantErr:   true,
			errSubstr: "cannot be empty",
		},
		{
			name: "NegativeValues",
			modifyConfig: func(c *Config) {
				c.Scheduler.RetryAttempts = -1
			},
			wantErr:   true,
			errSubstr: "cannot be negative",
		},
		{
			name: "ExtremeValues",
			modifyConfig: func(c *Config) {
				c.P2P.ValidationQuorum = 2.0
			},
			wantErr:   true,
			errSubstr: "must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Environment: "test",
				P2P: P2PConfig{
					Port:             9000,
					MaxPeers:         50,
					ValidationQuorum: 0.66,
				},
				Scripts: ScriptConfig{
					ScriptDir:   "scripts",
					MaxMemoryMB: 512,
				},
				Scheduler: SchedConfig{
					RetryAttempts: 3,
				},
			}

			tt.modifyConfig(cfg)
			err := cfg.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}