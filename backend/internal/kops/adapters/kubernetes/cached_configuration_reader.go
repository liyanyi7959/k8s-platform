package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

const cachedConfigurationListPageLimit int64 = 500

type CachedConfigurationResource string

const (
	CachedConfigMaps      CachedConfigurationResource = "configmaps"
	CachedSecrets         CachedConfigurationResource = "secrets"
	CachedServiceAccounts CachedConfigurationResource = "serviceaccounts"
	CachedHPAs            CachedConfigurationResource = "hpas"
	CachedPods            CachedConfigurationResource = "pods"
	CachedDeployments     CachedConfigurationResource = "deployments"
	CachedStatefulSets    CachedConfigurationResource = "statefulsets"
)

// CachedConfigurationTransport is the narrow retained transport contract for
// typed configuration reads. It deliberately contains no Kops policy: the
// runtime owns cache order, Secret masking and direct-read fallbacks.
type CachedConfigurationTransport interface {
	TypedClient(context.Context, uint64) (*kubernetes.Clientset, error)
	ObjectInformer(context.Context, uint64, string) (cache.SharedIndexInformer, error)
	ObjectCacheEnabled() bool
	ObjectCacheTTL() time.Duration
	ObjectCacheGet(context.Context, string) ([]byte, bool, error)
	ObjectCacheSet(context.Context, string, []byte, time.Duration) error
	NormalizeKubernetesRuntimeError(error) error
}

// CachedConfigurationReader owns read-through cache behaviour for typed
// configuration resources. The supplied transport can remain backed by the
// compatibility K8sService while the business policy lives in Kops.
type CachedConfigurationReader struct{ transport CachedConfigurationTransport }

func NewCachedConfigurationReader(transport CachedConfigurationTransport) *CachedConfigurationReader {
	return &CachedConfigurationReader{transport: transport}
}

func (r *CachedConfigurationReader) List(ctx context.Context, clusterID uint64, resource CachedConfigurationResource, namespace, sortBy, order string) ([]any, error) {
	return r.ListWithLabelSelector(ctx, clusterID, resource, namespace, sortBy, order, "")
}

// ListWithLabelSelector covers the typed resources whose API shape makes an
// informer snapshot more reliable than generic dynamic-list caching. The
// Kops runtime owns selector validation and cache fallbacks; K8sService only
// supplies typed clients, informers and Redis transport.
func (r *CachedConfigurationReader) ListWithLabelSelector(ctx context.Context, clusterID uint64, resource CachedConfigurationResource, namespace, sortBy, order, labelSelector string) ([]any, error) {
	if r == nil || r.transport == nil || !validCachedConfigurationResource(resource) {
		return nil, kopsapp.ErrConflict
	}
	selector, err := parseCachedLabelSelector(labelSelector)
	if err != nil {
		return nil, err
	}
	cacheEnabled := r.cacheEnabled(clusterID)
	if cacheEnabled {
		if objects, ok := r.fromRedis(ctx, clusterID, resource); ok {
			return r.present(resource, filterCachedObjects(objects, namespace, selector), sortBy, order), nil
		}
	}

	informer, informerErr := r.transport.ObjectInformer(ctx, clusterID, string(resource))
	if informerErr != nil || informer == nil || !informer.HasSynced() {
		objects, err := r.listDirect(ctx, clusterID, resource, namespace, labelSelector)
		if err != nil {
			return nil, err
		}
		if cacheEnabled && strings.TrimSpace(namespace) == "" && strings.TrimSpace(labelSelector) == "" {
			r.toRedis(clusterID, resource, r.maskSecrets(objects))
		}
		return r.present(resource, objects, sortBy, order), nil
	}

	objects := filterCachedObjects(r.fromInformer(informer, resource), namespace, selector)
	if cacheEnabled && strings.TrimSpace(namespace) == "" && strings.TrimSpace(labelSelector) == "" {
		r.toRedis(clusterID, resource, r.maskSecrets(objects))
	}
	return r.present(resource, objects, sortBy, order), nil
}

func validCachedConfigurationResource(resource CachedConfigurationResource) bool {
	switch resource {
	case CachedConfigMaps, CachedSecrets, CachedServiceAccounts, CachedHPAs, CachedPods, CachedDeployments, CachedStatefulSets:
		return true
	default:
		return false
	}
}

func parseCachedLabelSelector(value string) (labels.Selector, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	selector, err := labels.Parse(value)
	if err != nil {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "label_selector 格式无效")
	}
	return selector, nil
}

