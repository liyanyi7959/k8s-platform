<template>
  <header class="top-bar">
    <!-- 左侧：Hamburger + 面包屑 -->
    <div class="top-bar-left">
      <button class="hamburger-btn" @click="toggleFromTopBar" :title="panelOpen ? '收起侧栏' : '展开侧栏'">
        <el-icon :size="18">
          <Fold v-if="panelOpen" />
          <Expand v-else />
        </el-icon>
      </button>

      <nav class="breadcrumb">
        <span class="breadcrumb-item breadcrumb-root" @click="router.push('/')">
          <el-icon :size="14"><HomeFilled /></el-icon>
        </span>
        <template v-if="activeGroup && activeGroup.key !== 'dashboard'">
          <span class="breadcrumb-sep">/</span>
          <span
            class="breadcrumb-item"
            @click="activeGroup.path && router.push(activeGroup.path)"
          >{{ activeGroup.title }}</span>
        </template>
        <template v-if="currentTitle && activeGroup?.key !== 'dashboard'">
          <span class="breadcrumb-sep">/</span>
          <span class="breadcrumb-item breadcrumb-current">{{ currentTitle }}</span>
        </template>
      </nav>
    </div>

    <!-- 右侧：搜索 + 操作按钮 + 用户 -->
    <div class="top-bar-right">
      <!-- 搜索触发器 -->
      <button class="action-btn search-trigger" @click="openSearch" title="搜索 ⌘K">
        <el-icon :size="16"><Search /></el-icon>
        <span class="search-label hidden lg:inline">搜索...</span>
        <kbd class="search-kbd hidden lg:inline">⌘K</kbd>
      </button>

      <!-- 通知 -->
      <el-dropdown trigger="click">
        <button class="action-btn">
          <el-badge :value="messageCount" :hidden="messageCount <= 0" is-dot>
            <el-icon :size="16"><Bell /></el-icon>
          </el-badge>
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item disabled>暂无新消息</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>

      <!-- 主题切换 -->
      <el-tooltip :content="themeLabel" placement="bottom" :show-after="400">
        <button class="action-btn" @click="toggleTheme">
          <el-icon :size="16"><component :is="themeIcon" /></el-icon>
        </button>
      </el-tooltip>

      <!-- 分隔线 -->
      <div class="top-divider" />

      <!-- 用户头像 -->
      <el-dropdown @command="onUserCommand">
        <button class="user-btn">
          <span class="user-avatar-mini">
            {{ (userStore.me?.username ?? 'U').slice(0, 1).toUpperCase() }}
          </span>
          <span class="user-name-text hidden xl:inline">
            {{ userStore.me?.username ?? 'Admin' }}
          </span>
          <el-icon class="user-arrow hidden xl:inline" :size="12"><ArrowDown /></el-icon>
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="logout">
              <el-icon><SwitchButton /></el-icon>退出登录
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/app/store/user'
import { useMenu } from '@/app/composables/useMenu'
import { useSidebarState } from '@/app/composables/useSidebarState'
import { useTheme } from '@/app/composables/useTheme'
import {
  HomeFilled, Search, Bell, Moon, Sunny, Monitor,
  Fold, Expand, ArrowDown, SwitchButton
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { activeGroup } = useMenu()
const { panelOpen, toggleFromTopBar } = useSidebarState()
const { theme, themeLabel, toggleTheme } = useTheme()

const messageCount = ref(0)

/* ── 面包屑：当前页面标题 ─────────────────────────────────────────────── */
const currentTitle = computed(() => String(route.meta?.title ?? ''))

/* ── 主题图标 ─────────────────────────────────────────────────────────── */
const themeIcon = computed(() => {
  if (theme.value === 'dark') return Sunny
  if (theme.value === 'system') return Monitor
  return Moon
})

/* ── 搜索 ─────────────────────────────────────────────────────────────── */
function openSearch() {
  // TODO: Phase D — open CommandPalette
  // For now, trigger Ctrl+K / Cmd+K
}

/* Ctrl/Cmd + K 快捷键 */
function onKeyDown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault()
    openSearch()
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeyDown)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeyDown)
})

/* ── 用户菜单 ─────────────────────────────────────────────────────────── */
async function onUserCommand(cmd: string) {
  if (cmd === 'logout') {
    await userStore.logout()
    await router.push('/login')
  }
}
</script>

