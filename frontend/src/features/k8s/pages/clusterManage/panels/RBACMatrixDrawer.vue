<template>
  <el-dialog
    v-model="visible"
    title="最小权限 RBAC 可视化配置"
    fullscreen
    :close-on-click-modal="false"
    append-to-body
    class="rbac-matrix-dialog"
    destroy-on-close
  >
    <div class="rbac-shell">
      <div class="rbac-main">
        <section class="control-bar">
          <div class="control-bar__main">
            <div class="control-bar__field control-bar__field--compact">
              <span class="ctrl-label">ServiceAccount</span>
              <el-input v-model="matrix.service_account" size="small" class="ctrl-input ctrl-input--sa" placeholder="service account" />
            </div>

            <div class="control-bar__field control-bar__field--compact">
              <span class="ctrl-label">命名空间</span>
              <el-input v-model="matrix.sa_namespace" size="small" class="ctrl-input ctrl-input--ns" placeholder="namespace" />
            </div>

            <div class="control-bar__field control-bar__field--grow">
              <span class="ctrl-label">目标命名空间</span>
              <div class="ctrl-ns-field">
                <el-select
                  v-model="matrix.target_namespaces"
                  multiple
                  collapse-tags
                  collapse-tags-tooltip
                  :max-collapse-tags="1"
                  allow-create
                  filterable
                  clearable
                  :reserve-keyword="false"
                  size="small"
                  class="ctrl-ns-select"
                  popper-class="rbac-ns-select-popper"
                  no-data-text="暂无可选命名空间，可直接输入"
                  placeholder="选择或输入命名空间"
                >
                  <el-option v-for="ns in availableNamespaces" :key="ns" :label="ns" :value="ns" />
                </el-select>
              </div>
            </div>
          </div>

          <div class="control-bar__meta">
            <div class="ctrl-stats">
              <span class="ctrl-stat"><strong>{{ clusterRows.length }}</strong><em>集群</em></span>
              <span class="ctrl-stat"><strong>{{ namespaceRows.length }}</strong><em>命名空间</em></span>
              <span class="ctrl-stat" :class="!hasExplicitNamespaces ? 'ctrl-stat--warn' : ''"><strong>{{ targetNamespaceCount }}</strong><em>目标 NS</em></span>
            </div>
            <div class="ctrl-actions">
              <span v-if="!hasExplicitNamespaces" class="ctrl-hint">未选择时回退到 default</span>
              <el-button size="small" :icon="Refresh" @click="resetToDefault(true, true)">恢复推荐</el-button>
            </div>
          </div>
        </section>

        <section class="matrix-card">
          <div class="matrix-card__head">
            <div>
              <div class="matrix-card__title-row">
                <span class="scope-badge scope-badge--cluster">ClusterRole</span>
                <span class="matrix-card__title">集群级资源权限</span>
                <el-tooltip content="通过 ClusterRole 与 ClusterRoleBinding 授予，适用于节点、命名空间、集群级发现与存储资源。" placement="top">
                  <el-icon class="tip-icon"><QuestionFilled /></el-icon>
                </el-tooltip>
              </div>
              <div class="matrix-card__desc">保留必须的发现、查看和少量控制能力，避免使用宽泛的内置管理员角色。</div>
            </div>
            <div class="matrix-card__chips">
              <span class="chip">{{ clusterRows.length }} 行</span>
              <span class="chip">{{ clusterPermissionCount }} 项已授权</span>
            </div>
          </div>

          <div class="matrix-scroll">
            <table class="perm-table">
              <thead>
                <tr>
                  <th class="th-head th-sticky-col th-res">资源</th>
                  <th class="th-head th-sticky-col th-grp">API Group</th>
                  <th v-for="verb in ALL_VERBS" :key="verb" :class="['th-head', 'th-verb', `th-verb--${verbGroup(verb)}`]">{{ verb }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, index) in clusterRows" :key="rowKey(row, index)" :class="['perm-row', index % 2 ? 'perm-row--alt' : '']">
                  <td class="td-cell td-sticky-col td-res">
                    <div class="resource-name">{{ row.label }}</div>
                    <div class="resource-path">{{ row.resources.join(', ') }}</div>
                  </td>
                  <td class="td-cell td-sticky-col td-grp">
                    <code class="api-code">{{ row.api_group || 'core' }}</code>
                  </td>
                  <td
                    v-for="verb in ALL_VERBS"
                    :key="verb"
                    :class="['td-cell', 'td-verb', `td-verb--${verbGroup(verb)}`, isVerbDisabled(row, verb) ? 'td-verb--na' : '']"
                  >
                    <span v-if="isVerbDisabled(row, verb)" class="na-mark">—</span>
                    <button
                      v-else
                      type="button"
                      class="verb-toggle"
                      :class="row.verbs.includes(verb) ? `verb-toggle--on verb-toggle--${verbGroup(verb)}` : 'verb-toggle--off'"
                      :title="row.verbs.includes(verb) ? `撤销 ${verb}` : `授予 ${verb}`"
                      @click="toggleVerb(row, verb, !row.verbs.includes(verb))"
                    >
                      <svg v-if="row.verbs.includes(verb)" class="verb-toggle__icon" viewBox="0 0 12 12" aria-hidden="true">
                        <polyline points="1.5,6 4.5,9.5 10.5,2.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
                      </svg>
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="matrix-card">
          <div class="matrix-card__head">
            <div>
              <div class="matrix-card__title-row">
                <span class="scope-badge scope-badge--namespace">RoleBinding</span>
                <span class="matrix-card__title">命名空间资源权限</span>
                <el-tooltip content="通过 ClusterRole 与目标命名空间 RoleBinding 授予，适合部署、服务发现、日志和终端访问等日常操作。" placement="top">
                  <el-icon class="tip-icon"><QuestionFilled /></el-icon>
                </el-tooltip>
              </div>
              <div class="matrix-card__desc">命名空间表保留工作负载、服务、Ingress、日志与扩缩容所需能力，尽量不把权限扩散到集群范围。</div>
            </div>
            <div class="matrix-card__chips">
              <span class="chip">{{ namespaceRows.length }} 行</span>
              <span class="chip">{{ namespacePermissionCount }} 项已授权</span>
            </div>
          </div>

          <div class="matrix-scroll">
            <table class="perm-table">
              <thead>
                <tr>
                  <th class="th-head th-sticky-col th-res">资源</th>
                  <th class="th-head th-sticky-col th-grp">API Group</th>
                  <th v-for="verb in ALL_VERBS" :key="verb" :class="['th-head', 'th-verb', `th-verb--${verbGroup(verb)}`]">{{ verb }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, index) in namespaceRows" :key="rowKey(row, index)" :class="['perm-row', index % 2 ? 'perm-row--alt' : '']">
                  <td class="td-cell td-sticky-col td-res">
                    <div class="resource-name">{{ row.label }}</div>
                    <div class="resource-path">{{ row.resources.join(', ') }}</div>
                  </td>
                  <td class="td-cell td-sticky-col td-grp">
                    <code class="api-code">{{ row.api_group || 'core' }}</code>
                  </td>
                  <td
                    v-for="verb in ALL_VERBS"
                    :key="verb"
                    :class="['td-cell', 'td-verb', `td-verb--${verbGroup(verb)}`, isVerbDisabled(row, verb) ? 'td-verb--na' : '']"
                  >
                    <span v-if="isVerbDisabled(row, verb)" class="na-mark">—</span>
                    <button
                      v-else
                      type="button"
                      class="verb-toggle"
                      :class="row.verbs.includes(verb) ? `verb-toggle--on verb-toggle--${verbGroup(verb)}` : 'verb-toggle--off'"
                      :title="row.verbs.includes(verb) ? `撤销 ${verb}` : `授予 ${verb}`"
                      @click="toggleVerb(row, verb, !row.verbs.includes(verb))"
                    >
                      <svg v-if="row.verbs.includes(verb)" class="verb-toggle__icon" viewBox="0 0 12 12" aria-hidden="true">
                        <polyline points="1.5,6 4.5,9.5 10.5,2.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
                      </svg>
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="legend-card">
          <div class="legend-card__title">操作说明</div>
          <div class="legend-list">
            <span class="legend-item"><span class="legend-dot legend-dot--read" />只读操作：get / list / watch</span>
            <span class="legend-item"><span class="legend-dot legend-dot--write" />变更操作：create / update / patch</span>
            <span class="legend-item"><span class="legend-dot legend-dot--delete" />删除操作：delete</span>
            <span class="legend-item"><span class="na-sample">—</span> 当前资源不支持该操作</span>
          </div>
        </section>
      </div>

      <section class="preview-panel">
        <div class="preview-panel__head">
          <div class="preview-head-left">
            <span class="preview-panel__title">YAML 预览</span>
            <span class="preview-metric-inline"><strong>{{ clusterPermissionCount }}</strong> 集群 · <strong>{{ namespacePermissionCount }}</strong> 命名空间</span>
            <span class="preview-sync-badge" :class="`preview-sync-badge--${yamlSyncState}`">{{ yamlStatusText }}</span>
          </div>
          <div class="preview-actions">
            <el-button size="small" text @click="yamlPreviewExpanded = !yamlPreviewExpanded">
              <el-icon><component :is="yamlPreviewExpanded ? CaretBottom : CaretRight" /></el-icon>
              {{ yamlPreviewExpanded ? '收起预览' : '展开预览' }}
            </el-button>
            <el-button size="small" text :loading="yamlLoading" @click="refreshYaml">
              <el-icon><RefreshRight /></el-icon>
            </el-button>
            <el-button size="small" :icon="DocumentCopy" :disabled="!canUseYaml" @click="copyYaml">复制</el-button>
            <el-button size="small" :icon="Download" :disabled="!canUseYaml" @click="downloadYaml">下载</el-button>
          </div>
        </div>

        <div v-if="yamlPreviewExpanded" class="preview-body preview-body--full">
          <div v-if="yamlLoading && !yamlContent" class="yaml-loading">
            <el-icon class="is-loading"><Loading /></el-icon>
            <span>正在生成 YAML…</span>
          </div>
          <template v-else>
            <div v-if="yamlSyncState === 'error'" class="yaml-warning">{{ yamlWarningText }}</div>
            <CodeMirrorViewer :text="yamlContent" language="yaml" height="auto" theme="auto" />
          </template>
        </div>
        <div v-else class="preview-collapsed">
          <div class="preview-collapsed__title">YAML 预览默认收起</div>
          <div class="preview-collapsed__desc">点击右上角“展开预览”查看完整 YAML 内容，避免页面被大段文本挤压。</div>
        </div>
      </section>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { CaretBottom, CaretRight, DocumentCopy, Download, Loading, QuestionFilled, Refresh, RefreshRight } from '@element-plus/icons-vue'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import * as permissionAuditApi from '@/features/k8s/api/permissionAudit'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'

