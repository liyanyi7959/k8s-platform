/**
 * 仪表盘图表选项构建器
 *
 * 从 useClusterManageDashboard 中抽离的纯函数集合，
 * 负责根据后端数据 → 生成 ECharts Option。
 * 不涉及任何 Vue 响应式，可独立单测。
 */
import type { EChartsOption } from '@/shared/utils/echarts'
import * as dashboardApi from '@/features/dashboard/api/dashboard'

/* ── dark-mode helpers ── */
export function isDark(): boolean {
  return document.documentElement.classList.contains('dark')
}

/** ECharts 配色：根据当前暗色模式返回对应色值 */
export function chartColors() {
  const dark = isDark()
  return {
    textPrimary: dark ? 'rgba(226,232,240,0.92)' : 'rgba(2,6,23,0.86)',
    textSecondary: dark ? 'rgba(148,163,184,0.9)' : 'rgba(2,6,23,0.72)',
    textTertiary: dark ? 'rgba(148,163,184,0.8)' : 'rgba(100,116,139,0.9)',
    markLabel: dark ? 'rgba(226,232,240,0.7)' : 'rgba(2,6,23,0.65)',
    barLabel: dark ? 'rgba(226,232,240,0.8)' : 'rgba(2,6,23,0.7)',
    titleText: dark ? 'rgba(241,245,249,0.92)' : 'rgba(2,6,23,0.88)',
    subtitleText: dark ? 'rgba(148,163,184,0.92)' : 'rgba(100,116,139,0.92)',
    axisLine: dark ? 'rgba(226,232,240,0.12)' : 'rgba(2,6,23,0.08)',
    splitLine: dark ? 'rgba(226,232,240,0.08)' : 'rgba(2,6,23,0.06)',
    gaugeTrack: dark ? 'rgba(226,232,240,0.12)' : 'rgba(2,6,23,0.08)',
    pieBorder: dark ? 'rgba(30,41,59,0.7)' : 'rgba(255,255,255,0.7)',
    emptyText: dark ? 'rgba(148,163,184,0.8)' : 'rgba(100,116,139,0.9)',
  }
}

/* ── 空状态 & 工具 ── */

export function emptyDashboardOption(text = '暂无数据'): EChartsOption {
  const c = chartColors()
  return {
    xAxis: { type: 'category', show: false, data: [] },
    yAxis: { type: 'value', show: false },
    grid: { left: 12, right: 12, top: 12, bottom: 12 },
    series: [],
    graphic: {
      type: 'text',
      left: 'center',
      top: 'middle',
      style: { text, fill: c.emptyText, fontSize: 12, fontWeight: 700 }
    }
  }
}

export function clampTrendPercentAxisMax(v: number): number {
  if (!Number.isFinite(v) || v <= 0) return 50
  const raw = Math.ceil((v * 3) / 10) * 10
  return Math.min(100, Math.max(50, raw))
}

/* ── Chart Option Builders ── */

export function buildDashboardTrendOption(d: dashboardApi.DashboardClusterOverview | null): EChartsOption {
  if (!d) return emptyDashboardOption()
  const labels = d.charts.cpu_memory_24h.labels ?? []
  if (labels.length === 0) return emptyDashboardOption()
  const cpu = (d.charts.cpu_memory_24h.cpu ?? []).map((x) => Number(x) || 0)
  const mem = (d.charts.cpu_memory_24h.memory ?? []).map((x) => Number(x) || 0)
  const seriesMax = Math.max(0, ...cpu, ...mem)
  const axisMax = clampTrendPercentAxisMax(seriesMax)
  const c = chartColors()
  return {
    color: ['rgba(34, 211, 238, 0.95)', 'rgba(100, 116, 139, 0.9)'],
    tooltip: { trigger: 'axis' },
    legend: { top: 2, left: 6, itemWidth: 10, itemHeight: 10, textStyle: { color: c.textSecondary, fontWeight: 700 } },
    grid: { left: 12, right: 12, top: 36, bottom: 18, containLabel: true },
    xAxis: {
      type: 'category',
      data: labels,
      boundaryGap: false,
      axisLabel: { color: c.textTertiary },
      axisLine: { lineStyle: { color: c.axisLine } },
      axisTick: { show: false }
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: axisMax,
      axisLabel: { color: c.textTertiary, formatter: '{value}%' },
      splitLine: { lineStyle: { color: c.splitLine } }
    },
    series: [
      {
        name: 'CPU',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 3 },
        areaStyle: { opacity: 0.12 },
        data: cpu,
        markLine: {
          symbol: 'none',
          label: { show: true, formatter: 'CPU警戒 {c}%', color: c.markLabel, fontWeight: 800 },
          lineStyle: { color: 'rgba(239, 68, 68, 0.7)', width: 2, type: 'dashed' },
          data: [{ yAxis: 80 }]
        }
      },
      {
        name: '内存',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 3 },
        areaStyle: { opacity: 0.08 },
        data: mem,
        markLine: {
          symbol: 'none',
          label: { show: true, formatter: '内存警戒 {c}%', color: c.markLabel, fontWeight: 800 },
          lineStyle: { color: 'rgba(245, 158, 11, 0.7)', width: 2, type: 'dashed' },
          data: [{ yAxis: 85 }]
        }
      }
    ]
  }
}

