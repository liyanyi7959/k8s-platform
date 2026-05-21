<template>
  <aside :class="['app-sidebar', collapsed ? 'app-sidebar--collapsed' : '']">
    <div class="sidebar-header">
      <div class="header-logo" @click="router.push('/')">
        <img :src="logoUrl" alt="星枢" class="logo-img" />
        <transition name="fade-text">
          <span v-show="!collapsed" class="logo-text">星枢K8S平台</span>
        </transition>
      </div>
    </div>

    <el-scrollbar class="sidebar-scroll">
      <nav class="sidebar-nav">
        <template v-for="group in railGroups" :key="group.key">
          <template v-if="collapsed">
            <el-popover
              v-if="hasChildren(group)"
              placement="right-start"
              :width="196"
              trigger="hover"
              :show-after="80"
              :hide-after="100"
              :offset="10"
              popper-class="sidebar-popover"
            >
              <template #reference>
                <button
                  :class="['nav-item', 'nav-item--icon-only', isGroupActive(group) ? 'nav-item--active' : '']"
                  type="button"
                  @click="onGroupClick(group)"
                >
                  <el-icon class="nav-item-icon"><component :is="group.icon" /></el-icon>
                </button>
              </template>

              <div class="popover-title">{{ group.title }}</div>
              <div class="popover-menu">
                <button
                  v-for="child in group.children"
                  :key="child.path"
                  :class="['popover-item', isActive(child.path) ? 'popover-item--active' : '']"
                  type="button"
                  @click="router.push(child.path)"
                >
                  <el-icon class="popover-item-icon"><component :is="child.icon" /></el-icon>
                  <span>{{ child.title }}</span>
                </button>
              </div>
            </el-popover>

            <el-tooltip
              v-else
              :content="group.title"
              placement="right"
              :show-after="200"
            >
              <button
                :class="['nav-item', 'nav-item--icon-only', isGroupActive(group) ? 'nav-item--active' : '']"
                type="button"
                @click="onGroupClick(group)"
              >
                <el-icon class="nav-item-icon"><component :is="group.icon" /></el-icon>
              </button>
            </el-tooltip>
          </template>

          <section v-else class="group-panel">
            <button
              :class="[
                'group-trigger',
                isGroupActive(group) ? 'group-trigger--active' : '',
                isGroupOpen(group.key) ? 'group-trigger--open' : ''
              ]"
              type="button"
              @click="onExpandedGroupClick(group)"
            >
              <span class="group-trigger-main">
                <el-icon class="group-trigger-icon"><component :is="group.icon" /></el-icon>
                <span class="group-trigger-title">{{ group.title }}</span>
              </span>
              <span
                v-if="hasChildren(group)"
                :class="['group-trigger-caret', isGroupOpen(group.key) ? 'group-trigger-caret--open' : '']"
              />
            </button>

            <div :class="['group-collapse', isGroupOpen(group.key) ? 'group-collapse--open' : '']">
              <div class="group-collapse-inner">
                <div class="group-children">
                  <button
                    v-for="child in group.children ?? []"
                    :key="child.path"
                    :class="['nav-subitem', isActive(child.path) ? 'nav-subitem--active' : '']"
                    type="button"
                    @click="router.push(child.path)"
                  >
                    <span class="nav-subitem-marker" />
                    <el-icon class="nav-subitem-icon"><component :is="child.icon" /></el-icon>
                    <span class="nav-subitem-title">{{ child.title }}</span>
                  </button>
                </div>

                <div v-if="group.key === 'k8s' && shortcuts.length > 0" class="group-shortcuts">
                  <div class="group-shortcuts-label">集群快捷入口</div>
                  <div class="shortcut-list">
                    <div
                      v-for="shortcut in shortcuts"
                      :key="shortcut.id"
                      :class="['shortcut-row', isShortcutActive(shortcut.id) ? 'shortcut-row--active' : '']"
                    >
                      <button class="shortcut-btn" type="button" @click="openShortcut(shortcut)">
                        <span :class="['shortcut-dot', shortcut.status ? `shortcut-dot--${shortcut.status}` : '']" />
                        <span class="shortcut-name">{{ shortcut.name }}</span>
                        <span v-if="shortcut.nodeCount != null" class="shortcut-meta">{{ shortcut.nodeCount }}节点</span>
                      </button>
                      <button class="shortcut-remove" type="button" title="移除" @click="removeShortcut(shortcut.id)">
                        <el-icon :size="12"><Close /></el-icon>
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </template>

        <template v-if="collapsed && canShowShortcuts && shortcuts.length > 0">
          <div class="nav-divider" />

          <el-tooltip
            v-for="shortcut in shortcuts"
            :key="shortcut.id"
            :content="shortcut.name"
            placement="right"
            :show-after="220"
          >
            <button
              :class="[
                'nav-shortcut',
                isShortcutActive(shortcut.id) ? 'nav-shortcut--active' : '',
                shortcut.status ? `nav-shortcut-status--${shortcut.status}` : ''
              ]"
              type="button"
              @click="openShortcut(shortcut)"
              @contextmenu.prevent="removeShortcut(shortcut.id)"
            >
              <span class="nav-shortcut-dot" />
              <span class="nav-shortcut-text">{{ shortcutInitial(shortcut.name) }}</span>
            </button>
          </el-tooltip>
        </template>
      </nav>
    </el-scrollbar>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Close } from '@element-plus/icons-vue'
