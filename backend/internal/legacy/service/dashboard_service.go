package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	fleetapp "k8s-platform-backend/internal/fleet/application"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type cachedOverview struct {
	Data      map[string]any `json:"data"`
	ExpiresAt time.Time      `json:"expires_at"`
}

type trendDataCache struct {
	Samples []resourceTrendSample `json:"samples"`
}

type resourceTrendSample struct {
	Timestamp time.Time `json:"timestamp"`
	CPU       float64   `json:"cpu"`
	Memory    float64   `json:"memory"`
}

type cachedClusterCertRisks struct {
	Data      []map[string]any `json:"data"`
	ExpiresAt time.Time        `json:"expires_at"`
}

func cloneAnyMapSlice(items []map[string]any) []map[string]any {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item == nil {
			cloned = append(cloned, nil)
			continue
		}
		dup := make(map[string]any, len(item))
		for key, value := range item {
			dup[key] = value
		}
		cloned = append(cloned, dup)
	}
	return cloned
}

type DashboardService struct {
	db                 *gorm.DB
	clusterReg         *fleetapp.Registry
	k8sSvc             *K8sService
	cache              CacheStore
	certRiskMu         sync.Mutex
	certRiskCache      map[uint64]cachedClusterCertRisks
	certRiskRefreshing map[uint64]bool
	overviewGroup      singleflight.Group
	trendMu            sync.Mutex
	trendCache         map[string][]resourceTrendSample
}

// NewDashboardService 创建 DashboardService。
// DashboardService 负责聚合数据库与 Kubernetes 的统计数据，为前端仪表盘提供“概览”接口。
func NewDashboardService(db *gorm.DB, clusterReg *fleetapp.Registry, k8sSvc *K8sService, cache CacheStore) *DashboardService {
	return &DashboardService{
		db:                 db,
		clusterReg:         clusterReg,
		k8sSvc:             k8sSvc,
		cache:              cache,
		certRiskCache:      make(map[uint64]cachedClusterCertRisks),
		certRiskRefreshing: make(map[uint64]bool),
		trendCache:         make(map[string][]resourceTrendSample),
	}
}

func appendResourceTrendSample(samples []resourceTrendSample, sample resourceTrendSample) []resourceTrendSample {
	cutoff := sample.Timestamp.Add(-24 * time.Hour)
	filtered := make([]resourceTrendSample, 0, len(samples)+1)
	for _, existing := range samples {
		if existing.Timestamp.Before(cutoff) || existing.Timestamp.After(sample.Timestamp.Add(5*time.Minute)) {
			continue
		}
		if existing.CPU < 0 || existing.CPU > 100 || existing.Memory < 0 || existing.Memory > 100 {
			continue
		}
		filtered = append(filtered, existing)
	}
	if len(filtered) > 0 && sample.Timestamp.Sub(filtered[len(filtered)-1].Timestamp) < time.Minute {
		filtered[len(filtered)-1] = sample
	} else {
		filtered = append(filtered, sample)
	}
	return filtered
}

func (s *DashboardService) recordResourceTrend(ctx context.Context, clusterID uint64, scope string, sample resourceTrendSample) []resourceTrendSample {
	s.trendMu.Lock()
	defer s.trendMu.Unlock()

	trendID := fmt.Sprintf("%d:%s", clusterID, scope)
	samples := append([]resourceTrendSample(nil), s.trendCache[trendID]...)
	cacheKey := fmt.Sprintf("dashboard:trend:v3:%d:%s", clusterID, scope)
	if len(samples) == 0 && s.cache != nil && s.cache.Enabled() {
		readCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
		value, ok, _ := s.cache.Get(readCtx, cacheKey)
		cancel()
		if ok {
			var cached trendDataCache
			if json.Unmarshal(value, &cached) == nil {
				samples = cached.Samples
			}
		}
	}
	samples = appendResourceTrendSample(samples, sample)
	s.trendCache[trendID] = append([]resourceTrendSample(nil), samples...)
	if s.cache != nil && s.cache.Enabled() {
		if value, err := json.Marshal(trendDataCache{Samples: samples}); err == nil {
			writeCtx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			_ = s.cache.Set(writeCtx, cacheKey, value, 25*time.Hour)
			cancel()
		}
	}
	return samples
}

func clusterCertRiskCacheKey(clusterID uint64) string {
	return fmt.Sprintf("dashboard:certificate-risks:v1:cluster:%d", clusterID)
}

