<template>
  <el-dialog :model-value="modelValue" :title="dialogTitle" width="760px" destroy-on-close @close="emit('update:modelValue', false)">
    <div class="create-dialog-note">创建模式会校验同名资源冲突，不会覆盖已有对象。</div>

    <el-form label-width="118px" @submit.prevent>
      <el-form-item label="命名空间" required>
        <el-select v-model="form.namespace" placeholder="选择命名空间" style="width: 100%" filterable>
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
      </el-form-item>
      <el-form-item label="名称" required>
        <el-input v-model="form.name" :placeholder="namePlaceholder" clearable />
      </el-form-item>

      <template v-if="resource === 'configmaps'">
        <el-form-item label="不可变">
          <el-switch v-model="form.immutable" />
        </el-form-item>
        <el-divider content-position="left">数据项</el-divider>
        <div v-for="(entry, idx) in form.configEntries" :key="`cm-${idx}`" class="entry-row entry-row--stacked">
          <el-input v-model="entry.key" placeholder="键，例如 application.yaml" class="entry-row__key" />
          <el-input v-model="entry.value" type="textarea" :autosize="{ minRows: 2, maxRows: 6 }" placeholder="值" class="entry-row__value" />
          <el-button v-if="form.configEntries.length > 1" class="remove-btn" size="small" @click="removeConfigEntry(idx)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
        <el-button class="add-row-btn" @click="addConfigEntry">
          <el-icon><Plus /></el-icon>
          <span>添加数据项</span>
        </el-button>
      </template>

      <template v-else-if="resource === 'secrets'">
        <el-form-item label="类型">
          <el-select v-model="form.secretType" style="width: 100%">
            <el-option label="Opaque" value="Opaque" />
            <el-option label="kubernetes.io/basic-auth" value="kubernetes.io/basic-auth" />
            <el-option label="kubernetes.io/tls" value="kubernetes.io/tls" />
            <el-option label="kubernetes.io/dockerconfigjson" value="kubernetes.io/dockerconfigjson" />
          </el-select>
        </el-form-item>
        <el-form-item label="不可变">
          <el-switch v-model="form.immutable" />
        </el-form-item>
        <el-divider content-position="left">字符串数据</el-divider>
        <div v-for="(entry, idx) in form.secretEntries" :key="`secret-${idx}`" class="entry-row entry-row--stacked">
          <el-input v-model="entry.key" placeholder="键，例如 username" class="entry-row__key" />
          <el-input v-model="entry.value" type="textarea" :autosize="{ minRows: 2, maxRows: 6 }" placeholder="值" class="entry-row__value" />
          <el-button v-if="form.secretEntries.length > 1" class="remove-btn" size="small" @click="removeSecretEntry(idx)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
        <el-button class="add-row-btn" @click="addSecretEntry">
          <el-icon><Plus /></el-icon>
          <span>添加字符串数据</span>
        </el-button>
      </template>

      <template v-else-if="resource === 'serviceaccounts'">
        <el-form-item label="自动挂载令牌">
          <el-switch v-model="form.serviceAccountAutomount" />
        </el-form-item>
        <el-divider content-position="left">ImagePullSecrets</el-divider>
        <div v-for="(entry, idx) in form.serviceAccountImagePullSecrets" :key="`sapull-${idx}`" class="entry-row entry-row--single">
          <el-input v-model="entry.name" placeholder="Secret 名称，例如 registry-auth" class="entry-row__full" />
          <el-button v-if="form.serviceAccountImagePullSecrets.length > 1" class="remove-btn" size="small" @click="removeServiceAccountImagePullSecret(idx)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
        <el-button class="add-row-btn" @click="addServiceAccountImagePullSecret">
          <el-icon><Plus /></el-icon>
          <span>添加 ImagePullSecret</span>
        </el-button>
      </template>

      <template v-else-if="resource === 'hpas'">
        <el-form-item label="目标类型" required>
          <el-select v-model="form.hpaTargetKind" style="width: 100%">
            <el-option label="Deployment" value="Deployment" />
            <el-option label="StatefulSet" value="StatefulSet" />
          </el-select>
        </el-form-item>
        <el-form-item label="目标名称" required>
          <el-input v-model="form.hpaTargetName" placeholder="例如 sample-app" clearable />
        </el-form-item>
        <div class="grid-two">
          <el-form-item label="最小副本" required>
            <el-input-number v-model="form.hpaMinReplicas" :min="1" :max="1000" style="width: 100%" />
          </el-form-item>
          <el-form-item label="最大副本" required>
            <el-input-number v-model="form.hpaMaxReplicas" :min="1" :max="1000" style="width: 100%" />
          </el-form-item>
        </div>
        <el-form-item label="CPU 目标%" required>
          <el-input-number v-model="form.hpaCpuUtilization" :min="1" :max="100" style="width: 100%" />
        </el-form-item>
      </template>

      <template v-else-if="resource === 'pdbs'">
        <el-form-item label="选择器" required>
          <el-input v-model="form.pdbSelector" placeholder="例如 app=sample-app" clearable />
        </el-form-item>
        <el-form-item label="保护方式" required>
          <el-radio-group v-model="form.pdbMode">
            <el-radio-button label="minAvailable">MinAvailable</el-radio-button>
            <el-radio-button label="maxUnavailable">MaxUnavailable</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="form.pdbMode === 'minAvailable' ? '最少可用' : '最多不可用'" required>
          <el-input v-model="form.pdbValue" placeholder="例如 1 或 50%" clearable />
        </el-form-item>
      </template>

      <template v-else-if="resource === 'networkpolicies'">
        <el-form-item label="Pod 选择器">
          <el-input v-model="form.networkPolicyPodSelector" placeholder="留空表示全部 Pod，例如 app=sample-app" clearable />
        </el-form-item>
        <el-form-item label="策略类型" required>
          <el-checkbox-group v-model="form.networkPolicyTypes">
            <el-checkbox label="Ingress">Ingress</el-checkbox>
            <el-checkbox label="Egress">Egress</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item v-if="form.networkPolicyTypes.includes('Ingress')" label="Ingress 默认规则">
          <el-switch v-model="form.networkPolicyAllowSameNamespaceIngress" inline-prompt active-text="同命名空间放通" inactive-text="默认拒绝" />
        </el-form-item>
        <el-form-item v-if="form.networkPolicyTypes.includes('Egress')" label="Egress 默认规则">
          <el-switch v-model="form.networkPolicyAllowSameNamespaceEgress" inline-prompt active-text="同命名空间放通" inactive-text="默认拒绝" />
        </el-form-item>
      </template>

      <template v-else-if="resource === 'resourcequotas'">
        <el-divider content-position="left">Hard 配额</el-divider>
        <div v-for="(entry, idx) in form.resourceQuotaEntries" :key="`quota-${idx}`" class="entry-row entry-row--compact">
          <el-input v-model="entry.key" placeholder="资源键，例如 requests.cpu" class="entry-row__key" />
          <el-input v-model="entry.value" placeholder="配额值，例如 4 或 8Gi" class="entry-row__value" />
          <el-button v-if="form.resourceQuotaEntries.length > 1" class="remove-btn" size="small" @click="removeResourceQuotaEntry(idx)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
        <el-button class="add-row-btn" @click="addResourceQuotaEntry">
          <el-icon><Plus /></el-icon>
          <span>添加配额项</span>
        </el-button>
      </template>

      <template v-else-if="resource === 'limitranges'">
        <el-form-item label="限制类型" required>
          <el-select v-model="form.limitRangeType" style="width: 100%">
            <el-option label="Container" value="Container" />
            <el-option label="Pod" value="Pod" />
            <el-option label="PersistentVolumeClaim" value="PersistentVolumeClaim" />
          </el-select>
        </el-form-item>
        <div class="grid-two">
          <el-form-item label="默认 CPU">
            <el-input v-model="form.limitDefaultCpu" placeholder="例如 500m" clearable />
          </el-form-item>
          <el-form-item label="默认内存">
            <el-input v-model="form.limitDefaultMemory" placeholder="例如 512Mi" clearable />
          </el-form-item>
          <el-form-item label="默认请求 CPU">
            <el-input v-model="form.limitDefaultRequestCpu" placeholder="例如 100m" clearable />
          </el-form-item>
          <el-form-item label="默认请求内存">
            <el-input v-model="form.limitDefaultRequestMemory" placeholder="例如 128Mi" clearable />
          </el-form-item>
          <el-form-item label="最大 CPU">
            <el-input v-model="form.limitMaxCpu" placeholder="例如 1" clearable />
          </el-form-item>
          <el-form-item label="最大内存">
            <el-input v-model="form.limitMaxMemory" placeholder="例如 1Gi" clearable />
          </el-form-item>
          <el-form-item label="最小 CPU">
            <el-input v-model="form.limitMinCpu" placeholder="例如 50m" clearable />
          </el-form-item>
          <el-form-item label="最小内存">
            <el-input v-model="form.limitMinMemory" placeholder="例如 64Mi" clearable />
          </el-form-item>
        </div>
      </template>
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
import { Close, Plus } from '@element-plus/icons-vue'
import { stringify } from 'yaml'