import { useMenu, type NavGroup } from '@/app/composables/useMenu'
import { useSidebarState } from '@/app/composables/useSidebarState'
import { getClusterUnavailableMessage, useClusterShortcuts, type ClusterShortcut } from '@/app/composables/useClusterShortcuts'
import { notifyError } from '@/shared/utils/notify'
import logoUrl from '@/assets/images/logo.svg'

const route = useRoute()
const router = useRouter()
const { railGroups, activeGroupKey } = useMenu()
const { collapsed } = useSidebarState()
const { shortcuts, unpinCluster: removeShortcut } = useClusterShortcuts()

const canShowShortcuts = computed(() => railGroups.value.some((group) => group.key === 'k8s'))
const openGroupKey = ref('')

watch(
  [() => collapsed.value, () => activeGroupKey.value],
  ([isCollapsed, groupKey]) => {
    if (!isCollapsed && groupKey) {
      openGroupKey.value = groupKey
    }
  },
  { immediate: true }
)

function hasChildren(group: NavGroup): boolean {
  return (group.children?.length ?? 0) > 0
}

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(path + '/')
}

function isGroupActive(group: NavGroup): boolean {
  if (group.children?.some((child) => isActive(child.path))) return true
  return group.path ? isActive(group.path) : false
}

function isGroupOpen(key: string): boolean {
  return openGroupKey.value === key
}

function onGroupClick(group: NavGroup) {
  if (group.path) router.push(group.path)
}

function onExpandedGroupClick(group: NavGroup) {
  if (!hasChildren(group)) {
    if (group.path) router.push(group.path)
    return
  }

  openGroupKey.value = openGroupKey.value === group.key ? '' : group.key
}

function isShortcutActive(id: number) {
  return route.name === 'K8sClusterManage' && String(route.params.clusterId ?? '') === String(id)
}

