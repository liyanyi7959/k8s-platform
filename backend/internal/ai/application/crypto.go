package application

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

func deriveKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func DecryptText(secret, ciphertextB64 string) (string, error) {
	block, err := aes.NewCipher(deriveKey(secret))
	if err != nil {
		return "", ErrCrypto
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrCrypto
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", ErrCrypto
	}
	if len(raw) < gcm.NonceSize() {
		return "", ErrCrypto
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", ErrCrypto
	}
	return string(plaintext), nil
}

func encryptText(secret, plaintext string) (string, error) {
	block, err := aes.NewCipher(deriveKey(secret))
	if err != nil {
		return "", ErrCrypto
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrCrypto
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", ErrCrypto
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}
