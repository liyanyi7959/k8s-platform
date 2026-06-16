/**
 * useClusterManageViewState 单元测试
 * 覆盖关键字防抖、分页边界、命名空间筛选等核心状态逻辑
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref, computed, nextTick, shallowRef } from 'vue'
import { useClusterManageViewState } from '../useClusterManageViewState'
import type { ResourceKey, K8sLikeObject } from '@/features/k8s/pages/ClusterManageView.types'

function createStub(options?: {
  list?: K8sLikeObject[]
  currentResource?: ResourceKey
  namespaces?: string[]
}) {
  const list = shallowRef<K8sLikeObject[]>(options?.list ?? [])
  const currentResource = computed(() => options?.currentResource)
  const namespaces = computed(() => options?.namespaces ?? [])
  const clearPodSelection = vi.fn()

  return useClusterManageViewState({
    list,
    currentResource,
    namespaces,
    allNamespace: '__ALL__',
    clearPodSelection,
  })
}

describe('useClusterManageViewState', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  // ── 关键字防抖 ──
  describe('关键字防抖', () => {
    it('keywordInput 变更后 180ms 才更新 keyword', async () => {
      const sut = createStub({ currentResource: 'pods' })
      sut.keywordInput.value = 'nginx'
      // watch 是异步的，需要 nextTick 让 watcher 触发并注册 setTimeout
      await nextTick()
      expect(sut.keyword.value).toBe('')

      vi.advanceTimersByTime(179)
      expect(sut.keyword.value).toBe('')

      vi.advanceTimersByTime(1)
      expect(sut.keyword.value).toBe('nginx')
    })

    it('快速连续输入只保留最后一次', async () => {
      const sut = createStub({ currentResource: 'pods' })
      sut.keywordInput.value = 'a'
      await nextTick()
      vi.advanceTimersByTime(50)
      sut.keywordInput.value = 'ab'
      await nextTick()
      vi.advanceTimersByTime(50)
      sut.keywordInput.value = 'abc'
      await nextTick()
      vi.advanceTimersByTime(180)

      expect(sut.keyword.value).toBe('abc')
    })

    it('stopTimers 清除防抖定时器', () => {
      const sut = createStub({ currentResource: 'pods' })
      sut.keywordInput.value = 'test'
      sut.stopTimers()
      vi.advanceTimersByTime(200)

      expect(sut.keyword.value).toBe('')
    })
  })

  // ── 分页逻辑 ──
  describe('分页', () => {
    const makeItems = (n: number): K8sLikeObject[] =>
      Array.from({ length: n }, (_, i) => ({
        metadata: { namespace: 'default', name: `pod-${i}` },
        kind: 'Pod',
      }) as K8sLikeObject)

    it('pagedList 按 pageSize 分页', () => {
      const sut = createStub({ list: makeItems(50), currentResource: 'pods' })
      expect(sut.pagedList.value).toHaveLength(20)
      expect(sut.pageSize.value).toBe(20)
    })

    it('page > maxPage 时自动修正', async () => {
      // displayedTotal watcher 在 total 减少时修正 page
      const list = shallowRef<K8sLikeObject[]>(makeItems(100))
      const sut = useClusterManageViewState({
        list,
        currentResource: computed(() => 'pods' as ResourceKey),
        namespaces: computed(() => []),
        allNamespace: '__ALL__',
        clearPodSelection: vi.fn(),
      })
      sut.page.value = 5
      // 缩减列表使 maxPage < page
      list.value = makeItems(10)
      await nextTick()
      expect(sut.page.value).toBeLessThanOrEqual(sut.maxPage.value)
    })

    it('onPageChange 限制范围', () => {
      const sut = createStub({ list: makeItems(30), currentResource: 'pods' })
      sut.onPageChange(0)
      expect(sut.page.value).toBe(1)

      sut.onPageChange(999)
      expect(sut.page.value).toBe(sut.maxPage.value)
    })

    it('onPageSizeChange 重置 page 到 1', () => {
      const sut = createStub({ list: makeItems(100), currentResource: 'pods' })
      sut.page.value = 3
      sut.onPageSizeChange(50)
      expect(sut.page.value).toBe(1)
      expect(sut.pageSize.value).toBe(50)
    })

    it('keyword 变更重置 page 到 1', async () => {
      const sut = createStub({ list: makeItems(50), currentResource: 'pods' })
      sut.page.value = 3
      sut.keyword.value = 'test'
      await nextTick()
      expect(sut.page.value).toBe(1)
    })
  })

  // ── 搜索过滤 ──
  describe('搜索过滤', () => {
    it('keyword 过滤 displayedList', () => {
      const items = [
        { metadata: { namespace: 'default', name: 'nginx' }, kind: 'Pod' },
        { metadata: { namespace: 'default', name: 'redis' }, kind: 'Pod' },
      ] as K8sLikeObject[]
      const sut = createStub({ list: items, currentResource: 'pods' })
      sut.keyword.value = 'nginx'

      expect(sut.displayedList.value).toHaveLength(1)
      expect(sut.displayedList.value[0].metadata.name).toBe('nginx')
    })

    it('extraFilter 生效', () => {
      const items = [
        { metadata: { namespace: 'default', name: 'a' }, kind: 'Pod' },
        { metadata: { namespace: 'kube-system', name: 'b' }, kind: 'Pod' },
      ] as K8sLikeObject[]

      const list = shallowRef<K8sLikeObject[]>(items)
      const currentResource = computed(() => 'pods' as ResourceKey)
      const extraFilter = computed(() => (item: K8sLikeObject) => item.metadata.namespace === 'default')

      const sut = useClusterManageViewState({
        list,
        currentResource,
        namespaces: computed(() => []),
        allNamespace: '__ALL__',
        clearPodSelection: vi.fn(),
        extraFilter,
      })

      expect(sut.displayedList.value).toHaveLength(1)
    })
  })

  // ── showNamespaceSelect ──
  describe('showNamespaceSelect', () => {
    it('namespaced 资源返回 true', () => {
      const sut = createStub({ currentResource: 'pods' })
      expect(sut.showNamespaceSelect.value).toBe(true)
    })

    it('集群级资源返回 false', () => {
      const sut = createStub({ currentResource: 'nodes' })
      expect(sut.showNamespaceSelect.value).toBe(false)
    })

    it('undefined 资源返回 false', () => {
      const sut = createStub()
      expect(sut.showNamespaceSelect.value).toBe(false)
    })
  })

  // ── showPager ──
  describe('showPager', () => {
    it('dashboard 不显示分页', () => {
      const sut = createStub({ currentResource: 'dashboard' })
      expect(sut.showPager.value).toBe(false)
    })

    it('有数据的资源显示分页', () => {
      const items = [{ metadata: { namespace: 'default', name: 'a' }, kind: 'Pod' }] as K8sLikeObject[]
      const sut = createStub({ list: items, currentResource: 'pods' })
      expect(sut.showPager.value).toBe(true)
    })
  })

  // ── 排序 ──
  describe('排序', () => {
    it('onSortChange 更新 sortBy 和 order', () => {
      const sut = createStub({ currentResource: 'pods' })
      sut.onSortChange({ prop: 'name', order: 'ascending' })
      expect(sut.sortBy.value).toBe('name')
      expect(sut.order.value).toBe('asc')

      sut.onSortChange({ prop: 'name', order: 'descending' })
      expect(sut.order.value).toBe('desc')

      sut.onSortChange({ prop: undefined, order: null })
      expect(sut.sortBy.value).toBeUndefined()
      expect(sut.order.value).toBeUndefined()
    })
  })

  // ── clearPodSelection 调用 ──
  describe('Pod 选择清除', () => {
    it('onPageChange 在 pods 资源下调用 clearPodSelection', () => {
      const clearPodSelection = vi.fn()
      const list = shallowRef<K8sLikeObject[]>(
        Array.from({ length: 40 }, (_, i) => ({ metadata: { namespace: 'default', name: `p-${i}` }, kind: 'Pod' }) as K8sLikeObject)
      )
      const sut = useClusterManageViewState({
        list,
        currentResource: computed(() => 'pods' as ResourceKey),
        namespaces: computed(() => []),
        allNamespace: '__ALL__',
        clearPodSelection,
      })
      sut.onPageChange(2)
      expect(clearPodSelection).toHaveBeenCalled()
    })

    it('onPageSizeChange 在 pods 资源下调用 clearPodSelection', () => {
      const clearPodSelection = vi.fn()
      const list = shallowRef<K8sLikeObject[]>([])
      const sut = useClusterManageViewState({
        list,
        currentResource: computed(() => 'pods' as ResourceKey),
        namespaces: computed(() => []),
        allNamespace: '__ALL__',
        clearPodSelection,
      })
      sut.onPageSizeChange(50)
      expect(clearPodSelection).toHaveBeenCalled()
    })
  })
})
