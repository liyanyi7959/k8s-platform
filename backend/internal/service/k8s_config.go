package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

const maskedSecretValue = "***"

// ===========================================================================
// ConfigMaps
// ===========================================================================

func (s *K8sService) listConfigMapsCached(ctx context.Context, clusterID uint64, namespace string, sortBy, order string) ([]any, error) {
	entry, err := s.getOrStartObjCache(ctx, clusterID, "configmaps")
	if err != nil {
		return nil, err
	}

	cacheEnabled := s != nil && s.cache != nil && s.cache.Enabled() && s.podTTL > 0 && clusterID > 0
	if cacheEnabled {
		if out, ok := s.listConfigMapsFromRedisAll(ctx, clusterID, namespace, sortBy, order); ok {
			return out, nil
		}
	}

	if !entry.informer.HasSynced() {
		items, err := s.listConfigMapsDirectConfigMaps(ctx, clusterID, namespace)
		if err != nil {
			return nil, err
		}
		if cacheEnabled && strings.TrimSpace(namespace) == "" {
			s.setConfigMapsAllToRedis(clusterID, items)
		}
		sortConfigMaps(items, sortBy, order)
		return configMapsToAnyList(items), nil
	}

	items := filterConfigMapsFromInformer(entry.informer, namespace)
	if cacheEnabled && strings.TrimSpace(namespace) == "" {
		s.setConfigMapsAllToRedis(clusterID, items)
	}
	sortConfigMaps(items, sortBy, order)
	return configMapsToAnyList(items), nil
}

func configMapsFromInformerStore(informer cache.SharedIndexInformer) []*corev1.ConfigMap {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*corev1.ConfigMap, 0, len(objs))
	for _, obj := range objs {
		it, ok := obj.(*corev1.ConfigMap)
		if !ok || it == nil {
			continue
		}
		out = append(out, it.DeepCopy())
	}
	return out
}

func filterConfigMapsFromInformer(informer cache.SharedIndexInformer, namespace string) []*corev1.ConfigMap {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*corev1.ConfigMap, 0, len(objs))
	for _, obj := range objs {
		it, ok := obj.(*corev1.ConfigMap)
		if !ok || it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		out = append(out, it.DeepCopy())
	}
	return out
}

func configMapsToAnyList(items []*corev1.ConfigMap) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(it)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

func sortConfigMaps(items []*corev1.ConfigMap, sortBy, order string) {
	sb := strings.TrimSpace(sortBy)
	if sb == "" {
		return
	}
	desc := strings.ToLower(strings.TrimSpace(order)) == "desc"
	if sb == "metadata.name" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Name > items[j].Name
			}
			return items[i].Name < items[j].Name
		})
		return
	}
	if sb == "metadata.namespace" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Namespace > items[j].Namespace
			}
			return items[i].Namespace < items[j].Namespace
		})
	}
}

func (s *K8sService) listConfigMapsFromRedisAll(ctx context.Context, clusterID uint64, namespace string, sortBy, order string) ([]any, bool) {
	items, ok := s.configMapsFromRedisKey(ctx, s.objAllCacheKey(clusterID, "configmaps"))
	if !ok {
		return nil, false
	}
	filtered := make([]*corev1.ConfigMap, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		filtered = append(filtered, it.DeepCopy())
	}
	sortConfigMaps(filtered, sortBy, order)
	return configMapsToAnyList(filtered), true
}

func (s *K8sService) configMapsFromRedisKey(ctx context.Context, key string) ([]*corev1.ConfigMap, bool) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 {
		return nil, false
	}
	k := strings.TrimSpace(key)
	if k == "" {
		return nil, false
	}
	rctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	b, ok, err := s.cache.Get(rctx, k)
	cancel()
	if err != nil || !ok || b == nil {
		return nil, false
	}
	var arr []corev1.ConfigMap
	if json.Unmarshal(b, &arr) != nil {
		return nil, false
	}
	out := make([]*corev1.ConfigMap, 0, len(arr))
	for i := range arr {
		out = append(out, arr[i].DeepCopy())
	}
	return out, true
}

