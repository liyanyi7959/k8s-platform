<template>
  <K8sScaleDialog v-if="loaded.scaleDialog" ref="scaleDialogRef" :cluster-id="props.clusterId" :cluster-name="props.clusterName" @scaled="emit('refresh-list')" />

  <DeploymentEditDrawer v-if="loaded.deploymentEdit" ref="deploymentEditDrawerRef" :cluster-id="props.clusterId" :cluster-name="props.clusterName" @saved="emit('refresh-list')" />
  <DeploymentEditDrawer
    v-if="loaded.statefulSetEdit"
    ref="statefulSetEditDrawerRef"
    :cluster-id="props.clusterId"
    :cluster-name="props.clusterName"
    workload-kind="StatefulSet"
    @saved="emit('refresh-list')"
  />
  <DeploymentEditDrawer
    v-if="loaded.daemonSetEdit"
    ref="daemonSetEditDrawerRef"
    :cluster-id="props.clusterId"
    :cluster-name="props.clusterName"
    workload-kind="DaemonSet"
    @saved="emit('refresh-list')"
  />

  <K8sYamlDrawer v-if="loaded.yamlDrawer" ref="yamlDrawerRef" />
  <WorkloadRolloutDrawer v-if="loaded.workloadRollout" ref="workloadRolloutRef" :cluster-id="props.clusterId" @rolled-back="emit('refresh-list')" />

  <ServiceDetailDrawer
    v-if="loaded.serviceDetail"
    ref="serviceDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @toggle-editor-theme="emit('toggle-editor-theme')"
    @open-related-pod="emit('open-related-pod', $event)"
    @open-topology="emit('open-topology', $event)"
    @refresh-list="emit('refresh-list')"
  />
  <IngressDetailDrawer
    v-if="loaded.ingressDetail"
    ref="ingressDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @toggle-editor-theme="emit('toggle-editor-theme')"
    @open-related-pod="emit('open-related-pod', $event)"
    @refresh-list="emit('refresh-list')"
  />
  <PVCDetailDrawer
    v-if="loaded.pvcDetail"
    ref="pvcDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @toggle-editor-theme="emit('toggle-editor-theme')"
    @open-related-pod="emit('open-related-pod', $event)"
    @open-topology="emit('open-topology', $event)"
    @refresh-list="emit('refresh-list')"
  />
  <PVDetailDrawer
    v-if="loaded.pvDetail"
    ref="pvDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @toggle-editor-theme="emit('toggle-editor-theme')"
    @open-related-pod="emit('open-related-pod', $event)"
    @refresh-list="emit('refresh-list')"
  />
  <IngressClassDetailDrawer v-if="loaded.ingressClassDetail" ref="ingressClassDetailRef" :cluster-id="props.clusterId" :list="props.list" />
  <CustomResourceDefinitionDetailDrawer
    v-if="loaded.customResourceDefinitionDetail"
    ref="customResourceDefinitionDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @refresh-list="emit('refresh-list')"
  />
  <APIServiceDetailDrawer
    v-if="loaded.apiServiceDetail"
    ref="apiServiceDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @refresh-list="emit('refresh-list')"
  />
  <PriorityClassDetailDrawer
    v-if="loaded.priorityClassDetail"
    ref="priorityClassDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @refresh-list="emit('refresh-list')"
  />
  <RuntimeClassDetailDrawer
    v-if="loaded.runtimeClassDetail"
    ref="runtimeClassDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @refresh-list="emit('refresh-list')"
  />
  <WebhookConfigurationDetailDrawer
    v-if="loaded.webhookConfigurationDetail"
    ref="webhookConfigurationDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @refresh-list="emit('refresh-list')"
  />
  <ValidatingAdmissionPolicyDetailDrawer
    v-if="loaded.validatingAdmissionPolicyDetail"
    ref="validatingAdmissionPolicyDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @refresh-list="emit('refresh-list')"
  />
  <ValidatingAdmissionPolicyBindingDetailDrawer
    v-if="loaded.validatingAdmissionPolicyBindingDetail"
    ref="validatingAdmissionPolicyBindingDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @refresh-list="emit('refresh-list')"
  />
  <ConfigMapDetailDrawer
    v-if="loaded.configMapDetail"
    ref="configMapDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @toggle-editor-theme="emit('toggle-editor-theme')"
    @open-related-pod="emit('open-related-pod', $event)"
    @refresh-list="emit('refresh-list')"
  />
  <ServiceAccountDetailDrawer
    v-if="loaded.serviceAccountDetail"
    ref="serviceAccountDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @toggle-editor-theme="emit('toggle-editor-theme')"
    @open-related-pod="emit('open-related-pod', $event)"
    @refresh-list="emit('refresh-list')"
  />
  <ResourceQuotaDetailDrawer
    v-if="loaded.resourceQuotaDetail"
    ref="resourceQuotaDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @refresh-list="emit('refresh-list')"
  />
  <NetworkPolicyDetailDrawer
    v-if="loaded.networkPolicyDetail"
    ref="networkPolicyDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @open-related-pod="emit('open-related-pod', $event)"
    @refresh-list="emit('refresh-list')"
  />
  <RbacRoleDetailDrawer
    v-if="loaded.rbacRoleDetail"
    ref="rbacRoleDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @refresh-list="emit('refresh-list')"
  />
  <RbacBindingDetailDrawer
    v-if="loaded.rbacBindingDetail"
    ref="rbacBindingDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @refresh-list="emit('refresh-list')"
  />
  <HPADetailDrawer
    v-if="loaded.hpaDetail"
    ref="hpaDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @refresh-list="emit('refresh-list')"
  />
  <PdbDetailDrawer
    v-if="loaded.pdbDetail"
    ref="pdbDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @open-related-pod="emit('open-related-pod', $event)"
    @refresh-list="emit('refresh-list')"
  />
  <SecretDetailDrawer
    v-if="loaded.secretDetail"
    ref="secretDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @toggle-editor-theme="emit('toggle-editor-theme')"
    @open-related-pod="emit('open-related-pod', $event)"
    @refresh-list="emit('refresh-list')"
  />
  <JobDetailDrawer
    v-if="loaded.jobDetail"
    ref="jobDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @open-pod-detail="emit('open-pod-detail', $event)"
    @open-yaml="forwardOpenYaml"
    @open-related="emit('open-related', $event)"
    @refresh-list="emit('refresh-list')"
  />
  <CronJobDetailDrawer
    v-if="loaded.cronJobDetail"
    ref="cronJobDetailRef"
    :cluster-id="props.clusterId"
    :editor-theme="props.editorTheme"
    :editor-theme-effective-dark="props.editorThemeEffectiveDark"
    :list="props.list"
    @open-pod-detail="emit('open-pod-detail', $event)"
    @open-job-detail="emit('open-job-detail', $event)"
    @open-yaml="forwardOpenYaml"
    @refresh-list="emit('refresh-list')"
  />

  <PodDetailDrawer
    v-if="loaded.podDetail"
    ref="podDetailDrawerRef"
    :cluster-id="props.clusterId"
    @open-logs="emit('pod-detail-open-logs', $event)"
    @open-exec="emit('pod-detail-open-exec', $event)"
    @open-related="emit('pod-detail-open-related', $event)"
    @open-topology="emit('open-topology', $event)"
  />

  <NodeDetailDrawer
    v-if="loaded.nodeDetail"
    ref="nodeDetailDrawerRef"
    :cluster-id="props.clusterId"
    @open-pod-detail="emit('pod-detail', $event)"
    @open-topology="emit('open-topology', $event)"
  />

  <DeploymentDetailDrawer
    v-if="loaded.deploymentDetail"
    ref="deploymentDetailDrawerRef"
    :cluster-id="props.clusterId"
    @pod-detail="emit('pod-detail', $event)"
    @pod-log="emit('pod-log', $event)"
    @pod-exec="emit('pod-exec', $event)"
  />
  <StatefulSetDetailDrawer
    v-if="loaded.statefulSetDetail"
    ref="statefulSetDetailDrawerRef"
    :cluster-id="props.clusterId"
    @pod-detail="emit('pod-detail', $event)"
    @pod-log="emit('pod-log', $event)"
    @pod-exec="emit('pod-exec', $event)"
  />
