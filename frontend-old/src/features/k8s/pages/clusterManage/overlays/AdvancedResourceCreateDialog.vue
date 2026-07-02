<template>
  <el-dialog :model-value="modelValue" :title="dialogTitle" width="860px" destroy-on-close @close="emit('update:modelValue', false)">
    <div class="create-dialog-note">结构化创建会生成规范 Manifest 并以 create_only 模式提交，不会覆盖已有对象。</div>

    <el-form label-width="124px" @submit.prevent>
      <el-form-item v-if="showNamespaceField" label="命名空间" required>
        <el-select v-model="form.namespace" placeholder="选择命名空间" style="width: 100%" filterable>
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
      </el-form-item>

      <el-form-item v-if="showNameField" label="名称" required>
        <el-input v-model="form.name" :placeholder="namePlaceholder" clearable />
      </el-form-item>

      <template v-if="isRoleResource">
        <el-divider content-position="left">Rules</el-divider>
        <div v-for="(entry, idx) in form.roleRules" :key="`rule-${idx}`" class="entry-row entry-row--stacked">
          <div class="grid-two">
            <el-input v-model="entry.apiGroups" placeholder="API Groups，逗号分隔；留空表示 core" />
            <el-input v-model="entry.verbs" placeholder="Verbs，必填，例如 get,list,watch" />
            <el-input v-model="entry.resources" placeholder="Resources，逗号分隔，例如 pods,deployments" />
            <el-input v-model="entry.resourceNames" placeholder="ResourceNames，可选，逗号分隔" />
          </div>
          <el-input v-model="entry.nonResourceURLs" placeholder="NonResourceURLs，可选，逗号分隔，例如 /healthz,/metrics" />
          <el-button v-if="form.roleRules.length > 1" class="remove-btn" size="small" @click="removeRoleRule(idx)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
        <el-button class="add-row-btn" @click="addRoleRule">
          <el-icon><Plus /></el-icon>
          <span>添加 Rule</span>
        </el-button>
      </template>

      <template v-else-if="isBindingResource">
        <el-form-item label="RoleRef Kind" required>
          <el-select v-model="form.bindingRoleRefKind" style="width: 100%">
            <el-option v-for="item in roleRefKindOptions" :key="item" :label="item" :value="item" />
          </el-select>
        </el-form-item>
        <el-form-item label="RoleRef Name" required>
          <el-input v-model="form.bindingRoleRefName" placeholder="例如 view 或 app-reader" clearable />
        </el-form-item>

        <el-divider content-position="left">Subjects</el-divider>
        <div v-for="(entry, idx) in form.bindingSubjects" :key="`subject-${idx}`" class="entry-row entry-row--stacked">
          <div class="grid-three">
            <el-select v-model="entry.kind" placeholder="Kind">
              <el-option label="ServiceAccount" value="ServiceAccount" />
              <el-option label="User" value="User" />
              <el-option label="Group" value="Group" />
            </el-select>
            <el-input v-model="entry.name" placeholder="名称，例如 app-sa / alice" clearable />
            <el-input v-model="entry.namespace" :disabled="entry.kind !== 'ServiceAccount'" placeholder="ServiceAccount Namespace" clearable />
          </div>
          <el-button v-if="form.bindingSubjects.length > 1" class="remove-btn" size="small" @click="removeBindingSubject(idx)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
        <el-button class="add-row-btn" @click="addBindingSubject">
          <el-icon><Plus /></el-icon>
          <span>添加 Subject</span>
        </el-button>
      </template>

      <template v-else-if="resource === 'ingressclasses'">
        <el-form-item label="Controller" required>
          <el-input v-model="form.ingressClassController" placeholder="例如 k8s.io/ingress-nginx" clearable />
        </el-form-item>
        <el-form-item label="默认类">
          <el-switch v-model="form.ingressClassIsDefault" />
        </el-form-item>
        <el-form-item label="Parameters JSON">
          <el-input
            v-model="form.ingressClassParametersText"
            type="textarea"
            :autosize="{ minRows: 5, maxRows: 10 }"
            :placeholder="ingressClassParametersPlaceholder"
          />
        </el-form-item>
      </template>

      <template v-else-if="resource === 'storageclasses'">
        <el-form-item label="Provisioner" required>
          <el-input v-model="form.storageClassProvisioner" placeholder="例如 kubernetes.io/no-provisioner 或 csi.example.com" clearable />
        </el-form-item>
        <div class="grid-two">
          <el-form-item label="ReclaimPolicy">
            <el-select v-model="form.storageClassReclaimPolicy" style="width: 100%">
              <el-option label="Delete" value="Delete" />
              <el-option label="Retain" value="Retain" />
            </el-select>
          </el-form-item>
          <el-form-item label="BindingMode">
            <el-select v-model="form.storageClassVolumeBindingMode" style="width: 100%">
              <el-option label="Immediate" value="Immediate" />
              <el-option label="WaitForFirstConsumer" value="WaitForFirstConsumer" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item label="默认类">
          <el-switch v-model="form.storageClassIsDefault" />
        </el-form-item>
        <el-form-item label="允许扩容">
          <el-switch v-model="form.storageClassAllowVolumeExpansion" />
        </el-form-item>
        <el-form-item label="MountOptions">
          <el-input v-model="form.storageClassMountOptions" placeholder="可选，逗号分隔，例如 discard,noatime" clearable />
        </el-form-item>
        <el-divider content-position="left">Parameters</el-divider>
        <div v-for="(entry, idx) in form.storageClassParameters" :key="`sc-param-${idx}`" class="entry-row entry-row--compact">
          <el-input v-model="entry.key" placeholder="参数名" class="entry-row__key" />
          <el-input v-model="entry.value" placeholder="参数值" class="entry-row__value" />
          <el-button v-if="form.storageClassParameters.length > 1" class="remove-btn" size="small" @click="removeStorageClassParameter(idx)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
        <el-button class="add-row-btn" @click="addStorageClassParameter">
          <el-icon><Plus /></el-icon>
          <span>添加 Parameter</span>
        </el-button>
      </template>

      <template v-else-if="resource === 'customresourcedefinitions'">
        <div class="grid-two">
          <el-form-item label="API Group" required>
            <el-input v-model="form.crdGroup" placeholder="例如 apps.example.com" clearable />
          </el-form-item>
          <el-form-item label="Plural" required>
            <el-input v-model="form.crdPlural" placeholder="例如 widgets" clearable />
          </el-form-item>
          <el-form-item label="Singular">
            <el-input v-model="form.crdSingular" placeholder="默认自动推导" clearable />
          </el-form-item>
          <el-form-item label="Kind" required>
            <el-input v-model="form.crdKind" placeholder="例如 Widget" clearable />
          </el-form-item>
          <el-form-item label="Version" required>
            <el-input v-model="form.crdVersion" placeholder="例如 v1alpha1" clearable />
          </el-form-item>
          <el-form-item label="Scope" required>
            <el-select v-model="form.crdScope" style="width: 100%">
              <el-option label="Namespaced" value="Namespaced" />
              <el-option label="Cluster" value="Cluster" />
            </el-select>
          </el-form-item>
        </div>
        <div class="create-preview-name">metadata.name = {{ crdFullName }}</div>
        <el-divider content-position="left">Short Names</el-divider>
        <div v-for="(entry, idx) in form.crdShortNames" :key="`crd-short-${idx}`" class="entry-row entry-row--single">
          <el-input v-model="entry.name" placeholder="例如 wdg" class="entry-row__full" />
          <el-button v-if="form.crdShortNames.length > 1" class="remove-btn" size="small" @click="removeCrdShortName(idx)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
        <el-button class="add-row-btn" @click="addCrdShortName">
          <el-icon><Plus /></el-icon>
          <span>添加 Short Name</span>
        </el-button>
        <el-divider content-position="left">Categories</el-divider>
        <div v-for="(entry, idx) in form.crdCategories" :key="`crd-category-${idx}`" class="entry-row entry-row--single">
          <el-input v-model="entry.name" placeholder="例如 all" class="entry-row__full" />
          <el-button v-if="form.crdCategories.length > 1" class="remove-btn" size="small" @click="removeCrdCategory(idx)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
        <el-button class="add-row-btn" @click="addCrdCategory">
          <el-icon><Plus /></el-icon>
          <span>添加 Category</span>
        </el-button>
        <el-form-item label="OpenAPI Schema" required>
          <el-input
            v-model="form.crdSchemaText"
            type="textarea"
            :autosize="{ minRows: 10, maxRows: 20 }"
            placeholder="输入 OpenAPI v3 Schema 的 JSON"
          />
        </el-form-item>
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

