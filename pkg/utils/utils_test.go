package utils

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRetryWithBackoff(t *testing.T) {
	t.Run("SuccessfulOperation", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary error")
			}
			return nil
		}

		err := RetryWithBackoff(context.Background(), operation, DefaultRetryConfig())
		require.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("MaxAttemptsExceeded", func(t *testing.T) {
		attempts := 0
		operation := func() error {
			attempts++
			return errors.New("persistent error")
		}

		err := RetryWithBackoff(context.Background(), operation, DefaultRetryConfig())
		require.Error(t, err)
		assert.Equal(t, DefaultRetryConfig().MaxAttempts, attempts)
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0
		operation := func() error {
			attempts++
			cancel()
			return errors.New("error")
		}

		err := RetryWithBackoff(ctx, operation, DefaultRetryConfig())
		require.Error(t, err)
		assert.Equal(t, 1, attempts)
	})
}

func TestJSONHelper(t *testing.T) {
	helper := &JSONHelper{}

	t.Run("MarshalWithIndent", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		bytes, err := helper.MarshalWithIndent(data)
		require.NoError(t, err)
		assert.Contains(t, string(bytes), "key")
		assert.Contains(t, string(bytes), "value")
	})

	t.Run("UnmarshalSafely", func(t *testing.T) {
		json := []byte(`{"key": "value"}`)
		var data map[string]string
		err := helper.UnmarshalSafely(json, &data)
		require.NoError(t, err)
		assert.Equal(t, "value", data["key"])
	})
}

func TestFileHelper(t *testing.T) {
	helper := &FileHelper{}

	t.Run("WriteFileSafely", func(t *testing.T) {
		tempDir := t.TempDir()
		filename := tempDir + "/test.txt"
		data := []byte("test data")

		err := helper.WriteFileSafely(filename, data, 0644)
		require.NoError(t, err)

		readData, err := os.ReadFile(filename)
		require.NoError(t, err)
		assert.Equal(t, data, readData)
	})

	t.Run("EnsureDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		path := tempDir + "/test/nested"

		err := helper.EnsureDirectory(path, 0755)
		require.NoError(t, err)

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestCacheHelper(t *testing.T) {
	cache := NewCacheHelper()

	t.Run("SetAndGet", func(t *testing.T) {
		cache.Set("key", "value", time.Minute)
		value, exists := cache.Get("key")
		assert.True(t, exists)
		assert.Equal(t, "value", value)
	})

	t.Run("Expiration", func(t *testing.T) {
		cache.Set("temp", "value", time.Millisecond)
		time.Sleep(time.Millisecond * 2)
		_, exists := cache.Get("temp")
		assert.False(t, exists)
	})
}

func TestMetricsHelper(t *testing.T) {
	metrics := NewMetricsHelper()

	t.Run("RecordAndRetrieve", func(t *testing.T) {
		metrics.RecordValue("test", 1.0)
		metrics.RecordValue("test", 2.0)
		metrics.RecordValue("test", 3.0)

		stats := metrics.GetMetric("test")
		require.NotNil(t, stats)
		assert.Equal(t, float64(2.0), stats.Average)
		assert.Equal(t, int64(3), stats.Count)
		assert.Equal(t, float64(1.0), stats.Min)
		assert.Equal(t, float64(3.0), stats.Max)
	})
}

func TestThreadSafeSet(t *testing.T) {
	set := NewThreadSafeSet[string]()

	t.Run("BasicOperations", func(t *testing.T) {
		// Test Add and Contains
		set.Add("item1")
		assert.True(t, set.Contains("item1"))
		assert.False(t, set.Contains("item2"))

		// Test Remove
		set.Remove("item1")
		assert.False(t, set.Contains("item1"))

		// Test Items
		set.Add("item1")
		set.Add("item2")
		items := set.Items()
		assert.Len(t, items, 2)
		assert.Contains(t, items, "item1")
		assert.Contains(t, items, "item2")
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		set := NewThreadSafeSet[int]()
		const numGoroutines = 100
		const numOperations = 100

		// Concurrent additions
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(base int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					set.Add(base + j)
				}
			}(i * numOperations)
		}
		wg.Wait()

		// Verify results
		items := set.Items()
		assert.Len(t, items, numGoroutines*numOperations)
	})
}

func TestValidationHelper(t *testing.T) {
	helper := &ValidationHelper{}

	t.Run("ValidateEmail", func(t *testing.T) {
		tests := []struct {
			email string
			valid bool
		}{
			{"test@example.com", true},
			{"invalid.email", false},
			{"test@domain", false},
			{"test.name+tag@example.com", true},
			{"", false},
		}

		for _, tt := range tests {
			t.Run(tt.email, func(t *testing.T) {
				assert.Equal(t, tt.valid, helper.ValidateEmail(tt.email))
			})
		}
	})

	t.Run("ValidateURL", func(t *testing.T) {
		tests := []struct {
			url   string
			valid bool
		}{
			{"https://example.com", true},
			{"http://localhost:8080", true},
			{"invalid-url", false},
			{"ftp://server.com", true},
			{"", false},
		}

		for _, tt := range tests {
			t.Run(tt.url, func(t *testing.T) {
				assert.Equal(t, tt.valid, helper.ValidateURL(tt.url))
			})
		}
	})
}

