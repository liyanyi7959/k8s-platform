package problem

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TypeBaseURI is the stable namespace for the API's RFC 9457 problem types.
// Clients may use the URI as a machine-readable error contract.
const TypeBaseURI = "https://aiops.local/problems/"

// RequestIDHeader is echoed when the request has an id, so callers can match a
// problem response with the corresponding server-side trace.
const RequestIDHeader = "X-Request-ID"

// Kind identifies a problem in the platform's public API contract. Handlers
// should use WriteKind instead of repeating an HTTP status, type URI and title.
type Kind string

const (
	KindInvalidRequest Kind = "invalid-request"
	KindUnauthorized   Kind = "unauthorized"
	KindForbidden      Kind = "forbidden"
	KindNotFound       Kind = "not-found"
	KindConflict       Kind = "conflict"
	KindValidation     Kind = "validation"
	// KindClusterCredentialInvalid means the platform session is valid, but the
	// configured credentials for a downstream Kubernetes cluster are not.
	KindClusterCredentialInvalid  Kind = "cluster-credential-invalid"
	KindIncidentNotFound          Kind = "incident-not-found"
	KindVersionConflict           Kind = "version-conflict"
	KindInvalidIncidentTransition Kind = "invalid-incident-transition"
	KindDomainValidation          Kind = "domain-validation"
	KindInternal                  Kind = "internal"
)

// Definition is the status and localized title associated with a public
// problem type. Its URI is derived from TypeBaseURI and therefore remains
// consistent across every v2 endpoint.
type Definition struct {
	Kind   Kind
	Status int
	Title  string
}

// TypeURI returns the canonical URI for this problem definition.
func (definition Definition) TypeURI() string {
	return TypeBaseURI + string(definition.Kind)
}

var definitions = map[Kind]Definition{
	KindInvalidRequest: {
		Kind:   KindInvalidRequest,
		Status: http.StatusBadRequest,
		Title:  "请求参数错误",
	},
	KindUnauthorized: {
		Kind:   KindUnauthorized,
		Status: http.StatusUnauthorized,
		Title:  "认证失败",
	},
	KindForbidden: {
		Kind:   KindForbidden,
		Status: http.StatusForbidden,
		Title:  "权限不足",
	},
	KindNotFound: {
		Kind:   KindNotFound,
		Status: http.StatusNotFound,
		Title:  "资源不存在",
	},
	KindConflict: {
		Kind:   KindConflict,
		Status: http.StatusConflict,
		Title:  "资源状态冲突",
	},
	KindValidation: {
		Kind:   KindValidation,
		Status: http.StatusUnprocessableEntity,
		Title:  "请求校验失败",
	},
	KindClusterCredentialInvalid: {
		Kind:   KindClusterCredentialInvalid,
		Status: http.StatusFailedDependency,
		Title:  "集群凭据无效或已过期",
	},
	KindIncidentNotFound: {
		Kind:   KindIncidentNotFound,
		Status: http.StatusNotFound,
		Title:  "事件不存在",
	},
	KindVersionConflict: {
		Kind:   KindVersionConflict,
		Status: http.StatusConflict,
		Title:  "事件已被其他用户更新",
	},
	KindInvalidIncidentTransition: {
		Kind:   KindInvalidIncidentTransition,
		Status: http.StatusConflict,
		Title:  "当前状态不允许此操作",
	},
	KindDomainValidation: {
		Kind:   KindDomainValidation,
		Status: http.StatusUnprocessableEntity,
		Title:  "事件处置校验失败",
	},
	KindInternal: {
		Kind:   KindInternal,
		Status: http.StatusInternalServerError,
		Title:  "内部错误",
	},
}

// DefinitionFor resolves a catalogued problem type. The bool is false for an
// unknown kind so callers that need to inspect the catalog can reject it.
func DefinitionFor(kind Kind) (Definition, bool) {
	definition, ok := definitions[kind]
	return definition, ok
}

type Details struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail,omitempty"`
	Instance  string `json:"instance,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteKind writes one of the standardized problem definitions. Unknown kinds
// are deliberately rendered as an internal problem to avoid publishing an
// unreviewed API error contract.
func WriteKind(c *gin.Context, kind Kind, detail string) {
	definition, ok := DefinitionFor(kind)
	if !ok {
		definition, _ = DefinitionFor(KindInternal)
	}
	Write(c, definition.Status, definition.TypeURI(), definition.Title, detail)
}

// Write is the low-level escape hatch for an already-established bespoke
// problem. New v2 handlers should prefer WriteKind so status, URI and title do
// not drift between endpoints.
func Write(c *gin.Context, status int, problemType, title, detail string) {
	rid := requestID(c)
	c.Set("resp_code", status)
	c.Set("resp_message", title)
	c.Header("Content-Type", "application/problem+json")
	if rid != "" {
		c.Header(RequestIDHeader, rid)
	}
	c.JSON(status, Details{
		Type:      problemType,
		Title:     title,
		Status:    status,
		Detail:    detail,
		Instance:  requestInstance(c),
		RequestID: rid,
	})
}

func Unauthorized(c *gin.Context, detail string) {
	WriteKind(c, KindUnauthorized, detail)
}

func Forbidden(c *gin.Context, detail string) {
	WriteKind(c, KindForbidden, detail)
}

func requestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if requestID := strings.TrimSpace(c.GetString("request_id")); requestID != "" {
		return requestID
	}
	return strings.TrimSpace(c.GetHeader(RequestIDHeader))
}

func requestInstance(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return ""
	}
	return c.Request.URL.Path
}