const props = defineProps<{
  clusterId: number
  namespaces: string[]
  initialNamespaces?: string[]
}>()

const visible = defineModel<boolean>({ default: false })

const ALL_VERBS = ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] as const

const VERB_INAPPLICABLE: Record<string, string[]> = {
  'nodes/status': ['create', 'update', 'patch', 'delete'],
  'pods/log': ['list', 'watch', 'create', 'update', 'patch', 'delete'],
  'pods/exec': ['get', 'list', 'watch', 'update', 'patch', 'delete'],
  'pods/eviction': ['get', 'list', 'watch', 'update', 'patch', 'delete'],
  'deployments/scale': ['list', 'watch', 'create', 'delete'],
  'statefulsets/scale': ['list', 'watch', 'create', 'delete'],
  endpoints: ['create', 'update', 'patch', 'delete'],
  events: ['create', 'update', 'patch', 'delete'],
  'metrics.k8s.io/nodes': ['watch', 'create', 'update', 'patch', 'delete'],
  'metrics.k8s.io/pods': ['watch', 'create', 'update', 'patch', 'delete'],
}

const matrix = ref<permissionAuditApi.RBACMatrixRequest>({
  service_account: 'xingku-platform',
  sa_namespace: 'kube-system',
  target_namespaces: props.initialNamespaces ?? [...props.namespaces],
  cluster_rows: [],
  namespace_rows: [],
})

