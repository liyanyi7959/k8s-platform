package http

import (
	"github.com/gin-gonic/gin"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
	"strconv"
	"strings"
)

type ResourceCreatorController struct {
	service *kopsapp.ResourceCreatorService
}

func NewResourceCreatorController(service *kopsapp.ResourceCreatorService) *ResourceCreatorController {
	return &ResourceCreatorController{service: service}
}
func (ctl *ResourceCreatorController) CreateDeployment(c *gin.Context) {
	var v kopsapp.WorkloadCreateInput
	if !ctl.bind(c, &v) {
		return
	}
	ctl.respond(c, ctl.service.CreateDeployment(c.Request.Context(), creatorClusterID(c), v))
}
func (ctl *ResourceCreatorController) CreateStatefulSet(c *gin.Context) {
	var v kopsapp.WorkloadCreateInput
	if !ctl.bind(c, &v) {
		return
	}
	ctl.respond(c, ctl.service.CreateStatefulSet(c.Request.Context(), creatorClusterID(c), v))
}
func (ctl *ResourceCreatorController) CreateDaemonSet(c *gin.Context) {
	var v kopsapp.WorkloadCreateInput
	if !ctl.bind(c, &v) {
		return
	}
	ctl.respond(c, ctl.service.CreateDaemonSet(c.Request.Context(), creatorClusterID(c), v))
}
func (ctl *ResourceCreatorController) CreateService(c *gin.Context) {
	var v kopsapp.ServiceCreateInput
	if !ctl.bind(c, &v) {
		return
	}
	ctl.respond(c, ctl.service.CreateService(c.Request.Context(), creatorClusterID(c), v))
}
func (ctl *ResourceCreatorController) CreateIngress(c *gin.Context) {
	var v kopsapp.IngressCreateInput
	if !ctl.bind(c, &v) {
		return
	}
	ctl.respond(c, ctl.service.CreateIngress(c.Request.Context(), creatorClusterID(c), v))
}
func (ctl *ResourceCreatorController) bind(c *gin.Context, value any) bool {
	if err := c.ShouldBindJSON(value); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return false
	}
	if ctl == nil || ctl.service == nil {
		resp.Fail(c, 5000, "internal error")
		return false
	}
	return true
}
func (ctl *ResourceCreatorController) respond(c *gin.Context, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}
func creatorClusterID(c *gin.Context) uint64 {
	v, e := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if e != nil {
		return 0
	}
	return v
}
