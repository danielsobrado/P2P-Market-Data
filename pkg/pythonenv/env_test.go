package pythonenv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestManager(t *testing.T) (*Manager, string) {
	tempDir := t.TempDir()
	logger := zaptest.NewLogger(t)

	config := &EnvConfig{
		BaseDir:       tempDir,
		PythonVersion: "python3",
		Requirements:  []string{"requests"},
		Timeout:       time.Minute,
		EnvVars:       map[string]string{"TEST_VAR": "test_value"},
	}

	manager, err := NewManager(config, logger)
	require.NoError(t, err)

	return manager, tempDir
}

func TestEnvironmentCreation(t *testing.T) {
	manager, _ := setupTestManager(t)

	t.Run("CreateEnvironment", func(t *testing.T) {
		env, err := manager.CreateEnvironment(context.Background(), "test-env")
		require.NoError(t, err)
		assert.NotNil(t, env)
		assert.Equal(t, "test-env", env.ID)
		assert.Equal(t, EnvStatusReady, env.Status)

		// Verify directory structure
		assert.FileExists(t, env.PythonPath)
		assert.FileExists(t, env.PipPath)
	})

	t.Run("DuplicateEnvironment", func(t *testing.T) {
		_, err := manager.CreateEnvironment(context.Background(), "test-env")
		assert.Error(t, err)
	})

	t.Run("InvalidName", func(t *testing.T) {
		_, err := manager.CreateEnvironment(context.Background(), "")
		assert.Error(t, err)
	})
}

func TestEnvironmentOperations(t *testing.T) {
	manager, _ := setupTestManager(t)

	t.Run("GetEnvironment", func(t *testing.T) {
		// Create test environment
		_, err := manager.CreateEnvironment(context.Background(), "get-test")
		require.NoError(t, err)

		// Retrieve environment
		env, err := manager.GetEnvironment("get-test")
		require.NoError(t, err)
		assert.NotNil(t, env)
		assert.Equal(t, "get-test", env.ID)

		// Try to get non-existent environment
		_, err = manager.GetEnvironment("non-existent")
		assert.Error(t, err)
	})

	t.Run("UpdateEnvironment", func(t *testing.T) {
		env, err := manager.CreateEnvironment(context.Background(), "update-test")
		require.NoError(t, err)

		// Update environment
		err = manager.UpdateEnvironment(context.Background(), env.ID)
		require.NoError(t, err)

		// Verify update
		updatedEnv, err := manager.GetEnvironment(env.ID)
		require.NoError(t, err)
		assert.Equal(t, EnvStatusReady, updatedEnv.Status)
		assert.True(t, updatedEnv.LastUpdated.After(updatedEnv.Created))
	})

	t.Run("DestroyEnvironment", func(t *testing.T) {
		env, err := manager.CreateEnvironment(context.Background(), "destroy-test")
		require.NoError(t, err)

		// Verify environment exists
		assert.DirExists(t, env.Path)

		// Destroy environment
		err = manager.DestroyEnvironment(env.ID)
		require.NoError(t, err)

		// Verify environment is removed
		assert.NoDirExists(t, env.Path)
		_, err = manager.GetEnvironment(env.ID)
		assert.Error(t, err)
	})
}

func TestPackageManagement(t *testing.T) {
	manager, _ := setupTestManager(t)

	t.Run("InstallPackage", func(t *testing.T) {
		env, err := manager.CreateEnvironment(context.Background(), "package-test")
		require.NoError(t, err)

		// Install a package
		err = manager.InstallPackage(context.Background(), env.ID, "requests==2.26.0")
		require.NoError(t, err)

		// Verify installation
		cmd := exec.Command(env.PythonPath, "-c", "import requests; print(requests.__version__)")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), "2.26.0")
	})

	t.Run("InvalidPackage", func(t *testing.T) {
		env, err := manager.CreateEnvironment(context.Background(), "invalid-package-test")
		require.NoError(t, err)

		// Try to install non-existent package
		err = manager.InstallPackage(context.Background(), env.ID, "non-existent-package")
		assert.Error(t, err)
	})
}

func TestEnvironmentIsolation(t *testing.T) {
	manager, _ := setupTestManager(t)

	t.Run("PackageIsolation", func(t *testing.T) {
		// Create two environments
		env1, err := manager.CreateEnvironment(context.Background(), "isolation-test-1")
		require.NoError(t, err)

		env2, err := manager.CreateEnvironment(context.Background(), "isolation-test-2")
		require.NoError(t, err)

		// Install different package versions
		err = manager.InstallPackage(context.Background(), env1.ID, "requests==2.26.0")
		require.NoError(t, err)

		err = manager.InstallPackage(context.Background(), env2.ID, "requests==2.25.0")
		require.NoError(t, err)

		// Verify different versions
		checkVersion := func(pythonPath string) string {
			cmd := exec.Command(pythonPath, "-c", "import requests; print(requests.__version__)")
			output, err := cmd.Output()
			require.NoError(t, err)
			return strings.TrimSpace(string(output))
		}

		version1 := checkVersion(env1.PythonPath)
		version2 := checkVersion(env2.PythonPath)

		assert.NotEqual(t, version1, version2)
	})
}