const yamlContent = ref('')
const yamlLoading = ref(false)
const yamlSyncState = ref<'idle' | 'syncing' | 'synced' | 'error'>('idle')
const yamlPreviewExpanded = ref(false)
let yamlDebounce: ReturnType<typeof setTimeout> | null = null
let latestYamlRequestId = 0
let latestMatrixRequestId = 0
let suppressYamlRefresh = false

const clusterRows = computed(() => matrix.value.cluster_rows)
const namespaceRows = computed(() => matrix.value.namespace_rows)
const normalizedTargetNamespaces = computed(() => normalizeNamespaces(matrix.value.target_namespaces))
const effectiveTargetNamespaces = computed(() => resolveTargetNamespaces(matrix.value.target_namespaces))
const targetNamespaceCount = computed(() => effectiveTargetNamespaces.value.length)
const availableNamespaces = computed(() => Array.from(new Set([...props.namespaces, ...matrix.value.target_namespaces])))
const hasExplicitNamespaces = computed(() => normalizedTargetNamespaces.value.length > 0)
const clusterPermissionCount = computed(() => countEnabledVerbs(clusterRows.value))
const namespacePermissionCount = computed(() => countEnabledVerbs(namespaceRows.value))
const yamlStatusText = computed(() => {
  if (yamlSyncState.value === 'syncing') return '生成中'
  if (yamlSyncState.value === 'error') return '同步失败'
  if (yamlSyncState.value === 'synced') return '已同步'
  return '待生成'
})
const canUseYaml = computed(() => yamlSyncState.value === 'synced' && !!yamlContent.value)
const yamlWarningText = computed(() => (yamlContent.value ? 'YAML 同步失败，当前显示的是上一次成功生成的结果。' : 'YAML 同步失败，请重试或恢复推荐。'))

function normalizeNamespaces(list: string[]): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const item of list) {
    const normalized = item.trim()
    if (!normalized || seen.has(normalized)) continue
    seen.add(normalized)
    result.push(normalized)
  }
  return result
}

function resolveTargetNamespaces(list: string[]): string[] {
  const normalized = normalizeNamespaces(list)
  return normalized.length ? normalized : ['default']
}

function isSameStringArray(left: string[], right: string[]): boolean {
  if (left.length !== right.length) return false
  return left.every((item, index) => item === right[index])
}

function verbGroup(verb: string): 'read' | 'write' | 'delete' {
  if (verb === 'get' || verb === 'list' || verb === 'watch') return 'read'
  if (verb === 'delete') return 'delete'
  return 'write'
}

function rowKey(row: permissionAuditApi.RBACMatrixRow, index: number): string {
  return `${index}-${row.api_group}-${row.resources.join('.')}-${row.label}`
}

function countEnabledVerbs(rows: permissionAuditApi.RBACMatrixRow[]): number {
  return rows.reduce((sum, row) => sum + row.verbs.length, 0)
}

function isVerbDisabled(row: permissionAuditApi.RBACMatrixRow, verb: string): boolean {
  if (row.resources.length !== 1) return false
  const resource = row.resources[0]
  const key = row.api_group ? `${row.api_group}/${resource}` : resource
  const blocked = VERB_INAPPLICABLE[key] ?? VERB_INAPPLICABLE[resource]
  return blocked ? blocked.includes(verb) : false
}

