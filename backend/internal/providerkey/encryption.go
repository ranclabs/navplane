package providerkey

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// Encryptor handles envelope encryption for provider API keys.
// Uses AES-256-GCM for both KEK→DEK and DEK→data encryption.
type Encryptor struct {
	kek []byte // Key Encryption Key (from ENCRYPTION_KEY env var)
}

// NewEncryptor creates a new encryptor with the given Key Encryption Key.
// KEK must be exactly 32 bytes (256 bits).
func NewEncryptor(kek []byte) (*Encryptor, error) {
	if len(kek) != 32 {
		return nil, errors.New("KEK must be exactly 32 bytes")
	}
	return &Encryptor{kek: kek}, nil
}

// EncryptedData holds all the encrypted components of a provider key.
type EncryptedData struct {
	EncryptedKey []byte // The actual API key, encrypted with DEK
	KeyNonce     []byte // Nonce used for encrypting the key
	EncryptedDEK []byte // The DEK, encrypted with KEK
	DEKNonce     []byte // Nonce used for encrypting the DEK
}

// Encrypt encrypts a provider API key using envelope encryption.
// 1. Generate a random DEK
// 2. Encrypt the API key with the DEK
// 3. Encrypt the DEK with the KEK
func (e *Encryptor) Encrypt(plaintext []byte) (*EncryptedData, error) {
	// Generate a random 32-byte DEK
	dek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}

	// Encrypt the API key with the DEK
	encryptedKey, keyNonce, err := e.encryptWithKey(dek, plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt key: %w", err)
	}

	// Encrypt the DEK with the KEK
	encryptedDEK, dekNonce, err := e.encryptWithKey(e.kek, dek)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	return &EncryptedData{
		EncryptedKey: encryptedKey,
		KeyNonce:     keyNonce,
		EncryptedDEK: encryptedDEK,
		DEKNonce:     dekNonce,
	}, nil
}

// Decrypt decrypts a provider API key using envelope encryption.
// 1. Decrypt the DEK with the KEK
// 2. Decrypt the API key with the DEK
func (e *Encryptor) Decrypt(data *EncryptedData) ([]byte, error) {
	// Decrypt the DEK with the KEK
	dek, err := e.decryptWithKey(e.kek, data.EncryptedDEK, data.DEKNonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt DEK: %w", err)
	}

	// Decrypt the API key with the DEK
	plaintext, err := e.decryptWithKey(dek, data.EncryptedKey, data.KeyNonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	return plaintext, nil
}

// ReEncryptDEK re-encrypts a DEK with a new KEK (for key rotation).
func (e *Encryptor) ReEncryptDEK(encryptedDEK, dekNonce, newKEK []byte) ([]byte, []byte, error) {
	// Decrypt DEK with current KEK
	dek, err := e.decryptWithKey(e.kek, encryptedDEK, dekNonce)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt DEK: %w", err)
	}

	// Re-encrypt DEK with new KEK
	newEncryptedDEK, newDEKNonce, err := e.encryptWithKey(newKEK, dek)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to re-encrypt DEK: %w", err)
	}

	return newEncryptedDEK, newDEKNonce, nil
}

// encryptWithKey encrypts data using AES-256-GCM.
func (e *Encryptor) encryptWithKey(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// decryptWithKey decrypts data using AES-256-GCM.
func (e *Encryptor) decryptWithKey(key, ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