export function buildDashboardNodeReadyOption(d: dashboardApi.DashboardClusterOverview | null): EChartsOption {
  if (!d) return emptyDashboardOption()
  const ready = d.charts.node_ready.ready ?? 0
  const total = Math.max(1, d.charts.node_ready.total ?? 1)
  const percent = Math.round((ready / total) * 1000) / 10
  const c = chartColors()
  return {
    tooltip: { trigger: 'item', formatter: `{b}: ${ready}/${total} (${percent}%)` },
    series: [
      {
        name: '节点 Ready',
        type: 'gauge',
        min: 0,
        max: 100,
        startAngle: 220,
        endAngle: -40,
        progress: { show: true, width: 12, roundCap: true },
        axisLine: { lineStyle: { width: 12, color: [[1, c.gaugeTrack]] } },
        axisTick: { show: false },
        splitLine: { show: false },
        axisLabel: { show: false },
        pointer: { show: false },
        title: { show: true, offsetCenter: [0, '46%'], color: c.subtitleText, fontWeight: 800, fontSize: 12 },
        detail: { valueAnimation: true, offsetCenter: [0, '12%'], fontSize: 24, fontWeight: 900, color: c.textPrimary, formatter: '{value}%' },
        data: [{ value: percent, name: `Ready ${ready}/${total}` }]
      }
    ]
  }
}

export function buildDashboardHealthOption(score: number): EChartsOption {
  const s = Math.max(0, Math.min(100, Math.round(Number(score) || 0)))
  const color = s >= 80 ? 'rgba(16,185,129,0.92)' : s >= 60 ? 'rgba(245,158,11,0.92)' : 'rgba(239,68,68,0.92)'
  const c = chartColors()
  return {
    tooltip: { trigger: 'item', formatter: `{b}: ${s}` },
    series: [
      {
        name: '集群健康评分',
        type: 'gauge',
        min: 0,
        max: 100,
        startAngle: 220,
        endAngle: -40,
        progress: { show: true, width: 14, roundCap: true, itemStyle: { color } },
        axisLine: { lineStyle: { width: 14, color: [[1, c.gaugeTrack]] } },
        axisTick: { show: false },
        splitLine: { show: false },
        axisLabel: { show: false },
        pointer: { show: false },
        title: { show: true, offsetCenter: [0, '46%'], color: c.subtitleText, fontWeight: 800, fontSize: 12 },
        detail: {
          valueAnimation: true,
          offsetCenter: [0, '12%'],
          fontSize: 26,
          fontWeight: 900,
          color: c.textPrimary,
          formatter: '{value}'
        },
        data: [{ value: s, name: '健康评分' }]
      }
    ]
  }
}

export function buildDashboardPodPhaseOption(d: dashboardApi.DashboardClusterOverview | null): EChartsOption {
  if (!d) return emptyDashboardOption()
  const p = d.charts.pod_phase
  const c = chartColors()
  return {
    tooltip: { trigger: 'item' },
    legend: { top: 6, left: 'center', itemWidth: 10, itemHeight: 10, textStyle: { color: c.textSecondary, fontWeight: 700 } },
    title: {
      text: String(d.stats.pods.total ?? 0),
      subtext: 'Pods',
      left: 'center',
      top: 'middle',
      textStyle: { color: c.titleText, fontWeight: 900, fontSize: 22 },
      subtextStyle: { color: c.subtitleText, fontWeight: 700, fontSize: 12 }
    },
    color: ['rgba(16,185,129,0.9)', 'rgba(245,158,11,0.9)', 'rgba(239,68,68,0.9)', 'rgba(100,116,139,0.8)'],
    series: [
      {
        name: 'Pod Phase',
        type: 'pie',
        radius: ['55%', '78%'],
        itemStyle: { borderColor: c.pieBorder, borderWidth: 2 },
        label: { show: false },
        emphasis: { scale: true, scaleSize: 8 },
        data: [
          { name: 'Running', value: p.running },
          { name: 'Pending', value: p.pending },
          { name: 'Failed', value: p.failed },
          { name: 'Succeeded', value: p.succeeded }
        ]
      }
    ]
  }
}

