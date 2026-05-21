package controller

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type AuditController struct {
	svc *service.AuditService
}

func NewAuditController(svc *service.AuditService) *AuditController {
	return &AuditController{svc: svc}
}

// List 查询审计日志（分页 + 筛选）。
func (ac *AuditController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)

	params := service.AuditListParams{
		Page:      page,
		PageSize:  pageSize,
		Username:  strings.TrimSpace(c.Query("username")),
		Action:    strings.TrimSpace(c.Query("action")),
		Resource:  strings.TrimSpace(c.Query("resource")),
		ClusterID: clusterID,
	}

	if v := strings.TrimSpace(c.Query("start_time")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.StartTime = &t
		}
	}
	if v := strings.TrimSpace(c.Query("end_time")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			params.EndTime = &t
		}
	}

	result, err := ac.svc.List(c.Request.Context(), params)
	if err != nil {
		resp.Fail(c, 5000, "查询失败")
		return
	}
	resp.OK(c, result)
}