func (s *DashboardService) getCachedClusterCertRisks(ctx context.Context, clusterID uint64) ([]map[string]any, bool) {
	if s == nil || clusterID == 0 {
		return nil, false
	}
	s.certRiskMu.Lock()
	entry, ok := s.certRiskCache[clusterID]
	if ok && time.Now().After(entry.ExpiresAt) {
		delete(s.certRiskCache, clusterID)
		ok = false
	}
	s.certRiskMu.Unlock()
	if ok {
		return cloneAnyMapSlice(entry.Data), true
	}

	if s.cache == nil || !s.cache.Enabled() {
		return nil, false
	}
	readCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	value, found, err := s.cache.Get(readCtx, clusterCertRiskCacheKey(clusterID))
	cancel()
	if err != nil || !found || len(value) == 0 {
		return nil, false
	}
	var persisted cachedClusterCertRisks
	if json.Unmarshal(value, &persisted) != nil || time.Now().After(persisted.ExpiresAt) {
		return nil, false
	}
	s.certRiskMu.Lock()
	s.certRiskCache[clusterID] = persisted
	s.certRiskMu.Unlock()
	return cloneAnyMapSlice(persisted.Data), true
}

func (s *DashboardService) setCachedClusterCertRisks(ctx context.Context, clusterID uint64, data []map[string]any, ttl time.Duration) {
	if s == nil || clusterID == 0 {
		return
	}
	entry := cachedClusterCertRisks{
		Data:      cloneAnyMapSlice(data),
		ExpiresAt: time.Now().Add(ttl),
	}
	s.certRiskMu.Lock()
	s.certRiskCache[clusterID] = entry
	s.certRiskMu.Unlock()

	if s.cache != nil && s.cache.Enabled() {
		if value, err := json.Marshal(entry); err == nil {
			writeCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			_ = s.cache.Set(writeCtx, clusterCertRiskCacheKey(clusterID), value, ttl)
			cancel()
		}
	}
}

func (s *DashboardService) refreshClusterCertRisksAsync(clusterID uint64, apiOK bool) {
	if s == nil || clusterID == 0 || !apiOK {
		return
	}
	s.certRiskMu.Lock()
	if s.certRiskRefreshing[clusterID] {
		s.certRiskMu.Unlock()
		return
	}
	s.certRiskRefreshing[clusterID] = true
	s.certRiskMu.Unlock()

	go func() {
		defer func() {
			s.certRiskMu.Lock()
			delete(s.certRiskRefreshing, clusterID)
			s.certRiskMu.Unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		risks, cacheable := s.getClusterCertificateRisks(ctx, clusterID, apiOK)
		if cacheable {
			s.setCachedClusterCertRisks(ctx, clusterID, risks, 10*time.Minute)
		}
	}()
}

// GetClusterOverview 获取指定集群的概览数据。
// 聚合信息包含：
// - 集群基本信息（名称/状态）
// - 节点健康（ready/total）
// - Pod 相位统计与按命名空间聚合
// - 典型工作负载数量（deployments/statefulsets/daemonsets）
// - CPU/内存使用率（基于 allocatable 与 metrics.k8s.io 的节点 usage）
// GetClusterOverview merges concurrent cold requests for the same cluster so
// the Kubernetes API is queried once while the overview cache is empty.
func (s *DashboardService) GetClusterOverview(ctx context.Context, clusterID uint64) (map[string]any, error) {
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}
	key := fmt.Sprintf("cluster:%d", clusterID)
	value, err, _ := s.overviewGroup.Do(key, func() (any, error) {
		return s.getClusterOverview(ctx, clusterID)
	})
	if err != nil {
		return nil, err
	}
	overview, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("invalid overview result")
	}
	return overview, nil
}

