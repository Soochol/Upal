package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyHMAC(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"message":"hello"}`)

	// Generate valid signature.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		payload   []byte
		secret    string
		signature string
		valid     bool
	}{
		{"valid signature", payload, secret, validSig, true},
		{"wrong signature", payload, secret, "deadbeef", false},
		{"empty signature", payload, secret, "", false},
		{"wrong secret", payload, "other-secret", validSig, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := verifyHMAC(tt.payload, tt.secret, tt.signature)
			if got != tt.valid {
				t.Errorf("verifyHMAC() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestMapInputs(t *testing.T) {
	payload := map[string]any{
		"message": "hello world",
		"user":    "alice",
		"count":   42,
	}

	t.Run("with mapping", func(t *testing.T) {
		mapping := map[string]string{
			"query":    "message",
			"username": "user",
		}
		inputs := mapInputs(payload, mapping)
		if inputs["query"] != "hello world" {
			t.Errorf("expected query=hello world, got %v", inputs["query"])
		}
		if inputs["username"] != "alice" {
			t.Errorf("expected username=alice, got %v", inputs["username"])
		}
		if _, ok := inputs["count"]; ok {
			t.Error("count should not be in inputs (not in mapping)")
		}
	})

	t.Run("without mapping", func(t *testing.T) {
		inputs := mapInputs(payload, nil)
		if inputs["message"] != "hello world" {
			t.Errorf("expected full payload passthrough, got %v", inputs)
		}
	})
}
