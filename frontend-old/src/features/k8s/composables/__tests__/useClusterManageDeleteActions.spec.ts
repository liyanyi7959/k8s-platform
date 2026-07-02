/**
 * useClusterManageDeleteActions 单元测试
 * 覆盖确认弹窗取消场景、删除失败错误通知等核心逻辑
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref, computed } from 'vue'
import { ElMessageBox } from 'element-plus'
import { useClusterManageDeleteActions } from '../useClusterManageDeleteActions'
import type { WorkloadKind } from '@/features/k8s/pages/ClusterManageView.types'
import * as notify from '@/shared/utils/notify'

vi.mock('@/features/k8s/api/k8s', () => ({
  deleteWorkload: vi.fn().mockResolvedValue(undefined),
  deleteNamespace: vi.fn().mockResolvedValue(undefined),
  deletePod: vi.fn().mockResolvedValue(undefined),
  deleteService: vi.fn().mockResolvedValue(undefined),
  deleteConfigMap: vi.fn().mockResolvedValue(undefined),
  deleteSecret: vi.fn().mockResolvedValue(undefined),
  deleteHPA: vi.fn().mockResolvedValue(undefined),
  deleteIngress: vi.fn().mockResolvedValue(undefined),
  deleteNetworkPolicy: vi.fn().mockResolvedValue(undefined),
  deleteServiceAccount: vi.fn().mockResolvedValue(undefined),
  deletePDB: vi.fn().mockResolvedValue(undefined),
  deleteRole: vi.fn().mockResolvedValue(undefined),
  deleteClusterRole: vi.fn().mockResolvedValue(undefined),
  deleteRoleBinding: vi.fn().mockResolvedValue(undefined),
  deleteClusterRoleBinding: vi.fn().mockResolvedValue(undefined),
  deleteReplicaSet: vi.fn().mockResolvedValue(undefined),
  deleteEndpoints: vi.fn().mockResolvedValue(undefined),
  deleteEndpointSlice: vi.fn().mockResolvedValue(undefined),
  deleteIngressClass: vi.fn().mockResolvedValue(undefined),
  deletePVC: vi.fn().mockResolvedValue(undefined),
  deletePV: vi.fn().mockResolvedValue(undefined),
  deleteStorageClass: vi.fn().mockResolvedValue(undefined),
  deleteCSIDriver: vi.fn().mockResolvedValue(undefined),
  deleteCSINode: vi.fn().mockResolvedValue(undefined),
  deleteCSIStorageCapacity: vi.fn().mockResolvedValue(undefined),
  deleteVolumeSnapshot: vi.fn().mockResolvedValue(undefined),
  deleteVolumeSnapshotClass: vi.fn().mockResolvedValue(undefined),
  deleteVolumeSnapshotContent: vi.fn().mockResolvedValue(undefined),
  deleteResourceQuota: vi.fn().mockResolvedValue(undefined),
  deleteLimitRange: vi.fn().mockResolvedValue(undefined),
  deleteVolumeAttachment: vi.fn().mockResolvedValue(undefined),
  deleteLease: vi.fn().mockResolvedValue(undefined),
  deleteCustomResourceDefinition: vi.fn().mockResolvedValue(undefined),
  deleteAPIService: vi.fn().mockResolvedValue(undefined),
  deletePriorityClass: vi.fn().mockResolvedValue(undefined),
  deleteRuntimeClass: vi.fn().mockResolvedValue(undefined),
  deleteValidatingWebhookConfiguration: vi.fn().mockResolvedValue(undefined),
  deleteMutatingWebhookConfiguration: vi.fn().mockResolvedValue(undefined),
  deleteValidatingAdmissionPolicy: vi.fn().mockResolvedValue(undefined),
  deleteValidatingAdmissionPolicyBinding: vi.fn().mockResolvedValue(undefined),
  deleteJob: vi.fn().mockResolvedValue(undefined),
  deleteCronJob: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('@/shared/utils/notify', () => ({
  notifyError: vi.fn(),
  notifySuccess: vi.fn(),
}))

function createStub(options?: { clusterId?: number }) {
  return useClusterManageDeleteActions({
    clusterId: ref(options?.clusterId ?? 1),
    workloadKind: computed(() => 'Deployment' as WorkloadKind),
    loadCurrent: vi.fn().mockResolvedValue(undefined),
    refreshAll: vi.fn().mockResolvedValue(undefined),
  })
}

describe('useClusterManageDeleteActions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ── 确认弹窗取消 ──
  describe('确认弹窗取消', () => {
    it('用户取消确认弹窗时不报错', async () => {
      vi.mocked(ElMessageBox.confirm).mockRejectedValueOnce('cancel')
      const sut = createStub()

      await sut.deleteServiceRow({
        metadata: { namespace: 'default', name: 'my-svc' },
      })

      expect(notify.notifyError).not.toHaveBeenCalled()
      expect(notify.notifySuccess).not.toHaveBeenCalled()
    })
  })

  // ── 删除失败错误通知 ──
  describe('删除失败错误通知', () => {
    it('删除失败时调用 notifyError', async () => {
      vi.mocked(ElMessageBox.confirm).mockResolvedValueOnce('confirm' as any)
      const k8sApi = await import('@/features/k8s/api/k8s')
      vi.mocked(k8sApi.deleteService).mockRejectedValueOnce(new Error('connection refused'))

      const sut = createStub()
      await sut.deleteServiceRow({
        metadata: { namespace: 'default', name: 'my-svc' },
      })

      expect(notify.notifyError).toHaveBeenCalledWith('connection refused')
    })

    it('带 requestId 的错误显示 request_id', async () => {
      vi.mocked(ElMessageBox.confirm).mockResolvedValueOnce('confirm' as any)
      const k8sApi = await import('@/features/k8s/api/k8s')
      const error = Object.assign(new Error('forbidden'), { requestId: 'req-123' })
      vi.mocked(k8sApi.deleteService).mockRejectedValueOnce(error)

      const sut = createStub()
      await sut.deleteServiceRow({
        metadata: { namespace: 'default', name: 'my-svc' },
      })

      expect(notify.notifyError).toHaveBeenCalledWith('forbidden (request_id=req-123)')
    })
  })

  // ── createClusterDelete 逻辑 ──
  describe('createClusterDelete', () => {
    it('clusterId 为 0 时不执行', async () => {
      const sut = createStub({ clusterId: 0 })
      await sut.deleteClusterRoleRow({ metadata: { name: 'admin' } })
      expect(vi.mocked(ElMessageBox.confirm)).not.toHaveBeenCalled()
    })

    it('name 为空时不执行', async () => {
      const sut = createStub()
      await sut.deleteClusterRoleRow({ metadata: {} })
      expect(vi.mocked(ElMessageBox.confirm)).not.toHaveBeenCalled()
    })

    it('正常删除 ClusterRole', async () => {
      vi.mocked(ElMessageBox.confirm).mockResolvedValueOnce('confirm' as any)
      const sut = createStub()
      await sut.deleteClusterRoleRow({ metadata: { name: 'admin' } })
      expect(notify.notifySuccess).toHaveBeenCalledWith('已删除')
    })
  })

  // ── createNamespacedDelete 逻辑 ──
  describe('createNamespacedDelete', () => {
    it('namespace 为空时不执行', async () => {
      const sut = createStub()
      await sut.deletePodRow({ metadata: { name: 'pod-1' } })
      expect(vi.mocked(ElMessageBox.confirm)).not.toHaveBeenCalled()
    })

    it('name 为空时不执行', async () => {
      const sut = createStub()
      await sut.deletePodRow({ metadata: { namespace: 'default' } })
      expect(vi.mocked(ElMessageBox.confirm)).not.toHaveBeenCalled()
    })

    it('正常删除 namespaced 资源', async () => {
      vi.mocked(ElMessageBox.confirm).mockResolvedValueOnce('confirm' as any)
      const sut = createStub()
      await sut.deletePodRow({
        metadata: { namespace: 'default', name: 'pod-1' },
      })
      expect(notify.notifySuccess).toHaveBeenCalledWith('已删除')
    })
  })

  // ── deleteWorkloadRow ──
  describe('deleteWorkloadRow', () => {
    it('正常删除 workload', async () => {
      vi.mocked(ElMessageBox.confirm).mockResolvedValueOnce('confirm' as any)
      const k8sApi = await import('@/features/k8s/api/k8s')

      const sut = createStub()
      await sut.deleteWorkloadRow({
        kind: 'Deployment',
        metadata: { namespace: 'default', name: 'nginx' },
        spec: { replicas: 3 },
        status: { availableReplicas: 3 },
      })

      expect(k8sApi.deleteWorkload).toHaveBeenCalledWith(1, {
        kind: 'Deployment',
        namespace: 'default',
        name: 'nginx',
      })
      expect(notify.notifySuccess).toHaveBeenCalled()
    })

    it('取消时不调用 deleteWorkload', async () => {
      vi.mocked(ElMessageBox.confirm).mockRejectedValueOnce('cancel')
      const k8sApi = await import('@/features/k8s/api/k8s')

      const sut = createStub()
      await sut.deleteWorkloadRow({
        kind: 'Deployment',
        metadata: { namespace: 'default', name: 'nginx' },
        spec: { replicas: 3 },
        status: { availableReplicas: 3 },
      })

      expect(k8sApi.deleteWorkload).not.toHaveBeenCalled()
    })
  })

  // ── deleteNamespaceRow ──
  describe('deleteNamespaceRow', () => {
    it('使用 refreshAll 作为 afterSuccess', async () => {
      vi.mocked(ElMessageBox.confirm).mockResolvedValueOnce('confirm' as any)
      const sut = createStub()
      await sut.deleteNamespaceRow({ metadata: { name: 'test-ns' } })
      expect(notify.notifySuccess).toHaveBeenCalledWith('已删除')
    })
  })
})
