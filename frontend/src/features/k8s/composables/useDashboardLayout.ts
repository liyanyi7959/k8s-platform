/**
 * 仪表盘布局管理
 *
 * 负责网格布局的默认配置、用户偏好持久化和布局协调。
 * 从 useClusterManageDashboard 中抽离，仅处理布局相关逻辑。
 */
import { type Layout, type LayoutItem } from 'grid-layout-plus'
import { readStorageJson, writeStorageJson } from '@/features/k8s/pages/ClusterManageView.utils'

/* ── Types ── */

export type DashboardCardVm = { key: string; label: string; value: string | number; sub: string; glow: string }

export type DashboardWidgetKey =
  | 'core'
  | 'health'
  | 'alerts'
  | 'certRisk'
  | 'resourceTrend'
  | 'nodeReady'
  | 'nodeCompare'
  | 'podPhase'
  | 'workloadHealth'
  | 'nsTop'
  | 'nsResource'
  | 'workloadList'
  | 'nodeList'
  | 'events'
  | 'logs'

export type DashboardWidgetOption = { key: DashboardWidgetKey; label: string }

export type DashboardChartKey = 'health' | 'resourceTrend' | 'nodeReady' | 'nodeCompare' | 'podPhase' | 'workloadHealth' | 'nsTop' | 'nsResource'

/* ── Widget options ── */

export const dashboardWidgetOptions: DashboardWidgetOption[] = [
  { key: 'core', label: '集群核心状态' },
  { key: 'health', label: '健康评分' },
  { key: 'alerts', label: '实时告警' },
  { key: 'certRisk', label: '集群证书过期风险' },
  { key: 'resourceTrend', label: 'CPU/内存趋势' },
  { key: 'nodeReady', label: '节点 Ready' },
  { key: 'nodeCompare', label: '节点资源对比' },
  { key: 'workloadHealth', label: '工作负载健康' },
  { key: 'podPhase', label: 'Pod 状态分布' },
  { key: 'nsTop', label: 'Namespace Top Pods' },
  { key: 'nsResource', label: 'Namespace 资源占比' },
  { key: 'workloadList', label: '核心工作负载列表' },
  { key: 'nodeList', label: '节点详情' },
  { key: 'events', label: '最近事件' },
  { key: 'logs', label: '关键 Pod 日志预览' }
]

/* ── Layout item validator ── */

export function pickLayoutItem(v: any): LayoutItem | null {
  const i = v?.i
  const x = Number(v?.x)
  const y = Number(v?.y)
  const w = Number(v?.w)
  const h = Number(v?.h)
  if (i == null || !Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(w) || !Number.isFinite(h)) return null
  if (w <= 0 || h <= 0) return null
  return { i, x, y, w, h }
}

/* ── Default layout builder ── */

