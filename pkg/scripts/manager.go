package scripts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
)

// ScriptMetadata contains script information
type ScriptMetadata struct {
	ID           string
	Name         string
	Description  string
	Author       string
	Version      string
	Dependencies []string
	Created      time.Time
	Updated      time.Time
	Hash         string
	Size         int64
}

// ScriptManager handles script storage and management
type ScriptManager struct {
	Config    *config.ScriptConfig
	Executor  *ScriptExecutor
	logger    *zap.Logger
	scripts   map[string]*ScriptMetadata
	mu        sync.RWMutex
	isRunning bool
}

// NewScriptManager creates a new script manager
func NewScriptManager(config *config.ScriptConfig, logger *zap.Logger) (*ScriptManager, error) {
	executor, err := NewScriptExecutor(config, logger)
	if err != nil {
		return nil, fmt.Errorf("creating executor: %w", err)
	}

	manager := &ScriptManager{
		Config:   config,
		Executor: executor,
		logger:   logger,
		scripts:  make(map[string]*ScriptMetadata),
	}

	// Initialize script directory
	if err := manager.initializeScriptDirectory(); err != nil {
		return nil, fmt.Errorf("initializing script directory: %w", err)
	}

	// Load existing scripts
	if err := manager.loadScripts(); err != nil {
		return nil, fmt.Errorf("loading scripts: %w", err)
	}

	return manager, nil
}

// Start initializes the ScriptManager and starts any background processes
func (m *ScriptManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Starting Script Manager")

	// Initialize executor if needed
	if m.Executor == nil {
		executor, err := NewScriptExecutor(m.Config, m.logger)
		if err != nil {
			return fmt.Errorf("creating executor: %w", err)
		}
		m.Executor = executor
	}

	// Load existing scripts
	if err := m.loadScripts(); err != nil {
		return fmt.Errorf("loading scripts: %w", err)
	}

	m.isRunning = true

	return nil
}

// Stop gracefully shuts down the ScriptManager, performing any necessary cleanup.
func (m *ScriptManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Stopping Script Manager")

	// Stop the executor
	if m.Executor != nil {
		if err := m.Executor.Stop(ctx); err != nil {
			m.logger.Error("Error stopping executor", zap.Error(err))
			return fmt.Errorf("stopping executor: %w", err)
		}
	}

	m.isRunning = false

	return nil
}

// AddScript adds a new script
func (m *ScriptManager) AddScript(name string, content []byte, metadata *ScriptMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate script content
	if err := m.Executor.validateScript(name); err != nil {
		return fmt.Errorf("invalid script: %w", err)
	}

	// Generate script ID and hash
	hash := sha256.Sum256(content)
	scriptID := hex.EncodeToString(hash[:])

	// Check for duplicates
	if _, exists := m.scripts[scriptID]; exists {
		return fmt.Errorf("script already exists: %s", name)
	}

	// Create script file
	scriptPath := filepath.Join(m.Config.ScriptDir, name)
	if err := os.WriteFile(scriptPath, content, 0644); err != nil {
		return fmt.Errorf("writing script file: %w", err)
	}

	// Update metadata
	metadata.ID = scriptID
	metadata.Hash = hex.EncodeToString(hash[:])
	metadata.Size = int64(len(content))
	metadata.Created = time.Now()
	metadata.Updated = metadata.Created

	m.scripts[scriptID] = metadata

	m.logger.Info("Script added",
		zap.String("id", scriptID),
		zap.String("name", name),
		zap.Int64("size", metadata.Size))

	return nil
}

// GetScript retrieves a script by ID
func (m *ScriptManager) GetScript(scriptID string) (*ScriptMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	script, exists := m.scripts[scriptID]
	if !exists {
		return nil, fmt.Errorf("script not found: %s", scriptID)
	}

	return script, nil
}

// UpdateScript updates an existing script
func (m *ScriptManager) UpdateScript(scriptID string, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	script, exists := m.scripts[scriptID]
	if !exists {
		return fmt.Errorf("script not found: %s", scriptID)
	}

	// Validate new content
	if err := m.Executor.validateScript(script.Name); err != nil {
		return fmt.Errorf("invalid script content: %w", err)
	}

	// Generate new hash
	hash := sha256.Sum256(content)
	newHash := hex.EncodeToString(hash[:])

	// Update script file
	scriptPath := filepath.Join(m.Config.ScriptDir, script.Name)
	if err := os.WriteFile(scriptPath, content, 0644); err != nil {
		return fmt.Errorf("writing updated script: %w", err)
	}

	// Update metadata
	script.Hash = newHash
	script.Size = int64(len(content))
	script.Updated = time.Now()

	m.logger.Info("Script updated",
		zap.String("id", scriptID),
		zap.String("name", script.Name),
		zap.String("hash", newHash))

	return nil
}

// DeleteScript removes a script
func (m *ScriptManager) DeleteScript(scriptID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	script, exists := m.scripts[scriptID]
	if !exists {
		return fmt.Errorf("script not found: %s", scriptID)
	}

	// Remove script file
	scriptPath := filepath.Join(m.Config.ScriptDir, script.Name)
	if err := os.Remove(scriptPath); err != nil {
		return fmt.Errorf("removing script file: %w", err)
	}

	// Remove from metadata
	delete(m.scripts, scriptID)

	m.logger.Info("Script deleted",
		zap.String("id", scriptID),
		zap.String("name", script.Name))

	return nil
}