</template>

<script setup lang="ts">
import { defineAsyncComponent, nextTick, reactive, ref } from 'vue'

import type DeploymentEditDrawerComponent from '@/features/k8s/components/DeploymentEditDrawer.vue'
import type K8sScaleDialogComponent from '@/features/k8s/components/K8sScaleDialog.vue'
import type K8sYamlDrawerComponent from '@/features/k8s/components/K8sYamlDrawer.vue'
import type DeploymentDetailDrawerComponent from '@/features/k8s/components/DeploymentDetailDrawer.vue'
import type StatefulSetDetailDrawerComponent from '@/features/k8s/components/StatefulSetDetailDrawer.vue'
import type PodDetailDrawerComponent from '@/features/k8s/components/PodDetailDrawer.vue'
import type NodeDetailDrawerComponent from '@/features/k8s/components/NodeDetailDrawer.vue'
import type WorkloadRolloutDrawerComponent from './WorkloadRolloutDrawer.vue'
import type ServiceDetailDrawerComponent from '../details/ServiceDetailDrawer.vue'
import type IngressDetailDrawerComponent from '../details/IngressDetailDrawer.vue'
import type PVCDetailDrawerComponent from '../details/PVCDetailDrawer.vue'
import type PVDetailDrawerComponent from '../details/PVDetailDrawer.vue'
import type IngressClassDetailDrawerComponent from '../details/IngressClassDetailDrawer.vue'
import type CustomResourceDefinitionDetailDrawerComponent from '../details/CustomResourceDefinitionDetailDrawer.vue'
import type APIServiceDetailDrawerComponent from '../details/APIServiceDetailDrawer.vue'
import type PriorityClassDetailDrawerComponent from '../details/PriorityClassDetailDrawer.vue'
import type RuntimeClassDetailDrawerComponent from '../details/RuntimeClassDetailDrawer.vue'
import type WebhookConfigurationDetailDrawerComponent from '../details/WebhookConfigurationDetailDrawer.vue'
import type ValidatingAdmissionPolicyDetailDrawerComponent from '../details/ValidatingAdmissionPolicyDetailDrawer.vue'
import type ValidatingAdmissionPolicyBindingDetailDrawerComponent from '../details/ValidatingAdmissionPolicyBindingDetailDrawer.vue'
import type ConfigMapDetailDrawerComponent from '../details/ConfigMapDetailDrawer.vue'
import type ServiceAccountDetailDrawerComponent from '../details/ServiceAccountDetailDrawer.vue'
import type ResourceQuotaDetailDrawerComponent from '../details/ResourceQuotaDetailDrawer.vue'
import type NetworkPolicyDetailDrawerComponent from '../details/NetworkPolicyDetailDrawer.vue'
import type RbacRoleDetailDrawerComponent from '../details/RbacRoleDetailDrawer.vue'
import type RbacBindingDetailDrawerComponent from '../details/RbacBindingDetailDrawer.vue'
import type HPADetailDrawerComponent from '../details/HPADetailDrawer.vue'
import type PdbDetailDrawerComponent from '../details/PdbDetailDrawer.vue'
import type SecretDetailDrawerComponent from '../details/SecretDetailDrawer.vue'
import type JobDetailDrawerComponent from '../details/JobDetailDrawer.vue'
import type CronJobDetailDrawerComponent from '../details/CronJobDetailDrawer.vue'

