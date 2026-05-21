<template>
  <header class="top-bar">
    <!-- 左侧：面包屑 -->
    <div class="top-bar-left">
      <button
        :class="['sidebar-toggle-btn', collapsed ? 'sidebar-toggle-btn--collapsed' : '']"
        :title="collapsed ? '展开侧栏' : '收起侧栏'"
        @click="toggleCollapse"
      >
        <span class="sidebar-toggle-icon" aria-hidden="true">
          <span class="sidebar-toggle-pane sidebar-toggle-pane--side" />
          <span class="sidebar-toggle-pane sidebar-toggle-pane--content" />
        </span>
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

    <!-- 右侧：操作按钮 + 用户 -->
    <div class="top-bar-right">
      <!-- 通知 -->
      <el-dropdown trigger="click">
        <button class="action-btn">
          <span class="action-icon-shell">
            <el-badge class="action-icon-badge" :value="messageCount" :hidden="messageCount <= 0" is-dot>
              <el-icon class="action-icon action-icon--notice"><Bell /></el-icon>
            </el-badge>
          </span>
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
          <span class="action-icon-shell">
            <el-icon class="action-icon action-icon--theme"><component :is="themeIcon" /></el-icon>
          </span>
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
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/app/store/user'
import { useMenu } from '@/app/composables/useMenu'
import { useSidebarState } from '@/app/composables/useSidebarState'
import { useTheme } from '@/app/composables/useTheme'
import {
  HomeFilled, Bell, Moon, Sunny, Monitor,
  ArrowDown, SwitchButton
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { activeGroup } = useMenu()
const { collapsed, toggleCollapse } = useSidebarState()
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
  height: 60px;
  min-height: 60px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  background: transparent;
  border: none;
  border-radius: 0;
  z-index: 50;
  gap: 20px;
  flex-shrink: 0;
  box-shadow: none;
}

html.dark .top-bar {
  background: transparent;
  box-shadow: none;
}

/* ── Left section ──────────────────────────────────────────────────────── */
.top-bar-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.sidebar-toggle-btn {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--color-border-default, rgba(15, 23, 42, 0.08));
  background: rgba(255, 255, 255, 0.82);
  border-radius: 12px;
  cursor: pointer;
  color: var(--color-text-secondary, #64748b);
  line-height: 0;
  box-shadow: 0 4px 10px rgba(15, 23, 42, 0.04);
  transition:
    background 0.18s ease,
    color 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;
  flex-shrink: 0;
}

.sidebar-toggle-btn:hover {
  background: var(--color-bg-hover, #f8fafc);
  border-color: rgba(59, 130, 246, 0.18);
  color: var(--color-text-primary, #0f172a);
  box-shadow: 0 8px 18px rgba(59, 130, 246, 0.12);
}

html.dark .sidebar-toggle-btn {
  background: rgba(15, 23, 42, 0.28);
  border-color: rgba(255, 255, 255, 0.08);
  box-shadow: none;
}

html.dark .sidebar-toggle-btn:hover {
  background: rgba(255, 255, 255, 0.06);
  border-color: rgba(96, 165, 250, 0.2);
  color: #e2e8f0;
  box-shadow: none;
}

.sidebar-toggle-icon {
  position: relative;
  width: 18px;
  height: 18px;
  display: block;
}

.sidebar-toggle-pane {
  position: absolute;
  top: 2px;
  bottom: 2px;
  border-radius: 4px;
  transition:
    transform 0.28s cubic-bezier(0.4, 0, 0.2, 1),
    left 0.28s cubic-bezier(0.4, 0, 0.2, 1),
    right 0.28s cubic-bezier(0.4, 0, 0.2, 1),
    width 0.28s cubic-bezier(0.4, 0, 0.2, 1),
    opacity 0.18s ease,
    box-shadow 0.18s ease;
}

.sidebar-toggle-pane--side {
  left: 1px;
  width: 5px;
  background: currentColor;
  opacity: 0.92;
}

.sidebar-toggle-pane--content {
  right: 1px;
  width: 9px;
  border: 1.8px solid currentColor;
  background: transparent;
  opacity: 0.82;
}

.sidebar-toggle-btn:hover .sidebar-toggle-pane--side {
  box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.08);
}

.sidebar-toggle-btn:hover .sidebar-toggle-pane--content {
  transform: translateX(1px);
}

.sidebar-toggle-btn--collapsed .sidebar-toggle-pane--side {
  width: 3px;
  opacity: 0.68;
}

.sidebar-toggle-btn--collapsed .sidebar-toggle-pane--content {
  width: 11px;
  opacity: 0.92;
}

/* ── Breadcrumb ────────────────────────────────────────────────────────── */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  min-height: 34px;
  padding: 0;
  overflow: hidden;
  font-size: 13px;
}

.breadcrumb-item {
  color: var(--color-text-secondary, #64748b);
  cursor: pointer;
  white-space: nowrap;
  display: flex;
  align-items: center;
  transition: color 0.15s;
  font-weight: 600;
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
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  border: none;
  background: var(--color-bg-hover, #f8fafc);
  color: var(--color-accent-primary, #2563eb);
  flex-shrink: 0;
}

html.dark .breadcrumb-root {
  background: rgba(255, 255, 255, 0.06);
  color: #bfdbfe;
}

.breadcrumb-current {
  color: var(--color-text-primary, #0f172a);
  font-weight: 700;
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
  gap: 8px;
  flex-shrink: 0;
}

.top-bar-right > *:not(.top-divider) {
  display: flex;
  align-items: center;
}

/* ── Action buttons ────────────────────────────────────────────────────── */
.action-btn {
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  color: var(--color-text-secondary, #64748b);
  line-height: 0;
  transition: all 0.15s;
  flex-shrink: 0;
}

.action-btn:hover {
  background: var(--color-bg-hover, #f8fafc);
  color: var(--color-text-primary, #0f172a);
}

html.dark .action-btn {
  background: transparent;
}

html.dark .action-btn:hover {
  background: rgba(255, 255, 255, 0.06);
  color: #e2e8f0;
}

.action-icon-shell {
  width: 18px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 0;
}

.action-icon-badge {
  width: 18px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 0;
}

.action-icon {
  width: 18px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 0;
}

.action-icon--notice {
  transform: translateY(-0.5px);
}

.action-icon--theme {
  transform: translateY(-0.5px);
}

.action-icon :deep(svg) {
  width: 18px;
  height: 18px;
  display: block;
}

.action-btn :deep(.el-badge) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 0;
}

.action-btn :deep(.el-badge__content.is-dot) {
  top: 2px;
  right: 2px;
}

/* ── Divider ───────────────────────────────────────────────────────────── */
.top-divider {
  width: 1px;
  height: 16px;
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
  padding: 2px 6px 2px 2px;
  border: none;
  background: transparent;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
}

.user-btn:hover {
  background: var(--color-bg-hover, #f8fafc);
}

html.dark .user-btn {
  background: transparent;
}

html.dark .user-btn:hover {
  background: rgba(255, 255, 255, 0.06);
}

.user-avatar-mini {
  width: 28px;
  height: 28px;
  background: rgba(37, 99, 235, 0.12);
  border-radius: 999px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #2563eb;
  font-weight: 700;
  font-size: 12px;
  flex-shrink: 0;
}

html.dark .user-avatar-mini {
  background: rgba(96, 165, 250, 0.18);
  color: #bfdbfe;
}

.user-name-text {
  font-size: 13px;
  font-weight: 600;
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