func TestTimeHelper(t *testing.T) {
	helper := &TimeHelper{}

	t.Run("FormatDuration", func(t *testing.T) {
		tests := []struct {
			duration time.Duration
			expected string
		}{
			{500 * time.Millisecond, "500ms"},
			{1500 * time.Millisecond, "1.5s"},
			{70 * time.Second, "1m10s"},
			{90 * time.Minute, "1h30m"},
		}

		for _, tt := range tests {
			t.Run(tt.expected, func(t *testing.T) {
				assert.Equal(t, tt.expected, helper.FormatDuration(tt.duration))
			})
		}
	})

	t.Run("ParseDuration", func(t *testing.T) {
		tests := []struct {
			input    string
			expected time.Duration
			hasError bool
		}{
			{"1d", 24 * time.Hour, false},
			{"1w", 7 * 24 * time.Hour, false},
			{"1h30m", 90 * time.Minute, false},
			{"invalid", 0, true},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				duration, err := helper.ParseDuration(tt.input)
				if tt.hasError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.expected, duration)
				}
			})
		}
	})
}

func TestBytesHelper(t *testing.T) {
	helper := &BytesHelper{}

	t.Run("FormatBytes", func(t *testing.T) {
		tests := []struct {
			bytes    int64
			expected string
		}{
			{500, "500 B"},
			{1500, "1.5 KB"},
			{1500000, "1.4 MB"},
			{1500000000, "1.4 GB"},
		}

		for _, tt := range tests {
			t.Run(tt.expected, func(t *testing.T) {
				assert.Equal(t, tt.expected, helper.FormatBytes(tt.bytes))
			})
		}
	})

	t.Run("ParseBytes", func(t *testing.T) {
		tests := []struct {
			input    string
			expected int64
			hasError bool
		}{
			{"1KB", 1024, false},
			{"1.5MB", 1572864, false},
			{"1GB", 1073741824, false},
			{"invalid", 0, true},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				bytes, err := helper.ParseBytes(tt.input)
				if tt.hasError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.expected, bytes)
				}
			})
		}
	})
}

func TestSliceHelper(t *testing.T) {
	intHelper := &SliceHelper[int]{}
	stringHelper := &SliceHelper[string]{}

	t.Run("Chunk", func(t *testing.T) {
		slice := []int{1, 2, 3, 4, 5}
		chunks := intHelper.Chunk(slice, 2)
		assert.Len(t, chunks, 3)
		assert.Equal(t, []int{1, 2}, chunks[0])
		assert.Equal(t, []int{3, 4}, chunks[1])
		assert.Equal(t, []int{5}, chunks[2])
	})

	t.Run("Contains", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		assert.True(t, stringHelper.Contains(slice, "b"))
		assert.False(t, stringHelper.Contains(slice, "d"))
		assert.False(t, stringHelper.Contains(slice, "d"))
	})

	t.Run("Unique", func(t *testing.T) {
		slice := []int{1, 2, 3, 4, 4}
		unique := intHelper.Unique(slice)
		assert.Equal(t, []int{1, 2, 3, 4}, unique)
	})
}

func TestEnvironment(t *testing.T) {
	env := &Environment{}

	t.Run("GetEnvWithDefault", func(t *testing.T) {
		// Test with unset variable
		value := env.GetEnvWithDefault("TEST_VAR_UNSET", "default")
		assert.Equal(t, "default", value)

		// Test with set variable
		os.Setenv("TEST_VAR_SET", "value")
		defer os.Unsetenv("TEST_VAR_SET")
		value = env.GetEnvWithDefault("TEST_VAR_SET", "default")
		assert.Equal(t, "value", value)
	})

	t.Run("RequireEnv", func(t *testing.T) {
		// Test with set variable
		os.Setenv("TEST_VAR_REQUIRED", "value")
		defer os.Unsetenv("TEST_VAR_REQUIRED")
		value := env.RequireEnv("TEST_VAR_REQUIRED")
		assert.Equal(t, "value", value)

		// Test with unset variable
		assert.Panics(t, func() {
			env.RequireEnv("TEST_VAR_UNSET")
		})
	})

	t.Run("SetEnvIfNotExists", func(t *testing.T) {
		// Test setting new variable
		err := env.SetEnvIfNotExists("TEST_VAR_NEW", "value")
		assert.NoError(t, err)
		assert.Equal(t, "value", os.Getenv("TEST_VAR_NEW"))

		// Test not overwriting existing variable
		err = env.SetEnvIfNotExists("TEST_VAR_NEW", "new-value")
		assert.NoError(t, err)
		assert.Equal(t, "value", os.Getenv("TEST_VAR_NEW"))
	})
}

func TestSafeGo(t *testing.T) {
	logger := zap.NewExample()

	t.Run("NormalExecution", func(t *testing.T) {
		executed := make(chan bool)
		SafeGo(logger, func() {
			executed <- true
		})
		assert.True(t, <-executed)
	})

	t.Run("PanicRecovery", func(t *testing.T) {
		recovered := make(chan bool)
		SafeGo(logger, func() {
			defer func() {
				recovered <- true
			}()
			panic("test panic")
		})
		assert.True(t, <-recovered)
	})
}