const DeploymentEditDrawer = defineAsyncComponent(() => import('@/features/k8s/components/DeploymentEditDrawer.vue'))
const K8sScaleDialog = defineAsyncComponent(() => import('@/features/k8s/components/K8sScaleDialog.vue'))
const K8sYamlDrawer = defineAsyncComponent(() => import('@/features/k8s/components/K8sYamlDrawer.vue'))
const WorkloadRolloutDrawer = defineAsyncComponent(() => import('./WorkloadRolloutDrawer.vue'))
const DeploymentDetailDrawer = defineAsyncComponent(() => import('@/features/k8s/components/DeploymentDetailDrawer.vue'))
const StatefulSetDetailDrawer = defineAsyncComponent(() => import('@/features/k8s/components/StatefulSetDetailDrawer.vue'))
const PodDetailDrawer = defineAsyncComponent(() => import('@/features/k8s/components/PodDetailDrawer.vue'))
const NodeDetailDrawer = defineAsyncComponent(() => import('@/features/k8s/components/NodeDetailDrawer.vue'))
const ServiceDetailDrawer = defineAsyncComponent(() => import('../details/ServiceDetailDrawer.vue'))
const IngressDetailDrawer = defineAsyncComponent(() => import('../details/IngressDetailDrawer.vue'))
const PVCDetailDrawer = defineAsyncComponent(() => import('../details/PVCDetailDrawer.vue'))
const PVDetailDrawer = defineAsyncComponent(() => import('../details/PVDetailDrawer.vue'))
const IngressClassDetailDrawer = defineAsyncComponent(() => import('../details/IngressClassDetailDrawer.vue'))
const CustomResourceDefinitionDetailDrawer = defineAsyncComponent(() => import('../details/CustomResourceDefinitionDetailDrawer.vue'))
const APIServiceDetailDrawer = defineAsyncComponent(() => import('../details/APIServiceDetailDrawer.vue'))
const PriorityClassDetailDrawer = defineAsyncComponent(() => import('../details/PriorityClassDetailDrawer.vue'))
const RuntimeClassDetailDrawer = defineAsyncComponent(() => import('../details/RuntimeClassDetailDrawer.vue'))
const WebhookConfigurationDetailDrawer = defineAsyncComponent(() => import('../details/WebhookConfigurationDetailDrawer.vue'))
const ValidatingAdmissionPolicyDetailDrawer = defineAsyncComponent(() => import('../details/ValidatingAdmissionPolicyDetailDrawer.vue'))
const ValidatingAdmissionPolicyBindingDetailDrawer = defineAsyncComponent(() => import('../details/ValidatingAdmissionPolicyBindingDetailDrawer.vue'))
const ConfigMapDetailDrawer = defineAsyncComponent(() => import('../details/ConfigMapDetailDrawer.vue'))
const ServiceAccountDetailDrawer = defineAsyncComponent(() => import('../details/ServiceAccountDetailDrawer.vue'))
const ResourceQuotaDetailDrawer = defineAsyncComponent(() => import('../details/ResourceQuotaDetailDrawer.vue'))
const NetworkPolicyDetailDrawer = defineAsyncComponent(() => import('../details/NetworkPolicyDetailDrawer.vue'))
const RbacRoleDetailDrawer = defineAsyncComponent(() => import('../details/RbacRoleDetailDrawer.vue'))
const RbacBindingDetailDrawer = defineAsyncComponent(() => import('../details/RbacBindingDetailDrawer.vue'))
const HPADetailDrawer = defineAsyncComponent(() => import('../details/HPADetailDrawer.vue'))
const PdbDetailDrawer = defineAsyncComponent(() => import('../details/PdbDetailDrawer.vue'))
const SecretDetailDrawer = defineAsyncComponent(() => import('../details/SecretDetailDrawer.vue'))
const JobDetailDrawer = defineAsyncComponent(() => import('../details/JobDetailDrawer.vue'))
const CronJobDetailDrawer = defineAsyncComponent(() => import('../details/CronJobDetailDrawer.vue'))

