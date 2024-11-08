package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"go.uber.org/zap"
)

// RetryConfig holds retry operation configuration
type RetryConfig struct {
	MaxAttempts      int
	InitialDelay     time.Duration
	MaxDelay         time.Duration
	BackoffFactor    float64
	RetryableErrors  []error
	MaxJitterPercent float64
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:      3,
		InitialDelay:     100 * time.Millisecond,
		MaxDelay:         5 * time.Second,
		BackoffFactor:    2.0,
		MaxJitterPercent: 0.2,
	}
}

// RetryWithBackoff executes an operation with exponential backoff and jitter
func RetryWithBackoff(ctx context.Context, operation func() error, cfg *RetryConfig) error {
	if cfg == nil {
		cfg = DefaultRetryConfig()
	}

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := operation(); err != nil {
			lastErr = err

			// Check if error is retryable
			if !isRetryableError(err, cfg.RetryableErrors) {
				return err
			}

			// Check context before delay
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(addJitter(delay, cfg.MaxJitterPercent)):
			}

			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * cfg.BackoffFactor)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
			continue
		}
		return nil
	}

	return fmt.Errorf("operation failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// SafeGo executes a function in a goroutine with panic recovery
func SafeGo(logger *zap.Logger, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic recovered in goroutine",
					zap.Any("panic", r),
					zap.Stack("stack"))
			}
		}()
		fn()
	}()
}

// JSONHelper provides JSON encoding/decoding with error wrapping
type JSONHelper struct {
	mu sync.Mutex
}

// MarshalWithIndent marshals data to JSON with indentation
func (j *JSONHelper) MarshalWithIndent(v interface{}) ([]byte, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON: %w", err)
	}
	return data, nil
}

// UnmarshalSafely unmarshals JSON data with additional type checking
func (j *JSONHelper) UnmarshalSafely(data []byte, v interface{}) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshaling JSON: %w", err)
	}
	return nil
}

// FileHelper provides safe file operations
type FileHelper struct {
	mu sync.Mutex
}

// WriteFileSafely writes data to a file atomically
func (f *FileHelper) WriteFileSafely(filename string, data []byte, perm os.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Write to temporary file first
	tmpfile := filename + ".tmp"
	if err := os.WriteFile(tmpfile, data, perm); err != nil {
		return fmt.Errorf("writing temporary file: %w", err)
	}

	// Rename temporary file to target
	if err := os.Rename(tmpfile, filename); err != nil {
		os.Remove(tmpfile)
		return fmt.Errorf("renaming temporary file: %w", err)
	}

	return nil
}

// EnsureDirectory ensures a directory exists with correct permissions
func (f *FileHelper) EnsureDirectory(path string, perm os.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	return nil
}

// NetworkHelper provides network-related utilities
type NetworkHelper struct {
	mu sync.Mutex
}

// GetFreePort finds an available network port
func (n *NetworkHelper) GetFreePort() (int, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("resolving TCP address: %w", err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("listening on TCP address: %w", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// RandomHelper provides secure random number generation
type RandomHelper struct {
	mu sync.Mutex
}

// GenerateRandomBytes generates cryptographically secure random bytes
func (r *RandomHelper) GenerateRandomBytes(n int) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("generating random bytes: %w", err)
	}
	return b, nil
}

// GenerateRandomString generates a random string of specified length
func (r *RandomHelper) GenerateRandomString(length int) (string, error) {
	bytes, err := r.GenerateRandomBytes(length)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// Helper functions

func isRetryableError(err error, retryableErrors []error) bool {
	if len(retryableErrors) == 0 {
		return true
	}
	for _, retryableErr := range retryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}
	return false
}

func addJitter(delay time.Duration, maxJitterPercent float64) time.Duration {
	if maxJitterPercent <= 0 {
		return delay
	}

	jitter := delay * time.Duration(maxJitterPercent*rand.Float64())
	return delay + jitter
}

// TimeHelper provides time-related utilities
type TimeHelper struct {
	mu sync.Mutex
}

// FormatDuration formats a duration in a human-readable way
func (t *TimeHelper) FormatDuration(d time.Duration) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// ParseDuration parses a duration string with support for days and weeks
func (t *TimeHelper) ParseDuration(s string) (time.Duration, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	s = strings.ToLower(s)
	if strings.Contains(s, "w") {
		weeks := strings.Split(s, "w")[0]
		w, err := strconv.Atoi(weeks)
		if err != nil {
			return 0, fmt.Errorf("parsing weeks: %w", err)
		}
		return time.Duration(w) * 7 * 24 * time.Hour, nil
	}
	if strings.Contains(s, "d") {
		days := strings.Split(s, "d")[0]
		d, err := strconv.Atoi(days)
		if err != nil {
			return 0, fmt.Errorf("parsing days: %w", err)
		}
		return time.Duration(d) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

// BytesHelper provides byte manipulation utilities
type BytesHelper struct {
	mu sync.Mutex
}

// FormatBytes formats bytes in a human-readable way
func (b *BytesHelper) FormatBytes(bytes int64) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ParseBytes parses a human-readable byte string
func (b *BytesHelper) ParseBytes(s string) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)

	i := strings.IndexFunc(s, unicode.IsLetter)
	if i == -1 {
		return strconv.ParseInt(s, 10, 64)
	}

	bytesf, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0, fmt.Errorf("parsing bytes: %w", err)
	}

	unit := s[i:]
	switch unit {
	case "KB":
		return int64(bytesf * 1024), nil
	case "MB":
		return int64(bytesf * 1024 * 1024), nil
	case "GB":
		return int64(bytesf * 1024 * 1024 * 1024), nil
	case "TB":
		return int64(bytesf * 1024 * 1024 * 1024 * 1024), nil
	}

	return 0, fmt.Errorf("unknown unit: %s", unit)
}

