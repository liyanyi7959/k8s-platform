package controller

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/remotecommand"

	"k8s-platform-backend/internal/service"
)

// ──────────────────────────────────────────────────────────
//  K8sController 结构体与构造
// ──────────────────────────────────────────────────────────

// K8sController 负责把 HTTP 请求参数解析/校验后，转交给 K8sService 执行具体的
// Kubernetes 操作，并将 service 层错误映射为统一的业务错误码返回给前端。
//
// 各资源类型的接口按文件拆分，参见 k8s_ctrl_*.go 系列文件。
type K8sController struct {
	svc          *service.K8sService
	execSessions *service.ExecSessionStore
	logSessions  *service.PodLogSessionStore
}

type K8sEditRequest struct {
	Namespace string `json:"namespace"`
	YAML      string `json:"yaml"`
}

type K8sNamespacedEditRequest struct {
	Namespace string `json:"namespace" binding:"required"`
	YAML      string `json:"yaml" binding:"required"`
}

// NewK8sController 创建 K8sController。
func NewK8sController(svc *service.K8sService, execSessions *service.ExecSessionStore, logSessions *service.PodLogSessionStore) *K8sController {
	return &K8sController{svc: svc, execSessions: execSessions, logSessions: logSessions}
}

// ──────────────────────────────────────────────────────────
//  请求参数解析辅助
// ──────────────────────────────────────────────────────────

