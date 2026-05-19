<template>
  <div class="kd">
    <!-- ─── Loading ─── -->
    <div v-if="loadingDashboard && !clusterOverview" class="kd-loading">
      <div class="kd-loading__spinner" />
      <span class="kd-loading__text">加载集群数据…</span>
    </div>

    <!-- ─── Empty ─── -->
    <template v-else-if="!clusterOverview">
      <div class="kd-empty-root">
        <el-icon :size="40" class="kd-empty-root__icon"><Connection /></el-icon>
        <p class="kd-empty-root__text">暂无集群概览数据</p>
        <button class="kd-btn kd-btn--primary" @click="emit('load-dashboard')">
          <el-icon :size="14"><RefreshRight /></el-icon>重新加载
        </button>
      </div>
    </template>

    <template v-else>
        <!-- ──── Header ──── -->
        <header class="kd-header">
          <div class="kd-header__left">
            <h2 class="kd-header__title">{{ clusterOverview?.cluster?.name ?? '集群概览' }}</h2>
            <span class="kd-status" :class="statusClass">{{ clusterOverview?.cluster?.status ?? '-' }}</span>
          </div>
          <div class="kd-header__center">
            <div class="kd-header__metas">
              <div v-if="dashboardLastUpdatedAt" class="kd-meta">
                <el-icon :size="13"><Clock /></el-icon>
                <span>{{ new Date(dashboardLastUpdatedAt).toLocaleTimeString() }}</span>
              </div>
              <div v-if="dashboardNamespacesTotal != null" class="kd-meta">
                <el-icon :size="13"><Box /></el-icon>
                <span>{{ dashboardNamespacesTotal }} NS</span>
              </div>
              <div class="kd-meta">
                <el-icon :size="13"><Monitor /></el-icon>
                <span>Ready {{ dashboardReadyText }}</span>
              </div>
            </div>
            <div class="kd-header__controls">
              <el-switch
                :model-value="dashboardAutoRefresh"
                inline-prompt
                active-text="自动刷新"
                inactive-text="手动"
                @update:model-value="emit('update:auto-refresh', $event)"
              />
              <el-select
                :model-value="dashboardAutoRefreshSec"
                class="kd-refresh-select"
                size="small"
                :disabled="!dashboardAutoRefresh"
                @update:model-value="emit('update:auto-refresh-sec', Number($event || 30))"
              >
                <el-option :value="15" label="15 秒" />
                <el-option :value="30" label="30 秒" />
                <el-option :value="60" label="60 秒" />
                <el-option :value="120" label="120 秒" />
              </el-select>
              <button class="kd-btn kd-btn--ghost" @click="emit('load-dashboard')">
                <el-icon :size="14"><RefreshRight /></el-icon>立即刷新
              </button>
            </div>
          </div>
          <div class="kd-header__right">
            <div class="kd-score" :class="healthTier">
              <svg viewBox="0 0 44 44" class="kd-score__ring">
                <circle cx="22" cy="22" r="18" fill="none" stroke="currentColor" stroke-width="3" opacity="0.1" />
                <circle cx="22" cy="22" r="18" fill="none" stroke="currentColor" stroke-width="3"
                  stroke-linecap="round" :stroke-dasharray="`${healthArc} 999`"
                  transform="rotate(-90 22 22)" class="kd-score__arc" />
              </svg>
              <span class="kd-score__num">{{ dashboardHealthScore }}</span>
            </div>
            <div class="kd-score__info">
              <span class="kd-score__label">健康评分</span>
              <span class="kd-score__hint" :class="healthTier">{{ healthHint }}</span>
            </div>
          </div>
        </header>

        <section class="kd-health-grid">
          <div class="kd-health-card">
            <div class="kd-health-card__title">健康拆解</div>
            <div class="kd-health-list">
              <div v-for="item in healthBreakdown" :key="item.key" class="kd-health-item">
                <div class="kd-health-item__meta">
                  <span>{{ item.label }}</span>
                  <strong :class="item.tier">{{ item.score }}</strong>
                </div>
                <div class="kd-health-bar"><span :class="item.tier" :style="{ width: `${item.score}%` }" /></div>
                <div class="kd-health-item__hint">{{ item.hint }}</div>
              </div>
            </div>
          </div>
          <div class="kd-health-card kd-health-card--compact">
            <div class="kd-health-card__title">当前关注项</div>
            <div class="kd-focus-grid">
              <button class="kd-focus-chip" @click="emit('navigate-resource', 'cluster:nodes')">
                <span>未就绪节点</span>
                <strong>{{ unreadyNodes }}</strong>
              </button>
              <button class="kd-focus-chip" @click="emit('navigate-resource', 'workloads:pods')">
                <span>Pending Pods</span>
                <strong>{{ pendingPods }}</strong>
              </button>
              <button class="kd-focus-chip" @click="emit('navigate-resource', 'workloads:pods')">
                <span>失败 Pods</span>
                <strong>{{ failedPods.length }}</strong>
              </button>
              <button class="kd-focus-chip" @click="emit('navigate-resource', 'misc:events')">
                <span>Warning 事件</span>
                <strong>{{ recentAlertEvents.length }}</strong>
              </button>
              <button class="kd-focus-chip" @click="scrollToCertSection">
                <span>证书预警</span>
                <strong>{{ certRiskSummaryText }}</strong>
              </button>
              <button class="kd-focus-chip" @click="emit('navigate-resource', 'workloads:deployments')">
                <span>工作负载总量</span>
                <strong>{{ workloadsTotal }}</strong>
              </button>
            </div>
          </div>
        </section>

        <!-- ──── KPI Cards ──── -->
        <section v-if="dashboardEnabledWidgets.includes('core') && clusterOverview?.stats" class="kd-kpi-grid">
          <div class="kd-kpi-card kd-kpi-card--cyan" @click="onKpiClick('nodes')">
            <div class="kd-kpi-card__icon"><el-icon :size="18"><Monitor /></el-icon></div>
            <div class="kd-kpi-card__body">
              <span class="kd-kpi-card__label">节点</span>
              <span class="kd-kpi-card__value">{{ clusterOverview.stats.nodes?.total ?? 0 }}</span>
            </div>
            <span class="kd-kpi-card__sub">Ready {{ clusterOverview.stats.nodes?.ready ?? 0 }}/{{ clusterOverview.stats.nodes?.total ?? 0 }}</span>
            <el-icon class="kd-kpi-card__arrow" :size="12"><ArrowRight /></el-icon>
          </div>
          <div class="kd-kpi-card kd-kpi-card--blue" @click="onKpiClick('pods')">
            <div class="kd-kpi-card__icon"><el-icon :size="18"><Cloudy /></el-icon></div>
            <div class="kd-kpi-card__body">
              <span class="kd-kpi-card__label">Pods</span>
              <span class="kd-kpi-card__value">{{ clusterOverview.stats.pods?.total ?? 0 }}</span>
            </div>
            <span class="kd-kpi-card__sub">Running {{ clusterOverview.stats.pods?.running ?? 0 }}</span>
            <el-icon class="kd-kpi-card__arrow" :size="12"><ArrowRight /></el-icon>
          </div>
          <div class="kd-kpi-card kd-kpi-card--violet" @click="onKpiClick('cpu')">
            <div class="kd-kpi-card__icon"><el-icon :size="18"><Cpu /></el-icon></div>
            <div class="kd-kpi-card__body">
              <span class="kd-kpi-card__label">CPU</span>
              <span class="kd-kpi-card__value">{{ dashboardCpuText || '-' }}</span>
            </div>
            <span class="kd-kpi-card__sub">使用率</span>
            <el-icon class="kd-kpi-card__arrow" :size="12"><ArrowRight /></el-icon>
          </div>
          <div class="kd-kpi-card kd-kpi-card--amber" @click="onKpiClick('mem')">
            <div class="kd-kpi-card__icon"><el-icon :size="18"><Coin /></el-icon></div>
            <div class="kd-kpi-card__body">
              <span class="kd-kpi-card__label">内存</span>
              <span class="kd-kpi-card__value">{{ dashboardMemoryText || '-' }}</span>
            </div>
            <span class="kd-kpi-card__sub">使用率</span>
            <el-icon class="kd-kpi-card__arrow" :size="12"><ArrowRight /></el-icon>
          </div>
        </section>

        <!-- ──── Charts Grid ──── -->
        <section v-if="clusterOverview?.charts" class="kd-charts-grid">
          <!-- CPU / Memory 24h Trend -->
          <div v-if="dashboardEnabledWidgets.includes('resourceTrend')" class="kd-chart-card kd-chart-card--wide">
            <div class="kd-chart-card__header">
              <div class="kd-chart-card__title">
                <el-icon :size="14" style="color:var(--c-cyan)"><TrendCharts /></el-icon>
                <span>CPU / 内存 24h 趋势</span>
              </div>
              <span class="kd-chart-card__badge">近 24 小时</span>
            </div>
            <div ref="cpuMemChartEl" class="kd-chart-card__canvas" />
          </div>

          <!-- Pod Phase -->
          <div v-if="dashboardEnabledWidgets.includes('podPhase')" class="kd-chart-card">
            <div class="kd-chart-card__header">
              <div class="kd-chart-card__title">
                <el-icon :size="14" style="color:var(--c-blue)"><PieChart /></el-icon>
                <span>Pod 状态分布</span>
              </div>
              <span class="kd-chart-card__badge">{{ clusterOverview.stats.pods?.total ?? 0 }} 总计</span>
            </div>
            <div ref="podPhaseChartEl" class="kd-chart-card__canvas" />
          </div>

          <!-- Workload Distribution -->
          <div v-if="dashboardEnabledWidgets.includes('workloadHealth')" class="kd-chart-card">
            <div class="kd-chart-card__header">
              <div class="kd-chart-card__title">
                <el-icon :size="14" style="color:var(--c-violet)"><SetUp /></el-icon>
                <span>工作负载分布</span>
              </div>
              <span class="kd-chart-card__badge">{{ workloadsTotal }} 总计</span>
            </div>
            <div ref="workloadChartEl" class="kd-chart-card__canvas" />
          </div>

          <!-- Namespace Top Pods -->
          <div v-if="dashboardEnabledWidgets.includes('nsTop')" class="kd-chart-card kd-chart-card--wide">
            <div class="kd-chart-card__header">
              <div class="kd-chart-card__title">
                <el-icon :size="14" style="color:var(--c-amber)"><Histogram /></el-icon>
                <span>Namespace Pod 排行</span>
              </div>
              <span class="kd-chart-card__badge">Top {{ clusterOverview.charts.namespace_pods_top?.length ?? 0 }}</span>
            </div>
            <div ref="nsTopChartEl" class="kd-chart-card__canvas" />
          </div>

          <!-- Node Health -->
          <div v-if="dashboardEnabledWidgets.includes('nodeReady')" class="kd-chart-card">
            <div class="kd-chart-card__header">
              <div class="kd-chart-card__title">
                <el-icon :size="14" style="color:var(--c-green)"><CircleCheck /></el-icon>
                <span>节点就绪率</span>
              </div>
              <span class="kd-chart-card__badge">{{ clusterOverview.charts.node_ready?.ready ?? 0 }}/{{ clusterOverview.charts.node_ready?.total ?? 0 }}</span>
            </div>
            <div ref="nodeReadyChartEl" class="kd-chart-card__canvas" />
          </div>
        </section>

        <!-- ──── Failed Pods ──── -->
        <section v-if="failedPods.length > 0" class="kd-fpods">
          <div class="kd-fpods__bar">
            <div class="kd-fpods__title">
              <el-icon :size="15" class="kd-fpods__ic"><CircleClose /></el-icon>
              <span>异常 Pod</span>
              <span class="kd-fpods__cnt">{{ failedPods.length }}</span>
            </div>
          </div>
          <div class="kd-fpods__list">
            <button v-for="fp in failedPods" :key="fp.namespace + '/' + fp.name" class="kd-fpod" @click="openFailedPod(fp)">
              <span class="kd-fpod__sev" />
              <span class="kd-fpod__ns">{{ fp.namespace }}</span>
              <span class="kd-fpod__name">{{ fp.name }}</span>
              <span class="kd-fpod__reason">{{ fp.reason }}</span>
            </button>
          </div>
        </section>

        <!-- ──── Alerts ──── -->
        <section v-if="dashboardEnabledWidgets.includes('alerts') && recentAlertEvents.length > 0" class="kd-alerts">
          <div class="kd-alerts__bar">
            <div class="kd-alerts__title">
              <el-icon :size="15" class="kd-alerts__ic"><WarningFilled /></el-icon>
              <span>告警事件</span>
              <span class="kd-alerts__cnt">{{ recentAlertEvents.length }}</span>
            </div>
            <div class="kd-alerts__tags">
              <span v-if="dashboardAlertCounts.critical" class="kd-atag kd-atag--crit">
                <i class="kd-atag__dot" />{{ dashboardAlertCounts.critical }} 严重
              </span>
              <span v-if="dashboardAlertCounts.warning" class="kd-atag kd-atag--warn">
                <i class="kd-atag__dot" />{{ dashboardAlertCounts.warning }} 警告
              </span>
            </div>
          </div>
          <div class="kd-alerts__list">
            <div
              v-for="(e, idx) in recentAlertEvents"
              :key="eventKey(e, idx)"
              class="kd-arow"
              :class="[alertRowClass(e), { 'kd-arow--open': expandedAlertIdx === idx }]"
              @click="expandedAlertIdx = expandedAlertIdx === idx ? -1 : idx"
            >
              <span class="kd-arow__sev" />
              <el-tooltip :content="String(e?.reason ?? '-')" placement="top" :show-after="200" :offset="4">
                <span class="kd-arow__reason">{{ String(e?.reason ?? '-') }}</span>
              </el-tooltip>
              <span class="kd-arow__msg">{{ String(e?.message ?? '-') }}</span>
              <span class="kd-arow__time">{{ alertTimeAgo(e) }}</span>
              <!-- expanded detail -->
                <div v-if="expandedAlertIdx === idx" class="kd-arow__detail" @click.stop>
                  <div class="kd-arow__detail-label">完整信息</div>
                  <div class="kd-arow__code" v-html="highlightAlertMsg(String(e?.message ?? '-'))" />
                  <div class="kd-arow__tags">
                  <span v-if="e?.involvedObject?.name" class="kd-arow__tag kd-arow__tag--res">
                    <i class="kd-arow__tag-k">资源</i>{{ e.involvedObject.namespace }}/{{ e.involvedObject.name }}
                  </span>
                  <span v-if="e?.count" class="kd-arow__tag kd-arow__tag--cnt">
                    <i class="kd-arow__tag-k">次数</i>{{ e.count }}
                  </span>
                  <span v-if="e?.source?.component" class="kd-arow__tag kd-arow__tag--src">
                    <i class="kd-arow__tag-k">来源</i>{{ e.source.component }}
                  </span>
                    <span v-if="e?.lastTimestamp" class="kd-arow__tag kd-arow__tag--time">
                      <i class="kd-arow__tag-k">最近</i>{{ new Date(e.lastTimestamp).toLocaleString() }}
                    </span>
                  </div>
                  <div class="kd-arow__actions">
                    <button v-if="eventResourceTarget(e)" class="kd-btn kd-btn--ghost" @click="openEventResource(e)">定位资源</button>
                  </div>
                </div>
              </div>
            </div>
        </section>

        <!-- ──── Cert Risk ──── -->
        <section v-if="dashboardEnabledWidgets.includes('certRisk') && clusterCertRows.length > 0" ref="certSectionRef" class="kd-certs">
          <div class="kd-certs__bar">
            <div class="kd-certs__title">
              <el-icon :size="15"><Lock /></el-icon>
              <span>证书风险</span>
              <span class="kd-certs__cnt">{{ clusterCertRows.length }}</span>
            </div>
            <div class="kd-certs__tags">
              <span v-if="clusterCertCriticalCount" class="kd-atag kd-atag--crit">
                <i class="kd-atag__dot" />{{ clusterCertCriticalCount }} 紧急
              </span>
              <span v-if="clusterCertWarnCount" class="kd-atag kd-atag--warn">
                <i class="kd-atag__dot" />{{ clusterCertWarnCount }} 预警
              </span>
            </div>
          </div>
          <div class="kd-certs__cards">
            <el-tooltip
              v-for="row in clusterCertRows"
              :key="row.key"
              placement="bottom"
              :show-after="300"
              :offset="6"
              popper-class="kd-cert-tip"
            >
              <template #content>
                <div class="kd-tip">
                  <div class="kd-tip__row"><span class="kd-tip__k">证书</span><span>{{ row.name }}</span></div>
                  <div class="kd-tip__row"><span class="kd-tip__k">组件</span><span>{{ row.component }}</span></div>
                  <div class="kd-tip__row"><span class="kd-tip__k">用途</span><span>{{ row.purpose }}</span></div>
                  <div v-if="row.not_after" class="kd-tip__row"><span class="kd-tip__k">过期</span><span>{{ row.not_after }}</span></div>
                  <div v-if="row.days_left != null" class="kd-tip__row"><span class="kd-tip__k">剩余</span><span>{{ row.days_left.toLocaleString() }} 天</span></div>
                </div>
              </template>
              <div class="kd-cc" :class="`kd-cc--${row.status}`">
                <div class="kd-cc__bar" />
                <div class="kd-cc__body">
                  <span class="kd-cc__name">{{ row.name }}</span>
                  <span class="kd-cc__comp">{{ row.component }}</span>
                </div>
                <div class="kd-cc__right">
                  <template v-if="row.days_left != null">
                    <span class="kd-cc__num">{{ row.days_left.toLocaleString() }}</span>
                    <span class="kd-cc__unit">天</span>
                  </template>
                  <template v-else>
                    <span class="kd-cc__num kd-cc__num--na">—</span>
                  </template>
                </div>
                <span class="kd-cc__badge">
                  <i class="kd-cc__dot" />
                  {{ certStatusLabel(row.status) }}
                </span>
              </div>
            </el-tooltip>
          </div>
        </section>

    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import {
  Monitor, Box, Clock, Connection, RefreshRight, WarningFilled, Lock,
  Cpu, Coin, Cloudy, ArrowRight, TrendCharts, PieChart, SetUp,
  Histogram, CircleCheck, CircleClose,
} from '@element-plus/icons-vue'
import { echarts, type EChartsOption } from '@/shared/utils/echarts'
const props = defineProps<{
  clusterId: number
  clusterOverview: any
  loadingDashboard: boolean
  dashboardNamespacesTotal: number | null
  dashboardEvents: any[]
  dashboardLastUpdatedAt: number | null
  dashboardCertRiskLoading: boolean
  dashboardCertRiskUnavailable: boolean
  dashboardAlertCounts: { critical: number; warning: number; info: number }
  dashboardHealthScore: number
  dashboardAutoRefresh: boolean
  dashboardAutoRefreshSec: number
  dashboardCpuText: string
  dashboardMemoryText: string
  dashboardReadyText: string
  dashboardEnabledWidgets: string[]
}>()

