/**
 * 权限定义
 * 基于 initialState 中的用户信息定义权限
 */
export default function access(initialState: { currentUser?: import('@/features/iam/types').User }) {
  const { currentUser } = initialState || {}

  return {
    canAdmin: currentUser?.roles?.some((r) => r.code === 'admin') ?? false,
    canUser: currentUser?.roles?.some((r) => r.code === 'user' || r.code === 'admin') ?? false,
  }
}
