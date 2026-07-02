<template>
  <el-dialog v-model="visible" title="伸缩副本数" class="dialog-sm" @open="onOpen">
    <el-form label-width="120px">
      <el-form-item label="资源">
        <div class="meta">{{ meta }}</div>
      </el-form-item>
      <el-form-item label="当前状态">
        <div class="scale-current">
          <el-tag :type="currentReady >= currentDesired && currentDesired > 0 ? 'success' : currentDesired > 0 ? 'warning' : 'info'" size="small" effect="plain">
            Ready {{ currentReady }}/{{ currentDesired }}
          </el-tag>
        </div>
      </el-form-item>
      <el-form-item label="目标副本数">
        <div class="scale-target">
          <el-slider v-model="replicas" :min="0" :max="sliderMax" :step="1" show-input class="scale-target__slider" />
          <div class="scale-target__hint">拖动滑块或直接输入，当前可调范围 0 - {{ sliderMax }}</div>
        </div>
        <div v-if="replicas === 0 && currentDesired > 0" class="scale-warn">
          <el-icon color="var(--el-color-warning)"><WarningFilled /></el-icon>
          <span class="scale-warn-text">缩容到 0 将终止所有 Pod，服务不可用</span>
        </div>
      </el-form-item>
      <el-form-item v-if="hasHPA" label="">
        <el-alert type="warning" :closable="false" show-icon>
          <template #title>
            <span>该资源已绑定 HPA（{{ hpaName }}），手动伸缩可能被 HPA 自动覆盖</span>
          </template>
        </el-alert>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-space>
        <el-tooltip content="取消" placement="top">
          <el-button :icon="CircleClose" circle @click="close" />
        </el-tooltip>
        <el-tooltip content="确认" placement="top">
          <el-button type="primary" :disabled="!clusterId || !target" :loading="scaling" :icon="Check" circle @click="onSubmit" />
        </el-tooltip>
      </el-space>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Check, CircleClose, WarningFilled } from '@element-plus/icons-vue'
import { ElMessageBox } from 'element-plus'
import { useK8sScaleDialog, type K8sScaleTarget } from '@/features/k8s/composables/useK8sScaleDialog'
import * as k8sApi from '@/features/k8s/api/k8s'

const props = defineProps<{
  clusterId?: number
  clusterName?: string
}>()

const emit = defineEmits<{
  (e: 'scaled'): void
}>()

const clusterId = computed(() => props.clusterId)
const clusterName = computed(() => String(props.clusterName ?? ''))

const { visible, scaling, target, replicas, meta, open, close, submit } = useK8sScaleDialog({
  clusterId,
  clusterName,
  onScaled: async () => emit('scaled')
})

// ── 当前副本状态 ──
const currentDesired = ref(0)
const currentReady = ref(0)
const sliderMax = computed(() => Math.min(500, Math.max(10, currentDesired.value * 2, currentReady.value * 2, replicas.value)))

// ── HPA 检测 ──
const hasHPA = ref(false)
const hpaName = ref('')

function openWithState(nextTarget: K8sScaleTarget, nextReplicas: number, ready?: number) {
  currentDesired.value = Math.max(0, Number(nextReplicas) || 0)
  currentReady.value = Math.max(0, Number(ready) || 0)
  hasHPA.value = false
  hpaName.value = ''
  open(nextTarget, nextReplicas)
}

async function onOpen() {
  // 检测 HPA 关联
  if (!clusterId.value || !target.value) return
  try {
    const data = await k8sApi.listHPAs(clusterId.value, { namespace: target.value.namespace })
    const list: any[] = Array.isArray(data.list) ? data.list : []
    const matched = list.find((h) => {
      const ref = h?.spec?.scaleTargetRef
      if (!ref) return false
      return String(ref.kind ?? '') === target.value!.kind && String(ref.name ?? '') === target.value!.name
    })
    if (matched) {
      hasHPA.value = true
      hpaName.value = String(matched?.metadata?.name ?? '')
    }
  } catch {
    // 忽略 HPA 检测失败
  }
}

async function onSubmit() {
  // 缩容到 0 的二次确认
  if (replicas.value === 0 && currentDesired.value > 0) {
    try {
      await ElMessageBox.confirm(
        '确认将副本数缩容到 0？所有 Pod 将被终止，服务将不可用。',
        '⚠️ 缩容确认',
        { type: 'warning', confirmButtonText: '确认缩容到 0', cancelButtonText: '取消' }
      )
    } catch {
      return
    }
  }
  await submit()
}

defineExpose<{ open: (target: K8sScaleTarget, replicas: number, ready?: number) => void }>({ open: openWithState })
</script>

<style scoped>
.scale-current {
  display: flex;
  align-items: center;
  gap: 8px;
}

.scale-target {
  width: min(100%, 440px);
}

.scale-target__slider {
  width: 100%;
}

.scale-target__hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.scale-warn {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
  font-size: 12px;
}
.scale-warn-text {
  color: var(--el-color-warning);
}
</style>