func (s *DashboardService) getClusterOverview(ctx context.Context, clusterID uint64) (map[string]any, error) {
	cacheKey := fmt.Sprintf("dashboard:overview:cluster:%d", clusterID)
	if s.cache != nil && s.cache.Enabled() {
		readCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
		val, ok, _ := s.cache.Get(readCtx, cacheKey)
		cancel()
		if ok {
			var co cachedOverview
			if err := json.Unmarshal(val, &co); err == nil {
				if time.Now().Before(co.ExpiresAt) {
					return co.Data, nil
				}
			}
		}
	}

	if s.clusterReg == nil || s.k8sSvc == nil {
		return nil, errors.New("dependency missing")
	}
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}

	cluster, err := s.clusterReg.Get(ctx, clusterID)
	if err != nil {
		return nil, legacyClusterError(err)
	}

	ready, total := 0, 0
	apiOK := false
	k8sVersion := ""
	podsTotal, podsRunning, podsPending, podsFailed, podsSucceeded := 0, 0, 0, 0, 0
	nsPods := map[string]int{}
	var unscheduledPods []map[string]any

	var (
		deployments      []any
		statefulsets     []any
		daemonsets       []any
		cpuUsedPercent   float64
		memUsedPercent   float64
		metricsAvailable bool
		nodeUsage        []nodeUsagePercent
		failedPods       []map[string]any
		nodeItems        []corev1.Node // shared between health check & usage calc — used below in post-process
		recentEvents     []map[string]any
		wg               sync.WaitGroup
		nodeItemsReady   = make(chan struct{})
		mu               sync.Mutex // protects ready, total, nodeItems
	)

	// 0. Health check + node list — single call, reused by usage calc
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(nodeItemsReady)
		cs, err := s.k8sSvc.typedClient(ctx, clusterID)
		if err != nil {
			return
		}
		var (
			version    string
			versionErr error
			nodes      *corev1.NodeList
			nodesErr   error
			lookupWG   sync.WaitGroup
		)
		lookupWG.Add(2)
		go func() {
			defer lookupWG.Done()
			info, lookupErr := cs.Discovery().ServerVersion()
			versionErr = lookupErr
			if info != nil {
				version = strings.TrimSpace(info.GitVersion)
			}
		}()
		go func() {
			defer lookupWG.Done()
			nodes, nodesErr = cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		}()
		lookupWG.Wait()
		if nodesErr != nil || nodes == nil {
			mu.Lock()
			apiOK = versionErr == nil
			k8sVersion = version
			total = 0
			ready = 0
			mu.Unlock()
			return
		}
		r := 0
		for i := range nodes.Items {
			if isNodeReady(&nodes.Items[i]) {
				r++
			}
		}
		mu.Lock()
		apiOK = true
		k8sVersion = version
		nodeItems = nodes.Items
		total = len(nodes.Items)
		ready = r
		mu.Unlock()
	}()

	// 1. Pods — 优先使用 Informer 缓存，回退到 API List
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 尝试从 Informer 缓存读取 Pod（避免全量 List 遍历）
		if s.k8sSvc != nil && s.k8sSvc.podCache != nil {
			entry, err := s.k8sSvc.getOrStartPodCache(ctx, clusterID)
			if err == nil && entry != nil && entry.synced.Load() && entry.informer != nil {
				items := entry.informer.GetStore().List()
				podsTotal = len(items)
				for _, obj := range items {
					p, ok := obj.(*corev1.Pod)
					if !ok || p == nil {
						continue
					}
					switch p.Status.Phase {
					case corev1.PodRunning:
						podsRunning++
					case corev1.PodPending:
						podsPending++
						if p.Spec.NodeName == "" && len(unscheduledPods) < 10 {
							unscheduledPods = append(unscheduledPods, map[string]any{
								"name": p.Name, "namespace": p.Namespace, "reason": "Unschedulable",
							})
						}
					case corev1.PodFailed:
						podsFailed++
						if len(failedPods) < 10 {
							reason := string(p.Status.Phase)
							for _, cs := range p.Status.ContainerStatuses {
								if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
									reason = cs.State.Waiting.Reason
									break
								}
								if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
									reason = cs.State.Terminated.Reason
									break
								}
							}
							failedPods = append(failedPods, map[string]any{
								"name": p.Name, "namespace": p.Namespace, "reason": reason,
							})
						}
					case corev1.PodSucceeded:
						podsSucceeded++
					}
					ns := p.Namespace
					if ns != "" {
						nsPods[ns]++
					}
					if p.Status.Phase != corev1.PodFailed {
						if reason := unhealthyPodReason(p); reason != "" && len(failedPods) < 10 {
							failedPods = append(failedPods, map[string]any{"name": p.Name, "namespace": p.Namespace, "reason": reason})
						}
					}
				}
				return
			}
		}
		// 回退：使用分页 API List（兼容未启动 Informer 场景）
		cs, err := s.k8sSvc.typedClient(ctx, clusterID)
		if err != nil {
			return
		}
		opts := metav1.ListOptions{Limit: k8sListPageLimit}
		for {
			pods, err := cs.CoreV1().Pods("").List(ctx, opts)
			if err != nil {
				break
			}
			podsTotal += len(pods.Items)
			for i := range pods.Items {
				p := pods.Items[i]
				switch p.Status.Phase {
				case corev1.PodRunning:
					podsRunning++
				case corev1.PodPending:
					podsPending++
					if p.Spec.NodeName == "" && len(unscheduledPods) < 10 {
						unscheduledPods = append(unscheduledPods, map[string]any{
							"name": p.Name, "namespace": p.Namespace, "reason": "Unschedulable",
						})
					}
				case corev1.PodFailed:
					podsFailed++
					if len(failedPods) < 10 {
						reason := string(p.Status.Phase)
						for _, cs := range p.Status.ContainerStatuses {
							if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
								reason = cs.State.Waiting.Reason
								break
							}
							if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
								reason = cs.State.Terminated.Reason
								break
							}
						}
						failedPods = append(failedPods, map[string]any{
							"name": p.Name, "namespace": p.Namespace, "reason": reason,
						})
					}
				case corev1.PodSucceeded:
					podsSucceeded++
				}
				ns := p.Namespace
				if ns != "" {
					nsPods[ns]++
				}
				if p.Status.Phase != corev1.PodFailed {
					if reason := unhealthyPodReason(&p); reason != "" && len(failedPods) < 10 {
						failedPods = append(failedPods, map[string]any{"name": p.Name, "namespace": p.Namespace, "reason": reason})
					}
				}
			}
			token := strings.TrimSpace(pods.Continue)
			if token == "" {
				break
			}
			opts.Continue = token
		}
	}()

	// 2. Workloads — 优先使用 Informer 缓存
	wg.Add(3)
	go func() {
		defer wg.Done()
		if s.k8sSvc != nil && s.k8sSvc.objCache != nil {
			entry, err := s.k8sSvc.getOrStartObjCache(ctx, clusterID, "deployments")
			if err == nil && entry != nil && entry.synced.Load() && entry.informer != nil {
				items := deploymentsFromInformerStore(entry.informer)
				result := make([]any, 0, len(items))
				for _, it := range items {
					if it != nil {
						result = append(result, it)
					}
				}
				deployments = result
				return
			}
		}
		deployments, _ = s.k8sSvc.List(ctx, clusterID, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, "", "", "", nil)
	}()
	go func() {
		defer wg.Done()
		if s.k8sSvc != nil && s.k8sSvc.objCache != nil {
			entry, err := s.k8sSvc.getOrStartObjCache(ctx, clusterID, "statefulsets")
			if err == nil && entry != nil && entry.synced.Load() && entry.informer != nil {
				items := statefulSetsFromInformerStore(entry.informer)
				result := make([]any, 0, len(items))
				for _, it := range items {
					if it != nil {
						result = append(result, it)
					}
				}
				statefulsets = result
				return
			}
		}
		statefulsets, _ = s.k8sSvc.List(ctx, clusterID, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, "", "", "", nil)
	}()
	go func() {
		defer wg.Done()
		daemonsets, _ = s.k8sSvc.List(ctx, clusterID, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, "", "", "", nil)
	}()

	// 2.5. Recent Events — 拉取最近 1 小时内的 Warning 事件
	wg.Add(1)
	go func() {
		defer wg.Done()
		cs, err := s.k8sSvc.typedClient(ctx, clusterID)
		if err != nil {
			return
		}
		evList, err := cs.CoreV1().Events("").List(ctx, metav1.ListOptions{
			FieldSelector: "type=Warning",
			Limit:         50,
		})
		if err != nil {
			return
		}
		for i := range evList.Items {
			if len(recentEvents) >= 20 {
				break
			}
			ev := &evList.Items[i]
			// 过滤最近 1 小时内的事件
			ts := ev.LastTimestamp.Time
			if ts.IsZero() {
				ts = ev.EventTime.Time
			}
			if ts.IsZero() {
				continue
			}
			if time.Since(ts) > time.Hour {
				continue
			}
			recentEvents = append(recentEvents, map[string]any{
				"type":           ev.Type,
				"reason":         ev.Reason,
				"message":        ev.Message,
				"namespace":      ev.Namespace,
				"lastTimestamp":  ts.Format(time.RFC3339),
				"count":          ev.Count,
				"involvedObject": fmt.Sprintf("%s/%s", ev.InvolvedObject.Kind, ev.InvolvedObject.Name),
			})
		}
	}()

	// 3. Usage — reuse nodeItems from health check goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-nodeItemsReady
		mu.Lock()
		nodes := append([]corev1.Node(nil), nodeItems...)
		mu.Unlock()
		usage, available := s.getClusterUsageSnapshot(ctx, clusterID, nodes)
		cpuUsedPercent = usage.CPU
		memUsedPercent = usage.Memory
		nodeUsage = usage.Nodes
		metricsAvailable = available
	}()

	wg.Wait()

	// Post-process: Namespace Top Pods
	type nsCount struct {
		Name  string
		Count int
	}
	var nsList []nsCount
	for ns, c := range nsPods {
		nsList = append(nsList, nsCount{Name: ns, Count: c})
	}
	sort.Slice(nsList, func(i, j int) bool {
		return nsList[i].Count > nsList[j].Count
	})
	if len(nsList) > 10 {
		nsList = nsList[:10]
	}
	nsTop := make([]map[string]any, 0, len(nsList))
	for _, v := range nsList {
		nsTop = append(nsTop, map[string]any{"namespace": v.Name, "pods": v.Count})
	}

	// Post-process: 24h Trends. Only real metrics-server samples are retained.
	labels24h := make([]string, 0)
	timestamps24h := make([]string, 0)
	cpu24 := make([]float64, 0)
	mem24 := make([]float64, 0)
	nodeTrends24h := make([]map[string]any, 0, len(nodeItems))
	nodeUsageByName := make(map[string]nodeUsagePercent, len(nodeUsage))
	for _, usage := range nodeUsage {
		nodeUsageByName[usage.Name] = usage
	}
	if metricsAvailable {
		sampledAt := time.Now()
		samples := s.recordResourceTrend(ctx, clusterID, "cluster", resourceTrendSample{
			Timestamp: sampledAt,
			CPU:       cpuUsedPercent,
			Memory:    memUsedPercent,
		})
		labels24h = make([]string, 0, len(samples))
		timestamps24h = make([]string, 0, len(samples))
		cpu24 = make([]float64, 0, len(samples))
		mem24 = make([]float64, 0, len(samples))
		for _, sample := range samples {
			labels24h = append(labels24h, sample.Timestamp.Format("01-02 15:04"))
			timestamps24h = append(timestamps24h, sample.Timestamp.Format(time.RFC3339))
			cpu24 = append(cpu24, sample.CPU)
			mem24 = append(mem24, sample.Memory)
		}
		for _, node := range nodeItems {
			usage, ok := nodeUsageByName[node.Name]
			if !ok {
				continue
			}
			nodeSamples := s.recordResourceTrend(ctx, clusterID, "node:"+node.Name, resourceTrendSample{
				Timestamp: sampledAt,
				CPU:       usage.CPU,
				Memory:    usage.Memory,
			})
			nodeLabels := make([]string, 0, len(nodeSamples))
			nodeTimestamps := make([]string, 0, len(nodeSamples))
			nodeCPU := make([]float64, 0, len(nodeSamples))
			nodeMemory := make([]float64, 0, len(nodeSamples))
			for _, sample := range nodeSamples {
				nodeLabels = append(nodeLabels, sample.Timestamp.Format("01-02 15:04"))
				nodeTimestamps = append(nodeTimestamps, sample.Timestamp.Format(time.RFC3339))
				nodeCPU = append(nodeCPU, sample.CPU)
				nodeMemory = append(nodeMemory, sample.Memory)
			}
			nodeTrends24h = append(nodeTrends24h, map[string]any{
				"name": node.Name, "ip": usage.IP,
				"labels": nodeLabels, "timestamps": nodeTimestamps,
				"cpu": nodeCPU, "memory": nodeMemory, "sample_count": len(nodeLabels),
			})
		}
	}
	sort.Slice(nodeTrends24h, func(i, j int) bool {
		return nodeTrends24h[i]["name"].(string) < nodeTrends24h[j]["name"].(string)
	})

	out := map[string]any{
		"cluster": map[string]any{
			"id":          cluster.ID,
			"name":        cluster.Name,
			"status":      cluster.Status,
			"api_ok":      apiOK,
			"k8s_version": k8sVersion,
		},
		"stats": map[string]any{
			"nodes": map[string]any{"total": total, "ready": ready},
			"pods": map[string]any{
				"total":     podsTotal,
				"running":   podsRunning,
				"pending":   podsPending,
				"failed":    podsFailed,
				"succeeded": podsSucceeded,
			},
			"workloads": map[string]any{
				"deployments":  len(deployments),
				"statefulsets": len(statefulsets),
				"daemonsets":   len(daemonsets),
			},
			"cpu":    map[string]any{"used_percent": cpuUsedPercent},
			"memory": map[string]any{"used_percent": memUsedPercent},
		},
		"charts": map[string]any{
			"cpu_memory_24h": map[string]any{
				"labels": labels24h, "timestamps": timestamps24h, "cpu": cpu24, "memory": mem24,
				"scope": "cluster", "sample_count": len(labels24h),
			},
			"node_cpu_memory_24h": nodeTrends24h,
			"pod_phase": map[string]any{
				"running":   podsRunning,
				"pending":   podsPending,
				"failed":    podsFailed,
				"succeeded": podsSucceeded,
			},
			"namespace_pods_top": nsTop,
			"node_ready":         map[string]any{"ready": ready, "total": total},
		},
		"anomalies": map[string]any{
			"failed_pods":      failedPods,
			"unscheduled_pods": unscheduledPods,
		},
		"events": recentEvents,
		// 数据溯源：标注采集来源与更新时间
		"meta": map[string]any{
			"source":            "k8s-api",
			"updated_at":        time.Now().Format(time.RFC3339),
			"cached":            false,
			"metrics_available": metricsAvailable,
			"metrics_source":    "metrics.k8s.io/v1beta1 nodes",
			"metrics_scope":     "cluster",
			"metrics_basis":     "sum(sampled node usage) / sum(sampled node allocatable)",
		},
	}

	// 提取 Top5 工作负载快照（按副本数排序）
	topWorkloads := s.extractTopWorkloads(deployments, statefulsets, daemonsets)
	if len(topWorkloads) > 0 {
		out["top_workloads"] = topWorkloads
	}

	// 内联证书风险（不阻塞首屏，失败时忽略）
	if certRisks, ok := s.getCachedClusterCertRisks(ctx, clusterID); ok && len(certRisks) > 0 {
		out["risks"] = map[string]any{"certificates": certRisks}
	}

	if s.cache != nil && s.cache.Enabled() {
		co := cachedOverview{
			Data:      out,
			ExpiresAt: time.Now().Add(120 * time.Second),
		}
		if b, err := json.Marshal(co); err == nil {
			writeCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			_ = s.cache.Set(writeCtx, cacheKey, b, 120*time.Second)
			cancel()
		}
	}

	return out, nil
}