export function buildDashboardWorkloadHealthOption(list: any[]): EChartsOption {
  const items = Array.isArray(list) ? list : []
  if (items.length === 0) return emptyDashboardOption('暂无工作负载数据')

  const kinds = ['Deployment', 'StatefulSet', 'DaemonSet', 'CronJob', 'Job'] as const
  const summary = kinds.map((k) => ({ kind: k, total: 0, unhealthy: 0 }))

  for (const row of items) {
    const kind = String(row?.kind ?? '')
    const target = summary.find((s) => s.kind === kind)
    if (!target) continue
    target.total += 1

    const desired = Number(row?.spec?.replicas ?? row?.status?.desiredNumberScheduled ?? row?.status?.active ?? 0)
    const available = Number(row?.status?.availableReplicas ?? row?.status?.readyReplicas ?? row?.status?.numberAvailable ?? row?.status?.succeeded ?? 0)
    const isHealthy = desired <= 0 ? true : available >= desired
    if (!isHealthy) target.unhealthy += 1
  }

  const data = summary
    .filter((s) => s.total > 0)
    .map((s) => {
      const ratio = s.total <= 0 ? 1 : (s.total - s.unhealthy) / s.total
      const color = ratio >= 1 ? 'rgba(16,185,129,0.9)' : ratio >= 0.8 ? 'rgba(245,158,11,0.9)' : 'rgba(239,68,68,0.9)'
      const label = s.unhealthy > 0 ? `${s.kind} (${s.unhealthy}异常)` : s.kind
      return { name: label, value: s.total, itemStyle: { color } }
    })

  if (data.length === 0) return emptyDashboardOption('暂无工作负载数据')

  const c = chartColors()
  return {
    tooltip: { trigger: 'item' },
    legend: { top: 6, left: 'center', itemWidth: 10, itemHeight: 10, textStyle: { color: c.textSecondary, fontWeight: 700 } },
    title: {
      text: String(items.length),
      subtext: 'Workloads',
      left: 'center',
      top: 'middle',
      textStyle: { color: c.titleText, fontWeight: 900, fontSize: 22 },
      subtextStyle: { color: c.subtitleText, fontWeight: 700, fontSize: 12 }
    },
    series: [
      {
        name: '工作负载健康',
        type: 'pie',
        radius: ['55%', '78%'],
        itemStyle: { borderColor: c.pieBorder, borderWidth: 2 },
        label: { show: false },
        emphasis: { scale: true, scaleSize: 8 },
        data
      }
    ]
  }
}

export function buildDashboardNamespaceTopOption(d: dashboardApi.DashboardClusterOverview | null): EChartsOption {
  if (!d) return emptyDashboardOption()
  const list = d.charts.namespace_pods_top ?? []
  if (list.length === 0) return emptyDashboardOption()
  const names = list.map((it) => it.namespace)
  const values = list.map((it) => it.pods)
  const c = chartColors()
  return {
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: { left: 12, right: 12, top: 12, bottom: 12, containLabel: true },
    xAxis: {
      type: 'value',
      axisLabel: { color: c.textTertiary },
      splitLine: { lineStyle: { color: c.splitLine } }
    },
    yAxis: {
      type: 'category',
      data: names,
      axisLabel: { color: c.textTertiary, width: 120, overflow: 'truncate' },
      axisTick: { show: false },
      axisLine: { show: false }
    },
    series: [
      {
        type: 'bar',
        name: 'Pods',
        data: values,
        barWidth: 14,
        itemStyle: {
          borderRadius: 999,
          color: 'rgba(34, 211, 238, 0.94)'
        },
        label: { show: true, position: 'right', color: c.barLabel, fontWeight: 800 }
      }
    ]
  }
}

