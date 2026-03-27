package host

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHostState_InitialStateIsNotRunning verifies that a zero-value Host
// reports IsRunning() == false.  This guards against accidental initialisation
// of the state field to a truthy value.
func TestHostState_InitialStateIsNotRunning(t *testing.T) {
	h := &Host{}
	assert.False(t, h.IsRunning(), "a freshly-created host must not report itself as running")
}

// TestHostState_RunningStateIsDetected verifies that setting state to
// hostRunning causes IsRunning to return true.
func TestHostState_RunningStateIsDetected(t *testing.T) {
	h := &Host{state: hostRunning}
	assert.True(t, h.IsRunning(), "IsRunning must return true when state == hostRunning")
}

// TestHostState_StoppingIsNotRunning verifies that IsRunning returns false
// while the host is in the stopping state.  This is the key semantic
// improvement: callers that observe IsRunning()==false cannot distinguish
// "never started" from "shutting down", but they correctly know the host is
// not accepting new work.
func TestHostState_StoppingIsNotRunning(t *testing.T) {
	h := &Host{state: hostStopping}
	assert.False(t, h.IsRunning(), "IsRunning must return false while the host is stopping")
}

// TestHostState_StoppedIsNotRunning verifies that IsRunning returns false
// after the host has fully stopped.
func TestHostState_StoppedIsNotRunning(t *testing.T) {
	h := &Host{state: hostStopped}
	assert.False(t, h.IsRunning(), "IsRunning must return false after the host has stopped")
}