type AdvancedCreateResource =
  | 'roles'
  | 'clusterroles'
  | 'rolebindings'
  | 'clusterrolebindings'
  | 'ingressclasses'
  | 'storageclasses'
  | 'customresourcedefinitions'

type KeyValueEntry = { key: string; value: string }
type NameEntry = { name: string }
type RoleRuleEntry = {
  apiGroups: string
  resources: string
  verbs: string
  resourceNames: string
  nonResourceURLs: string
}
type SubjectEntry = { kind: 'ServiceAccount' | 'User' | 'Group'; name: string; namespace: string }

const RESOURCE_LABELS: Record<AdvancedCreateResource, string> = {
  roles: 'Role',
  clusterroles: 'ClusterRole',
  rolebindings: 'RoleBinding',
  clusterrolebindings: 'ClusterRoleBinding',
  ingressclasses: 'IngressClass',
  storageclasses: 'StorageClass',
  customresourcedefinitions: 'CRD',
}

const props = defineProps<{
  modelValue: boolean
  clusterId: number
  resource: AdvancedCreateResource
  namespaces: string[]
  defaultNamespace?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'created'): void
}>()

const CLUSTER_SCOPED_RESOURCES = new Set<AdvancedCreateResource>([
  'clusterroles',
  'clusterrolebindings',
  'ingressclasses',
  'storageclasses',
  'customresourcedefinitions',
])