type EditorTheme = 'auto' | 'light' | 'dark'
type YamlLoader = () => Promise<{ text: string }>
type YamlSaver = (text: string) => Promise<void>
type LazyKey =
  | 'scaleDialog'
  | 'deploymentEdit'
  | 'statefulSetEdit'
  | 'daemonSetEdit'
  | 'yamlDrawer'
  | 'workloadRollout'
  | 'serviceDetail'
  | 'ingressDetail'
  | 'pvcDetail'
  | 'pvDetail'
  | 'ingressClassDetail'
  | 'customResourceDefinitionDetail'
  | 'apiServiceDetail'
  | 'priorityClassDetail'
  | 'runtimeClassDetail'
  | 'webhookConfigurationDetail'
  | 'validatingAdmissionPolicyDetail'
  | 'validatingAdmissionPolicyBindingDetail'
  | 'configMapDetail'
  | 'serviceAccountDetail'
  | 'resourceQuotaDetail'
  | 'networkPolicyDetail'
  | 'rbacRoleDetail'
  | 'rbacBindingDetail'
  | 'hpaDetail'
  | 'pdbDetail'
  | 'secretDetail'
  | 'jobDetail'
  | 'cronJobDetail'
  | 'podDetail'
  | 'nodeDetail'
  | 'deploymentDetail'
  | 'statefulSetDetail'

