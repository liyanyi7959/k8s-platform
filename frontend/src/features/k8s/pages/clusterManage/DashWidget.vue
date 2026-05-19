<template>
  <el-card class="dash-widget" shadow="never">
    <template #header>
      <div class="widget-header">
        <div class="widget-head-left">
          <div class="widget-title">{{ title }}</div>
          <div v-if="sub" class="widget-sub">{{ sub }}</div>
        </div>
        <el-space v-if="$slots.tags" :size="6">
          <slot name="tags" />
        </el-space>
      </div>
    </template>
    <div v-if="editMode" class="dash-drag-handle" />
    <slot />
  </el-card>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  sub?: string
  editMode?: boolean
}>()
</script>

<style scoped>
.dash-widget {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 10px;
  border: 1px solid var(--k8s-card-border);
  background: var(--color-bg-card);
  transition: box-shadow 0.2s, border-color 0.2s;
}

.dash-widget.is-never-shadow {
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.03);
}

.dash-widget:hover {
  border-color: color-mix(in srgb, var(--color-accent-primary) 20%, var(--k8s-card-border));
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06) !important;
}

:global(html.dark) .dash-widget {
  background: rgba(15, 23, 42, 0.5);
}

:global(html.dark) .dash-widget.is-never-shadow {
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

:global(html.dark) .dash-widget:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3) !important;
}

.dash-widget :deep(> .el-card__header) {
  padding: 12px 16px;
  border-bottom: none;
  background: transparent;
}

.dash-widget :deep(> .el-card__body) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 0 16px 16px;
}

.widget-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.widget-head-left {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.widget-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--app-title);
  letter-spacing: -0.01em;
}

.widget-sub {
  font-size: 11px;
  color: var(--app-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 400px;
  margin-top: 2px;
  opacity: 0.8;
}

.widget-header :deep(.el-tag) {
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.dash-drag-handle {
  height: 8px;
  margin: 0 0 6px;
  border-radius: 999px;
  background: var(--k8s-card-border);
  cursor: grab;
}

@media (max-width: 768px) {
  .widget-sub {
    max-width: 220px;
  }
}
</style>
