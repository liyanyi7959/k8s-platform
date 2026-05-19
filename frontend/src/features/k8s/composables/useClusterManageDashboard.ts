/**
 * 仪表盘组合函数 — 精简版
 *
 * 原 1079 行现拆分为三个文件：
 *  - useDashboardCharts.ts   ← 纯函数：图表 Option 构建
 *  - useDashboardLayout.ts   ← 纯函数/类型：布局管理、持久化
 *  - useClusterManageDashboard.ts（本文件） ← Vue 组合函数：状态、加载、实例
 */
import { computed, onBeforeUnmount, onMounted, ref, watch, type ComputedRef } from 'vue'
import * as dashboardApi from '@/features/dashboard/api/dashboard'
import * as k8sApi from '@/features/k8s/api/k8s'
import { notifyError } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'

import {
  type DashboardCardVm,
  type DashboardWidgetKey,
  dashboardWidgetOptions,
} from './useDashboardLayout'

// re-export types so callers don't need to change imports
export type { DashboardCardVm, DashboardWidgetKey }
export { dashboardWidgetOptions }

export function useClusterManageDashboard(opts: { clusterId: ComputedRef<number>; currentResource: ComputedRef<string | undefined> }) {
  const { clusterId, currentResource } = opts

  /* ── 响应式状态 ── */
  const loadingDashboard = ref(false)
  const clusterOverview = ref<dashboardApi.DashboardClusterOverview | null>(null)
  const dashboardNamespacesTotal = ref<number | null>(null)
  const dashboardEvents = ref<any[]>([])
  const dashboardLastUpdatedAt = ref<number | null>(null)
  const dashboardCertRiskLoading = ref(false)
  const dashboardCertRiskUnavailable = ref(false)
  const dashboardAutoRefresh = ref(true)
  const dashboardAutoRefreshSec = ref(30)
  /** Namespace names fetched during dashboard load — shared to avoid duplicate listNamespaces call */
  const dashboardNamespaceNames = ref<string[] | null>(null)
  const dashboardNamespaceFetchedAt = ref<number | null>(null)
  const dashboardCertRiskFetchedAt = ref<number | null>(null)

  /* ── 派生计算 ── */
  const dashboardWorkloadsTotal = computed(() => {
    const w = clusterOverview.value?.stats.workloads
    if (!w) return 0
    return Number(w.deployments ?? 0) + Number(w.statefulsets ?? 0) + Number(w.daemonsets ?? 0)
  })

  const dashboardCpuText = computed(() => `${Math.round(Number(clusterOverview.value?.stats.cpu.used_percent ?? 0))}%`)
  const dashboardMemoryText = computed(() => `${Math.round(Number(clusterOverview.value?.stats.memory.used_percent ?? 0))}%`)
  const dashboardReadyText = computed(() => {
    const s = clusterOverview.value?.stats.nodes
    if (!s) return '-'
    return `${s.ready}/${s.total}`
  })
  const dashboardPodRunningText = computed(() => String(clusterOverview.value?.stats.pods.running ?? 0))
  const dashboardPodPendingText = computed(() => String(clusterOverview.value?.stats.pods.pending ?? 0))
  const dashboardPodFailedText = computed(() => String(clusterOverview.value?.stats.pods.failed ?? 0))
  const dashboardTopNsText = computed(() => String(clusterOverview.value?.charts.namespace_pods_top?.[0]?.namespace ?? '-'))

  const dashboardCards = computed<DashboardCardVm[]>(() => {
    const d = clusterOverview.value
    if (!d) {
      return [
        { key: 'nodes', label: '节点', value: '-', sub: 'Ready/Total', glow: 'stat-glow-cyan' },
        { key: 'pods', label: 'Pods', value: '-', sub: 'Running/Total', glow: 'stat-glow-blue' },
        { key: 'workloads', label: '工作负载', value: '-', sub: 'Deployment/StatefulSet/DaemonSet', glow: 'stat-glow-violet' },
        { key: 'status', label: '集群状态', value: '-', sub: 'active / degraded / disabled', glow: 'stat-glow-slate' }
      ]
    }
    return [
      { key: 'nodes', label: '节点', value: d.stats.nodes.total, sub: `Ready ${d.stats.nodes.ready}/${d.stats.nodes.total}`, glow: 'stat-glow-cyan' },
      { key: 'pods', label: 'Pods', value: d.stats.pods.total, sub: `Running ${d.stats.pods.running}/${d.stats.pods.total}`, glow: 'stat-glow-blue' },
      {
        key: 'workloads',
        label: '工作负载',
        value: dashboardWorkloadsTotal.value,
        sub: `D ${d.stats.workloads.deployments} / S ${d.stats.workloads.statefulsets} / Da ${d.stats.workloads.daemonsets}`,
        glow: 'stat-glow-violet'
      },
      { key: 'status', label: '集群状态', value: d.cluster.status, sub: d.cluster.name, glow: 'stat-glow-slate' }
    ]
  })

  const dashboardAlertCounts = computed(() => {
    const list = dashboardEvents.value
    let critical = 0
    let warning = 0
    let info = 0
    for (const e of list) {
      const type = String(e?.type ?? '')
      const reason = String(e?.reason ?? '')
      const msg = String(e?.message ?? '')
      const text = `${reason} ${msg}`.toLowerCase()
      if (type === 'Warning') {
        if (text.includes('failed') || text.includes('backoff') || text.includes('unhealthy') || text.includes('evict')) critical += 1
        else warning += 1
      } else {
        info += 1
      }
    }
    return { critical, warning, info }
  })

  const dashboardHealthScore = computed(() => {
    const d = clusterOverview.value
    if (!d) return 0
    const nodeTotal = Math.max(1, Number(d.stats.nodes.total ?? 1))
    const nodeReady = Math.max(0, Number(d.stats.nodes.ready ?? 0))
    const nodeHealth = nodeReady / nodeTotal
    const podTotal = Math.max(1, Number(d.stats.pods.total ?? 1))
    const podRunning = Math.max(0, Number(d.stats.pods.running ?? 0))
    const podHealth = podRunning / podTotal
    const cpuUsed = Math.max(0, Math.min(100, Number(d.stats.cpu.used_percent ?? 0)))
    const memUsed = Math.max(0, Math.min(100, Number(d.stats.memory.used_percent ?? 0)))
    const usage = (cpuUsed + memUsed) / 2
    const resourceScore = (100 - usage) / 100
    const configScore = 1
    const raw = nodeHealth * 30 + podHealth * 30 + resourceScore * 20 + configScore * 20
    return Math.max(0, Math.min(100, Math.round(raw)))
  })

  /* ── 看板配置 ── */
  const dashboardEnabledWidgets = ref<DashboardWidgetKey[]>(dashboardWidgetOptions.map((w) => w.key))

  function loadDashboardPrefs() {
    // 保留接口但不做持久化操作
  }

  /* ── 数据加载 ── */

  /* ── localStorage cache for instant display ── */
  const OVERVIEW_CACHE_PREFIX = 'cluster_overview_v1_'
  const CERT_RISK_TS_PREFIX = 'cluster_overview_cert_risk_ts_v1_'

  function getCachedCertRiskFetchedAt(id: number): number | null {
    try {
      const raw = localStorage.getItem(CERT_RISK_TS_PREFIX + id)
      const value = Number(raw)
      return Number.isFinite(value) && value > 0 ? value : null
    } catch {
      return null
    }
  }

  function setCachedCertRiskFetchedAt(id: number, ts: number) {
    try {
      localStorage.setItem(CERT_RISK_TS_PREFIX + id, String(ts))
    } catch {
      // ignore quota/storage failures
    }
  }

  function getCachedOverview(id: number): { data: dashboardApi.DashboardClusterOverview; ts: number } | null {
    try {
      const raw = localStorage.getItem(OVERVIEW_CACHE_PREFIX + id)
      if (!raw) return null
      const parsed = JSON.parse(raw) as { data: dashboardApi.DashboardClusterOverview; ts: number }
      // Use cache if less than 5 minutes old
      if (Date.now() - parsed.ts > 5 * 60 * 1000) return null
      return parsed
    } catch { return null }
  }
  function setCachedOverview(id: number, data: dashboardApi.DashboardClusterOverview) {
    try {
      localStorage.setItem(OVERVIEW_CACHE_PREFIX + id, JSON.stringify({ data, ts: Date.now() }))
    } catch { /* quota exceeded — ignore */ }
  }

  function setDashboardNamespaceSnapshot(names: string[]) {
    dashboardNamespaceNames.value = [...names]
    dashboardNamespacesTotal.value = names.length
    dashboardNamespaceFetchedAt.value = Date.now()
  }

  let dashboardLoadSeq = 0
  let dashboardCertRiskSeq = 0
  async function loadDashboardCertificateRisks(targetClusterId: number): Promise<void> {
    if (!targetClusterId || !clusterOverview.value) return
    const hasCertData = Array.isArray(clusterOverview.value.risks?.certificates) && clusterOverview.value.risks.certificates.length > 0
    const certRiskFresh = dashboardCertRiskFetchedAt.value != null && Date.now() - dashboardCertRiskFetchedAt.value < 5 * 60 * 1000
    if (hasCertData && certRiskFresh) {
      dashboardCertRiskLoading.value = false
      return
    }
    const seq = ++dashboardCertRiskSeq
    dashboardCertRiskLoading.value = true
    dashboardCertRiskUnavailable.value = false
    try {
      const risks = await dashboardApi.getClusterCertificateRisks(targetClusterId)
      if (seq !== dashboardCertRiskSeq || targetClusterId !== clusterId.value || !clusterOverview.value) return
      clusterOverview.value = {
        ...clusterOverview.value,
        risks: {
          ...(clusterOverview.value.risks ?? {}),
          certificates: risks,
        },
      }
      dashboardCertRiskFetchedAt.value = Date.now()
      setCachedCertRiskFetchedAt(targetClusterId, dashboardCertRiskFetchedAt.value)
      setCachedOverview(targetClusterId, clusterOverview.value)
    } catch {
      if (seq === dashboardCertRiskSeq) {
        dashboardCertRiskUnavailable.value = true
      }
    } finally {
      if (seq === dashboardCertRiskSeq) {
        dashboardCertRiskLoading.value = false
      }
    }
  }

  async function loadDashboard(): Promise<string[] | null> {
    if (!clusterId.value) return null
    const targetClusterId = clusterId.value
    const seq = ++dashboardLoadSeq

    // Show cached data instantly while fetching fresh data
    if (!clusterOverview.value) {
      const cached = getCachedOverview(targetClusterId)
      if (cached) {
        clusterOverview.value = cached.data
        dashboardLastUpdatedAt.value = cached.ts
        if (Array.isArray(cached.data.risks?.certificates) && cached.data.risks.certificates.length > 0) {
          dashboardCertRiskFetchedAt.value = getCachedCertRiskFetchedAt(targetClusterId)
        }
      }
    }

    loadingDashboard.value = true
    try {
      // Fire overview first; events are secondary.
      const overviewPromise = dashboardApi.getClusterOverview(clusterId.value)
      const eventsPromise = k8sApi.listEvents(clusterId.value, { sort_by: 'lastTimestamp', order: 'desc' }).catch(() => ({ list: [] as any[] }))

      // Render overview as soon as it arrives (don't wait for events)
      const overview = await overviewPromise
      if (seq !== dashboardLoadSeq || targetClusterId !== clusterId.value) return null
      const preservedCertificates = Array.isArray(clusterOverview.value?.risks?.certificates)
        ? clusterOverview.value?.risks?.certificates
        : undefined
      const mergedOverview = preservedCertificates && !Array.isArray(overview.risks?.certificates)
        ? {
            ...overview,
            risks: {
              ...(overview.risks ?? {}),
              certificates: preservedCertificates,
            },
          }
        : overview
      clusterOverview.value = mergedOverview
      dashboardLastUpdatedAt.value = Date.now()
      setCachedOverview(targetClusterId, mergedOverview)
      void loadDashboardCertificateRisks(targetClusterId)

      const shouldRefreshNamespaces = dashboardNamespaceFetchedAt.value == null
        ? dashboardLoadSeq > 1 && dashboardNamespacesTotal.value == null
        : Date.now() - dashboardNamespaceFetchedAt.value >= 60 * 1000
      if (shouldRefreshNamespaces) {
        void k8sApi.listNamespaces(targetClusterId).then((data) => {
          if (seq !== dashboardLoadSeq || targetClusterId !== clusterId.value) return
          setDashboardNamespaceSnapshot(data.list.map((it) => it.metadata.name))
        }).catch(() => {
          // Namespace total is secondary and should not fail the dashboard refresh.
        })
      }

      // Events resolve in background.
      const events = await eventsPromise
      if (seq !== dashboardLoadSeq || targetClusterId !== clusterId.value) return null
      dashboardEvents.value = Array.isArray(events.list) ? events.list : []
    } catch (e) {
      const err = e as ApiError
      notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
    } finally {
      if (seq === dashboardLoadSeq && targetClusterId === clusterId.value) {
        loadingDashboard.value = false
      }
    }
    return null
  }

  /* ── 自动刷新 ── */
  let autoRefreshTimer: number | null = null
  function stopAutoRefresh() {
    if (autoRefreshTimer != null) window.clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }

  function startAutoRefresh() {
    stopAutoRefresh()
    if (!dashboardAutoRefresh.value) return
    if (currentResource.value !== 'dashboard') return
    const ms = Math.max(5, Number(dashboardAutoRefreshSec.value || 30)) * 1000
    autoRefreshTimer = window.setInterval(() => {
      if (currentResource.value !== 'dashboard') return
      void loadDashboard()
    }, ms)
  }

  /* ── Watchers ── */
  watch(
    () => currentResource.value,
    (r, prev) => {
      if (prev === 'dashboard' && r !== 'dashboard') {
        stopAutoRefresh()
      }
      if (r === 'dashboard') {
        startAutoRefresh()
      }
    }
  )

  watch(
    () => [dashboardAutoRefresh.value, dashboardAutoRefreshSec.value, currentResource.value].join('|'),
    () => startAutoRefresh()
  )

  watch(
    () => clusterId.value,
    () => {
      dashboardLoadSeq += 1
      dashboardCertRiskSeq += 1
      clusterOverview.value = null
      dashboardNamespacesTotal.value = null
      dashboardEvents.value = []
      dashboardLastUpdatedAt.value = null
      dashboardCertRiskLoading.value = false
      dashboardCertRiskUnavailable.value = false
      loadingDashboard.value = false
      dashboardNamespaceNames.value = null
      dashboardNamespaceFetchedAt.value = null
      dashboardCertRiskFetchedAt.value = null
    }
  )

  onMounted(() => {
    startAutoRefresh()
  })

  onBeforeUnmount(() => {
    stopAutoRefresh()
  })

  return {
    clusterOverview,
    loadingDashboard,
    loadDashboard,
    loadDashboardPrefs,
    dashboardNamespacesTotal,
    dashboardEvents,
    dashboardLastUpdatedAt,
    dashboardCertRiskLoading,
    dashboardCertRiskUnavailable,
    dashboardAlertCounts,
    dashboardHealthScore,
    dashboardAutoRefresh,
    dashboardAutoRefreshSec,
    dashboardCpuText,
    dashboardMemoryText,
    dashboardReadyText,
    dashboardPodRunningText,
    dashboardPodPendingText,
    dashboardPodFailedText,
    dashboardTopNsText,
    dashboardWorkloadsTotal,
    dashboardEnabledWidgets,
    dashboardWidgetOptions,
    dashboardNamespaceNames,
    setDashboardNamespaceSnapshot,
  }
}