// toInt32 从 any 安全转 int32，兼容 json 解析后的 float64 / int64 / json.Number
func toInt32(v any) int32 {
	switch n := v.(type) {
	case int:
		return int32(n)
	case int64:
		return int32(n)
	case float64:
		return int32(n)
	case int32:
		return n
	default:
		return 0
	}
}

func unhealthyPodReason(p *corev1.Pod) string {
	if p == nil || p.DeletionTimestamp != nil || p.Status.Phase == corev1.PodSucceeded {
		return ""
	}
	statuses := append(append([]corev1.ContainerStatus{}, p.Status.InitContainerStatuses...), p.Status.ContainerStatuses...)
	for _, status := range statuses {
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" && status.State.Waiting.Reason != "ContainerCreating" && status.State.Waiting.Reason != "PodInitializing" {
			return status.State.Waiting.Reason
		}
		if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
			if status.State.Terminated.Reason != "" {
				return status.State.Terminated.Reason
			}
			return "ExitError"
		}
	}
	if p.Status.Phase == corev1.PodFailed {
		return string(corev1.PodFailed)
	}
	return ""
}

// extractTopWorkloads 从工作负载列表中提取 Top5 快照（按副本数排序）
func (s *DashboardService) extractTopWorkloads(deployments, statefulsets, daemonsets []any) []map[string]any {
	type wl struct {
		name      string
		namespace string
		kind      string
		replicas  int32
		ready     int32
	}
	var all []wl

	extract := func(items []any, kind string) {
		for _, item := range items {
			switch workload := item.(type) {
			case *appsv1.Deployment:
				all = append(all, wl{name: workload.Name, namespace: workload.Namespace, kind: kind, replicas: workload.Status.Replicas, ready: workload.Status.ReadyReplicas})
				continue
			case *appsv1.StatefulSet:
				all = append(all, wl{name: workload.Name, namespace: workload.Namespace, kind: kind, replicas: workload.Status.Replicas, ready: workload.Status.ReadyReplicas})
				continue
			case *appsv1.DaemonSet:
				all = append(all, wl{name: workload.Name, namespace: workload.Namespace, kind: kind, replicas: workload.Status.DesiredNumberScheduled, ready: workload.Status.NumberReady})
				continue
			}
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name, _ := m["name"].(string)
			if name == "" {
				if meta, ok := m["metadata"].(map[string]any); ok {
					name, _ = meta["name"].(string)
				}
			}
			if name == "" {
				continue
			}
			ns := ""
			if meta, ok := m["metadata"].(map[string]any); ok {
				ns, _ = meta["namespace"].(string)
			}
			var replicas, ready int32
			if spec, ok := m["spec"].(map[string]any); ok {
				replicas = toInt32(spec["replicas"])
			}
			if status, ok := m["status"].(map[string]any); ok {
				ready = toInt32(status["readyReplicas"])
				if kind == "DaemonSet" {
					replicas = toInt32(status["desiredNumberScheduled"])
					ready = toInt32(status["numberReady"])
				}
			}
			all = append(all, wl{name: name, namespace: ns, kind: kind, replicas: replicas, ready: ready})
		}
	}

	extract(deployments, "Deployment")
	extract(statefulsets, "StatefulSet")
	extract(daemonsets, "DaemonSet")

	sort.Slice(all, func(i, j int) bool {
		return all[i].replicas > all[j].replicas
	})

	if len(all) > 5 {
		all = all[:5]
	}

	result := make([]map[string]any, 0, len(all))
	for _, w := range all {
		result = append(result, map[string]any{
			"name":      w.name,
			"namespace": w.namespace,
			"kind":      w.kind,
			"replicas":  w.replicas,
			"ready":     w.ready,
		})
	}
	return result
}

