import { computed, ref, watch, type ComputedRef, type ShallowRef } from 'vue'

import type { K8sLikeObject, ResourceKey, SortOrder } from '@/features/k8s/pages/ClusterManageView.types'
import {
  computeNextNamespaceSelection,
  getListRowSearchText,
  isNamespacedResource,
  normalizeNamespaceSelection
} from '@/features/k8s/pages/ClusterManageView.utils'

export function useClusterManageViewState(options: {
  list: ShallowRef<K8sLikeObject[]>
  currentResource: ComputedRef<ResourceKey | undefined>
  namespaces: ComputedRef<string[]> | { value: string[] }
  allNamespace: string
  clearPodSelection: () => void
  extraFilter?: ComputedRef<((item: K8sLikeObject) => boolean) | undefined> | { value: ((item: K8sLikeObject) => boolean) | undefined }
}) {
  const sortBy = ref<string | undefined>(undefined)
  const order = ref<SortOrder | undefined>(undefined)
  const keywordInput = ref('')
  const keyword = ref('')
  const page = ref(1)
  const pageSize = ref(20)
  const pageSizeOptions = [20, 50, 100, 200]
  const namespace = ref<string[]>(['default'])
  const workloadLabelSelector = ref('')

  const showNamespaceSelect = computed(() => {
    const resource = options.currentResource.value
    if (!resource) return false
    return isNamespacedResource(resource)
  })

  let keywordDebounceTimer: number | null = null
  watch(keywordInput, (value) => {
    if (keywordDebounceTimer != null) window.clearTimeout(keywordDebounceTimer)
    keywordDebounceTimer = window.setTimeout(() => {
      keyword.value = String(value ?? '')
    }, 180)
  })

  const displayedList = computed(() => {
    const kw = keyword.value.trim().toLowerCase()
    const resource = options.currentResource.value
    const extraFilter = options.extraFilter?.value
    let rows = options.list.value
    if (kw) {
      rows = rows.filter((item) => {
        if (!item || typeof item !== 'object') return false
        return getListRowSearchText(item, resource).includes(kw)
      })
    }
    if (typeof extraFilter === 'function') {
      rows = rows.filter((item) => extraFilter(item))
    }
    return rows
  })

  const displayedTotal = computed(() => displayedList.value.length)
  const maxPage = computed(() => Math.max(1, Math.ceil(displayedTotal.value / Math.max(1, pageSize.value))))

  watch(
    () => displayedTotal.value,
    () => {
      if (page.value > maxPage.value) page.value = maxPage.value
    }
  )

  watch(
    () => keyword.value,
    () => {
      page.value = 1
    }
  )

  const pagedList = computed(() => {
    const start = (page.value - 1) * pageSize.value
    return displayedList.value.slice(start, start + pageSize.value)
  })

  const showPager = computed(() => {
    const resource = options.currentResource.value
    if (!resource || resource === 'dashboard' || resource === 'topology') return false
    return displayedTotal.value > 0
  })

  function onNamespaceSelectChange(nextValue: unknown) {
    const prev = normalizeNamespaceSelection(namespace.value, options.allNamespace)
    const next = computeNextNamespaceSelection(prev, nextValue, options.namespaces.value, options.allNamespace)
    if (prev.length === next.length && prev.every((item, index) => item === next[index])) return
    namespace.value = next
  }

  function onPageChange(nextPage: number) {
    const normalized = Number(nextPage) || 1
    page.value = Math.min(Math.max(1, normalized), maxPage.value)
    if (options.currentResource.value === 'pods') options.clearPodSelection()
  }

  function onPageSizeChange(nextPageSize: number) {
    const normalized = Number(nextPageSize) || 20
    pageSize.value = Math.min(Math.max(1, normalized), 500)
    page.value = 1
    if (options.currentResource.value === 'pods') options.clearPodSelection()
  }

  function onSortChange(value: { prop?: string; order?: 'ascending' | 'descending' | null }) {
    sortBy.value = value?.prop || undefined
    order.value = value?.order === 'ascending' ? 'asc' : value?.order === 'descending' ? 'desc' : undefined
  }

  function stopTimers() {
    if (keywordDebounceTimer != null) {
      window.clearTimeout(keywordDebounceTimer)
      keywordDebounceTimer = null
    }
  }

  return {
    sortBy,
    order,
    keywordInput,
    keyword,
    page,
    pageSize,
    pageSizeOptions,
    namespace,
    workloadLabelSelector,
    showNamespaceSelect,
    displayedList,
    displayedTotal,
    maxPage,
    pagedList,
    showPager,
    onNamespaceSelectChange,
    onPageChange,
    onPageSizeChange,
    onSortChange,
    stopTimers
  }
}
