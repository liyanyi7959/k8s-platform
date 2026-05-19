// service 包含核心业务逻辑（controller 的“下游”）。
//
// 本文件提供对称加密工具，用于加密落库的敏感数据，例如：
// - kubeconfig（集群凭据）
//
// 安全说明：
// - 采用 AES-256-GCM（具备机密性 + 完整性校验）
// - 使用随机 nonce（每次加密都不同），避免相同明文得到相同密文
// - secret 通过 SHA256 派生到固定长度密钥，便于配置任意长度的 master key
package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

// ErrCrypto 已统一迁移至 errors.go。

func deriveKey(secret string) []byte {
	// 将任意长度的 secret 固定映射为 32 字节，用于 AES-256。
	// 这里使用 SHA256 是为了避免对密钥长度的额外约束。
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func encryptText(secret, plaintext string) (string, error) {
	// encryptText 将明文加密后返回 base64 字符串，便于存入文本字段（longtext）。
	key := deriveKey(secret)
	block, err := aes.NewCipher(key)
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

	// GCM 输出为：密文 + tag（认证标签），Open 时会校验 tag，篡改将解密失败。
	ct := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	out := append(nonce, ct...)
	return base64.StdEncoding.EncodeToString(out), nil
}

func decryptText(secret, ciphertextB64 string) (string, error) {
	// decryptText 对应 encryptText 的逆过程：输入 base64 密文，输出明文。
	// 失败统一返回 ErrCrypto，避免向外暴露具体失败原因（降低攻击面）。
	key := deriveKey(secret)
	block, err := aes.NewCipher(key)
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
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", ErrCrypto
	}
	nonce := raw[:ns]
	ct := raw[ns:]

	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", ErrCrypto
	}
	return string(pt), nil
}
