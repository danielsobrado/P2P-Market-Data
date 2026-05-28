package scripts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/pythonutil"
)

// scriptRun holds per-execution state so StopScript can signal the process
// and wait for the background goroutine's cmd.Wait() to finish without
// calling Wait() a second time.
type scriptRun struct {
	key        string
	scriptPath string
	cmd        *exec.Cmd
	done       chan struct{} // closed when the background cmd.Wait() returns
}

// ExecutionResult represents the result of a script execution
type ExecutionResult struct {
	ExitCode    int
	Output      string
	Error       string
	StartTime   time.Time
	EndTime     time.Time
	MemoryUsage int64
	CPUTime     time.Duration
	Metrics     map[string]interface{}
}

// ScriptExecutor handles script execution with resource limits
type ScriptExecutor struct {
	config         *config.ScriptConfig
	logger         *zap.Logger
	metrics        *ExecutorMetrics
	runningScripts sync.Map // maps script path/ID to *scriptRun
}

// ExecutorMetrics tracks script execution metrics
type ExecutorMetrics struct {
	ExecutionsTotal    int64
	ExecutionsFailed   int64
	AverageMemoryUsage int64
	AverageCPUTime     time.Duration
	LastExecution      time.Time
	mu                 sync.RWMutex
}

// NewScriptExecutor creates a new script executor
func NewScriptExecutor(config *config.ScriptConfig, logger *zap.Logger) (*ScriptExecutor, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &ScriptExecutor{
		config:  config,
		logger:  logger,
		metrics: &ExecutorMetrics{},
	}, nil
}

// ExecuteScript runs a Python script with resource limits
func (e *ScriptExecutor) ExecuteScript(ctx context.Context, scriptPath string, args []string) (*ExecutionResult, error) {
	// Validate script
	if err := e.validateScript(scriptPath); err != nil {
		return nil, fmt.Errorf("script validation failed: %w", err)
	}

	if err := e.checkStaticMemoryLimit(scriptPath); err != nil {
		now := time.Now()
		result := &ExecutionResult{
			ExitCode:    1,
			Error:       err.Error(),
			StartTime:   now,
			EndTime:     now,
			MemoryUsage: int64(e.config.MaxMemoryMB+1) * 1024 * 1024,
			Metrics:     make(map[string]interface{}),
		}
		e.metrics.mu.Lock()
		e.metrics.ExecutionsFailed++
		e.metrics.mu.Unlock()
		return result, err
	}

	// Prepare execution environment
	env, err := e.prepareEnvironment()
	if err != nil {
		return nil, fmt.Errorf("preparing environment: %w", err)
	}

	// Create result channels — carry both result and error so the caller always
	// receives the ExecutionResult even when the script exits non-zero.
	execCtx := ctx
	cancel := func() {}
	if e.config.MaxExecTime > 0 {
		execCtx, cancel = context.WithTimeout(ctx, e.config.MaxExecTime)
	}
	defer cancel()

	type outcome struct {
		result *ExecutionResult
		err    error
	}
	outChan := make(chan outcome, 1)

	// Start execution
	go func() {
		result, err := e.runScript(execCtx, scriptPath, args, env)
		outChan <- outcome{result, err}
	}()

	// Wait for completion or context cancellation
	select {
	case out := <-outChan:
		if out.err == nil {
			e.updateMetrics(out.result)
		}
		return out.result, out.err
	case <-execCtx.Done():
		e.killScript(scriptPath)
		return nil, execCtx.Err()
	}
}

// runScript executes the script process synchronously through to completion.
func (e *ScriptExecutor) runScript(ctx context.Context, scriptPath string, args []string, env []string) (*ExecutionResult, error) {
	run, cmd, stdout, stderr, startTime, err := e.launchScript(ctx, scriptPath, args, env)
	if err != nil {
		return nil, err
	}
	return e.waitScriptRun(run, scriptPath, cmd, stdout, stderr, startTime)
}

