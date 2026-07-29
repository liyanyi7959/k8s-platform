package service

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"crypto/x509"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/yaml"

	fleetapp "k8s-platform-backend/internal/fleet/application"
	kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsruntime "k8s-platform-backend/internal/kops/adapters/runtime"
)

// K8s 哨兵错误已统一迁移至 errors.go（ErrK8s / ErrK8sNetwork / ErrK8sTimeout 等）。

// K8sService 封装对 Kubernetes API 的访问。
// 设计目标：
// - 屏蔽 kubeconfig/RESTConfig/client 初始化细节；
// - 统一将 K8s 常见错误（NotFound/AlreadyExists/BadRequest 等）归一化为业务错误；
// - 为 controller 层提供面向"资源 + 动作"的方法（List/GetYAML/Delete/Patch/Exec 等）。
type K8sService struct {
	clusterReg *fleetapp.Registry
	podCache   *podCacheManager
	objCache   *objCacheManager
	cache      CacheStore
	podTTL     time.Duration
	clients    kopsclient.ClientFactory

	apiResourcesMu       sync.Mutex
	apiResourcesInFlight map[string]*apiResourcesFlight
}

const k8sRequestTimeout = 60 * time.Second
const k8sListPageLimit int64 = 500

// NewK8sService 创建 K8sService。
func NewK8sService(clusterReg *fleetapp.Registry, cacheStore CacheStore, podCacheTTL time.Duration, insecureSkipTLS ...bool) *K8sService {
	if podCacheTTL <= 0 {
		podCacheTTL = 60 * time.Second
	}
	skip := false
	if len(insecureSkipTLS) > 0 {
		skip = insecureSkipTLS[0]
	}
	return &K8sService{clusterReg: clusterReg, podCache: newPodCacheManager(), objCache: newObjCacheManager(), cache: cacheStore, podTTL: podCacheTTL, clients: kopsclient.NewClientFactory(k8sRequestTimeout, skip)}
}

// ---------------------------------------------------------------------------
// Client helpers
// ---------------------------------------------------------------------------

// restConfig 根据集群 ID 获取访问该集群的 *rest.Config。
func (s *K8sService) restConfig(ctx context.Context, clusterID uint64) (*rest.Config, error) {
	if s.clusterReg == nil {
		return nil, errors.New("cluster registry is required")
	}
	kc, err := s.clusterReg.Kubeconfig(ctx, clusterID)
	if err != nil {
		return nil, legacyClusterError(err)
	}
	cfg, err := s.clientFactory().RESTConfig(kc)
	if err != nil {
		return nil, ErrWithMessage(ErrInvalidParams, "集群凭据无效")
	}
	return cfg, nil
}

// GetKubeconfig 获取指定集群的 kubeconfig 内容，供 Helm CLI 等外部工具使用。
func (s *K8sService) GetKubeconfig(ctx context.Context, clusterID uint64) (string, error) {
	if s.clusterReg == nil {
		return "", errors.New("cluster registry is required")
	}
	value, err := s.clusterReg.Kubeconfig(ctx, clusterID)
	return value, legacyClusterError(err)
}

func (s *K8sService) ValidateKubeconfig(ctx context.Context, kubeconfig string) error {
	err := s.clientFactory().Validate(ctx, kubeconfig)
	if errors.Is(err, kopsclient.ErrKubeconfigEmpty) {
		return ErrWithMessage(ErrInvalidParams, "kubeconfig 不能为空")
	}
	if errors.Is(err, kopsclient.ErrKubeconfigTooLarge) {
		return ErrWithMessage(ErrInvalidParams, "kubeconfig 内容不能超过 1MB")
	}
	if errors.Is(err, kopsclient.ErrInvalidKubeconfig) {
		return ErrWithMessage(ErrInvalidParams, "kubeconfig 格式无效，请上传原始 YAML/JSON 或其 Base64 内容")
	}
	return normalizeK8sErr(err)
}

// VerifyClusterAPI verifies that the saved cluster credential can reach the
// Kubernetes API and returns the server version used by deployment preflight.
// It intentionally performs a real discovery call instead of only validating
// kubeconfig syntax.
func (s *K8sService) VerifyClusterAPI(ctx context.Context, clusterID uint64) (string, error) {
	if clusterID == 0 {
		return "", ErrInvalidParams
	}
	client, err := s.discoveryClient(ctx, clusterID)
	if err != nil {
		return "", err
	}
	version, err := client.ServerVersion()
	if err != nil {
		return "", normalizeK8sErr(err)
	}
	if version == nil {
		return "", ErrWithMessage(ErrK8sNetwork, "Kubernetes API 未返回版本信息")
	}
	return version.GitVersion, nil
}

