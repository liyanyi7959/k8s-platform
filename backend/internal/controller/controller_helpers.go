// controller_helpers.go 集中存放 controller 包内的共享辅助函数。
//
// 设计目标：
// - 消除各 controller 中 writeServiceErr / parseInt 等工具函数的重复副本
// - 统一 service 层错误到前端业务错误码的映射逻辑
// - 提供可扩展的 WriteServiceErr，支持通过 Option 注入领域特定的错误分支
package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

// ──────────────────────────────────────────────────────────
//  统一错误映射
// ──────────────────────────────────────────────────────────

// errMapping 描述一个"哨兵错误 → 业务码 + 默认消息"的映射条目。
type errMapping struct {
	sentinel error
	code     int
	fallback string
}

// WriteServiceErr 将 service 层错误映射为前端约定的业务错误码。
// 通用映射（ErrInvalidParams → 4000, ErrNotFound → 4040, ErrConflict → 4090）
// 已内置。通过 extras 可在通用映射之前追加领域特定映射，如 K8s / SSH 错误。
//
// 用法：
//
//	WriteServiceErr(c, err)                    // 仅通用映射
//	WriteServiceErr(c, err, k8sErrMappings...) // 追加 K8s 映射
func WriteServiceErr(c *gin.Context, err error, extras ...errMapping) {
	// 优先匹配 extras（领域特定）
	for _, m := range extras {
		if errors.Is(err, m.sentinel) {
			msg := m.fallback
			if um, ok := service.UserMessage(err); ok {
				msg = um
			}
			resp.Fail(c, m.code, msg)
			return
		}
	}
	// 通用映射
	switch {
	case errors.Is(err, service.ErrInvalidParams):
		msg := "参数错误"
		if m, ok := service.UserMessage(err); ok {
			msg = m
		}
		resp.Fail(c, 4000, msg)
	case errors.Is(err, service.ErrNotFound):
		msg := "未找到"
		if m, ok := service.UserMessage(err); ok {
			msg = m
		}
		resp.Fail(c, 4040, msg)
	case errors.Is(err, service.ErrConflict):
		msg := "资源冲突"
		if m, ok := service.UserMessage(err); ok {
			msg = m
		}
		resp.Fail(c, 4090, msg)
	default:
		resp.Fail(c, 5000, "内部错误")
	}
}

// ──────────────────────────────────────────────────────────
//  领域特定错误映射（预定义集合，供各 controller 复用）
// ──────────────────────────────────────────────────────────

// K8sErrMappings 为 K8s 领域的错误映射集合，由 K8sController / ClusterManageController 使用。
var K8sErrMappings = []errMapping{
	{service.ErrK8sNetwork, 5000, "网络连接失败"},
	{service.ErrK8sTimeout, 5000, "连接超时"},
	{service.ErrK8sUnauthorized, 1002, "凭据无效或已过期"},
	{service.ErrK8sForbidden, 1003, "权限不足"},
	{service.ErrK8sTLS, 5000, "证书校验失败"},
	{service.ErrK8s, 5000, "集群访问失败"},
}

// ──────────────────────────────────────────────────────────
//  通用参数解析
// ──────────────────────────────────────────────────────────

// parseInt 将字符串解析为 int，解析失败返回 def 默认值。
func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// parseInt64 将字符串解析为 int64，解析失败返回 def 默认值。
func parseInt64(s string, def int64) int64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}

// parseInt64Ptr 将字符串解析为 *int64，空字符串或解析失败返回 nil。
func parseInt64Ptr(s string) *int64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}
