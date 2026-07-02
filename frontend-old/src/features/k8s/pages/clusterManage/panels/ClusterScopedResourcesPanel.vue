<template>
  <EnhancedTable
    ref="tableRef"
    :data="data"
    :columns="columns"
    :persist-key="persistKey"
    :show-tools="showTools"
    row-key="metadata.name"
    size="small"
    stripe
    border
    @sort-change="emit('sort-change', $event)"
  >
    <template #cell-name="{ row }">
      <span class="k8s-name">{{ String(row?.metadata?.name ?? '-') }}</span>
    </template>
    <template v-if="showCrdColumns" #cell-group="{ row }">
      <span class="k8s-age">{{ String(row?.spec?.group ?? '-') }}</span>
    </template>
    <template v-if="showCrdColumns" #cell-scope="{ row }">
      <span class="k8s-status k8s-status--info">{{ String(row?.spec?.scope ?? '-') }}</span>
    </template>
    <template v-if="showCrdColumns" #cell-resource="{ row }">
      <span class="k8s-age">{{ String(row?.spec?.names?.plural ?? '-') }}</span>
    </template>
    <template v-if="showCrdColumns" #cell-versions="{ row }">
      <span class="k8s-age" :title="getCrdVersionsText(row)">{{ getCrdVersionsText(row) }}</span>
    </template>
    <template v-if="showApiServiceColumns" #cell-version="{ row }">
      <span class="k8s-age">{{ getApiServiceVersionText(row) }}</span>
    </template>
    <template v-if="showApiServiceColumns" #cell-backend="{ row }">
      <span class="k8s-age" :title="getApiServiceBackendText(row)">{{ getApiServiceBackendText(row) }}</span>
    </template>
    <template v-if="showApiServiceColumns" #cell-tlsMode="{ row }">
      <span class="k8s-status k8s-status--info">{{ getApiServiceTlsModeText(row) }}</span>
    </template>
    <template v-if="showPriorityClassColumns" #cell-priorityValue="{ row }">
      <span class="k8s-num">{{ getPriorityValueText(row) }}</span>
    </template>
    <template v-if="showPriorityClassColumns" #cell-defaultClass="{ row }">
      <span :class="['k8s-status', row?.globalDefault ? 'k8s-status--ok' : 'k8s-status--info']">{{ row?.globalDefault ? 'yes' : 'no' }}</span>
    </template>
    <template v-if="showPriorityClassColumns" #cell-preemptionPolicy="{ row }">
      <span class="k8s-age">{{ getPriorityPreemptionPolicyText(row) }}</span>
    </template>
    <template v-if="showRuntimeClassColumns" #cell-handler="{ row }">
      <span class="k8s-age">{{ getRuntimeHandlerText(row) }}</span>
    </template>
    <template v-if="showRuntimeClassColumns" #cell-overhead="{ row }">
      <span class="k8s-num">{{ getRuntimeOverheadCount(row) }}</span>
    </template>
    <template v-if="showRuntimeClassColumns" #cell-scheduling="{ row }">
      <span class="k8s-age" :title="getRuntimeSchedulingText(row)">{{ getRuntimeSchedulingText(row) }}</span>
    </template>
    <template v-if="showWebhookColumns" #cell-webhooksCount="{ row }">
      <span class="k8s-num">{{ getWebhookCount(row) }}</span>
    </template>
    <template v-if="showWebhookColumns || showAdmissionPolicyColumns" #cell-failurePolicy="{ row }">
      <span class="k8s-age" :title="getFailurePolicyText(row)">{{ getFailurePolicyText(row) }}</span>
    </template>
    <template v-if="showWebhookColumns" #cell-rulesCount="{ row }">
      <span class="k8s-num">{{ getWebhookRulesCount(row) }}</span>
    </template>
    <template v-if="showAdmissionPolicyColumns" #cell-validations="{ row }">
      <span class="k8s-num">{{ getAdmissionPolicyValidationsCount(row) }}</span>
    </template>
    <template v-if="showAdmissionPolicyColumns" #cell-paramKind="{ row }">
      <span class="k8s-age" :title="getAdmissionPolicyParamKindText(row)">{{ getAdmissionPolicyParamKindText(row) }}</span>
    </template>
    <template v-if="showAdmissionPolicyBindingColumns" #cell-policy="{ row }">
      <span class="k8s-age" :title="getAdmissionPolicyBindingPolicyName(row)">{{ getAdmissionPolicyBindingPolicyName(row) }}</span>
    </template>
    <template v-if="showAdmissionPolicyBindingColumns" #cell-paramRef="{ row }">
      <span class="k8s-age" :title="getAdmissionPolicyBindingParamRefText(row)">{{ getAdmissionPolicyBindingParamRefText(row) }}</span>
    </template>
    <template v-if="showAdmissionPolicyBindingColumns" #cell-actionsText="{ row }">
      <span class="k8s-age" :title="getAdmissionPolicyBindingActionsText(row)">{{ getAdmissionPolicyBindingActionsText(row) }}</span>
    </template>
    <template #cell-summary="{ row }">
      <span v-if="showStructuredSummary" class="k8s-age">{{ props.getSummary(row) }}</span>
    </template>
    <template #cell-age="{ row }">
      <span class="k8s-age">{{ getCreationAgeText(row) }}</span>
    </template>
    <template #cell-actions="{ row }">
      <div class="k8s-act-group">
        <el-tooltip v-if="props.openDetail" content="详情" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--info" @click="props.openDetail(row)"><el-icon><View /></el-icon></button></el-tooltip>
        <el-tooltip v-if="props.canWrite" content="编辑" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--edit" @click="props.openEdit(row)"><el-icon><Edit /></el-icon></button></el-tooltip>
        <el-tooltip content="YAML" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--violet" @click="props.openYaml(row)"><el-icon><Document /></el-icon></button></el-tooltip>
        <el-tooltip v-if="props.canWrite" content="删除" placement="top" :show-after="300"><button class="k8s-act-btn k8s-act-btn--danger" @click="props.deleteRow(row)"><el-icon><Delete /></el-icon></button></el-tooltip>
      </div>
    </template>
  </EnhancedTable>