// ListScripts returns all available scripts
func (m *ScriptManager) ListScripts() []*ScriptMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	scripts := make([]*ScriptMetadata, 0, len(m.scripts))
	for _, script := range m.scripts {
		scripts = append(scripts, script)
	}
	return scripts
}

// ExecuteScript executes a script by ID
func (m *ScriptManager) ExecuteScript(ctx context.Context, scriptID string, args []string) (*ExecutionResult, error) {
	m.mu.RLock()
	script, exists := m.scripts[scriptID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("script not found: %s", scriptID)
	}

	scriptPath := filepath.Join(m.Config.ScriptDir, script.Name)
	return m.Executor.ExecuteScript(ctx, scriptPath, args)
}

// Add this method to ScriptManager struct
func (m *ScriptManager) GetConfig() *config.ScriptConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Config
}

// Private methods

func (m *ScriptManager) initializeScriptDirectory() error {
	// Create script directory if it doesn't exist
	if err := os.MkdirAll(m.Config.ScriptDir, 0755); err != nil {
		return fmt.Errorf("creating script directory: %w", err)
	}

	return nil
}

func (m *ScriptManager) loadScripts() error {
	return filepath.Walk(m.Config.ScriptDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".py") {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading script file: %w", err)
			}

			hash := sha256.Sum256(content)
			scriptID := hex.EncodeToString(hash[:])

			metadata := &ScriptMetadata{
				ID:      scriptID,
				Name:    info.Name(),
				Hash:    hex.EncodeToString(hash[:]),
				Size:    info.Size(),
				Created: info.ModTime(),
				Updated: info.ModTime(),
			}

			m.scripts[scriptID] = metadata
		}

		return nil
	})
}

// Additional functionality

// ValidateDependencies checks if all required dependencies are available
func (m *ScriptManager) ValidateDependencies(scriptID string) error {
	m.mu.RLock()
	script, exists := m.scripts[scriptID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("script not found: %s", scriptID)
	}

	for _, dep := range script.Dependencies {
		// Create temporary script to test import
		testScript := fmt.Sprintf("import %s", dep)
		tmpFile := filepath.Join(m.Config.ScriptDir, "_test_import.py")

		if err := os.WriteFile(tmpFile, []byte(testScript), 0644); err != nil {
			return fmt.Errorf("creating test script: %w", err)
		}
		defer os.Remove(tmpFile)

		// Try to execute the test import
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := m.Executor.ExecuteScript(ctx, tmpFile, nil); err != nil {
			return fmt.Errorf("dependency not available: %s", dep)
		}
	}

	return nil
}

// BackupScript creates a backup of a script
func (m *ScriptManager) BackupScript(scriptID string) error {
	m.mu.RLock()
	script, exists := m.scripts[scriptID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("script not found: %s", scriptID)
	}

	sourcePath := filepath.Join(m.Config.ScriptDir, script.Name)
	backupPath := filepath.Join(m.Config.ScriptDir, "backups",
		fmt.Sprintf("%s_%s.py", script.ID, time.Now().Format("20060102_150405")))

	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return fmt.Errorf("creating backup directory: %w", err)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer source.Close()

	backup, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("creating backup file: %w", err)
	}
	defer backup.Close()

	if _, err := io.Copy(backup, source); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	m.logger.Info("Script backup created",
		zap.String("id", scriptID),
		zap.String("backup", backupPath))

	return nil
}

// VerifyScriptIntegrity checks if a script has been modified
func (m *ScriptManager) VerifyScriptIntegrity(scriptID string) (bool, error) {
	m.mu.RLock()
	script, exists := m.scripts[scriptID]
	m.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("script not found: %s", scriptID)
	}

	scriptPath := filepath.Join(m.Config.ScriptDir, script.Name)
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return false, fmt.Errorf("reading script file: %w", err)
	}

	hash := sha256.Sum256(content)
	currentHash := hex.EncodeToString(hash[:])

	return currentHash == script.Hash, nil
}

// GetScriptHistory returns script modification history
func (m *ScriptManager) GetScriptHistory(scriptID string) ([]ScriptHistoryEntry, error) {
	m.mu.RLock()
	script, exists := m.scripts[scriptID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("script not found: %s", scriptID)
	}

	backupDir := filepath.Join(m.Config.ScriptDir, "backups")
	pattern := fmt.Sprintf("%s_*.py", script.ID)

	matches, err := filepath.Glob(filepath.Join(backupDir, pattern))
	if err != nil {
		return nil, fmt.Errorf("finding backup files: %w", err)
	}

	history := make([]ScriptHistoryEntry, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		content, err := os.ReadFile(match)
		if err != nil {
			continue
		}

		hash := sha256.Sum256(content)
		history = append(history, ScriptHistoryEntry{
			Timestamp: info.ModTime(),
			Hash:      hex.EncodeToString(hash[:]),
			Size:      info.Size(),
		})
	}

	return history, nil
}

// ScriptHistoryEntry represents a historical version of a script
type ScriptHistoryEntry struct {
	Timestamp time.Time
	Hash      string
	Size      int64
}

// isRunning
func (m *ScriptManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}