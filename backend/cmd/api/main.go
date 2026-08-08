// 程序入口（HTTP API 服务）。
//
// 该文件负责：
// - 读取配置（支持文件与环境变量覆盖）
// - 初始化数据库连接与迁移
// - 组装业务依赖（service/controller/router）
// - 启动 HTTP Server，并处理优雅退出
//
// 说明：
// - 业务逻辑放在 internal/<context>；未迁移实现统一位于 internal/legacy。
//
// @title 星枢K8S管理平台 API
// @version 1.0
// @description 星枢K8S管理平台后端接口文档（统一响应结构：code/message/data）
// @BasePath /api/v2
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

	auditmysql "k8s-platform-backend/internal/audit/adapters/mysql"
	auditapp "k8s-platform-backend/internal/audit/application"
	auditdomain "k8s-platform-backend/internal/audit/domain"
	"k8s-platform-backend/internal/auth"
	changemysql "k8s-platform-backend/internal/change/adapters/mysql"
	changeapp "k8s-platform-backend/internal/change/application"
	"k8s-platform-backend/internal/config"
	"k8s-platform-backend/internal/db"
	iamhttp "k8s-platform-backend/internal/iam/adapters/http"
	iammysql "k8s-platform-backend/internal/iam/adapters/mysql"
	iamsmtp "k8s-platform-backend/internal/iam/adapters/smtp"
	iamapp "k8s-platform-backend/internal/iam/application"
	"k8s-platform-backend/internal/router"
	cachetransport "k8s-platform-backend/internal/transport/cache"
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
	cacheStore, err := cachetransport.NewRedisCacheStore(cfg.Redis)
	if err != nil {
		zap.L().Fatal("init_redis_failed", zap.Error(err))
	}
	defer func() { _ = cacheStore.Close() }()

	auditSvc := auditapp.NewService(auditmysql.NewRepository(gdb))
	iamAuthRepository := iammysql.NewAuthRepository(gdb)
	iamAuthSvc := iamapp.NewAuthService(iamAuthRepository, iammysql.BcryptHasher{}, cacheStore, 15*time.Minute)
	captchaSvc := iamapp.NewCaptchaService(cacheStore)
	loginAttemptSvc := iamapp.NewLoginAttemptService(cacheStore)
	mailSvc := iamsmtp.NewMailService(cfg.Mail)
	pwdResetSvc := iamapp.NewPasswordResetService(iamAuthRepository, iammysql.BcryptHasher{}, cacheStore, mailSvc)
	// 内置初始化：首次启动自动创建管理员用户/角色/权限点，确保系统可登录可用。
	if err := iammysql.EnsureBuiltinRBAC(gdb, cfg.Auth.AdminUsername, cfg.Auth.AdminPassword); err != nil {
		zap.L().Fatal("ensure_builtin_rbac_failed", zap.Error(err))
	}
	if err := iamAuthSvc.InvalidateRole(context.Background(), "admin"); err != nil {
		zap.L().Warn("invalidate_builtin_rbac_cache_failed", zap.Error(err))
	}
	authAuditRecorder := iamhttp.AuthAuditRecorder(func(ctx context.Context, event iamhttp.AuthAuditEntry) {
		auditSvc.Record(ctx, auditdomain.Entry{
			UserID:       event.UserID,
			Username:     event.Username,
			Action:       event.Action,
			Resource:     event.Resource,
			ResourceName: event.ResourceName,
			Path:         event.Path,
			StatusCode:   event.StatusCode,
			Detail:       event.Detail,
			ClientIP:     event.ClientIP,
			RequestID:    event.RequestID,
		})
	})
	authCtl := iamhttp.NewAuthController(jwtMgr, iamAuthSvc, authAuditRecorder, captchaSvc, loginAttemptSvc, pwdResetSvc, cfg.ParsedTokenTTL())

	r, err := router.New(router.Deps{
		DB:                  gdb,
		JWTMgr:              jwtMgr,
		AuthCtl:             authCtl,
		AuthorizationReader: iamAuthSvc,
		IAMAuthService:      iamAuthSvc,
		EncryptionKey:       cfg.EncryptionKey(),
		AIUploadDir:         cfg.AI.UploadDir,
		CacheStore:          cacheStore,
		CacheTTL:            cfg.ParsedRedisDefaultTTL(),
		K8sInsecureTLS:      cfg.K8s.InsecureSkipTLSVerify,
	})
	if err != nil {
		zap.L().Fatal("new_router_failed", zap.Error(err))
	}

	// 6) Change 执行回收 worker：进程内轮询超时执行，兜底僵尸任务，
	//    只做状态与审计，不执行任何实际操作。
	changeRecoveryCtx, stopChangeRecovery := context.WithCancel(context.Background())
	changeSvc := changeapp.NewService(changemysql.NewRepository(gdb))
	go changeapp.RunExecutionRecoveryLoop(changeRecoveryCtx, changeSvc, 5*time.Minute, 30*time.Minute)
	defer stopChangeRecovery()

	// 7) Audit 保留策略 worker：按保留天数定期清理过期审计记录，防止日志无限增长。
	auditRetentionCtx, stopAuditRetention := context.WithCancel(context.Background())
	go auditapp.RunRetentionLoop(auditRetentionCtx, auditSvc, auditdomain.RetentionPolicy{Days: 180}, 24*time.Hour)
	defer stopAuditRetention()

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
