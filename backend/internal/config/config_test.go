package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// ────────── Default ──────────

func TestDefault_NonEmpty(t *testing.T) {
	cfg := Default()
	if cfg.Server.Addr == "" {
		t.Error("Server.Addr should not be empty")
	}
	if cfg.JWT.Secret == "" {
		t.Error("JWT.Secret should not be empty")
	}
	if cfg.Auth.AdminUsername == "" || cfg.Auth.AdminPassword == "" {
		t.Error("Auth admin credentials should not be empty")
	}
}

// ────────── Validate ──────────

func TestValidate_OK_WithDSN(t *testing.T) {
	cfg := Default()
	cfg.DB.MySQLDSN = "user:pass@tcp(127.0.0.1:3306)/db"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
}

func TestValidate_MissingAddr(t *testing.T) {
	cfg := Default()
	cfg.DB.MySQLDSN = "user:pass@tcp(127.0.0.1:3306)/db"
	cfg.Server.Addr = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail when Server.Addr is empty")
	}
}

func TestValidate_MissingDSN(t *testing.T) {
	cfg := Default()
	cfg.DB.MySQLDSN = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail when DB.MySQLDSN is empty")
	}
}

func TestValidate_MissingJWTSecret(t *testing.T) {
	cfg := Default()
	cfg.DB.MySQLDSN = "dsn"
	cfg.JWT.Secret = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail when JWT.Secret is empty")
	}
}

func TestValidate_MissingTokenTTL(t *testing.T) {
	cfg := Default()
	cfg.DB.MySQLDSN = "dsn"
	cfg.JWT.TokenTTL = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail when JWT.TokenTTL is empty")
	}
}

func TestValidate_MissingAdmin(t *testing.T) {
	cfg := Default()
	cfg.DB.MySQLDSN = "dsn"
	cfg.Auth.AdminUsername = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail when AdminUsername is empty")
	}
}

func TestValidate_RedisEnabled_MissingAddr(t *testing.T) {
	cfg := Default()
	cfg.DB.MySQLDSN = "dsn"
	cfg.Redis.Enabled = true
	cfg.Redis.Addr = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail when Redis enabled without addr")
	}
}

func TestValidate_LogLevel_Invalid(t *testing.T) {
	cfg := Default()
	cfg.DB.MySQLDSN = "dsn"
	cfg.Log.Level = "INVALID"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail for invalid log level")
	}
}

func TestValidate_LogLevel_Valid(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "warning", "error", "fatal"} {
		cfg := Default()
		cfg.DB.MySQLDSN = "dsn"
		cfg.Log.Level = level
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() failed for valid log level %q: %v", level, err)
		}
	}
}

func TestValidate_LogFormat_Invalid(t *testing.T) {
	cfg := Default()
	cfg.DB.MySQLDSN = "dsn"
	cfg.Log.Format = "xml"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail for invalid log format")
	}
}

// ────────── EncryptionKey ──────────

func TestEncryptionKey_UseMasterKey(t *testing.T) {
	cfg := Default()
	cfg.Crypto.MasterKey = "my-secret-master-key"
	if cfg.EncryptionKey() != "my-secret-master-key" {
		t.Errorf("EncryptionKey() = %q, want master key", cfg.EncryptionKey())
	}
}

func TestEncryptionKey_FallbackToJWT(t *testing.T) {
	cfg := Default()
	cfg.Crypto.MasterKey = ""
	cfg.JWT.Secret = "jwt-fallback"
	if cfg.EncryptionKey() != "jwt-fallback" {
		t.Errorf("EncryptionKey() = %q, want jwt-fallback", cfg.EncryptionKey())
	}
}

func TestEncryptionKey_WhitespaceMasterKey(t *testing.T) {
	cfg := Default()
	cfg.Crypto.MasterKey = "   "
	cfg.JWT.Secret = "jwt-used"
	if cfg.EncryptionKey() != "jwt-used" {
		t.Errorf("EncryptionKey() should fall back when MasterKey is whitespace only")
	}
}

// ────────── ParsedDuration 系列 ──────────

func TestParsedTokenTTL_Default(t *testing.T) {
	cfg := Default()
	d := cfg.ParsedTokenTTL()
	if d != 168*time.Hour {
		t.Errorf("ParsedTokenTTL() = %v, want 168h", d)
	}
}

func TestParsedTokenTTL_Custom(t *testing.T) {
	cfg := Default()
	cfg.JWT.TokenTTL = "24h"
	if cfg.ParsedTokenTTL() != 24*time.Hour {
		t.Errorf("ParsedTokenTTL() = %v, want 24h", cfg.ParsedTokenTTL())
	}
}

