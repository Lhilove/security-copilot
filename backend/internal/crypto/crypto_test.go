package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestEncryptDecrypt tests the Encrypt and Decrypt functions to ensure that they work correctly.
func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)

	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("github-access-token-example")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("decrypted plaintext does not match original")
	}
}

// TestEncryptionProducesDifferentCiphertext tests that encrypting the same plaintext multiple times produces different ciphertexts.
func TestEncryptionProducesDifferentCiphertext(t *testing.T) {
	key := make([]byte, 32)

	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("github-access-token-example")

	first, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	second, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(first, second) {
		t.Fatal("encrypting the same plaintext produced identical ciphertext")
	}
}

// TestTamperedCiphertextFails tests that tampering with the ciphertext results in a decryption failure.
func TestTamperedCiphertextFails(t *testing.T) {
	key := make([]byte, 32)

	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("github-access-token-example")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate someone modifying the encrypted value in the database.
	ciphertext[len(ciphertext)-1] ^= 1

	_, err = Decrypt(key, ciphertext)
	if err == nil {
		t.Fatal("expected tampered ciphertext to fail decryption")
	}
}