func (r *CachedConfigurationReader) cacheEnabled(clusterID uint64) bool {
	return r != nil && r.transport != nil && r.transport.ObjectCacheEnabled() && r.transport.ObjectCacheTTL() > 0 && clusterID > 0
}

func (r *CachedConfigurationReader) fromRedis(ctx context.Context, clusterID uint64, resource CachedConfigurationResource) ([]k8sruntime.Object, bool) {
	readContext, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	payload, ok, err := r.transport.ObjectCacheGet(readContext, kopsapp.ObjectListCacheKey(clusterID, string(resource)))
	cancel()
	if err != nil || !ok || len(payload) == 0 {
		return nil, false
	}
	objects, err := unmarshalCachedConfigurationObjects(resource, payload)
	if err != nil {
		return nil, false
	}
	return objects, true
}

func (r *CachedConfigurationReader) toRedis(clusterID uint64, resource CachedConfigurationResource, objects []k8sruntime.Object) {
	if !r.cacheEnabled(clusterID) {
		return
	}
	payload, err := marshalCachedConfigurationObjects(resource, objects)
	if err != nil || len(payload) == 0 {
		return
	}
	writeContext, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	_ = r.transport.ObjectCacheSet(writeContext, kopsapp.ObjectListCacheKey(clusterID, string(resource)), payload, r.transport.ObjectCacheTTL())
	cancel()
}

func (r *CachedConfigurationReader) listDirect(ctx context.Context, clusterID uint64, resource CachedConfigurationResource, namespace, labelSelector string) ([]k8sruntime.Object, error) {
	client, err := r.transport.TypedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}
	options := metav1.ListOptions{Limit: cachedConfigurationListPageLimit, LabelSelector: strings.TrimSpace(labelSelector)}
	objects := make([]k8sruntime.Object, 0, 128)
	for {
		var next string
		switch resource {
		case CachedConfigMaps:
			list, listErr := client.CoreV1().ConfigMaps(namespace).List(ctx, options)
			if listErr != nil {
				return nil, r.normalize(listErr)
			}
			for index := range list.Items {
				objects = append(objects, list.Items[index].DeepCopy())
			}
			next = list.Continue
		case CachedSecrets:
			list, listErr := client.CoreV1().Secrets(namespace).List(ctx, options)
			if listErr != nil {
				return nil, r.normalize(listErr)
			}
			for index := range list.Items {
				objects = append(objects, list.Items[index].DeepCopy())
			}
			next = list.Continue
		case CachedServiceAccounts:
			list, listErr := client.CoreV1().ServiceAccounts(namespace).List(ctx, options)
			if listErr != nil {
				return nil, r.normalize(listErr)
			}
			for index := range list.Items {
				objects = append(objects, list.Items[index].DeepCopy())
			}
			next = list.Continue
		case CachedHPAs:
			list, listErr := client.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, options)
			if listErr != nil {
				return nil, r.normalize(listErr)
			}
			for index := range list.Items {
				objects = append(objects, list.Items[index].DeepCopy())
			}
			next = list.Continue
		case CachedPods:
			list, listErr := client.CoreV1().Pods(namespace).List(ctx, options)
			if listErr != nil {
				return nil, r.normalize(listErr)
			}
			for index := range list.Items {
				objects = append(objects, list.Items[index].DeepCopy())
			}
			next = list.Continue
		case CachedDeployments:
			list, listErr := client.AppsV1().Deployments(namespace).List(ctx, options)
			if listErr != nil {
				return nil, r.normalize(listErr)
			}
			for index := range list.Items {
				objects = append(objects, list.Items[index].DeepCopy())
			}
			next = list.Continue
		case CachedStatefulSets:
			list, listErr := client.AppsV1().StatefulSets(namespace).List(ctx, options)
			if listErr != nil {
				return nil, r.normalize(listErr)
			}
			for index := range list.Items {
				objects = append(objects, list.Items[index].DeepCopy())
			}
			next = list.Continue
		}
		if strings.TrimSpace(next) == "" {
			break
		}
		options.Continue = next
	}
	return objects, nil
}

