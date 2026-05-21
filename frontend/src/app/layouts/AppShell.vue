<template>
  <div :class="['app-shell', isStandalone ? 'app-shell--standalone' : '']">
    <AppSidebar />

    <div class="shell-main">
      <div :class="['shell-topbar-host', isStandalone ? 'shell-topbar-host--collapsed' : '']">
        <TopBar />
      </div>

      <AppContent class="min-h-0 flex-1" :full="isFullContent">
        <RouterView />
      </AppContent>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppSidebar from './sidebar/AppSidebar.vue'
import TopBar from './TopBar.vue'
import AppContent from './AppContent.vue'

const route = useRoute()

const isStandalone = computed(() => Boolean(route.meta?.standalone))
const isFullContent = computed(() => isStandalone.value || Boolean(route.meta?.fullContent))

onMounted(() => {
  document.body.classList.add('art-v2-shell')
})

onBeforeUnmount(() => {
  document.body.classList.remove('art-v2-shell')
})
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

.shell-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.shell-topbar-host {
  height: 60px;
  min-height: 60px;
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