export function buildDashboardNamespaceResourceOption(d: dashboardApi.DashboardClusterOverview | null): EChartsOption {
  if (!d) return emptyDashboardOption()
  const list = d.charts.namespace_pods_top ?? []
  if (list.length === 0) return emptyDashboardOption()

  const totalPods = Math.max(1, Number(d.stats.pods.total ?? 0))
  const cpuUsed = Number(d.stats.cpu.used_percent ?? 0)
  const memUsed = Number(d.stats.memory.used_percent ?? 0)
  const safeCpuUsed = Number.isFinite(cpuUsed) ? cpuUsed : 0
  const safeMemUsed = Number.isFinite(memUsed) ? memUsed : 0

  const names = list.map((it) => it.namespace)
  const cpuValues = list.map((it) => Math.max(0, Math.round((safeCpuUsed * (Number(it.pods ?? 0) / totalPods)) * 10) / 10))
  const memValues = list.map((it) => Math.max(0, Math.round((safeMemUsed * (Number(it.pods ?? 0) / totalPods)) * 10) / 10))
  const maxV = Math.max(0, ...cpuValues, ...memValues)
  const axisMax = clampTrendPercentAxisMax(maxV)

  const c = chartColors()
  return {
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { top: 6, left: 'center', itemWidth: 10, itemHeight: 10, textStyle: { color: c.textSecondary, fontWeight: 700 } },
    grid: { left: 12, right: 12, top: 34, bottom: 12, containLabel: true },
    xAxis: {
      type: 'value',
      min: 0,
      max: axisMax,
      axisLabel: { color: c.textTertiary, formatter: '{value}%' },
      splitLine: { lineStyle: { color: c.splitLine } }
    },
    yAxis: {
      type: 'category',
      data: names,
      axisLabel: { color: c.textTertiary, width: 140, overflow: 'truncate' },
      axisTick: { show: false },
      axisLine: { show: false }
    },
    series: [
      {
        type: 'bar',
        name: 'CPU占比',
        data: cpuValues,
        barWidth: 12,
        itemStyle: { borderRadius: 999, color: 'rgba(34, 211, 238, 0.94)' },
        label: { show: true, position: 'right', color: c.barLabel, fontWeight: 800, formatter: '{c}%' }
      },
      {
        type: 'bar',
        name: '内存占比',
        data: memValues,
        barWidth: 12,
        itemStyle: { borderRadius: 999, color: 'rgba(59, 130, 246, 0.94)' },
        label: { show: true, position: 'right', color: c.barLabel, fontWeight: 800, formatter: '{c}%' }
      }
    ]
  }
}

/* ── Node 相关辅助 ── */

function pickNumberByPath(row: any, path: string): number | null {
  const parts = String(path || '')
    .split('.')
    .map((p) => p.trim())
    .filter(Boolean)
  let cur: any = row
  for (const p of parts) cur = cur?.[p]
  const v = Number(cur)
  return Number.isFinite(v) ? v : null
}

function pickNodeUsedPercent(row: any, kind: 'cpu' | 'memory'): number | null {
  const paths =
    kind === 'cpu'
      ? [
          'usage.cpu_percent',
          'usage.cpu.used_percent',
          'stats.cpu.used_percent',
          'cpu.used_percent',
          'cpu_used_percent',
          'cpuUsedPercent'
        ]
      : [
          'usage.memory_percent',
          'usage.memory.used_percent',
          'stats.memory.used_percent',
          'memory.used_percent',
          'memory_used_percent',
          'memUsedPercent'
        ]
  for (const p of paths) {
    const v = pickNumberByPath(row, p)
    if (v != null) return v
  }
  return null
}

export function buildDashboardNodeCompareOption(nodes: any[]): EChartsOption {
  const list = Array.isArray(nodes) ? nodes : []
  if (list.length === 0) return emptyDashboardOption('暂无节点数据')

  const rows = list
    .map((n) => {
      const name = String(n?.metadata?.name ?? '-')
      const cpu = pickNodeUsedPercent(n, 'cpu')
      const mem = pickNodeUsedPercent(n, 'memory')
      const score = cpu != null && mem != null ? (cpu + mem) / 2 : cpu != null ? cpu : mem != null ? mem : null
      return { name, cpu, mem, score }
    })
    .filter((r) => r.score != null)
    .sort((a, b) => Number(b.score) - Number(a.score))
    .slice(0, 8)

  if (rows.length === 0) return emptyDashboardOption('暂无节点 metrics')

  const c = chartColors()
  return {
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { top: 6, left: 'center', itemWidth: 10, itemHeight: 10, textStyle: { color: c.textSecondary, fontWeight: 700 } },
    grid: { left: 12, right: 12, top: 34, bottom: 12, containLabel: true },
    xAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { color: c.textTertiary, formatter: '{value}%' },
      splitLine: { lineStyle: { color: c.splitLine } }
    },
    yAxis: {
      type: 'category',
      data: rows.map((r) => r.name),
      axisLabel: { color: c.textTertiary, width: 120, overflow: 'truncate' },
      axisTick: { show: false },
      axisLine: { show: false }
    },
    series: [
      {
        type: 'bar',
        name: 'CPU',
        data: rows.map((r) => Math.max(0, Math.min(100, Number(r.cpu ?? 0)))),
        barWidth: 10,
        itemStyle: { borderRadius: 999, color: 'rgba(34, 211, 238, 0.94)' }
      },
      {
        type: 'bar',
        name: '内存',
        data: rows.map((r) => Math.max(0, Math.min(100, Number(r.mem ?? 0)))),
        barWidth: 10,
        itemStyle: { borderRadius: 999, color: 'rgba(59, 130, 246, 0.94)' }
      }
    ]
  }
}