import { applyManifest } from '@/features/k8s/api/manifest'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'

type SupportedCreateResource = 'configmaps' | 'secrets' | 'serviceaccounts' | 'hpas' | 'pdbs' | 'networkpolicies' | 'resourcequotas' | 'limitranges'

type KeyValueEntry = { key: string; value: string }
type NameEntry = { name: string }

const RESOURCE_LABELS: Record<SupportedCreateResource, string> = {
  configmaps: 'ConfigMap',
  secrets: 'Secret',
  serviceaccounts: 'ServiceAccount',
  hpas: 'HPA',
  pdbs: 'PDB',
  networkpolicies: 'NetworkPolicy',
  resourcequotas: 'ResourceQuota',
  limitranges: 'LimitRange',
}

const props = defineProps<{
  modelValue: boolean
  clusterId: number
  resource: SupportedCreateResource
  namespaces: string[]
  defaultNamespace?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'created'): void
}>()

const submitting = ref(false)

const form = reactive({
  namespace: '',
  name: '',
  immutable: false,
  configEntries: [] as KeyValueEntry[],
  secretType: 'Opaque',
  secretEntries: [] as KeyValueEntry[],
  serviceAccountAutomount: true,
  serviceAccountImagePullSecrets: [] as NameEntry[],
  hpaTargetKind: 'Deployment',
  hpaTargetName: '',
  hpaMinReplicas: 1,
  hpaMaxReplicas: 3,
  hpaCpuUtilization: 70,
  pdbSelector: '',
  pdbMode: 'minAvailable',
  pdbValue: '1',
  networkPolicyPodSelector: '',
  networkPolicyTypes: ['Ingress'] as string[],
  networkPolicyAllowSameNamespaceIngress: false,
  networkPolicyAllowSameNamespaceEgress: false,
  resourceQuotaEntries: [] as KeyValueEntry[],
  limitRangeType: 'Container',
  limitDefaultCpu: '',
  limitDefaultMemory: '',
  limitDefaultRequestCpu: '',
  limitDefaultRequestMemory: '',
  limitMaxCpu: '',
  limitMaxMemory: '',
  limitMinCpu: '',
  limitMinMemory: '',
})