// ValidateKubeconfigFormat 仅校验 kubeconfig 格式是否合法（能正确解析出 REST 配置），
// 不会实际连接 K8s API Server。适用于编辑/更新场景——用户可能在离线环境中更新凭据。
func (s *K8sService) ValidateKubeconfigFormat(_ context.Context, kubeconfig string) error {
	_, err := kopsclient.NormalizeKubeconfigContent(kubeconfig)
	return legacyKubeconfigError(err)
}

// dynamicClient 创建 dynamic client，用于访问任意 GVR（包含 CRD）资源。
func (s *K8sService) dynamicClient(ctx context.Context, clusterID uint64) (*dynamic.DynamicClient, error) {
	cfg, err := s.restConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return s.clientFactory().DynamicClient(cfg)
}

func (s *K8sService) discoveryClient(ctx context.Context, clusterID uint64) (*discovery.DiscoveryClient, error) {
	cfg, err := s.restConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return s.clientFactory().DiscoveryClient(cfg)
}

// typedClient 创建 typed client（Clientset），用于访问 core/apps 等内置资源的强类型接口。
func (s *K8sService) typedClient(ctx context.Context, clusterID uint64) (*kubernetes.Clientset, error) {
	cfg, err := s.restConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return s.clientFactory().TypedClient(cfg)
}

// TypedClient exposes the typed Kubernetes client to bounded-context runtime
// adapters. Business operations must stay outside this retained transport
// service so they can be owned by their respective application services.
func (s *K8sService) TypedClient(ctx context.Context, clusterID uint64) (*kubernetes.Clientset, error) {
	return s.typedClient(ctx, clusterID)
}

// RESTConfig exposes the bounded credential-to-REST-config bridge required by
// infrastructure runtimes that implement Kubernetes streaming protocols.
// Resource policy remains outside this retained transport service.
func (s *K8sService) RESTConfig(ctx context.Context, clusterID uint64) (*rest.Config, error) {
	return s.restConfig(ctx, clusterID)
}

// DynamicClient exposes the Kubernetes dynamic client to bounded-context
// runtime adapters. Resource policy and metrics aggregation remain outside
// this retained transport service.
func (s *K8sService) DynamicClient(ctx context.Context, clusterID uint64) (*dynamic.DynamicClient, error) {
	return s.dynamicClient(ctx, clusterID)
}

// DiscoveryClient exposes Kubernetes discovery to bounded-context runtime
// adapters. Resource selection and business aggregation must remain outside
// this retained transport service.
func (s *K8sService) DiscoveryClient(ctx context.Context, clusterID uint64) (*discovery.DiscoveryClient, error) {
	return s.discoveryClient(ctx, clusterID)
}

func (s *K8sService) typedClientForInformer(ctx context.Context, clusterID uint64) (*kubernetes.Clientset, error) {
	cfg, err := s.restConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return s.clientFactory().InformerClient(cfg)
}

// ObjectInformer exposes the retained informer lifecycle as a narrow
// transport bridge. Bounded Kops runtimes own resource read and cache policy.
func (s *K8sService) ObjectInformer(ctx context.Context, clusterID uint64, kind string) (cache.SharedIndexInformer, error) {
	if strings.TrimSpace(kind) == "pods" {
		entry, err := s.getOrStartPodCache(ctx, clusterID)
		if err != nil {
			return nil, err
		}
		if entry == nil {
			return nil, ErrK8s
		}
		return entry.informer, nil
	}
	entry, err := s.getOrStartObjCache(ctx, clusterID, kind)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, ErrK8s
	}
	return entry.informer, nil
}

func (s *K8sService) ObjectCacheEnabled() bool {
	return s != nil && s.cache != nil && s.cache.Enabled()
}

func (s *K8sService) ObjectCacheTTL() time.Duration {
	if s == nil {
		return 0
	}
	return s.podTTL
}

func (s *K8sService) ObjectCacheGet(ctx context.Context, key string) ([]byte, bool, error) {
	if s == nil || s.cache == nil || !s.cache.Enabled() {
		return nil, false, nil
	}
	return s.cache.Get(ctx, key)
}

func (s *K8sService) ObjectCacheSet(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if s == nil || s.cache == nil || !s.cache.Enabled() {
		return nil
	}
	return s.cache.Set(ctx, key, value, ttl)
}

func (s *K8sService) NormalizeKubernetesRuntimeError(err error) error {
	return normalizeK8sErr(err)
}

func (s *K8sService) clientFactory() kopsclient.ClientFactory {
	if s != nil {
		return s.clients
	}
	return kopsclient.NewClientFactory(k8sRequestTimeout, false)
}