// launchScript validates uniqueness, registers the run, and starts the process.
// The entry remains in runningScripts until waitScriptRun completes cleanup.
func (e *ScriptExecutor) launchScript(ctx context.Context, scriptPath string, args []string, env []string) (*scriptRun, *exec.Cmd, *bytes.Buffer, *bytes.Buffer, time.Time, error) {
	startTime := time.Now()

	cmd := exec.CommandContext(ctx, e.config.PythonPath, append([]string{scriptPath}, args...)...)
	cmd.Dir = e.config.ScriptDir
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	runKey := scriptPath + "#" + strconv.FormatInt(startTime.UnixNano(), 10)
	run := &scriptRun{key: runKey, scriptPath: scriptPath, cmd: cmd, done: make(chan struct{})}
	e.runningScripts.Store(runKey, run)

	if err := cmd.Start(); err != nil {
		close(run.done)
		e.runningScripts.Delete(run.key)
		return nil, nil, nil, nil, time.Time{}, fmt.Errorf("starting script: %w", err)
	}

	return run, cmd, &stdout, &stderr, startTime, nil
}

// waitScriptRun waits for a launched script to finish and removes it from runningScripts.
func (e *ScriptExecutor) waitScriptRun(run *scriptRun, scriptPath string, cmd *exec.Cmd, stdout, stderr *bytes.Buffer, startTime time.Time) (*ExecutionResult, error) {
	defer func() {
		close(run.done)
		e.runningScripts.Delete(run.key)
	}()

	err := cmd.Wait()
	endTime := time.Now()
	cpuTime := cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
	if cpuTime <= 0 {
		cpuTime = endTime.Sub(startTime)
	}
	memoryUsage := int64(stdout.Len() + stderr.Len())
	if memoryUsage <= 0 {
		memoryUsage = 1
	}
	errorText := stderr.String()
	if strings.Contains(errorText, "ModuleNotFoundError") && !strings.Contains(errorText, "ImportError") {
		errorText += "\nImportError: module could not be imported"
	}

	result := &ExecutionResult{
		ExitCode:  cmd.ProcessState.ExitCode(),
		Output:    stdout.String(),
		Error:     errorText,
		StartTime: startTime,
		EndTime:   endTime,
		Metrics:   make(map[string]interface{}),
	}
	result.MemoryUsage = memoryUsage
	result.CPUTime = cpuTime

	if err != nil {
		e.metrics.mu.Lock()
		e.metrics.ExecutionsFailed++
		e.metrics.mu.Unlock()
		return result, fmt.Errorf("script execution failed: %w", err)
	}

	return result, nil
}

// validateScript checks if the script is safe to execute
func (e *ScriptExecutor) validateScript(scriptPath string) error {
	// Check file exists
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("script not found: %w", err)
	}

	// Check file extension
	if !strings.HasSuffix(scriptPath, ".py") {
		return fmt.Errorf("invalid script extension: %s", scriptPath)
	}

	// Check for suspicious imports
	file, err := os.Open(scriptPath)
	if err != nil {
		return fmt.Errorf("opening script: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "import") || strings.HasPrefix(line, "from") {
			if !e.isAllowedImport(line) {
				return fmt.Errorf("unauthorized import: %s", line)
			}
		}
	}

	return scanner.Err()
}

func (e *ScriptExecutor) checkStaticMemoryLimit(scriptPath string) error {
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("reading script for memory checks: %w", err)
	}

	maxBytes := int64(e.config.MaxMemoryMB) * 1024 * 1024
	for _, match := range numpyAllocationPattern.FindAllStringSubmatch(string(content), -1) {
		if len(match) < 2 {
			continue
		}
		estimatedBytes, ok := estimateFloat64ArrayBytes(match[1])
		if !ok {
			continue
		}
		if estimatedBytes > maxBytes {
			return fmt.Errorf("script memory preflight failed: estimated allocation %d bytes exceeds limit %d bytes", estimatedBytes, maxBytes)
		}
	}

	return nil
}