func (s *DashboardService) GetClusterCertificateRisks(ctx context.Context, clusterID uint64) ([]map[string]any, error) {
	if s.clusterReg == nil || s.k8sSvc == nil {
		return nil, errors.New("dependency missing")
	}
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}
	if _, err := s.clusterReg.Get(ctx, clusterID); err != nil {
		return nil, legacyClusterError(err)
	}
	if cached, ok := s.getCachedClusterCertRisks(ctx, clusterID); ok {
		return cached, nil
	}

	apiOK := false
	apiCtx, apiCancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	if cs, err := s.k8sSvc.typedClient(apiCtx, clusterID); err == nil {
		_, err = cs.Discovery().ServerVersion()
		apiOK = err == nil
	}
	apiCancel()

	certCtx, certCancel := context.WithTimeout(ctx, 5*time.Second)
	defer certCancel()
	risks, cacheable := s.getClusterCertificateRisks(certCtx, clusterID, apiOK)
	if cacheable {
		s.setCachedClusterCertRisks(ctx, clusterID, risks, 10*time.Minute)
	} else if len(risks) > 0 {
		s.setCachedClusterCertRisks(ctx, clusterID, risks, 45*time.Second)
	}
	return risks, nil
}

type nodeUsagePercent struct {
	Name   string
	IP     string
	CPU    float64
	Memory float64
}

