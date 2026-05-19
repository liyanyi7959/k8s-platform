<template>
  <WorkloadDetailDrawerShell v-model="visible" title="Pod 详情" :loading="loading" :ns="podNamespace" :name="podName" @refresh="refresh">
    <template #actions>
      <el-tooltip content="打开资源关系图" placement="top">
        <el-button size="small" text @click="emit('open-topology', { mode: 'pod', namespace: podNamespace, name: podName })">关系图</el-button>
      </el-tooltip>
    </template>
    <template #summary>
      <div class="k8s-summary-grid">
          <div class="k8s-kv k8s-kv--info">
            <div class="k8s-k">名称:</div>
            <div class="k8s-v">{{ podName }}</div>
          </div>
          <div class="k8s-kv k8s-kv--info">
            <div class="k8s-k">命名空间:</div>
            <div class="k8s-v">{{ podNamespace }}</div>
          </div>
          <div class="k8s-kv k8s-kv--info">
            <div class="k8s-k">Pod IP:</div>
            <div class="k8s-v"><span class="k8s-link">{{ podIP }}</span></div>
            <div class="k8s-kv-actions">
              <el-tooltip content="复制" placement="top">
                <el-button size="small" text :icon="CopyDocument" @click="copyText(podIP)" />
              </el-tooltip>
            </div>
          </div>
          <div class="k8s-kv k8s-kv--info">
            <div class="k8s-k">节点:</div>
            <div class="k8s-v">{{ podNode }}</div>
          </div>
          <div :class="['k8s-kv', podSummaryPhaseAccentClass]">
            <div class="k8s-k">状态:</div>
            <div class="k8s-v k8s-v-inline">
              <span :class="['pod-status-dot', podStatusDotClass]" />
              <span class="pod-status-text">{{ podPhaseText }}</span>
            </div>
          </div>
          <div :class="['k8s-kv', podSummaryRestartAccentClass]">
            <div class="k8s-k">重启次数:</div>
            <div class="k8s-v">{{ podRestartCount }}</div>
          </div>
          <div :class="['k8s-kv', podSummaryReadyAccentClass]">
            <div class="k8s-k">就绪状态:</div>
            <div class="k8s-v k8s-v-inline">
              <el-icon v-if="podReady" class="pod-ready-icon"><Check /></el-icon>
              <span>{{ podReady ? 'Ready' : 'NotReady' }}</span>
            </div>
          </div>
          <div class="k8s-kv k8s-kv--muted">
            <div class="k8s-k">创建时间:</div>
            <div class="k8s-v">{{ podCreatedAt }}</div>
          </div>
        </div>
    </template>

      <el-tabs v-model="tab" class="k8s-detail-tabs">
        <el-tab-pane label="概览" name="overview">
          <div class="k8s-tab-pane">
            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">基础信息</div></template>
              <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
                <el-descriptions-item label="Pod名称">{{ podName }}</el-descriptions-item>
                <el-descriptions-item label="命名空间">{{ podNamespace }}</el-descriptions-item>
                <el-descriptions-item label="UID">{{ podUid }}</el-descriptions-item>
                <el-descriptions-item label="Pod IP">{{ podIP }}</el-descriptions-item>
                <el-descriptions-item label="主机名">{{ podHostname }}</el-descriptions-item>
                <el-descriptions-item label="DNS名称">{{ podDnsName }}</el-descriptions-item>
                <el-descriptions-item label="QoS等级">
                  <el-tag size="small" :type="getQosTagType(podQos)" effect="light" class="pod-qos-tag">{{ podQos }}</el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="重启策略">{{ podRestartPolicy }}</el-descriptions-item>
                <el-descriptions-item label="Pod Phase">
                  <span class="pod-v-inline">
                    <span :class="['pod-status-dot', podStatusDotClass]" />
                    <span>{{ podPhaseText }}</span>
                  </span>
                </el-descriptions-item>
                <el-descriptions-item label="创建时间">{{ podCreatedAt }}</el-descriptions-item>
                <el-descriptions-item label="最后更新时间">{{ podLastUpdatedAt }}</el-descriptions-item>
                <el-descriptions-item label="调度器">{{ podSchedulerName }}</el-descriptions-item>
              </el-descriptions>
            </el-card>

            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">状态条件</div></template>
              <el-table :data="podConditions" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="type" label="Type" width="160" />
                <el-table-column prop="status" label="Status" width="110" align="center" header-align="center">
                  <template #default="{ row }">
                    <el-tag :type="row.status === 'True' ? 'success' : row.status === 'False' ? 'danger' : 'info'" size="small">{{ row.status }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="reason" label="Reason" width="180" show-overflow-tooltip />
                <el-table-column prop="message" label="Message" min-width="260" show-overflow-tooltip />
                <el-table-column prop="lastTransitionTime" label="LastTransition" width="200" show-overflow-tooltip />
              </el-table>
            </el-card>

            <!-- Labels -->
            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">Labels</div></template>
              <div v-if="podLabels.length === 0" class="pod-empty">-</div>
              <div v-else class="pod-chip-row" style="flex-wrap:wrap;gap:6px">
                <el-tag v-for="lbl in podLabels" :key="lbl" size="small" effect="plain" class="mono" style="margin:0">{{ lbl }}</el-tag>
              </div>
            </el-card>

            <!-- Annotations -->
            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">Annotations</div></template>
              <el-table v-if="podAnnotations.length > 0" :data="podAnnotations" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="key" label="Key" width="280" show-overflow-tooltip />
                <el-table-column prop="value" label="Value" min-width="360" show-overflow-tooltip />
              </el-table>
              <div v-else class="pod-empty">-</div>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="容器信息" name="containers">
            <div class="k8s-tab-pane">
            <div class="k8s-pane-toolbar">
              <el-radio-group v-model="activeContainer" size="small">
                <el-radio-button v-for="c in containerOptions" :key="c.key" :value="c.key">{{ c.label }}</el-radio-button>
              </el-radio-group>
            </div>

            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header>
                <div class="k8s-section-title-row">
                  <div class="k8s-section-title">容器基础信息</div>
                  <div class="k8s-section-actions">
                    <el-space :size="8">
                      <el-button size="small" type="primary" plain class="pod-detail-action-btn" @click="emitOpenLogs">查看日志</el-button>
                      <el-button size="small" type="primary" plain class="pod-detail-action-btn" @click="emitOpenExec">执行命令</el-button>
                    </el-space>
                  </div>
                </div>
              </template>

              <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
                <el-descriptions-item label="容器名称">{{ selectedContainer?.displayName || '-' }}</el-descriptions-item>
                <el-descriptions-item label="状态">
                  <span class="pod-v-inline">
                    <span :class="['pod-status-dot', getContainerStatusDotClass(selectedContainer)]" />
                    <span>{{ selectedContainer?.state || '-' }}</span>
                  </span>
                </el-descriptions-item>
                <el-descriptions-item label="启动时间">{{ selectedContainer?.startedAt || '-' }}</el-descriptions-item>
                <el-descriptions-item label="重启次数">{{ selectedContainer?.restarts ?? '-' }}</el-descriptions-item>
                <el-descriptions-item label="镜像地址">{{ selectedContainer?.image || '-' }}</el-descriptions-item>
                <el-descriptions-item label="镜像拉取策略">{{ selectedContainer?.imagePullPolicy || '-' }}</el-descriptions-item>
              </el-descriptions>
            </el-card>

            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">容器资源配置</div></template>
              <el-table :data="containerResourceRows" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="name" label="容器" min-width="140" />
                <el-table-column prop="cpuRequests" label="CPU Requests" width="140" />
                <el-table-column prop="cpuLimits" label="CPU Limits" width="140" />
                <el-table-column prop="memRequests" label="Memory Requests" width="160" />
                <el-table-column prop="memLimits" label="Memory Limits" width="160" />
                <el-table-column prop="ephemeralRequests" label="Ephemeral Requests" width="180" />
                <el-table-column prop="ephemeralLimits" label="Ephemeral Limits" width="180" />
              </el-table>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="资源使用" name="resources">
          <div class="k8s-tab-pane">
            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">资源配置</div></template>
              <el-table :data="resourceSummaryRows" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="type" label="资源类型" width="140" />
                <el-table-column prop="requests" label="Requests（请求）" width="220" />
                <el-table-column prop="limits" label="Limits（限制）" width="220" />
                <el-table-column prop="note" label="备注" min-width="220" />
              </el-table>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="关联资源" name="related">
          <div class="k8s-tab-pane">
            <el-card shadow="never" class="k8s-section-card k8s-accent-card pod-related-list-card">
              <template #header><div class="k8s-section-title">关联资源（列表）</div></template>
              <EmptyState v-if="relatedRows.length === 0" type="no-data" description="暂无关联资源" />
              <el-table v-else :data="relatedRows" stripe size="small" class="k8s-detail-table pod-related-table">
                <el-table-column label="分类" width="120">
                  <template #default="{ row }">
                    <el-tag size="small" :type="getRelatedGroupTagType(row.group)" effect="light">{{ row.group }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="资源" min-width="340">
                  <template #default="{ row }">
                    <div class="related-row">
                      <img :src="row.iconUrl" class="related-row-icon" alt="">
                      <div class="related-row-main">
                        <div class="related-row-top">
                          <el-tag size="small" :type="row.kindTagType" effect="light" class="related-row-kind">{{ row.kind }}</el-tag>
                          <span class="related-row-name">{{ row.name }}</span>
                        </div>
                        <div v-if="row.summary" class="related-row-summary">{{ row.summary }}</div>
                      </div>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="120" align="right" header-align="right">
                  <template #default="{ row }">
                    <el-button size="small" text type="primary" :icon="View" class="pod-detail-action-btn" @click="emitOpenRelated(row)">详情</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="调度信息" name="scheduling">
          <div class="k8s-tab-pane">
            <el-card shadow="never" class="k8s-section-card k8s-accent-card pod-section-card--schedule-result">
              <template #header><div class="k8s-section-title">调度结果</div></template>
              <el-descriptions :column="2" size="small" border class="k8s-detail-desc">
                <el-descriptions-item label="调度节点">{{ podNode }}</el-descriptions-item>
                <el-descriptions-item label="调度器名称">{{ podSchedulerName }}</el-descriptions-item>
                <el-descriptions-item label="调度时间">{{ podScheduledAt }}</el-descriptions-item>
                <el-descriptions-item label="调度状态">
                  <span class="pod-v-inline">
                    <el-icon v-if="podScheduledAt !== '-'" class="pod-ready-icon"><Check /></el-icon>
                    <span>{{ podScheduledAt !== '-' ? '调度成功' : '-' }}</span>
                  </span>
                </el-descriptions-item>
              </el-descriptions>
            </el-card>

            <el-card shadow="never" class="k8s-section-card k8s-accent-card pod-section-card--schedule-rules">
              <template #header><div class="k8s-section-title">调度规则</div></template>
              <div class="pod-one-col">
                <div class="pod-box pod-box--selector">
                  <div class="pod-box-title">节点选择器 (nodeSelector)</div>
                  <div v-if="podNodeSelectorPairs.length === 0" class="pod-empty">-</div>
                  <div v-else class="pod-chip-row">
                    <div v-for="(it, idx) in podNodeSelectorPairs" :key="`${it.key}:${idx}`" :class="['pod-chip', `pod-kv-tone-${idx % 6}`]">
                      <span class="pod-chip-k mono">{{ it.key }}</span>
                      <span class="pod-chip-sep">=</span>
                      <span class="pod-chip-v mono">{{ it.value }}</span>
                    </div>
                  </div>
                </div>
                <div class="pod-box pod-box--affinity">
                  <div class="pod-box-title">亲和性规则</div>
                  <div v-if="podAffinityRules.length === 0" class="pod-empty">-</div>
                  <div v-else class="pod-rule-list">
                    <div v-for="(r, idx) in podAffinityRules" :key="`${r.title}:${idx}`" class="pod-rule">
                      <div class="pod-rule-title mono">{{ r.title }}</div>
                      <div v-if="r.pairs.length === 0" class="pod-empty">-</div>
                      <div v-else class="pod-chip-row">
                        <div v-for="(p, pIdx) in r.pairs" :key="`${p.key}:${pIdx}`" :class="['pod-chip', `pod-kv-tone-${pIdx % 6}`]">
                          <span class="pod-chip-k mono">{{ p.key }}</span>
                          <span class="pod-chip-sep">:</span>
                          <span class="pod-chip-v mono">{{ p.value }}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </el-card>

            <el-card shadow="never" class="k8s-section-card k8s-accent-card pod-section-card--schedule-taints">
              <template #header><div class="k8s-section-title">污点与容忍</div></template>
              <div class="pod-one-col">
                <div class="pod-box pod-box--taints">
                  <div class="pod-box-title">节点污点 (Taints)</div>
                  <div v-if="podNodeTaintRules.length === 0" class="pod-empty">{{ nodeInfoLoading ? '加载中…' : '-' }}</div>
                  <div v-else class="pod-rule-list">
                    <div v-for="(r, idx) in podNodeTaintRules" :key="`${r.title}:${idx}`" class="pod-rule">
                      <div class="pod-rule-title mono">{{ r.title }}</div>
                      <div v-if="r.pairs.length === 0" class="pod-empty">-</div>
                      <div v-else class="pod-chip-row">
                        <div v-for="(p, pIdx) in r.pairs" :key="`${p.key}:${pIdx}`" :class="['pod-chip', `pod-kv-tone-${pIdx % 6}`]">
                          <span class="pod-chip-k mono">{{ p.key }}</span>
                          <span class="pod-chip-sep">:</span>
                          <span class="pod-chip-v mono">{{ p.value }}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
                <div class="pod-box pod-box--tolerations">
                  <div class="pod-box-title">Pod容忍度 (Tolerations)</div>
                  <div v-if="podTolerationRules.length === 0" class="pod-empty">-</div>
                  <div v-else class="pod-rule-list">
                    <div v-for="(r, idx) in podTolerationRules" :key="`${r.title}:${idx}`" class="pod-rule">
                      <div class="pod-rule-title mono">{{ r.title }}</div>
                      <div v-if="r.pairs.length === 0" class="pod-empty">-</div>
                      <div v-else class="pod-chip-row">
                        <div v-for="(p, pIdx) in r.pairs" :key="`${p.key}:${pIdx}`" :class="['pod-chip', `pod-kv-tone-${pIdx % 6}`]">
                          <span class="pod-chip-k mono">{{ p.key }}</span>
                          <span class="pod-chip-sep">:</span>
                          <span class="pod-chip-v mono">{{ p.value }}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="网络存储" name="netstore">
          <div class="k8s-tab-pane">
            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">网络信息</div></template>
              <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
                <el-descriptions-item label="HostNetwork">{{ podHostNetwork }}</el-descriptions-item>
                <el-descriptions-item label="DNSPolicy">{{ podDnsPolicy }}</el-descriptions-item>
                <el-descriptions-item label="ServiceLinks">{{ podEnableServiceLinks }}</el-descriptions-item>
                <el-descriptions-item label="Hostname">{{ podHostname }}</el-descriptions-item>
                <el-descriptions-item label="Subdomain">{{ podSubdomain }}</el-descriptions-item>
                <el-descriptions-item label="Ports">{{ podPortsText }}</el-descriptions-item>
              </el-descriptions>
            </el-card>

            <el-card shadow="never" class="k8s-section-card k8s-accent-card">
              <template #header><div class="k8s-section-title">存储信息</div></template>
              <el-table :data="podVolumeRows" stripe size="small" class="k8s-detail-table">
                <el-table-column prop="name" label="Volume" min-width="160" />
                <el-table-column prop="type" label="Type" width="140" />
                <el-table-column prop="source" label="Source" min-width="220" show-overflow-tooltip />
                <el-table-column prop="mounts" label="Mounts" min-width="260" show-overflow-tooltip />
              </el-table>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane label="事件日志" name="events">
          <div class="k8s-tab-pane">
            <div class="k8s-pane-toolbar">
              <el-space :size="8">
                <el-tag v-if="eventsLoading" size="small" type="info" effect="light">加载中</el-tag>
                <el-tag v-else size="small" type="info" effect="light">共 {{ events.length }} 条</el-tag>
                <el-tooltip content="刷新事件" placement="top">
                  <el-button size="small" :icon="RefreshRight" circle :loading="eventsLoading" @click="loadEvents" />
                </el-tooltip>
              </el-space>
            </div>
            <el-table :data="events" stripe size="small" class="k8s-detail-table">
              <el-table-column prop="type" label="Type" width="110" align="center" header-align="center">
                <template #default="{ row }">
                  <el-tag :type="row.type === 'Warning' ? 'danger' : row.type === 'Normal' ? 'success' : 'info'" size="small">{{ row.type }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="reason" label="Reason" width="200" show-overflow-tooltip />
              <el-table-column prop="message" label="Message" min-width="360" show-overflow-tooltip />
              <el-table-column prop="count" label="Count" width="90" align="center" header-align="center" />
              <el-table-column prop="lastSeen" label="LastSeen" width="120" align="center" header-align="center" />
            </el-table>
          </div>
        </el-tab-pane>

        <el-tab-pane label="YAML配置" name="yaml">
          <div class="k8s-tab-pane">
            <div class="k8s-pane-toolbar">
              <el-space :size="8">
                <el-tooltip content="复制" placement="top">
                  <el-button size="small" :icon="CopyDocument" circle :disabled="!yamlViewText" @click="copyText(yamlViewText)" />
                </el-tooltip>
                <el-tooltip content="刷新" placement="top">
                  <el-button size="small" :icon="RefreshRight" circle :loading="yamlLoading" @click="loadYaml" />
                </el-tooltip>
              </el-space>
            </div>
            <pre class="k8s-detail-box">{{ yamlViewText }}</pre>
          </div>
        </el-tab-pane>
      </el-tabs>
  </WorkloadDetailDrawerShell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Check, Close, CopyDocument, RefreshRight, View } from '@element-plus/icons-vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import EmptyState from '@/shared/components/EmptyState.vue'
import WorkloadDetailDrawerShell from './WorkloadDetailDrawerShell.vue'
import type { ApiError } from '@/shared/utils/error'
import k8sIconCmUrl from '@/assets/images/k8s/cm.svg'
import k8sIconCronJobUrl from '@/assets/images/k8s/cronjob.svg'
import k8sIconDeploymentUrl from '@/assets/images/k8s/deploy.svg'
import k8sIconDaemonSetUrl from '@/assets/images/k8s/ds.svg'
import k8sIconGroupUrl from '@/assets/images/k8s/group.svg'
import k8sIconIngressUrl from '@/assets/images/k8s/ing.svg'
import k8sIconJobUrl from '@/assets/images/k8s/job.svg'
import k8sIconPvUrl from '@/assets/images/k8s/pv.svg'
import k8sIconPvcUrl from '@/assets/images/k8s/pvc.svg'
import k8sIconSecretUrl from '@/assets/images/k8s/secret.svg'
import k8sIconStatefulSetUrl from '@/assets/images/k8s/sts.svg'
import k8sIconServiceUrl from '@/assets/images/k8s/svc.svg'

type TabKey = 'overview' | 'containers' | 'resources' | 'related' | 'scheduling' | 'netstore' | 'events' | 'yaml'
type RelatedAction = 'owner' | 'configmap' | 'secret' | 'service' | 'ingress' | 'pvc' | 'pv'
type RelatedGroup = '控制器' | '配置' | '网络' | '存储'
type RelatedRow = {
  group: RelatedGroup
  kind: string
  name: string
  summary?: string
  iconUrl: string
  kindTagType: 'success' | 'warning' | 'danger' | 'info'
  action: RelatedAction
}

const props = defineProps<{
  clusterId: number
}>()

const emit = defineEmits<{
  (e: 'open-logs', v: { ns: string; name: string; container?: string }): void
  (e: 'open-exec', v: { row: any; container?: string }): void
  (e: 'open-related', v: { action: RelatedAction; kind?: string; name: string; namespace?: string }): void
  (e: 'open-topology', v: { mode: 'pod'; namespace: string; name: string }): void
}>()

const visible = ref(false)
const loading = ref(false)
const tab = ref<TabKey>('overview')
const podRow = ref<any>(null)

function getRowNamespace(row: any): string | null {
  const ns = row?.metadata?.namespace
  const v = ns != null ? String(ns).trim() : ''
  return v ? v : null
}

function formatTs(ts: any): string {
  if (!ts) return '-'
  const t = new Date(String(ts)).getTime()
  if (!Number.isFinite(t)) return '-'
  return new Date(t).toLocaleString()
}

function toStringList(v: any): string[] {
  if (!Array.isArray(v)) return []
  return v.map((x) => String(x ?? '').trim()).filter(Boolean)
}

function formatSelector(sel: any): string {
  if (!sel || typeof sel !== 'object') return '-'
  const parts: string[] = []
  const ml = sel?.matchLabels
  if (ml && typeof ml === 'object') {
    for (const [k, v] of Object.entries(ml as Record<string, any>)) {
      const kk = String(k ?? '').trim()
      if (!kk) continue
      const vv = v != null ? String(v) : ''
      parts.push(`${kk}=${vv}`)
    }
  }
  const me: any[] = Array.isArray(sel?.matchExpressions) ? sel.matchExpressions : []
  for (const e of me) {
    const key = String(e?.key ?? '').trim()
    const op = String(e?.operator ?? '').trim()
    const values = toStringList(e?.values)
    if (!key) continue
    const valText = values.length ? `[${values.join(', ')}]` : ''
    parts.push([key, op, valText].filter(Boolean).join(' '))
  }
  return parts.length ? parts.join(', ') : '-'
}

type PodRulePair = { key: string; value: string }
type PodRuleItem = { title: string; pairs: PodRulePair[] }

function addRulePair(pairs: PodRulePair[], key: string, value: any) {
  const k = String(key ?? '').trim()
  const v = value == null ? '' : String(value).trim()
  if (!k || !v || v === '-') return
  pairs.push({ key: k, value: v })
}

function pairsFromNodeSelectorTerm(term: any): PodRulePair[] {
  if (!term || typeof term !== 'object') return []
  const pairs: PodRulePair[] = []
  const exps: any[] = Array.isArray(term?.matchExpressions) ? term.matchExpressions : []
  for (const e of exps) {
    const key = String(e?.key ?? '').trim()
    const op = String(e?.operator ?? '').trim()
    const values = toStringList(e?.values)
    if (!key) continue
    const valText = values.length ? `[${values.join(', ')}]` : ''
    const value = [op, valText].filter(Boolean).join(' ')
    addRulePair(pairs, key, value)
  }
  const fields: any[] = Array.isArray(term?.matchFields) ? term.matchFields : []
  for (const f of fields) {
    const key = String(f?.key ?? '').trim()
    const op = String(f?.operator ?? '').trim()
    const values = toStringList(f?.values)
    if (!key) continue
    const valText = values.length ? `[${values.join(', ')}]` : ''
    const value = [op, valText].filter(Boolean).join(' ')
    addRulePair(pairs, `field:${key}`, value)
  }
  return pairs
}

function affinityToRuleItems(aff: any): PodRuleItem[] {
  if (!aff || typeof aff !== 'object') return []
  const out: PodRuleItem[] = []

  const nodeAff = aff?.nodeAffinity
  if (nodeAff && typeof nodeAff === 'object') {
    const req = nodeAff?.requiredDuringSchedulingIgnoredDuringExecution
    const terms: any[] = Array.isArray(req?.nodeSelectorTerms) ? req.nodeSelectorTerms : []
    for (let i = 0; i < terms.length; i++) {
      out.push({ title: `nodeAffinity(required) term#${i + 1}`, pairs: pairsFromNodeSelectorTerm(terms[i]) })
    }

    const pref: any[] = Array.isArray(nodeAff?.preferredDuringSchedulingIgnoredDuringExecution) ? nodeAff.preferredDuringSchedulingIgnoredDuringExecution : []
    for (let i = 0; i < pref.length; i++) {
      const pairs: PodRulePair[] = []
      const w = pref[i]?.weight != null && pref[i]?.weight !== '' ? String(pref[i].weight).trim() : ''
      addRulePair(pairs, 'weight', w)
      pairs.push(...pairsFromNodeSelectorTerm(pref[i]?.preference))
      out.push({ title: `nodeAffinity(preferred) term#${i + 1}`, pairs })
    }
  }

  const fmtPodAffinityTerms = (kind: 'podAffinity' | 'podAntiAffinity', raw: any) => {
    if (!raw || typeof raw !== 'object') return
    const req: any[] = Array.isArray(raw?.requiredDuringSchedulingIgnoredDuringExecution) ? raw.requiredDuringSchedulingIgnoredDuringExecution : []
    for (let i = 0; i < req.length; i++) {
      const pairs: PodRulePair[] = []
      const sel = formatSelector(req[i]?.labelSelector)
      const topo = String(req[i]?.topologyKey ?? '').trim()
      const ns = toStringList(req[i]?.namespaces)
      addRulePair(pairs, 'selector', sel)
      addRulePair(pairs, 'topologyKey', topo)
      addRulePair(pairs, 'namespaces', ns.length ? ns.join(', ') : '')
      out.push({ title: `${kind}(required) term#${i + 1}`, pairs })
    }
    const pref: any[] = Array.isArray(raw?.preferredDuringSchedulingIgnoredDuringExecution) ? raw.preferredDuringSchedulingIgnoredDuringExecution : []
    for (let i = 0; i < pref.length; i++) {
      const pairs: PodRulePair[] = []
      const w = pref[i]?.weight != null && pref[i]?.weight !== '' ? String(pref[i].weight).trim() : ''
      addRulePair(pairs, 'weight', w)
      const term = pref[i]?.podAffinityTerm
      const sel = formatSelector(term?.labelSelector)
      const topo = String(term?.topologyKey ?? '').trim()
      const ns = toStringList(term?.namespaces)
      addRulePair(pairs, 'selector', sel)
      addRulePair(pairs, 'topologyKey', topo)
      addRulePair(pairs, 'namespaces', ns.length ? ns.join(', ') : '')
      out.push({ title: `${kind}(preferred) term#${i + 1}`, pairs })
    }
  }

  fmtPodAffinityTerms('podAffinity', aff?.podAffinity)
  fmtPodAffinityTerms('podAntiAffinity', aff?.podAntiAffinity)

  return out
}

function normalizeMultilineText(input: string): string {
  let text = String(input ?? '')
  if (!text) return ''
  text = text.replace(/\r\n/g, '\n')
  const quoted = (text.startsWith('"') && text.endsWith('"')) || (text.startsWith("'") && text.endsWith("'"))
  if (quoted && text.includes('\\n')) {
    text = text.slice(1, -1)
  }
  const hasRealNewline = text.includes('\n')
  const hasEscapedNewline = text.includes('\\n')
  if (!hasRealNewline && hasEscapedNewline) {
    text = text.replace(/\\r\\n/g, '\n').replace(/\\n/g, '\n').replace(/\\t/g, '\t')
  }
  return text
}

function formatAgeMs(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return '-'
  const sec = Math.floor(ms / 1000)
  const min = Math.floor(sec / 60)
  const hour = Math.floor(min / 60)
  const day = Math.floor(hour / 24)
  if (day > 0) return `${day}d`
  if (hour > 0) return `${hour}h`
  if (min > 0) return `${min}m`
  return `${sec}s`
}

const podName = computed(() => String(podRow.value?.metadata?.name ?? ''))
const podNamespace = computed(() => String(podRow.value?.metadata?.namespace ?? ''))
const podNode = computed(() => String(podRow.value?.spec?.nodeName ?? '-'))
const podIP = computed(() => String(podRow.value?.status?.podIP ?? '-'))
const podQos = computed(() => String(podRow.value?.status?.qosClass ?? '-'))
const podUid = computed(() => String(podRow.value?.metadata?.uid ?? '-'))
const podUidShort = computed(() => {
  const uid = String(podRow.value?.metadata?.uid ?? '')
  if (!uid) return '-'
  return uid.length > 10 ? `${uid.slice(0, 6)}…${uid.slice(-4)}` : uid
})

const podLabels = computed(() => {
  const raw = podRow.value?.metadata?.labels
  if (!raw || typeof raw !== 'object') return [] as string[]
  return Object.entries(raw as Record<string, string>).map(([k, v]) => `${k}=${v}`)
})

const podAnnotations = computed(() => {
  const raw = podRow.value?.metadata?.annotations
  if (!raw || typeof raw !== 'object') return [] as { key: string; value: string }[]
  return Object.entries(raw as Record<string, string>).map(([k, v]) => ({ key: k, value: v }))
})
const podRestartPolicy = computed(() => String(podRow.value?.spec?.restartPolicy ?? '-'))
const podSchedulerName = computed(() => String(podRow.value?.spec?.schedulerName ?? '-'))
const podHostname = computed(() => String(podRow.value?.spec?.hostname ?? '-'))
const podSubdomain = computed(() => String(podRow.value?.spec?.subdomain ?? '-'))
const podDnsPolicy = computed(() => String(podRow.value?.spec?.dnsPolicy ?? '-'))
const podEnableServiceLinks = computed(() => {
  const v = podRow.value?.spec?.enableServiceLinks
  if (v == null) return '-'
  return v ? 'true' : 'false'
})
const podHostNetwork = computed(() => {
  const v = podRow.value?.spec?.hostNetwork
  if (v == null) return '-'
  return v ? 'true' : 'false'
})

const podCreatedAt = computed(() => formatTs(podRow.value?.metadata?.creationTimestamp))
const podLastUpdatedAt = computed(() => {
  const cond: any[] = Array.isArray(podRow.value?.status?.conditions) ? podRow.value.status.conditions : []
  const ts = cond
    .map((it) => it?.lastTransitionTime)
    .filter(Boolean)
    .map((it) => new Date(String(it)).getTime())
    .filter((t) => Number.isFinite(t))
    .sort((a, b) => b - a)[0]
  if (ts != null && Number.isFinite(ts)) return new Date(ts).toLocaleString()
  const cs: any[] = Array.isArray(podRow.value?.status?.containerStatuses) ? podRow.value.status.containerStatuses : []
  const ts2 = cs
    .map((it) => it?.state?.running?.startedAt ?? it?.state?.terminated?.finishedAt ?? it?.state?.waiting?.startedAt)
    .filter(Boolean)
    .map((it) => new Date(String(it)).getTime())
    .filter((t) => Number.isFinite(t))
    .sort((a, b) => b - a)[0]
  if (ts2 != null && Number.isFinite(ts2)) return new Date(ts2).toLocaleString()
  return podCreatedAt.value
})

const podPhaseText = computed(() => {
  const phase = String(podRow.value?.status?.phase ?? '')
  const reason = String(podRow.value?.status?.reason ?? '').trim()
  const msg = String(podRow.value?.status?.message ?? '').trim()
  const base = phase || '-'
  const extra = reason || msg
  return extra ? `${base} · ${extra}` : base
})

const podStatusDotClass = computed(() => {
  const phase = String(podRow.value?.status?.phase ?? '').toLowerCase()
  if (phase === 'running') return 'pod-status-dot--ok'
  if (phase === 'pending') return 'pod-status-dot--warn'
  if (phase === 'succeeded') return 'pod-status-dot--ok'
  if (phase === 'failed') return 'pod-status-dot--bad'
  if (phase === 'unknown') return 'pod-status-dot--muted'
  return 'pod-status-dot--muted'
})

const podSummaryPhaseAccentClass = computed(() => {
  const phase = String(podRow.value?.status?.phase ?? '').toLowerCase()
  if (phase === 'running') return 'k8s-kv--ok'
  if (phase === 'pending') return 'k8s-kv--warn'
  if (phase === 'succeeded') return 'k8s-kv--ok'
  if (phase === 'failed') return 'k8s-kv--bad'
  if (phase === 'unknown') return 'k8s-kv--muted'
  return 'k8s-kv--muted'
})

const podRestartCount = computed(() => {
  const cs: any[] = Array.isArray(podRow.value?.status?.containerStatuses) ? podRow.value.status.containerStatuses : []
  return cs.reduce((sum, it) => sum + (Number(it?.restartCount ?? 0) || 0), 0)
})

const podSummaryRestartAccentClass = computed(() => {
  const n = Number(podRestartCount.value) || 0
  return n > 0 ? 'k8s-kv--warn' : 'k8s-kv--ok'
})

const podReady = computed(() => {
  const cs: any[] = Array.isArray(podRow.value?.status?.containerStatuses) ? podRow.value.status.containerStatuses : []
  if (cs.length === 0) return false
  return cs.every((it) => Boolean(it?.ready))
})

const podSummaryReadyAccentClass = computed(() => (podReady.value ? 'k8s-kv--ok' : 'k8s-kv--bad'))

const podSummaryQosAccentClass = computed(() => {
  const q = String(podQos.value ?? '').toLowerCase()
  if (q === 'guaranteed') return 'k8s-kv--ok'
  if (q === 'burstable') return 'k8s-kv--warn'
  if (q === 'besteffort') return 'k8s-kv--bad'
  return 'k8s-kv--muted'
})

const podScheduledAt = computed(() => {
  const raw: any[] = Array.isArray(podRow.value?.status?.conditions) ? podRow.value.status.conditions : []
  const scheduled = raw.find((it) => String(it?.type ?? '') === 'PodScheduled' && String(it?.status ?? '') === 'True')
  return scheduled?.lastTransitionTime ? formatTs(scheduled.lastTransitionTime) : '-'
})

const podDnsName = computed(() => {
  const hn = String(podRow.value?.spec?.hostname ?? '').trim()
  const sd = String(podRow.value?.spec?.subdomain ?? '').trim()
  const ns = String(podRow.value?.metadata?.namespace ?? '').trim()
  if (!hn || !sd || !ns) return '-'
  return `${hn}.${sd}.${ns}.svc.cluster.local`
})

function getQosTagType(qos: string): 'success' | 'warning' | 'info' | 'danger' {
  const q = String(qos ?? '').toLowerCase()
  if (q === 'guaranteed') return 'success'
  if (q === 'burstable') return 'warning'
  if (q === 'besteffort') return 'info'
  return 'info'
}

type PodContainerOption = { key: string; kind: 'container' | 'initContainer'; name: string; label: string }
function makeContainerKey(kind: PodContainerOption['kind'], name: string): string {
  return `${kind}:${name}`
}

const containerOptions = computed<PodContainerOption[]>(() => {
  const containers: any[] = Array.isArray(podRow.value?.spec?.containers) ? podRow.value.spec.containers : []
  const initContainers: any[] = Array.isArray(podRow.value?.spec?.initContainers) ? podRow.value.spec.initContainers : []
  const out: PodContainerOption[] = []
  for (const c of initContainers) {
    const name = String(c?.name ?? '').trim()
    if (!name) continue
    out.push({ key: makeContainerKey('initContainer', name), kind: 'initContainer', name, label: `init/${name}` })
  }
  for (const c of containers) {
    const name = String(c?.name ?? '').trim()
    if (!name) continue
    out.push({ key: makeContainerKey('container', name), kind: 'container', name, label: name })
  }
  return out
})

const containerKeys = computed(() => containerOptions.value.map((it) => it.key))
const activeContainer = ref('')

watch(
  () => [visible.value, containerKeys.value.join('|')] as const,
  ([v]) => {
    if (!v) return
    if (!activeContainer.value || !containerKeys.value.includes(activeContainer.value)) {
      activeContainer.value = containerKeys.value[0] ?? ''
    }
  },
  { immediate: true }
)

const selectedContainer = computed(() => {
  const key = activeContainer.value || containerKeys.value[0] || ''
  if (!key) return null
  const opt = containerOptions.value.find((it) => it.key === key)
  const kind = opt?.kind ?? (key.startsWith('initContainer:') ? 'initContainer' : 'container')
  const name = opt?.name ?? key.split(':').slice(1).join(':')
  if (!name) return null

  const containersSpec: any[] = Array.isArray(podRow.value?.spec?.containers) ? podRow.value.spec.containers : []
  const initContainersSpec: any[] = Array.isArray(podRow.value?.spec?.initContainers) ? podRow.value.spec.initContainers : []
  const containersStatus: any[] = Array.isArray(podRow.value?.status?.containerStatuses) ? podRow.value.status.containerStatuses : []
  const initContainersStatus: any[] = Array.isArray(podRow.value?.status?.initContainerStatuses) ? podRow.value.status.initContainerStatuses : []

  const spec = kind === 'initContainer' ? initContainersSpec : containersSpec
  const status = kind === 'initContainer' ? initContainersStatus : containersStatus

  const s = spec.find((it) => String(it?.name ?? '') === name)
  const cs = status.find((it) => String(it?.name ?? '') === name)
  const image = String(s?.image ?? cs?.image ?? '')
  const imagePullPolicy = String(s?.imagePullPolicy ?? '-')
  const restarts = Number(cs?.restartCount ?? 0) || 0
  const stateObj = cs?.state ?? {}
  const running = stateObj?.running
  const waiting = stateObj?.waiting
  const terminated = stateObj?.terminated
  const state = running ? 'Running' : waiting ? 'Waiting' : terminated ? 'Terminated' : '-'
  const reason = String(waiting?.reason ?? terminated?.reason ?? cs?.lastState?.terminated?.reason ?? waiting?.message ?? terminated?.message ?? '').trim() || '-'
  const startedAt = formatTs(running?.startedAt ?? terminated?.startedAt ?? cs?.state?.running?.startedAt)
  let ready = Boolean(cs?.ready)
  if (kind === 'initContainer' && cs) {
    const exitCode = Number(terminated?.exitCode ?? NaN)
    if (Number.isFinite(exitCode)) ready = exitCode === 0
  }
  const displayName = opt?.label ?? (kind === 'initContainer' ? `init/${name}` : name)
  return { key, kind, name, displayName, image, imagePullPolicy, restarts, state, reason, startedAt, ready }
})

function getContainerStatusDotClass(container: any): string {
  if (!container) return 'pod-status-dot--muted'
  if (container.ready) return 'pod-status-dot--ok'
  const state = String(container.state ?? '').toLowerCase()
  if (state === 'waiting') return 'pod-status-dot--warn'
  if (state === 'terminated') return 'pod-status-dot--bad'
  return 'pod-status-dot--muted'
}

type ContainerResourceRow = {
  name: string
  cpuRequests: string
  cpuLimits: string
  memRequests: string
  memLimits: string
  ephemeralRequests: string
  ephemeralLimits: string
}

function getResVal(obj: any, key: string): string {
  if (!obj || typeof obj !== 'object') return '-'
  const v = obj[key]
  const s = String(v ?? '').trim()
  return s ? s : '-'
}

const containerResourceRows = computed<ContainerResourceRow[]>(() => {
  const containers: any[] = Array.isArray(podRow.value?.spec?.containers) ? podRow.value.spec.containers : []
  const initContainers: any[] = Array.isArray(podRow.value?.spec?.initContainers) ? podRow.value.spec.initContainers : []
  const spec = [
    ...initContainers.map((c) => ({ kind: 'initContainer' as const, c })),
    ...containers.map((c) => ({ kind: 'container' as const, c }))
  ]
  return spec
    .map((c) => {
      const rawName = String(c?.c?.name ?? '')
      const name = c.kind === 'initContainer' ? `init/${rawName}` : rawName
      if (!name) return null
      const req = c?.c?.resources?.requests ?? {}
      const lim = c?.c?.resources?.limits ?? {}
      return {
        name,
        cpuRequests: getResVal(req, 'cpu'),
        cpuLimits: getResVal(lim, 'cpu'),
        memRequests: getResVal(req, 'memory'),
        memLimits: getResVal(lim, 'memory'),
        ephemeralRequests: getResVal(req, 'ephemeral-storage'),
        ephemeralLimits: getResVal(lim, 'ephemeral-storage')
      }
    })
    .filter(Boolean) as ContainerResourceRow[]
})

type ResourceSummaryRow = { type: string; requests: string; limits: string; note: string }
function collectResByContainer(resourceKey: string): { requests: string; limits: string } {
  const containers: any[] = Array.isArray(podRow.value?.spec?.containers) ? podRow.value.spec.containers : []
  const initContainers: any[] = Array.isArray(podRow.value?.spec?.initContainers) ? podRow.value.spec.initContainers : []
  const spec = [
    ...initContainers.map((c) => ({ kind: 'initContainer' as const, c })),
    ...containers.map((c) => ({ kind: 'container' as const, c }))
  ]
  const reqParts: string[] = []
  const limParts: string[] = []
  for (const c of spec) {
    const rawName = String(c?.c?.name ?? '')
    const name = c.kind === 'initContainer' ? `init/${rawName}` : rawName
    if (!name) continue
    const reqVal = String(c?.c?.resources?.requests?.[resourceKey] ?? '').trim()
    const limVal = String(c?.c?.resources?.limits?.[resourceKey] ?? '').trim()
    if (reqVal) reqParts.push(`${name}:${reqVal}`)
    if (limVal) limParts.push(`${name}:${limVal}`)
  }
  return { requests: reqParts.length ? reqParts.join(', ') : '-', limits: limParts.length ? limParts.join(', ') : '-' }
}

const resourceSummaryRows = computed<ResourceSummaryRow[]>(() => {
  const cpu = collectResByContainer('cpu')
  const mem = collectResByContainer('memory')
  const eph = collectResByContainer('ephemeral-storage')
  return [
    { type: 'CPU', requests: cpu.requests, limits: cpu.limits, note: '按容器展示（仅资源配置）' },
    { type: 'Memory', requests: mem.requests, limits: mem.limits, note: '按容器展示（仅资源配置）' },
    { type: 'Ephemeral', requests: eph.requests, limits: eph.limits, note: '按容器展示（仅资源配置）' }
  ]
})

const podPortsText = computed(() => {
  const containers: any[] = Array.isArray(podRow.value?.spec?.containers) ? podRow.value.spec.containers : []
  const initContainers: any[] = Array.isArray(podRow.value?.spec?.initContainers) ? podRow.value.spec.initContainers : []
  const spec = [
    ...initContainers.map((c) => ({ kind: 'initContainer' as const, c })),
    ...containers.map((c) => ({ kind: 'container' as const, c }))
  ]
  const parts: string[] = []
  for (const c of spec) {
    const rawName = String(c?.c?.name ?? '').trim()
    const cName = c.kind === 'initContainer' ? `init/${rawName}` : rawName
    const ports: any[] = Array.isArray(c?.c?.ports) ? c.c.ports : []
    if (ports.length === 0) continue
    const pText = ports
      .map((p) => {
        const name = String(p?.name ?? '').trim()
        const port = p?.containerPort
        const proto = String(p?.protocol ?? 'TCP')
        const hp = p?.hostPort
        const base = name ? `${name}=${port}/${proto}` : `${port}/${proto}`
        return hp ? `${base}(host:${hp})` : base
      })
      .filter(Boolean)
      .join(', ')
    if (pText) parts.push(cName ? `${cName}: ${pText}` : pText)
  }
  return parts.length ? parts.join(' | ') : '-'
})

type PodVolumeRow = { name: string; type: string; source: string; mounts: string }
function getVolumeTypeAndSource(v: any): { type: string; source: string } {
  if (!v || typeof v !== 'object') return { type: '-', source: '-' }
  const name = String(v?.name ?? '')
  if (v?.persistentVolumeClaim) {
    const claim = String(v.persistentVolumeClaim?.claimName ?? '')
    return { type: 'PVC', source: claim ? `claim=${claim}` : '-' }
  }
  if (v?.configMap) {
    const cm = String(v.configMap?.name ?? '')
    return { type: 'ConfigMap', source: cm ? `name=${cm}` : '-' }
  }
  if (v?.secret) {
    const sec = String(v.secret?.secretName ?? '')
    return { type: 'Secret', source: sec ? `name=${sec}` : '-' }
  }
  if (v?.emptyDir) {
    const medium = String(v.emptyDir?.medium ?? '')
    const size = String(v.emptyDir?.sizeLimit ?? '')
    const extra = [medium ? `medium=${medium}` : '', size ? `size=${size}` : ''].filter(Boolean).join(', ')
    return { type: 'EmptyDir', source: extra || '-' }
  }
  if (v?.hostPath) {
    const path = String(v.hostPath?.path ?? '')
    const tp = String(v.hostPath?.type ?? '')
    const extra = [path ? `path=${path}` : '', tp ? `type=${tp}` : ''].filter(Boolean).join(', ')
    return { type: 'HostPath', source: extra || '-' }
  }
  if (v?.projected) return { type: 'Projected', source: '-' }
  if (v?.downwardAPI) return { type: 'DownwardAPI', source: '-' }
  if (v?.serviceAccountToken) return { type: 'SAToken', source: '-' }
  if (v?.nfs) {
    const server = String(v.nfs?.server ?? '')
    const path = String(v.nfs?.path ?? '')
    return { type: 'NFS', source: [server ? `server=${server}` : '', path ? `path=${path}` : ''].filter(Boolean).join(', ') || '-' }
  }
  const keys = Object.keys(v).filter((k) => k !== 'name')
  const first = keys.find((k) => v[k] && typeof v[k] === 'object')
  return { type: first ? first : name ? 'Volume' : '-', source: '-' }
}

const podVolumeRows = computed<PodVolumeRow[]>(() => {
  const vols: any[] = Array.isArray(podRow.value?.spec?.volumes) ? podRow.value.spec.volumes : []
  const containers: any[] = Array.isArray(podRow.value?.spec?.containers) ? podRow.value.spec.containers : []
  const initContainers: any[] = Array.isArray(podRow.value?.spec?.initContainers) ? podRow.value.spec.initContainers : []
  const specContainers = [
    ...initContainers.map((c) => ({ kind: 'initContainer' as const, c })),
    ...containers.map((c) => ({ kind: 'container' as const, c }))
  ]
  return vols
    .map((v) => {
      const name = String(v?.name ?? '')
      if (!name) return null
      const { type, source } = getVolumeTypeAndSource(v)
      const mounts: string[] = []
      for (const c of specContainers) {
        const rawName = String(c?.c?.name ?? '').trim()
        const cName = c.kind === 'initContainer' ? `init/${rawName}` : rawName
        const vms: any[] = Array.isArray(c?.c?.volumeMounts) ? c.c.volumeMounts : []
        for (const m of vms) {
          if (String(m?.name ?? '') !== name) continue
          const mp = String(m?.mountPath ?? '')
          const ro = Boolean(m?.readOnly)
          const sub = String(m?.subPath ?? '')
          const seg = [mp || '-', ro ? 'ro' : '', sub ? `sub=${sub}` : ''].filter(Boolean).join(' ')
          mounts.push(cName ? `${cName}:${seg}` : seg)
        }
      }
      return { name, type, source, mounts: mounts.length ? mounts.join(' | ') : '-' }
    })
    .filter(Boolean) as PodVolumeRow[]
})

const podAffinityRules = computed(() => affinityToRuleItems(podRow.value?.spec?.affinity))

const podNodeSelectorPairs = computed(() => {
  const raw = podRow.value?.spec?.nodeSelector
  if (!raw || typeof raw !== 'object') return []
  return Object.entries(raw as Record<string, any>)
    .map(([k, v]) => ({ key: String(k ?? '').trim(), value: v != null ? String(v) : '' }))
    .filter((it) => it.key)
})

const podTolerationRules = computed<PodRuleItem[]>(() => {
  const ts: any[] = Array.isArray(podRow.value?.spec?.tolerations) ? podRow.value?.spec?.tolerations : []
  return ts
    .map((t, idx) => {
      const pairs: PodRulePair[] = []
      const key = String(t?.key ?? '').trim()
      const op = String(t?.operator ?? '').trim()
      const value = String(t?.value ?? '').trim()
      const effect = String(t?.effect ?? '').trim()
      const seconds = t?.tolerationSeconds != null && t?.tolerationSeconds !== '' ? String(t.tolerationSeconds) : ''
      addRulePair(pairs, 'key', key)
      addRulePair(pairs, 'operator', op)
      addRulePair(pairs, 'value', value)
      addRulePair(pairs, 'effect', effect)
      addRulePair(pairs, 'seconds', seconds)
      if (!pairs.length) return null
      return { title: `toleration#${idx + 1}`, pairs }
    })
    .filter(Boolean) as PodRuleItem[]
})

const nodeInfo = ref<any | null>(null)
const nodeInfoLoading = ref(false)

async function loadNodeInfo() {
  if (!props.clusterId || !podRow.value) return
  const nodeName = String(podRow.value?.spec?.nodeName ?? '').trim()
  if (!nodeName) {
    nodeInfo.value = null
    return
  }
  if (nodeInfo.value && String(nodeInfo.value?.metadata?.name ?? '') === nodeName) return
  nodeInfoLoading.value = true
  try {
    const data = await k8sApi.listNodes(props.clusterId)
    const list: any[] = Array.isArray(data.list) ? data.list : []
    const found = list.find((it) => String(it?.metadata?.name ?? '') === nodeName) ?? null
    nodeInfo.value = found
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    nodeInfoLoading.value = false
  }
}

const podNodeTaintRules = computed<PodRuleItem[]>(() => {
  const taints: any[] = Array.isArray(nodeInfo.value?.spec?.taints) ? nodeInfo.value.spec.taints : []
  return taints
    .map((t, idx) => {
      const pairs: PodRulePair[] = []
      const key = String(t?.key ?? '').trim()
      const value = String(t?.value ?? '').trim()
      const effect = String(t?.effect ?? '').trim()
      const timeAdded = t?.timeAdded != null && t?.timeAdded !== '' ? String(t.timeAdded).trim() : ''
      addRulePair(pairs, 'key', key)
      addRulePair(pairs, 'value', value)
      addRulePair(pairs, 'effect', effect)
      addRulePair(pairs, 'timeAdded', timeAdded)
      if (!pairs.length) return null
      return { title: `taint#${idx + 1}`, pairs }
    })
    .filter(Boolean) as PodRuleItem[]
})

const yamlLoading = ref(false)
const yamlText = ref('')
const yamlViewText = computed(() => normalizeMultilineText(yamlText.value))

async function loadYaml() {
  if (!props.clusterId || !podRow.value) return
  const ns = getRowNamespace(podRow.value)
  const name = String(podRow.value?.metadata?.name ?? '')
  if (!ns || !name) return
  yamlLoading.value = true
  try {
    const data = await k8sApi.getPodYaml(props.clusterId, ns, name)
    yamlText.value = data.text ?? ''
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    yamlLoading.value = false
  }
}

type EventRow = { type: string; reason: string; message: string; count: number; lastSeen: string }
const events = ref<EventRow[]>([])
const eventsLoading = ref(false)

function getEventTimeMs(ev: any): number | null {
  const ts =
    ev?.lastTimestamp ??
    ev?.eventTime ??
    ev?.firstTimestamp ??
    ev?.deprecatedLastTimestamp ??
    ev?.deprecatedFirstTimestamp ??
    ev?.metadata?.creationTimestamp
  if (!ts) return null
  const t = new Date(String(ts)).getTime()
  return Number.isFinite(t) ? t : null
}

async function loadEvents() {
  if (!props.clusterId || !podRow.value) return
  const ns = getRowNamespace(podRow.value)
  const name = String(podRow.value?.metadata?.name ?? '')
  const uid = String(podRow.value?.metadata?.uid ?? '')
  if (!ns || !name) return
  eventsLoading.value = true
  try {
    const data = await k8sApi.listEvents(props.clusterId, { namespace: ns })
    const list: any[] = Array.isArray(data.list) ? data.list : []
    const filtered = list.filter((ev) => {
      const involved = ev?.involvedObject ?? ev?.regarding ?? {}
      const eNs = String(involved?.namespace ?? '')
      const eName = String(involved?.name ?? '')
      const eKind = String(involved?.kind ?? '')
      const eUid = String(involved?.uid ?? '')
      if (eKind && eKind !== 'Pod') return false
      if (eNs && eNs !== ns) return false
      if (uid && eUid) return eUid === uid
      return eName === name
    })

    const now = Date.now()
    const mapped = filtered
      .map((ev) => {
        const t = getEventTimeMs(ev)
        const tMs = t != null ? t : -1
        const lastSeen = t != null ? formatAgeMs(Math.max(0, now - t)) : '-'
        const count = Number(ev?.count ?? 0) || 0
        return {
          tMs,
          type: String(ev?.type ?? '') || '-',
          reason: String(ev?.reason ?? '') || '-',
          message: String(ev?.message ?? '') || '-',
          count,
          lastSeen
        }
      })
      .sort((a, b) => b.tMs - a.tMs)
      .map(({ tMs: _t, ...rest }) => rest)

    events.value = mapped
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    eventsLoading.value = false
  }
}

type RelatedItem = { name: string; summary?: string }
const services = ref<Array<{ name: string; summary?: string }>>([])
const ingressItemsRef = ref<RelatedItem[]>([])
const pvcItemsRef = ref<RelatedItem[]>([])
const pvItemsRef = ref<RelatedItem[]>([])

type OwnerRef = { kind: string; name: string }
const owners = computed<OwnerRef[]>(() => {
  const raw: any[] = Array.isArray(podRow.value?.metadata?.ownerReferences) ? podRow.value.metadata.ownerReferences : []
  return raw
    .map((it) => ({ kind: String(it?.kind ?? ''), name: String(it?.name ?? '') }))
    .filter((it) => it.kind && it.name)
})

const labels = computed<Record<string, string>>(() => {
  const raw = podRow.value?.metadata?.labels
  if (!raw || typeof raw !== 'object') return {}
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(raw)) {
    const key = String(k ?? '').trim()
    if (!key) continue
    out[key] = String(v ?? '')
  }
  return out
})

function getRelatedTagType(kind: string): 'success' | 'warning' | 'danger' | 'info' {
  const k = String(kind ?? '').trim()
  if (!k) return 'info'
  if (k === 'Deployment' || k === 'StatefulSet' || k === 'DaemonSet') return 'success'
  if (k === 'ReplicaSet') return 'info'
  if (k === 'Job' || k === 'CronJob') return 'warning'
  return 'info'
}

function getRelatedIconUrl(kind: string): string {
  const k = String(kind ?? '').trim()
  if (k === 'Deployment') return k8sIconDeploymentUrl
  if (k === 'DaemonSet') return k8sIconDaemonSetUrl
  if (k === 'StatefulSet') return k8sIconStatefulSetUrl
  if (k === 'ReplicaSet') return k8sIconGroupUrl
  if (k === 'Job') return k8sIconJobUrl
  if (k === 'CronJob') return k8sIconCronJobUrl
  if (k === 'ConfigMap') return k8sIconCmUrl
  if (k === 'Secret') return k8sIconSecretUrl
  if (k === 'Service') return k8sIconServiceUrl
  if (k === 'Ingress') return k8sIconIngressUrl
  if (k === 'PVC') return k8sIconPvcUrl
  if (k === 'PV') return k8sIconPvUrl
  return k8sIconGroupUrl
}

function getRelatedGroupTagType(group: string): 'success' | 'warning' | 'danger' | 'info' {
  const g = String(group ?? '').trim()
  if (g === '控制器') return 'success'
  if (g === '配置') return 'warning'
  if (g === '网络') return 'info'
  if (g === '存储') return 'info'
  return 'info'
}

function uniq(arr: string[]): string[] {
  const out: string[] = []
  const set = new Set<string>()
  for (const a of arr) {
    const s = String(a ?? '').trim()
    if (!s || set.has(s)) continue
    set.add(s)
    out.push(s)
  }
  return out
}

const configMaps = computed(() => {
  const out: string[] = []
  const vols: any[] = Array.isArray(podRow.value?.spec?.volumes) ? podRow.value.spec.volumes : []
  for (const v of vols) {
    const cm = String(v?.configMap?.name ?? '').trim()
    if (cm) out.push(cm)
  }
  const cs: any[] = Array.isArray(podRow.value?.spec?.containers) ? podRow.value.spec.containers : []
  for (const c of cs) {
    const envFrom: any[] = Array.isArray(c?.envFrom) ? c.envFrom : []
    for (const ef of envFrom) {
      const cm = String(ef?.configMapRef?.name ?? '').trim()
      if (cm) out.push(cm)
    }
    const env: any[] = Array.isArray(c?.env) ? c.env : []
    for (const e of env) {
      const cm = String(e?.valueFrom?.configMapKeyRef?.name ?? '').trim()
      if (cm) out.push(cm)
    }
  }
  return uniq(out)
})

const secrets = computed(() => {
  const out: string[] = []
  const vols: any[] = Array.isArray(podRow.value?.spec?.volumes) ? podRow.value.spec.volumes : []
  for (const v of vols) {
    const sec = String(v?.secret?.secretName ?? '').trim()
    if (sec) out.push(sec)
  }
  const cs: any[] = Array.isArray(podRow.value?.spec?.containers) ? podRow.value.spec.containers : []
  for (const c of cs) {
    const envFrom: any[] = Array.isArray(c?.envFrom) ? c.envFrom : []
    for (const ef of envFrom) {
      const sec = String(ef?.secretRef?.name ?? '').trim()
      if (sec) out.push(sec)
    }
    const env: any[] = Array.isArray(c?.env) ? c.env : []
    for (const e of env) {
      const sec = String(e?.valueFrom?.secretKeyRef?.name ?? '').trim()
      if (sec) out.push(sec)
    }
  }
  return uniq(out)
})

const serviceItems = computed<RelatedItem[]>(() => {
  const list: any[] = Array.isArray(services.value) ? (services.value as any[]) : []
  return list
    .map((it) => ({ name: String(it?.name ?? ''), summary: String(it?.summary ?? '') || undefined }))
    .filter((it) => it.name)
})
const ingressItems = computed<RelatedItem[]>(() => ingressItemsRef.value)
const pvcItems = computed<RelatedItem[]>(() => pvcItemsRef.value)
const pvItems = computed<RelatedItem[]>(() => pvItemsRef.value)

async function loadServices() {
  if (!props.clusterId || !podRow.value) return
  const ns = getRowNamespace(podRow.value)
  if (!ns) return
  try {
    const data = await k8sApi.listServices(props.clusterId, { namespace: ns })
    const list: any[] = Array.isArray(data.list) ? data.list : []
    const matched = list
      .map((svc) => {
        const sel = svc?.spec?.selector
        if (!sel || typeof sel !== 'object') return null
        const entries = Object.entries(sel).map(([k, v]) => [String(k ?? '').trim(), String(v ?? '')] as const).filter(([k]) => k)
        if (entries.length === 0) return null
        const ok = entries.every(([k, v]) => labels.value[k] === v)
        if (!ok) return null
        const name = String(svc?.metadata?.name ?? '')
        return name ? { name, summary: getSvcSummary(svc) } : null
      })
      .filter(Boolean) as Array<{ name: string; summary?: string }>
    const dedup = Array.from(new Map(matched.map((it) => [it.name, it])).values())
    services.value = dedup
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  }
}

function getSvcSummary(svc: any): string {
  const tp = String(svc?.spec?.type ?? '')
  const ip = String(svc?.spec?.clusterIP ?? '')
  const ports: any[] = Array.isArray(svc?.spec?.ports) ? svc.spec.ports : []
  const p = ports.length ? `${ports.length} ports` : ''
  const parts = [tp ? `type=${tp}` : '', ip ? `clusterIP=${ip}` : '', p].filter(Boolean)
  return parts.length ? parts.join(', ') : ''
}

async function loadRelated() {
  if (!props.clusterId || !podRow.value) return
  const ns = getRowNamespace(podRow.value)
  if (!ns) return

  if (services.value.length === 0) await loadServices()

  const podSvcNames = new Set(serviceItems.value.map((it) => it.name))
  const pvcNames = uniq(
    (Array.isArray(podRow.value?.spec?.volumes) ? podRow.value.spec.volumes : [])
      .map((v: any) => String(v?.persistentVolumeClaim?.claimName ?? '').trim())
      .filter(Boolean)
  )

  const tasks: Array<Promise<void>> = []

  tasks.push(
    (async () => {
      try {
        const data = await k8sApi.listIngresses(props.clusterId, { namespace: ns })
        const list: any[] = Array.isArray(data.list) ? data.list : []
        const matched = list
          .map((ig) => {
            const name = String(ig?.metadata?.name ?? '').trim()
            if (!name) return null
            const svcNames: string[] = []
            const spec = ig?.spec ?? {}
            const d = spec?.defaultBackend?.service?.name
            if (d) svcNames.push(String(d))
            const rules: any[] = Array.isArray(spec?.rules) ? spec.rules : []
            for (const r of rules) {
              const paths: any[] = Array.isArray(r?.http?.paths) ? r.http.paths : []
              for (const p of paths) {
                const s = p?.backend?.service?.name
                if (s) svcNames.push(String(s))
              }
            }
            const touched = svcNames.some((s) => podSvcNames.has(String(s)))
            if (!touched) return null
            const hosts = rules.map((r) => String(r?.host ?? '')).filter(Boolean)
            const summary = hosts.length ? `hosts=${hosts.join(', ')}` : ''
            return { name, summary: summary || undefined }
          })
          .filter(Boolean) as RelatedItem[]
        ingressItemsRef.value = matched
      } catch (e) {
        const err = e as ApiError
        notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
      }
    })()
  )

  tasks.push(
    (async () => {
      try {
        if (pvcNames.length === 0) {
          pvcItemsRef.value = []
          pvItemsRef.value = []
          return
        }
        const data = await k8sApi.listPVCs(props.clusterId, { namespace: ns })
        const list: any[] = Array.isArray(data.list) ? data.list : []
        const pvcByName = new Map<string, any>()
        for (const pvc of list) {
          const name = String(pvc?.metadata?.name ?? '').trim()
          if (name) pvcByName.set(name, pvc)
        }
        const pvNames: string[] = []
        const items: RelatedItem[] = pvcNames.map((name) => {
          const pvc = pvcByName.get(name)
          const sc = String(pvc?.spec?.storageClassName ?? '')
          const phase = String(pvc?.status?.phase ?? '')
          const size = String(pvc?.status?.capacity?.storage ?? '')
          const pv = String(pvc?.spec?.volumeName ?? '').trim()
          if (pv) pvNames.push(pv)
          const summary = [phase ? `phase=${phase}` : '', sc ? `sc=${sc}` : '', size ? `size=${size}` : ''].filter(Boolean).join(', ')
          return { name, summary: summary || undefined }
        })
        pvcItemsRef.value = items

        const pvData = await k8sApi.listPVs(props.clusterId)
        const pvList: any[] = Array.isArray(pvData.list) ? pvData.list : []
        const pvByName = new Map<string, any>()
        for (const pv of pvList) {
          const name = String(pv?.metadata?.name ?? '').trim()
          if (name) pvByName.set(name, pv)
        }
        const pvItemsOut: RelatedItem[] = uniq(pvNames)
          .map((name) => {
            const pv = pvByName.get(name)
            const phase = String(pv?.status?.phase ?? '')
            const sc = String(pv?.spec?.storageClassName ?? '')
            const cap = String(pv?.spec?.capacity?.storage ?? '')
            const summary = [phase ? `phase=${phase}` : '', sc ? `sc=${sc}` : '', cap ? `cap=${cap}` : ''].filter(Boolean).join(', ')
            return { name, summary: summary || undefined }
          })
          .filter((it) => it.name)
        pvItemsRef.value = pvItemsOut
      } catch (e) {
        const err = e as ApiError
        notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
      }
    })()
  )

  await Promise.all(tasks)
}

const relatedRows = computed<RelatedRow[]>(() => {
  const rows: RelatedRow[] = []
  for (const it of owners.value) {
    rows.push({
      group: '控制器',
      kind: it.kind,
      name: it.name,
      iconUrl: getRelatedIconUrl(it.kind),
      kindTagType: getRelatedTagType(it.kind),
      action: 'owner'
    })
  }

  for (const name of configMaps.value) {
    rows.push({
      group: '配置',
      kind: 'ConfigMap',
      name,
      iconUrl: getRelatedIconUrl('ConfigMap'),
      kindTagType: 'info',
      action: 'configmap'
    })
  }
  for (const name of secrets.value) {
    rows.push({
      group: '配置',
      kind: 'Secret',
      name,
      iconUrl: getRelatedIconUrl('Secret'),
      kindTagType: 'warning',
      action: 'secret'
    })
  }

  for (const s of serviceItems.value) {
    rows.push({
      group: '网络',
      kind: 'Service',
      name: s.name,
      summary: s.summary,
      iconUrl: getRelatedIconUrl('Service'),
      kindTagType: 'success',
      action: 'service'
    })
  }
  for (const ig of ingressItems.value) {
    rows.push({
      group: '网络',
      kind: 'Ingress',
      name: ig.name,
      summary: ig.summary,
      iconUrl: getRelatedIconUrl('Ingress'),
      kindTagType: 'info',
      action: 'ingress'
    })
  }

  for (const p of pvcItems.value) {
    rows.push({
      group: '存储',
      kind: 'PVC',
      name: p.name,
      summary: p.summary,
      iconUrl: getRelatedIconUrl('PVC'),
      kindTagType: 'info',
      action: 'pvc'
    })
  }
  for (const p of pvItems.value) {
    rows.push({
      group: '存储',
      kind: 'PV',
      name: p.name,
      summary: p.summary,
      iconUrl: getRelatedIconUrl('PV'),
      kindTagType: 'info',
      action: 'pv'
    })
  }

  return rows
})

const podConditions = computed(() => {
  const raw: any[] = Array.isArray(podRow.value?.status?.conditions) ? podRow.value.status.conditions : []
  return raw.map((it) => ({
    type: String(it?.type ?? ''),
    status: String(it?.status ?? ''),
    reason: String(it?.reason ?? '') || '-',
    message: String(it?.message ?? '') || '-',
    lastTransitionTime: String(it?.lastTransitionTime ?? '') || '-'
  }))
})

async function copyText(text: string) {
  const v = String(text ?? '')
  if (!v) return
  try {
    await navigator.clipboard.writeText(v)
    notifySuccess('已复制')
  } catch {
    try {
      const ta = document.createElement('textarea')
      ta.value = v
      ta.style.position = 'fixed'
      ta.style.left = '-9999px'
      ta.style.top = '0'
      document.body.appendChild(ta)
      ta.focus()
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
      notifySuccess('已复制')
    } catch (e) {
      const err = e as ApiError
      notifyError(err?.message ? `复制失败：${err.message}` : '复制失败')
    }
  }
}

function copyPodRef() {
  if (!podNamespace.value || !podName.value) return
  void copyText(`${podNamespace.value}/${podName.value}`)
}

function emitOpenLogs() {
  if (!podRow.value) return
  const ns = getRowNamespace(podRow.value)
  const name = String(podRow.value?.metadata?.name ?? '')
  if (!ns || !name) return
  const container = selectedContainer.value?.name || undefined
  emit('open-logs', { ns, name, container })
}

function emitOpenExec() {
  if (!podRow.value) return
  const container = selectedContainer.value?.name || undefined
  emit('open-exec', { row: podRow.value, container })
}

function emitOpenRelated(row: RelatedRow) {
  const action = row?.action
  const name = String(row?.name ?? '').trim()
  const kind = String(row?.kind ?? '').trim()
  const namespace = podNamespace.value ? String(podNamespace.value) : undefined
  if (!name) return
  emit('open-related', { action, kind, name, namespace })
}

async function refresh() {
  if (!props.clusterId || !podRow.value) return
  const ns = getRowNamespace(podRow.value)
  const name = String(podRow.value?.metadata?.name ?? '')
  if (!ns || !name) return
  loading.value = true
  try {
    const data = await k8sApi.listPods(props.clusterId, { namespace: ns })
    const found = (data.list ?? []).find((it: any) => String(it?.metadata?.name ?? '') === name)
    if (found) podRow.value = found
    if (tab.value === 'events') await loadEvents()
    if (tab.value === 'related') {
      if (services.value.length === 0) await loadServices()
      await loadRelated()
    }
    if (tab.value === 'yaml') await loadYaml()
    if (tab.value === 'scheduling') await loadNodeInfo()
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    loading.value = false
  }
}

function open(row: any) {
  podRow.value = row
  tab.value = 'overview'
  visible.value = true
  events.value = []
  services.value = []
  ingressItemsRef.value = []
  pvcItemsRef.value = []
  pvItemsRef.value = []
  yamlText.value = ''
  nodeInfo.value = null
  activeContainer.value = ''
}

watch(
  () => [visible.value, tab.value, podName.value, podNamespace.value] as const,
  ([v, t]) => {
    if (!v) return
    if (t === 'events' && events.value.length === 0) void loadEvents()
    if (t === 'related') {
      if (services.value.length === 0) void loadServices()
      if (ingressItemsRef.value.length === 0 && pvcItemsRef.value.length === 0 && pvItemsRef.value.length === 0) void loadRelated()
    }
    if (t === 'yaml' && !yamlText.value) void loadYaml()
    if (t === 'scheduling') void loadNodeInfo()
  }
)

watch(
  () => visible.value,
  (v) => {
    if (v) return
    podRow.value = null
    tab.value = 'overview'
    events.value = []
    services.value = []
    ingressItemsRef.value = []
    pvcItemsRef.value = []
    pvItemsRef.value = []
    yamlText.value = ''
    nodeInfo.value = null
    activeContainer.value = ''
  }
)

defineExpose({ open })
</script>

<style scoped>
.pod-detail-root {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
  height: 100%;
}

.pod-v-inline {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  justify-content: flex-start;
  min-width: 0;
}

.pod-status-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pod-ready-icon {
  color: rgba(16, 185, 129, 0.95);
}

.pod-status-dot {
  width: 10px;
  height: 10px;
  border-radius: 999px;
  display: inline-block;
  background: rgba(148, 163, 184, 0.9);
  box-shadow: 0 0 0 3px rgba(148, 163, 184, 0.15);
}
.pod-status-dot--ok {
  background: rgba(16, 185, 129, 0.95);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.16);
}
.pod-status-dot--warn {
  background: rgba(245, 158, 11, 0.95);
  box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.16);
}
.pod-status-dot--bad {
  background: rgba(239, 68, 68, 0.95);
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.16);
}
.pod-status-dot--muted {
  background: rgba(148, 163, 184, 0.9);
  box-shadow: 0 0 0 3px rgba(148, 163, 184, 0.14);
}

