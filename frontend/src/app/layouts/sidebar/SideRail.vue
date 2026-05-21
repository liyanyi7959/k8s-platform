<template>
  <nav class="side-rail">
    <!-- 顶部：Logo -->
    <div class="rail-logo" @click="router.push('/')">
      <img :src="logoUrl" alt="星枢" class="logo-img" />
    </div>

    <!-- 导航图标列表 -->
    <div class="rail-nav">
      <el-tooltip
        v-for="group in railGroups"
        :key="group.key"
        :content="group.title"
        placement="right"
        :show-after="300"
      >
        <button
          :class="[
            'rail-item',
            activeGroupKey === group.key ? 'rail-item--active' : ''
          ]"
          type="button"
          @click="onRailClick(group)"
          @mouseenter="onItemHover(group)"
          @mouseleave="onItemLeave"
        >
          <span class="rail-indicator" />
          <el-icon class="rail-icon"><component :is="group.icon" /></el-icon>
        </button>
      </el-tooltip>

      <div v-if="canShowShortcuts && shortcuts.length > 0" class="rail-divider" />

      <el-tooltip
        v-for="shortcut in canShowShortcuts ? shortcuts : []"
        :key="shortcut.id"
        :content="shortcut.name"
        placement="right"
        :show-after="300"
      >
        <button
          :class="[
            'rail-cluster',
            isShortcutActive(shortcut.id) ? 'rail-cluster--active' : '',
            shortcut.status ? `rail-cluster-status--${shortcut.status}` : ''
          ]"
          type="button"
          @click="openShortcut(shortcut)"
          @contextmenu.prevent="removeShortcut(shortcut.id)"
        >
          <span class="rail-cluster-dot" />
          <span class="rail-cluster-text">{{ shortcutInitial(shortcut.name) }}</span>
        </button>
      </el-tooltip>
    </div>

    <!-- 底部占位 -->
    <div class="rail-bottom" />
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMenu, type NavGroup } from '@/app/composables/useMenu'
import { getClusterUnavailableMessage, useClusterShortcuts, type ClusterShortcut } from '@/app/composables/useClusterShortcuts'
import { notifyError } from '@/shared/utils/notify'
import logoUrl from '@/assets/images/logo.svg'

const route = useRoute()
const router = useRouter()
const { railGroups, activeGroupKey } = useMenu()
const { shortcuts, unpinCluster: removeShortcut } = useClusterShortcuts()
const canShowShortcuts = computed(() => railGroups.value.some((group) => group.key === 'k8s'))

function onRailClick(group: NavGroup) {
  if (!group.children || group.children.length === 0) {
    // 无子菜单 — 直接导航
    if (group.path) router.push(group.path)
  } else {
    // 有子菜单 — 导航并打开/固定面板
    if (group.path) router.push(group.path)
  }
}

function onItemHover(_group: NavGroup) {
  // no-op (legacy)
}

function onItemLeave() {
  // no-op (legacy)
}

function shortcutInitial(name: string) {
  const text = String(name ?? '').trim()
  return text ? text.slice(0, 1).toUpperCase() : 'K'
}

function isShortcutActive(id: number) {
  return route.name === 'K8sClusterManage' && String(route.params.clusterId ?? '') === String(id)
}

async function openShortcut(shortcut: ClusterShortcut) {
  const msg = getClusterUnavailableMessage(shortcut.status)
  if (msg) {
    notifyError(msg)
    return
  }
  await router.push({ name: 'K8sClusterManage', params: { clusterId: String(shortcut.id) } })
}
</script>

