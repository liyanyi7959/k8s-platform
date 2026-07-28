package controller

import (
	"github.com/gin-gonic/gin"

	"k8s-platform-backend/pkg/resp"
)

func (kc *K8sController) GetNamespaceInspection(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if kc.namespaceDiagnosisSvc == nil {
		resp.Fail(c, 5000, "namespace inspection service unavailable")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	result, err := kc.namespaceDiagnosisSvc.GetNamespaceInspection(c.Request.Context(), id, ns)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, result)
}

func (kc *K8sController) GetNamespaceWorkloadInventory(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if kc.namespaceDiagnosisSvc == nil {
		resp.Fail(c, 5000, "namespace workload inventory service unavailable")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	result, err := kc.namespaceDiagnosisSvc.GetNamespaceWorkloadInventory(c.Request.Context(), id, ns)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, result)
}

func (kc *K8sController) GetPodInspection(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if kc.resourceInspectSvc == nil {
		resp.Fail(c, 5000, "pod inspection service unavailable")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("pod"))
	result, err := kc.resourceInspectSvc.InspectPod(c.Request.Context(), id, ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, result)
}