func (s *K8sService) setConfigMapsAllToRedis(clusterID uint64, items []*corev1.ConfigMap) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 || clusterID == 0 {
		return
	}
	arr := make([]corev1.ConfigMap, 0, len(items))
	for _, it := range items {
		if it != nil {
			arr = append(arr, *it)
		}
	}
	b, err := json.Marshal(arr)
	if err != nil || len(b) == 0 {
		return
	}
	ttl := s.podTTL
	wctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	_ = s.cache.Set(wctx, s.objAllCacheKey(clusterID, "configmaps"), b, ttl)
	cancel()
}

func (s *K8sService) listConfigMapsDirectConfigMaps(ctx context.Context, clusterID uint64, namespace string) ([]*corev1.ConfigMap, error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	opts := metav1.ListOptions{Limit: k8sListPageLimit}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	items := make([]*corev1.ConfigMap, 0, 128)
	for {
		ul, err := cs.CoreV1().ConfigMaps(ns).List(ctx, opts)
		if err != nil {
			return nil, normalizeK8sErr(err)
		}
		for i := range ul.Items {
			items = append(items, ul.Items[i].DeepCopy())
		}
		token := strings.TrimSpace(ul.Continue)
		if token == "" {
			break
		}
		opts.Continue = token
	}
	return items, nil
}

// ===========================================================================
// Secrets
// ===========================================================================

func (s *K8sService) listSecretsCached(ctx context.Context, clusterID uint64, namespace string, sortBy, order string) ([]any, error) {
	entry, err := s.getOrStartObjCache(ctx, clusterID, "secrets")
	if err != nil {
		return nil, err
	}

	cacheEnabled := s != nil && s.cache != nil && s.cache.Enabled() && s.podTTL > 0 && clusterID > 0
	if cacheEnabled {
		if out, ok := s.listSecretsFromRedisAll(ctx, clusterID, namespace, sortBy, order); ok {
			return out, nil
		}
	}

	if !entry.informer.HasSynced() {
		items, err := s.listSecretsDirectSecrets(ctx, clusterID, namespace)
		if err != nil {
			return nil, err
		}
		maskedItems := maskSecretsForList(items)
		if cacheEnabled && strings.TrimSpace(namespace) == "" {
			s.setSecretsAllToRedis(clusterID, maskedItems)
		}
		sortSecrets(maskedItems, sortBy, order)
		return secretsToAnyList(maskedItems), nil
	}

	items := filterSecretsFromInformer(entry.informer, namespace)
	maskedItems := maskSecretsForList(items)
	if cacheEnabled && strings.TrimSpace(namespace) == "" {
		s.setSecretsAllToRedis(clusterID, maskedItems)
	}
	sortSecrets(maskedItems, sortBy, order)
	return secretsToAnyList(maskedItems), nil
}

func maskSecretForList(secret *corev1.Secret) *corev1.Secret {
	if secret == nil {
		return nil
	}
	masked := secret.DeepCopy()
	if len(masked.Data) > 0 {
		data := make(map[string][]byte, len(masked.Data))
		for key := range masked.Data {
			data[key] = []byte(maskedSecretValue)
		}
		masked.Data = data
	}
	if len(masked.StringData) > 0 {
		stringData := make(map[string]string, len(masked.StringData))
		for key := range masked.StringData {
			stringData[key] = maskedSecretValue
		}
		masked.StringData = stringData
	}
	return masked
}

func maskSecretsForList(items []*corev1.Secret) []*corev1.Secret {
	out := make([]*corev1.Secret, 0, len(items))
	for _, item := range items {
		masked := maskSecretForList(item)
		if masked != nil {
			out = append(out, masked)
		}
	}
	return out
}

func secretsFromInformerStore(informer cache.SharedIndexInformer) []*corev1.Secret {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*corev1.Secret, 0, len(objs))
	for _, obj := range objs {
		it, ok := obj.(*corev1.Secret)
		if !ok || it == nil {
			continue
		}
		out = append(out, it.DeepCopy())
	}
	return out
}

func filterSecretsFromInformer(informer cache.SharedIndexInformer, namespace string) []*corev1.Secret {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*corev1.Secret, 0, len(objs))
	for _, obj := range objs {
		it, ok := obj.(*corev1.Secret)
		if !ok || it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		out = append(out, it.DeepCopy())
	}
	return out
}

func secretsToAnyList(items []*corev1.Secret) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(it)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