const emit = defineEmits<{
  (e: 'load-dashboard'): void
  (e: 'navigate-resource', treeNodeId: string): void
  (e: 'navigate-object', payload: { kind?: string; namespace?: string; name?: string }): void
  (e: 'update:auto-refresh', value: boolean): void
  (e: 'update:auto-refresh-sec', value: number): void
}>()

/* ── Resource navigation map ── */
const kpiNavMap: Record<string, string> = {
  nodes: 'cluster:nodes',
  pods:  'workloads:pods',
  cpu:   'cluster:nodes',
  mem:   'cluster:nodes',
}
function onKpiClick(key: string) {
  const target = kpiNavMap[key]
  if (target) emit('navigate-resource', target)
}
/* ── Health ── */
const healthArc = computed(() => (Math.max(0, Math.min(100, props.dashboardHealthScore)) / 100) * 113.1)
const healthTier = computed(() => {
  const s = props.dashboardHealthScore
  return s >= 80 ? 'kd--good' : s >= 60 ? 'kd--warn' : 'kd--bad'
})
const healthHint = computed(() => {
  const s = props.dashboardHealthScore
  return s >= 90 ? '运行优秀' : s >= 80 ? '运行良好' : s >= 60 ? '需要关注' : '存在风险'
})
const nodeTotal = computed(() => Number(props.clusterOverview?.stats?.nodes?.total ?? 0))
const nodeReady = computed(() => Number(props.clusterOverview?.stats?.nodes?.ready ?? 0))
const unreadyNodes = computed(() => Math.max(0, nodeTotal.value - nodeReady.value))
const pendingPods = computed(() => Number(props.clusterOverview?.stats?.pods?.pending ?? 0))
const failedPodCount = computed(() => Number(props.clusterOverview?.stats?.pods?.failed ?? 0))
const healthBreakdown = computed(() => {
  const apiScore = String(props.clusterOverview?.cluster?.status ?? '').toLowerCase().includes('degrad') ? 60 : 100
  const nodeScore = nodeTotal.value > 0 ? Math.round((nodeReady.value / Math.max(1, nodeTotal.value)) * 100) : 0
  const podTotal = Math.max(1, Number(props.clusterOverview?.stats?.pods?.total ?? 0))
  const podRunning = Number(props.clusterOverview?.stats?.pods?.running ?? 0)
  const podScore = Math.max(0, Math.min(100, Math.round((podRunning / podTotal) * 100)))
  const cpuUsed = Math.max(0, Math.min(100, Number(props.clusterOverview?.stats?.cpu?.used_percent ?? 0)))
  const memUsed = Math.max(0, Math.min(100, Number(props.clusterOverview?.stats?.memory?.used_percent ?? 0)))
  const pressureScore = Math.max(0, Math.round(100 - (cpuUsed + memUsed) / 2))
  const warningCount = recentAlertEvents.value.length
  const riskCount = clusterCertCriticalCount.value * 2 + clusterCertWarnCount.value
  const riskScore = Math.max(0, 100 - Math.min(80, warningCount * 12 + riskCount * 10 + failedPodCount.value * 8))
  const riskHint = props.dashboardCertRiskLoading && clusterCertRows.value.length === 0
    ? `Warning ${warningCount} / 证书检测中`
    : props.dashboardCertRiskUnavailable && clusterCertRows.value.length === 0
      ? `Warning ${warningCount} / 证书状态未知`
    : clusterCertUnknownCount.value > 0 && riskCount === 0
      ? `Warning ${warningCount} / 证书状态未知 ${clusterCertUnknownCount.value}`
      : `Warning ${warningCount} / 证书风险 ${riskCount}`
  const items = [
    { key: 'api', label: 'API / 控制面', score: apiScore, hint: apiScore >= 100 ? '接口连通正常' : '集群处于降级状态' },
    { key: 'nodes', label: '节点可用性', score: nodeScore, hint: `${nodeReady.value}/${nodeTotal.value} Ready` },
    { key: 'pods', label: 'Pod 运行率', score: podScore, hint: `${podRunning}/${podTotal} Running` },
    { key: 'pressure', label: '资源压力', score: pressureScore, hint: `CPU ${props.dashboardCpuText} / 内存 ${props.dashboardMemoryText}` },
    { key: 'risk', label: '事件与风险', score: riskScore, hint: riskHint },
  ]
  return items.map((item) => ({
    ...item,
    tier: item.score >= 80 ? 'kd--good' : item.score >= 60 ? 'kd--warn' : 'kd--bad'
  }))
})
const statusClass = computed(() => {
  const v = String(props.clusterOverview?.cluster?.status ?? '').toLowerCase()
  if (v.includes('active') || v.includes('ready') || v.includes('ok')) return 'kd-status--ok'
  if (v.includes('degrad') || v.includes('warn')) return 'kd-status--warn'
  return 'kd-status--bad'
})