const props = defineProps<{
  clusterId: number
  clusterName: string
  editorTheme: EditorTheme
  editorThemeEffectiveDark: boolean
  list: any[]
}>()

const emit = defineEmits<{
  (e: 'refresh-list'): void
  (e: 'toggle-editor-theme'): void
  (e: 'open-related-pod', row: any): void
  (e: 'open-topology', payload: { mode: string; namespace?: string; name: string }): void
  (e: 'open-pod-detail', row: any): void
  (e: 'open-job-detail', row: any): void
  (e: 'open-related', payload: { action: string; kind?: string; name: string; namespace?: string }): void
  (e: 'open-yaml', meta: string, loader: YamlLoader, saver?: YamlSaver): void
  (e: 'pod-detail-open-logs', payload: { ns: string; name: string; container?: string }): void
  (e: 'pod-detail-open-exec', payload: { row: any; container?: string }): void
  (e: 'pod-detail-open-related', payload: { action: string; kind?: string; name: string; namespace?: string }): void
  (e: 'pod-detail', row: any): void
  (e: 'pod-log', row: any): void
  (e: 'pod-exec', row: any): void
}>()

const loaded = reactive<Record<LazyKey, boolean>>({
  scaleDialog: false,
  deploymentEdit: false,
  statefulSetEdit: false,
  daemonSetEdit: false,
  yamlDrawer: false,
  workloadRollout: false,
  serviceDetail: false,
  ingressDetail: false,
  pvcDetail: false,
  pvDetail: false,
  ingressClassDetail: false,
  customResourceDefinitionDetail: false,
  apiServiceDetail: false,
  priorityClassDetail: false,
  runtimeClassDetail: false,
  webhookConfigurationDetail: false,
  validatingAdmissionPolicyDetail: false,
  validatingAdmissionPolicyBindingDetail: false,
  configMapDetail: false,
  serviceAccountDetail: false,
  resourceQuotaDetail: false,
  networkPolicyDetail: false,
  rbacRoleDetail: false,
  rbacBindingDetail: false,
  hpaDetail: false,
  pdbDetail: false,
  secretDetail: false,
  jobDetail: false,
  cronJobDetail: false,
  podDetail: false,
  nodeDetail: false,
  deploymentDetail: false,
  statefulSetDetail: false
})

