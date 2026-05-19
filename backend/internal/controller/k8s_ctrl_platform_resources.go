package controller

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

func listClusterScopedResource(c *gin.Context, kc *K8sController, gvr schema.GroupVersionResource) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvr, "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			resp.OK(c, gin.H{"list": []any{}})
			return
		}
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func getClusterScopedResourceYAML(c *gin.Context, kc *K8sController, gvr schema.GroupVersionResource) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvr, "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func editClusterScopedResource(c *gin.Context, kc *K8sController, gvr schema.GroupVersionResource) {
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
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvr, "", req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func deleteClusterScopedResource(c *gin.Context, kc *K8sController, gvr schema.GroupVersionResource) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvr, "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func listNamespacedResource(c *gin.Context, kc *K8sController, gvr schema.GroupVersionResource) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, gvr, ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			resp.OK(c, gin.H{"list": []any{}})
			return
		}
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func getNamespacedResourceYAML(c *gin.Context, kc *K8sController, gvr schema.GroupVersionResource) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvr, ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func editNamespacedResource(c *gin.Context, kc *K8sController, gvr schema.GroupVersionResource) {
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
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvr, strings.TrimSpace(req.Namespace), req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func deleteNamespacedResource(c *gin.Context, kc *K8sController, gvr schema.GroupVersionResource) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvr, ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) ListRuntimeClasses(c *gin.Context) { listClusterScopedResource(c, kc, gvrRuntimeClasses()) }
func (kc *K8sController) GetRuntimeClassYAML(c *gin.Context) { getClusterScopedResourceYAML(c, kc, gvrRuntimeClasses()) }
func (kc *K8sController) EditRuntimeClass(c *gin.Context) { editClusterScopedResource(c, kc, gvrRuntimeClasses()) }
func (kc *K8sController) DeleteRuntimeClass(c *gin.Context) { deleteClusterScopedResource(c, kc, gvrRuntimeClasses()) }

func (kc *K8sController) ListCSIDrivers(c *gin.Context) { listClusterScopedResource(c, kc, gvrCSIDrivers()) }
func (kc *K8sController) GetCSIDriverYAML(c *gin.Context) { getClusterScopedResourceYAML(c, kc, gvrCSIDrivers()) }
func (kc *K8sController) EditCSIDriver(c *gin.Context) { editClusterScopedResource(c, kc, gvrCSIDrivers()) }
func (kc *K8sController) DeleteCSIDriver(c *gin.Context) { deleteClusterScopedResource(c, kc, gvrCSIDrivers()) }

func (kc *K8sController) ListCSINodes(c *gin.Context) { listClusterScopedResource(c, kc, gvrCSINodes()) }
func (kc *K8sController) GetCSINodeYAML(c *gin.Context) { getClusterScopedResourceYAML(c, kc, gvrCSINodes()) }
func (kc *K8sController) EditCSINode(c *gin.Context) { editClusterScopedResource(c, kc, gvrCSINodes()) }
func (kc *K8sController) DeleteCSINode(c *gin.Context) { deleteClusterScopedResource(c, kc, gvrCSINodes()) }

func (kc *K8sController) ListCSIStorageCapacities(c *gin.Context) { listNamespacedResource(c, kc, gvrCSIStorageCapacities()) }
func (kc *K8sController) GetCSIStorageCapacityYAML(c *gin.Context) { getNamespacedResourceYAML(c, kc, gvrCSIStorageCapacities()) }
func (kc *K8sController) EditCSIStorageCapacity(c *gin.Context) { editNamespacedResource(c, kc, gvrCSIStorageCapacities()) }
func (kc *K8sController) DeleteCSIStorageCapacity(c *gin.Context) { deleteNamespacedResource(c, kc, gvrCSIStorageCapacities()) }

func (kc *K8sController) ListValidatingAdmissionPolicies(c *gin.Context) { listClusterScopedResource(c, kc, gvrValidatingAdmissionPolicies()) }
func (kc *K8sController) GetValidatingAdmissionPolicyYAML(c *gin.Context) { getClusterScopedResourceYAML(c, kc, gvrValidatingAdmissionPolicies()) }
func (kc *K8sController) EditValidatingAdmissionPolicy(c *gin.Context) { editClusterScopedResource(c, kc, gvrValidatingAdmissionPolicies()) }
func (kc *K8sController) DeleteValidatingAdmissionPolicy(c *gin.Context) { deleteClusterScopedResource(c, kc, gvrValidatingAdmissionPolicies()) }

func (kc *K8sController) ListValidatingAdmissionPolicyBindings(c *gin.Context) { listClusterScopedResource(c, kc, gvrValidatingAdmissionPolicyBindings()) }
func (kc *K8sController) GetValidatingAdmissionPolicyBindingYAML(c *gin.Context) { getClusterScopedResourceYAML(c, kc, gvrValidatingAdmissionPolicyBindings()) }
func (kc *K8sController) EditValidatingAdmissionPolicyBinding(c *gin.Context) { editClusterScopedResource(c, kc, gvrValidatingAdmissionPolicyBindings()) }
func (kc *K8sController) DeleteValidatingAdmissionPolicyBinding(c *gin.Context) { deleteClusterScopedResource(c, kc, gvrValidatingAdmissionPolicyBindings()) }