const dialogTitle = computed(() => `创建 ${RESOURCE_LABELS[props.resource]}`)

const namePlaceholder = computed(() => {
  switch (props.resource) {
    case 'configmaps':
      return '例如 app-config'
    case 'secrets':
      return '例如 app-secret'
    case 'serviceaccounts':
      return '例如 app-sa'
    case 'hpas':
      return '例如 app-hpa'
    case 'pdbs':
      return '例如 app-pdb'
    case 'networkpolicies':
      return '例如 default-deny'
    case 'resourcequotas':
      return '例如 team-quota'
    case 'limitranges':
      return '例如 team-limits'
  }
})

const submitDisabled = computed(() => {
  if (!props.clusterId || !form.namespace.trim() || !form.name.trim()) return true
  switch (props.resource) {
    case 'configmaps':
      return form.configEntries.length === 0 || form.configEntries.some((entry) => !entry.key.trim())
    case 'secrets':
      return form.secretEntries.length === 0 || form.secretEntries.some((entry) => !entry.key.trim())
    case 'serviceaccounts':
      return false
    case 'hpas':
      return !form.hpaTargetName.trim() || form.hpaMaxReplicas < form.hpaMinReplicas || form.hpaCpuUtilization <= 0
    case 'pdbs':
      return !form.pdbSelector.trim() || !form.pdbValue.trim()
    case 'networkpolicies':
      return form.networkPolicyTypes.length === 0
    case 'resourcequotas':
      return form.resourceQuotaEntries.length === 0 || form.resourceQuotaEntries.some((entry) => !entry.key.trim() || !entry.value.trim())
    case 'limitranges':
      return !hasAnyLimitRangeValue()
  }
})

