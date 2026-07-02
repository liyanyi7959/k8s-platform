<template>
  <el-card class="page-card">
    <el-form v-if="$slots.filters" :inline="true" class="filters" @submit.prevent>
      <slot name="filters" :expanded="filtersExpanded" />
      <slot v-if="filtersExpanded" name="filters-extra" />
      <el-form-item v-if="hasReset || hasSearch || hasExpand" class="filters-actions">
        <el-space wrap>
          <el-button v-if="hasReset" :icon="RefreshLeft" @click="emit('reset')">重置</el-button>
          <el-button v-if="hasSearch" type="primary" :icon="Search" @click="emit('search')">查询</el-button>
          <el-button v-if="hasExpand" :icon="filtersExpanded ? Fold : Expand" @click="toggleExpand">{{ filtersExpanded ? '收起' : '展开' }}</el-button>
        </el-space>
      </el-form-item>
    </el-form>

    <div v-if="title || total !== undefined" class="title-row">
      <div class="title-group">
        <div class="title">{{ title }}</div>
        <div v-if="subtitle" class="subtitle">{{ subtitle }}</div>
      </div>
      <div v-if="total !== undefined" class="total-pill">{{ total }}条数据</div>
    </div>

    <div v-if="$slots.actions || $slots['actions-right']" class="actions-row">
      <div class="actions-left">
        <slot name="actions" />
      </div>
      <div class="actions-right">
        <slot name="actions-right" />
      </div>
    </div>

    <!-- 骨架屏：首次加载时替换默认 slot -->
    <SkeletonCard v-if="skeleton" variant="table" :rows="skeletonRows" :columns="skeletonColumns" />
    <slot v-else />
  </el-card>
</template>

<script setup lang="ts">
import { computed, ref, useSlots } from 'vue'
import { Expand, Fold, RefreshLeft, Search } from '@element-plus/icons-vue'
import SkeletonCard from '@/shared/components/SkeletonCard.vue'

const props = defineProps<{
  title?: string
  subtitle?: string
  total?: number
  collapsibleFilters?: boolean
  showReset?: boolean
  showSearch?: boolean
  /** 是否显示骨架屏（首次加载时传 true） */
  skeleton?: boolean
  /** 骨架屏行数，默认 8 */
  skeletonRows?: number
  /** 骨架屏列数，默认 5 */
  skeletonColumns?: number
}>()

const emit = defineEmits<{
  (e: 'search'): void
  (e: 'reset'): void
}>()

const slots = useSlots()
const filtersExpanded = ref(false)
const hasExtraFilters = computed(() => !!slots['filters-extra'])
const hasExpand = computed(() => !!props.collapsibleFilters && hasExtraFilters.value)
const hasReset = computed(() => props.showReset !== false)
const hasSearch = computed(() => props.showSearch !== false)

function toggleExpand() {
  filtersExpanded.value = !filtersExpanded.value
}
</script>

<style scoped>
.filters {
  @apply mb-3 rounded-md border border-dashed border-slate-200/70 bg-white/55 p-3 backdrop-blur-xl;
}

:global(html.dark) .filters {
  @apply border-slate-600/30 bg-slate-900/40;
}

.filters-actions {
  @apply mb-0;
}

.title-row {
  @apply mb-3 flex items-center justify-between gap-3;
}

.title-group {
  @apply flex items-baseline gap-3;
}

.title {
  @apply text-sm font-extrabold text-slate-900;
}

.subtitle {
  @apply text-xs text-slate-400;
}

:global(html.dark) .title {
  @apply text-slate-100;
}

:global(html.dark) .subtitle {
  @apply text-slate-500;
}

.total-pill {
  @apply inline-flex items-center rounded border border-emerald-200/60 bg-emerald-50/60 px-3 py-1 text-xs font-bold text-emerald-700;
}

:global(html.dark) .total-pill {
  @apply border-emerald-500/30 bg-emerald-500/10 text-emerald-200;
}

.actions-row {
  @apply mb-3 flex items-center justify-between gap-3;
}

.actions-left,
.actions-right {
  @apply flex min-w-0 items-center gap-2;
}
</style>