func sortSecrets(items []*corev1.Secret, sortBy, order string) {
	sb := strings.TrimSpace(sortBy)
	if sb == "" {
		return
	}
	desc := strings.ToLower(strings.TrimSpace(order)) == "desc"
	if sb == "metadata.name" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Name > items[j].Name
			}
			return items[i].Name < items[j].Name
		})
		return
	}
	if sb == "metadata.namespace" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Namespace > items[j].Namespace
			}
			return items[i].Namespace < items[j].Namespace
		})
	}
}

func (s *K8sService) listSecretsFromRedisAll(ctx context.Context, clusterID uint64, namespace string, sortBy, order string) ([]any, bool) {
	items, ok := s.secretsFromRedisKey(ctx, s.objAllCacheKey(clusterID, "secrets"))
	if !ok {
		return nil, false
	}
	filtered := make([]*corev1.Secret, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		filtered = append(filtered, maskSecretForList(it))
	}
	sortSecrets(filtered, sortBy, order)
	return secretsToAnyList(filtered), true
}

func (s *K8sService) secretsFromRedisKey(ctx context.Context, key string) ([]*corev1.Secret, bool) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 {
		return nil, false
	}
	k := strings.TrimSpace(key)
	if k == "" {
		return nil, false
	}
	rctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	b, ok, err := s.cache.Get(rctx, k)
	cancel()
	if err != nil || !ok || b == nil {
		return nil, false
	}
	var arr []corev1.Secret
	if json.Unmarshal(b, &arr) != nil {
		return nil, false
	}
	out := make([]*corev1.Secret, 0, len(arr))
	for i := range arr {
		out = append(out, arr[i].DeepCopy())
	}
	return out, true
}

func (s *K8sService) setSecretsAllToRedis(clusterID uint64, items []*corev1.Secret) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 || clusterID == 0 {
		return
	}
	arr := make([]corev1.Secret, 0, len(items))
	for _, it := range items {
		if it != nil {
			arr = append(arr, *it)
		}
	}
	b, err := json.Marshal(arr)
	if err != nil || len(b) == 0 {
		return
	}
	ttl := s.podTTL
	wctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	_ = s.cache.Set(wctx, s.objAllCacheKey(clusterID, "secrets"), b, ttl)
	cancel()
}

func (s *K8sService) listSecretsDirectSecrets(ctx context.Context, clusterID uint64, namespace string) ([]*corev1.Secret, error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	opts := metav1.ListOptions{Limit: k8sListPageLimit}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	items := make([]*corev1.Secret, 0, 128)
	for {
		ul, err := cs.CoreV1().Secrets(ns).List(ctx, opts)
		if err != nil {
			return nil, normalizeK8sErr(err)
		}
		for i := range ul.Items {
			items = append(items, ul.Items[i].DeepCopy())
		}
		token := strings.TrimSpace(ul.Continue)
		if token == "" {
			break
		}
		opts.Continue = token
	}
	return items, nil
}

// ===========================================================================
// ServiceAccounts
// ===========================================================================

func (s *K8sService) listServiceAccountsCached(ctx context.Context, clusterID uint64, namespace string, sortBy, order string) ([]any, error) {
	entry, err := s.getOrStartObjCache(ctx, clusterID, "serviceaccounts")
	if err != nil {
		return nil, err
	}

	cacheEnabled := s != nil && s.cache != nil && s.cache.Enabled() && s.podTTL > 0 && clusterID > 0
	if cacheEnabled {
		if out, ok := s.listServiceAccountsFromRedisAll(ctx, clusterID, namespace, sortBy, order); ok {
			return out, nil
		}
	}

	if !entry.informer.HasSynced() {
		items, err := s.listServiceAccountsDirect(ctx, clusterID, namespace)
		if err != nil {
			return nil, err
		}
		if cacheEnabled && strings.TrimSpace(namespace) == "" {
			s.setServiceAccountsAllToRedis(clusterID, items)
		}
		sortServiceAccounts(items, sortBy, order)
		return serviceAccountsToAnyList(items), nil
	}

	items := filterServiceAccountsFromInformer(entry.informer, namespace)
	if cacheEnabled && strings.TrimSpace(namespace) == "" {
		s.setServiceAccountsAllToRedis(clusterID, items)
	}
	sortServiceAccounts(items, sortBy, order)
	return serviceAccountsToAnyList(items), nil
}

