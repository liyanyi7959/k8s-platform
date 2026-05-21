<template>
  <el-dialog :model-value="modelValue" title="创建 Deployment" width="640px" destroy-on-close @close="emit('update:modelValue', false)">
    <el-form label-width="92px" @submit.prevent>
      <el-form-item label="命名空间" required>
        <el-select v-model="form.namespace" placeholder="选择命名空间" style="width: 100%" filterable>
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
      </el-form-item>
      <el-form-item label="名称" required>
        <el-input v-model="form.name" placeholder="例如 nginx-deploy" clearable />
      </el-form-item>
      <el-form-item label="副本数">
        <el-input-number v-model="form.replicas" :min="1" :max="100" />
      </el-form-item>

      <el-divider content-position="left">容器配置</el-divider>
      <div v-for="(container, idx) in form.containers" :key="idx" class="container-block">
        <div class="container-header">
          <span class="container-label">容器 {{ idx + 1 }}</span>
          <el-button v-if="form.containers.length > 1" class="remove-btn" size="small" @click="removeContainer(idx)">
            <el-icon><Close /></el-icon>
            <span>移除</span>
          </el-button>
        </div>
        <el-form-item label="容器名" required>
          <el-input v-model="container.name" placeholder="例如 nginx" />
        </el-form-item>
        <el-form-item label="镜像" required>
          <el-input v-model="container.image" placeholder="例如 nginx:1.25" />
        </el-form-item>
        <el-form-item label="CPU">
          <el-input v-model="container.cpu" placeholder="例如 100m（可选）" />
        </el-form-item>
        <el-form-item label="内存">
          <el-input v-model="container.memory" placeholder="例如 128Mi（可选）" />
        </el-form-item>
      </div>
      <el-button class="add-container-btn" @click="addContainer">
        <el-icon><Plus /></el-icon>
        <span>添加容器</span>
      </el-button>
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
import { Plus, Close } from '@element-plus/icons-vue'
import { createDeployment } from '@/features/k8s/api/workload'
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

const submitting = ref(false)

function makeContainer() {
  return { name: '', image: '', cpu: '', memory: '' }
}

const form = reactive({
  namespace: '',
  name: '',
  replicas: 1,
  containers: [makeContainer()],
})

const submitDisabled = computed(() => {
  if (!props.clusterId || !form.namespace.trim() || !form.name.trim()) return true
  return form.containers.some(c => !c.name.trim() || !c.image.trim())
})

watch(() => props.modelValue, (visible) => {
  if (!visible) return
  form.namespace = props.defaultNamespace || props.namespaces[0] || ''
  form.name = ''
  form.replicas = 1
  form.containers = [makeContainer()]
})

function addContainer() {
  form.containers.push(makeContainer())
}

function removeContainer(idx: number) {
  form.containers.splice(idx, 1)
}

async function submit() {
  if (submitDisabled.value) return
  submitting.value = true
  try {
    await createDeployment(props.clusterId, {
      namespace: form.namespace.trim(),
      name: form.name.trim(),
      replicas: form.replicas,
      containers: form.containers.map(c => ({
        name: c.name.trim(),
        image: c.image.trim(),
        cpu: c.cpu.trim() || undefined,
        memory: c.memory.trim() || undefined,
      })),
    })
    notifySuccess('Deployment 已创建')
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

<style scoped>
.container-block {
  border: 1px solid var(--color-border-default, rgba(15, 23, 42, 0.08));
  border-radius: 10px;
  padding: 16px;
  margin-bottom: 12px;
  background: var(--color-bg-muted, #f8fafc);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}
.container-block:hover {
  border-color: rgba(59, 130, 246, 0.18);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.06);
}
.container-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--color-border-subtle, rgba(15, 23, 42, 0.06));
}
.container-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary, #4b5563);
}
.remove-btn {
  height: 28px;
  padding: 0 10px;
  border-radius: 6px;
  border: 1px solid var(--color-border-default, rgba(15, 23, 42, 0.08));
  background: rgba(239, 68, 68, 0.06);
  color: #ef4444;
  font-size: 12px;
  font-weight: 500;
  transition: all 0.2s ease;
}
.remove-btn:hover {
  background: rgba(239, 68, 68, 0.12);
  border-color: rgba(239, 68, 68, 0.3);
}
.add-container-btn {
  width: 100%;
  height: 40px;
  border-radius: 10px;
  border: 1px dashed var(--color-border-default, rgba(15, 23, 42, 0.12));
  background: transparent;
  color: var(--color-accent-primary, #3b82f6);
  font-weight: 600;
  font-size: 13px;
  transition: all 0.2s ease;
}
.add-container-btn:hover {
  border-color: rgba(59, 130, 246, 0.4);
  background: rgba(59, 130, 246, 0.04);
}
</style>
