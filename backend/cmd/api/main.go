// 程序入口（HTTP API 服务）。
//
// 该文件负责：
// - 读取配置（支持文件与环境变量覆盖）
// - 初始化数据库连接与迁移
// - 组装业务依赖（service/controller/router）
// - 启动 HTTP Server，并处理优雅退出
//
// 说明：
// - 业务逻辑尽量放在 internal/service 与 internal/controller 中，这里只做“胶水层”。
//
// @title 星枢K8S管理平台 API
// @version 1.0
// @description 星枢K8S管理平台后端接口文档（统一响应结构：code/message/data）
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"k8s-platform-backend/internal/auth"
	"k8s-platform-backend/internal/config"
	"k8s-platform-backend/internal/controller"
	"k8s-platform-backend/internal/db"
	"k8s-platform-backend/internal/router"
	"k8s-platform-backend/internal/service"
)

func main() {
	// 1) 加载配置：支持 CONFIG_PATH 指定配置文件路径（默认 config.yaml）。
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger, err := newLogger(cfg.Log)
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
	defer func() { _ = logger.Sync() }()

	// 2) 设置 Gin 运行模式（debug/release/test），由配置注入。
	if cfg.Server.GinMode != "" {
		gin.SetMode(cfg.Server.GinMode)
	}

	// 3) 初始化数据库连接（GORM + *sql.DB）。
	gdb, sdb, err := db.Open(db.Config{MySQLDSN: cfg.DB.MySQLDSN})
	if err != nil {
		zap.L().Fatal("open_db_failed", zap.Error(err))
	}
	defer func() { _ = sdb.Close() }()

	// 4) 迁移（可通过 migrate.auto 开关关闭）。
	// - 迁移文件位于 internal/db/migrations/*.sql，并通过 embed 打包进二进制。
	// - schema_migrations 记录已执行版本，保证幂等。
	if cfg.Migrate.Auto {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if err := db.Migrate(ctx, gdb); err != nil {
			zap.L().Fatal("db_migrate_failed", zap.Error(err))
		}
	}

	// 5) 组装依赖：JWT 管理器、RBAC 服务、控制器与路由。
	jwtMgr := auth.NewManager(cfg.JWT.Secret)
	cacheStore, err := service.NewRedisCacheStore(cfg.Redis)
	if err != nil {
		zap.L().Fatal("init_redis_failed", zap.Error(err))
	}
	defer func() { _ = cacheStore.Close() }()

	rbacSvc := service.NewRbacService(gdb, cacheStore, 15*time.Minute)
	// 内置初始化：首次启动自动创建管理员用户/角色/权限点，确保系统可登录可用。
	if err := service.EnsureBuiltinRBAC(gdb, cfg.Auth.AdminUsername, cfg.Auth.AdminPassword); err != nil {
		zap.L().Fatal("ensure_builtin_rbac_failed", zap.Error(err))
	}
	authCtl := controller.NewAuthController(jwtMgr, rbacSvc, cfg.ParsedTokenTTL())

	r, err := router.New(router.Deps{
		DB:             gdb,
		JWTMgr:         jwtMgr,
		AuthCtl:        authCtl,
		RbacSvc:        rbacSvc,
		EncryptionKey:  cfg.EncryptionKey(),
		CacheStore:     cacheStore,
		CacheTTL:       cfg.ParsedRedisDefaultTTL(),
		K8sInsecureTLS: cfg.K8s.InsecureSkipTLSVerify,
	})
	if err != nil {
		zap.L().Fatal("new_router_failed", zap.Error(err))
	}

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  cfg.ParsedReadTimeout(),
		WriteTimeout: cfg.ParsedWriteTimeout(),
		IdleTimeout:  cfg.ParsedIdleTimeout(),
	}

	go func() {
		// 6) 启动 HTTP 服务。ListenAndServe 为阻塞调用，这里放到 goroutine 中。
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("http_server_failed", zap.Error(err))
		}
	}()

	// 7) 优雅关闭：接收 SIGINT/SIGTERM 后，按配置的超时时间进行 shutdown。
	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-stopCtx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ParsedShutdownTimeout())
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func newLogger(cfg config.LogConfig) (*zap.Logger, error) {
	level := strings.ToLower(strings.TrimSpace(cfg.Level))
	if level == "" {
		level = "info"
	}
	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	if format == "" {
		format = "json"
	}

	var lvl zapcore.Level
	switch level {
	case "debug":
		lvl = zapcore.DebugLevel
	case "info":
		lvl = zapcore.InfoLevel
	case "warn", "warning":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	case "fatal":
		lvl = zapcore.FatalLevel
	default:
		lvl = zapcore.InfoLevel
	}

	var zcfg zap.Config
	switch format {
	case "console":
		zcfg = zap.NewDevelopmentConfig()
		zcfg.Encoding = "console"
	default:
		zcfg = zap.NewProductionConfig()
		zcfg.Encoding = "json"
	}
	zcfg.Level = zap.NewAtomicLevelAt(lvl)
	return zcfg.Build()
}
