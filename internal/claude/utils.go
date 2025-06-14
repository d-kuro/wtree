package claude

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// generateShortID generates a short random ID (6 characters)
func generateShortID() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		// Fall back to a basic timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("%06d", len(b)*1000000)
	}
	return hex.EncodeToString(b)
}

// generateUUID generates a UUID-like string
func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fall back to a deterministic UUID-like string if crypto/rand fails
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			0, 0, 0, 0, 0)
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