</template>

<script setup lang="ts">
import { Delete, Document, Edit, View } from '@element-plus/icons-vue'
import { computed, ref } from 'vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import { getCreationAgeText } from '@/features/k8s/pages/ClusterManageView.utils'

type ClusterScopedResourceKind =
  | 'customresourcedefinitions'
  | 'apiservices'
  | 'priorityclasses'
  | 'runtimeclasses'
  | 'validatingwebhookconfigurations'
  | 'mutatingwebhookconfigurations'
  | 'validatingadmissionpolicies'
  | 'validatingadmissionpolicybindings'

const props = defineProps<{
  data: any[]
  persistKey: string
  showTools: boolean
  canWrite: boolean
  summaryLabel?: string
  resourceKind?: ClusterScopedResourceKind
  getSummary: (row: any) => string
  openDetail?: (row: any) => void
  openYaml: (row: any) => void
  openEdit: (row: any) => void
  deleteRow: (row: any) => void
}>()

const emit = defineEmits<{ (e: 'sort-change', v: any): void }>()

const showCrdColumns = computed(() => props.resourceKind === 'customresourcedefinitions')
const showApiServiceColumns = computed(() => props.resourceKind === 'apiservices')
const showPriorityClassColumns = computed(() => props.resourceKind === 'priorityclasses')
const showRuntimeClassColumns = computed(() => props.resourceKind === 'runtimeclasses')
const showWebhookColumns = computed(() => props.resourceKind === 'validatingwebhookconfigurations' || props.resourceKind === 'mutatingwebhookconfigurations')
const showAdmissionPolicyColumns = computed(() => props.resourceKind === 'validatingadmissionpolicies')
const showAdmissionPolicyBindingColumns = computed(() => props.resourceKind === 'validatingadmissionpolicybindings')
const showStructuredSummary = computed(() => !(
  showCrdColumns.value ||
  showApiServiceColumns.value ||
  showPriorityClassColumns.value ||
  showRuntimeClassColumns.value ||
  showWebhookColumns.value ||
  showAdmissionPolicyColumns.value ||
  showAdmissionPolicyBindingColumns.value
))

