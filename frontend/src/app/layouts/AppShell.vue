<template>
  <div :class="['app-shell', isStandalone ? 'app-shell--standalone' : '']">
    <SideRail />

    <div :class="['shell-panel-host', panelHostCollapsed ? 'shell-panel-host--collapsed' : '']">
      <SidePanel />
    </div>

    <div class="shell-main">
      <div :class="['shell-topbar-host', isStandalone ? 'shell-topbar-host--collapsed' : '']">
        <TopBar />
      </div>

      <AppContent class="min-h-0 flex-1" :full="isStandalone">
        <RouterView />
      </AppContent>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import SideRail from './sidebar/SideRail.vue'
import SidePanel from './sidebar/SidePanel.vue'
import TopBar from './TopBar.vue'
import AppContent from './AppContent.vue'
import { useMenu } from '@/app/composables/useMenu'
import { useSidebarState } from '@/app/composables/useSidebarState'

const route = useRoute()
const { hasSidebarItems } = useMenu()
const { isPanelVisible } = useSidebarState()

const isStandalone = computed(() => Boolean(route.meta?.standalone))
const panelHostCollapsed = computed(() => isStandalone.value || !hasSidebarItems.value || !isPanelVisible.value)
</script>

<style scoped>
.app-shell {
  display: flex;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: var(--color-bg-page, #f8fafc);
}
html.dark .app-shell {
  background: var(--color-bg-page, #0f172a);
}

.shell-panel-host {
  width: 200px;
  min-width: 200px;
  overflow: hidden;
  flex-shrink: 0;
  transition:
    width 0.24s cubic-bezier(0.4, 0, 0.2, 1),
    min-width 0.24s cubic-bezier(0.4, 0, 0.2, 1),
    opacity 0.18s ease;
}

.shell-panel-host--collapsed {
  width: 0;
  min-width: 0;
  opacity: 0;
  pointer-events: none;
}

.shell-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.shell-topbar-host {
  height: 48px;
  min-height: 48px;
  overflow: hidden;
  flex-shrink: 0;
  transition:
    height 0.24s cubic-bezier(0.4, 0, 0.2, 1),
    min-height 0.24s cubic-bezier(0.4, 0, 0.2, 1),
    opacity 0.18s ease;
}

.shell-topbar-host--collapsed {
  height: 0;
  min-height: 0;
  opacity: 0;
  pointer-events: none;
}

.app-shell--standalone .shell-main {
  border-left: 1px solid rgba(255, 255, 255, 0.04);
}
</style>
