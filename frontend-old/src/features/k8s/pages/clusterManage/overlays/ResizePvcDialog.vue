<template>
  <el-dialog :model-value="modelValue" title="扩容 PVC" width="560px" destroy-on-close @close="emit('update:modelValue', false)">
    <el-form label-width="108px" @submit.prevent>
      <el-form-item label="命名空间">
        <div class="readonly-value">{{ namespaceText }}</div>
      </el-form-item>
      <el-form-item label="名称">
        <div class="readonly-value">{{ nameText }}</div>
      </el-form-item>
      <el-form-item label="StorageClass">
        <div class="readonly-value">{{ storageClassText }}</div>
      </el-form-item>
      <el-form-item label="当前请求量">
        <div class="readonly-value">{{ currentRequestText }}</div>
      </el-form-item>
      <el-form-item label="当前容量">
        <div class="readonly-value">{{ currentCapacityText }}</div>
      </el-form-item>
      <el-form-item label="新容量" required>
        <el-input v-model="newSize" placeholder="例如 20Gi" clearable />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-space>
        <el-button @click="emit('update:modelValue', false)">取消</el-button>
        <el-button type="primary" :loading="submitting" :disabled="submitDisabled" @click="submit">保存</el-button>
      </el-space>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { parse, stringify } from 'yaml'

import * as k8sApi from '@/features/k8s/api/k8s'
import { applyManifest } from '@/features/k8s/api/manifest'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import { getRowNamespace } from '../../ClusterManageView.utils'

const props = defineProps<{
  modelValue: boolean
  clusterId: number
  row: any | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved'): void
}>()

const submitting = ref(false)
const newSize = ref('')

const namespaceText = computed(() => getRowNamespace(props.row) || '-')
const nameText = computed(() => String(props.row?.metadata?.name ?? '-'))
const storageClassText = computed(() => String(props.row?.spec?.storageClassName ?? '-'))
const currentRequestText = computed(() => String(props.row?.spec?.resources?.requests?.storage ?? '-'))
const currentCapacityText = computed(() => String(props.row?.status?.capacity?.storage ?? '-'))
const submitDisabled = computed(() => {
  return !props.clusterId || !namespaceText.value || namespaceText.value === '-' || !nameText.value || nameText.value === '-' || !newSize.value.trim()
})

watch(
  () => [props.modelValue, props.row] as const,
  ([visible, row]) => {
    if (!visible) return
    newSize.value = String(row?.spec?.resources?.requests?.storage ?? '').trim()
  },
  { immediate: true },
)

async function submit() {
  if (submitDisabled.value) return
  const namespace = namespaceText.value
  const name = nameText.value
  if (!namespace || namespace === '-' || !name || name === '-') return
  submitting.value = true
  try {
    const current = await k8sApi.getPVCYaml(props.clusterId, namespace, name)
    const manifest = parse(current.text) as Record<string, any>
    if (!manifest || typeof manifest !== 'object' || Array.isArray(manifest)) {
      throw new Error('PVC YAML 解析失败')
    }
    manifest.spec = manifest.spec && typeof manifest.spec === 'object' ? manifest.spec : {}
    manifest.spec.resources = manifest.spec.resources && typeof manifest.spec.resources === 'object' ? manifest.spec.resources : {}
    manifest.spec.resources.requests = manifest.spec.resources.requests && typeof manifest.spec.resources.requests === 'object'
      ? manifest.spec.resources.requests
      : {}
    manifest.spec.resources.requests.storage = newSize.value.trim()

    await applyManifest(props.clusterId, {
      yaml: stringify(manifest),
      default_namespace: namespace,
      source_label: 'PVC 扩容',
      source_resource: 'pvcs',
    })
    notifySuccess('PVC 扩容配置已提交')
    emit('update:modelValue', false)
    emit('saved')
  } catch (error) {
    const err = error as ApiError
    const fallback = error instanceof Error ? error.message : 'PVC 扩容失败'
    const message = String(err?.message ?? '').trim() || fallback
    notifyError(err?.requestId ? `${message} (request_id=${err.requestId})` : message)
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.readonly-value {
  min-height: 32px;
  display: flex;
  align-items: center;
  color: var(--el-text-color-regular);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, Courier New, monospace;
}
</style>