type clusterUsageSnapshot struct {
	CPU    float64
	Memory float64
	Nodes  []nodeUsagePercent
}

func nodePrimaryIP(node corev1.Node) string {
	for _, addressType := range []corev1.NodeAddressType{corev1.NodeInternalIP, corev1.NodeExternalIP} {
		for _, address := range node.Status.Addresses {
			if address.Type == addressType && strings.TrimSpace(address.Address) != "" {
				return strings.TrimSpace(address.Address)
			}
		}
	}
	return ""
}

// getClusterUsageSnapshot 计算集群及各节点 CPU/内存使用率（0-100）。
// 计算方式：
// - 分母：本次成功返回指标的节点 allocatable 资源总和
// - 分子：同一批节点在 metrics.k8s.io/v1beta1 中的 usage 总和
// 若指标不可用或解析失败，返回 (0, 0)。
func (s *DashboardService) getClusterUsageSnapshot(ctx context.Context, clusterID uint64, nodeItems []corev1.Node) (clusterUsageSnapshot, bool) {
	if s.k8sSvc == nil || clusterID == 0 {
		return clusterUsageSnapshot{}, false
	}
	if len(nodeItems) == 0 {
		cs, err := s.k8sSvc.typedClient(ctx, clusterID)
		if err != nil {
			return clusterUsageSnapshot{}, false
		}
		nodes, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil || len(nodes.Items) == 0 {
			return clusterUsageSnapshot{}, false
		}
		nodeItems = nodes.Items
	}
	metrics, err := s.k8sSvc.List(ctx, clusterID, schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}, "", "", "", nil)
	if err != nil || len(metrics) == 0 {
		return clusterUsageSnapshot{}, false
	}
	return calculateClusterUsageSnapshot(nodeItems, metrics)
}