/* ── Cert ── */
type CertStatus = 'ok' | 'warn' | 'critical' | 'unknown'
type ClusterCertRow = { key: string; name: string; component: string; purpose: string; not_before?: string; not_after?: string; days_left?: number; status: CertStatus }
const clusterCertRows = computed<ClusterCertRow[]>(() => {
  const raw = (props.clusterOverview?.risks?.certificates ?? []) as unknown
  const list = Array.isArray(raw) ? (raw as any[]) : []
  return list.map(r => ({
    key: String(r?.key ?? r?.name ?? ''), name: String(r?.name ?? '-'),
    component: String(r?.component ?? '-'), purpose: String(r?.purpose ?? '-'),
    not_before: r?.not_before != null ? String(r.not_before) : undefined,
    not_after: r?.not_after != null ? String(r.not_after) : undefined,
    days_left: r?.days_left != null && Number.isFinite(Number(r.days_left)) ? Number(r.days_left) : undefined,
    status: (['ok','warn','critical','unknown'].includes(String(r?.status)) ? String(r.status) : 'unknown') as CertStatus,
  })).sort((a, b) => {
    const aK = a.days_left != null, bK = b.days_left != null
    if (aK !== bK) return aK ? -1 : 1
    if (aK && bK && a.days_left !== b.days_left) return a.days_left! - b.days_left!
    return String(a.key).localeCompare(String(b.key))
  })
})
const clusterCertCriticalCount = computed(() => clusterCertRows.value.filter(r => r.status === 'critical').length)
const clusterCertWarnCount = computed(() => clusterCertRows.value.filter(r => r.status === 'warn').length)
const clusterCertUnknownCount = computed(() => clusterCertRows.value.filter(r => r.status === 'unknown').length)
const certRiskSummaryText = computed(() => {
  if (props.dashboardCertRiskLoading && clusterCertRows.value.length === 0) {
    return '检测中'
  }
  if (props.dashboardCertRiskUnavailable && clusterCertRows.value.length === 0) {
    return '未知'
  }
  if (clusterCertUnknownCount.value > 0 && clusterCertWarnCount.value + clusterCertCriticalCount.value === 0) {
    return '未知'
  }
  return String(clusterCertWarnCount.value + clusterCertCriticalCount.value)
})

