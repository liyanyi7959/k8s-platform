<template>
  <el-dialog :model-value="modelValue" title="创建 Ingress" width="680px" destroy-on-close @close="emit('update:modelValue', false)">
    <el-form label-width="92px" @submit.prevent>
      <el-form-item label="命名空间" required>
        <el-select v-model="form.namespace" placeholder="选择命名空间" style="width: 100%" filterable>
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
      </el-form-item>
      <el-form-item label="名称" required>
        <el-input v-model="form.name" placeholder="例如 my-ingress" clearable />
      </el-form-item>
      <el-form-item label="IngressClass">
        <el-input v-model="form.ingressClass" placeholder="例如 nginx（可选）" clearable />
      </el-form-item>
      <el-form-item label="TLS Secret">
        <el-input v-model="form.tlsSecretName" placeholder="例如 tls-cert（可选）" clearable />
      </el-form-item>

      <el-divider content-position="left">路由规则</el-divider>
      <div v-for="(rule, rIdx) in form.rules" :key="rIdx" class="rule-block">
        <div class="rule-header">
          <el-input v-model="rule.host" placeholder="主机名 例如 example.com" class="rule-host" />
          <el-button v-if="form.rules.length > 1" class="remove-btn" size="small" @click="removeRule(rIdx)">
            <el-icon><Close /></el-icon>
            <span>移除</span>
          </el-button>
        </div>
        <div v-for="(path, pIdx) in rule.paths" :key="pIdx" class="path-row">
          <el-input v-model="path.path" placeholder="路径 /" class="path-field" />
          <el-select v-model="path.pathType" class="path-field-sm">
            <el-option label="Prefix" value="Prefix" />
            <el-option label="Exact" value="Exact" />
          </el-select>
          <el-input v-model="path.serviceName" placeholder="后端 Service" class="path-field" />
          <el-input-number v-model="path.servicePort" :min="1" :max="65535" placeholder="端口" class="path-field-sm" />
          <el-button v-if="rule.paths.length > 1" class="remove-icon-btn" size="small" @click="removePath(rIdx, pIdx)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
        <el-button class="add-path-btn" size="small" @click="addPath(rIdx)">
          <el-icon><Plus /></el-icon>
          <span>添加路径</span>
        </el-button>
      </div>
      <el-button class="add-rule-btn" @click="addRule">
        <el-icon><Plus /></el-icon>
        <span>添加规则</span>
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
import { createIngress } from '@/features/k8s/api/network'
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

function makePath() {
  return { path: '/', pathType: 'Prefix', serviceName: '', servicePort: 80 }
}

function makeRule() {
  return { host: '', paths: [makePath()] }
}

const form = reactive({
  namespace: '',
  name: '',
  ingressClass: '',
  tlsSecretName: '',
  rules: [makeRule()],
})

const submitDisabled = computed(() => {
  if (!props.clusterId || !form.namespace.trim() || !form.name.trim()) return true
  return form.rules.some(r => r.paths.some(p => !p.serviceName.trim() || !p.servicePort))
})

watch(() => props.modelValue, (visible) => {
  if (!visible) return
  form.namespace = props.defaultNamespace || props.namespaces[0] || ''
  form.name = ''
  form.ingressClass = ''
  form.tlsSecretName = ''
  form.rules = [makeRule()]
})

function addRule() {
  form.rules.push(makeRule())
}

function removeRule(idx: number) {
  form.rules.splice(idx, 1)
}

function addPath(ruleIdx: number) {
  form.rules[ruleIdx].paths.push(makePath())
}

function removePath(ruleIdx: number, pathIdx: number) {
  form.rules[ruleIdx].paths.splice(pathIdx, 1)
}

async function submit() {
  if (submitDisabled.value) return
  submitting.value = true
  try {
    await createIngress(props.clusterId, {
      namespace: form.namespace.trim(),
      name: form.name.trim(),
      ingress_class: form.ingressClass.trim() || undefined,
      tls_secret_name: form.tlsSecretName.trim() || undefined,
      rules: form.rules.map(r => ({
        host: r.host.trim(),
        paths: r.paths.map(p => ({
          path: p.path.trim() || '/',
          path_type: p.pathType,
          service_name: p.serviceName.trim(),
          service_port: p.servicePort,
        })),
      })),
    })
    notifySuccess('Ingress 已创建')
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
.rule-block {
  border: 1px solid var(--color-border-default, rgba(15, 23, 42, 0.08));
  border-radius: 10px;
  padding: 14px;
  margin-bottom: 12px;
  background: var(--color-bg-muted, #f8fafc);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}
.rule-block:hover {
  border-color: rgba(59, 130, 246, 0.18);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.06);
}
.rule-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--color-border-subtle, rgba(15, 23, 42, 0.06));
}
.rule-host {
  width: 260px;
}
.path-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--color-bg-card, #ffffff);
  border: 1px solid var(--color-border-subtle, rgba(15, 23, 42, 0.06));
}
.path-field {
  width: 130px;
}
.path-field-sm {
  width: 100px;
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
.remove-icon-btn {
  width: 26px;
  height: 26px;
  padding: 0;
  border-radius: 6px;
  border: 1px solid var(--color-border-default, rgba(15, 23, 42, 0.08));
  background: rgba(239, 68, 68, 0.06);
  color: #ef4444;
  flex-shrink: 0;
  transition: all 0.2s ease;
}
.remove-icon-btn:hover {
  background: rgba(239, 68, 68, 0.12);
  border-color: rgba(239, 68, 68, 0.3);
}
.add-path-btn {
  height: 30px;
  padding: 0 12px;
  border-radius: 8px;
  border: 1px dashed var(--color-border-default, rgba(15, 23, 42, 0.12));
  background: transparent;
  color: var(--color-accent-primary, #3b82f6);
  font-weight: 500;
  font-size: 12px;
  transition: all 0.2s ease;
}
.add-path-btn:hover {
  border-color: rgba(59, 130, 246, 0.4);
  background: rgba(59, 130, 246, 0.04);
}
.add-rule-btn {
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
.add-rule-btn:hover {
  border-color: rgba(59, 130, 246, 0.4);
  background: rgba(59, 130, 246, 0.04);
}
</style>