func legacyKubeconfigError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, kopsclient.ErrKubeconfigEmpty):
		return ErrWithMessage(ErrInvalidParams, "kubeconfig 不能为空")
	case errors.Is(err, kopsclient.ErrKubeconfigTooLarge):
		return ErrWithMessage(ErrInvalidParams, "kubeconfig 内容不能超过 1MB")
	default:
		return ErrWithMessage(ErrInvalidParams, "kubeconfig 格式无效，请上传原始 YAML/JSON 或其 Base64 内容")
	}
}

// ---------------------------------------------------------------------------
// Error normalization
// ---------------------------------------------------------------------------

// normalizeK8sErr 将 Kubernetes API 常见错误归一化为业务错误。
func normalizeK8sErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrK8sTimeout
	}
	switch {
	case apierrors.IsNotFound(err):
		return ErrNotFound
	case apierrors.IsAlreadyExists(err):
		return ErrConflict
	case apierrors.IsInvalid(err), apierrors.IsBadRequest(err):
		return ErrInvalidParams
	case apierrors.IsUnauthorized(err):
		return ErrK8sUnauthorized
	case apierrors.IsForbidden(err):
		return ErrK8sForbidden
	case apierrors.IsTimeout(err):
		return ErrK8sTimeout
	default:
		var uerr *url.Error
		if errors.As(err, &uerr) && uerr != nil {
			if inner := uerr.Unwrap(); inner != nil {
				err = inner
			}
		}
		var x509UnknownAuth *x509.UnknownAuthorityError
		var x509Hostname x509.HostnameError
		var x509Invalid x509.CertificateInvalidError
		lower := strings.ToLower(err.Error())
		if errors.As(err, &x509UnknownAuth) || errors.As(err, &x509Hostname) || errors.As(err, &x509Invalid) || strings.Contains(lower, "x509:") {
			return ErrK8sTLS
		}
		var nerr net.Error
		if errors.As(err, &nerr) {
			if nerr.Timeout() {
				return ErrK8sTimeout
			}
			return ErrK8sNetwork
		}
		if strings.Contains(lower, "connection refused") ||
			strings.Contains(lower, "no route to host") ||
			strings.Contains(lower, "i/o timeout") ||
			strings.Contains(lower, "timeout") {
			return ErrK8sNetwork
		}
		return ErrK8s
	}
}

// NormalizeKubernetesError translates raw Kubernetes client errors into the
// retained transport error vocabulary for context adapters.
func NormalizeKubernetesError(err error) error {
	return normalizeK8sErr(err)
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// sortUnstructuredList 对 unstructured 列表做简单排序。
func sortUnstructuredList(items []unstructured.Unstructured, sortBy, order string) []unstructured.Unstructured {
	sb := strings.TrimSpace(sortBy)
	if sb == "" {
		return items
	}
	desc := strings.ToLower(strings.TrimSpace(order)) == "desc"

	readString := func(u unstructured.Unstructured, path string) string {
		if path == "metadata.name" {
			return u.GetName()
		}
		if path == "metadata.namespace" {
			return u.GetNamespace()
		}
		return ""
	}

	sort.Slice(items, func(i, j int) bool {
		a := readString(items[i], sb)
		b := readString(items[j], sb)
		if desc {
			return a > b
		}
		return a < b
	})
	return items
}

// unstructuredToAnyList 将 unstructured.Unstructured 转为 JSON 可序列化的 []any。
func unstructuredToAnyList(items []unstructured.Unstructured) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		out = append(out, it.Object)
	}
	return out
}

// renderYAML 将任意对象渲染为 YAML 文本。
func renderYAML(obj any) (string, error) {
	b, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(b))
	if text == "" {
		return "---\n", nil
	}
	return fmt.Sprintf("---\n%s\n", text), nil
}

// mapToListOptions 将额外查询参数映射为 K8s ListOptions。
func mapToListOptions(m map[string]string) metav1.ListOptions {
	var opts metav1.ListOptions
	if m == nil {
		return opts
	}
	if fs := strings.TrimSpace(m["field_selector"]); fs != "" {
		opts.FieldSelector = fs
	}
	if ls := strings.TrimSpace(m["label_selector"]); ls != "" {
		opts.LabelSelector = ls
	}
	return opts
}

// ---------------------------------------------------------------------------
// GVR check helpers
// ---------------------------------------------------------------------------

func isCoreV1PodsGVR(gvr schema.GroupVersionResource) bool {
	return gvr.Group == "" && gvr.Version == "v1" && gvr.Resource == "pods"
}

func isAppsV1DeploymentsGVR(gvr schema.GroupVersionResource) bool {
	return gvr.Group == "apps" && gvr.Version == "v1" && gvr.Resource == "deployments"
}

func isAppsV1StatefulSetsGVR(gvr schema.GroupVersionResource) bool {
	return gvr.Group == "apps" && gvr.Version == "v1" && gvr.Resource == "statefulsets"
}

