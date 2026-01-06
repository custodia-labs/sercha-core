package postgres

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	// secretVersion is the version byte for the encrypted blob format.
	// This allows future format changes while maintaining backward compatibility.
	secretVersion = 0x01

	// nonceSize is the AES-GCM nonce size (12 bytes is standard)
	nonceSize = 12

	// keySize is the required key size for AES-256
	keySize = 32
)

var (
	// ErrInvalidKeySize is returned when the encryption key is not 32 bytes.
	ErrInvalidKeySize = errors.New("encryption key must be 32 bytes")

	// ErrInvalidBlobSize is returned when the encrypted blob is too small.
	ErrInvalidBlobSize = errors.New("encrypted blob is too small")

	// ErrUnsupportedVersion is returned when the blob version is not supported.
	ErrUnsupportedVersion = errors.New("unsupported secret blob version")

	// ErrDecryptionFailed is returned when decryption fails (wrong key or corrupted data).
	ErrDecryptionFailed = errors.New("failed to decrypt secret blob")
)

// SecretEncryptor handles AES-256-GCM encryption/decryption of secrets.
// The encrypted format is: version(1) || nonce(12) || ciphertext(N)
type SecretEncryptor struct {
	gcm cipher.AEAD
}

// NewSecretEncryptor creates a new encryptor with the given 32-byte key.
func NewSecretEncryptor(key []byte) (*SecretEncryptor, error) {
	if len(key) != keySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidKeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	return &SecretEncryptor{gcm: gcm}, nil
}

// Encrypt encrypts the given value to a blob.
// The value is JSON-marshaled before encryption.
// Format: version(1) || nonce(12) || ciphertext
func (e *SecretEncryptor) Encrypt(value any) ([]byte, error) {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal value: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := e.gcm.Seal(nil, nonce, plaintext, nil)

	// Build blob: version || nonce || ciphertext
	blob := make([]byte, 1+nonceSize+len(ciphertext))
	blob[0] = secretVersion
	copy(blob[1:1+nonceSize], nonce)
	copy(blob[1+nonceSize:], ciphertext)

	return blob, nil
}

// Decrypt decrypts a blob and unmarshals the result into the given value.
// The value should be a pointer to the target type.
func (e *SecretEncryptor) Decrypt(blob []byte, value any) error {
	minSize := 1 + nonceSize + e.gcm.Overhead()
	if len(blob) < minSize {
		return ErrInvalidBlobSize
	}

	version := blob[0]
	if version != secretVersion {
		return fmt.Errorf("%w: got version %d", ErrUnsupportedVersion, version)
	}

	nonce := blob[1 : 1+nonceSize]
	ciphertext := blob[1+nonceSize:]

	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ErrDecryptionFailed
	}

	if err := json.Unmarshal(plaintext, value); err != nil {
		return fmt.Errorf("unmarshal decrypted value: %w", err)
	}

	return nil
}

// EncryptString encrypts a simple string value.
func (e *SecretEncryptor) EncryptString(s string) ([]byte, error) {
	return e.Encrypt(s)
}

// DecryptString decrypts a blob to a string.
func (e *SecretEncryptor) DecryptString(blob []byte) (string, error) {
	var s string
	if err := e.Decrypt(blob, &s); err != nil {
		return "", err
	}
	return s, nil
}
