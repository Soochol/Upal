package upal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateID creates a random ID with the given prefix, e.g. "wf-abc123".
func GenerateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}