func TestConcurrentOperations(t *testing.T) {
	manager, _ := setupTestManager(t)

	t.Run("ConcurrentCreation", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 5)
		envs := make(chan *Environment, 5)

		// Create multiple environments concurrently
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				env, err := manager.CreateEnvironment(context.Background(),
					fmt.Sprintf("concurrent-test-%d", id))
				if err != nil {
					errors <- err
					return
				}
				envs <- env
			}(i)
		}

		wg.Wait()
		close(errors)
		close(envs)

		// Check for errors
		for err := range errors {
			assert.NoError(t, err)
		}

		// Verify environments
		createdEnvs := make([]*Environment, 0)
		for env := range envs {
			createdEnvs = append(createdEnvs, env)
		}
		assert.Len(t, createdEnvs, 5)
	})
}

func TestMetrics(t *testing.T) {
	manager, _ := setupTestManager(t)

	t.Run("TrackMetrics", func(t *testing.T) {
		// Create environment
		env, err := manager.CreateEnvironment(context.Background(), "metrics-test")
		require.NoError(t, err)

		// Install package
		err = manager.InstallPackage(context.Background(), env.ID, "requests")
		require.NoError(t, err)

		// Update environment
		err = manager.UpdateEnvironment(context.Background(), env.ID)
		require.NoError(t, err)

		// Destroy environment
		err = manager.DestroyEnvironment(env.ID)
		require.NoError(t, err)

		// Check metrics
		stats := manager.GetStats()
		assert.Equal(t, int64(1), stats.EnvsCreated)
		assert.Equal(t, int64(1), stats.EnvsDestroyed)
		assert.Equal(t, int64(1), stats.PackagesInstalled)
		assert.Greater(t, stats.AverageCreateTime, time.Duration(0))
		assert.Greater(t, stats.AverageUpdateTime, time.Duration(0))
	})
}

func TestErrorHandling(t *testing.T) {
	manager, _ := setupTestManager(t)

	t.Run("TimeoutHandling", func(t *testing.T) {
		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()

		// Try to create environment with timeout
		_, err := manager.CreateEnvironment(ctx, "timeout-test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), context.DeadlineExceeded.Error())
	})

	t.Run("InvalidPythonVersion", func(t *testing.T) {
		invalidConfig := &EnvConfig{
			BaseDir:       t.TempDir(),
			PythonVersion: "invalid-version",
			Timeout:       time.Minute,
		}

		_, err := NewManager(invalidConfig, zaptest.NewLogger(t))
		assert.NoError(t, err) // Manager creation should succeed

		// Try to create environment with invalid Python version
		_, err = manager.CreateEnvironment(context.Background(), "invalid-python-test")
		assert.Error(t, err)
	})

	t.Run("DiskSpaceExhaustion", func(t *testing.T) {
		if os.Getuid() != 0 { // Skip if not root
			t.Skip("Test requires root privileges")
		}

		// Set very low disk quota
		cmd := exec.Command("quota", "-w", "0")
		err := cmd.Run()
		require.NoError(t, err)
		defer exec.Command("quota", "-w", "unlimited").Run()

		// Try to create environment
		_, err = manager.CreateEnvironment(context.Background(), "disk-space-test")
		assert.Error(t, err)
	})
}

func TestEnvironmentCleanup(t *testing.T) {
	manager, baseDir := setupTestManager(t)

	t.Run("CleanupOnFailure", func(t *testing.T) {
		// Create environment that will fail
		_, err := manager.CreateEnvironment(context.Background(), "cleanup-test")
		require.NoError(t, err)

		// Corrupt the environment
		err = os.RemoveAll(filepath.Join(baseDir, "cleanup-test", getBinDir()))
		require.NoError(t, err)

		// Try to update the corrupted environment
		err = manager.UpdateEnvironment(context.Background(), "cleanup-test")
		assert.Error(t, err)

		// Verify environment is marked as error
		env, err := manager.GetEnvironment("cleanup-test")
		require.NoError(t, err)
		assert.Equal(t, EnvStatusError, env.Status)
	})
}

func TestRequirementsHandling(t *testing.T) {
	t.Run("RequirementsInstallation", func(t *testing.T) {
		config := &EnvConfig{
			BaseDir: t.TempDir(),
			Requirements: []string{
				"requests==2.26.0",
				"urllib3==1.26.7",
			},
			Timeout: time.Minute,
		}

		manager, err := NewManager(config, zaptest.NewLogger(t))
		require.NoError(t, err)

		env, err := manager.CreateEnvironment(context.Background(), "requirements-test")
		require.NoError(t, err)

		// Verify both packages are installed
		for _, pkg := range []string{"requests", "urllib3"} {
			cmd := exec.Command(env.PythonPath, "-c", fmt.Sprintf("import %s", pkg))
			err := cmd.Run()
			assert.NoError(t, err, "Package %s should be installed", pkg)
		}
	})
}