function toggleVerb(row: permissionAuditApi.RBACMatrixRow, verb: string, enabled: boolean) {
  if (enabled) {
    if (!row.verbs.includes(verb)) row.verbs.push(verb)
  } else {
    row.verbs = row.verbs.filter((item) => item !== verb)
  }
  scheduleYaml()
}

function scheduleYaml() {
  if (!visible.value || suppressYamlRefresh) return
  if (yamlDebounce) clearTimeout(yamlDebounce)
  yamlDebounce = setTimeout(refreshYaml, 320)
}

async function refreshYaml() {
  if (!visible.value) return
  if (yamlDebounce) {
    clearTimeout(yamlDebounce)
    yamlDebounce = null
  }

  const requestId = ++latestYamlRequestId
  yamlLoading.value = true
  yamlSyncState.value = 'syncing'
  try {
    const res = await permissionAuditApi.buildRBACFromMatrix(props.clusterId, {
      ...matrix.value,
      target_namespaces: effectiveTargetNamespaces.value,
    })
    if (requestId !== latestYamlRequestId) return
    yamlContent.value = res.yaml_content
    yamlSyncState.value = 'synced'
  } catch (error) {
    if (requestId !== latestYamlRequestId) return
    yamlSyncState.value = 'error'
    notifyError((error as ApiError).message)
  } finally {
    if (requestId === latestYamlRequestId) {
      yamlLoading.value = false
    }
  }
}

function clearYamlState() {
  latestYamlRequestId += 1
  yamlContent.value = ''
  yamlLoading.value = false
  yamlSyncState.value = 'idle'
}

function clearMatrixState(resetScopeAndIdentity = false) {
  const defaultNamespaces = getDefaultMatrixNamespaces(false)
  matrix.value = {
    ...matrix.value,
    service_account: resetScopeAndIdentity ? 'xingku-platform' : matrix.value.service_account,
    sa_namespace: resetScopeAndIdentity ? 'kube-system' : matrix.value.sa_namespace,
    target_namespaces: resetScopeAndIdentity ? defaultNamespaces : matrix.value.target_namespaces,
    cluster_rows: [],
    namespace_rows: [],
  }
}

function getDefaultMatrixNamespaces(preferCurrentSelection: boolean): string[] {
  if (preferCurrentSelection) return effectiveTargetNamespaces.value

  const initialNamespaces = normalizeNamespaces(props.initialNamespaces?.length ? props.initialNamespaces : props.namespaces)
  return initialNamespaces.length ? initialNamespaces : ['default']
}

async function resetToDefault(preferCurrentSelection = false, preserveStateOnError = false, scheduleAfterLoad = true) {
  const requestId = ++latestMatrixRequestId
  try {
    const requestNamespaces = getDefaultMatrixNamespaces(preferCurrentSelection)
    const data = await permissionAuditApi.getDefaultRBACMatrix(
      props.clusterId,
      requestNamespaces,
    )
    if (requestId !== latestMatrixRequestId || !visible.value) return
    matrix.value = {
      ...data,
      target_namespaces: resolveTargetNamespaces(data.target_namespaces),
    }
    if (scheduleAfterLoad) scheduleYaml()
    return true
  } catch (error) {
    if (requestId !== latestMatrixRequestId) return
    if (!preserveStateOnError) {
      clearMatrixState()
      matrix.value.target_namespaces = getDefaultMatrixNamespaces(preferCurrentSelection)
      clearYamlState()
      yamlSyncState.value = 'error'
    } else {
      yamlSyncState.value = 'error'
    }
    notifyError((error as ApiError).message)
    return false
  }
}

async function onOpen() {
  suppressYamlRefresh = true
  yamlPreviewExpanded.value = false
  clearMatrixState(true)
  clearYamlState()
  yamlLoading.value = true
  yamlSyncState.value = 'syncing'
  const loaded = await resetToDefault(false, false, false)
  suppressYamlRefresh = false
  if (loaded) scheduleYaml()
}

watch(
  visible,
  (isOpen) => {
    if (isOpen) {
      void onOpen()
      return
    }

    if (yamlDebounce) {
      clearTimeout(yamlDebounce)
      yamlDebounce = null
    }
    latestYamlRequestId += 1
    latestMatrixRequestId += 1
    yamlLoading.value = false
    yamlPreviewExpanded.value = false
  },
  { flush: 'sync' },
)

watch(
  () => props.namespaces,
  (value) => {
    if (!matrix.value.target_namespaces.length && value.length) {
      matrix.value.target_namespaces = normalizeNamespaces(value)
    }
  },
  { deep: true },
)

watch(
  () => matrix.value.target_namespaces,
  (value) => {
    const normalized = normalizeNamespaces(value)
    if (!isSameStringArray(value, normalized)) {
      matrix.value.target_namespaces = normalized
      return
    }
    scheduleYaml()
  },
  { deep: true },
)
watch(() => matrix.value.service_account, () => scheduleYaml())
watch(() => matrix.value.sa_namespace, () => scheduleYaml())

function copyYaml() {
  navigator.clipboard.writeText(yamlContent.value)
    .then(() => notifySuccess('YAML 已复制到剪贴板'))
    .catch(() => notifyError('复制失败，请手动选择 YAML 内容'))
}

