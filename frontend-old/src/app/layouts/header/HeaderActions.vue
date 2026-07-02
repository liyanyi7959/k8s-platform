<template>
  <div class="action-group">
    <el-tooltip content="刷新" placement="bottom">
      <button class="icon-btn-ghost btn-refresh" @click="emit('refresh')">
        <el-icon><RefreshRight /></el-icon>
      </button>
    </el-tooltip>

    <el-dropdown trigger="click">
      <button class="icon-btn-ghost btn-bell">
        <el-badge :value="messageCount" :hidden="messageCount <= 0" is-dot class="badge-dot">
          <el-icon><Bell /></el-icon>
        </el-badge>
      </button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item disabled>暂无新消息</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

    <el-tooltip :content="themeLabel" placement="bottom" :show-after="400">
      <button class="icon-btn-ghost btn-theme" @click="toggleTheme">
        <el-icon><component :is="themeIcon" /></el-icon>
      </button>
    </el-tooltip>
  </div>

  <div class="divider-vertical"></div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Bell, Moon, RefreshRight, Sunny, Monitor } from '@element-plus/icons-vue'

const emit = defineEmits<{ (e: 'refresh'): void }>()

const messageCount = ref(0)

// ──── Theme logic ────
const THEME_KEY = 'k8s-platform:theme'
type ThemeMode = 'light' | 'dark' | 'system'
const theme = ref<ThemeMode>('light')

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

const themeLabel = computed(() => {
  if (theme.value === 'light') return '浅色模式'
  if (theme.value === 'dark') return '深色模式'
  return '跟随系统'
})

const themeIcon = computed(() => {
  if (theme.value === 'dark') return Sunny
  if (theme.value === 'system') return Monitor
  return Moon
})

const systemMediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
function onSystemThemeChange() {
  if (theme.value === 'system') applyDomTheme(resolveSystemTheme())
}

onMounted(() => {
  const saved = localStorage.getItem(THEME_KEY) as ThemeMode | null
  const initial: ThemeMode = saved === 'dark' || saved === 'system' ? saved : 'light'
  applyTheme(initial)
  systemMediaQuery.addEventListener('change', onSystemThemeChange)
})

onBeforeUnmount(() => {
  systemMediaQuery.removeEventListener('change', onSystemThemeChange)
})
</script>

<style scoped>
.action-group {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px;
  background: rgba(255, 255, 255, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 14px;
  backdrop-filter: blur(8px);
  flex: 0 0 auto;
}
html.dark .action-group {
  background: rgba(30, 41, 59, 0.4);
  border-color: rgba(255, 255, 255, 0.08);
}

.icon-btn-ghost {
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  background: transparent;
  transition: background var(--duration-fast) var(--ease-default), color var(--duration-fast) var(--ease-default);
  cursor: pointer;
  border: none;
  position: relative;
  overflow: hidden;
}
.icon-btn-ghost:hover {
  opacity: 1;
}

.btn-refresh {
  color: var(--c-cyan-500);
}
.btn-refresh:hover {
  background: rgba(14, 165, 233, 0.1);
  color: var(--c-cyan-600);
}

.btn-bell {
  color: var(--c-amber-500);
}
.btn-bell:hover {
  background: rgba(245, 158, 11, 0.1);
  color: var(--c-amber-600);
}
.btn-bell :deep(.el-badge__content.is-dot) {
  background: #ef4444;
  border: 2px solid #fff;
  box-shadow: 0 0 0 1px rgba(255,255,255,0.5);
}

.btn-theme {
  color: #3b82f6;
}
.btn-theme:hover {
  background: rgba(59, 130, 246, 0.1);
  color: #2563eb;
}

.divider-vertical {
  width: 1px;
  height: 24px;
  background: rgba(203, 213, 225, 0.8);
  margin: 0 4px;
}
html.dark .divider-vertical {
  background: rgba(100, 116, 139, 0.4);
}
</style>