func filterServiceAccountsFromInformer(informer cache.SharedIndexInformer, namespace string) []*corev1.ServiceAccount {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*corev1.ServiceAccount, 0, len(objs))
	for _, obj := range objs {
		it, ok := obj.(*corev1.ServiceAccount)
		if !ok || it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		out = append(out, it.DeepCopy())
	}
	return out
}

func serviceAccountsToAnyList(items []*corev1.ServiceAccount) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(it)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

func sortServiceAccounts(items []*corev1.ServiceAccount, sortBy, order string) {
	sb := strings.TrimSpace(sortBy)
	if sb == "" {
		return
	}
	desc := strings.ToLower(strings.TrimSpace(order)) == "desc"
	if sb == "metadata.name" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Name > items[j].Name
			}
			return items[i].Name < items[j].Name
		})
		return
	}
	if sb == "metadata.namespace" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Namespace > items[j].Namespace
			}
			return items[i].Namespace < items[j].Namespace
		})
	}
}

func (s *K8sService) listServiceAccountsFromRedisAll(ctx context.Context, clusterID uint64, namespace string, sortBy, order string) ([]any, bool) {
	items, ok := s.serviceAccountsFromRedisKey(ctx, s.objAllCacheKey(clusterID, "serviceaccounts"))
	if !ok {
		return nil, false
	}
	filtered := make([]*corev1.ServiceAccount, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		filtered = append(filtered, it.DeepCopy())
	}
	sortServiceAccounts(filtered, sortBy, order)
	return serviceAccountsToAnyList(filtered), true
}

func (s *K8sService) serviceAccountsFromRedisKey(ctx context.Context, key string) ([]*corev1.ServiceAccount, bool) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 {
		return nil, false
	}
	k := strings.TrimSpace(key)
	if k == "" {
		return nil, false
	}
	rctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	b, ok, err := s.cache.Get(rctx, k)
	cancel()
	if err != nil || !ok || b == nil {
		return nil, false
	}
	var arr []corev1.ServiceAccount
	if json.Unmarshal(b, &arr) != nil {
		return nil, false
	}
	out := make([]*corev1.ServiceAccount, 0, len(arr))
	for i := range arr {
		out = append(out, arr[i].DeepCopy())
	}
	return out, true
}

func (s *K8sService) setServiceAccountsAllToRedis(clusterID uint64, items []*corev1.ServiceAccount) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 || clusterID == 0 {
		return
	}
	arr := make([]corev1.ServiceAccount, 0, len(items))
	for _, it := range items {
		if it != nil {
			arr = append(arr, *it)
		}
	}
	b, err := json.Marshal(arr)
	if err != nil || len(b) == 0 {
		return
	}
	ttl := s.podTTL
	wctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	_ = s.cache.Set(wctx, s.objAllCacheKey(clusterID, "serviceaccounts"), b, ttl)
	cancel()
}

func (s *K8sService) listServiceAccountsDirect(ctx context.Context, clusterID uint64, namespace string) ([]*corev1.ServiceAccount, error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	opts := metav1.ListOptions{Limit: k8sListPageLimit}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	items := make([]*corev1.ServiceAccount, 0, 128)
	for {
		ul, err := cs.CoreV1().ServiceAccounts(ns).List(ctx, opts)
		if err != nil {
			return nil, normalizeK8sErr(err)
		}
		for i := range ul.Items {
			items = append(items, ul.Items[i].DeepCopy())
		}
		token := strings.TrimSpace(ul.Continue)
		if token == "" {
			break
		}
		opts.Continue = token
	}
	return items, nil
}

// ===========================================================================
// HPAs (HorizontalPodAutoscaler)
// ===========================================================================

func (s *K8sService) listHPAsCached(ctx context.Context, clusterID uint64, namespace string, sortBy, order string) ([]any, error) {
	entry, err := s.getOrStartObjCache(ctx, clusterID, "hpas")
	if err != nil {
		return nil, err
	}

	cacheEnabled := s != nil && s.cache != nil && s.cache.Enabled() && s.podTTL > 0 && clusterID > 0
	if cacheEnabled {
		if out, ok := s.listHPAsFromRedisAll(ctx, clusterID, namespace, sortBy, order); ok {
			return out, nil
		}
	}

	if !entry.informer.HasSynced() {
		items, err := s.listHPAsDirect(ctx, clusterID, namespace)
		if err != nil {
			return nil, err
		}
		if cacheEnabled && strings.TrimSpace(namespace) == "" {
			s.setHPAsAllToRedis(clusterID, items)
		}
		sortHPAs(items, sortBy, order)
		return hpasToAnyList(items), nil
	}

	items := filterHPAsFromInformer(entry.informer, namespace)
	if cacheEnabled && strings.TrimSpace(namespace) == "" {
		s.setHPAsAllToRedis(clusterID, items)
	}
	sortHPAs(items, sortBy, order)
	return hpasToAnyList(items), nil
}

