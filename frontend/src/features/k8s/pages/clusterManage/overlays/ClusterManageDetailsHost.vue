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
import type ConfigMapDetailDrawerComponent from '../details/ConfigMapDetailDrawer.vue'
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
const ConfigMapDetailDrawer = defineAsyncComponent(() => import('../details/ConfigMapDetailDrawer.vue'))
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
  | 'configMapDetail'
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
  configMapDetail: false,
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
const configMapDetailRef = ref<InstanceType<typeof ConfigMapDetailDrawerComponent> | null>(null)
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

function openPVCDetail(row: any) {
  runWhenReady('pvcDetail', () => pvcDetailRef.value, (target) => target.open(row))
}

function openPVDetail(row: any) {
  runWhenReady('pvDetail', () => pvDetailRef.value, (target) => target.open(row))
}

function openConfigMapDetail(row: any) {
  runWhenReady('configMapDetail', () => configMapDetailRef.value, (target) => target.open(row))
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
  openPVCDetail,
  openPVDetail,
  openConfigMapDetail,
  openSecretDetail,
  openJobDetail,
  openCronJobDetail,
  openPodDetail,
  openNodeDetail,
  openDeploymentDetail,
  openStatefulSetDetail
})
</script>