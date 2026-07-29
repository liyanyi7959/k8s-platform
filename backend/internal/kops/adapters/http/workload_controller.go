package http

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type WorkloadController struct{ service *kopsapp.WorkloadService }

func NewWorkloadController(service *kopsapp.WorkloadService) *WorkloadController {
	return &WorkloadController{service: service}
}

func (ctl *WorkloadController) List(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.List(c.Request.Context(), kopsapp.WorkloadQuery{
		ClusterID:     namespaceClusterID(c),
		Kind:          kopsapp.WorkloadKind(c.Query("kind")),
		Namespace:     c.Query("namespace"),
		LabelSelector: c.Query("label_selector"),
		SortBy:        c.Query("sort_by"),
		Order:         c.Query("order"),
	})
	ctl.respond(c, value, err)
}

func (ctl *WorkloadController) History(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.History(c.Request.Context(), workloadReference(c, kopsapp.WorkloadDeployment))
	ctl.respond(c, value, err)
}

func (ctl *WorkloadController) Undo(c *gin.Context) {
	var request struct {
		Revision int `json:"revision"`
	}
	if !ctl.bind(c, &request) {
		return
	}
	ctl.respondOK(c, ctl.service.Undo(c.Request.Context(), workloadReference(c, kopsapp.WorkloadDeployment), request.Revision))
}

func (ctl *WorkloadController) Scale(c *gin.Context) {
	var request struct {
		Kind      kopsapp.WorkloadKind `json:"kind"`
		Namespace string                `json:"namespace"`
		Name      string                `json:"name"`
		Replicas  int                   `json:"replicas"`
	}
	if !ctl.bind(c, &request) {
		return
	}
	ctl.respondOK(c, ctl.service.Scale(c.Request.Context(), kopsapp.WorkloadScale{
		WorkloadRef: kopsapp.WorkloadRef{ClusterID: namespaceClusterID(c), Kind: request.Kind, Namespace: request.Namespace, Name: request.Name},
		Replicas:    request.Replicas,
	}))
}

func (ctl *WorkloadController) Restart(c *gin.Context) {
	var request struct {
		Kind      kopsapp.WorkloadKind `json:"kind"`
		Namespace string                `json:"namespace"`
		Name      string                `json:"name"`
	}
	if !ctl.bind(c, &request) {
		return
	}
	ctl.respondOK(c, ctl.service.Restart(c.Request.Context(), kopsapp.WorkloadRef{ClusterID: namespaceClusterID(c), Kind: request.Kind, Namespace: request.Namespace, Name: request.Name}))
}

func (ctl *WorkloadController) Image(c *gin.Context) {
	var input kopsapp.WorkloadImage
	if !ctl.bind(c, &input) {
		return
	}
	input.ClusterID = namespaceClusterID(c)
	ctl.respondOK(c, ctl.service.Image(c.Request.Context(), input))
}

func (ctl *WorkloadController) Pause(c *gin.Context) {
	var input kopsapp.WorkloadPause
	if !ctl.bind(c, &input) {
		return
	}
	input.ClusterID = namespaceClusterID(c)
	ctl.respondOK(c, ctl.service.Pause(c.Request.Context(), input))
}

func (ctl *WorkloadController) Edit(kind kopsapp.WorkloadKind) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input kopsapp.WorkloadEditInput
		if !ctl.bind(c, &input) {
			return
		}
		input.ClusterID = namespaceClusterID(c)
		input.Kind = kind
		ctl.respondOK(c, ctl.service.Edit(c.Request.Context(), input))
	}
}

func (ctl *WorkloadController) ApplyYAML(c *gin.Context) {
	var input kopsapp.WorkloadYAMLEdit
	if !ctl.bind(c, &input) {
		return
	}
	input.ClusterID = namespaceClusterID(c)
	ctl.respondOK(c, ctl.service.ApplyYAML(c.Request.Context(), input))
}

func (ctl *WorkloadController) Delete(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	if err := ctl.service.Delete(c.Request.Context(), workloadReference(c, kopsapp.WorkloadKind(pathValue(c, "kind")))); err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

func (ctl *WorkloadController) YAML(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	text, err := ctl.service.YAML(c.Request.Context(), workloadReference(c, kopsapp.WorkloadKind(pathValue(c, "kind"))))
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func (ctl *WorkloadController) bind(c *gin.Context, value any) bool {
	if err := c.ShouldBindJSON(value); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return false
	}
	return ctl.ready(c)
}

func (ctl *WorkloadController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}

func (ctl *WorkloadController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}

func (ctl *WorkloadController) respondOK(c *gin.Context, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

func workloadReference(c *gin.Context, kind kopsapp.WorkloadKind) kopsapp.WorkloadRef {
	return kopsapp.WorkloadRef{
		ClusterID: namespaceClusterID(c),
		Kind:      kind,
		Namespace: pathValue(c, "ns"),
		Name:      pathValue(c, "name"),
	}
}

func pathValue(c *gin.Context, key string) string {
	value := c.Param(key)
	if decoded, err := url.PathUnescape(value); err == nil {
		value = decoded
	}
	return strings.TrimSpace(value)
}