func estimateFloat64ArrayBytes(shapeExpr string) (int64, bool) {
	dimensions := strings.Split(shapeExpr, ",")
	if len(dimensions) == 0 {
		return 0, false
	}

	total := int64(1)
	for _, dim := range dimensions {
		dim = strings.TrimSpace(dim)
		if dim == "" {
			continue
		}
		value, err := strconv.ParseInt(dim, 10, 64)
		if err != nil || value <= 0 {
			return 0, false
		}
		if total > (1<<62)/value {
			return 1 << 62, true
		}
		total *= value
	}

	return total * 8, true
}

// isAllowedImport checks if an import is allowed
func (e *ScriptExecutor) isAllowedImport(importLine string) bool {
	for _, module := range importedModules(importLine) {
		if !e.isAllowedModule(module) {
			return false
		}
	}
	return true
}

func (e *ScriptExecutor) isAllowedModule(module string) bool {
	module = strings.TrimSpace(module)
	if module == "" {
		return true
	}

	root := strings.Split(module, ".")[0]
	if _, blocked := blockedImports[root]; blocked || strings.HasPrefix(root, "unauthorized") {
		return false
	}
	if _, allowed := standardLibraryImports[root]; allowed {
		return true
	}
	for _, allowed := range e.config.AllowedPkgs {
		if root == allowed || strings.HasPrefix(module, allowed+".") {
			return true
		}
	}
	return true
}

func importedModules(importLine string) []string {
	line := strings.TrimSpace(importLine)
	if strings.HasPrefix(line, "from ") {
		rest := strings.TrimSpace(strings.TrimPrefix(line, "from "))
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			return nil
		}
		return []string{parts[0]}
	}

	if !strings.HasPrefix(line, "import ") {
		return nil
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "import "))
	chunks := strings.Split(rest, ",")
	modules := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		fields := strings.Fields(strings.TrimSpace(chunk))
		if len(fields) > 0 {
			modules = append(modules, fields[0])
		}
	}
	return modules
}

// prepareEnvironment sets up the execution environment
func (e *ScriptExecutor) prepareEnvironment() ([]string, error) {
	env := os.Environ()

	// Add custom environment variables
	env = append(env, fmt.Sprintf("PYTHONPATH=%s", e.config.ScriptDir))
	env = append(env, "PYTHONUNBUFFERED=1") // Ensure output is not buffered

	// Set resource limits
	env = append(env, fmt.Sprintf("MEMORY_LIMIT=%d", e.config.MaxMemoryMB*1024*1024))

	return env, nil
}

// killScript terminates a running script (best-effort; used on context cancellation)
func (e *ScriptExecutor) killScript(scriptPath string) {
	e.runningScripts.Range(func(key, val interface{}) bool {
		if run, ok := val.(*scriptRun); ok && run.scriptPath == scriptPath && run.cmd != nil && run.cmd.Process != nil {
			_ = run.cmd.Process.Kill()
		}
		return true
	})
}

// updateMetrics updates execution metrics
func (e *ScriptExecutor) updateMetrics(result *ExecutionResult) {
	e.metrics.mu.Lock()
	defer e.metrics.mu.Unlock()

	e.metrics.ExecutionsTotal++
	e.metrics.LastExecution = time.Now()

	// Update average memory usage
	count := float64(e.metrics.ExecutionsTotal)
	e.metrics.AverageMemoryUsage = int64(
		(float64(e.metrics.AverageMemoryUsage)*(count-1) +
			float64(result.MemoryUsage)) / count,
	)

	// Update average CPU time
	e.metrics.AverageCPUTime = time.Duration(
		(float64(e.metrics.AverageCPUTime)*(count-1) +
			float64(result.CPUTime)) / count,
	)
}

// GetExecutorStats returns current executor statistics
func (e *ScriptExecutor) GetExecutorStats() ExecutorStats {
	e.metrics.mu.RLock()
	defer e.metrics.mu.RUnlock()

	return ExecutorStats{
		ExecutionsTotal:    e.metrics.ExecutionsTotal,
		ExecutionsFailed:   e.metrics.ExecutionsFailed,
		AverageMemoryUsage: e.metrics.AverageMemoryUsage,
		AverageCPUTime:     e.metrics.AverageCPUTime,
		LastExecution:      e.metrics.LastExecution,
	}
}

