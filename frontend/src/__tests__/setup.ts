/**
 * Vitest 全局测试环境配置
 * 提供 Vue 3 + Element Plus 测试所需的全局 mock
 */
import { vi } from 'vitest'

// Mock Element Plus 组件（避免测试中加载完整 UI 库）
vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
  ElMessageBox: { confirm: vi.fn().mockResolvedValue('confirm') },
  ElNotification: vi.fn(),
}))

// Mock vue-router
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn() }),
  useRoute: () => ({ params: {}, query: {}, path: '/' }),
}))
