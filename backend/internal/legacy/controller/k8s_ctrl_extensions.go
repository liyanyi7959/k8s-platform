package controller

import (
	"github.com/gin-gonic/gin"

	"k8s-platform-backend/pkg/resp"
)

func (kc *K8sController) ListCustomResourceDefinitions(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrCustomResourceDefinitions(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func (kc *K8sController) GetCustomResourceDefinitionYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrCustomResourceDefinitions(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func (kc *K8sController) EditCustomResourceDefinition(c *gin.Context) {
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
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvrCustomResourceDefinitions(), "", req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) DeleteCustomResourceDefinition(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrCustomResourceDefinitions(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) ListAPIServices(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrAPIServices(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func (kc *K8sController) GetAPIServiceYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrAPIServices(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func (kc *K8sController) EditAPIService(c *gin.Context) {
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
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvrAPIServices(), "", req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) DeleteAPIService(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrAPIServices(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) ListPriorityClasses(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrPriorityClasses(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func (kc *K8sController) GetPriorityClassYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrPriorityClasses(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func (kc *K8sController) EditPriorityClass(c *gin.Context) {
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
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvrPriorityClasses(), "", req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) DeletePriorityClass(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrPriorityClasses(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) ListValidatingWebhookConfigurations(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrValidatingWebhookConfigurations(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func (kc *K8sController) GetValidatingWebhookConfigurationYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrValidatingWebhookConfigurations(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func (kc *K8sController) EditValidatingWebhookConfiguration(c *gin.Context) {
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
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvrValidatingWebhookConfigurations(), "", req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) DeleteValidatingWebhookConfiguration(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrValidatingWebhookConfigurations(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) ListMutatingWebhookConfigurations(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrMutatingWebhookConfigurations(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func (kc *K8sController) GetMutatingWebhookConfigurationYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrMutatingWebhookConfigurations(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func (kc *K8sController) EditMutatingWebhookConfiguration(c *gin.Context) {
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
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvrMutatingWebhookConfigurations(), "", req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) DeleteMutatingWebhookConfiguration(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrMutatingWebhookConfigurations(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}
