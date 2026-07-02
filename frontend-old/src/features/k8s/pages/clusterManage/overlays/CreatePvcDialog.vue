<template>
  <el-dialog :model-value="modelValue" title="创建 PVC" width="560px" destroy-on-close @close="emit('update:modelValue', false)">
    <el-form label-width="92px" @submit.prevent>
      <el-form-item label="命名空间" required>
        <el-select v-model="form.namespace" placeholder="选择命名空间" style="width: 100%" filterable>
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
      </el-form-item>
      <el-form-item label="名称" required>
        <el-input v-model="form.name" placeholder="例如 nginx-data" clearable />
      </el-form-item>
      <el-form-item label="StorageClass">
        <el-select v-model="form.storageClass" placeholder="默认 StorageClass" style="width: 100%" filterable clearable :loading="loadingStorageClasses">
          <el-option label="默认 StorageClass" value="" />
          <el-option v-for="item in storageClassOptions" :key="item" :label="item" :value="item" />
        </el-select>
      </el-form-item>
      <el-form-item label="访问模式" required>
        <el-checkbox-group v-model="form.accessModes">
          <el-checkbox v-for="mode in accessModeOptions" :key="mode" :label="mode">{{ mode }}</el-checkbox>
        </el-checkbox-group>
      </el-form-item>
      <el-form-item label="容量" required>
        <el-input v-model="form.capacity" placeholder="例如 10Gi" clearable />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-space>
        <el-button @click="emit('update:modelValue', false)">取消</el-button>
        <el-button type="primary" :loading="submitting" :disabled="submitDisabled" @click="submit">创建</el-button>
      </el-space>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'

import * as k8sApi from '@/features/k8s/api/k8s'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'

const props = defineProps<{
  modelValue: boolean
  clusterId: number
  namespaces: string[]
  defaultNamespace?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'created'): void
}>()

const accessModeOptions = ['ReadWriteOnce', 'ReadOnlyMany', 'ReadWriteMany', 'ReadWriteOncePod']

const storageClassOptions = ref<string[]>([])
const loadingStorageClasses = ref(false)
const submitting = ref(false)
const form = reactive({
  namespace: '',
  name: '',
  storageClass: '',
  accessModes: ['ReadWriteOnce'],
  capacity: '10Gi'
})

const submitDisabled = computed(() => {
  return !props.clusterId || !form.namespace.trim() || !form.name.trim() || !form.capacity.trim() || form.accessModes.length === 0
})

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return
    resetForm()
    void loadStorageClasses()
  }
)

function resetForm() {
  form.namespace = props.defaultNamespace || props.namespaces[0] || ''
  form.name = ''
  form.storageClass = ''
  form.accessModes = ['ReadWriteOnce']
  form.capacity = '10Gi'
}

async function loadStorageClasses() {
  if (!props.clusterId) return
  loadingStorageClasses.value = true
  try {
    const resp = await k8sApi.listStorageClasses(props.clusterId, {})
    storageClassOptions.value = Array.from(
      new Set((Array.isArray(resp.list) ? resp.list : []).map((item: any) => String(item?.metadata?.name ?? '').trim()).filter(Boolean))
    )
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    loadingStorageClasses.value = false
  }
}

async function submit() {
  if (submitDisabled.value) return
  submitting.value = true
  try {
    await k8sApi.createPVC(props.clusterId, {
      namespace: form.namespace.trim(),
      name: form.name.trim(),
      storage_class: form.storageClass.trim() || undefined,
      access_modes: [...form.accessModes],
      capacity: form.capacity.trim()
    })
    notifySuccess('PVC 已创建')
    emit('update:modelValue', false)
    emit('created')
  } catch (error) {
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    submitting.value = false
  }
}
</script>