func (r *CachedConfigurationReader) fromInformer(informer cache.SharedIndexInformer, resource CachedConfigurationResource) []k8sruntime.Object {
	if informer == nil {
		return nil
	}
	objects := make([]k8sruntime.Object, 0, len(informer.GetStore().List()))
	for _, item := range informer.GetStore().List() {
		switch resource {
		case CachedConfigMaps:
			if value, ok := item.(*corev1.ConfigMap); ok && value != nil {
				objects = append(objects, value.DeepCopy())
			}
		case CachedSecrets:
			if value, ok := item.(*corev1.Secret); ok && value != nil {
				objects = append(objects, value.DeepCopy())
			}
		case CachedServiceAccounts:
			if value, ok := item.(*corev1.ServiceAccount); ok && value != nil {
				objects = append(objects, value.DeepCopy())
			}
		case CachedHPAs:
			if value, ok := item.(*autoscalingv2.HorizontalPodAutoscaler); ok && value != nil {
				objects = append(objects, value.DeepCopy())
			}
		case CachedPods:
			if value, ok := item.(*corev1.Pod); ok && value != nil {
				objects = append(objects, value.DeepCopy())
			}
		case CachedDeployments:
			if value, ok := item.(*appsv1.Deployment); ok && value != nil {
				objects = append(objects, value.DeepCopy())
			}
		case CachedStatefulSets:
			if value, ok := item.(*appsv1.StatefulSet); ok && value != nil {
				objects = append(objects, value.DeepCopy())
			}
		}
	}
	return objects
}

func (r *CachedConfigurationReader) present(resource CachedConfigurationResource, objects []k8sruntime.Object, sortBy, order string) []any {
	objects = r.maskSecrets(objects)
	sortCachedConfigurationObjects(objects, sortBy, order)
	values := make([]any, 0, len(objects))
	for _, object := range objects {
		if object == nil {
			continue
		}
		value, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(object)
		if err == nil {
			values = append(values, value)
		}
	}
	return values
}

func (r *CachedConfigurationReader) maskSecrets(objects []k8sruntime.Object) []k8sruntime.Object {
	result := make([]k8sruntime.Object, 0, len(objects))
	for _, object := range objects {
		if secret, ok := object.(*corev1.Secret); ok {
			result = append(result, kopsapp.MaskSecretForRead(secret))
			continue
		}
		if object != nil {
			result = append(result, object)
		}
	}
	return result
}

func (r *CachedConfigurationReader) normalize(err error) error {
	if err == nil || r == nil || r.transport == nil {
		return err
	}
	return r.transport.NormalizeKubernetesRuntimeError(err)
}

func filterCachedObjects(objects []k8sruntime.Object, namespace string, selector labels.Selector) []k8sruntime.Object {
	namespace = strings.TrimSpace(namespace)
	result := make([]k8sruntime.Object, 0, len(objects))
	for _, object := range objects {
		metadata, ok := object.(metav1.Object)
		if !ok || (namespace != "" && metadata.GetNamespace() != namespace) {
			continue
		}
		if selector != nil && !selector.Matches(labels.Set(metadata.GetLabels())) {
			continue
		}
		if object != nil {
			result = append(result, object.DeepCopyObject())
		}
	}
	return result
}

func cloneCachedObjects(objects []k8sruntime.Object) []k8sruntime.Object {
	result := make([]k8sruntime.Object, 0, len(objects))
	for _, object := range objects {
		if object != nil {
			result = append(result, object.DeepCopyObject())
		}
	}
	return result
}

func sortCachedConfigurationObjects(objects []k8sruntime.Object, sortBy, order string) {
	sortBy = strings.TrimSpace(sortBy)
	if sortBy != "metadata.name" && sortBy != "metadata.namespace" {
		return
	}
	descending := strings.EqualFold(strings.TrimSpace(order), "desc")
	sort.SliceStable(objects, func(left, right int) bool {
		leftObject, leftOK := objects[left].(metav1.Object)
		rightObject, rightOK := objects[right].(metav1.Object)
		if !leftOK || !rightOK {
			return false
		}
		leftValue, rightValue := leftObject.GetName(), rightObject.GetName()
		if sortBy == "metadata.namespace" {
			leftValue, rightValue = leftObject.GetNamespace(), rightObject.GetNamespace()
		}
		if descending {
			return leftValue > rightValue
		}
		return leftValue < rightValue
	})
}