const yamlDrawerRef = ref<InstanceType<typeof K8sYamlDrawerComponent> | null>(null)
const workloadRolloutRef = ref<InstanceType<typeof WorkloadRolloutDrawerComponent> | null>(null)
const scaleDialogRef = ref<InstanceType<typeof K8sScaleDialogComponent> | null>(null)
const deploymentEditDrawerRef = ref<InstanceType<typeof DeploymentEditDrawerComponent> | null>(null)
const statefulSetEditDrawerRef = ref<InstanceType<typeof DeploymentEditDrawerComponent> | null>(null)
const daemonSetEditDrawerRef = ref<InstanceType<typeof DeploymentEditDrawerComponent> | null>(null)

const serviceDetailRef = ref<InstanceType<typeof ServiceDetailDrawerComponent> | null>(null)
const ingressDetailRef = ref<InstanceType<typeof IngressDetailDrawerComponent> | null>(null)
const pvcDetailRef = ref<InstanceType<typeof PVCDetailDrawerComponent> | null>(null)
const pvDetailRef = ref<InstanceType<typeof PVDetailDrawerComponent> | null>(null)
const ingressClassDetailRef = ref<InstanceType<typeof IngressClassDetailDrawerComponent> | null>(null)
const customResourceDefinitionDetailRef = ref<InstanceType<typeof CustomResourceDefinitionDetailDrawerComponent> | null>(null)
const apiServiceDetailRef = ref<InstanceType<typeof APIServiceDetailDrawerComponent> | null>(null)
const priorityClassDetailRef = ref<InstanceType<typeof PriorityClassDetailDrawerComponent> | null>(null)
const runtimeClassDetailRef = ref<InstanceType<typeof RuntimeClassDetailDrawerComponent> | null>(null)
const webhookConfigurationDetailRef = ref<InstanceType<typeof WebhookConfigurationDetailDrawerComponent> | null>(null)
const validatingAdmissionPolicyDetailRef = ref<InstanceType<typeof ValidatingAdmissionPolicyDetailDrawerComponent> | null>(null)
const validatingAdmissionPolicyBindingDetailRef = ref<InstanceType<typeof ValidatingAdmissionPolicyBindingDetailDrawerComponent> | null>(null)
const configMapDetailRef = ref<InstanceType<typeof ConfigMapDetailDrawerComponent> | null>(null)
const serviceAccountDetailRef = ref<InstanceType<typeof ServiceAccountDetailDrawerComponent> | null>(null)
const resourceQuotaDetailRef = ref<InstanceType<typeof ResourceQuotaDetailDrawerComponent> | null>(null)
const networkPolicyDetailRef = ref<InstanceType<typeof NetworkPolicyDetailDrawerComponent> | null>(null)
const rbacRoleDetailRef = ref<InstanceType<typeof RbacRoleDetailDrawerComponent> | null>(null)
const rbacBindingDetailRef = ref<InstanceType<typeof RbacBindingDetailDrawerComponent> | null>(null)
const hpaDetailRef = ref<InstanceType<typeof HPADetailDrawerComponent> | null>(null)
const pdbDetailRef = ref<InstanceType<typeof PdbDetailDrawerComponent> | null>(null)
const secretDetailRef = ref<InstanceType<typeof SecretDetailDrawerComponent> | null>(null)
const jobDetailRef = ref<InstanceType<typeof JobDetailDrawerComponent> | null>(null)
const cronJobDetailRef = ref<InstanceType<typeof CronJobDetailDrawerComponent> | null>(null)
const podDetailDrawerRef = ref<InstanceType<typeof PodDetailDrawerComponent> | null>(null)
const nodeDetailDrawerRef = ref<InstanceType<typeof NodeDetailDrawerComponent> | null>(null)
const deploymentDetailDrawerRef = ref<InstanceType<typeof DeploymentDetailDrawerComponent> | null>(null)
const statefulSetDetailDrawerRef = ref<InstanceType<typeof StatefulSetDetailDrawerComponent> | null>(null)