watch(
  () => [props.modelValue, props.resource, props.defaultNamespace, props.namespaces.join('\u0000')],
  ([visible]) => {
    if (!visible) return
    resetForm()
  }
)

function makeKeyValueEntry(key = '', value = ''): KeyValueEntry {
  return { key, value }
}

function makeNameEntry(name = ''): NameEntry {
  return { name }
}

function resetForm() {
  form.namespace = props.defaultNamespace || props.namespaces[0] || 'default'
  form.name = ''
  form.immutable = false
  form.configEntries = [makeKeyValueEntry('application.yaml', 'key: value\n')]
  form.secretType = 'Opaque'
  form.secretEntries = [makeKeyValueEntry('username', 'admin'), makeKeyValueEntry('password', 'change-me')]
  form.serviceAccountAutomount = true
  form.serviceAccountImagePullSecrets = [makeNameEntry('')]
  form.hpaTargetKind = 'Deployment'
  form.hpaTargetName = ''
  form.hpaMinReplicas = 1
  form.hpaMaxReplicas = 3
  form.hpaCpuUtilization = 70
  form.pdbSelector = 'app=sample-app'
  form.pdbMode = 'minAvailable'
  form.pdbValue = '1'
  form.networkPolicyPodSelector = ''
  form.networkPolicyTypes = ['Ingress']
  form.networkPolicyAllowSameNamespaceIngress = false
  form.networkPolicyAllowSameNamespaceEgress = false
  form.resourceQuotaEntries = [
    makeKeyValueEntry('requests.cpu', '4'),
    makeKeyValueEntry('requests.memory', '8Gi'),
    makeKeyValueEntry('limits.cpu', '8'),
    makeKeyValueEntry('limits.memory', '16Gi'),
  ]
  form.limitRangeType = 'Container'
  form.limitDefaultCpu = '500m'
  form.limitDefaultMemory = '512Mi'
  form.limitDefaultRequestCpu = '100m'
  form.limitDefaultRequestMemory = '128Mi'
  form.limitMaxCpu = ''
  form.limitMaxMemory = ''
  form.limitMinCpu = ''
  form.limitMinMemory = ''
}

function addConfigEntry() {
  form.configEntries.push(makeKeyValueEntry())
}

function removeConfigEntry(index: number) {
  form.configEntries.splice(index, 1)
}

function addSecretEntry() {
  form.secretEntries.push(makeKeyValueEntry())
}

function removeSecretEntry(index: number) {
  form.secretEntries.splice(index, 1)
}

function addServiceAccountImagePullSecret() {
  form.serviceAccountImagePullSecrets.push(makeNameEntry())
}

function removeServiceAccountImagePullSecret(index: number) {
  form.serviceAccountImagePullSecrets.splice(index, 1)
}

function addResourceQuotaEntry() {
  form.resourceQuotaEntries.push(makeKeyValueEntry())
}

function removeResourceQuotaEntry(index: number) {
  form.resourceQuotaEntries.splice(index, 1)
}

function hasAnyLimitRangeValue(): boolean {
  return [
    form.limitDefaultCpu,
    form.limitDefaultMemory,
    form.limitDefaultRequestCpu,
    form.limitDefaultRequestMemory,
    form.limitMaxCpu,
    form.limitMaxMemory,
    form.limitMinCpu,
    form.limitMinMemory,
  ].some((value) => String(value ?? '').trim() !== '')
}

function collectEntries(entries: KeyValueEntry[], label: string, allowEmptyValue = true): Record<string, string> {
  const out: Record<string, string> = {}
  const seen = new Set<string>()
  for (const entry of entries) {
    const key = String(entry.key ?? '').trim()
    const value = String(entry.value ?? '')
    if (!key) throw new Error(`${label}存在空键，请先补全`)
    if (!allowEmptyValue && !value.trim()) throw new Error(`${label} ${key} 不能为空`)
    if (seen.has(key)) throw new Error(`${label} ${key} 重复，请调整后重试`)
    seen.add(key)
    out[key] = value
  }
  return out
}

function collectNames(entries: NameEntry[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const entry of entries) {
    const name = String(entry.name ?? '').trim()
    if (!name || seen.has(name)) continue
    seen.add(name)
    out.push(name)
  }
  return out
}

