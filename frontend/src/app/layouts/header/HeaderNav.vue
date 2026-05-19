<template>
  <nav class="top-nav hidden lg:flex">
    <div
      v-for="group in topNavGroups"
      :key="group.key"
      :class="['nav-item', (activeGroup?.key === group.key || hoveredGroup?.key === group.key) ? 'active' : '']"
      @click="handleGroupClick(group)"
      @mouseenter="hoverGroup(group)"
      @mouseleave="handleGroupLeave"
    >
      <el-icon class="nav-icon" v-if="group.icon"><component :is="group.icon" /></el-icon>
      <span class="nav-title">{{ group.title }}</span>
    </div>
  </nav>

  <!-- Mega Menu Overlay -->
  <transition name="fade">
    <div
      v-show="hoveredGroup && hoveredGroup.children && hoveredGroup.children.length > 0"
      class="mega-menu-overlay"
      @mouseenter="cancelLeave"
      @mouseleave="handleGroupLeave"
    >
      <div class="mega-menu-content">
        <div class="mega-menu-grid">
          <div
            v-for="child in hoveredGroup?.children"
            :key="child.path"
            class="mega-menu-item"
            @click.stop="handleChildClick(child)"
          >
            <div class="item-icon-wrapper">
              <el-icon class="item-icon"><component :is="child.icon" /></el-icon>
            </div>
            <div class="item-info">
              <div class="item-title">{{ child.title }}</div>
              <div class="item-desc" v-if="child.desc">{{ child.desc }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMenu, type NavGroup } from '@/app/composables/useMenu'

const router = useRouter()
const { railGroups: topNavGroups, activeGroup } = useMenu()

const hoveredGroup = ref<NavGroup | null>(null)
let leaveTimer: ReturnType<typeof setTimeout> | null = null

function hoverGroup(group: NavGroup) {
  if (leaveTimer) clearTimeout(leaveTimer)
  hoveredGroup.value = group
}

function handleGroupLeave() {
  leaveTimer = setTimeout(() => {
    hoveredGroup.value = null
  }, 150)
}

function cancelLeave() {
  if (leaveTimer) clearTimeout(leaveTimer)
}

function handleChildClick(child: any) {
  router.push(child.path)
  hoveredGroup.value = null
}

function handleGroupClick(group: NavGroup) {
  if (group.path) {
    router.push(group.path)
  }
}
</script>

<style scoped>
.top-nav {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 100%;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 38px;
  padding: 0 16px;
  border-radius: 10px;
  cursor: pointer;
  color: #64748b;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
.nav-item:hover {
  background: rgba(0, 0, 0, 0.04);
  color: #334155;
}
.nav-item.active {
  background: white;
  color: #3b82f6;
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 1px 2px -1px rgba(0, 0, 0, 0.1);
}
html.dark .nav-item {
  color: #94a3b8;
}
html.dark .nav-item:hover {
  background: rgba(255, 255, 255, 0.05);
  color: #f1f5f9;
}
html.dark .nav-item.active {
  background: rgba(30, 41, 59, 0.8);
  color: #60a5fa;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

/* Mega Menu Styles */
.mega-menu-overlay {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  width: 100%;
  background: #ffffff;
  border: 1px solid rgba(0, 0, 0, 0.05);
  border-radius: 16px;
  box-shadow: 0 8px 24px -8px rgba(0, 0, 0, 0.12);
  z-index: 9999;
  padding: 24px 0 32px;
}
html.dark .mega-menu-overlay {
  background: #1e293b;
  border-color: rgba(255, 255, 255, 0.05);
  box-shadow: 0 8px 24px -8px rgba(0, 0, 0, 0.4);
}
.mega-menu-content {
  width: 100%;
  padding: 0 24px;
}
.mega-menu-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 24px;
}
.mega-menu-item {
  display: flex;
  align-items: flex-start;
  padding: 16px;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid transparent;
  background: transparent;
}
.mega-menu-item:hover {
  background: #f8fafc;
  border-color: rgba(0, 0, 0, 0.05);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.03);
  transform: translateY(-1px);
}
html.dark .mega-menu-item:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: rgba(255, 255, 255, 0.05);
}
.item-icon-wrapper {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  flex-shrink: 0;
  color: #0284c7;
  transition: background var(--duration-normal) var(--ease-default), color var(--duration-normal) var(--ease-default);
}
html.dark .item-icon-wrapper {
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.15), rgba(2, 132, 199, 0.1));
}
.mega-menu-item:hover .item-icon-wrapper {
  background: linear-gradient(135deg, #0ea5e9 0%, #0284c7 100%);
  color: white;
  box-shadow: 0 8px 16px -4px rgba(2, 132, 199, 0.3);
}
.item-icon {
  font-size: 24px;
}
.item-info {
  flex: 1;
}
.item-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 6px;
}
html.dark .item-title {
  color: #f1f5f9;
}
.item-desc {
  font-size: 13px;
  color: #64748b;
  line-height: 1.5;
}
html.dark .item-desc {
  color: #94a3b8;
}

/* Transition */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s, transform 0.2s;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
