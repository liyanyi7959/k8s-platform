// errors.go 集中定义 service 层的哨兵错误、ServiceError 结构体与工具函数。
//
// 设计目标：
// - 所有 service 层共用的哨兵错误集中在此文件，消除散落在各文件的重复定义；
// - ServiceError 支持"错误类型 + 用户提示"双层语义，controller 层通过 UserMessage() 获取面向用户的错误消息；
// - 保持 errors.Is / errors.As 语义兼容。
package service

import (
	"errors"
	"strings"

	fleetdomain "k8s-platform-backend/internal/fleet/domain"
	platformapp "k8s-platform-backend/internal/platform/application"
)

// ─── 通用业务哨兵错误 ─────────────────────────────────────
var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrInvalidParams = errors.New("invalid params")
	ErrCrypto        = errors.New("crypto error")
)

// ─── K8s 哨兵错误 ──────────────────────────────────────────
var (
	ErrK8s             = errors.New("k8s error")
	ErrK8sNetwork      = errors.New("k8s network error")
	ErrK8sTimeout      = errors.New("k8s timeout")
	ErrK8sUnauthorized = errors.New("k8s unauthorized")
	ErrK8sForbidden    = errors.New("k8s forbidden")
	ErrK8sTLS          = errors.New("k8s tls error")
)

// ─── 任务哨兵错误 ───────────────────────────────────────────
var (
	ErrTaskNotFound     = platformapp.ErrTaskNotFound
	ErrTaskCannotCancel = platformapp.ErrTaskCannotCancel
)

// ─── ServiceError：错误类型 + 面向用户的消息 ─────────────────

// ServiceError 为 service 层的结构化错误类型。
// Kind 表示错误类型（如 ErrNotFound/ErrInvalidParams），支持 errors.Is 判断；
// Message 为面向用户的中文提示，controller 层通过 UserMessage() 读取。
type ServiceError struct {
	Kind    error
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

func (e *ServiceError) Unwrap() error {
	return e.Kind
}

// UserMessage lets bounded-context policies obtain the API-safe error text
// through a small interface, without importing the legacy error package.
func (e *ServiceError) UserMessage() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.Message)
}

// ErrWithMessage 创建一个携带用户提示的 ServiceError。
// kind 为哨兵错误，message 为面向用户的中文说明。
func ErrWithMessage(kind error, message string) error {
	if kind == nil {
		return errors.New(strings.TrimSpace(message))
	}
	msg := strings.TrimSpace(message)
	if msg == "" {
		return kind
	}
	return &ServiceError{Kind: kind, Message: msg}
}

// UserMessage 从 error 中提取用户可见的错误消息。
// 若 error 为 ServiceError 且 Message 非空，返回 (message, true)；否则 ("", false)。
func UserMessage(err error) (string, bool) {
	var se *ServiceError
	if errors.As(err, &se) && se != nil {
		msg := strings.TrimSpace(se.Message)
		if msg != "" {
			return msg, true
		}
	}
	var carrier interface{ UserMessage() string }
	if errors.As(err, &carrier) && carrier != nil {
		if msg := strings.TrimSpace(carrier.UserMessage()); msg != "" {
			return msg, true
		}
	}
	return "", false
}

// legacyClusterError translates Fleet domain failures for retained runtime
// code that still exposes the legacy service error vocabulary.
func legacyClusterError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, fleetdomain.ErrValidation):
		return ErrWithMessage(ErrInvalidParams, err.Error())
	case errors.Is(err, fleetdomain.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, fleetdomain.ErrConflict):
		return ErrConflict
	case errors.Is(err, fleetdomain.ErrCrypto):
		return ErrCrypto
	default:
		return err
	}
}

// ─── PageResult：通用分页返回结构 ────────────────────────────

// PageResult 为通用的分页返回结构，被所有 list 接口共用。
type PageResult[T any] struct {
	List     []T `json:"list"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
