import { ref, watch } from 'vue'

/* ── 持久化 Key ────────────────────────────────────────────────────────── */
const SIDEBAR_KEY = 'k8s-platform:sidebar'

/* ── 全局单例状态 ─────────────────────────────────────────────────────── */
/** Side Panel 是否展开 */
const panelOpen = ref(true)
/** 用户是否手动 pin 了面板 */
const pinned = ref(true)
/** Rail hover 时临时展开面板 (非 pin 状态下) */
const hoverOpen = ref(false)

/* ── 初始化（仅一次） ──────────────────────────────────────────────────── */
let _initialized = false
function init() {
  if (_initialized) return
  _initialized = true
  try {
    const raw = localStorage.getItem(SIDEBAR_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      panelOpen.value = parsed.panelOpen ?? true
      pinned.value = parsed.pinned ?? true
    }
  } catch {
    // ignore
  }
}

/* ── 持久化 ────────────────────────────────────────────────────────────── */
function persist() {
  localStorage.setItem(SIDEBAR_KEY, JSON.stringify({
    panelOpen: panelOpen.value,
    pinned: pinned.value
  }))
}

/**
 * 侧栏状态管理 composable — 全局单例
 */
export function useSidebarState() {
  init()

  /** 切换面板展开/折叠 */
  function togglePanel() {
    panelOpen.value = !panelOpen.value
    pinned.value = panelOpen.value
    hoverOpen.value = false
    persist()
  }

  /** 点击 hamburger 按钮 */
  function toggleFromTopBar() {
    togglePanel()
  }

  /** Rail 图标 hover 时临时展开 */
  function onRailHoverEnter() {
    if (!pinned.value) {
      hoverOpen.value = true
    }
  }

  /** Rail 图标 hover 结束 */
  function onRailHoverLeave() {
    hoverOpen.value = false
  }

  /** pin / unpin */
  function togglePin() {
    pinned.value = !pinned.value
    if (pinned.value) {
      panelOpen.value = true
      hoverOpen.value = false
    } else {
      panelOpen.value = false
    }
    persist()
  }

  /** 面板是否应该可见（pinned 展开 或 hover 临时展开） */
  const isPanelVisible = ref(false)
  watch(
    [panelOpen, hoverOpen],
    ([open, hover]) => {
      isPanelVisible.value = open || hover
    },
    { immediate: true }
  )

  return {
    panelOpen,
    pinned,
    hoverOpen,
    isPanelVisible,
    togglePanel,
    toggleFromTopBar,
    togglePin,
    onRailHoverEnter,
    onRailHoverLeave
  }
}