.pod-section-card--schedule-result {
  --k8s-card-accent: rgba(16, 185, 129, 0.85);
}
.pod-section-card--schedule-rules {
  --k8s-card-accent: rgba(168, 85, 247, 0.85);
}
.pod-section-card--schedule-taints {
  --k8s-card-accent: rgba(245, 158, 11, 0.85);
}

.pod-qos-tag {
  font-weight: 900;
}

.pod-detail-action-btn {
  font-weight: 900;
  background-image: none !important;
  background-color: rgba(37, 99, 235, 0.08) !important;
  border-color: rgba(37, 99, 235, 0.22) !important;
  color: rgba(37, 99, 235, 0.92) !important;
}
.pod-detail-action-btn:hover,
.pod-detail-action-btn:focus {
  background-color: rgba(37, 99, 235, 0.12) !important;
  border-color: rgba(37, 99, 235, 0.28) !important;
  color: rgba(37, 99, 235, 0.98) !important;
}
:global(html.dark) .pod-detail-action-btn {
  background-image: none !important;
  background-color: rgba(96, 165, 250, 0.14) !important;
  border-color: rgba(96, 165, 250, 0.24) !important;
  color: rgba(226, 232, 240, 0.92) !important;
}
:global(html.dark) .pod-detail-action-btn:hover,
:global(html.dark) .pod-detail-action-btn:focus {
  background-color: rgba(96, 165, 250, 0.18) !important;
  border-color: rgba(96, 165, 250, 0.3) !important;
  color: rgba(226, 232, 240, 0.96) !important;
}

