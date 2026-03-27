package discovery

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func makeRecord(t *testing.T, peerID string, version int, ts time.Time) []byte {
	t.Helper()
	r := Record{
		PeerID:    peerID,
		Data:      []byte("test"),
		Timestamp: ts,
		Version:   version,
	}
	b, err := json.Marshal(r)
	require.NoError(t, err)
	return b
}

func TestValidatorSelect_AllInvalid(t *testing.T) {
	v := NewValidator(zaptest.NewLogger(t))

	// Expired record
	old := makeRecord(t, "peer1", 1, time.Now().Add(-48*time.Hour))
	// Empty peer ID
	bad := func() []byte {
		b, _ := json.Marshal(Record{Data: []byte("x"), Timestamp: time.Now()})
		return b
	}()

	_, err := v.Select("key", [][]byte{old, bad})
	assert.Error(t, err, "Select should return error when all records are invalid")
}

func TestValidatorSelect_Empty(t *testing.T) {
	v := NewValidator(zaptest.NewLogger(t))
	_, err := v.Select("key", [][]byte{})
	assert.Error(t, err)
}

func TestValidatorSelect_PicksBest(t *testing.T) {
	v := NewValidator(zaptest.NewLogger(t))

	now := time.Now()
	r1 := makeRecord(t, "peer1", 1, now.Add(-5*time.Minute))
	r2 := makeRecord(t, "peer2", 2, now.Add(-2*time.Minute)) // higher version
	r3 := makeRecord(t, "peer3", 1, now.Add(-1*time.Minute)) // newer but lower version

	idx, err := v.Select("key", [][]byte{r1, r2, r3})
	require.NoError(t, err)
	assert.Equal(t, 1, idx, "should pick record with highest version")
}

func TestValidatorSelect_SameVersionPicksNewest(t *testing.T) {
	v := NewValidator(zaptest.NewLogger(t))

	now := time.Now()
	r1 := makeRecord(t, "peer1", 1, now.Add(-5*time.Minute))
	r2 := makeRecord(t, "peer2", 1, now.Add(-1*time.Minute)) // newer, same version

	idx, err := v.Select("key", [][]byte{r1, r2})
	require.NoError(t, err)
	assert.Equal(t, 1, idx, "should pick newest record when versions match")
}

func TestValidatorSelect_SkipsInvalidKeepsValid(t *testing.T) {
	v := NewValidator(zaptest.NewLogger(t))

	now := time.Now()
	invalid := makeRecord(t, "peer1", 1, now.Add(-48*time.Hour)) // too old
	valid := makeRecord(t, "peer2", 1, now.Add(-5*time.Minute))

	idx, err := v.Select("key", [][]byte{invalid, valid})
	require.NoError(t, err)
	assert.Equal(t, 1, idx, "should skip invalid and return valid record index")
}
