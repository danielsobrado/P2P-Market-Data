package host

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// newRunningHost returns a minimal Host whose running flag is already true.
// Because Start() returns immediately when running==true, the internal
// libp2p host field does not need to be initialised for these tests.
func newRunningHost(t *testing.T) *Host {
	t.Helper()
	return &Host{
		running: true,
		logger:  zaptest.NewLogger(t),
		status:  NewStatus(),
	}
}

// TestHostStart_GuardAlreadyRunning verifies that Start returns an error
// immediately when the host is already running, without touching the
// internal libp2p handle.
func TestHostStart_GuardAlreadyRunning(t *testing.T) {
	h := newRunningHost(t)
	err := h.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

// TestHostStop_IdempotentWhenNotRunning verifies that Stop is a no-op and
// returns nil when the host is not running.
func TestHostStop_IdempotentWhenNotRunning(t *testing.T) {
	h := &Host{
		running: false,
		logger:  zaptest.NewLogger(t),
		status:  NewStatus(),
	}
	err := h.Stop()
	require.NoError(t, err)
}

// TestHostStart_ConcurrentGuard verifies that, under concurrent Start()
// calls on an already-running host, every caller receives an error and
// no data race is introduced.  Run with: go test -race ./pkg/p2p/host/...
func TestHostStart_ConcurrentGuard(t *testing.T) {
	h := newRunningHost(t)

	const numCallers = 20
	errs := make([]error, numCallers)
	var wg sync.WaitGroup

	for i := 0; i < numCallers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = h.Start(context.Background())
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.Errorf(t, err, "caller %d should have received an error", i)
		assert.Contains(t, err.Error(), "already running",
			"caller %d unexpected error: %v", i, err)
	}

	// The running flag must still be true after all the rejected calls.
	assert.True(t, h.IsRunning())
}

// TestHostIsRunning_ThreadSafe verifies that IsRunning() can be called
// concurrently with no data race.
func TestHostIsRunning_ThreadSafe(t *testing.T) {
	h := newRunningHost(t)

	const numReaders = 50
	var wg sync.WaitGroup
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = h.IsRunning()
		}()
	}
	wg.Wait()
}
