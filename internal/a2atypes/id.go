package a2atypes

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateID generates a random ID with the given prefix.
func GenerateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}
