package postgres

import (
	"testing"
)

func TestSecretEncryptor_RoundTrip(t *testing.T) {
	// Generate a test key (32 bytes)
	key := []byte("01234567890123456789012345678901")

	encryptor, err := NewSecretEncryptor(key)
	if err != nil {
		t.Fatalf("NewSecretEncryptor: %v", err)
	}

	type testSecrets struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		APIKey       string `json:"api_key"`
	}

	original := testSecrets{
		AccessToken:  "gho_abc123",
		RefreshToken: "ghr_xyz789",
		APIKey:       "sk-test-key",
	}

	// Encrypt
	blob, err := encryptor.Encrypt(original)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Verify blob format
	if len(blob) < 1+nonceSize {
		t.Fatalf("blob too short: %d bytes", len(blob))
	}
	if blob[0] != secretVersion {
		t.Errorf("version byte: got %d, want %d", blob[0], secretVersion)
	}

	// Decrypt
	var decrypted testSecrets
	if err := encryptor.Decrypt(blob, &decrypted); err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	// Verify
	if decrypted.AccessToken != original.AccessToken {
		t.Errorf("AccessToken: got %q, want %q", decrypted.AccessToken, original.AccessToken)
	}
	if decrypted.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken: got %q, want %q", decrypted.RefreshToken, original.RefreshToken)
	}
	if decrypted.APIKey != original.APIKey {
		t.Errorf("APIKey: got %q, want %q", decrypted.APIKey, original.APIKey)
	}
}

func TestSecretEncryptor_InvalidKeySize(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
	}{
		{"too short", 16},
		{"too long", 64},
		{"empty", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			_, err := NewSecretEncryptor(key)
			if err == nil {
				t.Error("expected error for invalid key size")
			}
		})
	}
}

func TestSecretEncryptor_DecryptInvalidBlob(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	encryptor, _ := NewSecretEncryptor(key)

	tests := []struct {
		name string
		blob []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte{0x01, 0x02}},
		{"wrong version", append([]byte{0x99}, make([]byte, 100)...)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			err := encryptor.Decrypt(tt.blob, &result)
			if err == nil {
				t.Error("expected error for invalid blob")
			}
		})
	}
}

func TestSecretEncryptor_WrongKey(t *testing.T) {
	key1 := []byte("01234567890123456789012345678901")
	key2 := []byte("10987654321098765432109876543210")

	enc1, _ := NewSecretEncryptor(key1)
	enc2, _ := NewSecretEncryptor(key2)

	// Encrypt with key1
	blob, err := enc1.Encrypt("secret data")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Try to decrypt with key2
	var result string
	err = enc2.Decrypt(blob, &result)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestSecretEncryptor_UniqueNonce(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	encryptor, _ := NewSecretEncryptor(key)

	// Encrypt the same value multiple times
	blobs := make([][]byte, 10)
	for i := range blobs {
		blob, err := encryptor.Encrypt("same value")
		if err != nil {
			t.Fatalf("Encrypt %d: %v", i, err)
		}
		blobs[i] = blob
	}

	// Verify all nonces are unique
	nonces := make(map[string]bool)
	for i, blob := range blobs {
		nonce := string(blob[1 : 1+nonceSize])
		if nonces[nonce] {
			t.Errorf("duplicate nonce at index %d", i)
		}
		nonces[nonce] = true
	}
}

func TestSecretEncryptor_StringHelpers(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	encryptor, _ := NewSecretEncryptor(key)

	original := "my secret string"

	blob, err := encryptor.EncryptString(original)
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}

	decrypted, err := encryptor.DecryptString(blob)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}

	if decrypted != original {
		t.Errorf("got %q, want %q", decrypted, original)
	}
}
