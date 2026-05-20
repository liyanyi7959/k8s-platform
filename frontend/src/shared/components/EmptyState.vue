<template>
  <div class="empty-state" :class="`empty-state--${type}`">
    <div class="empty-state-illustration" aria-hidden="true">
      <svg viewBox="0 0 200 160" fill="none" xmlns="http://www.w3.org/2000/svg" class="empty-state-svg">
        <ellipse cx="100" cy="136" rx="54" ry="9" :fill="colors.shadow" />

        <path d="M68 32h44l18 18v62c0 7.732-6.268 14-14 14H68c-7.732 0-14-6.268-14-14V46c0-7.732 6.268-14 14-14Z" :fill="colors.panelFill" :stroke="colors.panelStroke" stroke-width="1.5" />
        <path d="M112 32v18h18" :stroke="colors.panelStroke" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        <rect x="70" y="56" width="44" height="6" rx="3" :fill="colors.lineStrong" />
        <rect x="70" y="72" width="32" height="6" rx="3" :fill="colors.lineSoft" />
        <rect x="70" y="88" width="40" height="6" rx="3" :fill="colors.lineSoft" />

        <circle cx="140" cy="96" r="20" :fill="colors.badgeFill" :stroke="colors.badgeStroke" stroke-width="1.5" />

        <template v-if="type === 'error'">
          <line x1="140" y1="86" x2="140" y2="98" :stroke="colors.badgeIcon" stroke-width="2.4" stroke-linecap="round" />
          <circle cx="140" cy="104" r="1.8" :fill="colors.badgeIcon" />
        </template>
        <template v-else-if="type === 'no-result'">
          <circle cx="137" cy="93" r="7.5" :stroke="colors.badgeIcon" stroke-width="2" fill="none" />
          <line x1="142.5" y1="98.5" x2="149" y2="105" :stroke="colors.badgeIcon" stroke-width="2" stroke-linecap="round" />
        </template>
        <template v-else>
          <line x1="132" y1="96" x2="148" y2="96" :stroke="colors.badgeIcon" stroke-width="2.4" stroke-linecap="round" />
        </template>

        <circle cx="52" cy="64" r="3" :fill="colors.dot" />
        <circle cx="156" cy="60" r="2.5" :fill="colors.dot" />
        <circle cx="46" cy="92" r="2.5" :fill="colors.dot" />
      </svg>
    </div>

    <div class="empty-state-text">
      <div v-if="title" class="empty-state-title">{{ title }}</div>
      <div class="empty-state-desc">{{ description || defaultDesc }}</div>
    </div>

    <div v-if="$slots.default" class="empty-state-action">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  /** 空状态类型 */
  type?: 'empty' | 'no-data' | 'no-result' | 'error'
  /** 标题（可选） */
  title?: string
  /** 描述文字 */
  description?: string
}>()

const defaultDesc = computed(() => {
  switch (props.type) {
    case 'no-data':   return '暂无数据'
    case 'no-result': return '没有找到匹配的结果'
    case 'error':     return '加载失败，请稍后重试'
    default:          return '暂无内容'
  }
})

const colors = computed(() => {
  const dark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
  const isError = props.type === 'error'
  if (dark) {
    return {
      shadow: 'rgba(2, 6, 23, 0.28)',
      panelFill: 'rgba(15, 23, 42, 0.92)',
      panelStroke: 'rgba(148, 163, 184, 0.2)',
      lineStrong: 'rgba(226, 232, 240, 0.18)',
      lineSoft: 'rgba(148, 163, 184, 0.16)',
      badgeFill: isError ? 'rgba(127, 29, 29, 0.36)' : 'rgba(30, 41, 59, 0.94)',
      badgeStroke: isError ? 'rgba(252, 165, 165, 0.28)' : 'rgba(148, 163, 184, 0.22)',
      badgeIcon: isError ? '#fca5a5' : '#cbd5e1',
      dot: 'rgba(148, 163, 184, 0.32)',
    }
  }
  return {
    shadow: 'rgba(15, 23, 42, 0.08)',
    panelFill: '#ffffff',
    panelStroke: 'rgba(148, 163, 184, 0.18)',
    lineStrong: 'rgba(15, 23, 42, 0.12)',
    lineSoft: 'rgba(148, 163, 184, 0.16)',
    badgeFill: isError ? 'rgba(254, 242, 242, 0.96)' : 'rgba(248, 250, 252, 0.98)',
    badgeStroke: isError ? 'rgba(248, 113, 113, 0.22)' : 'rgba(148, 163, 184, 0.18)',
    badgeIcon: isError ? '#dc2626' : '#475569',
    dot: 'rgba(148, 163, 184, 0.28)',
  }
})
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 32px 16px;
  max-width: 380px;
  margin: 0 auto;
}

.empty-state-illustration {
  position: relative;
  width: 164px;
}

.empty-state-svg {
  width: 164px;
  height: 132px;
}

.empty-state-text {
  text-align: center;
}

.empty-state-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: 4px;
}

.empty-state-desc {
  font-size: 13px;
  color: var(--color-text-muted);
  line-height: 1.6;
  max-width: 24em;
}

.empty-state-action {
  margin-top: 4px;
  display: flex;
  justify-content: center;
}
</style>
