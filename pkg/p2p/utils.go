package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

)


// generateMessageID generates a unique message ID
func generateMessageID() string {
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		panic("failed to generate random bytes for message ID")
	}
	randomPart := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%d-%s", timestamp, randomPart)
}
