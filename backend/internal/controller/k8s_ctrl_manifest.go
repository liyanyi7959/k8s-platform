package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type applyManifestReq struct {
	YAML             string `json:"yaml" binding:"required"`
	DefaultNamespace string `json:"default_namespace"`
	DryRun           bool   `json:"dry_run"`
	SourceLabel      string `json:"source_label"`
	SourceResource   string `json:"source_resource"`
	WorkloadKind     string `json:"workload_kind"`
}

type applyManifestResp struct {
	RecordID uint64                            `json:"record_id"`
	Status   string                            `json:"status"`
	DryRun   bool                              `json:"dry_run"`
	Summary  string                            `json:"summary"`
	Items    []service.ManifestApplyResultItem `json:"items"`
}

// ApplyManifest 统一应用多文档 YAML 清单。
// @Summary 应用 Manifest YAML
// @Description 统一应用多文档 YAML，支持按资源 create/update
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param body body applyManifestReq true "Manifest 内容"
// @Success 200 {object} resp.Result{data=applyManifestResp} "操作成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/manifests/apply [post]
func (kc *K8sController) ApplyManifest(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req applyManifestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	req.YAML = strings.TrimSpace(req.YAML)
	req.DefaultNamespace = strings.TrimSpace(req.DefaultNamespace)
	req.SourceLabel = strings.TrimSpace(req.SourceLabel)
	req.SourceResource = strings.TrimSpace(req.SourceResource)
	req.WorkloadKind = strings.TrimSpace(req.WorkloadKind)
	if req.YAML == "" {
		resp.Fail(c, 4000, "yaml is required")
		return
	}
	claims, _ := middleware.GetClaims(c)
	userID := uint64(0)
	username := ""
	if claims != nil {
		if claims.UserID > 0 {
			userID = uint64(claims.UserID)
		}
		username = strings.TrimSpace(claims.Username)
	}
	result, err := kc.manifestSvc.Execute(c.Request.Context(), service.ManifestApplyExecuteRequest{
		ClusterID:        id,
		YAML:             req.YAML,
		DefaultNamespace: req.DefaultNamespace,
		DryRun:           req.DryRun,
		SourceLabel:      req.SourceLabel,
		SourceResource:   req.SourceResource,
		WorkloadKind:     req.WorkloadKind,
		CreatedBy:        userID,
		CreatedByName:    username,
	})
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, applyManifestResp{RecordID: result.RecordID, Status: result.Status, DryRun: req.DryRun, Summary: result.Summary, Items: result.Items})
}

// ListManifestRecords 查询 Manifest 部署记录。
func (kc *K8sController) ListManifestRecords(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	result, err := kc.manifestSvc.List(c.Request.Context(), service.ManifestApplyRecordListParams{
		ClusterID:        id,
		Page:             parseInt(c.Query("page"), 1),
		PageSize:         parseInt(c.Query("page_size"), 20),
		Keyword:          strings.TrimSpace(c.Query("keyword")),
		Status:           strings.TrimSpace(c.Query("status")),
		Mode:             strings.TrimSpace(c.Query("mode")),
		DefaultNamespace: strings.TrimSpace(c.Query("default_namespace")),
	})
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, result)
}

// GetManifestRecord 查询单条 Manifest 部署记录详情。
func (kc *K8sController) GetManifestRecord(c *gin.Context) {
	clusterID, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	recordID, err := strconv.ParseUint(strings.TrimSpace(c.Param("recordId")), 10, 64)
	if err != nil || recordID == 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	result, svcErr := kc.manifestSvc.Get(c.Request.Context(), clusterID, recordID)
	if svcErr != nil {
		kc.writeServiceErr(c, svcErr)
		return
	}
	resp.OK(c, result)
}
