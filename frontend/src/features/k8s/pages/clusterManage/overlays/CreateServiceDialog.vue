<template>
  <el-dialog :model-value="modelValue" title="创建 Service" width="600px" destroy-on-close @close="emit('update:modelValue', false)">
    <el-form label-width="92px" @submit.prevent>
      <el-form-item label="命名空间" required>
        <el-select v-model="form.namespace" placeholder="选择命名空间" style="width: 100%" filterable>
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
      </el-form-item>
      <el-form-item label="名称" required>
        <el-input v-model="form.name" placeholder="例如 nginx-svc" clearable />
      </el-form-item>
      <el-form-item label="类型">
        <el-select v-model="form.type" style="width: 100%">
          <el-option label="ClusterIP" value="ClusterIP" />
          <el-option label="NodePort" value="NodePort" />
          <el-option label="LoadBalancer" value="LoadBalancer" />
        </el-select>
      </el-form-item>
      <el-form-item label="选择器">
        <el-input v-model="form.selectorStr" placeholder="app=nginx（key=value，逗号分隔）" />
      </el-form-item>

      <el-divider content-position="left">端口映射</el-divider>
      <div v-for="(port, idx) in form.ports" :key="idx" class="port-row">
        <el-input v-model="port.name" placeholder="名称（可选）" class="port-field" />
        <el-input-number v-model="port.port" placeholder="端口" :min="1" :max="65535" class="port-field" />
        <el-input-number v-model="port.targetPort" placeholder="目标端口" :min="1" :max="65535" class="port-field" />
        <el-select v-model="port.protocol" class="port-field-sm">
          <el-option label="TCP" value="TCP" />
          <el-option label="UDP" value="UDP" />
        </el-select>
        <el-button v-if="form.ports.length > 1" class="remove-btn" size="small" @click="removePort(idx)">
          <el-icon><Close /></el-icon>
        </el-button>
      </div>
      <el-button class="add-row-btn" @click="addPort">
        <el-icon><Plus /></el-icon>
        <span>添加端口</span>
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
import { createService } from '@/features/k8s/api/network'
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

function makePort() {
  return { name: '', port: 80, targetPort: 80, protocol: 'TCP' }
}

const form = reactive({
  namespace: '',
  name: '',
  type: 'ClusterIP',
  selectorStr: '',
  ports: [makePort()],
})

const submitDisabled = computed(() => {
  if (!props.clusterId || !form.namespace.trim() || !form.name.trim()) return true
  return form.ports.some(p => !p.port)
})

watch(() => props.modelValue, (visible) => {
  if (!visible) return
  form.namespace = props.defaultNamespace || props.namespaces[0] || ''
  form.name = ''
  form.type = 'ClusterIP'
  form.selectorStr = ''
  form.ports = [makePort()]
})

function addPort() {
  form.ports.push(makePort())
}

function removePort(idx: number) {
  form.ports.splice(idx, 1)
}

function parseSelector(str: string): Record<string, string> | undefined {
  const s = str.trim()
  if (!s) return undefined
  const result: Record<string, string> = {}
  for (const part of s.split(',')) {
    const [k, v] = part.split('=')
    if (k?.trim()) result[k.trim()] = v?.trim() ?? ''
  }
  return Object.keys(result).length ? result : undefined
}

async function submit() {
  if (submitDisabled.value) return
  submitting.value = true
  try {
    await createService(props.clusterId, {
      namespace: form.namespace.trim(),
      name: form.name.trim(),
      type: form.type,
      selector: parseSelector(form.selectorStr),
      ports: form.ports.map(p => ({
        name: p.name.trim() || undefined,
        port: p.port,
        target_port: p.targetPort || undefined,
        protocol: p.protocol,
      })),
    })
    notifySuccess('Service 已创建')
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
.port-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
  padding: 10px 12px;
  border-radius: 8px;
  background: var(--color-bg-muted, #f8fafc);
  border: 1px solid var(--color-border-subtle, rgba(15, 23, 42, 0.06));
}
.port-field {
  width: 120px;
}
.port-field-sm {
  width: 80px;
}
.remove-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  border-radius: 6px;
  border: 1px solid var(--color-border-default, rgba(15, 23, 42, 0.08));
  background: rgba(239, 68, 68, 0.06);
  color: #ef4444;
  flex-shrink: 0;
  transition: all 0.2s ease;
}
.remove-btn:hover {
  background: rgba(239, 68, 68, 0.12);
  border-color: rgba(239, 68, 68, 0.3);
}
.add-row-btn {
  width: 100%;
  height: 38px;
  border-radius: 10px;
  border: 1px dashed var(--color-border-default, rgba(15, 23, 42, 0.12));
  background: transparent;
  color: var(--color-accent-primary, #3b82f6);
  font-weight: 600;
  font-size: 13px;
  transition: all 0.2s ease;
}
.add-row-btn:hover {
  border-color: rgba(59, 130, 246, 0.4);
  background: rgba(59, 130, 246, 0.04);
}
</style>