func filterHPAsFromInformer(informer cache.SharedIndexInformer, namespace string) []*autoscalingv2.HorizontalPodAutoscaler {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*autoscalingv2.HorizontalPodAutoscaler, 0, len(objs))
	for _, obj := range objs {
		it, ok := obj.(*autoscalingv2.HorizontalPodAutoscaler)
		if !ok || it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		out = append(out, it.DeepCopy())
	}
	return out
}

func hpasToAnyList(items []*autoscalingv2.HorizontalPodAutoscaler) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(it)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

func sortHPAs(items []*autoscalingv2.HorizontalPodAutoscaler, sortBy, order string) {
	sb := strings.TrimSpace(sortBy)
	if sb == "" {
		return
	}
	desc := strings.ToLower(strings.TrimSpace(order)) == "desc"
	if sb == "metadata.name" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Name > items[j].Name
			}
			return items[i].Name < items[j].Name
		})
		return
	}
	if sb == "metadata.namespace" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Namespace > items[j].Namespace
			}
			return items[i].Namespace < items[j].Namespace
		})
	}
}

func (s *K8sService) listHPAsFromRedisAll(ctx context.Context, clusterID uint64, namespace string, sortBy, order string) ([]any, bool) {
	items, ok := s.hpasFromRedisKey(ctx, s.objAllCacheKey(clusterID, "hpas"))
	if !ok {
		return nil, false
	}
	filtered := make([]*autoscalingv2.HorizontalPodAutoscaler, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		filtered = append(filtered, it.DeepCopy())
	}
	sortHPAs(filtered, sortBy, order)
	return hpasToAnyList(filtered), true
}

func (s *K8sService) hpasFromRedisKey(ctx context.Context, key string) ([]*autoscalingv2.HorizontalPodAutoscaler, bool) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 {
		return nil, false
	}
	k := strings.TrimSpace(key)
	if k == "" {
		return nil, false
	}
	rctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	b, ok, err := s.cache.Get(rctx, k)
	cancel()
	if err != nil || !ok || b == nil {
		return nil, false
	}
	var arr []autoscalingv2.HorizontalPodAutoscaler
	if json.Unmarshal(b, &arr) != nil {
		return nil, false
	}
	out := make([]*autoscalingv2.HorizontalPodAutoscaler, 0, len(arr))
	for i := range arr {
		out = append(out, arr[i].DeepCopy())
	}
	return out, true
}

func (s *K8sService) setHPAsAllToRedis(clusterID uint64, items []*autoscalingv2.HorizontalPodAutoscaler) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 || clusterID == 0 {
		return
	}
	arr := make([]autoscalingv2.HorizontalPodAutoscaler, 0, len(items))
	for _, it := range items {
		if it != nil {
			arr = append(arr, *it)
		}
	}
	b, err := json.Marshal(arr)
	if err != nil || len(b) == 0 {
		return
	}
	ttl := s.podTTL
	wctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	_ = s.cache.Set(wctx, s.objAllCacheKey(clusterID, "hpas"), b, ttl)
	cancel()
}

func (s *K8sService) listHPAsDirect(ctx context.Context, clusterID uint64, namespace string) ([]*autoscalingv2.HorizontalPodAutoscaler, error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	opts := metav1.ListOptions{Limit: k8sListPageLimit}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	items := make([]*autoscalingv2.HorizontalPodAutoscaler, 0, 128)
	for {
		ul, err := cs.AutoscalingV2().HorizontalPodAutoscalers(ns).List(ctx, opts)
		if err != nil {
			return nil, normalizeK8sErr(err)
		}
		for i := range ul.Items {
			items = append(items, ul.Items[i].DeepCopy())
		}
		token := strings.TrimSpace(ul.Continue)
		if token == "" {
			break
		}
		opts.Continue = token
	}
	return items, nil
}