func TestParsedTokenTTL_Invalid(t *testing.T) {
	cfg := Default()
	cfg.JWT.TokenTTL = "not-a-duration"
	if cfg.ParsedTokenTTL() != 168*time.Hour {
		t.Error("ParsedTokenTTL() should fall back to 168h for invalid input")
	}
}

func TestParsedReadTimeout(t *testing.T) {
	cfg := Default()
	cfg.Server.ReadTimeout = "5s"
	if cfg.ParsedReadTimeout() != 5*time.Second {
		t.Errorf("ParsedReadTimeout() = %v, want 5s", cfg.ParsedReadTimeout())
	}
}

func TestParsedRedisDefaultTTL(t *testing.T) {
	cfg := Default()
	cfg.Redis.DefaultTTL = "1m"
	if cfg.ParsedRedisDefaultTTL() != time.Minute {
		t.Errorf("ParsedRedisDefaultTTL() = %v, want 1m", cfg.ParsedRedisDefaultTTL())
	}
}

func TestParsedDuration_Empty(t *testing.T) {
	cfg := Default()
	cfg.Server.ReadTimeout = ""
	if cfg.ParsedReadTimeout() != 30*time.Second {
		t.Errorf("ParsedReadTimeout() should default to 30s when empty")
	}
}

// ────────── 环境变量覆盖 ──────────

func TestApplyEnvOverrides(t *testing.T) {
	// 设置临时环境变量
	envs := map[string]string{
		"ADDR":       ":9090",
		"GIN_MODE":   "release",
		"MYSQL_DSN":  "env-dsn",
		"JWT_SECRET": "env-secret",
		"JWT_TTL":    "48h",
	}
	for k, v := range envs {
		t.Setenv(k, v)
	}

	cfg := Default()
	applyEnvOverrides(&cfg)

	if cfg.Server.Addr != ":9090" {
		t.Errorf("Addr = %q, want :9090", cfg.Server.Addr)
	}
	if cfg.Server.GinMode != "release" {
		t.Errorf("GinMode = %q, want release", cfg.Server.GinMode)
	}
	if cfg.DB.MySQLDSN != "env-dsn" {
		t.Errorf("MySQLDSN = %q, want env-dsn", cfg.DB.MySQLDSN)
	}
	if cfg.JWT.Secret != "env-secret" {
		t.Errorf("JWT.Secret = %q, want env-secret", cfg.JWT.Secret)
	}
	if cfg.JWT.TokenTTL != "48h" {
		t.Errorf("JWT.TokenTTL = %q, want 48h", cfg.JWT.TokenTTL)
	}
}

func TestApplyEnvOverrides_DB_DSN_Alt(t *testing.T) {
	// DB_DSN 作为替代环境变量名
	t.Setenv("DB_DSN", "alt-dsn")
	cfg := Default()
	applyEnvOverrides(&cfg)
	if cfg.DB.MySQLDSN != "alt-dsn" {
		t.Errorf("MySQLDSN = %q, want alt-dsn", cfg.DB.MySQLDSN)
	}
}

func TestApplyEnvOverrides_CryptoMasterKey(t *testing.T) {
	t.Setenv("CRYPTO_MASTER_KEY", "env-master-key")
	cfg := Default()
	applyEnvOverrides(&cfg)
	if cfg.Crypto.MasterKey != "env-master-key" {
		t.Errorf("Crypto.MasterKey = %q, want env-master-key", cfg.Crypto.MasterKey)
	}
}

func TestApplyEnvOverrides_AutoMigrate(t *testing.T) {
	t.Setenv("AUTO_MIGRATE", "false")
	cfg := Default()
	applyEnvOverrides(&cfg)
	if cfg.Migrate.Auto {
		t.Error("Migrate.Auto should be false")
	}
}

// ────────── LoadFromFile ──────────

func TestLoadFromFile_NonExistent_UsesDefault(t *testing.T) {
	// 确保：文件不存在时使用默认值
	// 需要先清理可能干扰的环境变量
	os.Unsetenv("MYSQL_DSN")
	os.Unsetenv("DB_DSN")

	_, err := LoadFromFile("/tmp/does-not-exist-xyzzy.yaml")
	// 应该因 db.mysql_dsn 为空而失败
	if err == nil {
		t.Error("LoadFromFile should fail when DSN is empty")
	}
	if !strings.Contains(err.Error(), "mysql_dsn") {
		t.Errorf("error should mention mysql_dsn, got: %v", err)
	}
}

func TestLoadFromFile_EmptyPath_UsesDefault(t *testing.T) {
	os.Unsetenv("MYSQL_DSN")
	os.Unsetenv("DB_DSN")

	_, err := LoadFromFile("")
	// 空路径直接使用 Default()，因 db.mysql_dsn 为空而失败
	if err == nil {
		t.Error("LoadFromFile(\"\") should fail when DSN is empty")
	}
}
