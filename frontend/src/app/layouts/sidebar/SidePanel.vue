<template>
  <transition name="panel-slide">
    <aside v-show="isPanelVisible && hasItems" class="side-panel">
      <!-- 面板 Header -->
      <div class="panel-header">
        <span class="panel-title">{{ activeGroup?.title ?? '' }}</span>
        <button
          class="pin-btn"
          :title="pinned ? '取消固定' : '固定面板'"
          @click="togglePin"
        >
          <el-icon>
            <component :is="pinned ? pinIcon : unpinIcon" />
          </el-icon>
        </button>
      </div>

      <!-- 集群提示 (资源管理分组特有) -->
      <div
        v-if="activeGroup?.key === 'k8s' && clustersTotal !== null && !hasClusters"
        class="cluster-hint"
      >
        <div class="hint-title">先创建集群</div>
        <div class="hint-desc">导入 kubeconfig 后才可进入资源浏览与运维操作。</div>
        <el-button size="small" type="primary" class="mt-2" @click="router.push('/clusters')">
          去创建
        </el-button>
      </div>

      <!-- 菜单列表 -->
      <el-scrollbar class="panel-scroll">
        <div class="panel-menu">
          <button
            v-for="item in sidebarItems"
            :key="item.path"
            :class="[
              'menu-item',
              isActive(item.path) ? 'menu-item--active' : ''
            ]"
            type="button"
            @click="router.push(item.path)"
          >
            <el-icon class="menu-item-icon"><component :is="item.icon" /></el-icon>
            <div class="menu-item-content">
              <span class="menu-item-title">{{ item.title }}</span>
              <span v-if="item.desc" class="menu-item-desc">{{ item.desc }}</span>
            </div>
          </button>

          <div v-if="activeGroup?.key === 'k8s'" class="cluster-shortcuts">
            <div class="section-title-row">
              <span class="section-title">集群快捷入口</span>
              <span v-if="shortcuts.length" class="section-count">{{ shortcuts.length }}</span>
            </div>

            <div v-if="shortcuts.length" class="shortcut-list">
              <div
                v-for="shortcut in shortcuts"
                :key="shortcut.id"
                :class="['shortcut-row', isShortcutActive(shortcut.id) ? 'shortcut-row--active' : '']"
                @contextmenu.prevent="removeShortcut(shortcut.id)"
              >
                <button class="shortcut-open" type="button" @click="openShortcut(shortcut)">
                  <span :class="['shortcut-status', shortcut.status ? `shortcut-status--${shortcut.status}` : '']" />
                  <span class="shortcut-name">{{ shortcut.name }}</span>
                  <span v-if="shortcut.nodeCount != null" class="shortcut-meta">{{ shortcut.nodeCount }} 节点</span>
                </button>
                <button class="shortcut-remove" type="button" title="移除快捷入口" @click="removeShortcut(shortcut.id)">
                  <el-icon><Close /></el-icon>
                </button>
              </div>
            </div>

            <div v-else class="shortcut-empty">
              在集群列表右键集群，或点击星标固定到这里。
            </div>
          </div>
        </div>
      </el-scrollbar>
    </aside>
  </transition>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, defineComponent, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMenu } from '@/app/composables/useMenu'
import { useSidebarState } from '@/app/composables/useSidebarState'
import { Close } from '@element-plus/icons-vue'
import { getClusterUnavailableMessage, useClusterShortcuts, type ClusterShortcut } from '@/app/composables/useClusterShortcuts'
import { notifyError } from '@/shared/utils/notify'
import * as clustersApi from '@/features/clusters/api/clusters'

/* 简易 Pin / Unpin 图标 */
const pinIcon = defineComponent({
  name: 'PinIcon',
  setup() {
    return () => h('svg', {
      viewBox: '0 0 24 24', width: '1em', height: '1em', fill: 'none',
      stroke: 'currentColor', 'stroke-width': '2', 'stroke-linecap': 'round', 'stroke-linejoin': 'round'
    }, [
      h('path', { d: 'M12 17v5' }),
      h('path', { d: 'M9 2h6l-1 7h-4L9 2z' }),
      h('path', { d: 'M5 9h14l-1 3H6L5 9z' })
    ])
  }
})
const unpinIcon = defineComponent({
  name: 'UnpinIcon',
  setup() {
    return () => h('svg', {
      viewBox: '0 0 24 24', width: '1em', height: '1em', fill: 'none',
      stroke: 'currentColor', 'stroke-width': '2', 'stroke-linecap': 'round', 'stroke-linejoin': 'round'
    }, [
      h('path', { d: 'M12 17v5' }),
      h('path', { d: 'M9 2h6l-1 7h-4L9 2z' }),
      h('path', { d: 'M5 9h14l-1 3H6L5 9z' }),
      h('line', { x1: '2', y1: '2', x2: '22', y2: '22' })
    ])
  }
})

