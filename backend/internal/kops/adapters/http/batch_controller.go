package http

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type BatchController struct{ service *kopsapp.BatchService }

func NewBatchController(service *kopsapp.BatchService) *BatchController {
	return &BatchController{service: service}
}

func (ctl *BatchController) List(resource kopsapp.BatchResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.List(c.Request.Context(), resource, kopsapp.BatchListQuery{ClusterID: namespaceClusterID(c), Namespace: c.Query("namespace"), SortBy: c.Query("sort_by"), Order: c.Query("order")})
		ctl.respond(c, value, err)
	}
}
func (ctl *BatchController) YAML(resource kopsapp.BatchResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.YAML(c.Request.Context(), resource, batchReference(c))
		ctl.respond(c, value, err)
	}
}
func (ctl *BatchController) Delete(resource kopsapp.BatchResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		if err := ctl.service.Delete(c.Request.Context(), resource, batchReference(c)); err != nil {
			writeKopsApplicationError(c, err)
			return
		}
		resp.OK(c, gin.H{})
	}
}
func (ctl *BatchController) EditJob(c *gin.Context) {
	var input kopsapp.JobEditInput
	if !ctl.bind(c, &input) {
		return
	}
	input.ClusterID = namespaceClusterID(c)
	ctl.respondOK(c, ctl.service.EditJob(c.Request.Context(), input))
}
func (ctl *BatchController) DeleteCompletedJobs(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	older, err := batchOlderThanHours(c)
	if err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	count, err := ctl.service.DeleteCompletedJobs(c.Request.Context(), namespaceClusterID(c), c.Query("namespace"), older)
	ctl.respond(c, gin.H{"deleted_count": count}, err)
}
func (ctl *BatchController) EditCronJob(c *gin.Context) {
	var input kopsapp.CronJobEditInput
	if !ctl.bind(c, &input) {
		return
	}
	input.ClusterID = namespaceClusterID(c)
	ctl.respondOK(c, ctl.service.EditCronJob(c.Request.Context(), input))
}
func (ctl *BatchController) TriggerCronJob(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.TriggerCronJob(c.Request.Context(), batchReference(c))
	ctl.respond(c, result, err)
}
func (ctl *BatchController) SuspendCronJob(c *gin.Context) {
	var request struct {
		Suspend *bool `json:"suspend"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.Suspend == nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if !ctl.ready(c) {
		return
	}
	err := ctl.service.SuspendCronJob(c.Request.Context(), batchReference(c), *request.Suspend)
	ctl.respond(c, gin.H{"suspend": *request.Suspend}, err)
}
func (ctl *BatchController) bind(c *gin.Context, input any) bool {
	if err := c.ShouldBindJSON(input); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return false
	}
	return ctl.ready(c)
}
func (ctl *BatchController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}
func (ctl *BatchController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}
func (ctl *BatchController) respondOK(c *gin.Context, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}
func batchReference(c *gin.Context) kopsapp.BatchReference {
	decode := func(value string) string {
		if decoded, err := url.PathUnescape(value); err == nil {
			value = decoded
		}
		return strings.TrimSpace(value)
	}
	return kopsapp.BatchReference{ClusterID: namespaceClusterID(c), Namespace: decode(c.Param("ns")), Name: decode(c.Param("name"))}
}
func batchOlderThanHours(c *gin.Context) (int, error) {
	value := strings.TrimSpace(c.Query("older_than_hours"))
	if value == "" {
		return 24, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, kopsapp.ErrInvalidParams
	}
	return parsed, nil
}