function shortcutInitial(name: string) {
  const text = String(name ?? '').trim()
  return text ? text.slice(0, 1).toUpperCase() : 'K'
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
.app-sidebar {
  --sb-bg: var(--color-bg-sidebar, #f7f9fc);
  --sb-border: var(--color-border-default, rgba(15, 23, 42, 0.08));
  --sb-text: var(--color-text-secondary, #475569);
  --sb-text-primary: var(--color-text-primary, #0f172a);
  --sb-text-muted: var(--color-text-muted, #94a3b8);
  --sb-hover-bg: var(--color-bg-hover, rgba(15, 23, 42, 0.04));
  --sb-active-bg: rgba(59, 130, 246, 0.1);
  --sb-active-color: #2563eb;
  --sb-active-indicator: #3b82f6;
  --sb-branch: rgba(148, 163, 184, 0.24);
}

:global(html.dark) .app-sidebar {
  --sb-bg: var(--color-bg-sidebar, #0c1322);
  --sb-border: rgba(255, 255, 255, 0.08);
  --sb-text: rgba(203, 213, 225, 0.82);
  --sb-text-primary: #e2e8f0;
  --sb-text-muted: rgba(148, 163, 184, 0.6);
  --sb-hover-bg: rgba(255, 255, 255, 0.06);
  --sb-active-bg: rgba(59, 130, 246, 0.18);
  --sb-active-color: #93c5fd;
  --sb-active-indicator: #60a5fa;
  --sb-branch: rgba(148, 163, 184, 0.16);
}

.app-sidebar {
  width: 204px;
  min-width: 204px;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--sb-bg);
  border-right: 1px solid var(--sb-border);
  z-index: 60;
  overflow: hidden;
  flex-shrink: 0;
  transition: width 0.28s cubic-bezier(0.4, 0, 0.2, 1), min-width 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}

.app-sidebar--collapsed {
  width: 56px;
  min-width: 56px;
}

.sidebar-header {
  height: 60px;
  min-height: 60px;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  padding: 0 14px;
  flex-shrink: 0;
}

.app-sidebar--collapsed .sidebar-header {
  justify-content: center;
  padding: 0;
}

.header-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  min-width: 0;
  overflow: hidden;
}

.logo-img {
  width: 26px;
  height: 26px;
  flex-shrink: 0;
}

.logo-text {
  font-size: 14px;
  font-weight: 700;
  color: var(--sb-text-primary);
  white-space: nowrap;
  overflow: hidden;
}

.sidebar-scroll {
  flex: 1;
  min-height: 0;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 8px 14px;
}

.app-sidebar--collapsed .sidebar-nav {
  align-items: center;
  gap: 6px;
  padding: 8px 0 14px;
}

.group-panel {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.group-trigger {
  width: 100%;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0 12px;
  border: none;
  background: transparent;
  border-radius: 12px;
  color: var(--sb-text);
  cursor: pointer;
  transition: background 0.18s ease, color 0.18s ease;
}

.group-trigger:hover {
  background: var(--sb-hover-bg);
  color: var(--sb-text-primary);
}

.group-trigger--active,
.group-trigger--open {
  color: var(--sb-text-primary);
}

.group-trigger-main {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.group-trigger-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.group-trigger-title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 600;
}

.group-trigger-caret {
  position: relative;
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  color: var(--sb-text-muted);
  transition: transform 0.24s ease, color 0.18s ease;
}

.group-trigger-caret::before {
  content: '';
  position: absolute;
  top: 3px;
  left: 4px;
  width: 6px;
  height: 6px;
  border-right: 1.8px solid currentColor;
  border-bottom: 1.8px solid currentColor;
  transform: rotate(45deg);
}

.group-trigger--open .group-trigger-caret {
  transform: rotate(180deg);
  color: var(--sb-active-color);
}

.group-collapse {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}

.group-collapse--open {
  grid-template-rows: 1fr;
}

.group-collapse-inner {
  overflow: hidden;
}

.group-children {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-left: 14px;
  padding-left: 14px;
}

.group-children::before {
  content: '';
  position: absolute;
  left: 0;
  top: 4px;
  bottom: 4px;
  width: 1px;
  background: var(--sb-branch);
}

.nav-subitem {
  position: relative;
  width: 100%;
  height: 38px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 12px 0 14px;
  border: none;
  background: transparent;
  border-radius: 10px;
  color: var(--sb-text);
  cursor: pointer;
  text-align: left;
  transition: background 0.18s ease, color 0.18s ease;
}

.nav-subitem:hover {
  background: var(--sb-hover-bg);
  color: var(--sb-text-primary);
}

.nav-subitem--active {
  background: var(--sb-active-bg);
  color: var(--sb-active-color);
}

.nav-subitem-marker {
  position: absolute;
  left: -3px;
  top: 50%;
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.4);
  transform: translateY(-50%);
  transition: background 0.18s ease, box-shadow 0.18s ease;
}

.nav-subitem--active .nav-subitem-marker {
  background: var(--sb-active-indicator);
  box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.12);
}

.nav-subitem-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.nav-subitem-title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 600;
}

.group-shortcuts {
  margin: 8px 0 0 28px;
  padding-top: 10px;
  border-top: 1px dashed var(--sb-branch);
}

.group-shortcuts-label {
  padding: 0 8px 6px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: var(--sb-text-muted);
}

.nav-item {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  padding: 0;
  border: none;
  background: transparent;
  border-radius: 10px;
  cursor: pointer;
  color: var(--sb-text);
  transition: background 0.18s ease, color 0.18s ease;
}

.nav-item:hover {
  background: var(--sb-hover-bg);
  color: var(--sb-text-primary);
}

.nav-item--active {
  background: var(--sb-active-bg);
  color: var(--sb-active-color);
}

.nav-item--active::before {
  content: '';
  position: absolute;
  left: -4px;
  top: 50%;
  width: 3px;
  height: 18px;
  border-radius: 0 3px 3px 0;
  background: var(--sb-active-indicator);
  transform: translateY(-50%);
}

.nav-item-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.nav-divider {
  width: 24px;
  height: 1px;
  margin: 8px 0 4px;
  background: var(--sb-border);
}

.nav-shortcut {
  position: relative;
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.56);
  color: var(--sb-text-primary);
  cursor: pointer;
  transition: all 0.18s ease;
}

.nav-shortcut:hover {
  border-color: rgba(59, 130, 246, 0.26);
  background: rgba(59, 130, 246, 0.1);
  color: var(--sb-active-color);
}

.nav-shortcut--active {
  border-color: rgba(59, 130, 246, 0.34);
  background: var(--sb-active-bg);
  color: var(--sb-active-color);
}

.nav-shortcut-text {
  max-width: 20px;
  overflow: hidden;
  font-size: 13px;
  font-weight: 800;
  line-height: 1;
  text-transform: uppercase;
}

.nav-shortcut-dot {
  position: absolute;
  right: 5px;
  bottom: 5px;
  width: 7px;
  height: 7px;
  border-radius: 999px;
  border: 1px solid var(--sb-bg);
  background: #94a3b8;
}

.nav-shortcut-status--active .nav-shortcut-dot {
  background: #22c55e;
}

.nav-shortcut-status--degraded .nav-shortcut-dot,
.nav-shortcut-status--creating .nav-shortcut-dot {
  background: #f59e0b;
}

.nav-shortcut-status--disabled .nav-shortcut-dot,
.nav-shortcut-status--deleting .nav-shortcut-dot {
  background: #ef4444;
}

:global(html.dark) .nav-shortcut {
  background: rgba(255, 255, 255, 0.04);
}

.shortcut-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.shortcut-row {
  display: flex;
  align-items: center;
  border-radius: 10px;
  transition: background 0.15s;
}

.shortcut-row:hover,
.shortcut-row--active {
  background: var(--sb-active-bg);
}

.shortcut-btn {
  flex: 1;
  min-width: 0;
  height: 34px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: none;
  background: transparent;
  color: var(--sb-text);
  cursor: pointer;
  text-align: left;
}

.shortcut-row:hover .shortcut-btn,
.shortcut-row--active .shortcut-btn {
  color: var(--sb-active-color);
}

.shortcut-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #94a3b8;
  flex-shrink: 0;
}