function runWhenReady<T>(key: LazyKey, getter: () => T | null | undefined, runner: (target: T) => void, attempt = 0) {
  if (!loaded[key]) loaded[key] = true
  void nextTick(() => {
    const target = getter()
    if (target) {
      runner(target)
      return
    }
    if (attempt >= 60) return
    window.setTimeout(() => runWhenReady(key, getter, runner, attempt + 1), 32)
  })
}

function openScale(payload: { kind: string; namespace: string; name: string }, desired: number, available: number) {
  runWhenReady('scaleDialog', () => scaleDialogRef.value, (target) => target.open(payload, desired, available))
}

function forwardOpenYaml(meta: string, loader: YamlLoader, saver?: YamlSaver) {
  emit('open-yaml', meta, loader, saver)
}

function openEditDeployment(row: any) {
  runWhenReady('deploymentEdit', () => deploymentEditDrawerRef.value, (target) => target.open(row))
}

function openEditDaemonSet(row: any) {
  runWhenReady('daemonSetEdit', () => daemonSetEditDrawerRef.value, (target) => target.open(row))
}

function openEditStatefulSet(row: any) {
  runWhenReady('statefulSetEdit', () => statefulSetEditDrawerRef.value, (target) => target.open(row))
}

function openYaml(meta: string, loader: YamlLoader, saver?: YamlSaver) {
  runWhenReady('yamlDrawer', () => yamlDrawerRef.value, (target) => target.open(meta, loader, saver))
}

function openWorkloadRollout(payload: { kind: string; namespace: string; name: string }) {
  runWhenReady('workloadRollout', () => workloadRolloutRef.value, (target) => target.open(payload))
}

function openServiceDetail(row: any) {
  runWhenReady('serviceDetail', () => serviceDetailRef.value, (target) => target.open(row))
}

function openIngressDetail(row: any) {
  runWhenReady('ingressDetail', () => ingressDetailRef.value, (target) => target.open(row))
}

function openIngressClassDetail(row: any) {
  runWhenReady('ingressClassDetail', () => ingressClassDetailRef.value, (target) => target.open(row))
}

function openCustomResourceDefinitionDetail(row: any) {
  runWhenReady('customResourceDefinitionDetail', () => customResourceDefinitionDetailRef.value, (target) => target.open(row))
}

function openAPIServiceDetail(row: any) {
  runWhenReady('apiServiceDetail', () => apiServiceDetailRef.value, (target) => target.open(row))
}

function openPriorityClassDetail(row: any) {
  runWhenReady('priorityClassDetail', () => priorityClassDetailRef.value, (target) => target.open(row))
}

function openRuntimeClassDetail(row: any) {
  runWhenReady('runtimeClassDetail', () => runtimeClassDetailRef.value, (target) => target.open(row))
}

function openValidatingWebhookConfigurationDetail(row: any) {
  runWhenReady('webhookConfigurationDetail', () => webhookConfigurationDetailRef.value, (target) => target.open(row, 'ValidatingWebhookConfiguration'))
}

function openMutatingWebhookConfigurationDetail(row: any) {
  runWhenReady('webhookConfigurationDetail', () => webhookConfigurationDetailRef.value, (target) => target.open(row, 'MutatingWebhookConfiguration'))
}

function openValidatingAdmissionPolicyDetail(row: any) {
  runWhenReady('validatingAdmissionPolicyDetail', () => validatingAdmissionPolicyDetailRef.value, (target) => target.open(row))
}

function openValidatingAdmissionPolicyBindingDetail(row: any) {
  runWhenReady('validatingAdmissionPolicyBindingDetail', () => validatingAdmissionPolicyBindingDetailRef.value, (target) => target.open(row))
}

function openPVCDetail(row: any) {
  runWhenReady('pvcDetail', () => pvcDetailRef.value, (target) => target.open(row))
}

function openPVDetail(row: any) {
  runWhenReady('pvDetail', () => pvDetailRef.value, (target) => target.open(row))
}