function downloadYaml() {
  const blob = new Blob([yamlContent.value], { type: 'text/yaml' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `minimum-rbac-${matrix.value.service_account}.yaml`
  anchor.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
:deep(.rbac-matrix-dialog .el-dialog__header) {
  padding: 10px 18px 8px;
  border-bottom: 1px solid var(--color-border-subtle);
}

:deep(.rbac-matrix-dialog .el-dialog__title) {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text-primary);
}

:deep(.rbac-matrix-dialog .el-dialog__body) {
  padding: 0;
  overflow-y: auto;
}

.rbac-shell {
  --rbac-shell-bg: linear-gradient(180deg, rgba(248, 250, 252, 0.92) 0%, rgba(241, 245, 249, 0.98) 100%);
  --rbac-surface-bg: rgba(255, 255, 255, 0.96);
  --rbac-surface-border: rgba(148, 163, 184, 0.18);
  --rbac-surface-shadow: 0 2px 8px rgba(15, 23, 42, 0.04);
  --rbac-text-primary: var(--color-text-primary);
  --rbac-text-secondary: var(--color-text-secondary);
  --rbac-text-muted: #94a3b8;
  --rbac-pill-bg: #eef4ff;
  --rbac-pill-text: #1e3a8a;
  --rbac-pill-muted: #3b5998;
  --rbac-pill-warn-bg: #ede9fe;
  --rbac-pill-warn-text: #6d28d9;
  --rbac-hint-text: #b45309;
  --rbac-chip-bg: #f1f5f9;
  --rbac-chip-text: #334155;
  --rbac-badge-cluster-bg: #dbeafe;
  --rbac-badge-cluster-text: #1d4ed8;
  --rbac-badge-namespace-bg: #dcfce7;
  --rbac-badge-namespace-text: #15803d;
  --rbac-table-bg: #ffffff;
  --rbac-table-border: #e2e8f0;
  --rbac-table-border-strong: #dbe4f0;
  --rbac-sticky-shadow: 1px 0 0 rgba(226, 232, 240, 0.92);
  --rbac-head-sticky-bg: #f8fafc;
  --rbac-head-sticky-text: #334155;
  --rbac-head-read-bg: #eff6ff;
  --rbac-head-read-text: #1d4ed8;
  --rbac-head-write-bg: #f0fdf4;
  --rbac-head-write-text: #15803d;
  --rbac-head-delete-bg: #fff1f2;
  --rbac-head-delete-text: #be123c;
  --rbac-row-alt-bg: rgba(248, 250, 252, 0.72);
  --rbac-row-hover-bg: rgba(239, 246, 255, 0.7);
  --rbac-api-bg: #eef2ff;
  --rbac-api-text: #4338ca;
  --rbac-na-bg: #e2e8f0;
  --rbac-na-text: #64748b;
  --rbac-toggle-off-bg: #ffffff;
  --rbac-toggle-off-border: #cbd5e1;
  --rbac-toggle-off-shadow: inset 0 0 0 1px rgba(148, 163, 184, 0.08);
  --rbac-toggle-off-hover-border: #94a3b8;
  --rbac-toggle-off-hover-shadow: 0 6px 16px rgba(148, 163, 184, 0.2);
  --rbac-legend-read: #2563eb;
  --rbac-legend-write: #16a34a;
  --rbac-legend-delete: #e11d48;
  --rbac-preview-bg: linear-gradient(180deg, #ffffff 0%, #f8fbff 100%);
  --rbac-preview-head-border: #e2e8f0;
  --rbac-preview-button-bg: #ffffff;
  --rbac-preview-button-border: #dbe4f0;
  --rbac-preview-button-text: #475569;
  --rbac-preview-button-hover-bg: #eff6ff;
  --rbac-preview-button-hover-border: #bfdbfe;
  --rbac-preview-button-hover-text: #0f172a;
  --rbac-preview-collapsed-bg: #f8fafc;
  --rbac-sync-idle-bg: #f1f5f9;
  --rbac-sync-idle-text: #64748b;
  --rbac-sync-syncing-bg: #dbeafe;
  --rbac-sync-syncing-text: #2563eb;
  --rbac-sync-synced-bg: #dcfce7;
  --rbac-sync-synced-text: #15803d;
  --rbac-sync-error-bg: #fee2e2;
  --rbac-sync-error-text: #dc2626;
  --rbac-warning-bg: #fffbeb;
  --rbac-warning-border: #fde68a;
  --rbac-warning-text: #b45309;
  display: flex;
  flex-direction: column;
  min-height: calc(100vh - 54px);
  background: var(--rbac-shell-bg);
  color: var(--rbac-text-primary);
}

:global(html.dark) .rbac-shell {
  --rbac-shell-bg: linear-gradient(180deg, rgba(2, 6, 23, 0.82) 0%, rgba(15, 23, 42, 0.96) 100%);
  --rbac-surface-bg: rgba(15, 23, 42, 0.84);
  --rbac-surface-border: rgba(148, 163, 184, 0.18);
  --rbac-surface-shadow: 0 20px 40px rgba(2, 6, 23, 0.24), inset 0 1px 0 rgba(148, 163, 184, 0.06);
  --rbac-text-muted: #94a3b8;
  --rbac-pill-bg: rgba(37, 99, 235, 0.16);
  --rbac-pill-text: #93c5fd;
  --rbac-pill-muted: #bfdbfe;
  --rbac-pill-warn-bg: rgba(139, 92, 246, 0.18);
  --rbac-pill-warn-text: #c4b5fd;
  --rbac-hint-text: #fbbf24;
  --rbac-chip-bg: rgba(148, 163, 184, 0.14);
  --rbac-chip-text: #cbd5e1;
  --rbac-badge-cluster-bg: rgba(37, 99, 235, 0.18);
  --rbac-badge-cluster-text: #93c5fd;
  --rbac-badge-namespace-bg: rgba(34, 197, 94, 0.18);
  --rbac-badge-namespace-text: #86efac;
  --rbac-table-bg: rgba(15, 23, 42, 0.9);
  --rbac-table-border: rgba(148, 163, 184, 0.16);
  --rbac-table-border-strong: rgba(148, 163, 184, 0.2);
  --rbac-sticky-shadow: 1px 0 0 rgba(148, 163, 184, 0.18);
  --rbac-head-sticky-bg: rgba(15, 23, 42, 0.98);
  --rbac-head-sticky-text: #cbd5e1;
  --rbac-head-read-bg: rgba(37, 99, 235, 0.16);
  --rbac-head-read-text: #93c5fd;
  --rbac-head-write-bg: rgba(22, 163, 74, 0.16);
  --rbac-head-write-text: #86efac;
  --rbac-head-delete-bg: rgba(225, 29, 72, 0.18);
  --rbac-head-delete-text: #fda4af;
  --rbac-row-alt-bg: rgba(255, 255, 255, 0.02);
  --rbac-row-hover-bg: rgba(59, 130, 246, 0.12);
  --rbac-api-bg: rgba(79, 70, 229, 0.18);
  --rbac-api-text: #c7d2fe;
  --rbac-na-bg: rgba(148, 163, 184, 0.14);
  --rbac-na-text: #94a3b8;
  --rbac-toggle-off-bg: rgba(15, 23, 42, 0.82);
  --rbac-toggle-off-border: rgba(148, 163, 184, 0.26);
  --rbac-toggle-off-shadow: inset 0 0 0 1px rgba(148, 163, 184, 0.08);
  --rbac-toggle-off-hover-border: rgba(96, 165, 250, 0.52);
  --rbac-toggle-off-hover-shadow: 0 8px 18px rgba(2, 6, 23, 0.28);
  --rbac-legend-read: #60a5fa;
  --rbac-legend-write: #4ade80;
  --rbac-legend-delete: #fb7185;
  --rbac-preview-bg: linear-gradient(180deg, rgba(15, 23, 42, 0.94) 0%, rgba(15, 23, 42, 0.86) 100%);
  --rbac-preview-head-border: rgba(148, 163, 184, 0.16);
  --rbac-preview-button-bg: rgba(15, 23, 42, 0.78);
  --rbac-preview-button-border: rgba(148, 163, 184, 0.18);
  --rbac-preview-button-text: #cbd5e1;
  --rbac-preview-button-hover-bg: rgba(30, 41, 59, 0.96);
  --rbac-preview-button-hover-border: rgba(96, 165, 250, 0.36);
  --rbac-preview-button-hover-text: #f8fafc;
  --rbac-preview-collapsed-bg: rgba(15, 23, 42, 0.72);
  --rbac-sync-idle-bg: rgba(148, 163, 184, 0.14);
  --rbac-sync-idle-text: #cbd5e1;
  --rbac-sync-syncing-bg: rgba(37, 99, 235, 0.18);
  --rbac-sync-syncing-text: #93c5fd;
  --rbac-sync-synced-bg: rgba(34, 197, 94, 0.18);
  --rbac-sync-synced-text: #86efac;
  --rbac-sync-error-bg: rgba(239, 68, 68, 0.18);
  --rbac-sync-error-text: #fca5a5;
  --rbac-warning-bg: rgba(120, 53, 15, 0.28);
  --rbac-warning-border: rgba(245, 158, 11, 0.36);
  --rbac-warning-text: #fcd34d;
}

.rbac-main {
  min-width: 0;
  padding: 10px 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.control-bar,
.matrix-card,
.legend-card {
  border: 1px solid var(--rbac-surface-border);
  border-radius: 10px;
  background: var(--rbac-surface-bg);
  box-shadow: var(--rbac-surface-shadow);
}

.control-bar {
  padding: 8px 12px;
  flex-shrink: 0;
}

.control-bar__main {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.control-bar__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 6px;
  min-width: 0;
}

.control-bar__field {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.control-bar__field--compact {
  flex: 0 0 auto;
}

.control-bar__field--grow {
  flex: 1 1 auto;
}

.ctrl-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.03em;
  color: var(--rbac-text-secondary);
  white-space: nowrap;
  flex-shrink: 0;
}

.ctrl-input {
  flex-shrink: 0;
}

.ctrl-input--sa {
  width: 140px;
}

.ctrl-input--ns {
  width: 110px;
}

.ctrl-ns-field {
  display: flex;
  flex: 1 1 auto;
  min-width: 240px;
}

.ctrl-ns-select {
  width: 100%;
  min-width: 0;
  max-width: none;
}

:deep(.ctrl-ns-select .el-select__wrapper) {
  min-height: 32px;
}

:deep(.ctrl-ns-select .el-tag) {
  max-width: 180px;
}

:deep(.ctrl-ns-select .el-tag__content) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.rbac-ns-select-popper) {
  max-width: 420px;
}

:deep(.rbac-ns-select-popper .el-select-dropdown__item) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ctrl-stats {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  min-width: 0;
}

.ctrl-stat {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--rbac-pill-bg);
  font-size: 11px;
  color: var(--rbac-pill-text);
}

.ctrl-stat strong {
  font-size: 12px;
  font-weight: 700;
}

.ctrl-stat em {
  font-style: normal;
  color: var(--rbac-pill-muted);
  opacity: 0.75;
}

.ctrl-stat--warn {
  background: var(--rbac-pill-warn-bg);
  color: var(--rbac-pill-warn-text);
}

.ctrl-stat--warn strong,
.ctrl-stat--warn em {
  color: var(--rbac-pill-warn-text);
}

.ctrl-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.ctrl-hint {
  font-size: 11px;
  color: var(--rbac-hint-text);
  white-space: nowrap;
}

.matrix-card {
  padding: 10px 12px 10px;
}

.matrix-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
}

.matrix-card__title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.matrix-card__title {
  font-size: 14px;
  font-weight: 700;
  color: var(--rbac-text-primary);
}

.matrix-card__desc {
  margin-top: 2px;
  font-size: 12px;
  color: var(--rbac-text-secondary);
  line-height: 1.5;
}

.matrix-card__chips {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: flex-end;
  flex-shrink: 0;
}

.chip,
.scope-badge {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
}

.chip {
  background: var(--rbac-chip-bg);
  color: var(--rbac-chip-text);
}

.scope-badge--cluster {
  background: var(--rbac-badge-cluster-bg);
  color: var(--rbac-badge-cluster-text);
}

.scope-badge--namespace {
  background: var(--rbac-badge-namespace-bg);
  color: var(--rbac-badge-namespace-text);
}

.tip-icon {
  color: var(--rbac-text-muted);
  cursor: help;
}

.matrix-scroll {
  position: relative;
  overflow: visible;
  border: 1px solid var(--rbac-table-border);
  border-radius: 8px;
  background: var(--rbac-table-bg);
}

.perm-table {
  width: 100%;
  min-width: 0;
  table-layout: fixed;
  border-collapse: separate;
  border-spacing: 0;
}

.th-head {
  position: sticky;
  top: 0;
  z-index: 8;
  padding: 7px 6px;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  border-bottom: 1px solid var(--rbac-table-border-strong);
  white-space: nowrap;
}

.th-sticky-col {
  z-index: 11;
  box-shadow: var(--rbac-sticky-shadow);
}

.th-res,
.td-res {
  width: 196px;
  min-width: 196px;
  left: 0;
}

.th-grp,
.td-grp {
  width: 126px;
  min-width: 126px;
  left: 196px;
}

.th-res,
.th-grp {
  background: var(--rbac-head-sticky-bg);
  color: var(--rbac-head-sticky-text);
  text-align: left;
}

.th-verb {
  width: 52px;
  min-width: 52px;
  text-align: center;
}

.th-verb--read {
  background: var(--rbac-head-read-bg);
  color: var(--rbac-head-read-text);
}

.th-verb--write {
  background: var(--rbac-head-write-bg);
  color: var(--rbac-head-write-text);
}

.th-verb--delete {
  background: var(--rbac-head-delete-bg);
  color: var(--rbac-head-delete-text);
}

.perm-row td {
  border-bottom: 1px solid var(--rbac-table-border);
}

.perm-row--alt td {
  background: var(--rbac-row-alt-bg);
}

.perm-row:hover td {
  background: var(--rbac-row-hover-bg);
}

.td-cell {
  padding: 7px 6px;
  text-align: center;
}

.td-sticky-col {
  position: sticky;
  z-index: 6;
  background: inherit;
  box-shadow: var(--rbac-sticky-shadow);
}

.perm-row--alt .td-sticky-col {
  background: var(--rbac-row-alt-bg);
}

.perm-row:not(.perm-row--alt) .td-sticky-col {
  background: var(--rbac-table-bg);
}

.perm-row:hover .td-sticky-col {
  background: var(--rbac-row-hover-bg);
}

.td-res {
  text-align: left;
}

.resource-name {
  font-size: 13px;
  font-weight: 700;
  color: var(--rbac-text-primary);
}

.resource-path {
  margin-top: 4px;
  font-size: 11px;
  color: var(--rbac-text-muted);
}

.api-code {
  display: inline-flex;
  align-items: center;
  min-height: 20px;
  max-width: 100%;
  padding: 0 5px;
  border-radius: 999px;
  background: var(--rbac-api-bg);
  color: var(--rbac-api-text);
  font-size: 10px;
  font-family: Consolas, Monaco, monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.td-verb {
  padding: 6px 4px;
}

.td-verb--na {
  background: var(--rbac-head-sticky-bg);
}

.na-mark,
.na-sample {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 6px;
  background: var(--rbac-na-bg);
  color: var(--rbac-na-text);
  font-size: 14px;
  font-weight: 800;
}

.verb-toggle {
  width: 24px;
  height: 24px;
  border-radius: 7px;
  border: 1px solid transparent;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease, background 0.15s ease, border-color 0.15s ease;
}

.verb-toggle:hover {
  transform: translateY(-1px);
}

.verb-toggle--off {
  background: var(--rbac-toggle-off-bg);
  border-color: var(--rbac-toggle-off-border);
  box-shadow: var(--rbac-toggle-off-shadow);
}

.verb-toggle--off:hover {
  border-color: var(--rbac-toggle-off-hover-border);
  box-shadow: var(--rbac-toggle-off-hover-shadow);
}

.verb-toggle--on {
  color: #fff;
}

.verb-toggle--read {
  background: linear-gradient(180deg, #3b82f6 0%, #2563eb 100%);
  box-shadow: 0 10px 18px rgba(37, 99, 235, 0.24);
}

.verb-toggle--write {
  background: linear-gradient(180deg, #22c55e 0%, #16a34a 100%);
  box-shadow: 0 10px 18px rgba(22, 163, 74, 0.22);
}

.verb-toggle--delete {
  background: linear-gradient(180deg, #f43f5e 0%, #e11d48 100%);
  box-shadow: 0 10px 18px rgba(225, 29, 72, 0.22);
}

.verb-toggle__icon {
  width: 12px;
  height: 12px;
}

.legend-card {
  padding: 10px 14px;
}

.legend-card__title {
  font-size: 12px;
  font-weight: 700;
  color: var(--rbac-text-primary);
}

.legend-list {
  display: flex;
  gap: 10px 18px;
  flex-wrap: wrap;
  margin-top: 8px;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--rbac-text-secondary);
}

.legend-dot {
  width: 12px;
  height: 12px;
  border-radius: 4px;
}

.legend-dot--read {
  background: var(--rbac-legend-read);
}

.legend-dot--write {
  background: var(--rbac-legend-write);
}

.legend-dot--delete {
  background: var(--rbac-legend-delete);
}

.preview-panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: var(--rbac-preview-bg);
  margin: 0 12px 14px;
  border: 1px solid var(--rbac-surface-border);
  border-radius: 10px;
  overflow: hidden;
}

.preview-panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--rbac-preview-head-border);
  flex-shrink: 0;
}

.preview-head-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.preview-panel__title {
  font-size: 14px;
  font-weight: 700;
  color: var(--rbac-text-primary);
  white-space: nowrap;
}

.preview-metric-inline {
  font-size: 12px;
  color: var(--rbac-text-secondary);
  white-space: nowrap;
}

.preview-metric-inline strong {
  color: var(--rbac-text-primary);
}

.preview-sync-badge {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.preview-sync-badge--idle {
  background: var(--rbac-sync-idle-bg);
  color: var(--rbac-sync-idle-text);
}

.preview-sync-badge--syncing {
  background: var(--rbac-sync-syncing-bg);
  color: var(--rbac-sync-syncing-text);
}

.preview-sync-badge--synced {
  background: var(--rbac-sync-synced-bg);
  color: var(--rbac-sync-synced-text);
}

.preview-sync-badge--error {
  background: var(--rbac-sync-error-bg);
  color: var(--rbac-sync-error-text);
}

.preview-actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

:deep(.preview-panel .el-button) {
  color: var(--rbac-preview-button-text);
  border-color: var(--rbac-preview-button-border);
  background: var(--rbac-preview-button-bg);
}

:deep(.preview-panel .el-button:hover) {
  color: var(--rbac-preview-button-hover-text);
  border-color: var(--rbac-preview-button-hover-border);
  background: var(--rbac-preview-button-hover-bg);
}

.preview-body {
  padding: 10px 14px 14px;
}

.preview-body--full {
  display: block;
}

.preview-body :deep(.cm-host) {
  height: auto;
}

.preview-body :deep(.cm-editor) {
  height: auto;
}

.preview-collapsed {
  padding: 14px;
  background: var(--rbac-preview-collapsed-bg);
}

.preview-collapsed__title {
  font-size: 13px;
  font-weight: 700;
  color: var(--rbac-text-primary);
}

.preview-collapsed__desc {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--rbac-text-secondary);
}

.yaml-loading {
  min-height: 120px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--rbac-text-secondary);
  font-size: 13px;
}

.yaml-warning {
  margin-bottom: 10px;
  padding: 10px 12px;
  border: 1px solid var(--rbac-warning-border);
  border-radius: 10px;
  background: var(--rbac-warning-bg);
  color: var(--rbac-warning-text);
  font-size: 12px;
  line-height: 1.6;
}

@media (max-width: 1480px) {
  .control-bar__main {
    flex-wrap: wrap;
  }

  .control-bar__field--grow {
    flex-basis: 100%;
  }
}

@media (max-width: 1180px) {
  .control-bar__meta {
    flex-wrap: wrap;
  }
}

@media (max-width: 860px) {
  .rbac-main {
    padding: 10px;
  }

  .control-bar__main,
  .control-bar__meta {
    flex-wrap: wrap;
  }

  .ctrl-actions {
    width: 100%;
    justify-content: space-between;
  }

  .matrix-card__head {
    flex-direction: column;
  }

  .th-res,
  .td-res {
    width: 160px;
    min-width: 160px;
  }

  .th-grp,
  .td-grp {
    width: 100px;
    min-width: 100px;
    left: 160px;
  }

  .th-verb {
    width: 44px;
    min-width: 44px;
  }
}
</style>
