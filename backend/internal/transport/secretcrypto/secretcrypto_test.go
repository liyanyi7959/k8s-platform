package secretcrypto

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	for _, plaintext := range []string{"hello world", "", "中文测试 🚀", string(make([]byte, 4096))} {
		ciphertext, err := Encrypt("test-master-key", plaintext)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}
		actual, err := Decrypt("test-master-key", ciphertext)
		if err != nil || actual != plaintext {
			t.Fatalf("Decrypt() = (%q, %v), want (%q, nil)", actual, err, plaintext)
		}
	}
}

func TestEncryptUsesFreshNonce(t *testing.T) {
	first, err := Encrypt("key", "identical")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Encrypt("key", "identical")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("equal plaintext must not produce equal ciphertext")
	}
}

func TestDecryptRejectsInvalidCiphertext(t *testing.T) {
	ciphertext, err := Encrypt("correct-key", "payload")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt("wrong-key", ciphertext); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("wrong key error = %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)/2] ^= 0xff
	if _, err := Decrypt("correct-key", base64.StdEncoding.EncodeToString(raw)); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("tampered ciphertext error = %v", err)
	}
	for _, invalid := range []string{"not-valid-base64!!!", base64.StdEncoding.EncodeToString([]byte("tiny"))} {
		if _, err := Decrypt("key", invalid); !errors.Is(err, ErrInvalidCiphertext) {
			t.Fatalf("invalid ciphertext error = %v", err)
		}
	}
}