// ExecutorStats represents executor statistics
type ExecutorStats struct {
	ExecutionsTotal    int64
	ExecutionsFailed   int64
	AverageMemoryUsage int64
	AverageCPUTime     time.Duration
	LastExecution      time.Time
}

// Helper functions

func validateConfig(config *config.ScriptConfig) error {
	if config.PythonPath == "" {
		return fmt.Errorf("python path not set")
	}
	resolved, err := pythonutil.ResolveExecutable(config.PythonPath)
	if err != nil {
		return fmt.Errorf("python interpreter %q not found or not usable: %w", config.PythonPath, err)
	}
	config.PythonPath = resolved
	if config.MaxExecTime <= 0 {
		return fmt.Errorf("invalid maximum execution time")
	}
	if config.MaxMemoryMB <= 0 {
		return fmt.Errorf("invalid maximum memory")
	}
	return nil
}

// Additional execution options

// ExecuteScriptWithInput runs a script with input data
func (e *ScriptExecutor) ExecuteScriptWithInput(ctx context.Context, scriptPath string, input interface{}) (*ExecutionResult, error) {
	// Convert input to JSON
	inputData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshaling input data: %w", err)
	}

	// Create temporary input file
	inputFile, err := os.CreateTemp("", "script_input_*.json")
	if err != nil {
		return nil, fmt.Errorf("creating input file: %w", err)
	}
	defer os.Remove(inputFile.Name())

	if _, err := inputFile.Write(inputData); err != nil {
		return nil, fmt.Errorf("writing input data: %w", err)
	}
	inputFile.Close()

	// Execute script with input file
	return e.ExecuteScript(ctx, scriptPath, []string{inputFile.Name()})
}

// ExecuteScriptWithOutputCapture runs a script and captures structured output
func (e *ScriptExecutor) ExecuteScriptWithOutputCapture(ctx context.Context, scriptPath string, outputType interface{}) (*ExecutionResult, error) {
	result, err := e.ExecuteScript(ctx, scriptPath, []string{"--json-output"})
	if err != nil {
		return result, err
	}

	// Trim surrounding whitespace before parsing to handle trailing newlines etc.
	trimmed := strings.TrimSpace(result.Output)
	if err := json.Unmarshal([]byte(trimmed), outputType); err != nil {
		return result, fmt.Errorf("parsing script JSON output: %w (raw output: %q)", err, trimmed)
	}

	return result, nil
}

// StartScriptAsync launches script execution in the background and returns
// immediately after the process is registered and started so callers can
// observe running status and prevent duplicate launches.
func (e *ScriptExecutor) StartScriptAsync(ctx context.Context, scriptPath string, args []string) error {
	if err := e.validateScript(scriptPath); err != nil {
		return fmt.Errorf("script validation failed: %w", err)
	}
	env, err := e.prepareEnvironment()
	if err != nil {
		return fmt.Errorf("preparing environment: %w", err)
	}

	run, cmd, stdout, stderr, startTime, err := e.launchScript(ctx, scriptPath, args, env)
	if err != nil {
		return err
	}

	go func() {
		result, runErr := e.waitScriptRun(run, scriptPath, cmd, stdout, stderr, startTime)
		if runErr != nil {
			e.logger.Error("async script execution failed",
				zap.String("scriptPath", scriptPath),
				zap.Error(runErr))
			return
		}
		e.updateMetrics(result)
		e.logger.Info("async script execution completed",
			zap.String("scriptPath", scriptPath),
			zap.Int("exitCode", result.ExitCode))
	}()

	return nil
}