<style scoped>
/* ── 主题变量 — Light（默认） ─────────────────────────────────────────── */
.side-rail {
  --rail-bg:            var(--color-bg-sidebar, #f0f4f8);
  --rail-border:        var(--color-border-default, rgba(0, 0, 0, 0.06));
  --rail-icon-color:    var(--color-text-muted, #94a3b8);
  --rail-icon-hover:    var(--color-text-primary, #0f172a);
  --rail-hover-bg:      var(--color-bg-hover, rgba(0, 0, 0, 0.04));
  --rail-active-color:  var(--color-accent-primary, #3b82f6);
  --rail-active-bg:     rgba(59, 130, 246, 0.12);
  --rail-indicator:     #3b82f6;
}

/* ── 主题变量 — Dark ─────────────────────────────────────────────────── */
:global(html.dark) .side-rail {
  --rail-bg:            var(--color-bg-sidebar, #0c1322);
  --rail-border:        rgba(255, 255, 255, 0.06);
  --rail-icon-color:    rgba(148, 163, 184, 0.8);
  --rail-icon-hover:    #e2e8f0;
  --rail-hover-bg:      rgba(255, 255, 255, 0.08);
  --rail-active-color:  #bfdbfe;
  --rail-active-bg:     rgba(59, 130, 246, 0.22);
  --rail-indicator:     #60a5fa;
}

.side-rail {
  width: 56px;
  min-width: 56px;
  display: flex;
  flex-direction: column;
  align-items: center;
  background: var(--rail-bg);
  border-right: 1px solid var(--rail-border);
  z-index: 60;
  overflow: hidden;
  flex-shrink: 0;
  transition: background 0.25s ease, border-color 0.25s ease;
}

/* ── Logo ──────────────────────────────────────────────────────────────── */
.rail-logo {
  width: 56px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
  transition: opacity 0.2s;
}
.rail-logo:hover {
  opacity: 0.85;
}
.logo-img {
  width: 28px;
  height: 28px;
}

/* ── Navigation ────────────────────────────────────────────────────────── */
.rail-nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 0;
  overflow-y: auto;
  overflow-x: hidden;
}

.rail-divider {
  width: 24px;
  height: 1px;
  margin: 8px 0 4px;
  background: var(--rail-border);
}

.rail-item {
  position: relative;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  border-radius: 10px;
  cursor: pointer;
  color: var(--rail-icon-color);
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
.rail-item:hover {
  color: var(--rail-icon-hover);
  background: var(--rail-hover-bg);
}
.rail-item--active {
  color: var(--rail-active-color);
  background: var(--rail-active-bg);
}

/* ── 左侧激活条 ────────────────────────────────────────────────────────── */
.rail-indicator {
  position: absolute;
  left: -8px;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 0;
  border-radius: 0 3px 3px 0;
  background: var(--rail-indicator);
  transition: height 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
.rail-item--active .rail-indicator {
  height: 20px;
}

/* ── Icon ──────────────────────────────────────────────────────────────── */
.rail-icon {
  font-size: 20px;
}

.rail-cluster {
  position: relative;
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.56);
  color: var(--rail-icon-hover);
  cursor: pointer;
  transition: all 0.18s ease;
}

.rail-cluster:hover {
  border-color: rgba(59, 130, 246, 0.26);
  background: rgba(59, 130, 246, 0.1);
  color: var(--rail-active-color);
}

.rail-cluster--active {
  border-color: rgba(59, 130, 246, 0.34);
  background: var(--rail-active-bg);
  color: var(--rail-active-color);
}

.rail-cluster-text {
  max-width: 20px;
  overflow: hidden;
  font-size: 13px;
  font-weight: 800;
  line-height: 1;
  text-transform: uppercase;
}

.rail-cluster-dot {
  position: absolute;
  right: 5px;
  bottom: 5px;
  width: 7px;
  height: 7px;
  border-radius: 999px;
  border: 1px solid var(--rail-bg);
  background: #94a3b8;
}

.rail-cluster-status--active .rail-cluster-dot {
  background: #22c55e;
}

.rail-cluster-status--degraded .rail-cluster-dot,
.rail-cluster-status--creating .rail-cluster-dot {
  background: #f59e0b;
}

.rail-cluster-status--disabled .rail-cluster-dot,
.rail-cluster-status--deleting .rail-cluster-dot {
  background: #ef4444;
}

:global(html.dark) .rail-cluster {
  background: rgba(255, 255, 255, 0.04);
}

/* ── Bottom ─────────────────────────────────────────────────────────────── */
.rail-bottom {
  flex-shrink: 0;
  height: 16px;
}
</style>
