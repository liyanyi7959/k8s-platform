<template>
  <div class="pod-phase-summary-bar">
    <button
      v-for="item in cards"
      :key="item.key"
      type="button"
      :class="['pod-phase-summary-bar__item', `pod-phase-summary-bar__item--${item.tone}`, props.activeFilter === item.key ? 'is-active' : '']"
      @click="emit('change', props.activeFilter === item.key ? '' : item.key)"
    >
      <span class="pod-phase-summary-bar__headline">
        <span class="pod-phase-summary-bar__label">{{ item.label }}</span>
        <strong class="pod-phase-summary-bar__value">{{ item.count }}</strong>
      </span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type PodPhaseQuickFilter = '' | 'Running' | 'Pending' | 'Failed' | 'Completed'

const props = defineProps<{
  summary: Record<Exclude<PodPhaseQuickFilter, ''>, number>
  activeFilter: PodPhaseQuickFilter
}>()

const emit = defineEmits<{
  (e: 'change', value: PodPhaseQuickFilter): void
}>()

const cards = computed(() => [
  { key: 'Running' as const, label: 'Running', count: props.summary.Running ?? 0, tone: 'running' },
  { key: 'Pending' as const, label: 'Pending', count: props.summary.Pending ?? 0, tone: 'pending' },
  { key: 'Failed' as const, label: 'Failed', count: props.summary.Failed ?? 0, tone: 'failed' },
  { key: 'Completed' as const, label: 'Completed', count: props.summary.Completed ?? 0, tone: 'completed' }
])
</script>

<style scoped>
.pod-phase-summary-bar {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 14px;
}

.pod-phase-summary-bar__item {
  --phase-tone: #64748b;
  --phase-border: rgba(148, 163, 184, 0.22);
  --phase-ring: rgba(148, 163, 184, 0.12);
  display: flex;
  align-items: center;
  min-height: 66px;
  padding: 10px 14px;
  border-radius: 14px;
  border: 1px solid var(--phase-border);
  background: rgba(255, 255, 255, 0.96);
  color: #0f172a;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease, background 0.18s ease;
}

.pod-phase-summary-bar__item::before {
  content: '';
  position: absolute;
  inset: 0 0 auto;
  height: 3px;
  background: var(--phase-tone);
}

.pod-phase-summary-bar__item:hover {
  transform: translateY(-1px);
  box-shadow: 0 10px 22px rgba(15, 23, 42, 0.05);
}

.pod-phase-summary-bar__item.is-active {
  box-shadow: 0 0 0 3px var(--phase-ring);
}

.pod-phase-summary-bar__headline {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
}

.pod-phase-summary-bar__label {
  font-size: 13px;
  font-weight: 700;
  color: #64748b;
}

.pod-phase-summary-bar__value {
  font-size: 30px;
  line-height: 1;
  letter-spacing: -0.03em;
  color: var(--phase-tone);
}

.pod-phase-summary-bar__item--running {
  --phase-tone: #059669;
  --phase-border: rgba(5, 150, 105, 0.24);
  --phase-ring: rgba(5, 150, 105, 0.14);
}

.pod-phase-summary-bar__item--pending {
  --phase-tone: #d97706;
  --phase-border: rgba(217, 119, 6, 0.24);
  --phase-ring: rgba(217, 119, 6, 0.14);
}

.pod-phase-summary-bar__item--failed {
  --phase-tone: #dc2626;
  --phase-border: rgba(220, 38, 38, 0.24);
  --phase-ring: rgba(220, 38, 38, 0.14);
}

.pod-phase-summary-bar__item--completed {
  --phase-tone: #64748b;
  --phase-border: rgba(100, 116, 139, 0.24);
  --phase-ring: rgba(100, 116, 139, 0.14);
}

:global(html.dark) .pod-phase-summary-bar__item {
  background: rgba(15, 23, 42, 0.88);
  color: #e2e8f0;
}

:global(html.dark) .pod-phase-summary-bar__label {
  color: #cbd5e1;
}

:global(html.dark) .pod-phase-summary-bar__item--running {
  --phase-border: rgba(16, 185, 129, 0.26);
  --phase-ring: rgba(16, 185, 129, 0.18);
}

:global(html.dark) .pod-phase-summary-bar__item--pending {
  --phase-border: rgba(245, 158, 11, 0.28);
  --phase-ring: rgba(245, 158, 11, 0.18);
}

:global(html.dark) .pod-phase-summary-bar__item--failed {
  --phase-border: rgba(248, 113, 113, 0.28);
  --phase-ring: rgba(248, 113, 113, 0.18);
}

:global(html.dark) .pod-phase-summary-bar__item--completed {
  --phase-border: rgba(148, 163, 184, 0.28);
  --phase-ring: rgba(148, 163, 184, 0.18);
}

@media (max-width: 960px) {
  .pod-phase-summary-bar {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .pod-phase-summary-bar__headline {
    gap: 6px;
  }

  .pod-phase-summary-bar__value {
    font-size: 28px;
  }
}
</style>