<template>
  <el-drawer v-model="visible" class="rollout-drawer" size="920px" :append-to-body="true" @closed="handleClosed">
    <template #header>
      <div class="rollout-drawer__header">
        <div>
          <div class="rollout-drawer__title">版本历史</div>
          <div class="rollout-drawer__subtitle">{{ targetLabel }}</div>
        </div>
        <el-tag size="small" type="info">{{ target.kind || '-' }}</el-tag>
      </div>
    </template>

    <div class="rollout-drawer__body">
      <section class="rollout-card">
        <div class="rollout-card__head">
          <div>
            <div class="rollout-card__title">Rollout 进度</div>
            <div class="rollout-card__desc">{{ progressDescription }}</div>
          </div>
          <el-tag :type="rolloutDone ? 'success' : 'warning'">{{ rolloutDone ? '已完成' : '进行中' }}</el-tag>
        </div>

        <el-progress :percentage="progressPercent" :status="rolloutDone ? 'success' : undefined" :stroke-width="10" />

        <div class="rollout-card__meta">
          <span>READY {{ readyText }}</span>
          <span>UPDATED {{ updatedText }}</span>
          <span v-if="currentRevisionText">当前版本 {{ currentRevisionText }}</span>
        </div>
      </section>

      <section class="rollout-card rollout-card--history">
        <div class="rollout-card__head">
          <div>
            <div class="rollout-card__title">历史版本</div>
            <div class="rollout-card__desc">按 revision 倒序展示，非当前版本可直接回滚</div>
          </div>
          <el-button size="small" :icon="RefreshRight" :loading="loadingHistory || loadingProgress" @click="reloadAll">刷新</el-button>
        </div>

        <el-alert v-if="loadError" :title="loadError" type="error" show-icon :closable="false" class="rollout-card__alert" />

        <el-skeleton v-else-if="loadingHistory && history.length === 0" animated :rows="6" />

        <el-empty v-else-if="history.length === 0" description="暂无版本历史" />

        <el-table v-else :data="history" size="small" border class="rollout-table">
          <el-table-column label="版本" min-width="96">
            <template #default="scope">
              <span class="revision-chip">r{{ scope.row.revision }}</span>
            </template>
          </el-table-column>

          <el-table-column label="镜像" min-width="240">
            <template #default="scope">
              <div class="image-list">
                <el-tag v-for="image in scope.row.images" :key="image" size="small" effect="plain" class="image-list__tag">{{ image }}</el-tag>
                <span v-if="scope.row.images.length === 0" class="muted-text">-</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="变更原因" min-width="220">
            <template #default="scope">
              <span class="cause-text">{{ scope.row.change_cause || '-' }}</span>
            </template>
          </el-table-column>

          <el-table-column label="时间" min-width="170">
            <template #default="scope">
              <span>{{ formatTs(scope.row.created_at) }}</span>
            </template>
          </el-table-column>

          <el-table-column label="当前" width="90" align="center">
            <template #default="scope">
              <el-tag v-if="scope.row.is_current" type="success" size="small">当前</el-tag>
              <span v-else class="muted-text">-</span>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="84" align="center" fixed="right">
            <template #default="scope">
              <ActionIconButton
                v-if="!scope.row.is_current"
                :icon="RefreshLeft"
                tooltip="回滚到此版本"
                variant="warn"
                :loading="rollingRevision === scope.row.revision"
                @click="confirmRollback(scope.row)"
              />
              <span v-else class="muted-text">-</span>
            </template>
          </el-table-column>
        </el-table>
      </section>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { RefreshLeft, RefreshRight } from '@element-plus/icons-vue'
import { computed, onBeforeUnmount, ref } from 'vue'

import ActionIconButton from '@/shared/components/ActionIconButton.vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import type { RolloutRevision } from '@/features/k8s/api/workload'
import {
  formatTs,
  getWorkloadAvailable,
  getWorkloadCurrentReplicas,
  getWorkloadDesired,
  getWorkloadProgressText,
  isWorkloadProgressing
} from '@/features/k8s/pages/ClusterManageView.utils'

