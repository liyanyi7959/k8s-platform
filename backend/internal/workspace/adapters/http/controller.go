package http

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/workspace/application"
	"k8s-platform-backend/internal/workspace/domain"
	"k8s-platform-backend/pkg/resp"
)

type Controller struct{ service *application.Service }

func NewController(service *application.Service) *Controller { return &Controller{service: service} }

func (ctl *Controller) ListProjects(c *gin.Context) {
	data, err := ctl.service.List(c.Request.Context(), parseInt(c.Query("page"), 1), parseInt(c.Query("page_size"), 20))
	if err != nil {
		writeError(c, err)
		return
	}
	resp.OK(c, data)
}
func (ctl *Controller) GetProject(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	data, err := ctl.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	resp.OK(c, data)
}
func (ctl *Controller) CreateProject(c *gin.Context) {
	var request application.SaveRequest
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	creatorID := uint64(0)
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		creatorID = uint64(claims.UserID)
	}
	id, err := ctl.service.Create(c.Request.Context(), request, creatorID)
	if err != nil {
		writeError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}
func (ctl *Controller) UpdateProject(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var request application.SaveRequest
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.service.Update(c.Request.Context(), id, request); err != nil {
		writeError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *Controller) DeleteProject(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	if err := ctl.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *Controller) GetProjectResources(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	data, err := ctl.service.Resources(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	resp.OK(c, data)
}
func (ctl *Controller) AssignNamespaces(c *gin.Context) {
	id, ok := projectID(c)
	if !ok {
		return
	}
	var request struct {
		Namespaces []string `json:"namespaces"`
	}
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.service.AssignNamespaces(c.Request.Context(), id, request.Namespaces); err != nil {
		writeError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func projectID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}
func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		resp.Fail(c, 4000, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		resp.Fail(c, 4040, "未找到")
	case errors.Is(err, domain.ErrConflict):
		resp.Fail(c, 4090, "资源冲突")
	default:
		resp.Fail(c, 5000, "内部错误")
	}
}
