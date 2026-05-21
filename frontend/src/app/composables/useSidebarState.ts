import { ref, computed } from 'vue'

/* ── 持久化 Key ────────────────────────────────────────────────────────── */
const SIDEBAR_KEY = 'k8s-platform:sidebar'

/* ── 全局单例状态 ─────────────────────────────────────────────────────── */
/** 侧边栏是否折叠 */
const collapsed = ref(false)

/* ── 初始化（仅一次） ──────────────────────────────────────────────────── */
let _initialized = false
function init() {
  if (_initialized) return
  _initialized = true
  try {
    const raw = localStorage.getItem(SIDEBAR_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      collapsed.value = parsed.collapsed ?? false
    }
  } catch {
    // ignore
  }
}

/* ── 持久化 ────────────────────────────────────────────────────────────── */
function persist() {
  localStorage.setItem(SIDEBAR_KEY, JSON.stringify({
    collapsed: collapsed.value
  }))
}

/**
 * 侧栏状态管理 composable — 全局单例
 */
export function useSidebarState() {
  init()

  /** 切换折叠/展开 */
  function toggleCollapse() {
    collapsed.value = !collapsed.value
    persist()
  }

  /** 设置折叠状态 */
  function setCollapsed(val: boolean) {
    collapsed.value = val
    persist()
  }

  return {
    collapsed,
    toggleCollapse,
    setCollapsed
  }
}