func marshalCachedConfigurationObjects(resource CachedConfigurationResource, objects []k8sruntime.Object) ([]byte, error) {
	switch resource {
	case CachedConfigMaps:
		values := make([]corev1.ConfigMap, 0, len(objects))
		for _, object := range objects {
			if value, ok := object.(*corev1.ConfigMap); ok && value != nil {
				values = append(values, *value.DeepCopy())
			}
		}
		return json.Marshal(values)
	case CachedSecrets:
		values := make([]corev1.Secret, 0, len(objects))
		for _, object := range objects {
			if value, ok := object.(*corev1.Secret); ok && value != nil {
				values = append(values, *value.DeepCopy())
			}
		}
		return json.Marshal(values)
	case CachedServiceAccounts:
		values := make([]corev1.ServiceAccount, 0, len(objects))
		for _, object := range objects {
			if value, ok := object.(*corev1.ServiceAccount); ok && value != nil {
				values = append(values, *value.DeepCopy())
			}
		}
		return json.Marshal(values)
	case CachedHPAs:
		values := make([]autoscalingv2.HorizontalPodAutoscaler, 0, len(objects))
		for _, object := range objects {
			if value, ok := object.(*autoscalingv2.HorizontalPodAutoscaler); ok && value != nil {
				values = append(values, *value.DeepCopy())
			}
		}
		return json.Marshal(values)
	case CachedPods:
		values := make([]corev1.Pod, 0, len(objects))
		for _, object := range objects {
			if value, ok := object.(*corev1.Pod); ok && value != nil {
				values = append(values, *value.DeepCopy())
			}
		}
		return json.Marshal(values)
	case CachedDeployments:
		values := make([]appsv1.Deployment, 0, len(objects))
		for _, object := range objects {
			if value, ok := object.(*appsv1.Deployment); ok && value != nil {
				values = append(values, *value.DeepCopy())
			}
		}
		return json.Marshal(values)
	case CachedStatefulSets:
		values := make([]appsv1.StatefulSet, 0, len(objects))
		for _, object := range objects {
			if value, ok := object.(*appsv1.StatefulSet); ok && value != nil {
				values = append(values, *value.DeepCopy())
			}
		}
		return json.Marshal(values)
	default:
		return nil, errors.New("unsupported cached configuration resource")
	}
}

func unmarshalCachedConfigurationObjects(resource CachedConfigurationResource, payload []byte) ([]k8sruntime.Object, error) {
	switch resource {
	case CachedConfigMaps:
		var values []corev1.ConfigMap
		if err := json.Unmarshal(payload, &values); err != nil {
			return nil, err
		}
		objects := make([]k8sruntime.Object, 0, len(values))
		for index := range values {
			objects = append(objects, values[index].DeepCopy())
		}
		return objects, nil
	case CachedSecrets:
		var values []corev1.Secret
		if err := json.Unmarshal(payload, &values); err != nil {
			return nil, err
		}
		objects := make([]k8sruntime.Object, 0, len(values))
		for index := range values {
			objects = append(objects, values[index].DeepCopy())
		}
		return objects, nil
	case CachedServiceAccounts:
		var values []corev1.ServiceAccount
		if err := json.Unmarshal(payload, &values); err != nil {
			return nil, err
		}
		objects := make([]k8sruntime.Object, 0, len(values))
		for index := range values {
			objects = append(objects, values[index].DeepCopy())
		}
		return objects, nil
	case CachedHPAs:
		var values []autoscalingv2.HorizontalPodAutoscaler
		if err := json.Unmarshal(payload, &values); err != nil {
			return nil, err
		}
		objects := make([]k8sruntime.Object, 0, len(values))
		for index := range values {
			objects = append(objects, values[index].DeepCopy())
		}
		return objects, nil
	case CachedPods:
		var values []corev1.Pod
		if err := json.Unmarshal(payload, &values); err != nil {
			return nil, err
		}
		objects := make([]k8sruntime.Object, 0, len(values))
		for index := range values {
			objects = append(objects, values[index].DeepCopy())
		}
		return objects, nil
	case CachedDeployments:
		var values []appsv1.Deployment
		if err := json.Unmarshal(payload, &values); err != nil {
			return nil, err
		}
		objects := make([]k8sruntime.Object, 0, len(values))
		for index := range values {
			objects = append(objects, values[index].DeepCopy())
		}
		return objects, nil
	case CachedStatefulSets:
		var values []appsv1.StatefulSet
		if err := json.Unmarshal(payload, &values); err != nil {
			return nil, err
		}
		objects := make([]k8sruntime.Object, 0, len(values))
		for index := range values {
			objects = append(objects, values[index].DeepCopy())
		}
		return objects, nil
	default:
		return nil, errors.New("unsupported cached configuration resource")
	}
}
