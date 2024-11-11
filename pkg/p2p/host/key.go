package host

import (
	"crypto/rand"
	"fmt"
	"os"

	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
)

// loadOrGenerateKey loads the private key from file or generates a new one
func loadOrGenerateKey(keyFile string) (libp2pCrypto.PrivKey, error) {
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		// Generate new key
		priv, _, err := libp2pCrypto.GenerateKeyPairWithReader(libp2pCrypto.RSA, 2048, rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate key: %w", err)
		}

		// Save key to file
		keyBytes, err := libp2pCrypto.MarshalPrivateKey(priv)
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
	priv, err := libp2pCrypto.UnmarshalPrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
	}

	return priv, nil
}