function openConfigMapDetail(row: any) {
  runWhenReady('configMapDetail', () => configMapDetailRef.value, (target) => target.open(row))
}

function openServiceAccountDetail(row: any) {
  runWhenReady('serviceAccountDetail', () => serviceAccountDetailRef.value, (target) => target.open(row))
}

function openResourceQuotaDetail(row: any) {
  runWhenReady('resourceQuotaDetail', () => resourceQuotaDetailRef.value, (target) => target.open(row))
}

function openNetworkPolicyDetail(row: any) {
  runWhenReady('networkPolicyDetail', () => networkPolicyDetailRef.value, (target) => target.open(row))
}

function openRoleDetail(row: any) {
  runWhenReady('rbacRoleDetail', () => rbacRoleDetailRef.value, (target) => target.open(row, 'Role'))
}

function openClusterRoleDetail(row: any) {
  runWhenReady('rbacRoleDetail', () => rbacRoleDetailRef.value, (target) => target.open(row, 'ClusterRole'))
}

function openRoleBindingDetail(row: any) {
  runWhenReady('rbacBindingDetail', () => rbacBindingDetailRef.value, (target) => target.open(row, 'RoleBinding'))
}

function openClusterRoleBindingDetail(row: any) {
  runWhenReady('rbacBindingDetail', () => rbacBindingDetailRef.value, (target) => target.open(row, 'ClusterRoleBinding'))
}

function openHPADetail(row: any) {
  runWhenReady('hpaDetail', () => hpaDetailRef.value, (target) => target.open(row))
}

function openPdbDetail(row: any) {
  runWhenReady('pdbDetail', () => pdbDetailRef.value, (target) => target.open(row))
}

function openSecretDetail(row: any) {
  runWhenReady('secretDetail', () => secretDetailRef.value, (target) => target.open(row))
}

function openJobDetail(row: any) {
  runWhenReady('jobDetail', () => jobDetailRef.value, (target) => target.open(row))
}

function openCronJobDetail(row: any) {
  runWhenReady('cronJobDetail', () => cronJobDetailRef.value, (target) => target.open(row))
}

function openPodDetail(row: any) {
  runWhenReady('podDetail', () => podDetailDrawerRef.value, (target) => target.open(row))
}

function openNodeDetail(row: any) {
  runWhenReady('nodeDetail', () => nodeDetailDrawerRef.value, (target) => target.open(row))
}

function openDeploymentDetail(row: any, kind?: string) {
  runWhenReady('deploymentDetail', () => deploymentDetailDrawerRef.value, (target) => target.open(row, kind))
}

function openStatefulSetDetail(row: any) {
  runWhenReady('statefulSetDetail', () => statefulSetDetailDrawerRef.value, (target) => target.open(row))
}

defineExpose({
  openScale,
  openEditDeployment,
  openEditDaemonSet,
  openEditStatefulSet,
  openYaml,
  openWorkloadRollout,
  openServiceDetail,
  openIngressDetail,
  openIngressClassDetail,
  openCustomResourceDefinitionDetail,
  openAPIServiceDetail,
  openPriorityClassDetail,
  openRuntimeClassDetail,
  openValidatingWebhookConfigurationDetail,
  openMutatingWebhookConfigurationDetail,
  openValidatingAdmissionPolicyDetail,
  openValidatingAdmissionPolicyBindingDetail,
  openPVCDetail,
  openPVDetail,
  openConfigMapDetail,
  openServiceAccountDetail,
  openResourceQuotaDetail,
  openNetworkPolicyDetail,
  openRoleDetail,
  openClusterRoleDetail,
  openRoleBindingDetail,
  openClusterRoleBindingDetail,
  openHPADetail,
  openPdbDetail,
  openSecretDetail,
  openJobDetail,
  openCronJobDetail,
  openPodDetail,
  openNodeDetail,
  openDeploymentDetail,
  openStatefulSetDetail
})
</script>