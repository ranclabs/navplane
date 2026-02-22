package providerkey

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func generateKey(t *testing.T) []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	return key
}

func TestNewEncryptor_ValidKey(t *testing.T) {
	key := generateKey(t)
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enc == nil {
		t.Error("expected encryptor to be non-nil")
	}
}

func TestNewEncryptor_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name    string
		keyLen  int
		wantErr bool
	}{
		{"16 bytes", 16, true},
		{"24 bytes", 24, true},
		{"32 bytes", 32, false},
		{"64 bytes", 64, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := NewEncryptor(key)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptor() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	kek := generateKey(t)
	enc, _ := NewEncryptor(kek)

	plaintext := []byte("sk-test-api-key-12345")

	encrypted, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted.EncryptedKey == nil {
		t.Error("EncryptedKey should not be nil")
	}
	if encrypted.KeyNonce == nil {
		t.Error("KeyNonce should not be nil")
	}
	if encrypted.EncryptedDEK == nil {
		t.Error("EncryptedDEK should not be nil")
	}
	if encrypted.DEKNonce == nil {
		t.Error("DEKNonce should not be nil")
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted text mismatch: got %s, want %s", decrypted, plaintext)
	}
}

func TestEncryptor_DifferentEncryptions(t *testing.T) {
	kek := generateKey(t)
	enc, _ := NewEncryptor(kek)

	plaintext := []byte("same-plaintext")

	enc1, _ := enc.Encrypt(plaintext)
	enc2, _ := enc.Encrypt(plaintext)

	// Same plaintext should produce different ciphertext due to random nonces
	if bytes.Equal(enc1.EncryptedKey, enc2.EncryptedKey) {
		t.Error("same plaintext should produce different ciphertext")
	}

	// But both should decrypt to the same value
	dec1, _ := enc.Decrypt(enc1)
	dec2, _ := enc.Decrypt(enc2)

	if !bytes.Equal(dec1, plaintext) || !bytes.Equal(dec2, plaintext) {
		t.Error("both should decrypt to original plaintext")
	}
}

func TestEncryptor_ReEncryptDEK(t *testing.T) {
	oldKEK := generateKey(t)
	newKEK := generateKey(t)

	enc, _ := NewEncryptor(oldKEK)

	plaintext := []byte("api-key-to-rotate")
	encrypted, _ := enc.Encrypt(plaintext)

	// Re-encrypt DEK with new KEK
	newEncryptedDEK, newDEKNonce, err := enc.ReEncryptDEK(encrypted.EncryptedDEK, encrypted.DEKNonce, newKEK)
	if err != nil {
		t.Fatalf("ReEncryptDEK failed: %v", err)
	}

	// Create new encryptor with new KEK
	newEnc, _ := NewEncryptor(newKEK)

	// Decrypt with new KEK
	decrypted, err := newEnc.Decrypt(&EncryptedData{
		EncryptedKey: encrypted.EncryptedKey,
		KeyNonce:     encrypted.KeyNonce,
		EncryptedDEK: newEncryptedDEK,
		DEKNonce:     newDEKNonce,
	})
	if err != nil {
		t.Fatalf("Decrypt with new KEK failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted text mismatch after rotation: got %s, want %s", decrypted, plaintext)
	}
}

func TestEncryptor_WrongKEK(t *testing.T) {
	kek1 := generateKey(t)
	kek2 := generateKey(t)

	enc1, _ := NewEncryptor(kek1)
	enc2, _ := NewEncryptor(kek2)

	plaintext := []byte("secret-data")
	encrypted, _ := enc1.Encrypt(plaintext)

	// Try to decrypt with wrong KEK
	_, err := enc2.Decrypt(encrypted)
	if err == nil {
		t.Error("expected error when decrypting with wrong KEK")
	}
}
