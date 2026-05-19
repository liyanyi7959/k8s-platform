import type { ComputedRef, Ref } from 'vue'

import * as k8sApi from '@/features/k8s/api/k8s'
import type { WorkloadKind } from '@/features/k8s/pages/ClusterManageView.types'
import { getRowNamespace } from '@/features/k8s/pages/ClusterManageView.utils'

type YamlLoader = () => Promise<{ text: string }>
type YamlSaver = (text: string) => Promise<void>
type OpenYaml = (meta: string, loader: YamlLoader, saver?: YamlSaver) => void

export function useClusterManageYamlActions(options: {
  clusterId: Ref<number>
  workloadKind: ComputedRef<WorkloadKind>
  openYaml: OpenYaml
  loadCurrent: () => Promise<void>
}) {
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

  function openClusterYaml(
    row: any,
    loader: (cluster: number, name: string) => Promise<{ text: string }>
  ) {
    const meta = getClusterName(row)
    if (!meta) return
    options.openYaml(`cluster=${meta.cluster}  ${meta.name}`, () => loader(meta.cluster, meta.name))
  }

  function openNamespacedYaml(
    row: any,
    loader: (cluster: number, namespace: string, name: string) => Promise<{ text: string }>
  ) {
    const meta = getClusterNsName(row)
    if (!meta) return
    options.openYaml(`cluster=${meta.cluster}  ${meta.namespace}/${meta.name}`, () => loader(meta.cluster, meta.namespace, meta.name))
  }

  function openClusterEditor(
    row: any,
    title: string,
    loader: (cluster: number, name: string) => Promise<{ text: string }>,
    saver: (cluster: number, name: string, text: string) => Promise<void>
  ) {
    const meta = getClusterName(row)
    if (!meta) return
    options.openYaml(
      `${title}: ${meta.name}`,
      () => loader(meta.cluster, meta.name),
      async (text) => {
        await saver(meta.cluster, meta.name, text)
        await options.loadCurrent()
      }
    )
  }

  function openNamespacedEditor(
    row: any,
    title: string,
    loader: (cluster: number, namespace: string, name: string) => Promise<{ text: string }>,
    saver: (cluster: number, namespace: string, name: string, text: string) => Promise<void>
  ) {
    const meta = getClusterNsName(row)
    if (!meta) return
    options.openYaml(
      `${title}: ${meta.namespace}/${meta.name}`,
      () => loader(meta.cluster, meta.namespace, meta.name),
      async (text) => {
        await saver(meta.cluster, meta.namespace, meta.name, text)
        await options.loadCurrent()
      }
    )
  }

  function openNamespaceYaml(row: any) {
    openClusterYaml({ metadata: { name: String(row?.metadata?.name ?? '') } }, (cluster, name) => k8sApi.getNamespaceYaml(cluster, name))
  }

  function openNodeYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getNodeYaml(cluster, name))
  }

  function openServiceYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getServiceYaml(cluster, namespace, name))
  }

  function openNetworkPolicyYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getNetworkPolicyYaml(cluster, namespace, name))
  }

  function openEndpointsYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getEndpointsYaml(cluster, namespace, name))
  }

  function openEndpointSliceYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getEndpointSliceYaml(cluster, namespace, name))
  }

  function openReplicaSetYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getReplicaSetYaml(cluster, namespace, name))
  }

  function openIngressYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getIngressYaml(cluster, namespace, name))
  }

  function openConfigMapYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getConfigMapYaml(cluster, namespace, name))
  }

  function openSecretYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getSecretYaml(cluster, namespace, name))
  }

  function openServiceAccountYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getServiceAccountYaml(cluster, namespace, name))
  }

  function openEditServiceAccount(row: any) {
    openNamespacedEditor(
      row,
      'Edit ServiceAccount',
      (cluster, namespace, name) => k8sApi.getServiceAccountYaml(cluster, namespace, name),
      async (cluster, namespace, _name, text) => {
        await k8sApi.editServiceAccount(cluster, { namespace, yaml: text })
      }
    )
  }

  function openPdbYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getPDBYaml(cluster, namespace, name))
  }

  function openEditPdb(row: any) {
    openNamespacedEditor(
      row,
      'Edit PDB',
      (cluster, namespace, name) => k8sApi.getPDBYaml(cluster, namespace, name),
      async (cluster, namespace, _name, text) => {
        await k8sApi.editPDB(cluster, { namespace, yaml: text })
      }
    )
  }

  function openRoleYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getRoleYaml(cluster, namespace, name))
  }

  function openClusterRoleYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getClusterRoleYaml(cluster, name))
  }

  function openEditRole(row: any) {
    openNamespacedEditor(
      row,
      'Edit Role',
      (cluster, namespace, name) => k8sApi.getRoleYaml(cluster, namespace, name),
      async (cluster, namespace, _name, text) => {
        await k8sApi.editRole(cluster, { namespace, yaml: text })
      }
    )
  }

  function openEditClusterRole(row: any) {
    openClusterEditor(
      row,
      'Edit ClusterRole',
      (cluster, name) => k8sApi.getClusterRoleYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editClusterRole(cluster, { yaml: text })
      }
    )
  }

  function openRoleBindingYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getRoleBindingYaml(cluster, namespace, name))
  }

  function openEditRoleBinding(row: any) {
    openNamespacedEditor(
      row,
      'Edit RoleBinding',
      (cluster, namespace, name) => k8sApi.getRoleBindingYaml(cluster, namespace, name),
      async (cluster, namespace, _name, text) => {
        await k8sApi.editRoleBinding(cluster, { namespace, yaml: text })
      }
    )
  }

  function openClusterRoleBindingYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getClusterRoleBindingYaml(cluster, name))
  }

  function openEditClusterRoleBinding(row: any) {
    openClusterEditor(
      row,
      'Edit ClusterRoleBinding',
      (cluster, name) => k8sApi.getClusterRoleBindingYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editClusterRoleBinding(cluster, { yaml: text })
      }
    )
  }

  function openHPAYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getHPAYaml(cluster, namespace, name))
  }

  function openEditHPA(row: any) {
    openNamespacedEditor(
      row,
      'Edit HPA',
      (cluster, namespace, name) => k8sApi.getHPAYaml(cluster, namespace, name),
      async (cluster, namespace, _name, text) => {
        await k8sApi.editHPA(cluster, { namespace, yaml: text })
      }
    )
  }

  function openWorkloadYaml(row: any) {
    const meta = getClusterNsName(row)
    if (!meta) return
    const kind = String(row?.kind ?? options.workloadKind.value)
    options.openYaml(
      `Edit ${kind}: ${meta.namespace}/${meta.name}`,
      () => k8sApi.getWorkloadYaml(meta.cluster, { kind, namespace: meta.namespace, name: meta.name }),
      async (text) => {
        await k8sApi.editWorkloadYaml(meta.cluster, { kind, namespace: meta.namespace, yaml: text })
        await options.loadCurrent()
      }
    )
  }

  function openPodYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getPodYaml(cluster, namespace, name))
  }

  function openIngressClassYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getIngressClassYaml(cluster, name))
  }

  function openPVCYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getPVCYaml(cluster, namespace, name))
  }

  function openPVYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getPVYaml(cluster, name))
  }

  function openStorageClassYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getStorageClassYaml(cluster, name))
  }

  function openCSIDriverYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getCSIDriverYaml(cluster, name))
  }

  function openCSINodeYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getCSINodeYaml(cluster, name))
  }

  function openCSIStorageCapacityYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getCSIStorageCapacityYaml(cluster, namespace, name))
  }

  function openVolumeAttachmentYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getVolumeAttachmentYaml(cluster, name))
  }

  function openVolumeSnapshotYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getVolumeSnapshotYaml(cluster, namespace, name))
  }

  function openVolumeSnapshotClassYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getVolumeSnapshotClassYaml(cluster, name))
  }

  function openVolumeSnapshotContentYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getVolumeSnapshotContentYaml(cluster, name))
  }

  function openResourceQuotaYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getResourceQuotaYaml(cluster, namespace, name))
  }

  function openEditResourceQuota(row: any) {
    openNamespacedEditor(
      row,
      'Edit ResourceQuota',
      (cluster, namespace, name) => k8sApi.getResourceQuotaYaml(cluster, namespace, name),
      async (cluster, _namespace, _name, text) => {
        await k8sApi.editResourceQuota(cluster, { yaml: text })
      }
    )
  }

  function openLimitRangeYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getLimitRangeYaml(cluster, namespace, name))
  }

  function openEditLimitRange(row: any) {
    openNamespacedEditor(
      row,
      'Edit LimitRange',
      (cluster, namespace, name) => k8sApi.getLimitRangeYaml(cluster, namespace, name),
      async (cluster, _namespace, _name, text) => {
        await k8sApi.editLimitRange(cluster, { yaml: text })
      }
    )
  }

  function openLeaseYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getLeaseYaml(cluster, namespace, name))
  }

  function openEditLease(row: any) {
    openNamespacedEditor(
      row,
      'Edit Lease',
      (cluster, namespace, name) => k8sApi.getLeaseYaml(cluster, namespace, name),
      async (cluster, namespace, _name, text) => {
        await k8sApi.editLease(cluster, { namespace, yaml: text })
      }
    )
  }

  function openCustomResourceDefinitionYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getCustomResourceDefinitionYaml(cluster, name))
  }

  function openEditCustomResourceDefinition(row: any) {
    openClusterEditor(
      row,
      'Edit CustomResourceDefinition',
      (cluster, name) => k8sApi.getCustomResourceDefinitionYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editCustomResourceDefinition(cluster, { yaml: text })
      }
    )
  }

  function openAPIServiceYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getAPIServiceYaml(cluster, name))
  }

  function openEditAPIService(row: any) {
    openClusterEditor(
      row,
      'Edit APIService',
      (cluster, name) => k8sApi.getAPIServiceYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editAPIService(cluster, { yaml: text })
      }
    )
  }

  function openPriorityClassYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getPriorityClassYaml(cluster, name))
  }

  function openRuntimeClassYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getRuntimeClassYaml(cluster, name))
  }

  function openValidatingWebhookConfigurationYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getValidatingWebhookConfigurationYaml(cluster, name))
  }

  function openMutatingWebhookConfigurationYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getMutatingWebhookConfigurationYaml(cluster, name))
  }

  function openValidatingAdmissionPolicyYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getValidatingAdmissionPolicyYaml(cluster, name))
  }

  function openValidatingAdmissionPolicyBindingYaml(row: any) {
    openClusterYaml(row, (cluster, name) => k8sApi.getValidatingAdmissionPolicyBindingYaml(cluster, name))
  }

  function openEditStorageClass(row: any) {
    openClusterEditor(
      row,
      'Edit StorageClass',
      (cluster, name) => k8sApi.getStorageClassYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editStorageClass(cluster, { yaml: text })
      }
    )
  }

  function openEditCSIDriver(row: any) {
    openClusterEditor(
      row,
      'Edit CSIDriver',
      (cluster, name) => k8sApi.getCSIDriverYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editCSIDriver(cluster, { yaml: text })
      }
    )
  }

  function openEditCSINode(row: any) {
    openClusterEditor(
      row,
      'Edit CSINode',
      (cluster, name) => k8sApi.getCSINodeYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editCSINode(cluster, { yaml: text })
      }
    )
  }

  function openEditCSIStorageCapacity(row: any) {
    openNamespacedEditor(
      row,
      'Edit CSIStorageCapacity',
      (cluster, namespace, name) => k8sApi.getCSIStorageCapacityYaml(cluster, namespace, name),
      async (cluster, namespace, _name, text) => {
        await k8sApi.editCSIStorageCapacity(cluster, { namespace, yaml: text })
      }
    )
  }

  function openEditVolumeSnapshot(row: any) {
    openNamespacedEditor(
      row,
      'Edit VolumeSnapshot',
      (cluster, namespace, name) => k8sApi.getVolumeSnapshotYaml(cluster, namespace, name),
      async (cluster, namespace, _name, text) => {
        await k8sApi.editVolumeSnapshot(cluster, { namespace, yaml: text })
      }
    )
  }

  function openEditVolumeSnapshotClass(row: any) {
    openClusterEditor(
      row,
      'Edit VolumeSnapshotClass',
      (cluster, name) => k8sApi.getVolumeSnapshotClassYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editVolumeSnapshotClass(cluster, { yaml: text })
      }
    )
  }

  function openEditVolumeSnapshotContent(row: any) {
    openClusterEditor(
      row,
      'Edit VolumeSnapshotContent',
      (cluster, name) => k8sApi.getVolumeSnapshotContentYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editVolumeSnapshotContent(cluster, { yaml: text })
      }
    )
  }

  function openEditNetworkPolicy(row: any) {
    openNamespacedEditor(
      row,
      'Edit NetworkPolicy',
      (cluster, namespace, name) => k8sApi.getNetworkPolicyYaml(cluster, namespace, name),
      async (cluster, _namespace, _name, text) => {
        await k8sApi.editNetworkPolicy(cluster, { yaml: text })
      }
    )
  }

  function openEditEndpoints(row: any) {
    openNamespacedEditor(
      row,
      'Edit Endpoints',
      (cluster, namespace, name) => k8sApi.getEndpointsYaml(cluster, namespace, name),
      async (cluster, namespace, _name, text) => {
        await k8sApi.editEndpoints(cluster, { namespace, yaml: text })
      }
    )
  }

  function openEditEndpointSlice(row: any) {
    openNamespacedEditor(
      row,
      'Edit EndpointSlice',
      (cluster, namespace, name) => k8sApi.getEndpointSliceYaml(cluster, namespace, name),
      async (cluster, namespace, _name, text) => {
        await k8sApi.editEndpointSlice(cluster, { namespace, yaml: text })
      }
    )
  }

  function openEditReplicaSet(row: any) {
    openNamespacedEditor(
      row,
      'Edit ReplicaSet',
      (cluster, namespace, name) => k8sApi.getReplicaSetYaml(cluster, namespace, name),
      async (cluster, namespace, _name, text) => {
        await k8sApi.editReplicaSet(cluster, { namespace, yaml: text })
      }
    )
  }

  function openEditPriorityClass(row: any) {
    openClusterEditor(
      row,
      'Edit PriorityClass',
      (cluster, name) => k8sApi.getPriorityClassYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editPriorityClass(cluster, { yaml: text })
      }
    )
  }

  function openEditRuntimeClass(row: any) {
    openClusterEditor(
      row,
      'Edit RuntimeClass',
      (cluster, name) => k8sApi.getRuntimeClassYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editRuntimeClass(cluster, { yaml: text })
      }
    )
  }

  function openEditValidatingWebhookConfiguration(row: any) {
    openClusterEditor(
      row,
      'Edit ValidatingWebhookConfiguration',
      (cluster, name) => k8sApi.getValidatingWebhookConfigurationYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editValidatingWebhookConfiguration(cluster, { yaml: text })
      }
    )
  }

  function openEditMutatingWebhookConfiguration(row: any) {
    openClusterEditor(
      row,
      'Edit MutatingWebhookConfiguration',
      (cluster, name) => k8sApi.getMutatingWebhookConfigurationYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editMutatingWebhookConfiguration(cluster, { yaml: text })
      }
    )
  }

  function openEditValidatingAdmissionPolicy(row: any) {
    openClusterEditor(
      row,
      'Edit ValidatingAdmissionPolicy',
      (cluster, name) => k8sApi.getValidatingAdmissionPolicyYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editValidatingAdmissionPolicy(cluster, { yaml: text })
      }
    )
  }

  function openEditValidatingAdmissionPolicyBinding(row: any) {
    openClusterEditor(
      row,
      'Edit ValidatingAdmissionPolicyBinding',
      (cluster, name) => k8sApi.getValidatingAdmissionPolicyBindingYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editValidatingAdmissionPolicyBinding(cluster, { yaml: text })
      }
    )
  }

  function openEditVolumeAttachment(row: any) {
    openClusterEditor(
      row,
      'Edit VolumeAttachment',
      (cluster, name) => k8sApi.getVolumeAttachmentYaml(cluster, name),
      async (cluster, _name, text) => {
        await k8sApi.editVolumeAttachment(cluster, { yaml: text })
      }
    )
  }

  function openJobYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getJobYaml(cluster, namespace, name))
  }

  function openCronJobYaml(row: any) {
    openNamespacedYaml(row, (cluster, namespace, name) => k8sApi.getCronJobYaml(cluster, namespace, name))
  }

  return {
    openNamespaceYaml,
    openNodeYaml,
    openServiceYaml,
    openNetworkPolicyYaml,
    openEndpointsYaml,
    openEndpointSliceYaml,
    openReplicaSetYaml,
    openIngressYaml,
    openConfigMapYaml,
    openSecretYaml,
    openServiceAccountYaml,
    openEditServiceAccount,
    openPdbYaml,
    openEditPdb,
    openRoleYaml,
    openClusterRoleYaml,
    openEditRole,
    openEditClusterRole,
    openRoleBindingYaml,
    openEditRoleBinding,
    openClusterRoleBindingYaml,
    openEditClusterRoleBinding,
    openHPAYaml,
    openEditHPA,
    openWorkloadYaml,
    openPodYaml,
    openIngressClassYaml,
    openPVCYaml,
    openPVYaml,
    openStorageClassYaml,
    openCSIDriverYaml,
    openCSINodeYaml,
    openCSIStorageCapacityYaml,
    openVolumeAttachmentYaml,
    openVolumeSnapshotYaml,
    openVolumeSnapshotClassYaml,
    openVolumeSnapshotContentYaml,
    openResourceQuotaYaml,
    openEditResourceQuota,
    openLimitRangeYaml,
    openEditLimitRange,
    openLeaseYaml,
    openEditLease,
    openCustomResourceDefinitionYaml,
    openEditCustomResourceDefinition,
    openAPIServiceYaml,
    openEditAPIService,
    openPriorityClassYaml,
    openRuntimeClassYaml,
    openValidatingWebhookConfigurationYaml,
    openMutatingWebhookConfigurationYaml,
    openValidatingAdmissionPolicyYaml,
    openValidatingAdmissionPolicyBindingYaml,
    openEditStorageClass,
    openEditCSIDriver,
    openEditCSINode,
    openEditCSIStorageCapacity,
    openEditVolumeSnapshot,
    openEditVolumeSnapshotClass,
    openEditVolumeSnapshotContent,
    openEditNetworkPolicy,
    openEditEndpoints,
    openEditEndpointSlice,
    openEditReplicaSet,
    openEditPriorityClass,
    openEditRuntimeClass,
    openEditValidatingWebhookConfiguration,
    openEditMutatingWebhookConfiguration,
    openEditValidatingAdmissionPolicy,
    openEditValidatingAdmissionPolicyBinding,
    openEditVolumeAttachment,
    openJobYaml,
    openCronJobYaml
  }
}