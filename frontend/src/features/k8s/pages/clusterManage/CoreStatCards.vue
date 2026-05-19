<template>
  <div class="core-stats">
    <div v-for="key in statKeys" :key="key" :class="['core-stat'].concat(classFn(key))">
      <div class="stat-label">{{ getCard(key)?.label ?? '-' }}</div>
      <div class="stat-value">{{ getCard(key)?.value ?? '-' }}</div>
      <div class="stat-sub">{{ getCard(key)?.sub ?? '-' }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
type CardVm = { key: string; label: string; value: string | number; sub: string; glow: string }

defineProps<{
  getCard: (key: string) => CardVm | undefined
  classFn: (key: string) => string[]
}>()

const statKeys = ['status', 'nodes', 'pods', 'workloads'] as const
</script>

<style scoped>
.core-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.core-stat {
  position: relative;
  overflow: hidden;
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, var(--core-stat-value-color, var(--app-accent-blue)) 18%, var(--k8s-card-border));
  background: linear-gradient(135deg, var(--k8s-card-bg) 60%, color-mix(in srgb, var(--core-stat-value-color, var(--app-accent-blue)) 4%, transparent));
  padding: 18px 20px;
  min-height: 96px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: flex-start;
  text-align: left;
  transition: border-color 0.2s, box-shadow 0.2s, transform 0.2s;
}

/* Left accent bar */
.core-stat::before {
  content: '';
  position: absolute;
  left: 0;
  top: 12px;
  bottom: 12px;
  width: 3px;
  border-radius: 0 3px 3px 0;
  background: var(--core-stat-value-color, var(--app-accent-blue));
  opacity: 0.75;
  transition: opacity 0.2s;
}

/* Top-right decorative orb */
.core-stat::after {
  content: '';
  position: absolute;
  right: -20px;
  top: -20px;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: radial-gradient(circle, color-mix(in srgb, var(--core-stat-value-color, var(--app-accent-blue)) 12%, transparent), transparent 70%);
  pointer-events: none;
}

.core-stat:hover {
  border-color: color-mix(in srgb, var(--core-stat-value-color, var(--app-accent-blue)) 35%, transparent);
  box-shadow: 0 6px 20px color-mix(in srgb, var(--core-stat-value-color, var(--app-accent-blue)) 10%, transparent);
  transform: translateY(-1px);
}

.core-stat:hover::before {
  opacity: 1;
}

/* ── Color variants ── */
.core-stat--cyan { --core-stat-value-color: var(--app-accent-cyan); }
.core-stat--blue { --core-stat-value-color: var(--app-accent-blue); }
.core-stat--violet { --core-stat-value-color: var(--app-accent-violet); }
.core-stat--slate { --core-stat-value-color: var(--app-title); }
.core-stat--status-ok { --core-stat-value-color: var(--c-emerald-500); }
.core-stat--status-warn { --core-stat-value-color: var(--c-amber-500); }
.core-stat--status-bad { --core-stat-value-color: var(--c-red-500); }

/* ── Typography ── */
.stat-label {
  font-size: 12px;
  color: var(--app-muted);
  font-weight: 600;
  letter-spacing: 0.02em;
}

.stat-value {
  margin-top: 10px;
  font-size: 28px;
  font-weight: 900;
  letter-spacing: -0.5px;
  line-height: 1;
  color: var(--core-stat-value-color, var(--app-title));
}

.stat-sub {
  margin-top: 10px;
  font-size: 12px;
  color: var(--app-muted);
  letter-spacing: 0.01em;
  opacity: 0.85;
}

/* ── Dark mode ── */
:global(html.dark) .core-stat {
  background: linear-gradient(135deg, rgba(2, 6, 23, 0.5) 60%, color-mix(in srgb, var(--core-stat-value-color, var(--app-accent-blue)) 8%, transparent));
  border-color: color-mix(in srgb, var(--core-stat-value-color, var(--app-accent-blue)) 15%, rgba(226, 232, 240, 0.10));
}

:global(html.dark) .core-stat::before {
  opacity: 0.6;
}

:global(html.dark) .core-stat::after {
  opacity: 0.7;
}

/* ── Responsive ── */
@media (max-width: 1280px) {
  .core-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .core-stats {
    grid-template-columns: repeat(1, minmax(0, 1fr));
  }
}
</style>
