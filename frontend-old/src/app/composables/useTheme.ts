import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const THEME_KEY = 'k8s-platform:theme'
export type ThemeMode = 'light' | 'dark' | 'system'

/* ── 全局单例状态 ─────────────────────────────────────────────────────── */
const theme = ref<ThemeMode>('light')
let _initialized = false

function resolveSystemTheme(): 'light' | 'dark' {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function applyDomTheme(effective: 'light' | 'dark') {
  document.documentElement.classList.toggle('dark', effective === 'dark')
}

function applyTheme(next: ThemeMode) {
  theme.value = next
  localStorage.setItem(THEME_KEY, next)
  applyDomTheme(next === 'system' ? resolveSystemTheme() : next)
}

function toggleTheme() {
  const order: ThemeMode[] = ['light', 'dark', 'system']
  const idx = order.indexOf(theme.value)
  applyTheme(order[(idx + 1) % order.length])
}

const systemMediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
function onSystemThemeChange() {
  if (theme.value === 'system') applyDomTheme(resolveSystemTheme())
}

/**
 * 主题管理 composable — 全局单例
 * 首次挂载时从 localStorage 恢复，此后多组件共享同一 ref
 */
export function useTheme() {
  const themeLabel = computed(() => {
    if (theme.value === 'light') return '浅色模式'
    if (theme.value === 'dark') return '深色模式'
    return '跟随系统'
  })

  const effectiveTheme = computed<'light' | 'dark'>(() =>
    theme.value === 'system' ? resolveSystemTheme() : theme.value
  )

  onMounted(() => {
    if (!_initialized) {
      const saved = localStorage.getItem(THEME_KEY) as ThemeMode | null
      const initial: ThemeMode = saved === 'dark' || saved === 'system' ? saved : 'light'
      applyTheme(initial)
      systemMediaQuery.addEventListener('change', onSystemThemeChange)
      _initialized = true
    }
  })

  onBeforeUnmount(() => {
    // 单例模式下不移除监听 — 全局生命周期
  })

  return {
    theme,
    themeLabel,
    effectiveTheme,
    toggleTheme,
    applyTheme
  }
}