function parseSelector(input: string, fieldLabel: string, allowEmpty = true): Record<string, string> {
  const raw = String(input ?? '').trim()
  if (!raw) {
    if (allowEmpty) return {}
    throw new Error(`${fieldLabel}不能为空`)
  }
  const out: Record<string, string> = {}
  for (const segment of raw.split(',')) {
    const part = segment.trim()
    if (!part) continue
    const equalIndex = part.indexOf('=')
    if (equalIndex <= 0 || equalIndex === part.length - 1) {
      throw new Error(`${fieldLabel}格式错误，请使用 key=value 并用逗号分隔`)
    }
    const key = part.slice(0, equalIndex).trim()
    const value = part.slice(equalIndex + 1).trim()
    if (!key || !value) {
      throw new Error(`${fieldLabel}格式错误，请使用 key=value 并用逗号分隔`)
    }
    out[key] = value
  }
  return out
}

function parseIntOrString(value: string): number | string {
  const text = String(value ?? '').trim()
  if (/^\d+$/.test(text)) return Number(text)
  return text
}

function compactRecord(input: Record<string, unknown>): Record<string, unknown> | undefined {
  const out: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(input)) {
    if (value == null) continue
    if (typeof value === 'string' && value.trim() === '') continue
    if (Array.isArray(value) && value.length === 0) continue
    if (typeof value === 'object' && !Array.isArray(value) && Object.keys(value as Record<string, unknown>).length === 0) continue
    out[key] = value
  }
  return Object.keys(out).length ? out : undefined
}

function buildManifestObject(): Record<string, unknown> {
  const metadata = { name: form.name.trim(), namespace: form.namespace.trim() }

  if (props.resource === 'configmaps') {
    const data = collectEntries(form.configEntries, 'ConfigMap 数据项')
    return compactRecord({
      apiVersion: 'v1',
      kind: 'ConfigMap',
      metadata,
      immutable: form.immutable ? true : undefined,
      data,
    }) as Record<string, unknown>
  }

  if (props.resource === 'secrets') {
    const stringData = collectEntries(form.secretEntries, 'Secret 字符串数据')
    return compactRecord({
      apiVersion: 'v1',
      kind: 'Secret',
      metadata,
      type: form.secretType,
      immutable: form.immutable ? true : undefined,
      stringData,
    }) as Record<string, unknown>
  }

  if (props.resource === 'serviceaccounts') {
    const imagePullSecrets = collectNames(form.serviceAccountImagePullSecrets).map((name) => ({ name }))
    return compactRecord({
      apiVersion: 'v1',
      kind: 'ServiceAccount',
      metadata,
      automountServiceAccountToken: form.serviceAccountAutomount ? undefined : false,
      imagePullSecrets,
    }) as Record<string, unknown>
  }

  if (props.resource === 'hpas') {
    return {
      apiVersion: 'autoscaling/v2',
      kind: 'HorizontalPodAutoscaler',
      metadata,
      spec: {
        scaleTargetRef: {
          apiVersion: 'apps/v1',
          kind: form.hpaTargetKind,
          name: form.hpaTargetName.trim(),
        },
        minReplicas: form.hpaMinReplicas,
        maxReplicas: form.hpaMaxReplicas,
        metrics: [
          {
            type: 'Resource',
            resource: {
              name: 'cpu',
              target: {
                type: 'Utilization',
                averageUtilization: form.hpaCpuUtilization,
              },
            },
          },
        ],
      },
    }
  }

  if (props.resource === 'pdbs') {
    return {
      apiVersion: 'policy/v1',
      kind: 'PodDisruptionBudget',
      metadata,
      spec: {
        selector: {
          matchLabels: parseSelector(form.pdbSelector, 'PDB 选择器', false),
        },
        [form.pdbMode]: parseIntOrString(form.pdbValue),
      },
    }
  }

  if (props.resource === 'networkpolicies') {
    const policyTypes = [...new Set(form.networkPolicyTypes.map((item) => String(item).trim()).filter(Boolean))]
    const spec: Record<string, unknown> = {
      podSelector: compactRecord({ matchLabels: parseSelector(form.networkPolicyPodSelector, 'NetworkPolicy Pod 选择器') }) ?? {},
      policyTypes,
    }
    if (policyTypes.includes('Ingress') && form.networkPolicyAllowSameNamespaceIngress) {
      spec.ingress = [{ from: [{ podSelector: {} }] }]
    }
    if (policyTypes.includes('Egress') && form.networkPolicyAllowSameNamespaceEgress) {
      spec.egress = [{ to: [{ podSelector: {} }] }]
    }
    return {
      apiVersion: 'networking.k8s.io/v1',
      kind: 'NetworkPolicy',
      metadata,
      spec,
    }
  }

  if (props.resource === 'resourcequotas') {
    const hard = collectEntries(form.resourceQuotaEntries, 'ResourceQuota 配额项', false)
    return {
      apiVersion: 'v1',
      kind: 'ResourceQuota',
      metadata,
      spec: { hard },
    }
  }

  const defaults = compactRecord({ cpu: form.limitDefaultCpu.trim(), memory: form.limitDefaultMemory.trim() })
  const defaultRequests = compactRecord({ cpu: form.limitDefaultRequestCpu.trim(), memory: form.limitDefaultRequestMemory.trim() })
  const max = compactRecord({ cpu: form.limitMaxCpu.trim(), memory: form.limitMaxMemory.trim() })
  const min = compactRecord({ cpu: form.limitMinCpu.trim(), memory: form.limitMinMemory.trim() })
  const limitItem = compactRecord({
    type: form.limitRangeType,
    default: defaults,
    defaultRequest: defaultRequests,
    max,
    min,
  })
  if (!limitItem || Object.keys(limitItem).length <= 1) {
    throw new Error('LimitRange 至少需要填写一项默认值、请求值、最大值或最小值')
  }
  return {
    apiVersion: 'v1',
    kind: 'LimitRange',
    metadata,
    spec: { limits: [limitItem] },
  }
}