// ValidationHelper provides validation utilities
type ValidationHelper struct {
	mu sync.Mutex
}

// ValidateEmail validates email format
func (v *ValidationHelper) ValidateEmail(email string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)
	return match
}

// ValidateURL validates URL format
func (v *ValidationHelper) ValidateURL(urlStr string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	u, err := url.Parse(urlStr)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// CacheHelper provides basic caching functionality
type CacheHelper struct {
	cache map[string]cacheEntry
	mu    sync.RWMutex
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

// NewCacheHelper creates a new cache helper
func NewCacheHelper() *CacheHelper {
	cache := &CacheHelper{
		cache: make(map[string]cacheEntry),
	}
	go cache.cleanupExpired()
	return cache
}

// Set adds an item to the cache with expiration
func (c *CacheHelper) Set(key string, value interface{}, expiration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = cacheEntry{
		value:      value,
		expiration: time.Now().Add(expiration),
	}
}

// Get retrieves an item from the cache
func (c *CacheHelper) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiration) {
		return nil, false
	}

	return entry.value, true
}

func (c *CacheHelper) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.cache {
			if now.After(entry.expiration) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}

// MetricsHelper provides basic metrics collection
type MetricsHelper struct {
	values map[string]*metricValue
	mu     sync.RWMutex
}

type metricValue struct {
	value     float64
	count     int64
	min       float64
	max       float64
	lastReset time.Time
}

// NewMetricsHelper creates a new metrics helper
func NewMetricsHelper() *MetricsHelper {
	return &MetricsHelper{
		values: make(map[string]*metricValue),
	}
}

// RecordValue records a metric value
func (m *MetricsHelper) RecordValue(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metric, exists := m.values[name]
	if !exists {
		metric = &metricValue{
			min:       value,
			max:       value,
			lastReset: time.Now(),
		}
		m.values[name] = metric
	}

	metric.value += value
	metric.count++
	metric.min = math.Min(metric.min, value)
	metric.max = math.Max(metric.max, value)
}

// GetMetric retrieves metric statistics
func (m *MetricsHelper) GetMetric(name string) *MetricStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metric, exists := m.values[name]
	if !exists {
		return nil
	}

	return &MetricStats{
		Average:   metric.value / float64(metric.count),
		Count:     metric.count,
		Min:       metric.min,
		Max:       metric.max,
		LastReset: metric.lastReset,
	}
}

// MetricStats represents metric statistics
type MetricStats struct {
	Average   float64
	Count     int64
	Min       float64
	Max       float64
	LastReset time.Time
}

// Environment provides environment management utilities
type Environment struct {
	mu sync.RWMutex
}

// GetEnvWithDefault gets an environment variable with a default value
func (e *Environment) GetEnvWithDefault(key, defaultValue string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// RequireEnv gets an environment variable or panics if not set
func (e *Environment) RequireEnv(key string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	value, exists := os.LookupEnv(key)
	if !exists {
		panic(fmt.Sprintf("required environment variable not set: %s", key))
	}
	return value
}

// SetEnvIfNotExists sets an environment variable if it doesn't exist
func (e *Environment) SetEnvIfNotExists(key, value string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := os.LookupEnv(key); !exists {
		return os.Setenv(key, value)
	}
	return nil
}

// SliceHelper provides slice manipulation utilities
type SliceHelper[T comparable] struct {
	mu sync.Mutex
}

// Chunk splits a slice into chunks of specified size
func (s *SliceHelper[T]) Chunk(slice []T, size int) [][]T {
	s.mu.Lock()
	defer s.mu.Unlock()

	var chunks [][]T
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// Contains checks if a slice contains an element
func (s *SliceHelper[T]) Contains(slice []T, element T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}

// Unique returns unique elements from a slice
func (s *SliceHelper[T]) Unique(slice []T) []T {
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[T]struct{})
	result := make([]T, 0)

	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// ThreadSafeSet provides a thread-safe set implementation
type ThreadSafeSet[T comparable] struct {
	items map[T]struct{}
	mu    sync.RWMutex
}

// NewThreadSafeSet creates a new thread-safe set
func NewThreadSafeSet[T comparable]() *ThreadSafeSet[T] {
	return &ThreadSafeSet[T]{
		items: make(map[T]struct{}),
	}
}

// Add adds an item to the set
func (s *ThreadSafeSet[T]) Add(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item] = struct{}{}
}

// Remove removes an item from the set
func (s *ThreadSafeSet[T]) Remove(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, item)
}

// Contains checks if an item exists in the set
func (s *ThreadSafeSet[T]) Contains(item T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.items[item]
	return exists
}

// Items returns all items in the set
func (s *ThreadSafeSet[T]) Items() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]T, 0, len(s.items))
	for item := range s.items {
		items = append(items, item)
	}
	return items
}