// parseClusterID 从路由参数中解析 cluster id。
// 兼容两种参数名：clusterId 与 id；解析失败或值为 0 时返回 (0, false)。
func parseClusterID(c *gin.Context) (uint64, bool) {
	raw := c.Param("clusterId")
	if raw == "" {
		raw = c.Param("id")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint64(id), true
}

// decodePathParam 对路径参数做 URL 解码。
// 主要用于兼容资源名/命名空间中包含需要转义的字符；解码失败时回退返回原字符串。
func decodePathParam(s string) string {
	v, err := url.PathUnescape(s)
	if err != nil {
		return s
	}
	return v
}

// ──────────────────────────────────────────────────────────
//  Swagger 文档占位类型
// ──────────────────────────────────────────────────────────

type K8sListResp struct {
	List []AnyMap `json:"list"`
}

type K8sTextResp struct {
	Text string `json:"text"`
}

type K8sExecSessionResp struct {
	SessionID string `json:"session_id"`
	WsURL     string `json:"ws_url"`
}

type K8sLogSessionResp struct {
	SessionID string `json:"session_id"`
	WsURL     string `json:"ws_url"`
}

type K8sOKResp struct {
	OK bool `json:"ok"`
}

// ──────────────────────────────────────────────────────────
//  通用 map 辅助函数
// ──────────────────────────────────────────────────────────

func mergeMap(cur any, add map[string]any) map[string]any {
	base, _ := cur.(map[string]any)
	if base == nil {
		base = map[string]any{}
	}
	for k, v := range add {
		if vm, ok := v.(map[string]any); ok {
			if curm, ok2 := base[k].(map[string]any); ok2 {
				base[k] = mergeMap(curm, vm)
			} else {
				base[k] = mergeMap(nil, vm)
			}
			continue
		}
		base[k] = v
	}
	return base
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return fmt.Sprint(v)
	}
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	arr, _ := v.([]any)
	return arr
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// ──────────────────────────────────────────────────────────
//  工作负载编辑辅助：资源、探针
// ──────────────────────────────────────────────────────────

func ensureResourceMaps(container map[string]any, upd *editDeploymentResources) {
	if container == nil || upd == nil {
		return
	}
	res, _ := container["resources"].(map[string]any)
	if res == nil {
		res = map[string]any{}
		container["resources"] = res
	}

	if upd.Requests != nil {
		reqs := map[string]any{}
		for k, v := range upd.Requests {
			kk := strings.TrimSpace(k)
			vv := strings.TrimSpace(v)
			if kk == "" || vv == "" {
				continue
			}
			reqs[kk] = vv
		}
		if len(reqs) > 0 {
			res["requests"] = reqs
		}
	}
	if upd.Limits != nil {
		lims := map[string]any{}
		for k, v := range upd.Limits {
			kk := strings.TrimSpace(k)
			vv := strings.TrimSpace(v)
			if kk == "" || vv == "" {
				continue
			}
			lims[kk] = vv
		}
		if len(lims) > 0 {
			res["limits"] = lims
		}
	}
}

func applyProbeTiming(container map[string]any, probeKey string, timing *editProbeTiming) {
	if container == nil || timing == nil {
		return
	}
	if timing.InitialDelaySeconds == nil &&
		timing.TimeoutSeconds == nil &&
		timing.PeriodSeconds == nil &&
		timing.SuccessThreshold == nil &&
		timing.FailureThreshold == nil {
		return
	}

	probe, _ := container[probeKey].(map[string]any)
	if probe == nil {
		probe = map[string]any{}
		container[probeKey] = probe
	}

	if timing.InitialDelaySeconds != nil {
		probe["initialDelaySeconds"] = *timing.InitialDelaySeconds
	}
	if timing.TimeoutSeconds != nil {
		probe["timeoutSeconds"] = *timing.TimeoutSeconds
	}
	if timing.PeriodSeconds != nil {
		probe["periodSeconds"] = *timing.PeriodSeconds
	}
	if timing.SuccessThreshold != nil {
		probe["successThreshold"] = *timing.SuccessThreshold
	}
	if timing.FailureThreshold != nil {
		probe["failureThreshold"] = *timing.FailureThreshold
	}
}

// ──────────────────────────────────────────────────────────
//  GVR 工厂函数
// ──────────────────────────────────────────────────────────

func gvrNamespaces() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
}
func gvrNodes() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}
}
func gvrPods() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
}
func gvrReplicaSets() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}
}
func gvrServices() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
}
func gvrEndpoints() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"}
}
func gvrConfigMaps() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
}
func gvrSecrets() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
}
func gvrEvents() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}
}
func gvrPVs() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"}
}
func gvrPVCs() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}
}
func gvrIngresses() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}
}
func gvrNetworkPolicies() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}
}
func gvrEndpointSlices() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"}
}
func gvrIngressClasses() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"}
}
func gvrStorageClasses() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"}
}
func gvrCSIDrivers() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csidrivers"}
}
func gvrCSINodes() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csinodes"}
}
func gvrCSIStorageCapacities() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csistoragecapacities"}
}
func gvrVolumeAttachments() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "volumeattachments"}
}
func gvrVolumeSnapshots() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshots"}
}
func gvrVolumeSnapshotClasses() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotclasses"}
}
func gvrVolumeSnapshotContents() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshotcontents"}
}
func gvrResourceQuotas() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "resourcequotas"}
}
func gvrLimitRanges() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "limitranges"}
}
func gvrCustomResourceDefinitions() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
}
func gvrAPIServices() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"}
}
func gvrPriorityClasses() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses"}
}
func gvrRuntimeClasses() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "node.k8s.io", Version: "v1", Resource: "runtimeclasses"}
}
func gvrValidatingWebhookConfigurations() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}
}
func gvrMutatingWebhookConfigurations() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}
}
func gvrValidatingAdmissionPolicies() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingadmissionpolicies"}
}
func gvrValidatingAdmissionPolicyBindings() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingadmissionpolicybindings"}
}
func gvrJobs() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
}
func gvrCronJobs() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
}
func gvrServiceAccounts() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}
}
func gvrPDBs() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"}
}
func gvrHPAs() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}
}
func gvrRoles() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"}
}
func gvrClusterRoles() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}
}
func gvrRoleBindings() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}
}
func gvrClusterRoleBindings() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"}
}
func gvrLeases() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1", Resource: "leases"}
}

func gvrWorkloadKind(kind string) (schema.GroupVersionResource, bool) {
	switch strings.TrimSpace(kind) {
	case "Deployment":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, true
	case "StatefulSet":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, true
	case "DaemonSet":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, true
	default:
		return schema.GroupVersionResource{}, false
	}
}

// ──────────────────────────────────────────────────────────
//  时间 / 错误映射 / WebSocket 辅助
// ──────────────────────────────────────────────────────────

func timeRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// writeServiceErr 将 service 层错误映射为前端约定的业务错误码。
// 委托给共享 WriteServiceErr，追加 K8s 领域特定映射。
func (kc *K8sController) writeServiceErr(c *gin.Context, err error) {
	WriteServiceErr(c, err, K8sErrMappings...)
}

type wsWriter struct {
	write func([]byte) error
}

// Write 实现 io.Writer，将后端 exec 输出转发到 WebSocket。
func (w *wsWriter) Write(p []byte) (int, error) {
	if w == nil || w.write == nil {
		return 0, io.ErrClosedPipe
	}
	if err := w.write(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

type terminalSizeQueue struct {
	ch <-chan remotecommand.TerminalSize
}

func (q *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	if q == nil || q.ch == nil {
		return nil
	}
	sz, ok := <-q.ch
	if !ok {
		return nil
	}
	return &sz
}
