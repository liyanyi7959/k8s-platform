package http

import (
	"github.com/gin-gonic/gin"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	"k8s-platform-backend/pkg/resp"
)

// ServerController adapts provisioning-host management to the existing deploy
// API. SSH probing and interactive terminal endpoints remain runtime routes.
type ServerController struct {
	svc *provisionapp.ServerService
}

func NewServerController(svc *provisionapp.ServerService) *ServerController {
	return &ServerController{svc: svc}
}

func (sc *ServerController) List(c *gin.Context) {
	data, err := sc.svc.List(c.Request.Context(), provisionapp.ListDeployServersRequest{
		Page: parseInt(c.Query("page"), 1), PageSize: parseInt(c.Query("page_size"), 20),
		Keyword: c.Query("keyword"), Status: c.Query("status"),
	})
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, data)
}

func (sc *ServerController) Summary(c *gin.Context) {
	data, err := sc.svc.Summary(c.Request.Context())
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, data)
}

func (sc *ServerController) Create(c *gin.Context) {
	var req provisionapp.CreateDeployServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := sc.svc.Create(c.Request.Context(), req)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (sc *ServerController) Get(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	data, err := sc.svc.Get(c.Request.Context(), id)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, data)
}

func (sc *ServerController) Update(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	var req provisionapp.UpdateDeployServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := sc.svc.Update(c.Request.Context(), id, req); err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (sc *ServerController) Delete(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	if err := sc.svc.Delete(c.Request.Context(), id); err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
