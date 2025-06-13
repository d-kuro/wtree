package claude

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// generateShortID generates a short random ID (6 characters)
func generateShortID() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// generateUUID generates a UUID-like string
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
