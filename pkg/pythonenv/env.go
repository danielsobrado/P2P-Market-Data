package pythonenv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EnvConfig holds configuration for Python environment
type EnvConfig struct {
	BaseDir       string            // Base directory for virtual environments
	PythonVersion string            // Required Python version
	Requirements  []string          // List of required packages
	AllowedHosts  []string          // Allowed pip hosts
	ExtraIndex    []string          // Additional package indexes
	EnvVars       map[string]string // Environment variables
	Timeout       time.Duration     // Operation timeout
}

// Environment represents a Python virtual environment
type Environment struct {
	ID           string
	Path         string
	PythonPath   string
	PipPath      string
	Requirements map[string]string
	Created      time.Time
	LastUpdated  time.Time
	Status       EnvStatus
	Dependencies []Dependency
}

// EnvStatus represents the state of an environment
type EnvStatus string

const (
	EnvStatusCreating   EnvStatus = "creating"
	EnvStatusReady      EnvStatus = "ready"
	EnvStatusUpdating   EnvStatus = "updating"
	EnvStatusError      EnvStatus = "error"
	EnvStatusDestroying EnvStatus = "destroying"
)

// Dependency represents a Python package dependency
type Dependency struct {
	Name         string
	Version      string
	Required     bool
	Installed    bool
	InstallTime  time.Time
	Dependencies []string
}

// Manager manages Python virtual environments
type Manager struct {
	config  *EnvConfig
	logger  *zap.Logger
	envs    map[string]*Environment
	metrics *EnvMetrics
	mu      sync.RWMutex
}

// EnvMetrics tracks environment metrics
type EnvMetrics struct {
	EnvsCreated       int64
	EnvsDestroyed     int64
	PackagesInstalled int64
	PackagesFailed    int64
	AverageCreateTime time.Duration
	AverageUpdateTime time.Duration
	LastOperation     time.Time
	mu                sync.RWMutex
}

// NewManager creates a new Python environment manager
func NewManager(config *EnvConfig, logger *zap.Logger) (*Manager, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &Manager{
		config:  config,
		logger:  logger,
		envs:    make(map[string]*Environment),
		metrics: &EnvMetrics{},
	}, nil
}

// CreateEnvironment creates a new Python virtual environment
func (m *Manager) CreateEnvironment(ctx context.Context, name string) (*Environment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	start := time.Now()

	// Check if environment already exists
	if _, exists := m.envs[name]; exists {
		return nil, fmt.Errorf("environment already exists: %s", name)
	}

	// Create environment directory
	envPath := filepath.Join(m.config.BaseDir, name)
	if err := os.MkdirAll(envPath, 0755); err != nil {
		return nil, fmt.Errorf("creating environment directory: %w", err)
	}

	// Create new environment
	env := &Environment{
		ID:           name,
		Path:         envPath,
		Requirements: make(map[string]string),
		Created:      time.Now(),
		LastUpdated:  time.Now(),
		Status:       EnvStatusCreating,
	}

	// Create virtual environment
	if err := m.createVirtualEnv(ctx, env); err != nil {
		os.RemoveAll(envPath)
		return nil, err
	}

	// Set paths
	env.PythonPath = filepath.Join(envPath, getBinDir(), "python")
	env.PipPath = filepath.Join(envPath, getBinDir(), "pip")

	// Install base requirements
	if err := m.installBaseRequirements(ctx, env); err != nil {
		os.RemoveAll(envPath)
		return nil, err
	}

	env.Status = EnvStatusReady
	m.envs[name] = env

	// Update metrics
	m.metrics.mu.Lock()
	m.metrics.EnvsCreated++
	m.metrics.AverageCreateTime = updateAverage(
		m.metrics.AverageCreateTime,
		time.Since(start),
		m.metrics.EnvsCreated,
	)
	m.metrics.LastOperation = time.Now()
	m.metrics.mu.Unlock()

	m.logger.Info("Environment created",
		zap.String("id", name),
		zap.String("path", envPath),
		zap.Duration("duration", time.Since(start)))

	return env, nil
}

// GetEnvironment retrieves an environment by name
func (m *Manager) GetEnvironment(name string) (*Environment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	env, exists := m.envs[name]
	if !exists {
		return nil, fmt.Errorf("environment not found: %s", name)
	}

	return env, nil
}

// UpdateEnvironment updates an environment's dependencies
func (m *Manager) UpdateEnvironment(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	start := time.Now()

	env, exists := m.envs[name]
	if !exists {
		return fmt.Errorf("environment not found: %s", name)
	}

	env.Status = EnvStatusUpdating

	// Update pip
	if err := m.runPip(ctx, env, "install", "--upgrade", "pip"); err != nil {
		env.Status = EnvStatusError
		return fmt.Errorf("updating pip: %w", err)
	}

	// Update requirements
	if err := m.installBaseRequirements(ctx, env); err != nil {
		env.Status = EnvStatusError
		return err
	}

	env.LastUpdated = time.Now()
	env.Status = EnvStatusReady

	// Update metrics
	m.metrics.mu.Lock()
	m.metrics.AverageUpdateTime = updateAverage(
		m.metrics.AverageUpdateTime,
		time.Since(start),
		int64(len(m.envs)),
	)
	m.metrics.LastOperation = time.Now()
	m.metrics.mu.Unlock()

	m.logger.Info("Environment updated",
		zap.String("id", name),
		zap.Duration("duration", time.Since(start)))

	return nil
}

