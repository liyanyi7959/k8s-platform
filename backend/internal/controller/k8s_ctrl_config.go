package controller

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/pkg/resp"
)

// ──────────────────────────────────────────────────────────
//  ConfigMap 相关接口
// ──────────────────────────────────────────────────────────

// ListConfigMaps 获取 ConfigMap 列表。
// @Summary ConfigMap 列表
// @Description 获取指定集群 ConfigMap 列表（可按 namespace 过滤）
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
// @Router /clusters/{id}/configmaps [get]
func (kc *K8sController) ListConfigMaps(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, gvrConfigMaps(), ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

type editConfigMapReq struct {
	Namespace string             `json:"namespace"`
	Name      string             `json:"name"`
	Labels    map[string]*string `json:"labels"`
	Data      map[string]*string `json:"data"`
}

func (kc *K8sController) EditConfigMap(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req editConfigMapReq
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
	if req.Labels != nil {
		patch["metadata"] = map[string]any{"labels": req.Labels}
	}
	if req.Data != nil {
		patch["data"] = req.Data
	}
	if len(patch) == 0 {
		resp.OK(c, gin.H{"ok": true})
		return
	}
	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvrConfigMaps(), ns, name, patch); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// DeleteConfigMap 删除 ConfigMap。
// @Summary 删除 ConfigMap
// @Description 删除指定 ConfigMap
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
// @Router /clusters/{id}/configmaps/{ns}/{name} [delete]
func (kc *K8sController) DeleteConfigMap(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrConfigMaps(), ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// GetConfigMapYAML 获取 ConfigMap YAML。
// @Summary 获取 ConfigMap YAML
// @Description 获取指定 ConfigMap 的 YAML 文本
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
// @Router /clusters/{id}/configmaps/{ns}/{name}/yaml [get]
func (kc *K8sController) GetConfigMapYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrConfigMaps(), ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// ──────────────────────────────────────────────────────────
//  ConfigMap / Secret 关联查询的公共类型与辅助函数
// ──────────────────────────────────────────────────────────

type relatedOwnerRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	UID  string `json:"uid,omitempty"`
}

type relatedController struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type relatedPod struct {
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Phase     string            `json:"phase"`
	Node      string            `json:"node"`
	Ready     string            `json:"ready"`
	Restarts  int               `json:"restarts"`
	Owners    []relatedOwnerRef `json:"owners"`
}

type relatedPodsResp struct {
	Pods        []relatedPod        `json:"pods"`
	Controllers []relatedController `json:"controllers"`
}

func podReadyAndRestarts(pod map[string]any) (string, int) {
	status := asMap(pod["status"])
	cs := asSlice(status["containerStatuses"])
	if len(cs) == 0 {
		return "-", 0
	}
	readyCount := 0
	restarts := 0
	for _, it := range cs {
		m := asMap(it)
		if b, ok := m["ready"].(bool); ok && b {
			readyCount++
		}
		if n, ok := m["restartCount"].(float64); ok {
			restarts += int(n)
		} else if n, ok := m["restartCount"].(int); ok {
			restarts += n
		}
	}
	return fmt.Sprintf("%d/%d", readyCount, len(cs)), restarts
}

func podOwners(pod map[string]any) []relatedOwnerRef {
	meta := asMap(pod["metadata"])
	refs := asSlice(meta["ownerReferences"])
	out := make([]relatedOwnerRef, 0, len(refs))
	for _, r := range refs {
		rm := asMap(r)
		kind := asString(rm["kind"])
		name := asString(rm["name"])
		if kind == "" || name == "" {
			continue
		}
		out = append(out, relatedOwnerRef{Kind: kind, Name: name, UID: asString(rm["uid"])})
	}
	return out
}

func podUsesConfigMap(pod map[string]any, cmName string) bool {
	spec := asMap(pod["spec"])

	vols := asSlice(spec["volumes"])
	for _, v := range vols {
		vm := asMap(v)
		cm := asMap(vm["configMap"])
		if asString(cm["name"]) == cmName {
			return true
		}
		proj := asMap(vm["projected"])
		sources := asSlice(proj["sources"])
		for _, s := range sources {
			sm := asMap(s)
			cms := asMap(sm["configMap"])
			if asString(cms["name"]) == cmName {
				return true
			}
		}
	}

	checkContainers := func(key string) bool {
		containers := asSlice(spec[key])
		for _, c := range containers {
			cm := asMap(c)
			envFrom := asSlice(cm["envFrom"])
			for _, ef := range envFrom {
				efm := asMap(ef)
				ref := asMap(efm["configMapRef"])
				if asString(ref["name"]) == cmName {
					return true
				}
			}
			env := asSlice(cm["env"])
			for _, e := range env {
				em := asMap(e)
				vf := asMap(em["valueFrom"])
				ref := asMap(vf["configMapKeyRef"])
				if asString(ref["name"]) == cmName {
					return true
				}
			}
		}
		return false
	}

	return checkContainers("containers") || checkContainers("initContainers")
}

func podUsesSecret(pod map[string]any, secretName string) bool {
	spec := asMap(pod["spec"])

	vols := asSlice(spec["volumes"])
	for _, v := range vols {
		vm := asMap(v)
		sec := asMap(vm["secret"])
		if asString(sec["secretName"]) == secretName {
			return true
		}
		proj := asMap(vm["projected"])
		sources := asSlice(proj["sources"])
		for _, s := range sources {
			sm := asMap(s)
			ss := asMap(sm["secret"])
			if asString(ss["name"]) == secretName {
				return true
			}
		}
		csi := asMap(vm["csi"])
		nodePublish := asMap(csi["nodePublishSecretRef"])
		if asString(nodePublish["name"]) == secretName {
			return true
		}
	}

	ips := asSlice(spec["imagePullSecrets"])
	for _, it := range ips {
		m := asMap(it)
		if asString(m["name"]) == secretName {
			return true
		}
	}

	checkContainers := func(key string) bool {
		containers := asSlice(spec[key])
		for _, c := range containers {
			cm := asMap(c)
			envFrom := asSlice(cm["envFrom"])
			for _, ef := range envFrom {
				efm := asMap(ef)
				ref := asMap(efm["secretRef"])
				if asString(ref["name"]) == secretName {
					return true
				}
			}
			env := asSlice(cm["env"])
			for _, e := range env {
				em := asMap(e)
				vf := asMap(em["valueFrom"])
				ref := asMap(vf["secretKeyRef"])
				if asString(ref["name"]) == secretName {
					return true
				}
			}
		}
		return false
	}

	return checkContainers("containers") || checkContainers("initContainers")
}

func (kc *K8sController) GetConfigMapRelated(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if strings.TrimSpace(ns) == "" || strings.TrimSpace(name) == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	list, err := kc.svc.List(c.Request.Context(), id, gvrPods(), ns, "", "", nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}

	pods := make([]relatedPod, 0, 64)
	ctrlSet := make(map[string]relatedController)
	for _, it := range list {
		obj := asMap(it)
		if obj == nil {
			continue
		}
		if !podUsesConfigMap(obj, name) {
			continue
		}
		meta := asMap(obj["metadata"])
		podName := asString(meta["name"])
		podNs := asString(meta["namespace"])
		if podName == "" || podNs == "" {
			continue
		}
		owners := podOwners(obj)
		for _, o := range owners {
			k := o.Kind + "/" + o.Name
			if _, exists := ctrlSet[k]; !exists {
				ctrlSet[k] = relatedController{Kind: o.Kind, Name: o.Name}
			}
		}
		ready, restarts := podReadyAndRestarts(obj)
		spec := asMap(obj["spec"])
		status := asMap(obj["status"])
		pods = append(pods, relatedPod{
			Namespace: podNs,
			Name:      podName,
			Phase:     asString(status["phase"]),
			Node:      asString(spec["nodeName"]),
			Ready:     ready,
			Restarts:  restarts,
			Owners:    owners,
		})
	}

	controllers := make([]relatedController, 0, len(ctrlSet))
	for _, v := range ctrlSet {
		controllers = append(controllers, v)
	}
	resp.OK(c, relatedPodsResp{Pods: pods, Controllers: controllers})
}

// ──────────────────────────────────────────────────────────
//  Secret 相关接口
// ──────────────────────────────────────────────────────────

// ListSecrets 获取 Secret 列表。
// @Summary Secret 列表
// @Description 获取指定集群 Secret 列表（可按 namespace 过滤）
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
// @Router /clusters/{id}/secrets [get]
func (kc *K8sController) ListSecrets(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, gvrSecrets(), ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func decodeSecretValueText(raw any) string {
	text, ok := raw.(string)
	if !ok {
		return asString(raw)
	}
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return text
	}
	if utf8.Valid(decoded) {
		return string(decoded)
	}
	return text
}

func buildSecretRevealText(obj map[string]any) (string, error) {
	meta := asMap(obj["metadata"])
	data := asMap(obj["data"])
	stringData := asMap(obj["stringData"])

	revealData := make(map[string]string, len(data))
	for key, raw := range data {
		revealData[key] = decodeSecretValueText(raw)
	}
	revealStringData := make(map[string]string, len(stringData))
	for key, raw := range stringData {
		revealStringData[key] = asString(raw)
	}

	payload := map[string]any{
		"apiVersion": asString(obj["apiVersion"]),
		"kind":       asString(obj["kind"]),
		"metadata": map[string]any{
			"namespace": asString(meta["namespace"]),
			"name":      asString(meta["name"]),
		},
		"type": obj["type"],
		"data": revealData,
	}
	if len(revealStringData) > 0 {
		payload["stringData"] = revealStringData
	}

	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetSecretReveal 获取 Secret 明文内容。
// @Summary 获取 Secret 明文内容
// @Description 获取指定 Secret 的明文内容，并记录查看审计日志
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
// @Router /clusters/{id}/secrets/{ns}/{name}/reveal [get]
func (kc *K8sController) GetSecretReveal(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if strings.TrimSpace(ns) == "" || strings.TrimSpace(name) == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	obj, err := kc.svc.GetObject(c.Request.Context(), id, gvrSecrets(), ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	text, err := buildSecretRevealText(obj)
	if err != nil {
		resp.Fail(c, 5000, "生成 Secret 明文失败")
		return
	}

	var userID uint64
	var username string
	if claims, ok := middleware.GetClaims(c); ok && claims != nil {
		if claims.UserID > 0 {
			userID = uint64(claims.UserID)
		}
		username = strings.TrimSpace(claims.Username)
	}
	zap.L().Warn("k8s_secret_reveal",
		zap.Uint64("user_id", userID),
		zap.String("username", username),
		zap.Uint64("cluster_id", id),
		zap.String("secret", ns+"/"+name),
		zap.String("request_id", c.GetString("request_id")),
	)

	resp.OK(c, gin.H{"text": text})
}

type editSecretReq struct {
	Namespace string             `json:"namespace"`
	Name      string             `json:"name"`
	Type      *string            `json:"type"`
	Labels    map[string]*string `json:"labels"`
	Data      map[string]*string `json:"data"`
}

func (kc *K8sController) EditSecret(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req editSecretReq
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
	if req.Type != nil {
		if t := strings.TrimSpace(*req.Type); t != "" {
			patch["type"] = t
		}
	}
	if req.Labels != nil {
		patch["metadata"] = map[string]any{"labels": req.Labels}
	}
	if req.Data != nil {
		patch["data"] = req.Data
	}
	if len(patch) == 0 {
		resp.OK(c, gin.H{"ok": true})
		return
	}
	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvrSecrets(), ns, name, patch); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// DeleteSecret 删除 Secret。
// @Summary 删除 Secret
// @Description 删除指定 Secret
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
// @Router /clusters/{id}/secrets/{ns}/{name} [delete]
func (kc *K8sController) DeleteSecret(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrSecrets(), ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// GetSecretYAML 获取 Secret YAML。
// @Summary 获取 Secret YAML
// @Description 获取指定 Secret 的 YAML 文本
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
// @Router /clusters/{id}/secrets/{ns}/{name}/yaml [get]
func (kc *K8sController) GetSecretYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrSecrets(), ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func (kc *K8sController) GetSecretRelated(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if strings.TrimSpace(ns) == "" || strings.TrimSpace(name) == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	list, err := kc.svc.List(c.Request.Context(), id, gvrPods(), ns, "", "", nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}

	pods := make([]relatedPod, 0, 64)
	ctrlSet := make(map[string]relatedController)
	for _, it := range list {
		obj := asMap(it)
		if obj == nil {
			continue
		}
		if !podUsesSecret(obj, name) {
			continue
		}
		meta := asMap(obj["metadata"])
		podName := asString(meta["name"])
		podNs := asString(meta["namespace"])
		if podName == "" || podNs == "" {
			continue
		}
		owners := podOwners(obj)
		for _, o := range owners {
			k := o.Kind + "/" + o.Name
			if _, exists := ctrlSet[k]; !exists {
				ctrlSet[k] = relatedController{Kind: o.Kind, Name: o.Name}
			}
		}
		ready, restarts := podReadyAndRestarts(obj)
		spec := asMap(obj["spec"])
		status := asMap(obj["status"])
		pods = append(pods, relatedPod{
			Namespace: podNs,
			Name:      podName,
			Phase:     asString(status["phase"]),
			Node:      asString(spec["nodeName"]),
			Ready:     ready,
			Restarts:  restarts,
			Owners:    owners,
		})
	}

	controllers := make([]relatedController, 0, len(ctrlSet))
	for _, v := range ctrlSet {
		controllers = append(controllers, v)
	}
	resp.OK(c, relatedPodsResp{Pods: pods, Controllers: controllers})
}
