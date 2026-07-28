/**
 * 用户全局状态模型
 * 使用 UmiJS plugin-model 管理用户状态
 */
import { useState, useCallback } from 'react'
import { history } from '@umijs/max'
import { getCurrentUser, logout as logoutApi } from '@/features/iam/api'
import type { User } from '@/shared/types'

export default function useUserModel() {
  const [user, setUser] = useState<User | null>(null)
  const [permissions, setPermissions] = useState<string[]>([])
  const [loading, setLoading] = useState(false)

  /** 获取当前用户信息 */
  const fetchUser = useCallback(async () => {
    setLoading(true)
    try {
      const currentUser = await getCurrentUser()
      setUser(currentUser)
      setPermissions(currentUser.roles.flatMap((r) => r.permissions.map((p) => p.code)))
      return currentUser
    } catch {
      setUser(null)
      setPermissions([])
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  /** 登出 */
  const logout = useCallback(async () => {
    try {
      await logoutApi()
    } finally {
      setUser(null)
      setPermissions([])
      history.push('/login')
    }
  }, [])

  /** 检查是否有指定权限 */
  const hasPermission = useCallback(
    (code: string) => {
      return permissions.includes(code)
    },
    [permissions],
  )

  return {
    user,
    permissions,
    loading,
    fetchUser,
    logout,
    hasPermission,
  }
}
