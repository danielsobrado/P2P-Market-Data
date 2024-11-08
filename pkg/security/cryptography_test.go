package security

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestCryptoManager(t *testing.T) {
	// Generate test key pair
	keyPair, err := GenerateKeyPair()
	require.NoError(t, err)

	// Create crypto manager
	jwtSecret := []byte("test-secret")
	cm, err := NewCryptoManager(keyPair, jwtSecret)
	require.NoError(t, err)

	t.Run("SignAndVerify", func(t *testing.T) {
		data := []byte("test data")

		signature, err := cm.Sign(data)
		require.NoError(t, err)

		valid := cm.Verify(data, signature, keyPair.PublicKey)
		assert.True(t, valid)

		// Test invalid signature
		invalidSig := append(signature, byte(0))
		valid = cm.Verify(data, invalidSig, keyPair.PublicKey)
		assert.False(t, valid)
	})

	t.Run("EncryptAndDecrypt", func(t *testing.T) {
		plaintext := []byte("secret message")

		ciphertext, err := cm.Encrypt(plaintext)
		require.NoError(t, err)

		decrypted, err := cm.Decrypt(ciphertext)
		require.NoError(t, err)

		assert.Equal(t, plaintext, decrypted)

		// Test invalid ciphertext
		_, err = cm.Decrypt([]byte("invalid"))
		assert.Error(t, err)
	})

	t.Run("TokenGeneration", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "test-user",
			"exp": time.Now().Add(time.Hour).Unix(),
		}

		token, err := cm.GenerateToken(claims, time.Hour)
		require.NoError(t, err)

		validatedClaims, err := cm.ValidateToken(token.Value)
		require.NoError(t, err)

		assert.Equal(t, claims["sub"], validatedClaims.(jwt.MapClaims)["sub"])
	})
}

func TestReputationManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rm := NewReputationManager(nil, logger, 0.5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, rm.Start(ctx))

	t.Run("ReputationUpdates", func(t *testing.T) {
		peerID := peer.ID("test-peer")

		// Initial reputation
		score, err := rm.GetPeerReputation(peerID)
		require.NoError(t, err)
		assert.Equal(t, InitialScore, score)

		// Update reputation - valid data
		err = rm.UpdatePeerReputation(peerID, ValidData, 1.0)
		require.NoError(t, err)

		score, err = rm.GetPeerReputation(peerID)
		require.NoError(t, err)
		assert.Greater(t, score, InitialScore)

		// Update reputation - invalid data
		err = rm.UpdatePeerReputation(peerID, InvalidData, 1.0)
		require.NoError(t, err)

		score, err = rm.GetPeerReputation(peerID)
		require.NoError(t, err)
		assert.Less(t, score, InitialScore)
	})

	t.Run("BatchUpdates", func(t *testing.T) {
		updates := map[peer.ID]ReputationUpdate{
			"peer1": {Action: ValidData, Value: 1.0},
			"peer2": {Action: InvalidData, Value: 1.0},
		}

		err := rm.BatchUpdateReputations(updates)
		require.NoError(t, err)

		score1, err := rm.GetPeerReputation("peer1")
		require.NoError(t, err)
		assert.Greater(t, score1, InitialScore)

		score2, err := rm.GetPeerReputation("peer2")
		require.NoError(t, err)
		assert.Less(t, score2, InitialScore)
	})

	t.Run("ReputationStats", func(t *testing.T) {
		// Add some test data
		for i := 0; i < 5; i++ {
			peerID := peer.ID(fmt.Sprintf("test-peer-%d", i))
			err := rm.UpdatePeerReputation(peerID, ValidData, 1.0)
			require.NoError(t, err)
		}

		stats := rm.GetReputationStats()
		assert.Greater(t, stats.HighRepPeers, 0)
		assert.Greater(t, stats.AverageScore, 0.0)
	})

	t.Run("PeerStats", func(t *testing.T) {
		peerID := peer.ID("test-peer-stats")

		// Perform some actions
		err := rm.UpdatePeerReputation(peerID, ValidData, 1.0)
		require.NoError(t, err)
		err = rm.UpdatePeerReputation(peerID, InvalidData, 0.5)
		require.NoError(t, err)

		stats, err := rm.GetPeerStats(peerID)
		require.NoError(t, err)

		assert.Equal(t, uint64(1), stats.ValidData)
		assert.Equal(t, uint64(1), stats.InvalidData)
		assert.Equal(t, uint64(2), stats.TotalActions)
	})
}