.shortcut-dot--active { background: #22c55e; }
.shortcut-dot--degraded,
.shortcut-dot--creating { background: #f59e0b; }
.shortcut-dot--disabled,
.shortcut-dot--deleting { background: #ef4444; }

.shortcut-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 600;
}

.shortcut-meta {
  flex-shrink: 0;
  color: var(--sb-text-muted);
  font-size: 11px;
}

.shortcut-remove {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  border-radius: 6px;
  color: var(--sb-text-muted);
  cursor: pointer;
  opacity: 0;
  transition: all 0.15s ease;
}

.shortcut-row:hover .shortcut-remove,
.shortcut-row--active .shortcut-remove {
  opacity: 1;
}

.shortcut-remove:hover {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.fade-text-enter-active,
.fade-text-leave-active {
  transition: opacity 0.2s ease;
}

.fade-text-enter-from,
.fade-text-leave-to {
  opacity: 0;
}
</style>

<style>
.sidebar-popover.el-popover {
  --el-popover-padding: 8px !important;
  border-radius: 12px !important;
  min-width: 176px !important;
}

.popover-title {
  padding: 6px 10px 6px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
  color: var(--el-text-color-secondary);
}

.popover-menu {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.popover-item {
  width: 100%;
  height: 36px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border: none;
  background: transparent;
  border-radius: 8px;
  color: var(--el-text-color-regular);
  cursor: pointer;
  text-align: left;
  transition: all 0.15s ease;
}

.popover-item:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}

.popover-item--active {
  background: rgba(59, 130, 246, 0.1);
  color: #2563eb;
  font-weight: 600;
}

.popover-item-icon {
  font-size: 16px;
}
</style>
