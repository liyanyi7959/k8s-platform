package controller

import (
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/pkg/resp"
)

// ──────────────────────────────────────────────────────────
//  Service 相关接口
// ──────────────────────────────────────────────────────────

// ListServices 获取 Service 列表。
// @Summary Service 列表
// @Description 获取指定集群 Service 列表（可按 namespace 过滤）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间（不填表示全部）"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/services [get]
func (kc *K8sController) ListServices(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, gvrServices(), ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

type editServiceReq struct {
	Namespace   string             `json:"namespace"`
	Name        string             `json:"name"`
	Type        *string            `json:"type"`
	Labels      map[string]*string `json:"labels"`
	Annotations map[string]*string `json:"annotations"`
	Selector    map[string]*string `json:"selector"`
}

func (kc *K8sController) EditService(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req editServiceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(req.Namespace)
	name := strings.TrimSpace(req.Name)
	if ns == "" || name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	patch := map[string]any{}
	if req.Labels != nil || req.Annotations != nil {
		meta := map[string]any{}
		if req.Labels != nil {
			meta["labels"] = req.Labels
		}
		if req.Annotations != nil {
			meta["annotations"] = req.Annotations
		}
		if len(meta) > 0 {
			patch["metadata"] = meta
		}
	}
	if req.Type != nil || req.Selector != nil {
		spec := map[string]any{}
		if req.Type != nil {
			if t := strings.TrimSpace(*req.Type); t != "" {
				spec["type"] = t
			} else {
				spec["type"] = nil
			}
		}
		if req.Selector != nil {
			spec["selector"] = req.Selector
		}
		if len(spec) > 0 {
			patch["spec"] = spec
		}
	}
	if len(patch) == 0 {
		resp.OK(c, gin.H{"ok": true})
		return
	}
	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvrServices(), ns, name, patch); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// DeleteService 删除 Service。
// @Summary 删除 Service
// @Description 删除指定 Service
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/services/{ns}/{name} [delete]
func (kc *K8sController) DeleteService(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrServices(), ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// GetServiceYAML 获取 Service YAML。
// @Summary 获取 Service YAML
// @Description 获取指定 Service 的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/services/{ns}/{name}/yaml [get]
func (kc *K8sController) GetServiceYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrServices(), ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// ──────────────────────────────────────────────────────────
//  Ingress 相关接口
// ──────────────────────────────────────────────────────────

// ListIngresses 获取 Ingress 列表。
// @Summary Ingress 列表
// @Description 获取指定集群 Ingress 列表（可按 namespace 过滤）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间（不填表示全部）"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/ingresses [get]
func (kc *K8sController) ListIngresses(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, gvrIngresses(), ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

type editIngressReq struct {
	Namespace        string             `json:"namespace"`
	Name             string             `json:"name"`
	IngressClassName *string            `json:"ingressClassName"`
	Labels           map[string]*string `json:"labels"`
	Annotations      map[string]*string `json:"annotations"`
}

func (kc *K8sController) EditIngress(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req editIngressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(req.Namespace)
	name := strings.TrimSpace(req.Name)
	if ns == "" || name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	patch := map[string]any{}
	if req.Labels != nil || req.Annotations != nil {
		meta := map[string]any{}
		if req.Labels != nil {
			meta["labels"] = req.Labels
		}
		if req.Annotations != nil {
			meta["annotations"] = req.Annotations
		}
		if len(meta) > 0 {
			patch["metadata"] = meta
		}
	}
	if req.IngressClassName != nil {
		spec := map[string]any{}
		if cn := strings.TrimSpace(*req.IngressClassName); cn != "" {
			spec["ingressClassName"] = cn
		} else {
			spec["ingressClassName"] = nil
		}
		patch["spec"] = spec
	}
	if len(patch) == 0 {
		resp.OK(c, gin.H{"ok": true})
		return
	}
	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvrIngresses(), ns, name, patch); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// DeleteIngress 删除 Ingress。
// @Summary 删除 Ingress
// @Description 删除指定 Ingress
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/ingresses/{ns}/{name} [delete]
func (kc *K8sController) DeleteIngress(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrIngresses(), ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// GetIngressYAML 获取 Ingress YAML。
// @Summary 获取 Ingress YAML
// @Description 获取指定 Ingress 的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/ingresses/{ns}/{name}/yaml [get]
func (kc *K8sController) GetIngressYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrIngresses(), ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// ──────────────────────────────────────────────────────────
//  IngressClass 相关接口
// ──────────────────────────────────────────────────────────

// ListIngressClasses 获取 IngressClass 列表。
// @Summary IngressClass 列表
// @Description 获取指定集群 IngressClass 列表
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/ingressclasses [get]
func (kc *K8sController) ListIngressClasses(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrIngressClasses(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

type editIngressClassReq struct {
	Name        string             `json:"name"`
	Controller  *string            `json:"controller"`
	IsDefault   *bool              `json:"isDefault"`
	Labels      map[string]*string `json:"labels"`
	Annotations map[string]*string `json:"annotations"`
}

func (kc *K8sController) EditIngressClass(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req editIngressClassReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	patch := map[string]any{}
	if req.Labels != nil || req.Annotations != nil || req.IsDefault != nil {
		meta := map[string]any{}
		if req.Labels != nil {
			meta["labels"] = req.Labels
		}
		if req.Annotations != nil || req.IsDefault != nil {
			ann := map[string]*string{}
			for k, v := range req.Annotations {
				ann[k] = v
			}
			if req.IsDefault != nil {
				key := "ingressclass.kubernetes.io/is-default-class"
				if *req.IsDefault {
					val := "true"
					ann[key] = &val
				} else {
					ann[key] = nil
				}
			}
			meta["annotations"] = ann
		}
		if len(meta) > 0 {
			patch["metadata"] = meta
		}
	}
	if req.Controller != nil {
		spec := map[string]any{}
		if ctl := strings.TrimSpace(*req.Controller); ctl != "" {
			spec["controller"] = ctl
		} else {
			resp.Fail(c, 4000, "invalid params")
			return
		}
		patch["spec"] = spec
	}
	if len(patch) == 0 {
		resp.OK(c, gin.H{"ok": true})
		return
	}
	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvrIngressClasses(), "", name, patch); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// GetIngressClassYAML 获取 IngressClass YAML。
// @Summary 获取 IngressClass YAML
// @Description 获取指定 IngressClass 的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/ingressclasses/{name}/yaml [get]
func (kc *K8sController) GetIngressClassYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrIngressClasses(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// DeleteIngressClass 删除 IngressClass。
// @Summary 删除 IngressClass
// @Description 删除指定 IngressClass
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/ingressclasses/{name} [delete]
func (kc *K8sController) DeleteIngressClass(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrIngressClasses(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}
