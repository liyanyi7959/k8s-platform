import { createRouter, createWebHistory } from 'vue-router'

import { routes } from './routes'
import { useUserStore } from '@/app/store/user'
import { notifyError } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import * as clustersApi from '@/features/clusters/api/clusters'
import { clearToken, getSessionId, getToken, isTokenExpired } from '@/shared/utils/auth'

export const router = createRouter({
  history: createWebHistory(),
  routes
})

let lastAuthExpiredHandledAt = 0

function handleAuthExpired(reason = '未登录或登录已过期') {
  const now = Date.now()
  if (now - lastAuthExpiredHandledAt < 800) return
  lastAuthExpiredHandledAt = now

  const userStore = useUserStore()
  userStore.logoutLocal()

  const cur = router.currentRoute.value
  if (cur.path !== '/login') {
    notifyError(reason)
    router.replace({ path: '/login', query: { redirect: cur.fullPath } }).catch(() => {})
  }
}

if (typeof window !== 'undefined') {
  window.addEventListener('auth:expired', () => handleAuthExpired('登录已过期，请重新登录'))
  window.addEventListener('storage', (e: StorageEvent) => {
    if (e.key !== 'k8s_platform_session_id') return
    const userStore = useUserStore()
    if (!userStore.isAuthed) return
    const sid = getSessionId()
    if (sid && userStore.sessionId && sid !== userStore.sessionId) {
      handleAuthExpired('账号已在其他窗口登录，本窗口已退出')
    }
  })

  window.setInterval(() => {
    const token = getToken()
    if (!token) return
    if (isTokenExpired()) {
      clearToken()
      handleAuthExpired('登录已过期，请重新登录')
    }
  }, 10_000)
}

let cachedClustersTotal: number | null = null
let cachedClustersAt = 0

async function getClustersTotal(): Promise<number> {
  const now = Date.now()
  if (cachedClustersTotal !== null && now - cachedClustersAt < 10_000) return cachedClustersTotal
  const data = await clustersApi.listClusters({ page: 1, page_size: 1 })
  cachedClustersTotal = data.total ?? data.list.length
  cachedClustersAt = now
  return cachedClustersTotal
}

function hasPermission(perms: string[], perm?: string | string[]): boolean {
  if (!perm) return true
  if (Array.isArray(perm)) return perm.some((p) => perms.includes(p))
  return perms.includes(perm)
}

router.beforeEach(async (to) => {
  const userStore = useUserStore()

  if (to.path === '/login') {
    if (userStore.isAuthed) {
      if (isTokenExpired()) {
        clearToken()
        userStore.logoutLocal()
        return true
      }
      await userStore.fetchMe().catch(() => {})
      return { path: '/' }
    }
    return true
  }

  if (to.meta.requiresAuth === false) {
    return true
  }

  if (userStore.isAuthed && isTokenExpired()) {
    clearToken()
    userStore.logoutLocal()
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  if (!userStore.isAuthed) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  if (!userStore.me && !userStore.loadingMe) {
    try {
      await userStore.fetchMe()
    } catch (e) {
      const err = e as ApiError
      notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
      return { path: '/login', query: { redirect: to.fullPath } }
    }
  }

  const perms = userStore.permissions
  if (!hasPermission(perms, to.meta.perm)) {
    notifyError('无权限访问')
    return { path: '/' }
  }

  if (to.path.startsWith('/k8s/')) {
    const clusterId = Number(to.params.clusterId)
    if (Number.isFinite(clusterId) && clusterId > 0) {
      return true
    }
    try {
      const total = await getClustersTotal()
      if (total <= 0) {
        notifyError('请先导入集群')
        return { path: '/clusters' }
      }
    } catch {
      notifyError('请先导入集群')
      return { path: '/clusters' }
    }
  }

  return true
})