<style scoped>
.top-bar {
  height: 48px;
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  background: var(--color-bg-card, #fff);
  border-bottom: 1px solid var(--color-border-subtle, rgba(226, 232, 240, 0.6));
  z-index: 50;
  gap: 12px;
  flex-shrink: 0;
}
html.dark .top-bar {
  background: var(--color-bg-card, #1e293b);
  border-bottom-color: rgba(51, 65, 85, 0.5);
}

/* ── Left section ──────────────────────────────────────────────────────── */
.top-bar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.hamburger-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  color: var(--color-text-secondary, #64748b);
  transition: all 0.15s;
  flex-shrink: 0;
}
.hamburger-btn:hover {
  background: var(--color-bg-hover, rgba(0, 0, 0, 0.04));
  color: var(--color-text-primary, #0f172a);
}
html.dark .hamburger-btn:hover {
  background: rgba(255, 255, 255, 0.06);
  color: #e2e8f0;
}

/* ── Breadcrumb ────────────────────────────────────────────────────────── */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  min-width: 0;
  overflow: hidden;
}
.breadcrumb-item {
  color: var(--color-text-secondary, #64748b);
  cursor: pointer;
  white-space: nowrap;
  display: flex;
  align-items: center;
  transition: color 0.15s;
}
.breadcrumb-item:hover {
  color: var(--color-text-primary, #0f172a);
}
html.dark .breadcrumb-item:hover {
  color: #e2e8f0;
}
.breadcrumb-root {
  display: flex;
  align-items: center;
}
.breadcrumb-current {
  color: var(--color-text-primary, #0f172a);
  font-weight: 500;
  cursor: default;
  overflow: hidden;
  text-overflow: ellipsis;
}
html.dark .breadcrumb-current {
  color: #f1f5f9;
}
.breadcrumb-sep {
  color: var(--color-text-muted, #94a3b8);
  font-size: 12px;
  user-select: none;
}

/* ── Right section ─────────────────────────────────────────────────────── */
.top-bar-right {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

/* ── Action buttons ────────────────────────────────────────────────────── */
.action-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  color: var(--color-text-secondary, #64748b);
  transition: all 0.15s;
  flex-shrink: 0;
}
.action-btn:hover {
  background: rgba(59, 130, 246, 0.1);
  color: #2563eb;
}
html.dark .action-btn:hover {
  background: rgba(59, 130, 246, 0.18);
  color: #93c5fd;
}

/* ── Search trigger ────────────────────────────────────────────────────── */
.search-trigger {
  width: auto;
  gap: 8px;
  padding: 0 10px;
  border: 1px solid var(--color-border-subtle, rgba(226, 232, 240, 0.6));
  border-radius: 8px;
  height: 32px;
}
.search-trigger:hover {
  border-color: rgba(59, 130, 246, 0.4);
}
html.dark .search-trigger {
  border-color: rgba(100, 116, 139, 0.3);
}
html.dark .search-trigger:hover {
  border-color: rgba(96, 165, 250, 0.45);
}
.search-label {
  font-size: 13px;
  color: var(--color-text-muted, #94a3b8);
}
.search-kbd {
  font-size: 11px;
  font-family: 'Inter', system-ui, sans-serif;
  color: var(--color-text-muted, #94a3b8);
  background: var(--color-bg-hover, rgba(0, 0, 0, 0.04));
  padding: 1px 5px;
  border-radius: 4px;
  border: 1px solid var(--color-border-subtle, rgba(226, 232, 240, 0.6));
  font-weight: 600;
  line-height: 1;
}
html.dark .search-kbd {
  background: rgba(255, 255, 255, 0.06);
  border-color: rgba(100, 116, 139, 0.3);
}

/* ── Divider ───────────────────────────────────────────────────────────── */
.top-divider {
  width: 1px;
  height: 20px;
  background: var(--color-border-subtle, rgba(226, 232, 240, 0.6));
  margin: 0 4px;
  flex-shrink: 0;
}
html.dark .top-divider {
  background: rgba(100, 116, 139, 0.3);
}

/* ── User button ───────────────────────────────────────────────────────── */
.user-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px 4px 4px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
}
.user-btn:hover {
  background: rgba(59, 130, 246, 0.1);
}
html.dark .user-btn:hover {
  background: rgba(59, 130, 246, 0.18);
}

.user-avatar-mini {
  width: 26px;
  height: 26px;
  background: linear-gradient(135deg, #2563eb, #60a5fa);
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 700;
  font-size: 12px;
  flex-shrink: 0;
}

.user-name-text {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary, #1e293b);
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
html.dark .user-name-text {
  color: #e2e8f0;
}

.user-arrow {
  color: var(--color-text-muted, #94a3b8);
}
</style>
