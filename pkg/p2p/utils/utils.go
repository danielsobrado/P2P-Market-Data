package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateMessageID creates a unique message identifier
func GenerateMessageID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// In practice, this should almost never happen
		panic("failed to generate random message ID")
	}
	return hex.EncodeToString(bytes)
}
