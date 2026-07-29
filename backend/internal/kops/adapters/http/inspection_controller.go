package http

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type InspectionController struct{ service *kopsapp.InspectionService }

func NewInspectionController(service *kopsapp.InspectionService) *InspectionController {
	return &InspectionController{service: service}
}

func (ctl *InspectionController) Namespace(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.Namespace(c.Request.Context(), namespaceClusterID(c), inspectionPath(c, "ns"))
	ctl.respond(c, value, err)
}

func (ctl *InspectionController) NamespaceWorkloadInventory(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.NamespaceWorkloadInventory(c.Request.Context(), namespaceClusterID(c), inspectionPath(c, "ns"))
	ctl.respond(c, value, err)
}

func (ctl *InspectionController) Pod(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.Pod(c.Request.Context(), namespaceClusterID(c), inspectionPath(c, "ns"), inspectionPath(c, "pod"))
	ctl.respond(c, value, err)
}

func (ctl *InspectionController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}

func (ctl *InspectionController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}

func inspectionPath(c *gin.Context, key string) string {
	value := c.Param(key)
	if decoded, err := url.PathUnescape(value); err == nil {
		value = decoded
	}
	return strings.TrimSpace(value)
}
