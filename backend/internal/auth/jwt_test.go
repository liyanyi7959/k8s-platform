package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ────────── NewManager ──────────

func TestNewManager_DefaultSecret(t *testing.T) {
	m := NewManager("")
	if string(m.secret) != "dev_secret_change_me" {
		t.Errorf("default secret = %q, want %q", string(m.secret), "dev_secret_change_me")
	}
}

func TestNewManager_CustomSecret(t *testing.T) {
	m := NewManager("my-prod-secret")
	if string(m.secret) != "my-prod-secret" {
		t.Errorf("secret = %q, want %q", string(m.secret), "my-prod-secret")
	}
}

// ────────── IssueToken / ParseToken 回环 ──────────

func TestJWT_RoundTrip(t *testing.T) {
	m := NewManager("test-secret-32bytes-long-enough!")

	params := Claims{
		UserID:   42,
		Username: "admin",
		Roles:    []string{"admin", "viewer"},
		Perms:    []string{"sys:user_admin", "server:read"},
	}

	tokenStr, err := m.IssueToken(params, 1*time.Hour)
	if err != nil {
		t.Fatalf("IssueToken error: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := m.ParseToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}

	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Errorf("Username = %q, want %q", claims.Username, "admin")
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "admin" {
		t.Errorf("Roles = %v, want [admin viewer]", claims.Roles)
	}
	if len(claims.Perms) != 2 || claims.Perms[1] != "server:read" {
		t.Errorf("Perms = %v, want [sys:user_admin server:read]", claims.Perms)
	}
}

func TestJWT_ExpiresAt(t *testing.T) {
	m := NewManager("secret")
	params := Claims{UserID: 1, Username: "u"}

	tokenStr, _ := m.IssueToken(params, 2*time.Hour)
	claims, err := m.ParseToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt should not be nil")
	}
	// ExpiresAt 应大约在 2 小时后（容忍 5 秒误差）
	diff := time.Until(claims.ExpiresAt.Time)
	if diff < 119*time.Minute || diff > 121*time.Minute {
		t.Errorf("ExpiresAt diff = %v, want ~2h", diff)
	}
}

// ────────── 过期 token ──────────

func TestJWT_ExpiredToken(t *testing.T) {
	m := NewManager("secret")
	// 签发一个已经过期的 token（ttl 为负）
	tokenStr, _ := m.IssueToken(Claims{UserID: 1, Username: "u"}, -1*time.Hour)
	_, err := m.ParseToken(tokenStr)
	if err == nil {
		t.Fatal("expired token should fail ParseToken")
	}
}

// ────────── 错误 secret ──────────

func TestJWT_WrongSecret(t *testing.T) {
	m1 := NewManager("secret-a")
	m2 := NewManager("secret-b")

	tokenStr, _ := m1.IssueToken(Claims{UserID: 1, Username: "u"}, 1*time.Hour)
	_, err := m2.ParseToken(tokenStr)
	if err == nil {
		t.Fatal("wrong secret should fail ParseToken")
	}
}

// ────────── 篡改 token ──────────

func TestJWT_TamperedToken(t *testing.T) {
	m := NewManager("secret")
	tokenStr, _ := m.IssueToken(Claims{UserID: 1, Username: "u"}, 1*time.Hour)

	// 篡改 payload 中的一个字符
	tampered := tokenStr[:len(tokenStr)-4] + "XXXX"
	_, err := m.ParseToken(tampered)
	if err == nil {
		t.Fatal("tampered token should fail ParseToken")
	}
}

// ────────── 算法降级攻击 ──────────

func TestJWT_AlgorithmDowngrade(t *testing.T) {
	// 手动用 "none" 算法签发 token，应被 ParseToken 拒绝
	claims := &Claims{
		UserID:   99,
		Username: "hacker",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to create none-signed token: %v", err)
	}

	m := NewManager("secret")
	_, err = m.ParseToken(tokenStr)
	if err == nil {
		t.Fatal("none-signed token should be rejected")
	}
}

// ────────── 空 token ──────────

func TestJWT_EmptyToken(t *testing.T) {
	m := NewManager("secret")
	_, err := m.ParseToken("")
	if err == nil {
		t.Fatal("empty token should fail")
	}
}

// ────────── 无角色/权限的最小 Claims ──────────

func TestJWT_MinimalClaims(t *testing.T) {
	m := NewManager("secret")
	tokenStr, _ := m.IssueToken(Claims{UserID: 7, Username: "basic"}, 1*time.Hour)
	claims, err := m.ParseToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if claims.UserID != 7 {
		t.Errorf("UserID = %d, want 7", claims.UserID)
	}
	if len(claims.Roles) != 0 {
		t.Errorf("Roles should be empty, got %v", claims.Roles)
	}
	if len(claims.Perms) != 0 {
		t.Errorf("Perms should be empty, got %v", claims.Perms)
	}
}
