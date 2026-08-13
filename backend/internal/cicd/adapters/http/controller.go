package http

import (
	"errors"
	"github.com/gin-gonic/gin"
	"k8s-platform-backend/internal/cicd/application"
	"k8s-platform-backend/pkg/resp"
	"net/http"
	"strconv"
)

type Controller struct{ service *application.Service }

func NewController(service *application.Service) *Controller { return &Controller{service: service} }
func (ctl *Controller) Summary(c *gin.Context) {
	data, err := ctl.service.Summary(c.Request.Context())
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, data)
}
func (ctl *Controller) ListPipelines(c *gin.Context) {
	data, err := ctl.service.ListPipelines(c.Request.Context(), atoi(c.Query("page"), 1), atoi(c.Query("page_size"), 20), c.Query("keyword"), c.Query("status"), c.Query("trigger_type"))
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, data)
}
func (ctl *Controller) CreatePipeline(c *gin.Context) {
	var in application.PipelineInput
	if c.ShouldBindJSON(&in) != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	p, err := ctl.service.CreatePipeline(c.Request.Context(), in, 0)
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, p)
}
func (ctl *Controller) GetPipeline(c *gin.Context) {
	p, err := ctl.service.GetPipeline(c.Request.Context(), id(c))
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, p)
}
func (ctl *Controller) UpdatePipeline(c *gin.Context) {
	var in application.PipelineInput
	if c.ShouldBindJSON(&in) != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := ctl.service.UpdatePipeline(c.Request.Context(), id(c), in); err != nil {
		fail(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *Controller) DeletePipeline(c *gin.Context) {
	if err := ctl.service.DeletePipeline(c.Request.Context(), id(c)); err != nil {
		fail(c, err)
		return
	}
	resp.OK[any](c, nil)
}

type runInput struct {
	TriggerType   string `json:"trigger_type"`
	Branch        string `json:"branch"`
	CommitSHA     string `json:"commit_sha"`
	CommitMessage string `json:"commit_message"`
}

func (ctl *Controller) CreateRun(c *gin.Context) {
	var in runInput
	_ = c.ShouldBindJSON(&in)
	if in.TriggerType == "" {
		in.TriggerType = "manual"
	}
	r, err := ctl.service.CreateRun(c.Request.Context(), id(c), in.TriggerType, in.Branch, in.CommitSHA, in.CommitMessage)
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, r)
}
func (ctl *Controller) ListRuns(c *gin.Context) {
	pid := uint64(atoi(c.Query("pipeline_id"), 0))
	data, err := ctl.service.ListRuns(c.Request.Context(), atoi(c.Query("page"), 1), atoi(c.Query("page_size"), 20), pid, c.Query("status"), c.Query("keyword"))
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, data)
}
func (ctl *Controller) GetRun(c *gin.Context) {
	r, err := ctl.service.GetRun(c.Request.Context(), id(c))
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, r)
}
func (ctl *Controller) CancelRun(c *gin.Context) {
	if err := ctl.service.CancelRun(c.Request.Context(), id(c)); err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, gin.H{"status": "canceled"})
}
func (ctl *Controller) ListArtifacts(c *gin.Context) {
	data, err := ctl.service.ListArtifacts(c.Request.Context(), atoi(c.Query("page"), 1), atoi(c.Query("page_size"), 20), c.Query("type"), c.Query("keyword"))
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, data)
}
func (ctl *Controller) GetArtifact(c *gin.Context) {
	a, err := ctl.service.GetArtifact(c.Request.Context(), id(c))
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, a)
}
func (ctl *Controller) ListEnvironments(c *gin.Context) {
	rows, err := ctl.service.ListEnvironments(c.Request.Context())
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, gin.H{"list": rows, "total": len(rows)})
}
func (ctl *Controller) GetEnvironment(c *gin.Context) {
	e, err := ctl.service.GetEnvironment(c.Request.Context(), id(c))
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, e)
}
func (ctl *Controller) CreateEnvironment(c *gin.Context) {
	var in application.EnvironmentInput
	if c.ShouldBindJSON(&in) != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	e, err := ctl.service.CreateEnvironment(c.Request.Context(), in)
	if err != nil {
		fail(c, err)
		return
	}
	resp.OK(c, e)
}
func (ctl *Controller) UpdateEnvironment(c *gin.Context) {
	var in application.EnvironmentInput
	if c.ShouldBindJSON(&in) != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := ctl.service.UpdateEnvironment(c.Request.Context(), id(c), in); err != nil {
		fail(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *Controller) DeleteEnvironment(c *gin.Context) {
	if err := ctl.service.DeleteEnvironment(c.Request.Context(), id(c)); err != nil {
		fail(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func id(c *gin.Context) uint64 {
	v, e := strconv.ParseUint(c.Param("id"), 10, 64)
	if e != nil || v == 0 {
		resp.Fail(c, 4000, "invalid params")
		return 0
	}
	return v
}
func atoi(v string, fallback int) int {
	n, e := strconv.Atoi(v)
	if e != nil {
		return fallback
	}
	return n
}
func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, application.ErrNotFound):
		resp.Fail(c, http.StatusNotFound, "not found")
	case errors.Is(err, application.ErrConflict):
		resp.Fail(c, http.StatusConflict, "conflict")
	default:
		resp.Fail(c, 5000, err.Error())
	}
}