func calculateClusterUsageSnapshot(nodeItems []corev1.Node, metrics []any) (clusterUsageSnapshot, bool) {
	type nodeAllocatable struct {
		CPU    int64
		Memory int64
		Node   *corev1.Node
	}
	allocatableByNode := make(map[string]nodeAllocatable, len(nodeItems))
	for i := range nodeItems {
		node := &nodeItems[i]
		allocatableByNode[node.Name] = nodeAllocatable{
			CPU:    node.Status.Allocatable.Cpu().MilliValue(),
			Memory: node.Status.Allocatable.Memory().Value(),
			Node:   node,
		}
	}
	var usedCPU int64
	var usedMem int64
	var allocCPU int64
	var allocMem int64
	nodeUsages := make([]nodeUsagePercent, 0, len(metrics))
	for i := range metrics {
		m, ok := metrics[i].(map[string]any)
		if !ok {
			continue
		}
		usage, ok := m["usage"].(map[string]any)
		if !ok {
			continue
		}
		metadata, ok := m["metadata"].(map[string]any)
		if !ok {
			continue
		}
		name, _ := metadata["name"].(string)
		allocatable, ok := allocatableByNode[name]
		if !ok || allocatable.CPU <= 0 || allocatable.Memory <= 0 {
			continue
		}
		cpuRaw, cpuOK := usage["cpu"].(string)
		memoryRaw, memoryOK := usage["memory"].(string)
		if !cpuOK || !memoryOK {
			continue
		}
		cpuQuantity, cpuErr := resource.ParseQuantity(cpuRaw)
		memoryQuantity, memoryErr := resource.ParseQuantity(memoryRaw)
		if cpuErr != nil || memoryErr != nil {
			continue
		}
		usedCPU += cpuQuantity.MilliValue()
		usedMem += memoryQuantity.Value()
		allocCPU += allocatable.CPU
		allocMem += allocatable.Memory
		nodeUsages = append(nodeUsages, nodeUsagePercent{
			Name: name, IP: nodePrimaryIP(*allocatable.Node),
			CPU:    usagePercent(cpuQuantity.MilliValue(), allocatable.CPU),
			Memory: usagePercent(memoryQuantity.Value(), allocatable.Memory),
		})
	}
	if usedCPU < 0 || usedMem < 0 || allocCPU <= 0 || allocMem <= 0 {
		return clusterUsageSnapshot{}, false
	}
	sort.Slice(nodeUsages, func(i, j int) bool { return nodeUsages[i].Name < nodeUsages[j].Name })
	return clusterUsageSnapshot{
		CPU: usagePercent(usedCPU, allocCPU), Memory: usagePercent(usedMem, allocMem), Nodes: nodeUsages,
	}, true
}