func isCoreV1ConfigMapsGVR(gvr schema.GroupVersionResource) bool {
	return gvr.Group == "" && gvr.Version == "v1" && gvr.Resource == "configmaps"
}

func isCoreV1SecretsGVR(gvr schema.GroupVersionResource) bool {
	return gvr.Group == "" && gvr.Version == "v1" && gvr.Resource == "secrets"
}

func isCoreV1ServiceAccountsGVR(gvr schema.GroupVersionResource) bool {
	return gvr.Group == "" && gvr.Version == "v1" && gvr.Resource == "serviceaccounts"
}

func isAutoscalingV2HPAGVR(gvr schema.GroupVersionResource) bool {
	return gvr.Group == "autoscaling" && gvr.Version == "v2" && gvr.Resource == "horizontalpodautoscalers"
}

func needsVersionCompatibility(gvr schema.GroupVersionResource) bool {
	switch {
	case gvr.Group == "autoscaling" && gvr.Resource == "horizontalpodautoscalers":
		return true
	case gvr.Group == "policy" && gvr.Resource == "poddisruptionbudgets":
		return true
	case gvr.Group == "batch" && gvr.Resource == "cronjobs":
		return true
	case gvr.Group == "networking.k8s.io" && gvr.Resource == "ingresses":
		return true
	case gvr.Group == "networking.k8s.io" && gvr.Resource == "ingressclasses":
		return true
	case gvr.Group == "networking.k8s.io" && gvr.Resource == "networkpolicies":
		return true
	case gvr.Group == "discovery.k8s.io" && gvr.Resource == "endpointslices":
		return true
	case gvr.Group == "storage.k8s.io" && gvr.Resource == "volumeattachments":
		return true
	case gvr.Group == "coordination.k8s.io" && gvr.Resource == "leases":
		return true
	case gvr.Group == "rbac.authorization.k8s.io" && gvr.Resource == "clusterroles":
		return true
	case gvr.Group == "apiextensions.k8s.io" && gvr.Resource == "customresourcedefinitions":
		return true
	case gvr.Group == "apiregistration.k8s.io" && gvr.Resource == "apiservices":
		return true
	case gvr.Group == "scheduling.k8s.io" && gvr.Resource == "priorityclasses":
		return true
	case gvr.Group == "admissionregistration.k8s.io" && gvr.Resource == "validatingwebhookconfigurations":
		return true
	case gvr.Group == "admissionregistration.k8s.io" && gvr.Resource == "mutatingwebhookconfigurations":
		return true
	case gvr.Group == "apps" && gvr.Resource == "replicasets":
		return true
	case gvr.Group == "snapshot.storage.k8s.io" && gvr.Resource == "volumesnapshots":
		return true
	case gvr.Group == "snapshot.storage.k8s.io" && gvr.Resource == "volumesnapshotclasses":
		return true
	case gvr.Group == "snapshot.storage.k8s.io" && gvr.Resource == "volumesnapshotcontents":
		return true
	default:
		return false
	}
}

