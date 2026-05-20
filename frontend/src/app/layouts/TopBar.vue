<template>
  <header class="top-bar">
    <!-- 左侧：面包屑 -->
    <div class="top-bar-left">
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
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/app/store/user'
import { useMenu } from '@/app/composables/useMenu'
import { useTheme } from '@/app/composables/useTheme'
import {
  HomeFilled, Bell, Moon, Sunny, Monitor,
  ArrowDown, SwitchButton
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { activeGroup } = useMenu()
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
  border-bottom: 1px solid var(--color-border-subtle, rgba(226, 232, 240, 0.7));
  border-radius: 0;
  z-index: 50;
  gap: 20px;
  flex-shrink: 0;
  box-shadow: none;
}

html.dark .top-bar {
  background: transparent;
  border-color: rgba(51, 65, 85, 0.6);
  box-shadow: none;
}

/* ── Left section ──────────────────────────────────────────────────────── */
.top-bar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
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
