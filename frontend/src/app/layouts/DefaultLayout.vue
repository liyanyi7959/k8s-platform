<template>
  <div class="flex h-screen w-full overflow-hidden flex-col">
    <template v-if="isStandalone">
      <AppContent class="min-h-0 flex-1" full>
        <RouterView :key="viewKey" />
      </AppContent>
    </template>
    <template v-else>
      <!-- Top Header -->
      <AppHeader @refresh="refreshView" />
      
      <!-- Main Content Area -->
      <div class="flex flex-1 overflow-hidden relative">
        <AppContent class="min-h-0 flex-1">
          <RouterView :key="viewKey" />
        </AppContent>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppContent from './AppContent.vue'
import AppHeader from './AppHeader.vue'

const route = useRoute()
const viewKey = ref(0)

const isStandalone = computed(() => Boolean(route.meta?.standalone))

function refreshView() {
  viewKey.value += 1
}
</script>
