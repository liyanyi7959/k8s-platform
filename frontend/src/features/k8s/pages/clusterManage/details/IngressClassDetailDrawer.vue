<template>
  <WorkloadDetailDrawerShell
    v-model="visible"
    :title="detailTitle"
    :loading="refreshing"
    ns="-"
    :name="detailName"
    @refresh="refreshDetail"
  >
    <template #summary>
      <div class="k8s-summary-grid">
        <div class="k8s-kv k8s-kv--info"><div class="k8s-k">名称:</div><div class="k8s-v">{{ detailName }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Controller:</div><div class="k8s-v"><span class="k8s-link">{{ controllerText }}</span></div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">Default:</div><div class="k8s-v">{{ defaultText }}</div></div>
        <div class="k8s-kv k8s-kv--muted"><div class="k8s-k">创建时间:</div><div class="k8s-v">{{ createdAtText }}</div></div>
      </div>
    </template>

    <el-tabs v-model="tab" class="k8s-detail-tabs">
      <el-tab-pane label="概览" name="overview">
        <div class="k8s-tab-pane">
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">基础信息</div></template>
            <el-descriptions :column="3" size="small" border class="k8s-detail-desc">
              <el-descriptions-item label="名称">{{ detailName }}</el-descriptions-item>
              <el-descriptions-item label="controller">{{ controllerText }}</el-descriptions-item>
              <el-descriptions-item label="default">{{ defaultText }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ createdAtText }}</el-descriptions-item>
              <el-descriptions-item label="parameters">{{ parametersText }}</el-descriptions-item>
              <el-descriptions-item label="labels">{{ labelsCount }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Parameters</div></template>
            <CodeMirrorViewer :text="parametersViewText" language="json" height="200px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
          <el-card shadow="never" class="k8s-section-card k8s-accent-card">
            <template #header><div class="k8s-section-title">Labels</div></template>
            <CodeMirrorViewer :text="labelsViewText" language="json" height="200px" class="k8s-detail-box k8s-detail-box--fill" />
          </el-card>
        </div>
      </el-tab-pane>
    </el-tabs>
  </WorkloadDetailDrawerShell>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import WorkloadDetailDrawerShell from '@/features/k8s/components/WorkloadDetailDrawerShell.vue'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import { formatTs } from '../../ClusterManageView.utils'

const props = defineProps<{
  clusterId: number
  list: any[]
}>()

const emit = defineEmits<{
  (e: 'refresh-list'): void
}>()

function findRow(name: string): any | null {
  for (const it of props.list ?? []) {
    if (String(it?.metadata?.name ?? '') === name) return it
  }
  return null
}

// ── state ──
const visible = ref(false)
const refreshing = ref(false)
const tab = ref<'overview'>('overview')
const row = ref<any>(null)

const detailName = computed(() => String(row.value?.metadata?.name ?? ''))
const detailTitle = computed(() => (detailName.value ? `IngressClass 详情：${detailName.value}` : 'IngressClass 详情'))
const createdAtText = computed(() => formatTs(row.value?.metadata?.creationTimestamp))
const controllerText = computed(() => String(row.value?.spec?.controller ?? '-'))
const labelsViewText = computed(() => JSON.stringify(row.value?.metadata?.labels ?? {}, null, 2))
const labelsCount = computed(() => Object.keys((row.value?.metadata?.labels ?? {}) as Record<string, string>).length)
const parametersViewText = computed(() => JSON.stringify(row.value?.spec?.parameters ?? {}, null, 2))
const parametersText = computed(() => {
  const p = row.value?.spec?.parameters
  if (!p || typeof p !== 'object') return '-'
  const kind = p?.kind != null ? String(p.kind) : ''
  const name = p?.name != null ? String(p.name) : ''
  if (!kind && !name) return '-'
  return kind && name ? `${kind}/${name}` : kind || name
})
const defaultText = computed(() => {
  const ann = row.value?.metadata?.annotations
  if (!ann || typeof ann !== 'object') return 'no'
  const v = ann['ingressclass.kubernetes.io/is-default-class'] ?? ann['ingressclass.k8s.io/is-default-class'] ?? ann['is-default-class']
  return String(v ?? '').toLowerCase() === 'true' ? 'yes' : 'no'
})

async function refreshDetail() {
  if (!visible.value) return
  try {
    refreshing.value = true
    emit('refresh-list'); await new Promise(r => setTimeout(r, 300))
    const next = findRow(detailName.value)
    if (next) row.value = next
  } finally { refreshing.value = false }
}

watch(() => visible.value, (v) => {
  if (v) return
  tab.value = 'overview'; row.value = null
})

function open(r: any) { row.value = r; tab.value = 'overview'; visible.value = true }
function close() { visible.value = false }
defineExpose({ open, close })
</script>
