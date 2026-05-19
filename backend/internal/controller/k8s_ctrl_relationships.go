package controller

import (
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/pkg/resp"
)

func (kc *K8sController) ListReplicaSets(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, gvrReplicaSets(), ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func (kc *K8sController) GetReplicaSetYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrReplicaSets(), ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func (kc *K8sController) EditReplicaSet(c *gin.Context) {
	var req K8sEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvrReplicaSets(), strings.TrimSpace(req.Namespace), req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) DeleteReplicaSet(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrReplicaSets(), ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) ListVolumeAttachments(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrVolumeAttachments(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func (kc *K8sController) GetVolumeAttachmentYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrVolumeAttachments(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func (kc *K8sController) EditVolumeAttachment(c *gin.Context) {
	var req K8sEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvrVolumeAttachments(), "", req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) DeleteVolumeAttachment(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrVolumeAttachments(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}