/* ── Events ── */
const recentAlertEvents = computed(() => {
  const all = Array.isArray(props.dashboardEvents) ? props.dashboardEvents : []
  const w = all.filter((e: any) => String(e?.type ?? '') === 'Warning')
  return (w.length ? w : all).slice(0, 5)
})
/* ── Cert helpers ── */
function certStatusLabel(s: string): string {
  if (s === 'critical') return '紧急'
  if (s === 'warn') return '预警'
  if (s === 'ok') return '正常'
  return '未知'
}
/* ── Alert state ── */
const expandedAlertIdx = ref(-1)
const certSectionRef = ref<HTMLElement | null>(null)

function scrollToCertSection() {
  nextTick(() => {
    certSectionRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  })
}

/* ── Alert helpers ── */
function highlightAlertMsg(raw: string): string {
  let s = raw.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  // error keywords → red bold
  s = s.replace(/\b(error|Error|ERROR|failed|Failed|FAILED|refused|timeout|Timeout|exceeded|unhealthy|Unhealthy|OOMKilled|CrashLoopBackOff|DeadlineExceeded)\b/g, '<b class="kd-hl--err">$1</b>')
  // JSON keys "key": → dim
  s = s.replace(/(&quot;|")(\w[\w./-]*)(&quot;|")(\s*:)/g, '<span class="kd-hl--dim">$1$2$3</span>$4')
  return s
}

