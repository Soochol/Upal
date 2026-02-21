package crypto

import (
	"crypto/rand"
	"testing"
)

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}

	original := "my-secret-token-12345"
	ciphertext, err := enc.Encrypt(original)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if ciphertext == original {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if decrypted != original {
		t.Fatalf("decrypted %q != original %q", decrypted, original)
	}
}

func TestEncryptDecrypt_NoopMode(t *testing.T) {
	enc, err := NewEncryptor(nil)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}

	text := "plaintext-secret"
	ct, err := enc.Encrypt(text)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if ct != text {
		t.Fatalf("noop encrypt should return plaintext, got %q", ct)
	}

	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if pt != text {
		t.Fatalf("noop decrypt should return plaintext, got %q", pt)
	}
}

func TestNewEncryptor_InvalidKeyLength(t *testing.T) {
	_, err := NewEncryptor([]byte("too-short"))
	if err == nil {
		t.Fatal("expected error for invalid key length")
	}
}
