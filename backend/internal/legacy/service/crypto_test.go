package service

import (
	"encoding/base64"
	"errors"
	"testing"
)

// ────────── deriveKey ──────────

func TestDeriveKey_Deterministic(t *testing.T) {
	k1 := deriveKey("my-secret")
	k2 := deriveKey("my-secret")
	if len(k1) != 32 {
		t.Fatalf("deriveKey length = %d, want 32", len(k1))
	}
	for i := range k1 {
		if k1[i] != k2[i] {
			t.Fatal("deriveKey should be deterministic for same input")
		}
	}
}

func TestDeriveKey_DifferentSecrets(t *testing.T) {
	k1 := deriveKey("secret-a")
	k2 := deriveKey("secret-b")
	same := true
	for i := range k1 {
		if k1[i] != k2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different secrets should produce different keys")
	}
}

// ────────── encryptText / decryptText 回环 ──────────

func TestCrypto_RoundTrip(t *testing.T) {
	secret := "test-master-key-2024"
	cases := []string{
		"hello world",
		"",                         // 空明文
		"中文测试 🚀",                   // unicode
		string(make([]byte, 4096)), // 较长内容
	}
	for _, plain := range cases {
		ct, err := encryptText(secret, plain)
		if err != nil {
			t.Fatalf("encryptText(%q) error: %v", plain, err)
		}
		got, err := decryptText(secret, ct)
		if err != nil {
			t.Fatalf("decryptText error: %v", err)
		}
		if got != plain {
			t.Errorf("round-trip mismatch: got %q, want %q", got, plain)
		}
	}
}

func TestCrypto_DifferentCiphertext(t *testing.T) {
	// 由于随机 nonce，相同明文加密两次应得到不同密文
	secret := "same-key"
	ct1, _ := encryptText(secret, "identical")
	ct2, _ := encryptText(secret, "identical")
	if ct1 == ct2 {
		t.Error("same plaintext should produce different ciphertext due to random nonce")
	}
}

func TestCrypto_WrongSecret(t *testing.T) {
	ct, err := encryptText("correct-key", "sensitive data")
	if err != nil {
		t.Fatalf("encrypt error: %v", err)
	}
	_, err = decryptText("wrong-key", ct)
	if err == nil {
		t.Fatal("decrypt with wrong secret should fail")
	}
	if !errors.Is(err, ErrCrypto) {
		t.Errorf("error should be ErrCrypto, got %v", err)
	}
}

func TestCrypto_TamperedCiphertext(t *testing.T) {
	ct, _ := encryptText("key123", "payload")
	// 篡改密文中间部分
	raw, _ := base64.StdEncoding.DecodeString(ct)
	if len(raw) > 5 {
		raw[len(raw)/2] ^= 0xFF
	}
	tampered := base64.StdEncoding.EncodeToString(raw)

	_, err := decryptText("key123", tampered)
	if err == nil {
		t.Fatal("tampered ciphertext should fail")
	}
	if !errors.Is(err, ErrCrypto) {
		t.Errorf("error should be ErrCrypto, got %v", err)
	}
}

func TestCrypto_InvalidBase64(t *testing.T) {
	_, err := decryptText("key", "not-valid-base64!!!")
	if err == nil {
		t.Fatal("invalid base64 should fail")
	}
	if !errors.Is(err, ErrCrypto) {
		t.Errorf("error should be ErrCrypto, got %v", err)
	}
}

func TestCrypto_TooShortCiphertext(t *testing.T) {
	// nonce 至少 12 字节（GCM），传一个更短的
	short := base64.StdEncoding.EncodeToString([]byte("tiny"))
	_, err := decryptText("key", short)
	if err == nil {
		t.Fatal("too-short ciphertext should fail")
	}
	if !errors.Is(err, ErrCrypto) {
		t.Errorf("error should be ErrCrypto, got %v", err)
	}
}
