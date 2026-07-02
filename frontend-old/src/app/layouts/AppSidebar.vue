<template>
  <aside
    v-if="sidebarItems.length > 0"
    :class="[
      collapsed ? 'w-[72px]' : 'w-[240px]',
      'sidebar'
    ]"
  >
    <div
      v-if="!collapsed && activeGroup?.key === 'k8s' && clustersTotal !== null && !hasClusters"
      class="rounded-2xl border border-amber-300/30 bg-amber-50/70 p-3 text-slate-900 mx-3 mt-3 mb-2"
    >
      <div class="text-sm font-bold">先创建集群</div>
      <div class="mt-1 text-xs text-slate-600">导入 kubeconfig 后才可在集群管理页进入资源浏览与运维操作。</div>
      <div class="mt-2">
        <el-button size="small" type="primary" @click="goClusters">去创建</el-button>
      </div>
    </div>

    <el-scrollbar class="flex-1 min-h-0" @scroll="onNavScroll">
      <div v-if="collapsed" class="collapsed-nav">
        <el-tooltip v-for="item in sidebarItems" :key="item.path" :content="item.title" placement="right">
          <button
            :class="['nav-icon', item.path === activeMenu ? 'nav-icon-active' : '']"
            type="button"
            @click="goPath(item.path)"
          >
            <el-icon><component :is="item.icon" /></el-icon>
          </button>
        </el-tooltip>
      </div>

      <el-menu v-else :default-active="activeMenu" router class="menu">
        <template v-for="item in sidebarItems" :key="item.path">
          <el-sub-menu v-if="item.children && item.children.length > 0" :index="item.path">
            <template #title>
              <el-icon class="menu-icon"><component :is="item.icon" /></el-icon>
              <span>{{ item.title }}</span>
            </template>
            <el-menu-item v-for="sub in item.children" :key="sub.path" :index="sub.path">
              <el-icon class="menu-icon"><component :is="sub.icon" /></el-icon>
              <span>{{ sub.title }}</span>
            </el-menu-item>
          </el-sub-menu>
          <el-menu-item v-else :index="item.path">
            <el-icon class="menu-icon"><component :is="item.icon" /></el-icon>
            <span>{{ item.title }}</span>
          </el-menu-item>
        </template>
      </el-menu>
    </el-scrollbar>
  </aside>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMenu } from '@/app/composables/useMenu'
import * as clustersApi from '@/features/clusters/api/clusters'

const props = defineProps<{ collapsed: boolean }>()

const route = useRoute()
const router = useRouter()
const { sidebarItems, activeGroup } = useMenu()

const clustersTotal = ref<number | null>(null)

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

const hasClusters = computed(() => (clustersTotal.value ?? 0) > 0)

const activeMenu = computed(() => {
  const p = route.path
  if (p.startsWith('/admin/')) return p
  if (p.startsWith('/apps/')) return p
  return p
})

function goClusters() {
  router.push('/clusters')
}

function goPath(path: string) {
  router.push(path)
}

function onNavScroll() {
  // Optional: handle scroll
}
</script>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: transparent; /* sidebar bg is transparent, shows body bg or layout bg */
  transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  border-right: 1px solid rgba(255, 255, 255, 0.1); /* light separator */
  position: relative;
  z-index: 10;
}

.menu {
  border-right: none;
  background-color: transparent;
  padding: 8px;
}

:deep(.el-menu) {
  background-color: transparent;
  border-right: none;
}

:deep(.el-menu-item), :deep(.el-sub-menu__title) {
  height: 42px;
  line-height: 42px;
  border-radius: 8px;
  margin-bottom: 4px;
  color: #64748b;
  font-size: 15px; /* Increased from 14px */
}

:deep(.el-menu-item:hover), :deep(.el-sub-menu__title:hover) {
  background-color: rgba(255, 255, 255, 0.5);
  color: #334155;
}

:deep(.el-menu-item.is-active) {
  background: linear-gradient(90deg, rgba(59, 130, 246, 0.1), rgba(147, 197, 253, 0.1));
  color: #2563eb;
  font-weight: 500;
}

.menu-icon {
  margin-right: 8px;
  font-size: 18px;
}

/* Collapsed state styles */
.collapsed-nav {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 16px;
  gap: 8px;
}

.nav-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  color: #64748b;
  font-size: 20px;
  cursor: pointer;
  border: none;
  background: transparent;
  transition: all 0.2s;
}

.nav-icon:hover {
  background-color: rgba(255, 255, 255, 0.5);
  color: #334155;
}

.nav-icon-active {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: white;
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.3);
}

.nav-icon-active:hover {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
  color: white;
}
</style>