// DestroyEnvironment removes a Python environment
func (m *Manager) DestroyEnvironment(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	env, exists := m.envs[name]
	if !exists {
		return fmt.Errorf("environment not found: %s", name)
	}

	env.Status = EnvStatusDestroying

	// Remove environment directory
	if err := os.RemoveAll(env.Path); err != nil {
		env.Status = EnvStatusError
		return fmt.Errorf("removing environment directory: %w", err)
	}

	delete(m.envs, name)

	// Update metrics
	m.metrics.mu.Lock()
	m.metrics.EnvsDestroyed++
	m.metrics.LastOperation = time.Now()
	m.metrics.mu.Unlock()

	m.logger.Info("Environment destroyed",
		zap.String("id", name))

	return nil
}

// InstallPackage installs a Python package in an environment
func (m *Manager) InstallPackage(ctx context.Context, name string, pkg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	env, exists := m.envs[name]
	if !exists {
		return fmt.Errorf("environment not found: %s", name)
	}

	if err := m.runPip(ctx, env, "install", pkg); err != nil {
		m.metrics.mu.Lock()
		m.metrics.PackagesFailed++
		m.metrics.LastOperation = time.Now()
		m.metrics.mu.Unlock()
		return fmt.Errorf("installing package %s: %w", pkg, err)
	}

	m.metrics.mu.Lock()
	m.metrics.PackagesInstalled++
	m.metrics.LastOperation = time.Now()
	m.metrics.mu.Unlock()

	return nil
}

// Private methods

func (m *Manager) createVirtualEnv(ctx context.Context, env *Environment) error {
	args := []string{"-m", "venv", env.Path}
	if m.config.PythonVersion != "" {
		args = append([]string{"-m", "venv", "--python=" + m.config.PythonVersion}, args...)
	}

	cmd := exec.CommandContext(ctx, "python3", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("creating virtual environment: %w", err)
	}

	return nil
}

func (m *Manager) installBaseRequirements(ctx context.Context, env *Environment) error {
	if len(m.config.Requirements) == 0 {
		return nil
	}

	args := []string{"install", "--no-cache-dir"}

	// Add allowed hosts
	for _, host := range m.config.AllowedHosts {
		args = append(args, "--trusted-host", host)
	}

	// Add extra indexes
	for _, index := range m.config.ExtraIndex {
		args = append(args, "--extra-index-url", index)
	}

	args = append(args, m.config.Requirements...)

	if err := m.runPip(ctx, env, args...); err != nil {
		return fmt.Errorf("installing base requirements: %w", err)
	}

	return nil
}

func (m *Manager) runPip(ctx context.Context, env *Environment, args ...string) error {
	cmd := exec.CommandContext(ctx, env.PipPath, args...)
	cmd.Env = m.buildEnvVars()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pip command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (m *Manager) buildEnvVars() []string {
	vars := os.Environ()
	for k, v := range m.config.EnvVars {
		vars = append(vars, fmt.Sprintf("%s=%s", k, v))
	}
	return vars
}

// Helper functions

func validateConfig(config *EnvConfig) error {
	if config.BaseDir == "" {
		return fmt.Errorf("base directory not specified")
	}
	if config.Timeout <= 0 {
		return fmt.Errorf("invalid timeout")
	}
	return nil
}

func getBinDir() string {
	if runtime.GOOS == "windows" {
		return "Scripts"
	}
	return "bin"
}

func updateAverage(current time.Duration, new time.Duration, count int64) time.Duration {
	if count == 1 {
		return new
	}
	return time.Duration(int64(current)*(count-1)/count + int64(new)/count)
}

// Additional functionality

// ListEnvironments returns all environments
func (m *Manager) ListEnvironments() []*Environment {
	m.mu.RLock()
	defer m.mu.RUnlock()

	envs := make([]*Environment, 0, len(m.envs))
	for _, env := range m.envs {
		envs = append(envs, env)
	}
	return envs
}

// GetStats returns environment manager statistics
func (m *Manager) GetStats() EnvStats {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	return EnvStats{
		EnvsCreated:       m.metrics.EnvsCreated,
		EnvsDestroyed:     m.metrics.EnvsDestroyed,
		PackagesInstalled: m.metrics.PackagesInstalled,
		PackagesFailed:    m.metrics.PackagesFailed,
		AverageCreateTime: m.metrics.AverageCreateTime,
		AverageUpdateTime: m.metrics.AverageUpdateTime,
		LastOperation:     m.metrics.LastOperation,
	}
}

// EnvStats represents environment manager statistics
type EnvStats struct {
	EnvsCreated       int64
	EnvsDestroyed     int64
	PackagesInstalled int64
	PackagesFailed    int64
	AverageCreateTime time.Duration
	AverageUpdateTime time.Duration
	LastOperation     time.Time
}