func candidateGVRs(gvr schema.GroupVersionResource) []schema.GroupVersionResource {
	if !needsVersionCompatibility(gvr) {
		return []schema.GroupVersionResource{gvr}
	}

	seen := map[string]bool{}
	appendUnique := func(out []schema.GroupVersionResource, item schema.GroupVersionResource) []schema.GroupVersionResource {
		key := item.Group + "/" + item.Version + "/" + item.Resource
		if seen[key] {
			return out
		}
		seen[key] = true
		return append(out, item)
	}

	out := make([]schema.GroupVersionResource, 0, 4)
	out = appendUnique(out, gvr)

	switch {
	case gvr.Group == "autoscaling" && gvr.Resource == "horizontalpodautoscalers":
		out = appendUnique(out, schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "autoscaling", Version: "v2beta2", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "autoscaling", Version: "v1", Resource: gvr.Resource})
	case gvr.Group == "policy" && gvr.Resource == "poddisruptionbudgets":
		out = appendUnique(out, schema.GroupVersionResource{Group: "policy", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "policy", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "batch" && gvr.Resource == "cronjobs":
		out = appendUnique(out, schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "batch", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "networking.k8s.io" && gvr.Resource == "ingresses":
		out = appendUnique(out, schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "networking.k8s.io" && gvr.Resource == "ingressclasses":
		out = appendUnique(out, schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "networking.k8s.io" && gvr.Resource == "networkpolicies":
		out = appendUnique(out, schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "discovery.k8s.io" && gvr.Resource == "endpointslices":
		out = appendUnique(out, schema.GroupVersionResource{Group: "discovery.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "discovery.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "storage.k8s.io" && gvr.Resource == "volumeattachments":
		out = appendUnique(out, schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "coordination.k8s.io" && gvr.Resource == "leases":
		out = appendUnique(out, schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "rbac.authorization.k8s.io" && gvr.Resource == "clusterroles":
		out = appendUnique(out, schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "apiextensions.k8s.io" && gvr.Resource == "customresourcedefinitions":
		out = appendUnique(out, schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "apiregistration.k8s.io" && gvr.Resource == "apiservices":
		out = appendUnique(out, schema.GroupVersionResource{Group: "apiregistration.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "apiregistration.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "scheduling.k8s.io" && gvr.Resource == "priorityclasses":
		out = appendUnique(out, schema.GroupVersionResource{Group: "scheduling.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "scheduling.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "admissionregistration.k8s.io" && gvr.Resource == "validatingwebhookconfigurations":
		out = appendUnique(out, schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "admissionregistration.k8s.io" && gvr.Resource == "mutatingwebhookconfigurations":
		out = appendUnique(out, schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "apps" && gvr.Resource == "replicasets":
		out = appendUnique(out, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "apps", Version: "v1beta2", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "apps", Version: "v1beta1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "snapshot.storage.k8s.io" && gvr.Resource == "volumesnapshots":
		out = appendUnique(out, schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "snapshot.storage.k8s.io" && gvr.Resource == "volumesnapshotclasses":
		out = appendUnique(out, schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	case gvr.Group == "snapshot.storage.k8s.io" && gvr.Resource == "volumesnapshotcontents":
		out = appendUnique(out, schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1", Resource: gvr.Resource})
		out = appendUnique(out, schema.GroupVersionResource{Group: "snapshot.storage.k8s.io", Version: "v1beta1", Resource: gvr.Resource})
	}

	return out
}

func isMissingAPIResourceErr(err error) bool {
	if err == nil {
		return false
	}
	if apierrors.IsNotFound(err) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "the server could not find the requested resource") ||
		strings.Contains(lower, "could not find the requested resource") ||
		strings.Contains(lower, "the server doesn't have a resource type") ||
		strings.Contains(lower, "unable to recognize") ||
		strings.Contains(lower, "no matches for kind")
}

type apiResourcesCachePayload struct {
	Missing   bool     `json:"missing,omitempty"`
	Resources []string `json:"resources,omitempty"`
}

type apiResourcesFlight struct {
	done      chan struct{}
	resources map[string]struct{}
	ok        bool
	err       error
}

func cloneStringSet(src map[string]struct{}) map[string]struct{} {
	if len(src) == 0 {
		if src == nil {
			return nil
		}
		return map[string]struct{}{}
	}
	dst := make(map[string]struct{}, len(src))
	for key := range src {
		dst[key] = struct{}{}
	}
	return dst
}

func clampTTL(v, minTTL, maxTTL time.Duration) time.Duration {
	if v <= 0 {
		v = minTTL
	}
	if v < minTTL {
		return minTTL
	}
	if maxTTL > 0 && v > maxTTL {
		return maxTTL
	}
	return v
}

func (s *K8sService) apiResourceSupportBaseTTL() time.Duration {
	if s == nil || s.podTTL <= 0 {
		return 20 * time.Second
	}
	return s.podTTL
}

func (s *K8sService) apiResourcesCacheTTL(gv schema.GroupVersion) time.Duration {
	base := s.apiResourceSupportBaseTTL()
	switch gv.Group {
	case "metrics.k8s.io":
		return clampTTL(base/2, 5*time.Second, 15*time.Second)
	case "snapshot.storage.k8s.io":
		return clampTTL(base*2, 30*time.Second, 2*time.Minute)
	default:
		return clampTTL(base*6, 2*time.Minute, 10*time.Minute)
	}
}

func (s *K8sService) apiResourcesCacheKey(clusterID uint64, gv schema.GroupVersion) string {
	group := strings.TrimSpace(gv.Group)
	if group == "" {
		group = "_core"
	}
	version := strings.TrimSpace(gv.Version)
	return fmt.Sprintf("k8s:apiresources:v2:cluster:%d:%s:%s", clusterID, group, version)
}

func (s *K8sService) apiResourcesForGroupVersion(ctx context.Context, clusterID uint64, gv schema.GroupVersion) (resources map[string]struct{}, ok bool, err error) {
	cacheEnabled := s != nil && s.cache != nil && s.cache.Enabled() && clusterID > 0
	cacheKey := ""
	if cacheEnabled {
		cacheKey = s.apiResourcesCacheKey(clusterID, gv)
		rctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		b, ok, err := s.cache.Get(rctx, cacheKey)
		cancel()
		if err == nil && ok && len(b) > 0 {
			var payload apiResourcesCachePayload
			if json.Unmarshal(b, &payload) == nil {
				if payload.Missing {
					return nil, false, nil
				}
				resources = make(map[string]struct{}, len(payload.Resources))
				for _, name := range payload.Resources {
					resourceName := strings.TrimSpace(name)
					if resourceName == "" {
						continue
					}
					resources[resourceName] = struct{}{}
				}
				return resources, true, nil
			}
		}
	}

	flightKey := cacheKey
	if strings.TrimSpace(flightKey) == "" {
		flightKey = s.apiResourcesCacheKey(clusterID, gv)
	}
	var flight *apiResourcesFlight
	if s != nil {
		s.apiResourcesMu.Lock()
		if s.apiResourcesInFlight == nil {
			s.apiResourcesInFlight = make(map[string]*apiResourcesFlight)
		}
		if existing := s.apiResourcesInFlight[flightKey]; existing != nil {
			s.apiResourcesMu.Unlock()
			select {
			case <-existing.done:
				return cloneStringSet(existing.resources), existing.ok, existing.err
			case <-ctx.Done():
				return nil, false, ctx.Err()
			}
		}
		flight = &apiResourcesFlight{done: make(chan struct{})}
		s.apiResourcesInFlight[flightKey] = flight
		s.apiResourcesMu.Unlock()
		defer func() {
			flight.resources = cloneStringSet(resources)
			flight.ok = ok
			flight.err = err
			close(flight.done)
			s.apiResourcesMu.Lock()
			delete(s.apiResourcesInFlight, flightKey)
			s.apiResourcesMu.Unlock()
		}()
	}

	dc, err := s.discoveryClient(ctx, clusterID)
	if err != nil {
		return nil, false, err
	}
	list, err := dc.ServerResourcesForGroupVersion(gv.String())
	if err != nil {
		if isMissingAPIResourceErr(err) {
			if cacheEnabled && cacheKey != "" {
				payload := apiResourcesCachePayload{Missing: true}
				if b, marshalErr := json.Marshal(payload); marshalErr == nil {
					wctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
					_ = s.cache.Set(wctx, cacheKey, b, s.apiResourcesCacheTTL(gv))
					cancel()
				}
			}
			return nil, false, nil
		}
		return nil, false, normalizeK8sErr(err)
	}

	resources = make(map[string]struct{}, len(list.APIResources))
	resourceNames := make([]string, 0, len(list.APIResources))
	for _, item := range list.APIResources {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		if _, exists := resources[name]; exists {
			continue
		}
		resources[name] = struct{}{}
		resourceNames = append(resourceNames, name)
	}
	sort.Strings(resourceNames)
	if cacheEnabled && cacheKey != "" {
		payload := apiResourcesCachePayload{Resources: resourceNames}
		if b, marshalErr := json.Marshal(payload); marshalErr == nil {
			wctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			_ = s.cache.Set(wctx, cacheKey, b, s.apiResourcesCacheTTL(gv))
			cancel()
		}
	}
	return resources, true, nil
}

func (s *K8sService) supportsGVR(ctx context.Context, clusterID uint64, gvr schema.GroupVersionResource) (bool, error) {
	gv := schema.GroupVersion{Group: gvr.Group, Version: gvr.Version}
	resources, ok, err := s.apiResourcesForGroupVersion(ctx, clusterID, gv)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	_, exists := resources[gvr.Resource]
	return exists, nil
}

func (s *K8sService) SupportsCompatibleGVR(ctx context.Context, clusterID uint64, gvr schema.GroupVersionResource) (bool, error) {
	for _, candidate := range candidateGVRs(gvr) {
		ok, err := s.supportsGVR(ctx, clusterID, candidate)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func (s *K8sService) resolveCompatibleGVR(ctx context.Context, clusterID uint64, gvr schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	candidates := candidateGVRs(gvr)
	if len(candidates) == 1 {
		return gvr, nil
	}
	for _, candidate := range candidates {
		ok, err := s.supportsGVR(ctx, clusterID, candidate)
		if err != nil {
			// Discovery API 出错时跳过该候选，继续尝试下一个版本
			continue
		}
		if ok {
			return candidate, nil
		}
	}
	return gvr, nil
}

// ---------------------------------------------------------------------------
// Generic CRUD operations
// ---------------------------------------------------------------------------

// List 列出指定 GVR 的资源列表。
func (s *K8sService) List(ctx context.Context, clusterID uint64, gvr schema.GroupVersionResource, namespace string, sortBy, order string, extraListOptions map[string]string) ([]any, error) {
	if isCoreV1PodsGVR(gvr) {
		ls := ""
		if extraListOptions != nil {
			ls = strings.TrimSpace(extraListOptions["label_selector"])
		}
		return kopsruntime.NewCachedConfigurationReader(s).ListWithLabelSelector(ctx, clusterID, kopsruntime.CachedPods, namespace, sortBy, order, ls)
	}
	if isAppsV1DeploymentsGVR(gvr) {
		ls := ""
		if extraListOptions != nil {
			ls = strings.TrimSpace(extraListOptions["label_selector"])
		}
		return kopsruntime.NewCachedConfigurationReader(s).ListWithLabelSelector(ctx, clusterID, kopsruntime.CachedDeployments, namespace, sortBy, order, ls)
	}
	if isAppsV1StatefulSetsGVR(gvr) {
		ls := ""
		if extraListOptions != nil {
			ls = strings.TrimSpace(extraListOptions["label_selector"])
		}
		return kopsruntime.NewCachedConfigurationReader(s).ListWithLabelSelector(ctx, clusterID, kopsruntime.CachedStatefulSets, namespace, sortBy, order, ls)
	}
	if isCoreV1ConfigMapsGVR(gvr) {
		return kopsruntime.NewCachedConfigurationReader(s).List(ctx, clusterID, kopsruntime.CachedConfigMaps, namespace, sortBy, order)
	}
	if isCoreV1SecretsGVR(gvr) {
		return kopsruntime.NewCachedConfigurationReader(s).List(ctx, clusterID, kopsruntime.CachedSecrets, namespace, sortBy, order)
	}
	if isCoreV1ServiceAccountsGVR(gvr) {
		return kopsruntime.NewCachedConfigurationReader(s).List(ctx, clusterID, kopsruntime.CachedServiceAccounts, namespace, sortBy, order)
	}
	if isAutoscalingV2HPAGVR(gvr) {
		return kopsruntime.NewCachedConfigurationReader(s).List(ctx, clusterID, kopsruntime.CachedHPAs, namespace, sortBy, order)
	}

	resolvedGVR, err := s.resolveCompatibleGVR(ctx, clusterID, gvr)
	if err != nil {
		return nil, err
	}
	gvr = resolvedGVR

	cacheEnabled := s != nil && s.cache != nil && s.cache.Enabled() && s.podTTL > 0 && clusterID > 0
	cacheKey := ""
	if cacheEnabled {
		cacheKey = s.k8sListCacheKey(clusterID, gvr, namespace, sortBy, order, extraListOptions)
		rctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		b, ok, err := s.cache.Get(rctx, cacheKey)
		cancel()
		if err == nil && ok && len(b) > 0 {
			var out []any
			if json.Unmarshal(b, &out) == nil {
				return out, nil
			}
		}
	}

	dc, err := s.dynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	var ri dynamic.ResourceInterface
	if namespace != "" {
		ri = dc.Resource(gvr).Namespace(namespace)
	} else {
		ri = dc.Resource(gvr)
	}
	opts := mapToListOptions(extraListOptions)
	opts.Limit = k8sListPageLimit
	var all []unstructured.Unstructured
	for {
		ul, err := ri.List(ctx, opts)
		if err != nil {
			return nil, normalizeK8sErr(err)
		}
		all = append(all, ul.Items...)
		token := ul.GetContinue()
		if strings.TrimSpace(token) == "" {
			break
		}
		opts.Continue = token
	}
	items := sortUnstructuredList(all, sortBy, order)
	out := unstructuredToAnyList(items)
	if cacheEnabled && cacheKey != "" {
		if payload, err := json.Marshal(out); err == nil && len(payload) > 0 {
			wctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			_ = s.cache.Set(wctx, cacheKey, payload, s.podTTL)
			cancel()
		}
	}
	return out, nil
}

func (s *K8sService) k8sListCacheKey(clusterID uint64, gvr schema.GroupVersionResource, namespace string, sortBy, order string, extraListOptions map[string]string) string {
	g := strings.TrimSpace(gvr.Group)
	if g == "" {
		g = "_core"
	}
	v := strings.TrimSpace(gvr.Version)
	r := strings.TrimSpace(gvr.Resource)

	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "_all"
	}
	ns = strings.ReplaceAll(ns, ":", "_")

	ls := ""
	fs := ""
	if extraListOptions != nil {
		ls = strings.TrimSpace(extraListOptions["label_selector"])
		fs = strings.TrimSpace(extraListOptions["field_selector"])
	}
	sb := strings.TrimSpace(sortBy)
	od := strings.ToLower(strings.TrimSpace(order))

	raw := "g=" + g + "|v=" + v + "|r=" + r + "|ns=" + ns + "|ls=" + ls + "|fs=" + fs + "|sb=" + sb + "|od=" + od
	sum := sha1.Sum([]byte(raw))
	return fmt.Sprintf("k8s:v1:cluster:%d:list:%s:%s:%s:%x", clusterID, g, v, r, sum[:])
}

// GetYAML 获取指定资源对象，并返回其 YAML 表示。
func (s *K8sService) GetYAML(ctx context.Context, clusterID uint64, gvr schema.GroupVersionResource, namespace, name string) (string, error) {
	resolvedGVR, err := s.resolveCompatibleGVR(ctx, clusterID, gvr)
	if err != nil {
		return "", err
	}
	gvr = resolvedGVR

	dc, err := s.dynamicClient(ctx, clusterID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(name) == "" {
		return "", ErrInvalidParams
	}
	var ri dynamic.ResourceInterface
	if namespace != "" {
		ri = dc.Resource(gvr).Namespace(namespace)
	} else {
		ri = dc.Resource(gvr)
	}
	obj, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", normalizeK8sErr(err)
	}
	return renderYAML(obj.Object)
}

func (s *K8sService) GetObject(ctx context.Context, clusterID uint64, gvr schema.GroupVersionResource, namespace, name string) (map[string]any, error) {
	resolvedGVR, err := s.resolveCompatibleGVR(ctx, clusterID, gvr)
	if err != nil {
		return nil, err
	}
	gvr = resolvedGVR

	dc, err := s.dynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrInvalidParams
	}
	var ri dynamic.ResourceInterface
	if namespace != "" {
		ri = dc.Resource(gvr).Namespace(namespace)
	} else {
		ri = dc.Resource(gvr)
	}
	obj, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, normalizeK8sErr(err)
	}
	return obj.Object, nil
}

// Delete 删除指定资源对象。
func (s *K8sService) Delete(ctx context.Context, clusterID uint64, gvr schema.GroupVersionResource, namespace, name string) error {
	resolvedGVR, err := s.resolveCompatibleGVR(ctx, clusterID, gvr)
	if err != nil {
		return err
	}
	gvr = resolvedGVR

	dc, err := s.dynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return ErrInvalidParams
	}
	var ri dynamic.ResourceInterface
	if namespace != "" {
		ri = dc.Resource(gvr).Namespace(namespace)
	} else {
		ri = dc.Resource(gvr)
	}
	return normalizeK8sErr(ri.Delete(ctx, name, metav1.DeleteOptions{}))
}

// ApplyYAML 应用 YAML 更新资源（使用 Update 操作）。
func (s *K8sService) ApplyYAML(ctx context.Context, clusterID uint64, gvr schema.GroupVersionResource, namespace, yamlContent string) error {
	resolvedGVR, err := s.resolveCompatibleGVR(ctx, clusterID, gvr)
	if err != nil {
		return err
	}
	gvr = resolvedGVR

	client, err := s.dynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}

	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(yamlContent), obj); err != nil {
		return fmt.Errorf("invalid yaml: %w", err)
	}

	if namespace != "" && obj.GetNamespace() != namespace {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(namespace)
		} else {
			return fmt.Errorf("namespace mismatch: url=%s, yaml=%s", namespace, obj.GetNamespace())
		}
	}
	name := obj.GetName()
	if name == "" {
		return fmt.Errorf("missing metadata.name in yaml")
	}

	var res dynamic.ResourceInterface
	if namespace != "" {
		res = client.Resource(gvr).Namespace(namespace)
	} else {
		res = client.Resource(gvr)
	}

	_, err = res.Update(ctx, obj, metav1.UpdateOptions{})
	return normalizeK8sErr(err)
}

// PatchJSON 对指定资源执行 MergePatch。
func (s *K8sService) PatchJSON(ctx context.Context, clusterID uint64, gvr schema.GroupVersionResource, namespace, name string, patch any) error {
	resolvedGVR, err := s.resolveCompatibleGVR(ctx, clusterID, gvr)
	if err != nil {
		return err
	}
	gvr = resolvedGVR

	dc, err := s.dynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return ErrInvalidParams
	}
	b, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	var ri dynamic.ResourceInterface
	if namespace != "" {
		ri = dc.Resource(gvr).Namespace(namespace)
	} else {
		ri = dc.Resource(gvr)
	}
	_, err = ri.Patch(ctx, name, types.MergePatchType, b, metav1.PatchOptions{})
	return normalizeK8sErr(err)
}

func (s *K8sService) StopClusterCaches(clusterID uint64) {
	if s == nil || clusterID == 0 {
		return
	}
	if s.podCache != nil {
		s.podCache.stop(clusterID)
	}
	if s.objCache != nil {
		s.objCache.stop(clusterID)
	}
}

// StopUnusedInformers 停止不在 activeIDs 集合中的所有 Informer 缓存。
// 用于集群删除后清理遗留缓存，防止内存泄漏。
func (s *K8sService) StopUnusedInformers(activeIDs map[uint64]bool) {
	if s == nil {
		return
	}
	if s.podCache != nil {
		for _, id := range s.podCache.cachedIDs() {
			if !activeIDs[id] {
				s.podCache.stop(id)
			}
		}
	}
	if s.objCache != nil {
		for _, id := range s.objCache.cachedIDs() {
			if !activeIDs[id] {
				s.objCache.stop(id)
			}
		}
	}
}