type RolloutTarget = {
  kind: string
  namespace: string
  name: string
}

const props = defineProps<{
  clusterId: number
}>()

const emit = defineEmits<{
  (e: 'rolled-back'): void
}>()

const visible = ref(false)
const loadingHistory = ref(false)
const loadingProgress = ref(false)
const rollingRevision = ref<number | null>(null)
const loadError = ref('')
const history = ref<RolloutRevision[]>([])
const currentRow = ref<any | null>(null)
const target = ref<RolloutTarget>({ kind: '', namespace: '', name: '' })

let pollTimer: number | null = null

const targetLabel = computed(() => {
  if (!target.value.namespace && !target.value.name) return '-'
  return `${target.value.namespace || '-'}/${target.value.name || '-'}`
})

const desiredReplicas = computed(() => getWorkloadDesired(currentRow.value))
const availableReplicas = computed(() => getWorkloadAvailable(currentRow.value))
const currentReplicas = computed(() => getWorkloadCurrentReplicas(currentRow.value))
const updatedReplicas = computed(() => Number(currentRow.value?.status?.updatedReplicas ?? availableReplicas.value))
const progressPercent = computed(() => {
  if (!currentRow.value) return 0
  if (!isWorkloadProgressing(currentRow.value)) return 100
  if (desiredReplicas.value <= 0) return 99
  const updatedRatio = Math.max(0, Math.min(1, updatedReplicas.value / desiredReplicas.value))
  const availableRatio = Math.max(0, Math.min(1, availableReplicas.value / desiredReplicas.value))
  const raw = Math.round(((updatedRatio + availableRatio) / 2) * 100)
  return Math.max(5, Math.min(99, raw))
})
const rolloutDone = computed(() => {
  if (!currentRow.value) return false
  if (desiredReplicas.value <= 0) return true
  return !isWorkloadProgressing(currentRow.value) && updatedReplicas.value >= desiredReplicas.value && availableReplicas.value >= desiredReplicas.value
})
const readyText = computed(() => `${availableReplicas.value}/${desiredReplicas.value}`)
const updatedText = computed(() => `${updatedReplicas.value}/${desiredReplicas.value}`)
const currentRevisionText = computed(() => {
  const currentRevision = history.value.find((item) => item.is_current)?.revision
  return currentRevision ? `r${currentRevision}` : ''
})
const progressDescription = computed(() => {
  if (!target.value.kind) return '等待选择工作负载'
  if (!currentRow.value) return '正在同步当前工作负载状态'
  const progressText = getWorkloadProgressText(currentRow.value)
  const currentPodsText = currentReplicas.value > desiredReplicas.value && desiredReplicas.value > 0
    ? `，当前 Pod ${currentReplicas.value}/${desiredReplicas.value}`
    : ''
  if (progressText) return `${progressText}，已更新 ${updatedText.value}，可用 ${readyText.value}${currentPodsText}`
  return `已更新 ${updatedText.value}，可用 ${readyText.value}`
})

async function open(payload: RolloutTarget) {
  target.value = {
    kind: String(payload.kind ?? '').trim(),
    namespace: String(payload.namespace ?? '').trim(),
    name: String(payload.name ?? '').trim()
  }
  visible.value = true
  await reloadAll()
  startPolling()
}

async function reloadAll() {
  if (!target.value.kind || !target.value.namespace || !target.value.name) return
  loadError.value = ''
  await Promise.all([loadHistory(), loadProgress()])
}

async function loadHistory() {
  loadingHistory.value = true
  try {
    const data = await k8sApi.getRolloutHistory(props.clusterId, target.value.namespace, target.value.name, target.value.kind)
    history.value = Array.isArray(data.history) ? data.history : []
  } catch (error) {
    history.value = []
    loadError.value = resolveErrorMessage(error, '加载版本历史失败')
  } finally {
    loadingHistory.value = false
  }
}

