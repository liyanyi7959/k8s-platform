<template>
  <div class="empty-state" :class="`empty-state--${type}`">
    <div class="empty-state-illustration">
      <!-- 通用空状态 SVG 插画 -->
      <svg viewBox="0 0 200 160" fill="none" xmlns="http://www.w3.org/2000/svg" class="empty-state-svg">
        <!-- 底部椭圆阴影 -->
        <ellipse cx="100" cy="140" rx="70" ry="10" :fill="colors.shadow" />

        <!-- 主体矩形（文件/盒子） -->
        <rect x="55" y="40" width="90" height="80" rx="8" :fill="colors.cardBg" :stroke="colors.cardBorder" stroke-width="1.5" />

        <!-- 装饰线条 -->
        <rect x="72" y="62" width="56" height="5" rx="2.5" :fill="colors.line1" />
        <rect x="72" y="76" width="40" height="5" rx="2.5" :fill="colors.line2" />
        <rect x="72" y="90" width="48" height="5" rx="2.5" :fill="colors.line2" />

        <!-- 右上角装饰图标 -->
        <circle v-if="type === 'empty'" cx="135" cy="45" r="16" :fill="colors.accent" opacity="0.15" />
        <template v-if="type === 'empty'">
          <line x1="130" y1="45" x2="140" y2="45" :stroke="colors.accent" stroke-width="2" stroke-linecap="round" />
          <line x1="135" y1="40" x2="135" y2="50" :stroke="colors.accent" stroke-width="2" stroke-linecap="round" />
        </template>

        <template v-if="type === 'no-data'">
          <circle cx="135" cy="45" r="16" :fill="colors.accent" opacity="0.15" />
          <path d="M128 45h14M135 38v14" :stroke="colors.accent" stroke-width="2" stroke-linecap="round" opacity="0.4" />
          <line x1="128" y1="45" x2="142" y2="45" :stroke="colors.accent" stroke-width="2.5" stroke-linecap="round" />
        </template>

        <template v-if="type === 'no-result'">
          <circle cx="135" cy="42" r="12" :stroke="colors.accent" stroke-width="2" fill="none" />
          <line x1="144" y1="50" x2="150" y2="56" :stroke="colors.accent" stroke-width="2" stroke-linecap="round" />
        </template>

        <template v-if="type === 'error'">
          <circle cx="135" cy="45" r="16" fill="rgba(239,68,68,0.12)" />
          <line x1="130" y1="40" x2="140" y2="50" stroke="#ef4444" stroke-width="2" stroke-linecap="round" />
          <line x1="140" y1="40" x2="130" y2="50" stroke="#ef4444" stroke-width="2" stroke-linecap="round" />
        </template>

        <!-- 左边小装饰 -->
        <circle cx="50" cy="65" r="4" :fill="colors.dot1" opacity="0.4" />
        <circle cx="44" cy="85" r="3" :fill="colors.dot2" opacity="0.3" />
        <circle cx="156" cy="90" r="3.5" :fill="colors.dot1" opacity="0.35" />
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

const isArtV2 = computed(() => typeof document !== 'undefined' && document.body.classList.contains('art-v2-shell'))

const colors = computed(() => {
  const dark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
  if (isArtV2.value) {
    if (dark) {
      return {
        shadow:     'rgba(2,6,23,0.28)',
        cardBg:     'rgba(15,23,42,0.94)',
        cardBorder: 'rgba(148,163,184,0.18)',
        line1:      'rgba(96,165,250,0.26)',
        line2:      'rgba(148,163,184,0.14)',
        accent:     '#60a5fa',
        dot1:       '#60a5fa',
        dot2:       '#22d3ee',
      }
    }
    return {
      shadow:     'rgba(15,23,42,0.08)',
      cardBg:     'rgba(255,255,255,0.98)',
      cardBorder: 'rgba(148,163,184,0.14)',
      line1:      'rgba(59,130,246,0.18)',
      line2:      'rgba(148,163,184,0.1)',
      accent:     '#3b82f6',
      dot1:       '#3b82f6',
      dot2:       '#06b6d4',
    }
  }
  if (dark) {
    return {
      shadow:     'rgba(0,0,0,0.2)',
      cardBg:     '#1e293b',
      cardBorder: 'rgba(148,163,184,0.15)',
      line1:      'rgba(148,163,184,0.25)',
      line2:      'rgba(148,163,184,0.12)',
      accent:     '#818cf8',
      dot1:       '#818cf8',
      dot2:       '#22d3ee',
    }
  }
  return {
    shadow:     'rgba(2,6,23,0.06)',
    cardBg:     '#ffffff',
    cardBorder: 'rgba(2,6,23,0.08)',
    line1:      'rgba(2,6,23,0.12)',
    line2:      'rgba(2,6,23,0.06)',
    accent:     '#6366f1',
    dot1:       '#6366f1',
    dot2:       '#06b6d4',
  }
})
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  padding: 32px 16px;
  max-width: 360px;
  margin: 0 auto;
}

.empty-state-illustration {
  position: relative;
}

.empty-state-svg {
  width: 160px;
  height: 128px;
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
}

.empty-state-action {
  margin-top: 4px;
}
</style>
