// deps.go 定义路由层的依赖容器。
//
// 设计目标：
// - 将 router.New() 所需的 8+ 个参数聚合为一个 Deps 结构体
// - 显式区分"必需依赖"与"可选依赖"（DB 相关的在 DB 为 nil 时跳过）
// - 为后续添加新依赖（如 metrics、trace）提供统一扩展点
package router

import (
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/auth"
	"k8s-platform-backend/internal/config"
	"k8s-platform-backend/internal/controller"
	"k8s-platform-backend/internal/service"
)

// Deps 聚合路由层构建所需的全部外部依赖。
//
// 用法：
//
//	deps := router.Deps{DB: gdb, JWTMgr: jwtMgr, ...}
//	r, err := router.New(deps)
type Deps struct {
	// ── 必需 ──
	JWTMgr  *auth.Manager
	AuthCtl *controller.AuthController
	RbacCtl *controller.RbacController
	RbacSvc *service.RbacService

	// ── 可选：DB 存在时才注入 ──
	DB             *gorm.DB
	EncryptionKey  string
	OSSCfg         config.OSSConfig
	CacheStore     service.CacheStore
	CacheTTL       time.Duration
	K8sInsecureTLS bool
}
