<template>
  <div class="app-header-tabs">
    <el-scrollbar class="tabs-scroll" wrap-class="tabs-wrap">
      <div class="tabs-row">
        <div
          v-for="t in tabs"
          :key="t.path"
          :class="['tab-item', t.path === activePath ? 'active' : '']"
          @click="goTab(t.path)"
        >
          <span class="tab-icon"><el-icon><component :is="t.icon || Document" /></el-icon></span>
          <span class="tab-title">{{ t.title }}</span>
          <span v-if="t.closable" class="tab-close" @click.stop="closeTab(t.path)">
            <el-icon><Close /></el-icon>
          </span>
          <div class="active-indicator" v-if="t.path === activePath"></div>
        </div>
      </div>
    </el-scrollbar>

    <div class="tabs-actions">
      <el-dropdown trigger="click" @command="onTabsCommand">
        <button class="icon-btn-tiny">
          <el-icon><ArrowDown /></el-icon>
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="closeOther">关闭其他</el-dropdown-item>
            <el-dropdown-item command="closeAll">关闭所有</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowDown, Close, Document, Monitor
} from '@element-plus/icons-vue'
import k8sLogoUrl from '@/assets/images/k8s-official-icon.svg'

const K8sLogoIcon = defineComponent({
  name: 'K8sLogoIcon',
  setup() {
    return () =>
      h('img', {
        src: k8sLogoUrl,
        alt: 'Kubernetes',
        style: { width: '1em', height: '1em', display: 'block' }
      })
  }
})

type TabItem = { path: string; title: string; closable: boolean; icon?: any }

const emit = defineEmits<{ (e: 'refresh'): void }>()

const route = useRoute()
const router = useRouter()

const TABS_KEY = 'k8s-platform:tabs'
const tabs = ref<TabItem[]>([])

const activePath = computed(() => route.path)

function getIconForPath(path: string) {
  if (path === '/' || path === '/clusters') return K8sLogoIcon
  if (path.startsWith('/clusters')) return K8sLogoIcon
  if (path.startsWith('/k8s')) return Monitor
  return Document
}

function ensureRootTab() {
  const idx = tabs.value.findIndex((t) => t.path === '/clusters')
  if (idx < 0) {
    tabs.value.unshift({ path: '/clusters', title: '集群管理', closable: false, icon: K8sLogoIcon })
    return
  }
  tabs.value[idx].title = '集群管理'
  tabs.value[idx].closable = false
  tabs.value[idx].icon = K8sLogoIcon
}

function persistTabs() {
  localStorage.setItem(TABS_KEY, JSON.stringify(tabs.value))
}

function restoreTabs() {
  try {
    const raw = localStorage.getItem(TABS_KEY)
    if (!raw) return
    const parsed = JSON.parse(raw) as TabItem[]
    if (!Array.isArray(parsed)) return
    tabs.value = parsed.map(t => ({
      ...t,
      icon: getIconForPath(t.path)
    }))
  } catch {
    tabs.value = []
  } finally {
    ensureRootTab()
  }
}

function upsertTab(path: string, title: string) {
  if (!title) return
  const icon = getIconForPath(path)
  const existed = tabs.value.find((t) => t.path === path)
  if (existed) {
    existed.title = title
    existed.icon = icon
    return
  }
  tabs.value.push({ path, title, closable: path !== '/clusters', icon })
}

async function goTab(path: string) {
  if (path === activePath.value) return
  await router.push(path)
}

async function closeTab(path: string) {
  if (path === '/clusters') return
  const idx = tabs.value.findIndex((t) => t.path === path)
  if (idx < 0) return
  tabs.value.splice(idx, 1)
  ensureRootTab()
  persistTabs()

  if (activePath.value !== path) return
  const next = tabs.value[Math.min(idx, tabs.value.length - 1)] ?? tabs.value[0]
  if (next) await router.push(next.path)
}

async function closeOthers() {
  const cur = activePath.value
  tabs.value = tabs.value.filter((t) => t.path === '/clusters' || t.path === cur)
  ensureRootTab()
  persistTabs()
}

async function closeAll() {
  tabs.value = [{ path: '/clusters', title: '集群管理', closable: false, icon: K8sLogoIcon }]
  persistTabs()
  await router.push('/clusters')
}

async function onTabsCommand(cmd: string) {
  if (cmd === 'refresh') {
    emit('refresh')
    return
  }
  if (cmd === 'closeOther') {
    await closeOthers()
    return
  }
  if (cmd === 'closeAll') {
    await closeAll()
  }
}

onMounted(() => {
  restoreTabs()
  upsertTab(route.path, String(route.meta.title ?? ''))
  ensureRootTab()
  persistTabs()
})

watch(
  () => route.path,
  () => {
    upsertTab(route.path, String(route.meta.title ?? ''))
    ensureRootTab()
    persistTabs()
  }
)
</script>

<style scoped>
/* ── Tab bar: 44px, flush with header ── */
.app-header-tabs {
  height: 44px;
  display: flex;
  align-items: stretch;
  padding: 0 24px;
  position: relative;
  z-index: 10;
}

.tabs-scroll {
  flex: 1;
  margin-right: 12px;
}

.tabs-row {
  display: flex;
  align-items: stretch;
  gap: 0;
  height: 44px;
}

/* ── Tab item: text + bottom indicator ── */
.tab-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 16px;
  font-size: 13px;
  cursor: pointer;
  user-select: none;
  flex-shrink: 0;
  white-space: nowrap;
  color: #64748b;
  background: transparent;
  border: none;
  transition: color var(--duration-fast) var(--ease-default);
}
html.dark .tab-item {
  color: #94a3b8;
}

.tab-title {
  font-weight: 500;
}

.tab-item:hover {
  color: #334155;
}
html.dark .tab-item:hover {
  color: #e2e8f0;
}

.tab-item.active {
  color: var(--el-color-primary, #0284c7);
  font-weight: 600;
}
html.dark .tab-item.active {
  color: #38bdf8;
}

/* ── Active indicator: 2px bottom bar ── */
.active-indicator {
  display: block;
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: 0;
  height: 2px;
  border-radius: 2px 2px 0 0;
  background: var(--el-color-primary, #0284c7);
  transition: background var(--duration-fast) var(--ease-default);
}
html.dark .active-indicator {
  background: #38bdf8;
}

/* ── Icon ── */
.tab-icon {
  display: flex;
  align-items: center;
  font-size: 14px;
  color: inherit;
  opacity: 0.7;
  transition: opacity var(--duration-fast) var(--ease-default);
}
.tab-item.active .tab-icon {
  opacity: 1;
}

/* ── Close button ── */
.tab-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  font-size: 12px;
  color: #94a3b8;
  transition: all var(--duration-fast) var(--ease-default);
  margin-left: 4px;
  opacity: 0;
}
.tab-item:hover .tab-close {
  opacity: 1;
}
.tab-close:hover {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

/* ── Tab actions dropdown ── */
.tabs-actions {
  display: flex;
  align-items: center;
  padding-left: 12px;
}

.icon-btn-tiny {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  border-radius: 6px;
  transition: all var(--duration-fast) var(--ease-default);
}
.icon-btn-tiny:hover {
  background: rgba(0, 0, 0, 0.04);
  color: #475569;
}
html.dark .icon-btn-tiny:hover {
  background: rgba(255, 255, 255, 0.06);
  color: #e2e8f0;
}
</style>