async function loadProgress() {
  loadingProgress.value = true
  try {
    const data = await k8sApi.listWorkloads(props.clusterId, { kind: target.value.kind, namespace: target.value.namespace })
    const list = Array.isArray(data.list) ? data.list : []
    currentRow.value = list.find((item: any) => String(item?.metadata?.name ?? '').trim() === target.value.name) ?? null
  } catch (error) {
    currentRow.value = null
    if (!loadError.value) {
      loadError.value = resolveErrorMessage(error, '加载当前 Rollout 状态失败')
    }
  } finally {
    loadingProgress.value = false
  }
}

async function confirmRollback(row: RolloutRevision) {
  if (!target.value.kind || !target.value.namespace || !target.value.name) return
  try {
    await ElMessageBox.confirm(
      `将 ${target.value.namespace}/${target.value.name} 回滚到 r${row.revision}。该操作会触发新的 Deployment 滚动更新，是否继续？`,
      '确认回滚',
      {
        type: 'warning',
        confirmButtonText: '确认回滚',
        cancelButtonText: '取消'
      }
    )
  } catch {
    return
  }

  rollingRevision.value = row.revision
  try {
    await k8sApi.rolloutUndo(props.clusterId, target.value.namespace, target.value.name, {
      kind: target.value.kind,
      revision: row.revision
    })
    ElMessage.success(`已提交回滚到 r${row.revision}`)
    visible.value = false
    emit('rolled-back')
  } catch (error) {
    ElMessage.error(resolveErrorMessage(error, '提交回滚失败'))
  } finally {
    rollingRevision.value = null
  }
}

function startPolling() {
  stopPolling()
  pollTimer = window.setInterval(() => {
    if (!visible.value) return
    void loadProgress()
  }, 3000)
}

function stopPolling() {
  if (pollTimer != null) {
    window.clearInterval(pollTimer)
    pollTimer = null
  }
}

function handleClosed() {
  stopPolling()
  history.value = []
  currentRow.value = null
  loadError.value = ''
  rollingRevision.value = null
  target.value = { kind: '', namespace: '', name: '' }
}

function resolveErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error) {
    const message = String(error.message || '').trim()
    return message || fallback
  }
  const message = String(error ?? '').trim()
  return message || fallback
}

onBeforeUnmount(() => {
  stopPolling()
})

defineExpose({ open })
</script>

<style scoped>
.rollout-drawer__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.rollout-drawer__title {
  font-size: 18px;
  font-weight: 700;
  color: #152033;
}

.rollout-drawer__subtitle {
  margin-top: 4px;
  font-size: 13px;
  color: #667085;
}

.rollout-drawer__body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.rollout-card {
  border: 1px solid #e5e7eb;
  border-radius: 16px;
  padding: 16px;
  background: linear-gradient(180deg, #fbfdff 0%, #f8fafc 100%);
}

.rollout-card--history {
  background: #fff;
}

.rollout-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.rollout-card__title {
  font-size: 15px;
  font-weight: 700;
  color: #111827;
}

.rollout-card__desc {
  margin-top: 4px;
  font-size: 12px;
  color: #667085;
}

.rollout-card__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 12px;
  font-size: 12px;
  color: #475467;
}

.rollout-card__alert {
  margin-bottom: 12px;
}

.rollout-table :deep(.el-table__cell) {
  vertical-align: top;
}

.revision-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 48px;
  padding: 4px 10px;
  border-radius: 999px;
  background: #e0f2fe;
  color: #075985;
  font-weight: 700;
}

.image-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.image-list__tag {
  max-width: 100%;
}

.cause-text {
  color: #344054;
  line-height: 1.5;
}

.muted-text {
  color: #98a2b3;
}

@media (max-width: 960px) {
  .rollout-card__head {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>