const route = useRoute()
const router = useRouter()
const { activeGroup, sidebarItems } = useMenu()
const { isPanelVisible, pinned, togglePin } = useSidebarState()
const { shortcuts, unpinCluster: removeShortcut } = useClusterShortcuts()

const hasItems = computed(() => sidebarItems.value.length > 0)

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(path + '/')
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

/* ── 集群数量检测（资源管理分组） ──────────────────────────────────────── */
const clustersTotal = ref<number | null>(null)
const hasClusters = computed(() => (clustersTotal.value ?? 0) > 0)

async function refreshClustersTotal() {
  try {
    const data = await clustersApi.listClusters({ page: 1, page_size: 1 })
    clustersTotal.value = data.total ?? data.list.length
  } catch {
    clustersTotal.value = 0
  }
}

function onClustersChanged() {
  void refreshClustersTotal()
}

onMounted(() => {
  void refreshClustersTotal()
  window.addEventListener('k8s-platform:clusters-changed', onClustersChanged)
})

onUnmounted(() => {
  window.removeEventListener('k8s-platform:clusters-changed', onClustersChanged)
})
</script>

<style scoped>
/* ── 主题变量 — Light（默认） ─────────────────────────────────────────── */
.side-panel {
  --panel-bg:            var(--color-bg-sidebar, #f5f7fa);
  --panel-border:        var(--color-border-default, rgba(0, 0, 0, 0.06));
  --panel-title-color:   var(--color-text-primary, #0f172a);
  --panel-pin-color:     var(--color-text-muted, #94a3b8);
  --panel-pin-hover:     var(--color-text-primary, #0f172a);
  --panel-pin-hover-bg:  var(--color-bg-hover, rgba(0, 0, 0, 0.04));
  --panel-item-color:    var(--color-text-secondary, #475569);
  --panel-item-hover-bg: var(--color-bg-hover, rgba(0, 0, 0, 0.04));
  --panel-item-hover-color: var(--color-text-primary, #0f172a);
  --panel-item-active-bg:   rgba(59, 130, 246, 0.1);
  --panel-item-active-color: #2563eb;
  --panel-item-active-hover: rgba(59, 130, 246, 0.15);
  --panel-icon-active-color: #2563eb;
  --panel-desc-color:    var(--color-text-muted, #94a3b8);
  --panel-desc-active:   rgba(37, 99, 235, 0.62);
  --panel-hint-bg:       rgba(251, 191, 36, 0.08);
  --panel-hint-border:   rgba(251, 191, 36, 0.25);
  --panel-hint-title:    #d97706;
  --panel-hint-desc:     var(--color-text-secondary, #475569);
}

/* ── 主题变量 — Dark ─────────────────────────────────────────────────── */
:global(html.dark) .side-panel {
  --panel-bg:            var(--color-bg-sidebar, #111827);
  --panel-border:        rgba(255, 255, 255, 0.06);
  --panel-title-color:   rgba(226, 232, 240, 0.9);
  --panel-pin-color:     rgba(148, 163, 184, 0.6);
  --panel-pin-hover:     #e2e8f0;
  --panel-pin-hover-bg:  rgba(255, 255, 255, 0.08);
  --panel-item-color:    rgba(203, 213, 225, 0.8);
  --panel-item-hover-bg: rgba(255, 255, 255, 0.06);
  --panel-item-hover-color: #e2e8f0;
  --panel-item-active-bg:   rgba(59, 130, 246, 0.18);
  --panel-item-active-color: #93c5fd;
  --panel-item-active-hover: rgba(59, 130, 246, 0.24);
  --panel-icon-active-color: #60a5fa;
  --panel-desc-color:    rgba(148, 163, 184, 0.5);
  --panel-desc-active:   rgba(147, 197, 253, 0.68);
  --panel-hint-bg:       rgba(251, 191, 36, 0.1);
  --panel-hint-border:   rgba(251, 191, 36, 0.2);
  --panel-hint-title:    #fbbf24;
  --panel-hint-desc:     rgba(226, 232, 240, 0.6);
}

.side-panel {
  width: 200px;
  min-width: 200px;
  display: flex;
  flex-direction: column;
  background: var(--panel-bg);
  border-right: 1px solid var(--panel-border);
  z-index: 55;
  overflow: hidden;
  flex-shrink: 0;
  transition: background 0.25s ease, border-color 0.25s ease;
}

/* ── Slide transition ──────────────────────────────────────────────────── */
.panel-slide-enter-active,
.panel-slide-leave-active {
  transition: width 0.25s cubic-bezier(0.4, 0, 0.2, 1),
              min-width 0.25s cubic-bezier(0.4, 0, 0.2, 1),
              opacity 0.2s ease;
}
.panel-slide-enter-from,
.panel-slide-leave-to {
  width: 0;
  min-width: 0;
  opacity: 0;
}

/* ── Header ────────────────────────────────────────────────────────────── */
.panel-header {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--panel-border);
}
.panel-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--panel-title-color);
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pin-btn {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: var(--panel-pin-color);
  cursor: pointer;
  border-radius: 6px;
  transition: all 0.15s;
  flex-shrink: 0;
}
.pin-btn:hover {
  color: var(--panel-pin-hover);
  background: var(--panel-pin-hover-bg);
}

/* ── 集群提示卡片 ──────────────────────────────────────────────────────── */
.cluster-hint {
  margin: 12px 12px 0;
  padding: 12px;
  border-radius: 8px;
  background: var(--panel-hint-bg);
  border: 1px solid var(--panel-hint-border);
}
.hint-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--panel-hint-title);
}
.hint-desc {
  font-size: 12px;
  color: var(--panel-hint-desc);
  margin-top: 4px;
  line-height: 1.4;
}

/* ── Scroll ────────────────────────────────────────────────────────────── */
.panel-scroll {
  flex: 1;
  min-height: 0;
}

/* ── Menu ──────────────────────────────────────────────────────────────── */
.panel-menu {
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.menu-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  text-align: left;
  color: var(--panel-item-color);
  transition: all 0.15s cubic-bezier(0.4, 0, 0.2, 1);
  width: 100%;
}
.menu-item:hover {
  background: var(--panel-item-hover-bg);
  color: var(--panel-item-hover-color);
}
.menu-item--active {
  background: var(--panel-item-active-bg);
  color: var(--panel-item-active-color);
}
.menu-item--active:hover {
  background: var(--panel-item-active-hover);
}

.menu-item-icon {
  font-size: 16px;
  margin-top: 2px;
  flex-shrink: 0;
  opacity: 0.7;
}
.menu-item--active .menu-item-icon {
  opacity: 1;
  color: var(--panel-icon-active-color);
}

.menu-item-content {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.menu-item-title {
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.3;
}
.menu-item-desc {
  font-size: 11px;
  color: var(--panel-desc-color);
  margin-top: 2px;
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.menu-item--active .menu-item-desc {
  color: var(--panel-desc-active);
}

.cluster-shortcuts {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--panel-border);
}

.section-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 8px 8px;
  gap: 8px;
}

.section-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 700;
  color: var(--panel-title-color);
}

.section-count {
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 999px;
  background: rgba(59, 130, 246, 0.1);
  color: var(--panel-item-active-color);
  font-size: 11px;
  font-weight: 700;
  line-height: 20px;
  text-align: center;
}

.shortcut-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.shortcut-row {
  display: flex;
  align-items: center;
  gap: 4px;
  border-radius: 8px;
  transition: background 0.15s ease;
}

.shortcut-row:hover,
.shortcut-row--active {
  background: var(--panel-item-active-bg);
}

.shortcut-open {
  display: flex;
  min-width: 0;
  flex: 1;
  height: 38px;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: none;
  background: transparent;
  color: var(--panel-item-color);
  cursor: pointer;
  text-align: left;
}

.shortcut-row:hover .shortcut-open,
.shortcut-row--active .shortcut-open {
  color: var(--panel-item-active-color);
}

.shortcut-status {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #94a3b8;
  flex-shrink: 0;
}

.shortcut-status--active {
  background: #22c55e;
}

.shortcut-status--degraded,
.shortcut-status--creating {
  background: #f59e0b;
}

.shortcut-status--disabled,
.shortcut-status--deleting {
  background: #ef4444;
}

.shortcut-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 600;
}

.shortcut-meta {
  flex-shrink: 0;
  color: var(--panel-desc-color);
  font-size: 11px;
  font-weight: 600;
}

.shortcut-remove {
  display: flex;
  width: 28px;
  height: 28px;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--panel-pin-color);
  cursor: pointer;
  opacity: 0;
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.shortcut-row:hover .shortcut-remove,
.shortcut-row--active .shortcut-remove {
  opacity: 1;
}

.shortcut-remove:hover {
  background: rgba(239, 68, 68, 0.09);
  color: #ef4444;
}

.shortcut-empty {
  margin: 0 8px;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px dashed var(--panel-border);
  color: var(--panel-desc-color);
  font-size: 12px;
  line-height: 1.5;
}
</style>
