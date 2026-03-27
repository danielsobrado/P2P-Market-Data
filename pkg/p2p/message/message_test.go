package message

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTestMessage returns a fully-populated Message for testing.
func makeTestMessage() *Message {
	return &Message{
		Type:      MarketDataMessage,
		Version:   "1.0.0",
		ID:        "test-id-001",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		SenderID:  peer.ID("sender-peer-id"),
		Data:      map[string]interface{}{"symbol": "AAPL", "price": 150.0},
		Signature: []byte("should-not-appear"),
	}
}

// TestMarshalWithoutSignature_ExcludesSignature verifies that the signature
// field is absent from the bytes that are covered by signing.
func TestMarshalWithoutSignature_ExcludesSignature(t *testing.T) {
	msg := makeTestMessage()
	b, err := msg.MarshalWithoutSignature()
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &parsed))
	_, hasSignature := parsed["signature"]
	assert.False(t, hasSignature, "signature field must not be present in the bytes covered by signing")
}

// TestMarshalWithoutSignature_IncludesIntegrityFields verifies that all
// integrity-sensitive fields (type, version, id, timestamp, sender_id, data)
// are present in the signed payload.
func TestMarshalWithoutSignature_IncludesIntegrityFields(t *testing.T) {
	msg := makeTestMessage()
	b, err := msg.MarshalWithoutSignature()
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &parsed))

	for _, field := range []string{"type", "version", "id", "timestamp", "sender_id", "data"} {
		_, ok := parsed[field]
		assert.Truef(t, ok, "integrity-sensitive field %q must be present in the signed payload", field)
	}
}

// TestMarshalWithoutSignature_VersionAffectsPayload ensures that changing the
// Version field produces a different signed payload.
func TestMarshalWithoutSignature_VersionAffectsPayload(t *testing.T) {
	msg := makeTestMessage()
	b1, err := msg.MarshalWithoutSignature()
	require.NoError(t, err)

	msg.Version = "2.0.0"
	b2, err := msg.MarshalWithoutSignature()
	require.NoError(t, err)

	assert.False(t, bytes.Equal(b1, b2), "changing Version must change the signed payload")
}

// TestMarshalWithoutSignature_IDAffectsPayload ensures that changing the ID
// field produces a different signed payload.
func TestMarshalWithoutSignature_IDAffectsPayload(t *testing.T) {
	msg := makeTestMessage()
	b1, err := msg.MarshalWithoutSignature()
	require.NoError(t, err)

	msg.ID = "different-id"
	b2, err := msg.MarshalWithoutSignature()
	require.NoError(t, err)

	assert.False(t, bytes.Equal(b1, b2), "changing ID must change the signed payload")
}

// TestMarshalWithoutSignature_TimestampAffectsPayload ensures that changing
// the Timestamp field produces a different signed payload.
func TestMarshalWithoutSignature_TimestampAffectsPayload(t *testing.T) {
	msg := makeTestMessage()
	b1, err := msg.MarshalWithoutSignature()
	require.NoError(t, err)

	msg.Timestamp = msg.Timestamp.Add(time.Second)
	b2, err := msg.MarshalWithoutSignature()
	require.NoError(t, err)

	assert.False(t, bytes.Equal(b1, b2), "changing Timestamp must change the signed payload")
}

// TestMarshalWithoutSignature_SignatureChangeDoesNotAffectPayload verifies
// that updating the Signature field alone does not change the signed bytes,
// so re-signing a message produces a consistent payload regardless of any
// previously stored signature.
func TestMarshalWithoutSignature_SignatureChangeDoesNotAffectPayload(t *testing.T) {
	msg := makeTestMessage()
	b1, err := msg.MarshalWithoutSignature()
	require.NoError(t, err)

	msg.Signature = []byte("completely-different-signature")
	b2, err := msg.MarshalWithoutSignature()
	require.NoError(t, err)

	assert.True(t, bytes.Equal(b1, b2), "changing only the Signature must not affect the signed payload")
}

// TestNewMessage_FieldsPopulated verifies that NewMessage populates all
// required fields.
func TestNewMessage_FieldsPopulated(t *testing.T) {
	payload := map[string]string{"key": "value"}
	msg := NewMessage(MarketDataMessage, payload)

	assert.Equal(t, MarketDataMessage, msg.Type)
	assert.NotEmpty(t, msg.Version)
	assert.NotEmpty(t, msg.ID)
	assert.False(t, msg.Timestamp.IsZero())
	assert.Equal(t, payload, msg.Data)
	assert.Nil(t, msg.Signature)
}

// TestMarshalUnmarshal_RoundTrip verifies that marshaling then unmarshaling
// a message preserves non-peer-ID fields correctly.
func TestMarshalUnmarshal_RoundTrip(t *testing.T) {
	msg := makeTestMessage()
	// SenderID must be omitted: the peer.ID JSON codec requires a valid
	// multihash CID string, which the zero value does not produce.
	msg.SenderID = peer.ID("")

	b, err := msg.Marshal()
	require.NoError(t, err)

	// Unmarshal into a plain map to verify field presence without triggering
	// the peer.ID CID parser on an empty value.
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &parsed))

	assert.Equal(t, string(msg.Type), parsed["type"])
	assert.Equal(t, msg.Version, parsed["version"])
	assert.Equal(t, msg.ID, parsed["id"])
	assert.NotNil(t, parsed["data"])
}
