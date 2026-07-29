package http

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/pkg/resp"
)

// ManifestController owns the HTTP boundary for manifest applications and
// their audit records. Kubernetes execution is delegated to the Kops
// application service through its runtime port.
type ManifestController struct{ service *kopsapp.ManifestService }

func NewManifestController(service *kopsapp.ManifestService) *ManifestController {
	return &ManifestController{service: service}
}

type applyManifestRequest struct {
	YAML             string `json:"yaml" binding:"required"`
	DefaultNamespace string `json:"default_namespace"`
	DryRun           bool   `json:"dry_run"`
	CreateOnly       bool   `json:"create_only"`
	SourceLabel      string `json:"source_label"`
	SourceResource   string `json:"source_resource"`
	WorkloadKind     string `json:"workload_kind"`
}

func (ctl *ManifestController) Apply(c *gin.Context) {
	if ctl == nil || ctl.service == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	clusterID, ok := manifestPathID(c, "id")
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var request applyManifestRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	userID, username := manifestCurrentUser(c)
	result, err := ctl.service.Apply(c.Request.Context(), kopsapp.ManifestApplyInput{
		ClusterID: clusterID, YAML: request.YAML, DefaultNamespace: request.DefaultNamespace, DryRun: request.DryRun,
		CreateOnly: request.CreateOnly, SourceLabel: request.SourceLabel, SourceResource: request.SourceResource,
		WorkloadKind: request.WorkloadKind, CreatedBy: userID, CreatedByName: username,
	})
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *ManifestController) List(c *gin.Context) {
	if ctl == nil || ctl.service == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	clusterID, ok := manifestPathID(c, "id")
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	result, err := ctl.service.List(c.Request.Context(), kopsapp.ManifestRecordQuery{
		ClusterID: clusterID, Page: manifestQueryInt(c, "page", 1), PageSize: manifestQueryInt(c, "page_size", 20),
		Keyword: c.Query("keyword"), Status: c.Query("status"), Mode: c.Query("mode"), DefaultNamespace: c.Query("default_namespace"),
	})
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func (ctl *ManifestController) Get(c *gin.Context) {
	if ctl == nil || ctl.service == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	clusterID, ok := manifestPathID(c, "id")
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	recordID, ok := manifestPathID(c, "recordId")
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	result, err := ctl.service.Get(c.Request.Context(), clusterID, recordID)
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, result)
}

func manifestPathID(c *gin.Context, name string) (uint64, bool) {
	value, err := strconv.ParseUint(strings.TrimSpace(c.Param(name)), 10, 64)
	return value, err == nil && value > 0
}

func manifestQueryInt(c *gin.Context, name string, fallback int) int {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func manifestCurrentUser(c *gin.Context) (uint64, string) {
	claims, _ := middleware.GetClaims(c)
	if claims == nil {
		return 0, ""
	}
	if claims.UserID <= 0 {
		return 0, strings.TrimSpace(claims.Username)
	}
	return uint64(claims.UserID), strings.TrimSpace(claims.Username)
}

func writeKopsApplicationError(c *gin.Context, err error) {
	if resp.HandleK8sError(c, err) {
		return
	}
	switch {
	case errors.Is(err, kopsapp.ErrInvalidParams):
		resp.Fail(c, 4000, kopsErrorMessage(err, "invalid params"))
	case errors.Is(err, kopsapp.ErrNotFound):
		resp.Fail(c, 4040, kopsErrorMessage(err, "not found"))
	case errors.Is(err, kopsapp.ErrConflict):
		resp.Fail(c, 4090, kopsErrorMessage(err, "conflict"))
	case errors.Is(err, kopsapp.ErrRuntimeUnauthorized):
		resp.Fail(c, 1002, kopsErrorMessage(err, "credentials invalid or expired"))
	case errors.Is(err, kopsapp.ErrRuntimeForbidden):
		resp.Fail(c, 1003, kopsErrorMessage(err, "permission denied"))
	case errors.Is(err, kopsapp.ErrRuntimeNetwork):
		resp.Fail(c, 5000, kopsErrorMessage(err, "network connection failed"))
	case errors.Is(err, kopsapp.ErrRuntimeTimeout):
		resp.Fail(c, 5000, kopsErrorMessage(err, "connection timed out"))
	case errors.Is(err, kopsapp.ErrRuntimeTLS):
		resp.Fail(c, 5000, kopsErrorMessage(err, "certificate verification failed"))
	case errors.Is(err, kopsapp.ErrRuntime):
		resp.Fail(c, 5000, kopsErrorMessage(err, "cluster access failed"))
	default:
		resp.Fail(c, 5000, kopsErrorMessage(err, "internal error"))
	}
}

func kopsErrorMessage(err error, fallback string) string {
	var withMessage interface{ UserMessage() string }
	if errors.As(err, &withMessage) {
		if message := strings.TrimSpace(withMessage.UserMessage()); message != "" {
			return message
		}
	}
	return fallback
}