const criticalReasons = new Set(['Unhealthy', 'OOMKilling', 'OOMKilled', 'CrashLoopBackOff', 'FailedScheduling', 'FailedMount', 'EvictionThresholdMet', 'NodeNotReady'])
function alertRowClass(e: any): string {
  const reason = String(e?.reason ?? '')
  return criticalReasons.has(reason) ? 'kd-arow--crit' : 'kd-arow--warn'
}
function alertTimeAgo(e: any): string {
  const ts = e?.lastTimestamp || e?.eventTime || e?.metadata?.creationTimestamp
  if (!ts) return ''
  const diff = Date.now() - new Date(String(ts)).getTime()
  if (diff < 0 || !Number.isFinite(diff)) return ''
  const m = Math.floor(diff / 60000)
  if (m < 1) return '刚刚'
  if (m < 60) return `${m}分钟前`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}小时前`
  return `${Math.floor(h / 24)}天前`
}

/* ── Helpers ── */
function eventKey(e: any, idx: number): string {
  const uid = String(e?.metadata?.uid ?? '')
  if (uid) return uid
  return [e?.metadata?.namespace, e?.metadata?.name, e?.lastTimestamp, e?.reason, idx].filter(Boolean).join('|') || String(idx)
}

/* ── Failed Pods (from anomalies) ── */
const failedPods = computed(() => {
  const raw = props.clusterOverview?.anomalies?.failed_pods
  if (!Array.isArray(raw)) return []
  return raw.slice(0, 10).map((p: any) => ({
    name: String(p?.name ?? '-'),
    namespace: String(p?.namespace ?? '-'),
    reason: String(p?.reason ?? 'Unknown'),
  }))
})
function openFailedPod(pod: { namespace: string; name: string }) {
  emit('navigate-object', { kind: 'Pod', namespace: pod.namespace, name: pod.name })
}
function eventResourceTarget(e: any) {
  const kind = String(e?.involvedObject?.kind ?? '').trim()
  const name = String(e?.involvedObject?.name ?? '').trim()
  if (!kind || !name) return null
  return {
    kind,
    namespace: String(e?.involvedObject?.namespace ?? '').trim() || undefined,
    name,
  }
}
function openEventResource(e: any) {
  const target = eventResourceTarget(e)
  if (target) emit('navigate-object', target)
}

/* ══════════════════ CHARTS ══════════════════ */
const cpuMemChartEl = ref<HTMLDivElement | null>(null)
const podPhaseChartEl = ref<HTMLDivElement | null>(null)
const workloadChartEl = ref<HTMLDivElement | null>(null)
const nsTopChartEl = ref<HTMLDivElement | null>(null)
const nodeReadyChartEl = ref<HTMLDivElement | null>(null)

const workloadsTotal = computed(() => {
  const w = props.clusterOverview?.stats?.workloads
  if (!w) return 0
  return Number(w.deployments ?? 0) + Number(w.statefulsets ?? 0) + Number(w.daemonsets ?? 0)
})

/* ── Dark mode detect ── */
function isDark() { return document.documentElement.classList.contains('dark') }

/* ── Color context ── */
function cc() {
  const dark = isDark()
  return {
    tipBg: dark ? '#1e293b' : '#fff',
    tipBorder: dark ? '#334155' : '#e2e8f0',
    tipText: dark ? '#e2e8f0' : '#334155',
    legend: dark ? '#94a3b8' : '#64748b',
    axisLabel: dark ? '#64748b' : '#94a3b8',
    split: dark ? 'rgba(148,163,184,0.08)' : 'rgba(148,163,184,0.15)',
    splitSub: dark ? 'rgba(148,163,184,0.06)' : 'rgba(148,163,184,0.1)',
    pieBorder: dark ? '#0f172a' : '#fff',
  }
}

/* ── Chart instance management ── */
const chartInst = new Map<string, echarts.ECharts>()
const chartObs = new Map<string, ResizeObserver>()

function ensureChart(key: string, el: HTMLDivElement): echarts.ECharts {
  let inst = chartInst.get(key)
  if (inst) return inst
  inst = echarts.init(el)
  chartInst.set(key, inst)
  const ro = new ResizeObserver(() => inst!.resize())
  ro.observe(el)
  chartObs.set(key, ro)
  return inst
}
function setOpt(key: string, el: HTMLDivElement, opt: EChartsOption) {
  ensureChart(key, el).setOption(opt, { notMerge: true, lazyUpdate: true })
}
function disposeCharts() {
  chartObs.forEach(ro => ro.disconnect())
  chartObs.clear()
  chartInst.forEach(inst => inst.dispose())
  chartInst.clear()
}

/* ── Option Builders ── */

function buildCpuMemOption(): EChartsOption {
  const ch = props.clusterOverview?.charts?.cpu_memory_24h
  if (!ch) return {}
  const c = cc()
  return {
    color: ['#0891b2', '#7c3aed'],
    tooltip: {
      trigger: 'axis',
      backgroundColor: c.tipBg, borderColor: c.tipBorder,
      textStyle: { color: c.tipText, fontSize: 12 }, padding: [8, 12],
      formatter(params: any) {
        if (!Array.isArray(params) || !params.length) return ''
        let s = `<div style="font-weight:700;margin-bottom:4px">${params[0].axisValueLabel}</div>`
        for (const p of params) s += `<div style="display:flex;align-items:center;gap:6px">${p.marker}<span>${p.seriesName}</span><b style="margin-left:auto">${Number(p.value).toFixed(1)}%</b></div>`
        return s
      }
    },
    legend: { top: 0, right: 0, icon: 'circle', itemWidth: 8, itemGap: 16, textStyle: { color: c.legend, fontSize: 11 } },
    grid: { left: 8, right: 16, top: 36, bottom: 8, containLabel: true },
    xAxis: {
      type: 'category', data: ch.labels ?? [], boundaryGap: false,
      axisLabel: { color: c.axisLabel, fontSize: 10, interval: 'auto', rotate: 0, formatter: (v: string) => v.length > 5 ? v.slice(-5) : v },
      axisLine: { show: false }, axisTick: { show: false }
    },
    yAxis: {
      type: 'value', min: 0, max: 100,
      axisLabel: { color: c.axisLabel, fontSize: 10, formatter: '{value}%' },
      splitLine: { lineStyle: { color: c.split, type: 'dashed' } }
    },
    series: [
      {
        name: 'CPU', type: 'line', smooth: 0.35, showSymbol: false, lineStyle: { width: 2 },
        markLine: {
          silent: true, symbol: 'none',
          lineStyle: { type: 'dashed', width: 1 },
          label: { fontSize: 9, position: 'insideEndTop' },
          data: [
            { yAxis: 80, lineStyle: { color: '#d97706' }, label: { formatter: '警告 80%', color: '#d97706' } },
            { yAxis: 90, lineStyle: { color: '#dc2626' }, label: { formatter: '危险 90%', color: '#dc2626' } }
          ]
        },
        areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: 'rgba(8,145,178,0.20)' }, { offset: 1, color: 'rgba(8,145,178,0)' }]) },
        data: ch.cpu ?? []
      },
      {
        name: '内存', type: 'line', smooth: 0.35, showSymbol: false, lineStyle: { width: 2 },
        areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: 'rgba(124,58,237,0.18)' }, { offset: 1, color: 'rgba(124,58,237,0)' }]) },
        data: ch.memory ?? []
      }
    ]
  }
}

function buildPodPhaseOption(): EChartsOption {
  const p = props.clusterOverview?.charts?.pod_phase
  if (!p) return {}
  const c = cc()
  const total = (p.running ?? 0) + (p.pending ?? 0) + (p.failed ?? 0) + (p.succeeded ?? 0)
  return {
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)', backgroundColor: c.tipBg, borderColor: c.tipBorder, textStyle: { color: c.tipText, fontSize: 12 }, padding: [8, 12] },
    legend: { bottom: 0, icon: 'circle', itemWidth: 8, itemGap: 12, textStyle: { color: c.legend, fontSize: 11 } },
    color: ['#10b981', '#f59e0b', '#ef4444', '#94a3b8'],
    series: [{
      name: 'Pod', type: 'pie', radius: ['46%', '70%'], center: ['50%', '42%'],
      itemStyle: { borderRadius: 4, borderColor: c.pieBorder, borderWidth: 2 },
      label: { show: false },
      emphasis: { label: { show: true, fontSize: 13, fontWeight: '600', formatter: '{b}\n{c}' }, scaleSize: 5 },
      data: [
        { name: 'Running', value: p.running ?? 0 },
        { name: 'Pending', value: p.pending ?? 0 },
        { name: 'Failed', value: p.failed ?? 0 },
        { name: 'Succeeded', value: p.succeeded ?? 0 }
      ].filter(d => d.value > 0)
    }],
    graphic: [{
      type: 'group', left: 'center', top: '36%',
      children: [
        { type: 'text', style: { text: String(total), fontSize: 22, fontWeight: 'bold', fill: isDark() ? '#f1f5f9' : '#1e293b', textAlign: 'center' }, left: 'center' },
        { type: 'text', style: { text: 'Pods', fontSize: 11, fill: '#94a3b8', textAlign: 'center' }, left: 'center', top: 28 }
      ]
    }]
  }
}

function buildWorkloadOption(): EChartsOption {
  const w = props.clusterOverview?.stats?.workloads
  if (!w) return {}
  const c = cc()
  const total = Number(w.deployments ?? 0) + Number(w.statefulsets ?? 0) + Number(w.daemonsets ?? 0)
  return {
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)', backgroundColor: c.tipBg, borderColor: c.tipBorder, textStyle: { color: c.tipText, fontSize: 12 }, padding: [8, 12] },
    legend: { bottom: 0, icon: 'circle', itemWidth: 8, itemGap: 12, textStyle: { color: c.legend, fontSize: 11 } },
    color: ['#3b82f6', '#8b5cf6', '#f97316'],
    series: [{
      name: 'Workload', type: 'pie', radius: ['46%', '70%'], center: ['50%', '42%'],
      itemStyle: { borderRadius: 4, borderColor: c.pieBorder, borderWidth: 2 },
      label: { show: false },
      emphasis: { label: { show: true, fontSize: 13, fontWeight: '600', formatter: '{b}\n{c}' }, scaleSize: 5 },
      data: [
        { name: 'Deployment', value: w.deployments ?? 0 },
        { name: 'StatefulSet', value: w.statefulsets ?? 0 },
        { name: 'DaemonSet', value: w.daemonsets ?? 0 }
      ].filter(d => d.value > 0)
    }],
    graphic: [{
      type: 'group', left: 'center', top: '36%',
      children: [
        { type: 'text', style: { text: String(total), fontSize: 22, fontWeight: 'bold', fill: isDark() ? '#f1f5f9' : '#1e293b', textAlign: 'center' }, left: 'center' },
        { type: 'text', style: { text: '负载', fontSize: 11, fill: '#94a3b8', textAlign: 'center' }, left: 'center', top: 28 }
      ]
    }]
  }
}

function buildNsTopOption(): EChartsOption {
  const top = props.clusterOverview?.charts?.namespace_pods_top
  if (!top || !top.length) return {}
  const c = cc()
  const sorted = [...top].sort((a, b) => a.pods - b.pods)
  return {
    tooltip: {
      trigger: 'axis', axisPointer: { type: 'shadow' },
      backgroundColor: c.tipBg, borderColor: c.tipBorder,
      textStyle: { color: c.tipText, fontSize: 12 }, padding: [8, 12]
    },
    grid: { left: 8, right: 24, top: 12, bottom: 8, containLabel: true },
    xAxis: {
      type: 'value', minInterval: 1,
      axisLabel: { color: c.axisLabel, fontSize: 10 },
      splitLine: { lineStyle: { color: c.split, type: 'dashed' } }
    },
    yAxis: {
      type: 'category', data: sorted.map(i => i.namespace),
      axisLabel: { color: c.axisLabel, fontSize: 11, width: 100, overflow: 'truncate' },
      axisLine: { show: false }, axisTick: { show: false }
    },
    series: [{
      type: 'bar', barMaxWidth: 20,
      itemStyle: {
        borderRadius: [0, 3, 3, 0],
        color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
          { offset: 0, color: '#d97706' },
          { offset: 1, color: '#fbbf24' }
        ])
      },
      label: { show: true, position: 'right', fontSize: 11, fontWeight: 700, color: c.legend },
      data: sorted.map(i => i.pods)
    }]
  }
}

function buildNodeReadyOption(): EChartsOption {
  const nr = props.clusterOverview?.charts?.node_ready
  if (!nr) return {}
  const total = Math.max(nr.total ?? 1, 1)
  const ready = nr.ready ?? 0
  const pct = Math.round((ready / total) * 100)
  const color = pct >= 90 ? '#059669' : pct >= 70 ? '#d97706' : '#dc2626'
  return {
    series: [{
      type: 'gauge',
      startAngle: 220, endAngle: -40,
      radius: '90%', center: ['50%', '55%'],
      min: 0, max: 100,
      progress: { show: true, width: 14, roundCap: true, itemStyle: { color } },
      axisLine: { lineStyle: { width: 14, color: [[1, isDark() ? 'rgba(148,163,184,0.08)' : 'rgba(148,163,184,0.12)']] } },
      axisTick: { show: false },
      splitLine: { show: false },
      axisLabel: { show: false },
      pointer: { show: false },
      title: { offsetCenter: [0, '30%'], fontSize: 12, fontWeight: 600, color: '#94a3b8' },
      detail: {
        offsetCenter: [0, '-8%'], fontSize: 28, fontWeight: 900,
        valueAnimation: true, color,
        formatter: '{value}%'
      },
      data: [{ value: pct, name: `${ready} / ${total} Ready` }]
    }]
  }
}

/* ── Render all charts ── */
async function renderCharts() {
  await nextTick()
  if (!props.clusterOverview?.charts) return
  if (cpuMemChartEl.value) setOpt('cpuMem', cpuMemChartEl.value, buildCpuMemOption())
  if (podPhaseChartEl.value) setOpt('podPhase', podPhaseChartEl.value, buildPodPhaseOption())
  if (workloadChartEl.value) setOpt('workload', workloadChartEl.value, buildWorkloadOption())
  if (nsTopChartEl.value) setOpt('nsTop', nsTopChartEl.value, buildNsTopOption())
  if (nodeReadyChartEl.value) setOpt('nodeReady', nodeReadyChartEl.value, buildNodeReadyOption())
}

/* ── Watch data changes → re-render ── */
watch(() => props.clusterOverview, () => { void renderCharts() }, { deep: true })
watch(() => props.dashboardEnabledWidgets, () => { void renderCharts() }, { deep: true })

onBeforeUnmount(() => { disposeCharts() })
</script>

<style scoped>
/*
 * K8s Cluster Dashboard — Clean Professional Design
 * Palette: neutral slate + 5 semantic accent colors
 * No emoji. Icons via @element-plus/icons-vue.
 */

/* ─── Tokens ─── */
.kd {
  --g: 12px;
  --r: 8px;
  --border: color-mix(in srgb, var(--app-title, #1e293b) 7%, transparent);
  --bg: var(--color-bg-card, #fff);
  --bg2: color-mix(in srgb, var(--app-title, #1e293b) 2.5%, var(--bg));
  --fg: var(--app-title, #1e293b);
  --fg2: var(--app-muted, #64748b);
  --c-cyan:   #0891b2;
  --c-blue:   #2563eb;
  --c-violet: #7c3aed;
  --c-amber:  #d97706;
  --c-green:  #059669;
  --c-red:    #dc2626;

  display: flex; flex-direction: column; gap: var(--g);
  padding: 0 0 20px;
  height: 100%; min-height: 0; overflow: auto;
  color: var(--fg); font-size: 13px;
}
:global(html.dark) .kd {
  --border: rgba(226, 232, 240, 0.06);
  --bg: rgba(15, 23, 42, 0.5);
  --bg2: rgba(15, 23, 42, 0.3);
}

/* ─── Loading ─── */
.kd-loading {
  display: flex; align-items: center; justify-content: center; gap: 10px;
  padding: 80px 0; color: var(--fg2); font-size: 13px; font-weight: 600;
}
.kd-loading__spinner {
  width: 20px; height: 20px; border: 2.5px solid var(--border);
  border-top-color: var(--c-cyan); border-radius: 50%;
  animation: spin .65s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg) } }

/* ─── Empty ─── */
.kd-empty-root {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 14px; padding: 80px 20px;
}
.kd-empty-root__icon { color: var(--fg2); opacity: .4; }
.kd-empty-root__text { font-size: 14px; font-weight: 600; color: var(--fg2); margin: 0; }

/* ─── Button ─── */
.kd-btn {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 7px 18px; border: none; border-radius: var(--r); cursor: pointer;
  font-size: 12px; font-weight: 700; transition: opacity .15s;
}
.kd-btn:hover { opacity: .85; }
.kd-btn--primary { background: var(--c-cyan); color: #fff; }

/* ═══════════════ HEADER ═══════════════ */
.kd-header {
  display: flex; align-items: center; gap: 14px;
  padding: 12px 18px;
  background: var(--bg); border: 1px solid var(--border); border-radius: var(--r);
}
.kd-header__left { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.kd-header__title { margin: 0; font-size: 16px; font-weight: 800; letter-spacing: -.01em; }

.kd-status {
  display: inline-block; font-size: 10px; font-weight: 800; letter-spacing: .03em;
  padding: 2px 10px; border-radius: 99px; text-transform: uppercase;
}
.kd-status--ok   { background: color-mix(in srgb, var(--c-green) 12%, transparent); color: var(--c-green); }
.kd-status--warn { background: color-mix(in srgb, var(--c-amber) 12%, transparent); color: var(--c-amber); }
.kd-status--bad  { background: color-mix(in srgb, var(--c-red) 12%, transparent);   color: var(--c-red); }

.kd-header__center { flex: 1; min-width: 0; }
.kd-header__metas { display: flex; gap: 16px; justify-content: center; }
.kd-meta {
  display: flex; align-items: center; gap: 4px;
  font-size: 11.5px; font-weight: 600; color: var(--fg2);
}
.kd-header__controls {
  display: flex; align-items: center; justify-content: center; gap: 10px;
  margin-top: 10px;
}
.kd-refresh-select { width: 96px; }

.kd-header__right { display: flex; align-items: center; gap: 12px; flex-shrink: 0; }
.kd-score { position: relative; width: 48px; height: 48px; }
.kd-score__ring { width: 100%; height: 100%; }
.kd-score__arc { transition: stroke-dasharray .5s ease; }
.kd-score__num {
  position: absolute; inset: 0; display: flex; align-items: center; justify-content: center;
  font-size: 14px; font-weight: 900;
}
.kd-score__info { display: flex; flex-direction: column; gap: 1px; }
.kd-score__label { font-size: 11px; font-weight: 700; color: var(--fg2); }
.kd-score__hint { font-size: 11px; font-weight: 700; }
.kd--good { color: var(--c-green); }
.kd--warn { color: var(--c-amber); }
.kd--bad  { color: var(--c-red); }
.kd-btn--ghost {
  background: color-mix(in srgb, var(--fg2) 8%, transparent);
  color: var(--fg);
}

.kd-health-grid {
  display: grid;
  grid-template-columns: 1.6fr 1fr;
  gap: 10px;
}
.kd-health-card {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--r);
  padding: 16px 18px;
}
.kd-health-card__title {
  font-size: 13px;
  font-weight: 800;
  margin-bottom: 12px;
}
.kd-health-list { display: flex; flex-direction: column; gap: 12px; }
.kd-health-item__meta {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 6px; font-size: 12px; font-weight: 700;
}
.kd-health-bar {
  width: 100%; height: 8px; border-radius: 999px; overflow: hidden;
  background: color-mix(in srgb, var(--fg2) 12%, transparent);
}
.kd-health-bar > span { display: block; height: 100%; border-radius: 999px; }
.kd-health-bar > .kd--good { background: var(--c-green); }
.kd-health-bar > .kd--warn { background: var(--c-amber); }
.kd-health-bar > .kd--bad { background: var(--c-red); }
.kd-health-item__hint { margin-top: 6px; color: var(--fg2); font-size: 11px; }
.kd-focus-grid {
  display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px;
}
.kd-focus-chip {
  border: 1px solid var(--border); background: var(--bg2); color: var(--fg);
  border-radius: 10px; padding: 12px; cursor: pointer;
  display: flex; flex-direction: column; align-items: flex-start; gap: 6px;
}
.kd-focus-chip span { font-size: 11px; color: var(--fg2); font-weight: 700; }
.kd-focus-chip strong { font-size: 20px; line-height: 1; }

/* --- KPI 4-card grid --- */
.kd-kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
}
.kd-kpi-card {
  display: flex; align-items: center; gap: 10px;
  padding: 14px 16px;
  background: var(--bg); border: 1px solid var(--border); border-radius: var(--r);
  cursor: pointer; user-select: none;
  transition: border-color .15s, box-shadow .15s;
  position: relative;
  border-bottom: 2px solid var(--_a, var(--c-cyan));
}
.kd-kpi-card:hover {
  border-color: color-mix(in srgb, var(--_a, var(--c-cyan)) 30%, var(--border));
  box-shadow: 0 2px 8px color-mix(in srgb, var(--_a, var(--c-cyan)) 10%, transparent);
}
.kd-kpi-card--cyan   { --_a: var(--c-cyan); }
.kd-kpi-card--blue   { --_a: var(--c-blue); }
.kd-kpi-card--violet { --_a: var(--c-violet); }
.kd-kpi-card--amber  { --_a: var(--c-amber); }

.kd-kpi-card__icon {
  width: 36px; height: 36px; border-radius: 8px; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  background: color-mix(in srgb, var(--_a, var(--c-cyan)) 10%, transparent);
  color: var(--_a);
}
.kd-kpi-card__body {
  display: flex; flex-direction: column; flex: 1; min-width: 0;
}
.kd-kpi-card__label {
  font-size: 11px; font-weight: 600; color: var(--fg2); line-height: 1.2;
}
.kd-kpi-card__value {
  font-size: 22px; font-weight: 900; letter-spacing: -.5px; line-height: 1.15;
  color: var(--_a);
}
.kd-kpi-card__sub {
  position: absolute; right: 14px; bottom: 6px;
  font-size: 10px; color: var(--fg2); white-space: nowrap;
}
.kd-kpi-card__arrow {
  position: absolute; right: 8px; top: 8px;
  color: var(--fg2); opacity: 0;
  transition: opacity .15s;
}
.kd-kpi-card:hover .kd-kpi-card__arrow { opacity: .6; }

/* ═══════════════ CHARTS GRID ═══════════════ */
.kd-charts-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}
.kd-chart-card {
  display: flex; flex-direction: column;
  background: var(--bg); border: 1px solid var(--border); border-radius: var(--r);
  padding: 14px 16px 10px;
  min-height: 0;
}
.kd-chart-card--wide {
  grid-column: span 2;
}
.kd-chart-card__header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 8px; flex-shrink: 0;
}
.kd-chart-card__title {
  display: flex; align-items: center; gap: 6px;
  font-size: 13px; font-weight: 800; color: var(--fg);
}
.kd-chart-card__badge {
  font-size: 11px; font-weight: 700; color: var(--fg2);
  padding: 2px 10px; border-radius: 10px;
  background: color-mix(in srgb, var(--fg2) 6%, transparent);
}
.kd-chart-card__canvas {
  flex: 1; min-height: 220px; width: 100%;
}

/* ═══════════════ FAILED PODS ═══════════════ */
.kd-fpods {
  padding: 18px 22px; border-radius: var(--r);
  border: 1px solid var(--border); background: var(--bg);
}
.kd-fpods__bar {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 12px; padding-bottom: 10px;
  border-bottom: 1px solid var(--border);
}
.kd-fpods__title {
  display: flex; align-items: center; gap: 8px;
  font-size: 14px; font-weight: 800; color: var(--fg);
}
.kd-fpods__ic { color: var(--c-red); }
.kd-fpods__cnt {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 22px; height: 22px; padding: 0 7px;
  border-radius: 11px; font-size: 11.5px; font-weight: 800;
  background: color-mix(in srgb, var(--c-red) 10%, transparent); color: var(--c-red);
}
.kd-fpods__list { display: flex; flex-direction: column; gap: 0; }
.kd-fpod {
  appearance: none;
  border: none;
  width: 100%;
  background: transparent;
  display: grid; grid-template-columns: 4px 120px 1fr auto; gap: 0 10px;
  align-items: center;
  padding: 8px 6px; font-size: 13px; line-height: 1.4;
  text-align: left;
  cursor: pointer;
}
.kd-fpod + .kd-fpod { border-top: 1px solid var(--border); }
.kd-fpod:hover { background: color-mix(in srgb, var(--fg2) 5%, transparent); }
.kd-fpod__sev {
  width: 3.5px; height: 18px; border-radius: 2px; background: #dc2626; flex-shrink: 0;
}
.kd-fpod__ns {
  font-size: 12px; font-weight: 700; color: var(--fg2);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.kd-fpod__name {
  font-size: 13px; font-weight: 600; color: var(--fg);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.kd-fpod__reason {
  font-size: 11.5px; font-weight: 700; padding: 2px 10px; border-radius: 4px;
  background: color-mix(in srgb, var(--c-red) 8%, transparent);
  color: #b91c1c; white-space: nowrap;
}
:global(html.dark) .kd-fpod__reason { color: #fca5a5; }

/* ═══════════════ ALERTS ═══════════════ */
.kd-alerts {
  padding: 18px 22px; border-radius: var(--r);
  border: 1px solid var(--border);
  background: var(--bg);
}
.kd-alerts__bar {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 14px; padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}
.kd-alerts__title {
  display: flex; align-items: center; gap: 8px;
  font-size: 14px; font-weight: 800; color: var(--fg);
}
.kd-alerts__ic { color: var(--c-red); }
.kd-alerts__cnt {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 22px; height: 22px; padding: 0 7px;
  border-radius: 11px; font-size: 11.5px; font-weight: 800;
  background: color-mix(in srgb, var(--c-red) 10%, transparent); color: var(--c-red);
}
.kd-alerts__tags { display: flex; gap: 10px; }

/* ── Alert severity tag ── */
.kd-atag {
  display: inline-flex; align-items: center; gap: 6px;
  font-size: 12px; font-weight: 700; padding: 4px 12px; border-radius: 12px;
}
.kd-atag__dot {
  display: inline-block; width: 7px; height: 7px; border-radius: 50%;
}
.kd-atag--crit {
  background: color-mix(in srgb, var(--c-red) 10%, transparent);
  color: #b91c1c;
}
.kd-atag--crit .kd-atag__dot { background: #dc2626; }
.kd-atag--warn {
  background: color-mix(in srgb, var(--c-amber) 10%, transparent);
  color: #b45309;
}
.kd-atag--warn .kd-atag__dot { background: #d97706; }
:global(html.dark) .kd-atag--crit { color: #fca5a5; }
:global(html.dark) .kd-atag--warn { color: #fcd34d; }

/* ── Alert list ── */
.kd-alerts__list { display: flex; flex-direction: column; gap: 0; }

/* ── Alert row ── */
.kd-arow {
  display: grid; grid-template-columns: 4px 140px 1fr auto; gap: 0 12px;
  align-items: center;
  padding: 10px 8px; border-radius: 6px;
  font-size: 13px; line-height: 1.5;
  cursor: pointer; user-select: none;
  transition: background .12s;
}
.kd-arow:hover { background: var(--bg2); }
.kd-arow--open { background: var(--bg2); }
.kd-arow + .kd-arow { border-top: 1px solid var(--border); }

/* severity indicator bar */
.kd-arow__sev {
  width: 3.5px; height: 22px; border-radius: 2px; flex-shrink: 0;
}
.kd-arow--crit .kd-arow__sev { background: #dc2626; }
.kd-arow--warn .kd-arow__sev { background: #f59e0b; }

/* reason badge — fixed width, centered */
.kd-arow__reason {
  font-size: 12.5px; font-weight: 800; letter-spacing: .01em;
  padding: 3px 0; border-radius: 4px; text-align: center;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.kd-arow--crit .kd-arow__reason {
  background: color-mix(in srgb, #dc2626 10%, transparent);
  color: #b91c1c;
}
.kd-arow--warn .kd-arow__reason {
  background: color-mix(in srgb, #f59e0b 10%, transparent);
  color: #92400e;
}
:global(html.dark) .kd-arow--crit .kd-arow__reason { color: #fca5a5; }
:global(html.dark) .kd-arow--warn .kd-arow__reason { color: #fcd34d; }

/* message text — single line */
.kd-arow__msg {
  color: var(--fg); font-size: 13px; font-weight: 500;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
:global(html.dark) .kd-arow__msg { color: #cbd5e1; }

/* time */
.kd-arow__time {
  font-size: 12px; color: var(--fg2); white-space: nowrap;
  font-weight: 500; font-variant-numeric: tabular-nums;
}

/* expanded detail area */
.kd-arow__detail {
  grid-column: 2 / -1;
  padding: 10px 0 4px;
  display: flex; flex-direction: column; gap: 8px;
  animation: kd-expand .18s ease;
  cursor: default; user-select: text;
}
@keyframes kd-expand {
  from { opacity: 0; transform: translateY(-6px); }
  to { opacity: 1; transform: none; }
}

/* detail label */
.kd-arow__detail-label {
  font-size: 11px; font-weight: 700; color: var(--fg2);
  text-transform: uppercase; letter-spacing: .05em;
}

/* terminal-style code block with syntax highlighting */
.kd-arow__code {
  margin: 0;
  font-family: ui-monospace, 'SFMono-Regular', Menlo, Consolas, monospace;
  font-size: 12px; line-height: 1.75;
  color: #cbd5e1; background: #0f172a;
  padding: 12px 16px; border-radius: 8px;
  white-space: pre-wrap; word-break: break-word;
  max-height: 220px; overflow: auto;
  border: 1px solid rgba(51, 65, 85, .5);
}
:global(html.dark) .kd-arow__code { background: #020617; border-color: rgba(51,65,85,.4); }

/* syntax highlight tokens (v-html needs :deep to pierce scoped) */
.kd-arow__code :deep(.kd-hl--err) { color: #f87171; font-weight: 700; }
.kd-arow__code :deep(.kd-hl--dim) { color: #64748b; }

/* meta tag chips — colored by type */
.kd-arow__tags {
  display: flex; flex-wrap: wrap; gap: 6px;
}
.kd-arow__actions {
  margin-top: 8px;
  display: flex;
  justify-content: flex-end;
}
.kd-arow__tag {
  --_tc: var(--fg2);
  display: inline-flex; align-items: center;
  font-size: 11.5px; font-weight: 600; color: var(--fg);
  padding: 4px 12px; border-radius: 6px;
  background: color-mix(in srgb, var(--_tc) 6%, var(--bg));
  border: 1px solid color-mix(in srgb, var(--_tc) 15%, var(--border));
}
.kd-arow__tag-k {
  font-style: normal; font-weight: 800; color: var(--_tc);
  margin-right: 6px; font-size: 10.5px; text-transform: uppercase; letter-spacing: .03em;
}
.kd-arow__tag--res  { --_tc: #2563eb; }
.kd-arow__tag--cnt  { --_tc: #d97706; }
.kd-arow__tag--src  { --_tc: #7c3aed; }
.kd-arow__tag--time { --_tc: #0891b2; }
:global(html.dark) .kd-arow__tag--res  { --_tc: #60a5fa; }
:global(html.dark) .kd-arow__tag--cnt  { --_tc: #fbbf24; }
:global(html.dark) .kd-arow__tag--src  { --_tc: #a78bfa; }
:global(html.dark) .kd-arow__tag--time { --_tc: #22d3ee; }

/* ═══════════════ CERT RISK ═══════════════ */
.kd-certs {
  padding: 18px 22px; border-radius: var(--r);
  border: 1px solid var(--border); background: var(--bg);
}
.kd-certs__bar {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 16px; padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}
.kd-certs__title {
  display: flex; align-items: center; gap: 8px;
  font-size: 14px; font-weight: 800; color: var(--fg);
}
.kd-certs__cnt {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 22px; height: 22px; padding: 0 7px;
  border-radius: 11px; font-size: 11.5px; font-weight: 800;
  background: color-mix(in srgb, var(--fg) 6%, transparent); color: var(--fg2);
}
.kd-certs__tags { display: flex; gap: 10px; }

/* ── Cert card list ── */
.kd-certs__cards {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 10px;
}

/* ── Single cert card ── */
.kd-cc {
  --_c: var(--fg2);
  display: flex; align-items: center; gap: 0;
  border: 1px solid var(--border); border-radius: 8px;
  background: var(--bg); overflow: hidden; cursor: default;
  transition: box-shadow .15s, border-color .15s;
}
.kd-cc:hover {
  border-color: color-mix(in srgb, var(--_c) 30%, var(--border));
  box-shadow: 0 2px 12px color-mix(in srgb, var(--_c) 6%, transparent);
}

/* left color bar */
.kd-cc__bar { width: 4px; align-self: stretch; flex-shrink: 0; background: var(--_c); }

/* body: name + component */
.kd-cc__body {
  flex: 1; min-width: 0; padding: 10px 14px;
  display: flex; flex-direction: column; gap: 2px;
}
.kd-cc__name {
  font-size: 13px; font-weight: 800; color: var(--fg);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.kd-cc__comp {
  font-size: 11.5px; font-weight: 600; color: var(--fg2);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}

/* right: days number */
.kd-cc__right {
  flex-shrink: 0; padding: 0 14px;
  display: flex; align-items: baseline; gap: 3px;
}
.kd-cc__num {
  font-size: 20px; font-weight: 900; line-height: 1;
  font-variant-numeric: tabular-nums; color: var(--_c);
}
.kd-cc__num--na { font-size: 16px; opacity: .35; color: var(--fg2); }
.kd-cc__unit {
  font-size: 11px; font-weight: 600; color: var(--fg2);
}

/* status badge (at end) */
.kd-cc__badge {
  display: inline-flex; align-items: center; gap: 4px; flex-shrink: 0;
  font-size: 11px; font-weight: 700; padding: 4px 12px;
  border-left: 1px solid var(--border);
  color: var(--_c); white-space: nowrap; align-self: stretch;
  justify-content: center; min-width: 60px;
}
.kd-cc__dot {
  display: inline-block; width: 6px; height: 6px; border-radius: 50%;
  background: var(--_c);
}

/* ── Status color overrides ── */
.kd-cc--ok       { --_c: #059669; }
.kd-cc--warn     { --_c: #d97706; }
.kd-cc--critical { --_c: #dc2626; }
.kd-cc--unknown  { --_c: #94a3b8; }

:global(html.dark) .kd-cc--ok       { --_c: #34d399; }
:global(html.dark) .kd-cc--warn     { --_c: #fbbf24; }
:global(html.dark) .kd-cc--critical { --_c: #f87171; }
:global(html.dark) .kd-cc--unknown  { --_c: #64748b; }

/* ── Tooltip ── */
.kd-tip { display: flex; flex-direction: column; gap: 5px; min-width: 200px; }
.kd-tip__row { display: flex; gap: 10px; font-size: 12.5px; line-height: 1.5; }
.kd-tip__k {
  flex-shrink: 0; width: 36px; font-weight: 700; opacity: .6;
}

@media (max-width: 900px) {
  .kd-certs__cards { grid-template-columns: 1fr; }
}

/* ═══════════════ RESPONSIVE ═══════════════ */
@media (max-width: 1280px) {
  .kd-kpi-grid { grid-template-columns: repeat(2,1fr); }
  .kd-health-grid { grid-template-columns: 1fr; }
  .kd-charts-grid { grid-template-columns: repeat(2, 1fr); }
  .kd-chart-card--wide { grid-column: span 2; }
}
@media (max-width: 900px) {
  .kd-header__controls { justify-content: flex-start; flex-wrap: wrap; }
  .kd-charts-grid { grid-template-columns: 1fr; }
  .kd-chart-card--wide { grid-column: span 1; }
}
@media (max-width: 640px) {
  .kd { --g: 10px; }
  .kd-kpi-grid { grid-template-columns: 1fr 1fr; }
  .kd-header { flex-wrap: wrap; }
  .kd-header__center { display: none; }
  .kd-focus-grid { grid-template-columns: 1fr; }
  .kd-charts-grid { grid-template-columns: 1fr; }
  .kd-chart-card--wide { grid-column: span 1; }
}
</style>