const RBAC_RESOURCES = new Set<AdvancedCreateResource>(['roles', 'clusterroles', 'rolebindings', 'clusterrolebindings'])

const submitting = ref(false)
const ingressClassParametersPlaceholder = `可选，例如 {
  "apiGroup": "k8s.example.com",
  "kind": "IngressParameters",
  "name": "default"
}`

const form = reactive({
  namespace: '',
  name: '',
  roleRules: [] as RoleRuleEntry[],
  bindingRoleRefKind: 'Role' as 'Role' | 'ClusterRole',
  bindingRoleRefName: '',
  bindingSubjects: [] as SubjectEntry[],
  ingressClassController: '',
  ingressClassIsDefault: false,
  ingressClassParametersText: '',
  storageClassProvisioner: '',
  storageClassReclaimPolicy: 'Delete',
  storageClassVolumeBindingMode: 'Immediate',
  storageClassAllowVolumeExpansion: true,
  storageClassIsDefault: false,
  storageClassMountOptions: '',
  storageClassParameters: [] as KeyValueEntry[],
  crdGroup: '',
  crdPlural: '',
  crdSingular: '',
  crdKind: '',
  crdVersion: 'v1alpha1',
  crdScope: 'Namespaced',
  crdShortNames: [] as NameEntry[],
  crdCategories: [] as NameEntry[],
  crdSchemaText: '',
})

const isRoleResource = computed(() => props.resource === 'roles' || props.resource === 'clusterroles')
const isBindingResource = computed(() => props.resource === 'rolebindings' || props.resource === 'clusterrolebindings')
const showNamespaceField = computed(() => !CLUSTER_SCOPED_RESOURCES.has(props.resource))
const showNameField = computed(() => props.resource !== 'customresourcedefinitions')
const dialogTitle = computed(() => `创建 ${RESOURCE_LABELS[props.resource]}`)
const crdFullName = computed(() => {
  const plural = String(form.crdPlural ?? '').trim()
  const group = String(form.crdGroup ?? '').trim()
  if (!plural || !group) return '-'
  return `${plural}.${group}`
})

const namePlaceholder = computed(() => {
  switch (props.resource) {
    case 'roles':
      return '例如 app-reader'
    case 'clusterroles':
      return '例如 audit-reader'
    case 'rolebindings':
      return '例如 app-reader-binding'
    case 'clusterrolebindings':
      return '例如 audit-reader-binding'
    case 'ingressclasses':
      return '例如 nginx-public'
    case 'storageclasses':
      return '例如 fast-ssd'
    case 'customresourcedefinitions':
      return ''
  }
})