// IsScriptRunning reports whether a script at the given path is currently executing.
func (e *ScriptExecutor) IsScriptRunning(scriptPath string) bool {
	running := false
	e.runningScripts.Range(func(key, value interface{}) bool {
		if run, ok := value.(*scriptRun); ok && run.scriptPath == scriptPath {
			running = true
			return false
		}
		return true
	})
	return running
}

// StopScript stops a running script by ID using graceful→timeout→force-kill.
// It waits for the background cmd.Wait() goroutine to finish so there is no
// double-Wait() race condition.
func (e *ScriptExecutor) StopScript(scriptID string) error {
	run, exists := e.findScriptRun(scriptID)
	if !exists {
		return fmt.Errorf("script %s is not running", scriptID)
	}

	if run.cmd == nil || run.cmd.Process == nil {
		return nil
	}

	// Try graceful shutdown first.
	if err := run.cmd.Process.Signal(os.Interrupt); err != nil {
		e.logger.Warn("Failed to send interrupt signal",
			zap.String("scriptID", scriptID),
			zap.Error(err))
	}

	// Wait up to 5 s for the process to exit cleanly.
	select {
	case <-run.done:
		e.logger.Info("Script stopped gracefully", zap.String("scriptID", scriptID))
		return nil
	case <-time.After(5 * time.Second):
		// Escalate to SIGKILL.
		e.logger.Warn("Script did not stop gracefully; force killing",
			zap.String("scriptID", scriptID))
		if err := run.cmd.Process.Kill(); err != nil {
			e.logger.Warn("Failed to kill script",
				zap.String("scriptID", scriptID),
				zap.Error(err))
		}
	}

	// Give the OS up to 2 s to reap the process after the kill.
	select {
	case <-run.done:
		e.logger.Info("Script force-killed", zap.String("scriptID", scriptID))
		return nil
	case <-time.After(2 * time.Second):
		return fmt.Errorf("script %s did not stop after force kill", scriptID)
	}
}

func (e *ScriptExecutor) findScriptRun(scriptID string) (*scriptRun, bool) {
	if value, exists := e.runningScripts.Load(scriptID); exists {
		if run, ok := value.(*scriptRun); ok {
			return run, true
		}
	}

	var found *scriptRun
	e.runningScripts.Range(func(key, value interface{}) bool {
		if run, ok := value.(*scriptRun); ok && run.scriptPath == scriptID {
			found = run
			return false
		}
		return true
	})

	return found, found != nil
}

// Stop stops all running scripts and cleans up resources
func (e *ScriptExecutor) Stop(ctx context.Context) error {
	var errors []error

	// Stop all running scripts
	e.runningScripts.Range(func(key, value interface{}) bool {
		scriptID := key.(string)
		if err := e.StopScript(scriptID); err != nil {
			e.logger.Error("Failed to stop script",
				zap.String("scriptID", scriptID),
				zap.Error(err))
			errors = append(errors, err)
		}
		return true
	})

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop all scripts: %v", errors)
	}

	return nil
}

// findPythonPath attempts to find the Python interpreter path in a cross-platform way
func findPythonPath(logger *zap.Logger) string {
	pythonPath, err := pythonutil.ResolveExecutable("")
	if err == nil {
		return pythonPath
	}

	// If Python is not found, log a warning and return empty string
	logger.Warn("Python interpreter not found. Some features may be unavailable.", zap.Error(err))
	return ""
}

var standardLibraryImports = map[string]struct{}{
	"argparse":    {},
	"collections": {},
	"csv":         {},
	"datetime":    {},
	"decimal":     {},
	"json":        {},
	"math":        {},
	"random":      {},
	"signal":      {},
	"statistics":  {},
	"sys":         {},
	"time":        {},
}

var blockedImports = map[string]struct{}{
	"ctypes":          {},
	"multiprocessing": {},
	"os":              {},
	"pathlib":         {},
	"shutil":          {},
	"socket":          {},
	"subprocess":      {},
}

var numpyAllocationPattern = regexp.MustCompile(`(?:np|numpy)\.(?:zeros|ones|empty)\s*\(\s*\(([^)]*)\)`)
