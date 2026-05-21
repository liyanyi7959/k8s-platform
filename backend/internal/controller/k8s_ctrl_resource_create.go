package controller

import (
	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

// ──────────────────────────────────────────────────────────
// 资源表单创建
// ──────────────────────────────────────────────────────────

type createWorkloadReq struct {
	Namespace  string                              `json:"namespace"`
	Name       string                              `json:"name"`
	Replicas   int32                               `json:"replicas"`
	Containers []service.CreateDeploymentContainer `json:"containers"`
	Labels     map[string]string                   `json:"labels"`
}

func bindCreateWorkload(c *gin.Context) (uint64, createWorkloadReq, bool) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return 0, createWorkloadReq{}, false
	}
	var req createWorkloadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return 0, createWorkloadReq{}, false
	}
	return id, req, true
}

// CreateDeployment 通过表单创建 Deployment。
func (kc *K8sController) CreateDeployment(c *gin.Context) {
	id, req, ok := bindCreateWorkload(c)
	if !ok {
		return
	}
	if err := kc.svc.CreateDeployment(c.Request.Context(), id, service.CreateDeploymentInput{
		Namespace:  req.Namespace,
		Name:       req.Name,
		Replicas:   req.Replicas,
		Containers: req.Containers,
		Labels:     req.Labels,
	}); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// CreateStatefulSet 通过表单创建 StatefulSet。
func (kc *K8sController) CreateStatefulSet(c *gin.Context) {
	id, req, ok := bindCreateWorkload(c)
	if !ok {
		return
	}
	if err := kc.svc.CreateStatefulSet(c.Request.Context(), id, service.CreateDeploymentInput{
		Namespace:  req.Namespace,
		Name:       req.Name,
		Replicas:   req.Replicas,
		Containers: req.Containers,
		Labels:     req.Labels,
	}); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// CreateDaemonSet 通过表单创建 DaemonSet。
func (kc *K8sController) CreateDaemonSet(c *gin.Context) {
	id, req, ok := bindCreateWorkload(c)
	if !ok {
		return
	}
	if err := kc.svc.CreateDaemonSet(c.Request.Context(), id, service.CreateDeploymentInput{
		Namespace:  req.Namespace,
		Name:       req.Name,
		Replicas:   req.Replicas,
		Containers: req.Containers,
		Labels:     req.Labels,
	}); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

type createServiceReq struct {
	Namespace string                      `json:"namespace"`
	Name      string                      `json:"name"`
	Type      string                      `json:"type"`
	Selector  map[string]string           `json:"selector"`
	Ports     []service.CreateServicePort `json:"ports"`
}

// CreateService 通过表单创建 Service。
func (kc *K8sController) CreateService(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req createServiceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.CreateService(c.Request.Context(), id, service.CreateServiceInput{
		Namespace: req.Namespace,
		Name:      req.Name,
		Type:      req.Type,
		Selector:  req.Selector,
		Ports:     req.Ports,
	}); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

type createIngressReq struct {
	Namespace     string                      `json:"namespace"`
	Name          string                      `json:"name"`
	IngressClass  string                      `json:"ingress_class"`
	Rules         []service.CreateIngressRule `json:"rules"`
	TLSSecretName string                      `json:"tls_secret_name"`
	Annotations   map[string]string           `json:"annotations"`
}

// CreateIngress 通过表单创建 Ingress。
func (kc *K8sController) CreateIngress(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req createIngressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.CreateIngress(c.Request.Context(), id, service.CreateIngressInput{
		Namespace:     req.Namespace,
		Name:          req.Name,
		IngressClass:  req.IngressClass,
		Rules:         req.Rules,
		TLSSecretName: req.TLSSecretName,
		Annotations:   req.Annotations,
	}); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}