async function submit() {
  if (submitDisabled.value) return
  submitting.value = true
  try {
    const resource = buildManifestObject()
    const yaml = stringify(resource)
    await applyManifest(props.clusterId, {
      yaml,
      default_namespace: form.namespace.trim(),
      create_only: true,
      source_label: dialogTitle.value,
      source_resource: props.resource,
    })
    notifySuccess(`${RESOURCE_LABELS[props.resource]} 已创建`)
    emit('update:modelValue', false)
    emit('created')
  } catch (error) {
    const err = error as ApiError
    const fallback = error instanceof Error ? error.message : '创建失败'
    const message = String(err?.message ?? '').trim() || fallback
    notifyError(err?.requestId ? `${message} (request_id=${err.requestId})` : message)
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.create-dialog-note {
  margin-bottom: 16px;
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid rgba(59, 130, 246, 0.16);
  background: rgba(59, 130, 246, 0.06);
  color: var(--color-text-secondary, #475569);
  font-size: 13px;
}

.grid-two {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 12px;
}

.entry-row {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  margin-bottom: 10px;
  padding: 12px;
  border-radius: 10px;
  border: 1px solid var(--color-border-subtle, rgba(15, 23, 42, 0.08));
  background: var(--color-bg-muted, #f8fafc);
}

.entry-row--stacked {
  flex-direction: column;
}

.entry-row--compact {
  align-items: center;
}

.entry-row--single {
  align-items: center;
}

.entry-row__key {
  width: 240px;
  flex-shrink: 0;
}

.entry-row__value {
  flex: 1;
}

.entry-row__full {
  flex: 1;
}

.remove-btn {
  width: 32px;
  height: 32px;
  padding: 0;
  border-radius: 8px;
  border: 1px solid rgba(239, 68, 68, 0.2);
  background: rgba(239, 68, 68, 0.08);
  color: #ef4444;
  flex-shrink: 0;
}

.remove-btn:hover {
  background: rgba(239, 68, 68, 0.14);
  border-color: rgba(239, 68, 68, 0.32);
}

.add-row-btn {
  width: 100%;
  height: 40px;
  border-radius: 10px;
  border: 1px dashed var(--color-border-default, rgba(15, 23, 42, 0.12));
  background: transparent;
  color: var(--color-accent-primary, #3b82f6);
  font-weight: 600;
}

.add-row-btn:hover {
  border-color: rgba(59, 130, 246, 0.36);
  background: rgba(59, 130, 246, 0.04);
}

@media (max-width: 900px) {
  .grid-two {
    grid-template-columns: 1fr;
  }

  .entry-row--compact {
    flex-direction: column;
    align-items: stretch;
  }

  .entry-row__key {
    width: 100%;
  }
}
</style>