export function buildDefaultDashboardLayout(enabled: DashboardWidgetKey[]): Layout {
  const has = new Set(enabled)
  const items: Layout = []
  if (has.has('core')) items.push({ i: 'core', x: 0, y: 0, w: 12, h: 4, minW: 8, minH: 3 })
  if (has.has('alerts')) items.push({ i: 'alerts', x: 0, y: 4, w: 12, h: 3, minW: 6, minH: 2 })

  if (has.has('resourceTrend')) items.push({ i: 'resourceTrend', x: 0, y: 7, w: 8, h: 8, minW: 6, minH: 6 })
  if (has.has('nodeCompare')) items.push({ i: 'nodeCompare', x: 8, y: 7, w: 4, h: 8, minW: 4, minH: 6 })

  if (has.has('health')) items.push({ i: 'health', x: 0, y: 15, w: 4, h: 7, minW: 4, minH: 3 })
  if (has.has('podPhase')) items.push({ i: 'podPhase', x: 4, y: 15, w: 4, h: 7, minW: 4, minH: 6 })
  if (has.has('workloadHealth')) items.push({ i: 'workloadHealth', x: 8, y: 15, w: 4, h: 7, minW: 4, minH: 6 })

  if (has.has('nsTop')) items.push({ i: 'nsTop', x: 0, y: 22, w: 4, h: 7, minW: 4, minH: 6 })
  if (has.has('workloadList')) items.push({ i: 'workloadList', x: 4, y: 22, w: 8, h: 8, minW: 6, minH: 6 })

  if (has.has('nsResource')) items.push({ i: 'nsResource', x: 0, y: 30, w: 8, h: 6, minW: 6, minH: 5 })
  if (has.has('nodeList')) items.push({ i: 'nodeList', x: 8, y: 30, w: 4, h: 6, minW: 4, minH: 6 })

  if (has.has('events')) items.push({ i: 'events', x: 0, y: 36, w: 6, h: 7, minW: 6, minH: 5 })
  if (has.has('logs')) items.push({ i: 'logs', x: 6, y: 36, w: 6, h: 7, minW: 6, minH: 5 })
  if (has.has('certRisk')) items.push({ i: 'certRisk', x: 0, y: 43, w: 12, h: 7, minW: 8, minH: 5 })
  return items
}

/* ── Layout reconciler ── */

export function reconcileDashboardLayout(enabled: DashboardWidgetKey[], base: Layout): Layout {
  const enabledSet = new Set(enabled)
  const existing = new Map<string, LayoutItem>()
  for (const it of base) {
    const key = String(it.i)
    if (!enabledSet.has(key as DashboardWidgetKey)) continue
    existing.set(key, it)
  }
  const defaults = buildDefaultDashboardLayout(enabled)
  return defaults.map((d) => {
    const e = existing.get(String(d.i))
    if (!e) return d
    return { ...d, x: e.x, y: e.y, w: e.w, h: Number(e.h ?? d.h) }
  })
}

/* ── Persistence helpers ── */

export function dashboardStorageKey(clusterId: number, suffix: string): string {
  return `k8s:dashboard:${clusterId}:${suffix}`
}

export function loadDashboardPrefsFromStorage(
  clusterId: number
): { enabled: DashboardWidgetKey[]; layout: Layout } {
  const allKeys = new Set(dashboardWidgetOptions.map((w) => w.key))

  try {
    const enabledList = readStorageJson<unknown>(dashboardStorageKey(clusterId, 'enabled'))
    const layoutList = readStorageJson<unknown>(dashboardStorageKey(clusterId, 'layout'))

    const enabled = Array.isArray(enabledList) ? enabledList.map((k) => String(k)) : []
    const normalizedEnabled = enabled.filter((k) => allKeys.has(k as DashboardWidgetKey)) as DashboardWidgetKey[]

    const finalEnabled = normalizedEnabled.length
      ? ([...normalizedEnabled, ...dashboardWidgetOptions.map((w) => w.key).filter((k) => !normalizedEnabled.includes(k))] as DashboardWidgetKey[])
      : dashboardWidgetOptions.map((w) => w.key)

    const picked = Array.isArray(layoutList) ? layoutList.map(pickLayoutItem).filter(Boolean) : []
    const layout = reconcileDashboardLayout(finalEnabled, picked as Layout)

    return { enabled: finalEnabled, layout }
  } catch {
    const enabled = dashboardWidgetOptions.map((w) => w.key)
    return { enabled, layout: buildDefaultDashboardLayout(enabled) }
  }
}

export function saveDashboardPrefsToStorage(
  clusterId: number,
  enabled: DashboardWidgetKey[],
  layout: Layout
): void {
  const compactLayout = layout.map((it) => ({ i: it.i, x: it.x, y: it.y, w: it.w, h: it.h }))
  writeStorageJson(dashboardStorageKey(clusterId, 'enabled'), enabled)
  writeStorageJson(dashboardStorageKey(clusterId, 'layout'), compactLayout)
}
