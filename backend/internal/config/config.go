// config 负责“应用配置”的定义、加载与校验。
//
// 设计目标：
// - 配置集中化：端口、DB、JWT、迁移开关、OSS 等统一从 Config 注入
// - 兼容多环境：优先读取 YAML 文件，再用环境变量覆盖部分字段
// - 失败尽早：启动时进行必填项校验，避免运行到一半才暴露配置问题
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 为后端服务的总配置对象，对应 config.yaml 的根节点。
// 结构体 tag 使用 yaml 映射到配置文件字段名。
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	DB      DBConfig      `yaml:"db"`
	Migrate MigrateConfig `yaml:"migrate"`
	JWT     JWTConfig     `yaml:"jwt"`
	Auth    AuthConfig    `yaml:"auth"`
	Crypto  CryptoConfig  `yaml:"crypto"`
	OSS     OSSConfig     `yaml:"oss"`
	Redis   RedisConfig   `yaml:"redis"`
	Log     LogConfig     `yaml:"log"`
	K8s     K8sConfig     `yaml:"k8s"`
}

// ServerConfig 描述 HTTP Server 的启动参数。
// 注意：read/write/idle/shutdown 使用字符串表示 duration，便于在 YAML 中书写（如 "30s"、"2m"）。
type ServerConfig struct {
	Addr            string `yaml:"addr"`
	GinMode         string `yaml:"gin_mode"`
	ReadTimeout     string `yaml:"read_timeout"`
	WriteTimeout    string `yaml:"write_timeout"`
	IdleTimeout     string `yaml:"idle_timeout"`
	ShutdownTimeout string `yaml:"shutdown_timeout"`
}

// DBConfig 描述数据库连接参数。
type DBConfig struct {
	MySQLDSN string `yaml:"mysql_dsn"`
}

// MigrateConfig 控制是否在启动时自动执行数据库迁移。
type MigrateConfig struct {
	Auto bool `yaml:"auto"`
}

// JWTConfig 描述鉴权 token 的签名密钥与有效期。
type JWTConfig struct {
	Secret   string `yaml:"secret"`
	TokenTTL string `yaml:"token_ttl"`
}

// AuthConfig 用于初始化内置管理员账号（首次启动写入 DB）。
type AuthConfig struct {
	AdminUsername string `yaml:"admin_username"`
	AdminPassword string `yaml:"admin_password"`
}

// OSSConfig 描述对象存储（阿里云 OSS）的接入参数。
//
// - Enabled=false 时，相关功能（例如知识库图片上传）会回退到本地 uploads 目录。
// - PublicBaseURL 为空时，服务会根据 bucket+region 推导一个默认公网访问域名。
// CryptoConfig 描述对称加密的 master key 配置。
// 用于加密存储数据库中的敏感字段（kubeconfig、SSH 密码/私钥等）。
// 与 JWT secret 独立，避免密钥复用带来的安全风险。
type CryptoConfig struct {
	MasterKey string `yaml:"master_key"`
}