func usagePercent(used, allocatable int64) float64 {
	value := float64(used) * 100 / float64(allocatable)
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	return float64(int(value*10+0.5)) / 10
}

// lastNDaysLabels 生成最近 n 天（含当天）的日期标签列表，格式为 MM-DD。
func lastNDaysLabels(n int) []string {
	if n <= 0 {
		return []string{}
	}
	out := make([]string, 0, n)
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -(n - 1))
	for i := 0; i < n; i++ {
		d := start.AddDate(0, 0, i)
		out = append(out, d.Format("01-02"))
	}
	return out
}

// lastNHoursLabels 生成最近 n 小时（含当前小时）的小时标签列表，格式为 HH:00（UTC）。
func lastNHoursLabels(n int) []string {
	if n <= 0 {
		return []string{}
	}
	out := make([]string, 0, n)
	now := time.Now().UTC().Truncate(time.Hour)
	start := now.Add(-time.Duration(n-1) * time.Hour)
	for i := 0; i < n; i++ {
		t := start.Add(time.Duration(i) * time.Hour)
		out = append(out, t.Format("15:00"))
	}
	return out
}

// topNamespacePods 将命名空间 -> Pod 数量映射排序后取 Top N。
// 当 pods 数量相同，按命名空间名称字典序稳定排序。
func topNamespacePods(m map[string]int, limit int) []map[string]any {
	type kv struct {
		k string
		v int
	}
	all := make([]kv, 0, len(m))
	for k, v := range m {
		all = append(all, kv{k: k, v: v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].v == all[j].v {
			return all[i].k < all[j].k
		}
		return all[i].v > all[j].v
	})
	if limit <= 0 || limit > len(all) {
		limit = len(all)
	}
	out := make([]map[string]any, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, map[string]any{"namespace": all[i].k, "pods": all[i].v})
	}
	return out
}
