import { ElMessageBox } from 'element-plus'
import type { ComputedRef, Ref } from 'vue'

import * as k8sApi from '@/features/k8s/api/k8s'
import type { WorkloadKind } from '@/features/k8s/pages/ClusterManageView.types'
import { getRowNamespace, getWorkloadAvailable, getWorkloadDesired } from '@/features/k8s/pages/ClusterManageView.utils'
import type { ApiError } from '@/shared/utils/error'
import { notifyError, notifySuccess } from '@/shared/utils/notify'

type ClusterDeleteRunner = (cluster: number, name: string) => Promise<unknown>
type NamespacedDeleteRunner = (cluster: number, namespace: string, name: string) => Promise<unknown>

type DeleteConfirmOptions = {
  title?: string
  boxOptions?: Parameters<typeof ElMessageBox.confirm>[2]
  successMessage?: string
  afterSuccess?: () => Promise<void>
}

export function useClusterManageDeleteActions(options: {
  clusterId: Ref<number>
  workloadKind: ComputedRef<WorkloadKind>
  loadCurrent: () => Promise<void>
  refreshAll: () => Promise<void>
}) {
  function handleDeleteError(error: unknown) {
    if (error === 'cancel') return
    const err = error as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  }

  async function confirmAndDelete(message: string, runner: () => Promise<unknown>, config?: DeleteConfirmOptions) {
    try {
      await ElMessageBox.confirm(message, config?.title ?? '提示', config?.boxOptions ?? { type: 'warning' })
      await runner()
      notifySuccess(config?.successMessage ?? '已删除')
      await (config?.afterSuccess ?? options.loadCurrent)()
    } catch (error) {
      handleDeleteError(error)
    }
  }

  function getClusterName(row: any) {
    const cluster = options.clusterId.value
    if (!cluster) return null
    const name = String(row?.metadata?.name ?? '')
    if (!name) return null
    return { cluster, name }
  }

  function getClusterNsName(row: any) {
    const cluster = options.clusterId.value
    const namespace = getRowNamespace(row)
    if (!cluster || !namespace) return null
    const name = String(row?.metadata?.name ?? '')
    if (!name) return null
    return { cluster, namespace, name }
  }

  function createClusterDelete(label: string, runner: ClusterDeleteRunner, config?: DeleteConfirmOptions) {
    return async (row: any) => {
      const meta = getClusterName(row)
      if (!meta) return
      await confirmAndDelete(`确认删除 ${label} ${meta.name}？`, () => runner(meta.cluster, meta.name), config)
    }
  }

  function createNamespacedDelete(label: string, runner: NamespacedDeleteRunner, config?: DeleteConfirmOptions) {
    return async (row: any) => {
      const meta = getClusterNsName(row)
      if (!meta) return
      await confirmAndDelete(`确认删除 ${label} ${meta.namespace}/${meta.name}？`, () => runner(meta.cluster, meta.namespace, meta.name), config)
    }
  }

  async function deleteWorkloadRow(row: any) {
    const meta = getClusterNsName(row)
    if (!meta) return
    const kind = String(row?.kind ?? options.workloadKind.value)
    const desired = getWorkloadDesired(row)
    const available = getWorkloadAvailable(row)
    const podHint = desired > 0
      ? `\n\nℹ️ 此操作将级联删除其管理的 ${available}/${desired} 个 Pod 和关联的 ReplicaSet。`
      : ''
    await confirmAndDelete(
      `确认删除 ${kind} ${meta.namespace}/${meta.name}？${podHint}`,
      () => k8sApi.deleteWorkload(meta.cluster, { kind, namespace: meta.namespace, name: meta.name }),
      {
        title: '危险操作',
        boxOptions: {
          type: 'error',
          confirmButtonText: '确认删除',
          cancelButtonText: '取消',
          confirmButtonClass: 'el-button--danger'
        }
      }
    )
  }

  async function deleteNamespaceRow(row: any) {
    const meta = getClusterName(row)
    if (!meta) return
    await confirmAndDelete(
      `确认删除 Namespace ${meta.name}？`,
      () => k8sApi.deleteNamespace(meta.cluster, meta.name),
      { afterSuccess: options.refreshAll }
    )
  }

  const deleteHPARow = createNamespacedDelete('HPA', (cluster, namespace, name) => k8sApi.deleteHPA(cluster, namespace, name))
  const deleteServiceRow = createNamespacedDelete('Service', (cluster, namespace, name) => k8sApi.deleteService(cluster, namespace, name))
  const deleteNetworkPolicyRow = createNamespacedDelete('NetworkPolicy', (cluster, namespace, name) => k8sApi.deleteNetworkPolicy(cluster, namespace, name))
  const deleteIngressRow = createNamespacedDelete('Ingress', (cluster, namespace, name) => k8sApi.deleteIngress(cluster, namespace, name))
  const deleteConfigMapRow = createNamespacedDelete('ConfigMap', (cluster, namespace, name) => k8sApi.deleteConfigMap(cluster, namespace, name))
  const deleteSecretRow = createNamespacedDelete('Secret', (cluster, namespace, name) => k8sApi.deleteSecret(cluster, namespace, name))
  const deleteServiceAccountRow = createNamespacedDelete('ServiceAccount', (cluster, namespace, name) => k8sApi.deleteServiceAccount(cluster, namespace, name))
  const deletePdbRow = createNamespacedDelete('PDB', (cluster, namespace, name) => k8sApi.deletePDB(cluster, namespace, name))
  const deleteRoleRow = createNamespacedDelete('Role', (cluster, namespace, name) => k8sApi.deleteRole(cluster, namespace, name))
  const deleteClusterRoleRow = createClusterDelete('ClusterRole', (cluster, name) => k8sApi.deleteClusterRole(cluster, name))
  const deleteRoleBindingRow = createNamespacedDelete('RoleBinding', (cluster, namespace, name) => k8sApi.deleteRoleBinding(cluster, namespace, name))
  const deleteClusterRoleBindingRow = createClusterDelete('ClusterRoleBinding', (cluster, name) => k8sApi.deleteClusterRoleBinding(cluster, name))
  const deletePodRow = createNamespacedDelete('Pod', (cluster, namespace, name) => k8sApi.deletePod(cluster, namespace, name))
  const deleteEndpointsRow = createNamespacedDelete('Endpoints', (cluster, namespace, name) => k8sApi.deleteEndpoints(cluster, namespace, name))
  const deleteEndpointSliceRow = createNamespacedDelete('EndpointSlice', (cluster, namespace, name) => k8sApi.deleteEndpointSlice(cluster, namespace, name))
  const deleteReplicaSetRow = createNamespacedDelete('ReplicaSet', (cluster, namespace, name) => k8sApi.deleteReplicaSet(cluster, namespace, name))
  const deleteIngressClassRow = createClusterDelete('IngressClass', (cluster, name) => k8sApi.deleteIngressClass(cluster, name))
  const deletePVCRow = createNamespacedDelete('PVC', (cluster, namespace, name) => k8sApi.deletePVC(cluster, namespace, name))
  const deletePVRow = createClusterDelete('PV', (cluster, name) => k8sApi.deletePV(cluster, name))
  const deleteStorageClassRow = createClusterDelete('StorageClass', (cluster, name) => k8sApi.deleteStorageClass(cluster, name))
  const deleteCSIDriverRow = createClusterDelete('CSIDriver', (cluster, name) => k8sApi.deleteCSIDriver(cluster, name))
  const deleteCSINodeRow = createClusterDelete('CSINode', (cluster, name) => k8sApi.deleteCSINode(cluster, name))
  const deleteCSIStorageCapacityRow = createNamespacedDelete('CSIStorageCapacity', (cluster, namespace, name) => k8sApi.deleteCSIStorageCapacity(cluster, namespace, name))
  const deleteVolumeSnapshotRow = createNamespacedDelete('VolumeSnapshot', (cluster, namespace, name) => k8sApi.deleteVolumeSnapshot(cluster, namespace, name))
  const deleteVolumeSnapshotClassRow = createClusterDelete('VolumeSnapshotClass', (cluster, name) => k8sApi.deleteVolumeSnapshotClass(cluster, name))
  const deleteVolumeSnapshotContentRow = createClusterDelete('VolumeSnapshotContent', (cluster, name) => k8sApi.deleteVolumeSnapshotContent(cluster, name))
  const deleteResourceQuotaRow = createNamespacedDelete('ResourceQuota', (cluster, namespace, name) => k8sApi.deleteResourceQuota(cluster, namespace, name))
  const deleteLimitRangeRow = createNamespacedDelete('LimitRange', (cluster, namespace, name) => k8sApi.deleteLimitRange(cluster, namespace, name))
  const deleteVolumeAttachmentRow = createClusterDelete('VolumeAttachment', (cluster, name) => k8sApi.deleteVolumeAttachment(cluster, name))
  const deleteLeaseRow = createNamespacedDelete('Lease', (cluster, namespace, name) => k8sApi.deleteLease(cluster, namespace, name))
  const deleteCustomResourceDefinitionRow = createClusterDelete('CRD', (cluster, name) => k8sApi.deleteCustomResourceDefinition(cluster, name))
  const deleteAPIServiceRow = createClusterDelete('APIService', (cluster, name) => k8sApi.deleteAPIService(cluster, name))
  const deletePriorityClassRow = createClusterDelete('PriorityClass', (cluster, name) => k8sApi.deletePriorityClass(cluster, name))
  const deleteRuntimeClassRow = createClusterDelete('RuntimeClass', (cluster, name) => k8sApi.deleteRuntimeClass(cluster, name))
  const deleteValidatingWebhookConfigurationRow = createClusterDelete('ValidatingWebhookConfiguration', (cluster, name) => k8sApi.deleteValidatingWebhookConfiguration(cluster, name))
  const deleteMutatingWebhookConfigurationRow = createClusterDelete('MutatingWebhookConfiguration', (cluster, name) => k8sApi.deleteMutatingWebhookConfiguration(cluster, name))
  const deleteValidatingAdmissionPolicyRow = createClusterDelete('ValidatingAdmissionPolicy', (cluster, name) => k8sApi.deleteValidatingAdmissionPolicy(cluster, name))
  const deleteValidatingAdmissionPolicyBindingRow = createClusterDelete('ValidatingAdmissionPolicyBinding', (cluster, name) => k8sApi.deleteValidatingAdmissionPolicyBinding(cluster, name))
  const deleteJobRow = createNamespacedDelete('Job', (cluster, namespace, name) => k8sApi.deleteJob(cluster, namespace, name))
  const deleteCronJobRow = createNamespacedDelete('CronJob', (cluster, namespace, name) => k8sApi.deleteCronJob(cluster, namespace, name))

  return {
    deleteWorkloadRow,
    deleteHPARow,
    deleteServiceRow,
    deleteNetworkPolicyRow,
    deleteIngressRow,
    deleteConfigMapRow,
    deleteSecretRow,
    deleteServiceAccountRow,
    deletePdbRow,
    deleteRoleRow,
    deleteClusterRoleRow,
    deleteRoleBindingRow,
    deleteClusterRoleBindingRow,
    deletePodRow,
    deleteNamespaceRow,
    deleteEndpointsRow,
    deleteEndpointSliceRow,
    deleteReplicaSetRow,
    deleteIngressClassRow,
    deletePVCRow,
    deletePVRow,
    deleteStorageClassRow,
    deleteCSIDriverRow,
    deleteCSINodeRow,
    deleteCSIStorageCapacityRow,
    deleteVolumeSnapshotRow,
    deleteVolumeSnapshotClassRow,
    deleteVolumeSnapshotContentRow,
    deleteResourceQuotaRow,
    deleteLimitRangeRow,
    deleteVolumeAttachmentRow,
    deleteLeaseRow,
    deleteCustomResourceDefinitionRow,
    deleteAPIServiceRow,
    deletePriorityClassRow,
    deleteRuntimeClassRow,
    deleteValidatingWebhookConfigurationRow,
    deleteMutatingWebhookConfigurationRow,
    deleteValidatingAdmissionPolicyRow,
    deleteValidatingAdmissionPolicyBindingRow,
    deleteJobRow,
    deleteCronJobRow
  }
}