type OSSConfig struct {
	Enabled         bool   `yaml:"enabled"`
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	Bucket          string `yaml:"bucket"`
	Region          string `yaml:"region"`
	StoragePath     string `yaml:"storage_path"`
	PublicBaseURL   string `yaml:"public_base_url"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// K8sConfig 描述连接 K8s 集群的行为参数。
type K8sConfig struct {
	// InsecureSkipTLSVerify 为 true 时跳过 API Server 的 TLS 证书校验。
	// 适用于自签名证书、证书 SAN 不匹配等开发/测试场景。
	// 生产环境建议保持 false 并确保证书链完整。
	InsecureSkipTLSVerify bool `yaml:"insecure_skip_tls_verify"`
}

type RedisConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Addr       string `yaml:"addr"`
	Password   string `yaml:"password"`
	DB         int    `yaml:"db"`
	DefaultTTL string `yaml:"default_ttl"`
}

// Default 返回一份“可运行的默认配置骨架”。
// 说明：生产环境务必通过 config.yaml 或环境变量覆盖敏感信息（如 JWT secret、DB DSN 等）。
func Default() Config {
	return Config{
		Server: ServerConfig{
			Addr:            ":8080",
			GinMode:         "",
			ReadTimeout:     "30s",
			WriteTimeout:    "30s",
			IdleTimeout:     "60s",
			ShutdownTimeout: "10s",
		},
		DB:      DBConfig{MySQLDSN: ""},
		Migrate: MigrateConfig{Auto: true},
		K8s:     K8sConfig{InsecureSkipTLSVerify: false},
		JWT: JWTConfig{
			Secret:   "dev_secret_change_me",
			TokenTTL: "168h",
		},
		Auth: AuthConfig{
			AdminUsername: "admin",
			AdminPassword: "admin",
		},
		Crypto: CryptoConfig{
			MasterKey: "", // 空值时回退使用 JWT secret（兼容旧配置），建议生产环境独立配置
		},
		OSS: OSSConfig{
			Enabled:       false,
			StoragePath:   "images/",
			PublicBaseURL: "",
		},
		Redis: RedisConfig{
			Enabled:    false,
			Addr:       "127.0.0.1:6379",
			Password:   "",
			DB:         0,
			DefaultTTL: "20s",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

func Load() (Config, error) {
	// 支持通过 CONFIG_PATH 指定配置文件路径（默认读取当前工作目录的 config.yaml）。
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config.yaml"
	}
	return LoadFromFile(path)
}

func LoadFromFile(path string) (Config, error) {
	// 加载策略：
	// 1) 以 Default() 为基础
	// 2) 若文件存在则读取并覆盖 Default
	// 3) 再应用环境变量覆盖（优先级最高）
	cfg := Default()

	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			if err := yaml.Unmarshal(b, &cfg); err != nil {
				return Config{}, err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return Config{}, err
		}
	}

	applyEnvOverrides(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	// 启动期配置校验：避免服务启动后才发现关键配置缺失。
	if c.Server.Addr == "" {
		return errors.New("server.addr is required")
	}
	if c.DB.MySQLDSN == "" {
		return errors.New("db.mysql_dsn is required")
	}
	if c.JWT.Secret == "" {
		return errors.New("jwt.secret is required")
	}
	if c.JWT.TokenTTL == "" {
		return errors.New("jwt.token_ttl is required")
	}
	if c.Auth.AdminUsername == "" || c.Auth.AdminPassword == "" {
		return errors.New("auth.admin_username/auth.admin_password is required")
	}
	// 安全警告：检测默认凭据/密钥，避免生产环境使用不安全配置。
	if c.JWT.Secret == "dev_secret_change_me" {
		_, _ = fmt.Fprintln(os.Stderr, "[WARN] jwt.secret is using default value 'dev_secret_change_me', please change it for production")
	}
	if c.Auth.AdminPassword == "admin" {
		_, _ = fmt.Fprintln(os.Stderr, "[WARN] auth.admin_password is using default value 'admin', please change it for production")
	}
	if strings.TrimSpace(c.Crypto.MasterKey) == "" {
		_, _ = fmt.Fprintln(os.Stderr, "[WARN] crypto.master_key is not set, falling back to jwt.secret for encryption; consider setting a dedicated key for production")
	}
	if c.OSS.Enabled {
		if c.OSS.AccessKeyID == "" || c.OSS.AccessKeySecret == "" {
			return errors.New("oss.access_key_id/oss.access_key_secret is required")
		}
		if c.OSS.Bucket == "" {
			return errors.New("oss.bucket is required")
		}
		if c.OSS.Region == "" {
			return errors.New("oss.region is required")
		}
	}
	if c.Redis.Enabled {
		if strings.TrimSpace(c.Redis.Addr) == "" {
			return errors.New("redis.addr is required when redis.enabled=true")
		}
	}
	if c.Log.Level != "" {
		switch strings.ToLower(strings.TrimSpace(c.Log.Level)) {
		case "debug", "info", "warn", "warning", "error", "fatal":
		default:
			return errors.New("log.level must be one of debug/info/warn/error/fatal")
		}
	}
	if c.Log.Format != "" {
		switch strings.ToLower(strings.TrimSpace(c.Log.Format)) {
		case "json", "console":
		default:
			return errors.New("log.format must be json or console")
		}
	}
	return nil
}

func (c Config) ParsedTokenTTL() time.Duration {
	d, err := time.ParseDuration(c.JWT.TokenTTL)
	if err != nil {
		return 168 * time.Hour
	}
	return d
}

func (c Config) ParsedReadTimeout() time.Duration {
	return parseDurationOrDefault(c.Server.ReadTimeout, 30*time.Second)
}
func (c Config) ParsedWriteTimeout() time.Duration {
	return parseDurationOrDefault(c.Server.WriteTimeout, 30*time.Second)
}
func (c Config) ParsedIdleTimeout() time.Duration {
	return parseDurationOrDefault(c.Server.IdleTimeout, 60*time.Second)
}
func (c Config) ParsedShutdownTimeout() time.Duration {
	return parseDurationOrDefault(c.Server.ShutdownTimeout, 10*time.Second)
}
func (c Config) ParsedRedisDefaultTTL() time.Duration {
	return parseDurationOrDefault(c.Redis.DefaultTTL, 20*time.Second)
}

// EncryptionKey 返回数据加密的有效密钥。
// 优先使用 crypto.master_key；若未配置则回退使用 jwt.secret（兼容旧版配置）。
func (c Config) EncryptionKey() string {
	if strings.TrimSpace(c.Crypto.MasterKey) != "" {
		return c.Crypto.MasterKey
	}
	return c.JWT.Secret
}

func parseDurationOrDefault(raw string, def time.Duration) time.Duration {
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return def
	}
	return d
}

func applyEnvOverrides(cfg *Config) {
	// 环境变量覆盖：便于容器化部署/CI 环境注入配置。
	// 约定：仅覆盖常用字段，避免把所有配置都拆成环境变量导致不可维护。
	if v := os.Getenv("ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("GIN_MODE"); v != "" {
		cfg.Server.GinMode = v
	}

	if v := os.Getenv("MYSQL_DSN"); v != "" {
		cfg.DB.MySQLDSN = v
	} else if v := os.Getenv("DB_DSN"); v != "" {
		cfg.DB.MySQLDSN = v
	}

	if v := os.Getenv("AUTO_MIGRATE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Migrate.Auto = b
		}
	}

	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("JWT_TTL"); v != "" {
		cfg.JWT.TokenTTL = v
	}

	if v := os.Getenv("ADMIN_USERNAME"); v != "" {
		cfg.Auth.AdminUsername = v
	}
	if v := os.Getenv("ADMIN_PASSWORD"); v != "" {
		cfg.Auth.AdminPassword = v
	}

	if v := os.Getenv("CRYPTO_MASTER_KEY"); v != "" {
		cfg.Crypto.MasterKey = v
	}

	if v := os.Getenv("OSS_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.OSS.Enabled = b
		}
	}
	if v := os.Getenv("OSS_ACCESS_KEY_ID"); v != "" {
		cfg.OSS.AccessKeyID = v
	}
	if v := os.Getenv("OSS_ACCESS_KEY_SECRET"); v != "" {
		cfg.OSS.AccessKeySecret = v
	}
	if v := os.Getenv("OSS_BUCKET"); v != "" {
		cfg.OSS.Bucket = v
	}
	if v := os.Getenv("OSS_REGION"); v != "" {
		cfg.OSS.Region = v
	}
	if v := os.Getenv("OSS_STORAGE_PATH"); v != "" {
		cfg.OSS.StoragePath = v
	}
	if v := os.Getenv("OSS_PUBLIC_BASE_URL"); v != "" {
		cfg.OSS.PublicBaseURL = v
	}

	if v := os.Getenv("REDIS_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Redis.Enabled = b
		}
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = n
		}
	}
	if v := os.Getenv("REDIS_DEFAULT_TTL"); v != "" {
		cfg.Redis.DefaultTTL = v
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}
}
