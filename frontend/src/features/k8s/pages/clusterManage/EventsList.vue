<template>
  <div class="events-list">
    <div v-if="events.length === 0" class="events-empty">暂无事件</div>
    <div v-for="(e, idx) in events" :key="eventKey(e, idx)" class="events-item">
      <el-tag size="small" :type="String(e?.type ?? '') === 'Warning' ? 'warning' : 'info'">{{ String(e?.type ?? '-') }}</el-tag>
      <div class="events-text">
        <div class="events-title">
          <span class="events-reason">{{ String(e?.reason ?? '-') }}</span>
          <span class="events-time">{{ String(e?.lastTimestamp ?? e?.eventTime ?? e?.firstTimestamp ?? '') }}</span>
        </div>
        <div class="events-sub">{{ String(e?.message ?? '-') }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  events: any[]
  eventKey: (e: any, idx: number) => string
}>()
</script>

<style scoped>
.events-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow: auto;
  padding: 2px 2px 8px;
}

.events-empty {
  color: var(--app-muted);
  font-size: 12px;
  font-weight: 600;
  padding: 20px 16px;
  text-align: center;
  border-radius: 10px;
  border: 1px dashed var(--k8s-card-border);
  background: var(--k8s-card-bg);
  opacity: 0.8;
}

.events-item {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 10px;
  align-items: start;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--k8s-card-border);
  background: var(--k8s-card-bg);
  transition: border-color 0.15s;
}

.events-item:hover {
  border-color: color-mix(in srgb, var(--app-accent-blue) 20%, transparent);
}

.events-title {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  font-weight: 900;
}

.events-sub {
  margin-top: 2px;
  font-size: 12px;
  color: var(--app-muted);
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
}

.events-time {
  font-weight: 700;
  color: var(--app-muted);
  white-space: nowrap;
}

:global(html.dark) .events-item {
  border-color: rgba(226, 232, 240, 0.10);
  background: rgba(2, 6, 23, 0.38);
}

:global(html.dark) .events-empty {
  border-color: rgba(226, 232, 240, 0.14);
  background: rgba(2, 6, 23, 0.22);
}
</style>