const roleRefKindOptions = computed(() => {
  if (props.resource === 'clusterrolebindings') return ['ClusterRole']
  return ['Role', 'ClusterRole']
})

const submitDisabled = computed(() => {
  if (!props.clusterId) return true
  if (showNamespaceField.value && !form.namespace.trim()) return true
  if (showNameField.value && !form.name.trim()) return true

  if (isRoleResource.value) {
    return form.roleRules.length === 0 || form.roleRules.some((entry) => !String(entry.verbs ?? '').trim())
  }
  if (isBindingResource.value) {
    return !form.bindingRoleRefName.trim() || form.bindingSubjects.length === 0 || form.bindingSubjects.some((entry) => !entry.name.trim())
  }
  if (props.resource === 'ingressclasses') {
    return !form.ingressClassController.trim()
  }
  if (props.resource === 'storageclasses') {
    return !form.storageClassProvisioner.trim()
  }
  return !form.crdGroup.trim() || !form.crdPlural.trim() || !form.crdKind.trim() || !form.crdVersion.trim() || !form.crdSchemaText.trim()
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

function makeRoleRuleEntry(): RoleRuleEntry {
  return {
    apiGroups: '',
    resources: '',
    verbs: 'get,list,watch',
    resourceNames: '',
    nonResourceURLs: '',
  }
}

function makeSubjectEntry(kind: SubjectEntry['kind'] = 'ServiceAccount', name = '', namespace = ''): SubjectEntry {
  return { kind, name, namespace }
}

function inferSingular(plural: string): string {
  const value = String(plural ?? '').trim()
  if (!value) return ''
  if (value.endsWith('ies') && value.length > 3) return `${value.slice(0, -3)}y`
  if (value.endsWith('s') && value.length > 1) return value.slice(0, -1)
  return value
}

function defaultCrdSchemaText() {
  return JSON.stringify({
    type: 'object',
    properties: {
      spec: {
        type: 'object',
        'x-kubernetes-preserve-unknown-fields': true,
      },
      status: {
        type: 'object',
        'x-kubernetes-preserve-unknown-fields': true,
      },
    },
  }, null, 2)
}

function resetForm() {
  form.namespace = props.defaultNamespace || props.namespaces[0] || 'default'
  form.name = ''
  form.roleRules = [makeRoleRuleEntry()]
  form.bindingRoleRefKind = props.resource === 'clusterrolebindings' ? 'ClusterRole' : 'Role'
  form.bindingRoleRefName = ''
  form.bindingSubjects = [makeSubjectEntry('ServiceAccount', '', form.namespace)]
  form.ingressClassController = 'k8s.io/ingress-nginx'
  form.ingressClassIsDefault = false
  form.ingressClassParametersText = ''
  form.storageClassProvisioner = ''
  form.storageClassReclaimPolicy = 'Delete'
  form.storageClassVolumeBindingMode = 'Immediate'
  form.storageClassAllowVolumeExpansion = true
  form.storageClassIsDefault = false
  form.storageClassMountOptions = ''
  form.storageClassParameters = [makeKeyValueEntry()]
  form.crdGroup = 'apps.example.com'
  form.crdPlural = 'widgets'
  form.crdSingular = 'widget'
  form.crdKind = 'Widget'
  form.crdVersion = 'v1alpha1'
  form.crdScope = 'Namespaced'
  form.crdShortNames = [makeNameEntry('wdg')]
  form.crdCategories = [makeNameEntry('all')]
  form.crdSchemaText = defaultCrdSchemaText()
}

function addRoleRule() {
  form.roleRules.push(makeRoleRuleEntry())
}

function removeRoleRule(index: number) {
  form.roleRules.splice(index, 1)
}

function addBindingSubject() {
  form.bindingSubjects.push(makeSubjectEntry('ServiceAccount', '', showNamespaceField.value ? form.namespace : 'default'))
}

function removeBindingSubject(index: number) {
  form.bindingSubjects.splice(index, 1)
}

function addStorageClassParameter() {
  form.storageClassParameters.push(makeKeyValueEntry())
}

function removeStorageClassParameter(index: number) {
  form.storageClassParameters.splice(index, 1)
}

function addCrdShortName() {
  form.crdShortNames.push(makeNameEntry())
}

function removeCrdShortName(index: number) {
  form.crdShortNames.splice(index, 1)
}

function addCrdCategory() {
  form.crdCategories.push(makeNameEntry())
}

function removeCrdCategory(index: number) {
  form.crdCategories.splice(index, 1)
}

function parseCsvList(input: string, fieldLabel: string, required = false): string[] {
  const text = String(input ?? '').trim()
  if (!text) {
    if (required) throw new Error(`${fieldLabel}不能为空`)
    return []
  }
  const out = text.split(',').map((item) => item.trim()).filter(Boolean)
  if (!out.length && required) throw new Error(`${fieldLabel}不能为空`)
  return Array.from(new Set(out))
}

function parseApiGroups(input: string): string[] {
  const text = String(input ?? '').trim()
  if (!text) return ['']
  const values = text.split(',').map((item) => item.trim()).filter((item, index, arr) => item.length > 0 && arr.indexOf(item) === index)
  if (!values.length) return ['']
  return values.map((item) => (item === 'core' ? '' : item))
}

function collectEntries(entries: KeyValueEntry[], label: string): Record<string, string> {
  const out: Record<string, string> = {}
  for (const entry of entries) {
    const key = String(entry.key ?? '').trim()
    const value = String(entry.value ?? '').trim()
    if (!key && !value) continue
    if (!key) throw new Error(`${label}存在空键，请先补全`)
    out[key] = value
  }
  return out
}

function collectNames(entries: NameEntry[]): string[] {
  return Array.from(new Set(entries.map((entry) => String(entry.name ?? '').trim()).filter(Boolean)))
}

function parseJsonObject(text: string, fieldLabel: string): Record<string, unknown> {
  const raw = String(text ?? '').trim()
  if (!raw) throw new Error(`${fieldLabel}不能为空`)
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    throw new Error(`${fieldLabel} 不是合法 JSON`)
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error(`${fieldLabel} 必须是 JSON 对象`)
  return parsed as Record<string, unknown>
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

function buildMetadata(): Record<string, unknown> {
  const name = props.resource === 'customresourcedefinitions' ? crdFullName.value : form.name.trim()
  return compactRecord({
    name,
    namespace: showNamespaceField.value ? form.namespace.trim() : undefined,
  }) as Record<string, unknown>
}

function buildRoleRules() {
  return form.roleRules.map((entry, index) => {
    const verbs = parseCsvList(entry.verbs, `Rule ${index + 1} Verbs`, true)
    const resources = parseCsvList(entry.resources, `Rule ${index + 1} Resources`)
    const nonResourceURLs = parseCsvList(entry.nonResourceURLs, `Rule ${index + 1} NonResourceURLs`)
    if (!resources.length && !nonResourceURLs.length) {
      throw new Error(`Rule ${index + 1} 至少需要填写 Resources 或 NonResourceURLs`)
    }
    return compactRecord({
      apiGroups: resources.length ? parseApiGroups(entry.apiGroups) : undefined,
      resources,
      verbs,
      resourceNames: parseCsvList(entry.resourceNames, `Rule ${index + 1} ResourceNames`),
      nonResourceURLs,
    }) as Record<string, unknown>
  })
}

function buildSubjects() {
  return form.bindingSubjects.map((entry, index) => {
    const kind = entry.kind
    const name = String(entry.name ?? '').trim()
    if (!name) throw new Error(`Subject ${index + 1} 名称不能为空`)
    const subjectNamespace = String(entry.namespace ?? '').trim()
    if (kind === 'ServiceAccount' && !subjectNamespace) {
      throw new Error(`Subject ${index + 1} 为 ServiceAccount 时必须填写 Namespace`)
    }
    return compactRecord({
      kind,
      name,
      namespace: kind === 'ServiceAccount' ? subjectNamespace : undefined,
      apiGroup: kind === 'ServiceAccount' ? undefined : 'rbac.authorization.k8s.io',
    }) as Record<string, unknown>
  })
}

function buildManifestObject(): Record<string, unknown> {
  const metadata = buildMetadata()

  if (isRoleResource.value) {
    return {
      apiVersion: 'rbac.authorization.k8s.io/v1',
      kind: props.resource === 'roles' ? 'Role' : 'ClusterRole',
      metadata,
      rules: buildRoleRules(),
    }
  }

  if (isBindingResource.value) {
    const roleRefKind = props.resource === 'clusterrolebindings' ? 'ClusterRole' : form.bindingRoleRefKind
    return {
      apiVersion: 'rbac.authorization.k8s.io/v1',
      kind: props.resource === 'rolebindings' ? 'RoleBinding' : 'ClusterRoleBinding',
      metadata,
      subjects: buildSubjects(),
      roleRef: {
        apiGroup: 'rbac.authorization.k8s.io',
        kind: roleRefKind,
        name: form.bindingRoleRefName.trim(),
      },
    }
  }

  if (props.resource === 'ingressclasses') {
    const annotations = compactRecord({
      'ingressclass.kubernetes.io/is-default-class': form.ingressClassIsDefault ? 'true' : undefined,
    })
    return compactRecord({
      apiVersion: 'networking.k8s.io/v1',
      kind: 'IngressClass',
      metadata: compactRecord({ ...metadata, annotations }),
      spec: compactRecord({
        controller: form.ingressClassController.trim(),
        parameters: form.ingressClassParametersText.trim() ? parseJsonObject(form.ingressClassParametersText, 'IngressClass Parameters') : undefined,
      }),
    }) as Record<string, unknown>
  }

  if (props.resource === 'storageclasses') {
    const annotations = compactRecord({
      'storageclass.kubernetes.io/is-default-class': form.storageClassIsDefault ? 'true' : undefined,
    })
    return compactRecord({
      apiVersion: 'storage.k8s.io/v1',
      kind: 'StorageClass',
      metadata: compactRecord({ ...metadata, annotations }),
      provisioner: form.storageClassProvisioner.trim(),
      reclaimPolicy: form.storageClassReclaimPolicy,
      volumeBindingMode: form.storageClassVolumeBindingMode,
      allowVolumeExpansion: form.storageClassAllowVolumeExpansion,
      mountOptions: parseCsvList(form.storageClassMountOptions, 'MountOptions'),
      parameters: collectEntries(form.storageClassParameters, 'StorageClass Parameters'),
    }) as Record<string, unknown>
  }

  return {
    apiVersion: 'apiextensions.k8s.io/v1',
    kind: 'CustomResourceDefinition',
    metadata,
    spec: {
      group: form.crdGroup.trim(),
      scope: form.crdScope,
      names: compactRecord({
        plural: form.crdPlural.trim(),
        singular: form.crdSingular.trim() || inferSingular(form.crdPlural),
        kind: form.crdKind.trim(),
        listKind: `${form.crdKind.trim()}List`,
        shortNames: collectNames(form.crdShortNames),
        categories: collectNames(form.crdCategories),
      }),
      versions: [
        {
          name: form.crdVersion.trim(),
          served: true,
          storage: true,
          schema: {
            openAPIV3Schema: parseJsonObject(form.crdSchemaText, 'CRD OpenAPI Schema'),
          },
        },
      ],
    },
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
      default_namespace: showNamespaceField.value ? form.namespace.trim() : undefined,
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

.create-preview-name {
  margin: 0 0 16px;
  padding: 10px 12px;
  border-radius: 10px;
  background: var(--color-bg-muted, #f8fafc);
  border: 1px solid var(--color-border-subtle, rgba(15, 23, 42, 0.08));
  color: var(--color-text-secondary, #475569);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, Courier New, monospace;
  font-size: 12px;
}

.grid-two {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 12px;
}

.grid-three {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
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

.entry-row--compact,
.entry-row--single {
  align-items: center;
}

.entry-row__key {
  width: 240px;
  flex-shrink: 0;
}

.entry-row__value,
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
  .grid-two,
  .grid-three {
    grid-template-columns: 1fr;
  }

  .entry-row--compact,
  .entry-row--single {
    flex-direction: column;
    align-items: stretch;
  }

  .entry-row__key {
    width: 100%;
  }
}
</style>