.pod-related-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 10px;
}

.pod-related-table :deep(.el-table__cell) {
  vertical-align: top;
}
.related-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
}
.related-row-icon {
  width: 22px;
  height: 22px;
  flex: 0 0 auto;
  border-radius: 6px;
  padding: 2px;
  border: 1px solid rgba(2, 6, 23, 0.06);
  background: rgba(255, 255, 255, 0.85);
}
:global(html.dark) .related-row-icon {
  border-color: rgba(226, 232, 240, 0.12);
  background: rgba(2, 6, 23, 0.35);
}
.related-row-main {
  min-width: 0;
}
.related-row-top {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.related-row-kind {
  flex: 0 0 auto;
}
.related-row-name {
  font-weight: 900;
  color: rgba(2, 6, 23, 0.86);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
:global(html.dark) .related-row-name {
  color: rgba(226, 232, 240, 0.92);
}
.related-row-summary {
  margin-top: 2px;
  font-size: 12px;
  color: rgba(2, 6, 23, 0.56);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
:global(html.dark) .related-row-summary {
  color: rgba(226, 232, 240, 0.62);
}

.pod-related-card {
  border-radius: 16px;
  border: 1px solid rgba(2, 6, 23, 0.06);
  background: rgba(255, 255, 255, 0.72);
}
:global(html.dark) .pod-related-card {
  border-color: rgba(226, 232, 240, 0.08);
  background: rgba(2, 6, 23, 0.35);
}
.pod-related-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.related-subtitle {
  margin-top: 2px;
  font-size: 12px;
  font-weight: 900;
  color: rgba(2, 6, 23, 0.62);
}
:global(html.dark) .related-subtitle {
  color: rgba(226, 232, 240, 0.68);
}
.related-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 14px;
  border: 1px solid rgba(2, 6, 23, 0.06);
  background: rgba(255, 255, 255, 0.58);
  transition: transform 120ms ease, background 120ms ease, border-color 120ms ease;
}
.related-item:hover {
  background: rgba(59, 130, 246, 0.06);
  border-color: rgba(59, 130, 246, 0.18);
  transform: translateY(-1px);
}
:global(html.dark) .related-item {
  border-color: rgba(226, 232, 240, 0.08);
  background: rgba(2, 6, 23, 0.35);
}
:global(html.dark) .related-item:hover {
  background: rgba(96, 165, 250, 0.12);
  border-color: rgba(96, 165, 250, 0.22);
}
.related-item-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1 1 auto;
}
.related-item-icon {
  width: 22px;
  height: 22px;
  flex: 0 0 auto;
  border-radius: 6px;
  padding: 2px;
  border: 1px solid rgba(2, 6, 23, 0.06);
  background: rgba(255, 255, 255, 0.85);
}
:global(html.dark) .related-item-icon {
  border-color: rgba(226, 232, 240, 0.12);
  background: rgba(2, 6, 23, 0.35);
}
.related-item-main {
  min-width: 0;
}
.related-item-top {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.related-kind-tag {
  flex: 0 0 auto;
}
.related-item-kind {
  font-size: 12px;
  color: rgba(2, 6, 23, 0.56);
  font-weight: 900;
}
:global(html.dark) .related-item-kind {
  color: rgba(226, 232, 240, 0.62);
}
.related-item-name {
  margin-top: 0;
  font-weight: 900;
  color: rgba(2, 6, 23, 0.86);
  max-width: 520px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
:global(html.dark) .related-item-name {
  color: rgba(226, 232, 240, 0.92);
}
.related-item-sub {
  margin-top: 2px;
  font-size: 12px;
  color: rgba(2, 6, 23, 0.56);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
:global(html.dark) .related-item-sub {
  color: rgba(226, 232, 240, 0.62);
}

.pod-detail-pane-actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 10px;
}
.pod-pane-top {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 10px;
}
.pod-two-col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  align-items: start;
}
@media (max-width: 1024px) {
  .pod-two-col {
    grid-template-columns: 1fr;
  }
}

.pod-one-col {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.pod-box {
  border-radius: 16px;
  border: 1px solid rgba(2, 6, 23, 0.06);
  background: rgba(255, 255, 255, 0.58);
  padding: 10px 12px 12px;
}
:global(html.dark) .pod-box {
  border-color: rgba(226, 232, 240, 0.08);
  background: rgba(2, 6, 23, 0.35);
}

.pod-box--selector {
  border-color: rgba(37, 99, 235, 0.14);
  background: linear-gradient(180deg, rgba(37, 99, 235, 0.08), rgba(255, 255, 255, 0.52));
}
.pod-box--affinity {
  border-color: rgba(168, 85, 247, 0.14);
  background: linear-gradient(180deg, rgba(168, 85, 247, 0.08), rgba(255, 255, 255, 0.52));
}
.pod-box--taints {
  border-color: rgba(239, 68, 68, 0.14);
  background: linear-gradient(180deg, rgba(239, 68, 68, 0.08), rgba(255, 255, 255, 0.52));
}
.pod-box--tolerations {
  border-color: rgba(245, 158, 11, 0.14);
  background: linear-gradient(180deg, rgba(245, 158, 11, 0.08), rgba(255, 255, 255, 0.52));
}

:global(html.dark) .pod-box--selector {
  border-color: rgba(37, 99, 235, 0.18);
  background: linear-gradient(180deg, rgba(37, 99, 235, 0.16), rgba(2, 6, 23, 0.28));
}
:global(html.dark) .pod-box--affinity {
  border-color: rgba(168, 85, 247, 0.18);
  background: linear-gradient(180deg, rgba(168, 85, 247, 0.16), rgba(2, 6, 23, 0.28));
}
:global(html.dark) .pod-box--taints {
  border-color: rgba(239, 68, 68, 0.18);
  background: linear-gradient(180deg, rgba(239, 68, 68, 0.16), rgba(2, 6, 23, 0.28));
}
:global(html.dark) .pod-box--tolerations {
  border-color: rgba(245, 158, 11, 0.2);
  background: linear-gradient(180deg, rgba(245, 158, 11, 0.16), rgba(2, 6, 23, 0.28));
}
.pod-box-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  font-weight: 900;
  color: rgba(2, 6, 23, 0.62);
  margin-bottom: 8px;
}
.pod-box-title::before {
  content: '';
  width: 4px;
  height: 14px;
  border-radius: 99px;
  background: rgba(148, 163, 184, 0.8);
  flex: 0 0 auto;
}
.pod-box--selector .pod-box-title::before {
  background: rgba(37, 99, 235, 0.95);
}
.pod-box--affinity .pod-box-title::before {
  background: rgba(168, 85, 247, 0.95);
}
.pod-box--taints .pod-box-title::before {
  background: rgba(239, 68, 68, 0.95);
}
.pod-box--tolerations .pod-box-title::before {
  background: rgba(245, 158, 11, 0.95);
}
:global(html.dark) .pod-box-title {
  color: rgba(226, 232, 240, 0.68);
}
.pod-pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.45;
  color: rgba(2, 6, 23, 0.86);
}
:global(html.dark) .pod-pre {
  color: rgba(226, 232, 240, 0.9);
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  opacity: 0.92;
}

.pod-empty {
  font-size: 12px;
  color: rgba(2, 6, 23, 0.56);
}
:global(html.dark) .pod-empty {
  color: rgba(226, 232, 240, 0.62);
}

.pod-kv-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 8px;
  align-items: start;
}
.pod-kv-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 6px 10px;
  align-items: center;
  padding: 8px 10px;
  border-radius: 12px;
  border: 1px dashed rgba(2, 6, 23, 0.12);
  background: rgba(255, 255, 255, 0.5);
  min-width: 0;
}
:global(html.dark) .pod-kv-item {
  border-color: rgba(226, 232, 240, 0.14);
  background: rgba(2, 6, 23, 0.18);
}
.pod-kv-tone-0 {
  border-color: rgba(37, 99, 235, 0.22);
  background: rgba(37, 99, 235, 0.08);
}
.pod-kv-tone-1 {
  border-color: rgba(16, 185, 129, 0.22);
  background: rgba(16, 185, 129, 0.08);
}
.pod-kv-tone-2 {
  border-color: rgba(168, 85, 247, 0.22);
  background: rgba(168, 85, 247, 0.08);
}
.pod-kv-tone-3 {
  border-color: rgba(245, 158, 11, 0.25);
  background: rgba(245, 158, 11, 0.08);
}
.pod-kv-tone-4 {
  border-color: rgba(239, 68, 68, 0.22);
  background: rgba(239, 68, 68, 0.08);
}
.pod-kv-tone-5 {
  border-color: rgba(148, 163, 184, 0.22);
  background: rgba(148, 163, 184, 0.08);
}
.pod-kv-key {
  font-weight: 900;
  color: rgba(2, 6, 23, 0.68);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
:global(html.dark) .pod-kv-key {
  color: rgba(226, 232, 240, 0.68);
}
.pod-kv-val {
  font-weight: 900;
  color: rgba(2, 6, 23, 0.86);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: right;
}
:global(html.dark) .pod-kv-val {
  color: rgba(226, 232, 240, 0.92);
}

.pod-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.pod-chip {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: baseline;
  padding: 6px 10px;
  border-radius: 999px;
  border: 1px solid rgba(2, 6, 23, 0.06);
  background: rgba(255, 255, 255, 0.58);
  color: rgba(2, 6, 23, 0.86);
  max-width: 100%;
  box-shadow: 0 1px 0 rgba(2, 6, 23, 0.04);
}
:global(html.dark) .pod-chip {
  border-color: rgba(226, 232, 240, 0.08);
  background: rgba(2, 6, 23, 0.24);
  color: rgba(226, 232, 240, 0.9);
  box-shadow: 0 1px 0 rgba(0, 0, 0, 0.22);
}

.pod-chip.pod-kv-tone-0 {
  border-color: rgba(37, 99, 235, 0.22);
  background: rgba(37, 99, 235, 0.1);
}
.pod-chip.pod-kv-tone-1 {
  border-color: rgba(16, 185, 129, 0.22);
  background: rgba(16, 185, 129, 0.1);
}
.pod-chip.pod-kv-tone-2 {
  border-color: rgba(168, 85, 247, 0.22);
  background: rgba(168, 85, 247, 0.1);
}
.pod-chip.pod-kv-tone-3 {
  border-color: rgba(245, 158, 11, 0.25);
  background: rgba(245, 158, 11, 0.1);
}
.pod-chip.pod-kv-tone-4 {
  border-color: rgba(239, 68, 68, 0.22);
  background: rgba(239, 68, 68, 0.1);
}
.pod-chip.pod-kv-tone-5 {
  border-color: rgba(148, 163, 184, 0.22);
  background: rgba(148, 163, 184, 0.1);
}

:global(html.dark) .pod-chip.pod-kv-tone-0 {
  border-color: rgba(37, 99, 235, 0.28);
  background: rgba(37, 99, 235, 0.18);
}
:global(html.dark) .pod-chip.pod-kv-tone-1 {
  border-color: rgba(16, 185, 129, 0.28);
  background: rgba(16, 185, 129, 0.18);
}
:global(html.dark) .pod-chip.pod-kv-tone-2 {
  border-color: rgba(168, 85, 247, 0.3);
  background: rgba(168, 85, 247, 0.18);
}
:global(html.dark) .pod-chip.pod-kv-tone-3 {
  border-color: rgba(245, 158, 11, 0.3);
  background: rgba(245, 158, 11, 0.18);
}
:global(html.dark) .pod-chip.pod-kv-tone-4 {
  border-color: rgba(239, 68, 68, 0.3);
  background: rgba(239, 68, 68, 0.18);
}
:global(html.dark) .pod-chip.pod-kv-tone-5 {
  border-color: rgba(148, 163, 184, 0.3);
  background: rgba(148, 163, 184, 0.14);
}

.pod-chip-k {
  font-weight: 900;
  color: rgba(2, 6, 23, 0.72);
}
:global(html.dark) .pod-chip-k {
  color: rgba(226, 232, 240, 0.72);
}

.pod-chip-sep {
  opacity: 0.6;
}

.pod-chip-v {
  font-weight: 900;
  word-break: break-word;
}

.pod-rule-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.pod-rule {
  border-radius: 14px;
  border: 1px solid rgba(2, 6, 23, 0.06);
  background: rgba(255, 255, 255, 0.58);
  padding: 10px;
}
:global(html.dark) .pod-rule {
  border-color: rgba(226, 232, 240, 0.08);
  background: rgba(2, 6, 23, 0.24);
}

.pod-box--selector .pod-rule {
  border-color: rgba(37, 99, 235, 0.16);
  background: rgba(37, 99, 235, 0.06);
}
.pod-box--affinity .pod-rule {
  border-color: rgba(168, 85, 247, 0.16);
  background: rgba(168, 85, 247, 0.06);
}
.pod-box--taints .pod-rule {
  border-color: rgba(239, 68, 68, 0.16);
  background: rgba(239, 68, 68, 0.06);
}
.pod-box--tolerations .pod-rule {
  border-color: rgba(245, 158, 11, 0.18);
  background: rgba(245, 158, 11, 0.06);
}

:global(html.dark) .pod-box--selector .pod-rule {
  border-color: rgba(37, 99, 235, 0.22);
  background: rgba(37, 99, 235, 0.1);
}
:global(html.dark) .pod-box--affinity .pod-rule {
  border-color: rgba(168, 85, 247, 0.22);
  background: rgba(168, 85, 247, 0.1);
}
:global(html.dark) .pod-box--taints .pod-rule {
  border-color: rgba(239, 68, 68, 0.22);
  background: rgba(239, 68, 68, 0.1);
}
:global(html.dark) .pod-box--tolerations .pod-rule {
  border-color: rgba(245, 158, 11, 0.24);
  background: rgba(245, 158, 11, 0.1);
}

.pod-rule-title {
  font-weight: 900;
  color: rgba(2, 6, 23, 0.72);
  margin-bottom: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
:global(html.dark) .pod-rule-title {
  color: rgba(226, 232, 240, 0.72);
}

.pod-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.pod-list-item {
  padding: 6px 8px;
  border-radius: 10px;
  border: 1px solid rgba(2, 6, 23, 0.06);
  background: rgba(255, 255, 255, 0.58);
  color: rgba(2, 6, 23, 0.86);
  overflow: hidden;
  text-overflow: ellipsis;
}
:global(html.dark) .pod-list-item {
  border-color: rgba(226, 232, 240, 0.08);
  background: rgba(2, 6, 23, 0.24);
  color: rgba(226, 232, 240, 0.9);
}
</style>
