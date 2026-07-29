package http

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/audit/application"
	"k8s-platform-backend/internal/audit/ports"
	"k8s-platform-backend/pkg/resp"
)

type Controller struct{ service *application.Service }

func NewController(service *application.Service) *Controller { return &Controller{service: service} }

func (ctl *Controller) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	query := ports.ListQuery{
		Page: page, PageSize: pageSize, Keyword: strings.TrimSpace(c.Query("keyword")),
		Username: strings.TrimSpace(c.Query("username")), Action: strings.TrimSpace(c.Query("action")),
		Resource: strings.TrimSpace(c.Query("resource")), ClusterID: clusterID,
		Status: strings.TrimSpace(c.Query("status")),
	}
	query.StartTime = parseTime(c.Query("start_time"))
	query.EndTime = parseTime(c.Query("end_time"))
	result, err := ctl.service.List(c.Request.Context(), query)
	if err != nil {
		resp.Fail(c, 5000, "查询失败")
		return
	}
	resp.OK(c, result)
}

func parseTime(value string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return nil
	}
	return &parsed
}
