import { computed, ref, type ComputedRef } from 'vue'
import * as k8sApi from '@/features/k8s/api/k8s'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'

export type K8sScaleTarget = { kind: string; namespace: string; name: string }

export function useK8sScaleDialog(opts: {
  clusterId: ComputedRef<number | undefined>
  clusterName?: ComputedRef<string>
  onScaled?: () => void | Promise<void>
}) {
  const visible = ref(false)
  const scaling = ref(false)
  const target = ref<K8sScaleTarget | null>(null)
  const replicas = ref(1)

  const meta = computed(() => {
    const cid = opts.clusterId.value
    const t = target.value
    if (!cid || !t) return ''
    const cn = String(opts.clusterName?.value ?? '').trim()
    const clusterText = cn ? cn : String(cid)
    return `cluster=${clusterText}  ${t.kind}  ${t.namespace}/${t.name}`
  })

  function open(nextTarget: K8sScaleTarget, nextReplicas: number) {
    if (!opts.clusterId.value) return
    target.value = nextTarget
    replicas.value = Math.max(0, Number(nextReplicas) || 0)
    visible.value = true
  }

  function close() {
    visible.value = false
  }

  async function submit() {
    const cid = opts.clusterId.value
    if (!cid || !target.value) return
    scaling.value = true
    try {
      await k8sApi.scaleWorkload(cid, {
        kind: target.value.kind,
        namespace: target.value.namespace,
        name: target.value.name,
        replicas: replicas.value
      })
      notifySuccess('已提交伸缩')
      visible.value = false
      await opts.onScaled?.()
    } catch (e) {
      const err = e as ApiError
      notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
    } finally {
      scaling.value = false
    }
  }

  return { visible, scaling, target, replicas, meta, open, close, submit }
}

