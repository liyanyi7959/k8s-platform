<template>
  <header class="app-header">
    <div class="app-header-main">
      <div class="header-left">
        <HeaderBrand />
        <HeaderNav />
      </div>

      <div class="header-right">
        <HeaderSearch />
        <HeaderActions class="ml-4" @refresh="emit('refresh')" />
        <HeaderUserMenu />
      </div>
    </div>

    <HeaderTabs @refresh="emit('refresh')" />
  </header>
</template>

<script setup lang="ts">
import HeaderBrand from './header/HeaderBrand.vue'
import HeaderNav from './header/HeaderNav.vue'
import HeaderSearch from './header/HeaderSearch.vue'
import HeaderActions from './header/HeaderActions.vue'
import HeaderUserMenu from './header/HeaderUserMenu.vue'
import HeaderTabs from './header/HeaderTabs.vue'

const emit = defineEmits<{ (e: 'refresh'): void }>()
</script>

<style scoped>
/* ── Unified Header Container ── */
.app-header {
  display: flex;
  flex-direction: column;
  z-index: 50;
  position: relative;
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(20px) saturate(180%);
  border-bottom: 1px solid rgba(226, 232, 240, 0.6);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: background var(--duration-normal) var(--ease-default),
              border-color var(--duration-normal) var(--ease-default);
}
html.dark .app-header {
  background: rgba(15, 23, 42, 0.78);
  border-bottom-color: rgba(51, 65, 85, 0.5);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

/* ── Main navigation bar (56px) ── */
.app-header-main {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  gap: 24px;
  position: relative;
  z-index: 100;
}

/* ── Separator between header & tabs ── */
.app-header-main::after {
  content: '';
  position: absolute;
  left: 24px;
  right: 24px;
  bottom: 0;
  height: 1px;
  background: rgba(226, 232, 240, 0.5);
}
html.dark .app-header-main::after {
  background: rgba(51, 65, 85, 0.4);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 0 0 auto;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1 1 auto;
  min-width: 0;
  justify-content: flex-end;
}
</style>
