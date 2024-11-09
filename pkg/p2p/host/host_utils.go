package host

import (
	"crypto/rand"
	"fmt"
	"os"
	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/p2p/message"

	"github.com/libp2p/go-libp2p-core/crypto"
)

// loadOrGenerateKey loads the private key from file or generates a new one
func loadOrGenerateKey(keyFile string) (crypto.PrivKey, error) {
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		// Generate new key
		priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate key: %w", err)
		}

		// Save key to file
		keyBytes, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal private key: %w", err)
		}
		if err := os.WriteFile(keyFile, keyBytes, 0600); err != nil {
			return nil, fmt.Errorf("failed to save key to file: %w", err)
		}

		return priv, nil
	}

	// Load key from file
	keyBytes, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	priv, err := crypto.UnmarshalPrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
	}

	return priv, nil
}

// verifyMessage verifies the message signature
func (h *Host) verifyMessage(msg *message.Message) error {
	// Serialize the message without the signature
	dataToVerify, err := msg.MarshalWithoutSignature()
	if err != nil {
		return fmt.Errorf("failed to marshal message for verification: %w", err)
	}

	// Get the public key of the sender
	pubKey := h.host.Peerstore().PubKey(msg.SenderID)
	if pubKey == nil {
		return fmt.Errorf("public key not found for sender: %s", msg.SenderID)
	}

	// Verify the signature
	if ok, err := pubKey.Verify(dataToVerify, msg.Signature); err != nil || !ok {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// validateConfig validates the P2P configuration
func validateConfig(cfg *config.P2PConfig) error {
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", cfg.Port)
	}
	return nil
}