const columns = computed<EnhancedColumn[]>(() => {
  const base: EnhancedColumn[] = [{ key: 'name', label: '名称', prop: 'metadata.name', minWidth: 280, sortable: 'custom', defaultVisible: true }]
  if (showCrdColumns.value) {
    base.push(
      { key: 'group', label: 'Group', minWidth: 220, defaultVisible: true },
      { key: 'scope', label: 'Scope', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'resource', label: 'Resource', minWidth: 180, defaultVisible: true },
      { key: 'versions', label: 'Versions', minWidth: 220, defaultVisible: true }
    )
  }
  if (showApiServiceColumns.value) {
    base.push(
      { key: 'version', label: 'Version', minWidth: 180, defaultVisible: true },
      { key: 'backend', label: 'Backend', minWidth: 220, defaultVisible: true },
      { key: 'tlsMode', label: 'TLS', width: 120, align: 'center', headerAlign: 'center', defaultVisible: true }
    )
  }
  if (showPriorityClassColumns.value) {
    base.push(
      { key: 'priorityValue', label: 'Value', width: 140, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'defaultClass', label: 'Default', width: 100, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'preemptionPolicy', label: 'Preemption', minWidth: 180, defaultVisible: true }
    )
  }
  if (showRuntimeClassColumns.value) {
    base.push(
      { key: 'handler', label: 'Handler', minWidth: 200, defaultVisible: true },
      { key: 'overhead', label: 'Overhead', width: 100, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'scheduling', label: 'Scheduling', minWidth: 220, defaultVisible: true }
    )
  }
  if (showWebhookColumns.value) {
    base.push(
      { key: 'webhooksCount', label: 'Webhooks', width: 100, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'failurePolicy', label: 'FailurePolicy', minWidth: 160, defaultVisible: true },
      { key: 'rulesCount', label: 'Rules', width: 90, align: 'center', headerAlign: 'center', defaultVisible: true }
    )
  }
  if (showAdmissionPolicyColumns.value) {
    base.push(
      { key: 'failurePolicy', label: 'FailurePolicy', minWidth: 160, defaultVisible: true },
      { key: 'validations', label: 'Validations', width: 110, align: 'center', headerAlign: 'center', defaultVisible: true },
      { key: 'paramKind', label: 'ParamKind', minWidth: 200, defaultVisible: true }
    )
  }
  if (showAdmissionPolicyBindingColumns.value) {
    base.push(
      { key: 'policy', label: 'Policy', minWidth: 220, defaultVisible: true },
      { key: 'paramRef', label: 'ParamRef', minWidth: 220, defaultVisible: true },
      { key: 'actionsText', label: 'Actions', minWidth: 180, defaultVisible: true }
    )
  }
  if (showStructuredSummary.value) {
    base.push({ key: 'summary', label: props.summaryLabel || '摘要', minWidth: 420, defaultVisible: true })
  }
  base.push(
    { key: 'age', label: 'AGE', prop: 'metadata.creationTimestamp', width: 110, sortable: 'custom', align: 'center', headerAlign: 'center', defaultVisible: true },
    { key: 'actions', label: '操作', width: 160, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
  )
  return base
})

function getCrdVersionsText(row: any): string {
  const versions: any[] = Array.isArray(row?.spec?.versions) ? row.spec.versions : []
  const parts = versions
    .map((item) => {
      const name = String(item?.name ?? '').trim()
      if (!name) return ''
      return item?.storage ? `${name}*` : name
    })
    .filter(Boolean)
  return parts.join(', ') || '-'
}

function getApiServiceVersionText(row: any): string {
  const group = String(row?.spec?.group ?? '').trim()
  const version = String(row?.spec?.version ?? '').trim()
  if (!group && !version) return '-'
  return `${group || 'core'}/${version || '-'}`
}

function getApiServiceBackendText(row: any): string {
  const ns = String(row?.spec?.service?.namespace ?? '').trim()
  const name = String(row?.spec?.service?.name ?? '').trim()
  if (!ns && !name) return 'local'
  return ns && name ? `${ns}/${name}` : ns || name
}

function getApiServiceTlsModeText(row: any): string {
  if (row?.spec?.insecureSkipTLSVerify === true) return 'skip-verify'
  const caBundle = String(row?.spec?.caBundle ?? '').trim()
  return caBundle ? 'ca-bundle' : 'default'
}

function getPriorityValueText(row: any): string {
  return row?.value != null ? String(row.value) : '-'
}

function getPriorityPreemptionPolicyText(row: any): string {
  const value = String(row?.preemptionPolicy ?? '').trim()
  return value || 'PreemptLowerPriority'
}

function getRuntimeHandlerText(row: any): string {
  const value = String(row?.handler ?? '').trim()
  return value || '-'
}

function getRuntimeOverheadCount(row: any): number {
  const podFixed = row?.overhead?.podFixed
  if (!podFixed || typeof podFixed !== 'object' || Array.isArray(podFixed)) return 0
  return Object.keys(podFixed as Record<string, unknown>).length
}

function getRuntimeSchedulingText(row: any): string {
  const nodeSelector = row?.scheduling?.nodeSelector
  const tolerations: any[] = Array.isArray(row?.scheduling?.tolerations) ? row.scheduling.tolerations : []
  const selectorCount = nodeSelector && typeof nodeSelector === 'object' && !Array.isArray(nodeSelector)
    ? Object.keys(nodeSelector as Record<string, unknown>).length
    : 0
  if (selectorCount === 0 && tolerations.length === 0) return '-'
  return `selector=${selectorCount}, tolerations=${tolerations.length}`
}

function getWebhookCount(row: any): number {
  return Array.isArray(row?.webhooks) ? row.webhooks.length : 0
}

function getFailurePolicyText(row: any): string {
  if (showAdmissionPolicyColumns.value) {
    const value = String(row?.spec?.failurePolicy ?? '').trim()
    return value || '-'
  }
  const webhooks: any[] = Array.isArray(row?.webhooks) ? row.webhooks : []
  const values = Array.from(
    new Set(
      webhooks.map((item) => String(item?.failurePolicy ?? '').trim()).filter(Boolean)
    )
  )
  return values.join(', ') || '-'
}

function getWebhookRulesCount(row: any): number {
  const webhooks: any[] = Array.isArray(row?.webhooks) ? row.webhooks : []
  return webhooks.reduce((sum, item) => sum + (Array.isArray(item?.rules) ? item.rules.length : 0), 0)
}

function getAdmissionPolicyValidationsCount(row: any): number {
  return Array.isArray(row?.spec?.validations) ? row.spec.validations.length : 0
}

function getAdmissionPolicyParamKindText(row: any): string {
  const apiVersion = String(row?.spec?.paramKind?.apiVersion ?? '').trim()
  const kind = String(row?.spec?.paramKind?.kind ?? '').trim()
  if (!apiVersion && !kind) return '-'
  return apiVersion && kind ? `${apiVersion}/${kind}` : apiVersion || kind
}

function getAdmissionPolicyBindingPolicyName(row: any): string {
  return String(row?.spec?.policyName ?? '-').trim() || '-'
}

function getAdmissionPolicyBindingParamRefText(row: any): string {
  const ns = String(row?.spec?.paramRef?.namespace ?? '').trim()
  const name = String(row?.spec?.paramRef?.name ?? '').trim()
  if (!ns && !name) return '-'
  return ns && name ? `${ns}/${name}` : ns || name
}

function getAdmissionPolicyBindingActionsText(row: any): string {
  const actions: any[] = Array.isArray(row?.spec?.validationActions) ? row.spec.validationActions : []
  const values = actions.map((item) => String(item ?? '').trim()).filter(Boolean)
  return values.join(', ') || '-'
}

const tableRef = ref<any>(null)
defineExpose({ getTable: () => tableRef.value })
</script>
