<template>
  <el-dialog :model-value="modelValue" title="编辑 PV" width="640px" destroy-on-close @close="emit('update:modelValue', false)">
    <el-form label-width="116px" @submit.prevent>
      <el-form-item label="名称">
        <div class="readonly-value">{{ nameText }}</div>
      </el-form-item>
      <el-form-item label="StorageClass">
        <div class="readonly-value">{{ storageClassText }}</div>
      </el-form-item>
      <el-form-item label="Claim">
        <div class="readonly-value">{{ claimText }}</div>
      </el-form-item>
      <el-form-item label="ReclaimPolicy" required>
        <el-select v-model="reclaimPolicy" style="width: 100%">
          <el-option label="Delete" value="Delete" />
          <el-option label="Retain" value="Retain" />
          <el-option label="Recycle" value="Recycle" />
        </el-select>
      </el-form-item>
      <el-form-item label="MountOptions">
        <el-input v-model="mountOptionsText" placeholder="可选，逗号分隔，例如 discard,noatime" clearable />
      </el-form-item>
      <el-form-item label="Labels JSON">
        <el-input
          v-model="labelsText"
          type="textarea"
          :autosize="{ minRows: 6, maxRows: 12 }"
          :placeholder="labelsPlaceholder"
        />
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
const reclaimPolicy = ref('Delete')
const mountOptionsText = ref('')
const labelsText = ref('{}')
const labelsPlaceholder = `例如 {
  "env": "prod"
}`

const nameText = computed(() => String(props.row?.metadata?.name ?? '-'))
const storageClassText = computed(() => String(props.row?.spec?.storageClassName ?? '-'))
const claimText = computed(() => {
  const namespace = String(props.row?.spec?.claimRef?.namespace ?? '').trim()
  const name = String(props.row?.spec?.claimRef?.name ?? '').trim()
  if (!namespace && !name) return '-'
  return namespace && name ? `${namespace}/${name}` : name || namespace
})
const submitDisabled = computed(() => !props.clusterId || !nameText.value || nameText.value === '-' || !reclaimPolicy.value.trim())

watch(
  () => [props.modelValue, props.row] as const,
  ([visible, row]) => {
    if (!visible) return
    reclaimPolicy.value = String(row?.spec?.persistentVolumeReclaimPolicy ?? 'Delete')
    mountOptionsText.value = Array.isArray(row?.spec?.mountOptions)
      ? row.spec.mountOptions.map((item: unknown) => String(item ?? '').trim()).filter(Boolean).join(', ')
      : ''
    labelsText.value = JSON.stringify(row?.metadata?.labels ?? {}, null, 2)
  },
  { immediate: true },
)

function parseLabels(): Record<string, string> {
  const raw = String(labelsText.value ?? '').trim()
  if (!raw) return {}
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    throw new Error('Labels JSON 不是合法对象')
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('Labels JSON 必须是对象')
  }
  return Object.fromEntries(
    Object.entries(parsed as Record<string, unknown>)
      .map(([key, value]) => [String(key).trim(), value == null ? '' : String(value)])
      .filter(([key]) => Boolean(key))
  )
}

async function submit() {
  if (submitDisabled.value) return
  const name = nameText.value
  if (!name || name === '-') return
  submitting.value = true
  try {
    const current = await k8sApi.getPVYaml(props.clusterId, name)
    const manifest = parse(current.text) as Record<string, any>
    if (!manifest || typeof manifest !== 'object' || Array.isArray(manifest)) {
      throw new Error('PV YAML 解析失败')
    }
    manifest.spec = manifest.spec && typeof manifest.spec === 'object' ? manifest.spec : {}
    manifest.metadata = manifest.metadata && typeof manifest.metadata === 'object' ? manifest.metadata : {}
    manifest.spec.persistentVolumeReclaimPolicy = reclaimPolicy.value.trim()
    const mountOptions = mountOptionsText.value
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
    if (mountOptions.length) manifest.spec.mountOptions = mountOptions
    else delete manifest.spec.mountOptions
    manifest.metadata.labels = parseLabels()

    await applyManifest(props.clusterId, {
      yaml: stringify(manifest),
      source_label: 'PV 编辑',
      source_resource: 'pvs',
    })
    notifySuccess('PV 编辑已保存')
    emit('update:modelValue', false)
    emit('saved')
  } catch (error) {
    const err = error as ApiError
    const fallback = error instanceof Error ? error.message : 'PV 编辑失败'
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