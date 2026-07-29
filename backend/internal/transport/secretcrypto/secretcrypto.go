// Package secretcrypto provides the shared envelope format for secrets stored
// by infrastructure adapters. It has no dependency on any bounded context.
package secretcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

var ErrInvalidCiphertext = errors.New("invalid encrypted secret")

func Encrypt(secret, plaintext string) (string, error) {
	block, err := aes.NewCipher(deriveKey(secret))
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", ErrInvalidCiphertext
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func Decrypt(secret, ciphertext string) (string, error) {
	block, err := aes.NewCipher(deriveKey(secret))
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", ErrInvalidCiphertext
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	return string(plaintext), nil
}

func deriveKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}
