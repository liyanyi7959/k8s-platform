package http

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/pkg/resp"
)

type ConfigurationController struct{ service *kopsapp.ConfigurationService }

func NewConfigurationController(service *kopsapp.ConfigurationService) *ConfigurationController {
	return &ConfigurationController{service: service}
}

func (ctl *ConfigurationController) List(resource kopsapp.ConfigurationResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.List(c.Request.Context(), resource, kopsapp.ConfigurationListQuery{ClusterID: namespaceClusterID(c), Namespace: c.Query("namespace"), SortBy: c.Query("sort_by"), Order: c.Query("order")})
		ctl.respond(c, value, err)
	}
}

func (ctl *ConfigurationController) YAML(resource kopsapp.ConfigurationResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.YAML(c.Request.Context(), resource, configurationReference(c))
		ctl.respond(c, value, err)
	}
}

func (ctl *ConfigurationController) Delete(resource kopsapp.ConfigurationResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		if err := ctl.service.Delete(c.Request.Context(), resource, configurationReference(c)); err != nil {
			writeKopsApplicationError(c, err)
			return
		}
		resp.OK(c, gin.H{})
	}
}

func (ctl *ConfigurationController) EditConfigMap(c *gin.Context) {
	var input kopsapp.ConfigMapEditInput
	if !ctl.bind(c, &input) {
		return
	}
	input.ClusterID = namespaceClusterID(c)
	ctl.respondOK(c, ctl.service.EditConfigMap(c.Request.Context(), input))
}

func (ctl *ConfigurationController) EditSecret(c *gin.Context) {
	var input kopsapp.SecretEditInput
	if !ctl.bind(c, &input) {
		return
	}
	input.ClusterID = namespaceClusterID(c)
	ctl.respondOK(c, ctl.service.EditSecret(c.Request.Context(), input))
}

func (ctl *ConfigurationController) RevealSecret(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	ref := configurationReference(c)
	text, err := ctl.service.RevealSecret(c.Request.Context(), ref)
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	var userID uint64
	var username string
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		if claims.UserID > 0 {
			userID = uint64(claims.UserID)
		}
		username = strings.TrimSpace(claims.Username)
	}
	zap.L().Warn("k8s_secret_reveal", zap.Uint64("user_id", userID), zap.String("username", username), zap.Uint64("cluster_id", ref.ClusterID), zap.String("secret", ref.Namespace+"/"+ref.Name), zap.String("request_id", c.GetString("request_id")))
	resp.OK(c, gin.H{"text": text})
}

func (ctl *ConfigurationController) Related(resource kopsapp.ConfigurationResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.Related(c.Request.Context(), resource, configurationReference(c))
		ctl.respond(c, value, err)
	}
}

func (ctl *ConfigurationController) bind(c *gin.Context, value any) bool {
	if err := c.ShouldBindJSON(value); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return false
	}
	return ctl.ready(c)
}

func (ctl *ConfigurationController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}

func (ctl *ConfigurationController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}

func (ctl *ConfigurationController) respondOK(c *gin.Context, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

func configurationReference(c *gin.Context) kopsapp.ConfigurationReference {
	decode := func(value string) string {
		if decoded, err := url.PathUnescape(value); err == nil {
			value = decoded
		}
		return strings.TrimSpace(value)
	}
	return kopsapp.ConfigurationReference{ClusterID: namespaceClusterID(c), Namespace: decode(c.Param("ns")), Name: decode(c.Param("name"))}
}
