import { computed, ref } from 'vue'
import type { ClusterItem } from '@/features/clusters/api/clusters'

export interface ClusterShortcut {
  id: number
  name: string
  status?: ClusterItem['status']
  k8sVersion?: string
  nodeCount?: number
  updatedAt: number
}

const STORAGE_KEY = 'k8s-platform:cluster-shortcuts'
const CHANGE_EVENT = 'k8s-platform:cluster-shortcuts-changed'
const MAX_SHORTCUTS = 12

const shortcutsState = ref<ClusterShortcut[]>([])
let initialized = false

function isBrowser() {
  return typeof window !== 'undefined' && typeof localStorage !== 'undefined'
}

function normalizeShortcut(raw: unknown): ClusterShortcut | null {
  const item = raw as Partial<ClusterShortcut> | null
  const id = Number(item?.id)
  const name = String(item?.name ?? '').trim()
  if (!Number.isFinite(id) || id <= 0 || !name) return null
  return {
    id,
    name,
    status: item?.status,
    k8sVersion: item?.k8sVersion,
    nodeCount: Number.isFinite(Number(item?.nodeCount)) ? Number(item?.nodeCount) : undefined,
    updatedAt: Number.isFinite(Number(item?.updatedAt)) ? Number(item?.updatedAt) : Date.now()
  }
}

function loadShortcuts() {
  if (!isBrowser()) return
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    if (!Array.isArray(parsed)) {
      shortcutsState.value = []
      return
    }
    const seen = new Set<number>()
    shortcutsState.value = parsed
      .map(normalizeShortcut)
      .filter((item): item is ClusterShortcut => {
        if (!item || seen.has(item.id)) return false
        seen.add(item.id)
        return true
      })
      .slice(0, MAX_SHORTCUTS)
  } catch {
    shortcutsState.value = []
  }
}

function persistShortcuts() {
  if (!isBrowser()) return
  localStorage.setItem(STORAGE_KEY, JSON.stringify(shortcutsState.value))
  window.dispatchEvent(new CustomEvent(CHANGE_EVENT))
}

function toShortcut(cluster: ClusterItem): ClusterShortcut {
  return {
    id: Number(cluster.id),
    name: String(cluster.name ?? '').trim() || `cluster-${cluster.id}`,
    status: cluster.status,
    k8sVersion: cluster.k8s_version,
    nodeCount: cluster.node_count,
    updatedAt: Date.now()
  }
}

function sameShortcut(a: ClusterShortcut, b: ClusterShortcut) {
  return (
    a.id === b.id &&
    a.name === b.name &&
    a.status === b.status &&
    a.k8sVersion === b.k8sVersion &&
    a.nodeCount === b.nodeCount
  )
}

function init() {
  if (initialized) return
  initialized = true
  loadShortcuts()
  if (!isBrowser()) return
  window.addEventListener('storage', (event) => {
    if (event.key === STORAGE_KEY) loadShortcuts()
  })
}

export function getClusterUnavailableMessage(status?: ClusterItem['status']): string | null {
  if (!status || status === 'active') return null
  if (status === 'disabled') return '集群不可用：已禁用'
  if (status === 'degraded') return '集群异常：健康检查失败'
  if (status === 'creating') return '集群正在创建中，暂不可进入'
  if (status === 'deleting') return '集群正在删除中，暂不可进入'
  return '集群不可用'
}

export function useClusterShortcuts() {
  init()

  const shortcuts = computed(() => shortcutsState.value)

  function isPinned(id: number | string) {
    const target = Number(id)
    return shortcutsState.value.some((item) => item.id === target)
  }

  function pinCluster(cluster: ClusterItem) {
    const next = toShortcut(cluster)
    const rest = shortcutsState.value.filter((item) => item.id !== next.id)
    shortcutsState.value = [next, ...rest].slice(0, MAX_SHORTCUTS)
    persistShortcuts()
  }

  function unpinCluster(id: number | string) {
    const target = Number(id)
    const next = shortcutsState.value.filter((item) => item.id !== target)
    if (next.length === shortcutsState.value.length) return
    shortcutsState.value = next
    persistShortcuts()
  }

  function toggleCluster(cluster: ClusterItem) {
    if (isPinned(cluster.id)) {
      unpinCluster(cluster.id)
      return false
    }
    pinCluster(cluster)
    return true
  }

  function syncShortcutsFromClusters(clusters: ClusterItem[]) {
    if (!clusters.length || !shortcutsState.value.length) return
    const byId = new Map(clusters.map((cluster) => [Number(cluster.id), toShortcut(cluster)]))
    let changed = false
    const next = shortcutsState.value.map((item) => {
      const latest = byId.get(item.id)
      if (!latest) return item
      const merged = { ...item, ...latest, updatedAt: item.updatedAt }
      if (!sameShortcut(item, merged)) changed = true
      return merged
    })
    if (!changed) return
    shortcutsState.value = next
    persistShortcuts()
  }

  return {
    shortcuts,
    isPinned,
    pinCluster,
    unpinCluster,
    toggleCluster,
    syncShortcutsFromClusters,
    getClusterUnavailableMessage
  }
}
