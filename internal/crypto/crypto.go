package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Encryptor provides AES-256-GCM encryption and decryption for secrets.
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor creates an Encryptor with the given 32-byte key.
// If the key is empty, a no-op encryptor is returned that stores values as plaintext.
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) == 0 {
		return &Encryptor{}, nil
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	return &Encryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext and returns a base64-encoded ciphertext.
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if e.gcm == nil {
		return plaintext, nil // no-op mode
	}
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts a base64-encoded ciphertext.
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if e.gcm == nil {
		return ciphertext, nil // no-op mode
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}
	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ct := data[:nonceSize], data[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}
