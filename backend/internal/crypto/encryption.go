package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

func Encrypt(key, plaintext []byte) ([]byte, error) {
	// GitHub access tokens are encrypted at rest using authenticated
	// encryption so tampering with stored ciphertext is detected.

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Decrypt decrypts the given ciphertext using the provided key.
func Decrypt(key, ciphertext []byte) ([]byte, error) {
	// Initialize AES using the decryption key.
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// The ciphertext must contain at least the IV before decryption can begin.
	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext is too short")
	}

	// Use AES-GCM to decrypt the ciphertext with the generated IV.
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
