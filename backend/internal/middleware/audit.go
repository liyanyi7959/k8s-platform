package middleware

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
)

// AuditLogger 写操作审计中间件。
// 仅对 POST/PUT/PATCH/DELETE 请求生效，自动从路由信息中提取资源和动作。
func AuditLogger(auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		path := c.Request.URL.Path
		if !shouldAuditRequest(method, path) {
			c.Next()
			return
		}

		c.Next()

		// 请求处理完毕后记录审计日志
		status := c.Writer.Status()
		if v, ok := c.Get("resp_code"); ok {
			switch code := v.(type) {
			case int:
				status = code
			case int32:
				status = int(code)
			case int64:
				status = int(code)
			case uint:
				status = int(code)
			case uint32:
				status = int(code)
			case uint64:
				status = int(code)
			}
		}
		detail := ""
		if v, ok := c.Get("resp_message"); ok {
			if s, ok := v.(string); ok {
				detail = s
			}
		}

		userID := uint64(0)
		username := ""
		if v, ok := c.Get("user_id"); ok {
			switch id := v.(type) {
			case int64:
				userID = uint64(id)
			case uint64:
				userID = id
			case float64:
				userID = uint64(id)
			}
		}
		if v, ok := c.Get("username"); ok {
			if s, ok := v.(string); ok {
				username = s
			}
		}

		rid := ""
		if v, ok := c.Get("request_id"); ok {
			if s, ok := v.(string); ok {
				rid = s
			}
		}

		action := inferAction(method, path)
		resource, resourceName, clusterID, namespace := parsePath(path)

		entry := service.AuditEntry{
			UserID:       userID,
			Username:     username,
			Action:       action,
			Resource:     resource,
			ResourceName: resourceName,
			ClusterID:    clusterID,
			Namespace:    namespace,
			Path:         path,
			StatusCode:   status,
			Detail:       detail,
			ClientIP:     c.ClientIP(),
			RequestID:    rid,
		}

		// 审计日志属于后台补充记录，不应受请求取消影响。
		auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		go func() {
			defer cancel()
			auditSvc.Record(auditCtx, entry)
		}()
	}
}

func shouldAuditRequest(method, path string) bool {
	switch method {
	case "GET", "HEAD", "OPTIONS":
		return false
	case "POST":
		lowerPath := strings.ToLower(strings.TrimSpace(path))
		if strings.HasSuffix(lowerPath, "/logs/session") {
			// 日志会话只是查看日志前的只读握手，不应误记为“创建 Pod”。
			return false
		}
	}
	return true
}

func inferAction(method, path string) string {
	lower := strings.ToLower(path)
	switch method {
	case "DELETE":
		return "delete"
	case "POST":
		// 判断特殊操作
		if strings.Contains(lower, "/exec") {
			return "exec"
		}
		if strings.Contains(lower, "/drain") {
			return "drain"
		}
		if strings.Contains(lower, "/cordon") || strings.Contains(lower, "/uncordon") {
			return "cordon"
		}
		if strings.Contains(lower, "/scale") {
			return "scale"
		}
		if strings.Contains(lower, "/restart") {
			return "restart"
		}
		if strings.Contains(lower, "/rollout") {
			return "rollout"
		}
		if strings.Contains(lower, "/login") || strings.Contains(lower, "/logout") || strings.Contains(lower, "/change-password") || strings.Contains(lower, "/reset-password") {
			return "auth"
		}
		return "create"
	case "PUT", "PATCH":
		return "update"
	default:
		return method
	}
}

// clusterResourcePattern 匹配 /api/v1/clusters/:id/... 路径
var clusterResourcePattern = regexp.MustCompile(
	`/api/v1/clusters/(\d+)/([^/]+)(?:/([^/]+))?(?:/([^/]+))?(?:/([^/]+))?`,
)

// topLevelPattern 匹配 /api/v1/resource/... 路径
var topLevelPattern = regexp.MustCompile(
	`/api/v1/([^/]+)(?:/([^/]+))?`,
)

func parsePath(path string) (resource, resourceName string, clusterID uint64, namespace string) {
	if m := clusterResourcePattern.FindStringSubmatch(path); m != nil {
		clusterID, _ = strconv.ParseUint(m[1], 10, 64)
		resource = singularize(m[2])

		// /clusters/:id/:resource/:ns/:name 或 /clusters/:id/:resource/:name
		switch {
		case m[5] != "":
			// 5 段: /clusters/1/pods/ns/name/action
			namespace = m[3]
			resourceName = m[4]
		case m[4] != "":
			// 4 段: /clusters/1/pods/ns/name
			namespace = m[3]
			resourceName = m[4]
		case m[3] != "":
			// 3 段: /clusters/1/namespaces/name
			resourceName = m[3]
		}
		return
	}

	if m := topLevelPattern.FindStringSubmatch(path); m != nil {
		resource = singularize(m[1])
		if m[2] != "" {
			resourceName = m[2]
		}
		return
	}
	return
}

func singularize(s string) string {
	s = strings.TrimRight(s, "/")
	mapping := map[string]string{
		"clusters":     "cluster",
		"namespaces":   "namespace",
		"pods":         "pod",
		"deployments":  "deployment",
		"services":     "service",
		"ingresses":    "ingress",
		"configmaps":   "configmap",
		"secrets":      "secret",
		"nodes":        "node",
		"statefulsets": "statefulset",
		"daemonsets":   "daemonset",
		"jobs":         "job",
		"cronjobs":     "cronjob",
		"pvcs":         "pvc",
		"pvs":          "pv",
		"users":        "user",
		"roles":        "role",
	}
	if v, ok := mapping[s]; ok {
		return v
	}
	return s
}
