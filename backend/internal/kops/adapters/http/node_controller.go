package http

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type NodeController struct{ service *kopsapp.NodeService }

func NewNodeController(service *kopsapp.NodeService) *NodeController {
	return &NodeController{service: service}
}

func (ctl *NodeController) List(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.List(c.Request.Context(), kopsapp.NodeListQuery{ClusterID: nodeClusterID(c), SortBy: c.Query("sort_by"), Order: c.Query("order")})
	ctl.respond(c, value, err)
}

func (ctl *NodeController) Detail(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.Detail(c.Request.Context(), nodeReference(c))
	ctl.respond(c, value, err)
}

func (ctl *NodeController) YAML(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.YAML(c.Request.Context(), nodeReference(c))
	ctl.respond(c, value, err)
}

func (ctl *NodeController) Pods(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.Pods(c.Request.Context(), nodeReference(c), c.Query("sort_by"), c.Query("order"))
	ctl.respond(c, value, err)
}

func (ctl *NodeController) Events(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.Events(c.Request.Context(), nodeReference(c))
	ctl.respond(c, value, err)
}

func (ctl *NodeController) Cordon(c *gin.Context) { ctl.setSchedulable(c, true) }

func (ctl *NodeController) Uncordon(c *gin.Context) { ctl.setSchedulable(c, false) }

func (ctl *NodeController) Drain(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	timeoutSeconds, err := nodeTimeout(c)
	if err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	err = ctl.service.Drain(c.Request.Context(), kopsapp.NodeDrainInput{
		NodeReference:    nodeReference(c),
		TimeoutSeconds:   timeoutSeconds,
		Force:            nodeBool(c.Query("force"), false),
		IgnoreDaemonSets: nodeBool(c.Query("ignore_daemonsets"), true),
	})
	ctl.respondEmpty(c, err)
}

func (ctl *NodeController) Delete(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	ctl.respondEmpty(c, ctl.service.Delete(c.Request.Context(), nodeReference(c)))
}

func (ctl *NodeController) setSchedulable(c *gin.Context, unschedulable bool) {
	if !ctl.ready(c) {
		return
	}
	ctl.respondEmpty(c, ctl.service.SetSchedulable(c.Request.Context(), nodeReference(c), unschedulable))
}

func (ctl *NodeController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}

func (ctl *NodeController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}

func (ctl *NodeController) respondEmpty(c *gin.Context, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func nodeClusterID(c *gin.Context) uint64 {
	value, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func nodeReference(c *gin.Context) kopsapp.NodeReference {
	name := c.Param("name")
	if decoded, err := url.PathUnescape(name); err == nil {
		name = decoded
	}
	return kopsapp.NodeReference{ClusterID: nodeClusterID(c), Name: strings.TrimSpace(name)}
}

func nodeTimeout(c *gin.Context) (int, error) {
	value := strings.TrimSpace(c.Query("timeout_seconds"))
	if value == "" {
		return 0, nil
	}
	timeout, err := strconv.Atoi(value)
	if err != nil || timeout <= 0 {
		return 0, kopsapp.ErrInvalidParams
	}
	return timeout, nil
}

func nodeBool(value string, fallback bool) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}
