<template>
  <header class="top-bar">
    <!-- 左侧：面包屑 -->
    <div class="top-bar-left">
      <button
        :class="['sidebar-toggle-btn', collapsed ? 'sidebar-toggle-btn--collapsed' : '']"
        :title="collapsed ? '展开侧栏' : '收起侧栏'"
        @click="toggleCollapse"
      >
        <svg class="sidebar-toggle-glyph" viewBox="0 0 1024 1024" aria-hidden="true">
          <path
            class="sidebar-toggle-glyph__path"
            d="M106.24 535.893L271.787 651.52c20.053 14.08 45.653 0 45.653-22.613V395.093c0-5.973-2.133-11.946-5.547-17.066-9.386-12.374-27.306-14.934-40.106-5.547L106.24 488.107c-14.08 11.093-14.08 33.706 0 47.786z m23.04-322.56h785.067c18.773 0 34.133-15.36 34.133-34.133s-15.36-34.133-34.133-34.133H129.28c-18.773 0-34.133 15.36-34.133 34.133s15.36 34.133 34.133 34.133z m0 665.6h785.067c18.773 0 34.133-15.36 34.133-34.133s-15.36-34.133-34.133-34.133H129.28c-18.773 0-34.133 15.36-34.133 34.133s15.36 34.133 34.133 34.133zM419.413 435.2h494.934c18.773 0 34.133-15.36 34.133-34.133s-15.36-34.134-34.133-34.134H419.413c-18.773 0-34.133 15.36-34.133 34.134s15.36 34.133 34.133 34.133z m0 221.867h495.36c18.774 0 34.134-15.36 34.134-34.134s-15.36-34.133-34.134-34.133h-495.36c-18.773 0-34.133 15.36-34.133 34.133v0.427c0 18.347 15.36 33.707 34.133 33.707z"
          />
        </svg>
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

.sidebar-toggle-glyph {
  width: 19px;
  height: 19px;
  display: block;
  fill: currentColor;
  transition: transform 0.24s cubic-bezier(0.4, 0, 0.2, 1);
}

.sidebar-toggle-glyph__path {
  fill: currentColor;
}

.sidebar-toggle-btn:hover .sidebar-toggle-glyph {
  transform: scale(1.03);
}

.sidebar-toggle-btn--collapsed .sidebar-toggle-glyph {
  transform: scaleX(-1);
}

.sidebar-toggle-btn--collapsed:hover .sidebar-toggle-glyph {
  transform: scaleX(-1) scale(1.03);
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
