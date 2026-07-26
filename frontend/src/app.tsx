import { history, useModel, type RequestConfig } from '@umijs/max'
import { isValidElement, useEffect, useRef, useState, type ReactElement } from 'react'
import {
  App as AntdApp,
  Avatar,
  Badge,
  ConfigProvider,
  Dropdown,
  Select,
  Space,
  Tag,
  message,
  type MenuProps,
} from 'antd'
import {
  AlertOutlined,
  AppstoreOutlined,
  ArrowLeftOutlined,
  AuditOutlined,
  CloudServerOutlined,
  ClusterOutlined,
  DashboardOutlined,
  DoubleLeftOutlined,
  DoubleRightOutlined,
  DownOutlined,
  FileSearchOutlined,
  LockOutlined,
  LogoutOutlined,
  PlusOutlined,
  RobotOutlined,
  RocketOutlined,
  SettingOutlined,
  UnorderedListOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query'
import { getCurrentUser, logout as requestLogout } from '@/services/auth'
import { listClusters } from '@/services/clusters'
import { listIncidents } from '@/services/monitor'
import { ChangePasswordModal } from '@/components'
import type { User } from '@/types'
import type { Cluster as ModelCluster } from '@/models/cluster'
import {
  enterClusterWorkspace,
  getClusterStatusColor,
  getClusterStatusText,
} from '@/utils'
import defaultSettings from '../config/defaultSettings'

// ============================================================
// React Query 全局配置
// ============================================================
// Keep the client in this module instead of Umi's generated runtime. The
// generated runtime imports app.tsx before @umijs/max has finished exporting,
// which otherwise creates a circular dependency during startup.
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      refetchInterval: false,
    },
    mutations: {
      retry: 0,
    },
  },
})

// ============================================================
// Antd Static Holder (用于 message/notification 等静态方法)
// ============================================================
let staticHolderConfigured = false
const BRAND_FAVICON_HREF = '/brand/aiops-mark.svg'

const ensureStaticHolder = () => {
  if (staticHolderConfigured) return
  ConfigProvider.config({
    holderRender: (children) => <AntdApp>{children}</AntdApp>,
  })
  staticHolderConfigured = true
}

const AppGlobalCursor = () => {
  const cursorRef = useRef<HTMLSpanElement>(null)
  const scrollbarShieldRef = useRef<HTMLSpanElement>(null)

  useEffect(() => {
    const cursor = cursorRef.current
    const scrollbarShield = scrollbarShieldRef.current
    if (!cursor || !scrollbarShield || typeof window === 'undefined') return

    const finePointer = window.matchMedia('(hover: hover) and (pointer: fine)')
    let pointerPressed = false
    type CursorKind = 'default' | 'pointer' | 'text'
    type ScrollbarAxis = 'horizontal' | 'vertical'
    type ScrollbarHit = {
      element: HTMLElement
      axis: ScrollbarAxis
      left: number
      top: number
      width: number
      height: number
      trackLength: number
      thumbLength: number
      thumbOffset: number
      maxScroll: number
    }
    type ScrollbarDrag = {
      element: HTMLElement
      axis: ScrollbarAxis
      grabOffset: number
      pointerId: number
    }
    let currentScrollbar: ScrollbarHit | null = null
    let scrollbarDrag: ScrollbarDrag | null = null

    const textCursorSelector = [
      'textarea',
      '[contenteditable]:not([contenteditable="false"])',
      'input:not([type])',
      'input[type="text"]',
      'input[type="search"]',
      'input[type="email"]',
      'input[type="password"]',
      'input[type="tel"]',
      'input[type="url"]',
      'input[type="number"]',
      '.monaco-editor .view-lines',
    ].join(',')
    const pointerCursorSelector = [
      'a[href]',
      'button',
      'summary',
      'select',
      'label[for]',
      '[role="button"]',
      '[role="link"]',
      '[role="menuitem"]',
      '[role="option"]',
      '[role="tab"]',
      '[role="checkbox"]',
      '[role="radio"]',
      '[role="switch"]',
      '[role="combobox"]',
      '[aria-haspopup]',
      '[tabindex]:not([tabindex="-1"])',
      'input[type="button"]',
      'input[type="submit"]',
      'input[type="reset"]',
      'input[type="checkbox"]',
      'input[type="radio"]',
      'input[type="range"]',
      'input[type="file"]',
      '.ant-btn',
      '.ant-select-selector',
      '.ant-dropdown-menu-item',
      '.ant-pagination-item',
    ].join(',')

    const hideCursor = () => {
      cursor.style.opacity = '0'
    }
    const resolveCursorKind = (clientX: number, clientY: number): CursorKind => {
      const target = document.elementFromPoint(clientX, clientY)
      if (!(target instanceof Element) || target === scrollbarShield) return 'default'

      const textTarget = target.closest(textCursorSelector)
      if (textTarget && !textTarget.matches(':disabled, [aria-disabled="true"]')) return 'text'

      const pointerTarget = target.closest(pointerCursorSelector)
      if (pointerTarget && !pointerTarget.matches(':disabled, [aria-disabled="true"]')) return 'pointer'
      return 'default'
    }
    const positionCursor = (clientX: number, clientY: number, kind: CursorKind = 'default') => {
      const offsets: Record<CursorKind, { x: number; y: number }> = {
        default: { x: 2, y: 1 },
        pointer: { x: 10, y: 10 },
        text: { x: 5, y: 10 },
      }
      const offset = offsets[kind]
      const x = Math.round(clientX - offset.x)
      const y = Math.round(clientY - offset.y)
      cursor.dataset.kind = kind
      cursor.style.transform = `translate3d(${x}px, ${y}px, 0)`
      cursor.style.opacity = '1'
    }
    const hideScrollbarShield = () => {
      currentScrollbar = null
      scrollbarShield.style.display = 'none'
    }
    const measureScrollbar = (element: HTMLElement, axis: ScrollbarAxis): ScrollbarHit | null => {
      const rect = element.getBoundingClientRect()
      const style = window.getComputedStyle(element)
      const borderLeft = Number.parseFloat(style.borderLeftWidth) || 0
      const borderRight = Number.parseFloat(style.borderRightWidth) || 0
      const borderTop = Number.parseFloat(style.borderTopWidth) || 0
      const borderBottom = Number.parseFloat(style.borderBottomWidth) || 0
      const verticalWidth = Math.max(0, element.offsetWidth - element.clientWidth - borderLeft - borderRight)
      const horizontalHeight = Math.max(0, element.offsetHeight - element.clientHeight - borderTop - borderBottom)

      if (axis === 'horizontal') {
        const maxScroll = element.scrollWidth - element.clientWidth
        const trackLength = element.clientWidth
        if (!['auto', 'scroll'].includes(style.overflowX) || maxScroll <= 1 || horizontalHeight <= 0 || trackLength <= 0) {
          return null
        }
        const thumbLength = Math.min(trackLength, Math.max(24, trackLength * (element.clientWidth / element.scrollWidth)))
        const thumbTravel = Math.max(0, trackLength - thumbLength)
        return {
          element,
          axis,
          left: rect.left + borderLeft,
          top: rect.bottom - borderBottom - horizontalHeight,
          width: trackLength,
          height: horizontalHeight,
          trackLength,
          thumbLength,
          thumbOffset: maxScroll > 0 ? (element.scrollLeft / maxScroll) * thumbTravel : 0,
          maxScroll,
        }
      }

      const maxScroll = element.scrollHeight - element.clientHeight
      const trackLength = element.clientHeight
      if (!['auto', 'scroll'].includes(style.overflowY) || maxScroll <= 1 || verticalWidth <= 0 || trackLength <= 0) {
        return null
      }
      const thumbLength = Math.min(trackLength, Math.max(24, trackLength * (element.clientHeight / element.scrollHeight)))
      const thumbTravel = Math.max(0, trackLength - thumbLength)
      return {
        element,
        axis,
        left: rect.right - borderRight - verticalWidth,
        top: rect.top + borderTop,
        width: verticalWidth,
        height: trackLength,
        trackLength,
        thumbLength,
        thumbOffset: maxScroll > 0 ? (element.scrollTop / maxScroll) * thumbTravel : 0,
        maxScroll,
      }
    }
    const findScrollbarAtPoint = (clientX: number, clientY: number): ScrollbarHit | null => {
      const previousPointerEvents = scrollbarShield.style.pointerEvents
      scrollbarShield.style.pointerEvents = 'none'
      const elements = document.elementsFromPoint(clientX, clientY)
      scrollbarShield.style.pointerEvents = previousPointerEvents

      const candidates = new Set<HTMLElement>()
      elements.forEach((start) => {
        let element: Element | null = start
        while (element) {
          if (element instanceof HTMLElement && element !== scrollbarShield) candidates.add(element)
          element = element.parentElement
        }
      })

      for (const element of candidates) {
        const horizontal = measureScrollbar(element, 'horizontal')
        if (
          horizontal &&
          clientX >= horizontal.left &&
          clientX <= horizontal.left + horizontal.width &&
          clientY >= horizontal.top &&
          clientY <= horizontal.top + horizontal.height
        ) {
          return horizontal
        }
        const vertical = measureScrollbar(element, 'vertical')
        if (
          vertical &&
          clientX >= vertical.left &&
          clientX <= vertical.left + vertical.width &&
          clientY >= vertical.top &&
          clientY <= vertical.top + vertical.height
        ) {
          return vertical
        }
      }
      return null
    }
    const showScrollbarShield = (hit: ScrollbarHit) => {
      currentScrollbar = hit
      scrollbarShield.style.display = 'block'
      scrollbarShield.style.left = `${Math.round(hit.left)}px`
      scrollbarShield.style.top = `${Math.round(hit.top)}px`
      scrollbarShield.style.width = `${Math.ceil(hit.width)}px`
      scrollbarShield.style.height = `${Math.ceil(hit.height)}px`
    }
    const updateScrollbarDrag = (clientX: number, clientY: number) => {
      if (!scrollbarDrag) return
      const hit = measureScrollbar(scrollbarDrag.element, scrollbarDrag.axis)
      if (!hit) return
      const pointerPosition = scrollbarDrag.axis === 'horizontal' ? clientX - hit.left : clientY - hit.top
      const thumbTravel = Math.max(0, hit.trackLength - hit.thumbLength)
      const nextThumbOffset = Math.min(thumbTravel, Math.max(0, pointerPosition - scrollbarDrag.grabOffset))
      const nextScroll = thumbTravel > 0 ? (nextThumbOffset / thumbTravel) * hit.maxScroll : 0
      if (scrollbarDrag.axis === 'horizontal') {
        scrollbarDrag.element.scrollLeft = nextScroll
      } else {
        scrollbarDrag.element.scrollTop = nextScroll
      }
      showScrollbarShield(measureScrollbar(scrollbarDrag.element, scrollbarDrag.axis) || hit)
    }

    const handlePointerMove = (event: PointerEvent) => {
      if (!finePointer.matches || event.pointerType === 'touch') {
        hideCursor()
        return
      }

      if (scrollbarDrag) {
        positionCursor(event.clientX, event.clientY)
        updateScrollbarDrag(event.clientX, event.clientY)
        return
      }
      const hit = findScrollbarAtPoint(event.clientX, event.clientY)
      if (hit) {
        showScrollbarShield(hit)
        positionCursor(event.clientX, event.clientY)
      } else {
        hideScrollbarShield()
        positionCursor(event.clientX, event.clientY, resolveCursorKind(event.clientX, event.clientY))
      }
    }
    // Native scrollbar dragging can suppress pointer events on some Chromium/
    // Windows combinations. Mouse events provide a fallback while pressed.
    const handleMouseMove = (event: MouseEvent) => {
      if (!finePointer.matches || event.buttons === 0) return
      pointerPressed = true
      positionCursor(
        event.clientX,
        event.clientY,
        scrollbarDrag ? 'default' : resolveCursorKind(event.clientX, event.clientY),
      )
      updateScrollbarDrag(event.clientX, event.clientY)
    }
    const handlePointerDown = (event: PointerEvent) => {
      pointerPressed = true
      cursor.classList.add('is-pressed')
      handlePointerMove(event)
      const hit = currentScrollbar || findScrollbarAtPoint(event.clientX, event.clientY)
      if (!hit) return

      event.preventDefault()
      event.stopPropagation()
      showScrollbarShield(hit)
      const pointerPosition = hit.axis === 'horizontal' ? event.clientX - hit.left : event.clientY - hit.top
      const onThumb = pointerPosition >= hit.thumbOffset && pointerPosition <= hit.thumbOffset + hit.thumbLength
      scrollbarDrag = {
        element: hit.element,
        axis: hit.axis,
        grabOffset: onThumb ? pointerPosition - hit.thumbOffset : hit.thumbLength / 2,
        pointerId: event.pointerId,
      }
      try {
        scrollbarShield.setPointerCapture(event.pointerId)
      } catch {
        // The window-level listeners still keep manual dragging functional.
      }
      updateScrollbarDrag(event.clientX, event.clientY)
    }
    const handlePointerUp = (event: PointerEvent | MouseEvent) => {
      pointerPressed = false
      cursor.classList.remove('is-pressed')
      if (scrollbarDrag && 'pointerId' in event && scrollbarShield.hasPointerCapture(scrollbarDrag.pointerId)) {
        scrollbarShield.releasePointerCapture(scrollbarDrag.pointerId)
      }
      scrollbarDrag = null
    }
    const handlePointerOut = (event: PointerEvent) => {
      if (event.relatedTarget === null && !pointerPressed) hideCursor()
    }
    const handlePointerCapabilityChange = () => {
      if (!finePointer.matches) hideCursor()
    }
    const handleScrollbarWheel = (event: WheelEvent) => {
      if (!currentScrollbar) return
      event.preventDefault()
      const delta = currentScrollbar.axis === 'horizontal'
        ? (event.deltaX || event.deltaY)
        : (event.deltaY || event.deltaX)
      if (currentScrollbar.axis === 'horizontal') {
        currentScrollbar.element.scrollLeft += delta
      } else {
        currentScrollbar.element.scrollTop += delta
      }
      const nextHit = measureScrollbar(currentScrollbar.element, currentScrollbar.axis)
      if (nextHit) showScrollbarShield(nextHit)
    }
    const handleWindowBlur = () => {
      pointerPressed = false
      cursor.classList.remove('is-pressed')
      scrollbarDrag = null
      hideScrollbarShield()
      hideCursor()
    }

    window.addEventListener('pointermove', handlePointerMove, { passive: true })
    window.addEventListener('mousemove', handleMouseMove, { capture: true, passive: true })
    window.addEventListener('pointerdown', handlePointerDown, true)
    window.addEventListener('pointerup', handlePointerUp, true)
    window.addEventListener('mouseup', handlePointerUp, true)
    window.addEventListener('pointercancel', handlePointerUp, true)
    window.addEventListener('pointerout', handlePointerOut)
    window.addEventListener('blur', handleWindowBlur)
    scrollbarShield.addEventListener('wheel', handleScrollbarWheel, { passive: false })
    finePointer.addEventListener('change', handlePointerCapabilityChange)

    return () => {
      window.removeEventListener('pointermove', handlePointerMove)
      window.removeEventListener('mousemove', handleMouseMove, true)
      window.removeEventListener('pointerdown', handlePointerDown, true)
      window.removeEventListener('pointerup', handlePointerUp, true)
      window.removeEventListener('mouseup', handlePointerUp, true)
      window.removeEventListener('pointercancel', handlePointerUp, true)
      window.removeEventListener('pointerout', handlePointerOut)
      window.removeEventListener('blur', handleWindowBlur)
      scrollbarShield.removeEventListener('wheel', handleScrollbarWheel)
      finePointer.removeEventListener('change', handlePointerCapabilityChange)
    }
  }, [])

  return (
    <>
      <span ref={scrollbarShieldRef} className="app-scrollbar-pointer-shield" aria-hidden="true" />
      <span ref={cursorRef} className="app-global-cursor" data-kind="default" aria-hidden="true">
        <img className="app-global-cursor__asset app-global-cursor__default" src="/brand/aiops-cursor.svg?v=5" alt="" draggable={false} />
        <img className="app-global-cursor__asset app-global-cursor__pointer" src="/brand/aiops-cursor-pointer.svg?v=2" alt="" draggable={false} />
        <img className="app-global-cursor__asset app-global-cursor__text" src="/brand/aiops-cursor-text.svg?v=2" alt="" draggable={false} />
      </span>
    </>
  )
}

const AppRuntimeBranding = ({ children }: { children: React.ReactNode }) => {
  useEffect(() => {
    if (typeof document === 'undefined') {
      return
    }

    const ensureFaviconLink = (rel: 'icon' | 'shortcut icon') => {
      let link = document.querySelector<HTMLLinkElement>(`link[rel="${rel}"]`)
      if (!link) {
        link = document.createElement('link')
        link.rel = rel
        document.head.appendChild(link)
      }
      link.type = 'image/svg+xml'
      link.href = BRAND_FAVICON_HREF
    }

    ensureFaviconLink('icon')
    ensureFaviconLink('shortcut icon')
  }, [])

  return (
    <>
      {children}
      <AppGlobalCursor />
    </>
  )
}

export const rootContainer = (container: React.ReactNode) => {
  ensureStaticHolder()
  return (
    <QueryClientProvider client={queryClient}>
      <AntdApp>
        <AppRuntimeBranding>{container}</AppRuntimeBranding>
      </AntdApp>
    </QueryClientProvider>
  )
}

const LOGIN_PATH = '/login'

const isLoginRoute = () => {
  const runtimeLocation = typeof window !== 'undefined' ? window.location : undefined
  const pathname = history.location?.pathname || runtimeLocation?.pathname || ''
  const hashPath = (runtimeLocation?.hash || '').replace(/^#/, '').split('?')[0]
  return pathname === LOGIN_PATH || hashPath === LOGIN_PATH
}

type RequestBusinessError = Error & {
  code?: number
  data?: unknown
}

const createRequestError = (messageText: string, code?: number, data?: unknown): RequestBusinessError => {
  const error = new Error(messageText) as RequestBusinessError
  error.name = 'BusinessError'
  error.code = code
  error.data = data
  return error
}

// ============================================================
// 初始状态
// ============================================================
export async function getInitialState(): Promise<{
  currentUser?: User
}> {
  if (isLoginRoute()) {
    return {}
  }

  try {
    const currentUser = await getCurrentUser()
    return { currentUser }
  } catch {
    if (!isLoginRoute()) {
      history.replace(LOGIN_PATH)
    }
    return {}
  }
}

// ============================================================
// 类型定义
// ============================================================
type MenuItem = {
  name?: React.ReactNode
  path?: string
  key?: string
  icon?: React.ReactNode
  children?: MenuItem[]
  component?: string
  redirect?: string
  className?: string
  [key: string]: any
}

// ============================================================
// 菜单 A：全局模式（currentCluster === null）
// 参照 frontend-old 的菜单结构，确保所有父级菜单可正常展开
// ============================================================
const buildGlobalMenuItems = (): MenuItem[] => [
  // ---- 仪表盘 ----
  {
    key: '/dashboard',
    path: '/dashboard',
    name: '仪表盘',
    icon: <DashboardOutlined />,
  },
  // ---- 集群管理（集群列表、集群导入、部署K8S集群） ----
  {
    key: '/clusters',
    path: '/clusters',
    name: '集群管理',
    icon: <ClusterOutlined />,
    children: [
      { key: '/clusters', path: '/clusters', name: '集群列表', icon: <UnorderedListOutlined /> },
      { key: '/clusters/import', path: '/clusters/import', name: '集群导入', icon: <PlusOutlined /> },
      { key: '/clusters/provision', path: '/clusters/provision', name: '部署K8S集群', icon: <RocketOutlined /> },
    ],
  },
  // ---- 服务器管理（主机资源池等服务器相关功能） ----
  {
    key: '/servers',
    path: '/clusters/hosts',
    name: '服务器管理',
    icon: <CloudServerOutlined />,
    children: [
      { key: '/clusters/hosts', path: '/clusters/hosts', name: '主机资源池', icon: <CloudServerOutlined /> },
    ],
  },
  // ---- 项目管理 ----
  {
    key: '/projects',
    path: '/projects',
    name: '项目管理',
    icon: <AppstoreOutlined />,
  },
  // ---- 应用商店 ----
  {
    key: '/app-store',
    path: '/app-store',
    name: '应用商店',
    icon: <AppstoreOutlined />,
    children: [
      { key: '/app-store/yaml', path: '/app-store/yaml', name: 'YAML 模板' },
      { key: '/app-store/helm', path: '/app-store/helm', name: 'Helm Chart' },
    ],
  },
  // ---- 自动化中心：承载可复用运行手册与执行资产，而不是某个具体集群的创建入口 ----
  {
    key: '/automation',
    path: '/automation/overview',
    name: '自动化中心',
    icon: <RocketOutlined />,
    children: [
      { key: '/automation/overview', path: '/automation/overview', name: '自动化总览' },
      { key: '/automation/assets', path: '/automation/assets', name: '运行手册资产' },
    ],
  },
  // ---- 监控告警 ----
  {
    key: '/monitor',
    path: '/monitor/dashboard',
    name: '监控告警',
    icon: <AlertOutlined />,
    children: [
      { key: '/monitor/dashboard', path: '/monitor/dashboard', name: '监控仪表盘' },
      { key: '/monitor/alerts', path: '/monitor/alerts', name: '告警规则' },
      { key: '/monitor/events', path: '/monitor/events', name: '事件流' },
    ],
  },
  // ---- 日志分析 ----
  {
    key: '/logs',
    path: '/logs',
    name: '日志分析',
    icon: <FileSearchOutlined />,
  },
  // ---- AI 运维助手 ----
  {
    key: '/ai',
    path: '/ai/chat',
    name: 'AI 运维助手',
    icon: <RobotOutlined />,
    children: [
      { key: '/ai/chat', path: '/ai/chat', name: '对话式运维' },
      { key: '/ai/history', path: '/ai/history', name: '历史记录' },
      { key: '/ai/settings', path: '/ai/settings', name: '模型配置' },
    ],
  },
]

const buildAdminMenuItems = (): MenuItem[] => [
  {
    key: 'return-global',
    path: '/dashboard',
    name: (
      <span style={{ fontWeight: 600 }}>返回工作台</span>
    ),
    icon: <ArrowLeftOutlined />,
    className: 'app-menu-return-global',
  },
  {
    key: 'group-admin-access',
    path: '/config/users',
    name: '账号与权限',
    icon: <UserOutlined />,
    children: [
      { key: '/config/users', path: '/config/users', name: '用户管理' },
      { key: '/config/roles', path: '/config/roles', name: '角色管理' },
    ],
  },
  {
    key: 'group-admin-platform',
    path: '/config/settings',
    name: '平台配置',
    icon: <SettingOutlined />,
    children: [
      { key: '/config/settings', path: '/config/settings', name: '系统设置' },
      { key: '/config/credentials', path: '/config/credentials', name: '凭据库' },
    ],
  },
  {
    key: 'group-admin-audit',
    path: '/config/audit-logs',
    name: '审计与追踪',
    icon: <AuditOutlined />,
    children: [
      { key: '/config/audit-logs', path: '/config/audit-logs', name: '审计日志' },
    ],
  },
]

// ============================================================
// 菜单 B：集群钻取模式（currentCluster 有值）
// K8s 资源目录树 — 参照 frontend-old 的 buildTree 完整结构
// ============================================================
const buildClusterMenuItems = (clusterId: string): MenuItem[] => [
  // ---- 特殊置顶项：返回全局概览 ----
  {
    key: 'return-global',
    path: '/dashboard',
    name: (
      <span style={{ fontWeight: 600 }}>返回全局概览</span>
    ),
    icon: <ArrowLeftOutlined />,
    className: 'app-menu-return-global',
  },
  // ---- 仪表盘 ----
  {
    key: 'group-dashboard',
    path: `/k8s/${clusterId}/dashboard`,
    name: '应用健康',
    icon: <DashboardOutlined />,
    children: [
      { key: `/k8s/${clusterId}/dashboard`, path: `/k8s/${clusterId}/dashboard`, name: '健康总览' },
    ],
  },
  // ---- 事件处置 ----
  {
    key: 'group-operations',
    path: `/k8s/${clusterId}/events`,
    name: '事件处置',
    icon: <AlertOutlined />,
    children: [
      { key: `/k8s/${clusterId}/events`, path: `/k8s/${clusterId}/events`, name: '集群 Events' },
      { key: `/k8s/${clusterId}/log-workbench`, path: `/k8s/${clusterId}/log-workbench`, name: '日志工作台' },
      { key: `/k8s/${clusterId}/resource-metrics`, path: `/k8s/${clusterId}/resource-metrics`, name: '资源指标' },
    ],
  },
  // ---- 交付与变更 ----
  {
    key: 'group-delivery',
    path: `/k8s/${clusterId}/manifest-apply`,
    name: '交付与变更',
    icon: <RocketOutlined />,
    children: [
      { key: `/k8s/${clusterId}/manifest-apply`, path: `/k8s/${clusterId}/manifest-apply`, name: 'YAML 部署' },
      { key: `/k8s/${clusterId}/helm-releases`, path: `/k8s/${clusterId}/helm-releases`, name: 'Helm 发布' },
      { key: `/k8s/${clusterId}/helm-repos`, path: `/k8s/${clusterId}/helm-repos`, name: 'Helm 仓库' },
      { key: `/k8s/${clusterId}/permission-audits`, path: `/k8s/${clusterId}/permission-audits`, name: '权限分析' },
    ],
  },
  // ---- 高级资源浏览器 ----
  {
    key: 'group-resource-browser',
    path: `/k8s/${clusterId}/advanced-resources`,
    name: '资源浏览器',
    icon: <ClusterOutlined />,
    children: [
      { key: `/k8s/${clusterId}/advanced-resources`, path: `/k8s/${clusterId}/advanced-resources`, name: '资源目录' },
      {
        key: 'resource-workloads',
        name: '应用与运行时',
        children: [
          { key: `/k8s/${clusterId}/pods`, path: `/k8s/${clusterId}/pods`, name: 'Pods' },
          { key: `/k8s/${clusterId}/deployments`, path: `/k8s/${clusterId}/deployments`, name: 'Deployments' },
          { key: `/k8s/${clusterId}/statefulsets`, path: `/k8s/${clusterId}/statefulsets`, name: 'StatefulSets' },
          { key: `/k8s/${clusterId}/daemonsets`, path: `/k8s/${clusterId}/daemonsets`, name: 'DaemonSets' },
          { key: `/k8s/${clusterId}/replicasets`, path: `/k8s/${clusterId}/replicasets`, name: 'ReplicaSets' },
          { key: `/k8s/${clusterId}/jobs`, path: `/k8s/${clusterId}/jobs`, name: 'Jobs' },
          { key: `/k8s/${clusterId}/cronjobs`, path: `/k8s/${clusterId}/cronjobs`, name: 'CronJobs' },
          { key: `/k8s/${clusterId}/hpas`, path: `/k8s/${clusterId}/hpas`, name: 'HPAs' },
          { key: `/k8s/${clusterId}/pdbs`, path: `/k8s/${clusterId}/pdbs`, name: 'PDBs' },
        ],
      },
      {
        key: 'resource-network',
        name: '网络与入口',
        children: [
          { key: `/k8s/${clusterId}/services`, path: `/k8s/${clusterId}/services`, name: 'Services' },
          { key: `/k8s/${clusterId}/endpoints`, path: `/k8s/${clusterId}/endpoints`, name: 'Endpoints' },
          { key: `/k8s/${clusterId}/endpointslices`, path: `/k8s/${clusterId}/endpointslices`, name: 'EndpointSlices' },
          { key: `/k8s/${clusterId}/ingresses`, path: `/k8s/${clusterId}/ingresses`, name: 'Ingresses' },
          { key: `/k8s/${clusterId}/ingress-classes`, path: `/k8s/${clusterId}/ingress-classes`, name: 'IngressClasses' },
          { key: `/k8s/${clusterId}/network-policies`, path: `/k8s/${clusterId}/network-policies`, name: 'NetworkPolicies' },
        ],
      },
      {
        key: 'resource-storage',
        name: '存储与配额',
        children: [
          { key: `/k8s/${clusterId}/pvcs`, path: `/k8s/${clusterId}/pvcs`, name: 'PVCs' },
          { key: `/k8s/${clusterId}/pvs`, path: `/k8s/${clusterId}/pvs`, name: 'PVs' },
          { key: `/k8s/${clusterId}/storage-classes`, path: `/k8s/${clusterId}/storage-classes`, name: 'StorageClasses' },
          { key: `/k8s/${clusterId}/volume-snapshots`, path: `/k8s/${clusterId}/volume-snapshots`, name: 'VolumeSnapshots' },
          { key: `/k8s/${clusterId}/resource-quotas`, path: `/k8s/${clusterId}/resource-quotas`, name: 'ResourceQuotas' },
          { key: `/k8s/${clusterId}/limit-ranges`, path: `/k8s/${clusterId}/limit-ranges`, name: 'LimitRanges' },
          { key: `/k8s/${clusterId}/csi-drivers`, path: `/k8s/${clusterId}/csi-drivers`, name: 'CSIDrivers' },
          { key: `/k8s/${clusterId}/volume-attachments`, path: `/k8s/${clusterId}/volume-attachments`, name: 'VolumeAttachments' },
        ],
      },
      {
        key: 'resource-access',
        name: '配置与访问控制',
        children: [
          { key: `/k8s/${clusterId}/configmaps`, path: `/k8s/${clusterId}/configmaps`, name: 'ConfigMaps' },
          { key: `/k8s/${clusterId}/secrets`, path: `/k8s/${clusterId}/secrets`, name: 'Secrets' },
          { key: `/k8s/${clusterId}/service-accounts`, path: `/k8s/${clusterId}/service-accounts`, name: 'ServiceAccounts' },
          { key: `/k8s/${clusterId}/roles`, path: `/k8s/${clusterId}/roles`, name: 'Roles' },
          { key: `/k8s/${clusterId}/role-bindings`, path: `/k8s/${clusterId}/role-bindings`, name: 'RoleBindings' },
          { key: `/k8s/${clusterId}/cluster-roles`, path: `/k8s/${clusterId}/cluster-roles`, name: 'ClusterRoles' },
          { key: `/k8s/${clusterId}/cluster-role-bindings`, path: `/k8s/${clusterId}/cluster-role-bindings`, name: 'ClusterRoleBindings' },
        ],
      },
      {
        key: 'resource-cluster',
        name: '集群与扩展治理',
        children: [
          { key: `/k8s/${clusterId}/namespaces`, path: `/k8s/${clusterId}/namespaces`, name: 'Namespaces' },
          { key: `/k8s/${clusterId}/nodes`, path: `/k8s/${clusterId}/nodes`, name: 'Nodes' },
          { key: `/k8s/${clusterId}/leases`, path: `/k8s/${clusterId}/leases`, name: 'Leases' },
          { key: `/k8s/${clusterId}/crds`, path: `/k8s/${clusterId}/crds`, name: 'CRDs' },
          { key: `/k8s/${clusterId}/api-services`, path: `/k8s/${clusterId}/api-services`, name: 'APIServices' },
          { key: `/k8s/${clusterId}/priority-classes`, path: `/k8s/${clusterId}/priority-classes`, name: 'PriorityClasses' },
          { key: `/k8s/${clusterId}/runtime-classes`, path: `/k8s/${clusterId}/runtime-classes`, name: 'RuntimeClasses' },
          { key: `/k8s/${clusterId}/validating-webhooks`, path: `/k8s/${clusterId}/validating-webhooks`, name: 'ValidatingWebhooks' },
          { key: `/k8s/${clusterId}/mutating-webhooks`, path: `/k8s/${clusterId}/mutating-webhooks`, name: 'MutatingWebhooks' },
        ],
      },
      { key: `/k8s/${clusterId}/topology`, path: `/k8s/${clusterId}/topology`, name: '资源关系图' },
    ],
  },
]

// ============================================================
// 根据当前 pathname 确定需要展开的菜单 key
// 用于全局模式（"集群管理"）和集群模式（K8s 资源组）
// ============================================================
const getGlobalOpenKeys = (pathname: string): string[] => {
  if (pathname === '/clusters/hosts' || pathname.startsWith('/clusters/hosts/')) {
    return ['/servers']
  }
  const openKeys: string[] = []
  if (pathname.startsWith('/clusters')) {
    openKeys.push('/clusters')
  }
  if (pathname.startsWith('/automation')) {
    openKeys.push('/automation')
  }
  if (pathname.startsWith('/app-store')) {
    openKeys.push('/app-store')
  }
  if (pathname.startsWith('/monitor')) {
    openKeys.push('/monitor')
  }
  if (pathname.startsWith('/ai')) {
    openKeys.push('/ai')
  }
  return openKeys
}

const getAdminOpenKeys = (pathname: string): string[] => {
  if (pathname.startsWith('/config/users') || pathname.startsWith('/config/roles')) {
    return ['group-admin-access']
  }
  if (pathname.startsWith('/config/audit-logs')) {
    return ['group-admin-audit']
  }
  if (pathname.startsWith('/config/credentials')) {
    return ['group-admin-platform']
  }
  return ['group-admin-platform']
}

const getClusterOpenKeys = (pathname: string): string[] => {
  const resource = pathname.match(/\/k8s\/[^/]+\/([^/?]+)/)?.[1] || ''

  const dashboard = ['dashboard']
  const operations = ['events', 'log-workbench', 'resource-metrics']
  const delivery = ['manifest-apply', 'helm-releases', 'helm-repos', 'permission-audits']
  const browser = ['advanced-resources', 'topology', 'pods', 'podmetrics', 'deployments', 'statefulsets', 'daemonsets', 'replicasets', 'pdbs', 'hpas', 'jobs', 'cronjobs', 'services', 'endpoints', 'endpointslices', 'network-policies', 'ingresses', 'ingress-classes', 'pvcs', 'pvs', 'volume-snapshots', 'volume-snapshot-classes', 'volume-snapshot-contents', 'storage-classes', 'csi-drivers', 'csi-nodes', 'csi-storage-capacities', 'volume-attachments', 'resource-quotas', 'limit-ranges', 'configmaps', 'secrets', 'service-accounts', 'roles', 'role-bindings', 'cluster-roles', 'cluster-role-bindings', 'namespaces', 'nodes', 'leases', 'crds', 'api-services', 'priority-classes', 'runtime-classes', 'validating-webhooks', 'mutating-webhooks', 'validating-admission-policies', 'validating-admission-policy-bindings']

  if (dashboard.includes(resource)) return ['group-dashboard']
  if (operations.includes(resource)) return ['group-operations']
  if (delivery.includes(resource)) return ['group-delivery']
  const resourceGroups: Record<string, string> = {
    pods: 'resource-workloads', deployments: 'resource-workloads', statefulsets: 'resource-workloads', daemonsets: 'resource-workloads', replicasets: 'resource-workloads', jobs: 'resource-workloads', cronjobs: 'resource-workloads', hpas: 'resource-workloads', pdbs: 'resource-workloads',
    services: 'resource-network', endpoints: 'resource-network', endpointslices: 'resource-network', 'network-policies': 'resource-network', ingresses: 'resource-network', 'ingress-classes': 'resource-network',
    pvcs: 'resource-storage', pvs: 'resource-storage', 'volume-snapshots': 'resource-storage', 'storage-classes': 'resource-storage', 'csi-drivers': 'resource-storage', 'volume-attachments': 'resource-storage', 'resource-quotas': 'resource-storage', 'limit-ranges': 'resource-storage',
    configmaps: 'resource-access', secrets: 'resource-access', 'service-accounts': 'resource-access', roles: 'resource-access', 'role-bindings': 'resource-access', 'cluster-roles': 'resource-access', 'cluster-role-bindings': 'resource-access',
    namespaces: 'resource-cluster', nodes: 'resource-cluster', leases: 'resource-cluster', crds: 'resource-cluster', 'api-services': 'resource-cluster', 'priority-classes': 'resource-cluster', 'runtime-classes': 'resource-cluster', 'validating-webhooks': 'resource-cluster', 'mutating-webhooks': 'resource-cluster',
  }

  if (browser.includes(resource)) {
    return resourceGroups[resource]
      ? ['group-resource-browser', resourceGroups[resource]]
      : ['group-resource-browser']
  }
  return ['group-dashboard']
}

// ============================================================
// 辅助函数
// ============================================================
const getUserDisplayName = (user?: Partial<User> & { name?: string }) =>
  user?.nickname || user?.name || user?.username || 'Admin'

// ============================================================
// Layout 导出（ProLayout 配置）
// ============================================================
export const layout = ({ initialState, setInitialState }: any) => {
  const { currentCluster, setCurrentCluster, setClusterList } = useModel('cluster')
  const [changePasswordOpen, setChangePasswordOpen] = useState(false)

  // 使用真实后端 API 获取集群列表（替代 mock 数据）
  const { data: clusterListRes } = useQuery({
    queryKey: ['cluster-options'],
    queryFn: () => listClusters(),
  })

  // 将后端 Cluster 映射为 Model Cluster 格式，并同步到 Model
  useEffect(() => {
    const backendList = clusterListRes?.items
    if (backendList && backendList.length > 0) {
      const mapped: ModelCluster[] = backendList.map((c) => ({
        id: String(c.id),
        name: c.name,
        version: c.k8sVersion || '',
        status: c.status,
      }))
      setClusterList(mapped)
    }
  }, [clusterListRes, setClusterList])

  const pathname = history.location.pathname
  // 优先从 URL path 解析 clusterId（/k8s/:clusterId/...），确保刷新/直达时菜单立即正确
  const pathClusterId = pathname.match(/^\/k8s\/([^/]+)/)?.[1] ?? null
  const currentClusterId = pathClusterId ?? currentCluster?.id ?? null
  const isAdminMode = pathname.startsWith('/config')
  const isClusterMode = pathname.startsWith('/k8s/') && Boolean(currentClusterId)
  const currentUserName = getUserDisplayName(initialState?.currentUser)
  const currentUsername = initialState?.currentUser?.username || 'admin'

  const { data: navIncidents } = useQuery({
    queryKey: ['monitor-nav-incidents'],
    queryFn: ({ signal }) => listIncidents({ page: 1, pageSize: 100 }, signal),
    refetchInterval: 30_000,
  })
  const activeNavIncidents = (navIncidents?.items || []).filter((item) => item.status !== 'resolved')
  const criticalNavIncidents = activeNavIncidents.filter((item) => item.severity === 'critical').length

  // 集群列表（用于 Select 下拉）
  const clusters = clusterListRes?.items || []

  // 根据 currentCluster 决定使用哪套菜单
  const menuData = isAdminMode
    ? buildAdminMenuItems()
    : isClusterMode
      ? buildClusterMenuItems(currentClusterId!)
      : buildGlobalMenuItems()

  // 根据当前路径自动展开对应子菜单
  const defaultOpenKeys = isAdminMode
    ? getAdminOpenKeys(pathname)
    : isClusterMode
      ? getClusterOpenKeys(pathname)
      : getGlobalOpenKeys(pathname)
  const [menuOpenKeys, setMenuOpenKeys] = useState<string[]>(defaultOpenKeys)

  useEffect(() => {
    setMenuOpenKeys(defaultOpenKeys)
  }, [pathname, isAdminMode, isClusterMode, currentClusterId])

  const handleMenuOpenChange = (keys: string[]) => {
    if (isAdminMode || isClusterMode) {
      setMenuOpenKeys(keys)
      return
    }

    const globalParentKeys = new Set(['/clusters', '/servers', '/app-store', '/automation', '/monitor', '/ai'])
    const latestOpenedKey = keys.find((key) => !menuOpenKeys.includes(key))
    if (latestOpenedKey && globalParentKeys.has(latestOpenedKey)) {
      setMenuOpenKeys([latestOpenedKey])
      return
    }
    setMenuOpenKeys(keys.filter((key) => globalParentKeys.has(key)).slice(-1))
  }

  // ---- 退出登录 ----
  const handleLogout = async () => {
    try {
      await requestLogout()
    } catch {
      // ignore
    } finally {
      localStorage.removeItem('token')
      setInitialState({ currentUser: undefined })
      if (!isLoginRoute()) {
        history.replace(LOGIN_PATH)
      }
    }
  }

  // ---- 用户菜单 ----
  const userMenuItems: MenuProps['items'] = [
    {
      key: 'identity',
      disabled: true,
      label: (
        <div className="app-sider-account-dropdown__identity">
          <strong>{currentUserName}</strong>
          <span>{currentUsername}</span>
        </div>
      ),
    },
    { type: 'divider' },
    { key: 'admin', label: '管理后台', icon: <UserOutlined /> },
    { key: 'change-password', label: '修改密码', icon: <LockOutlined /> },
    { type: 'divider' },
    { key: 'logout', label: '退出登录', icon: <LogoutOutlined />, danger: true },
  ]

  const handleUserMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (key === 'logout') { handleLogout(); return }
    if (key === 'admin') { history.push('/config/users'); return }
    if (key === 'change-password') { setChangePasswordOpen(true); return }
  }

  return {
    ...defaultSettings,

    // ========== 动态菜单数据 ==========
    menuDataRender: () => menuData,

    // ========== 菜单选中与展开控制 ==========
    menuProps: {
      selectedKeys: [pathname],
      openKeys: menuOpenKeys,
      onOpenChange: handleMenuOpenChange,
    },

    // ========== 菜单项渲染（处理点击跳转） ==========
    // 关键修复：父级菜单项有 path 时，通过 children 判断是否为 SubMenu
    // 有 children 的项交给 ProLayout 处理展开/折叠，不拦截点击事件
    menuItemRender: (item: MenuItem, dom: React.ReactNode) => {
      // 特殊处理：返回全局概览按钮
      if (item.key === 'return-global') {
        return (
          <a
            onClick={() => {
              setCurrentCluster(null)
              history.push('/dashboard')
            }}
          >
            {dom}
          </a>
        )
      }

      // 父级菜单项（有 children）—— 交给 ProLayout 控制展开/折叠
      if (item.children && item.children.length > 0) {
        return dom
      }

      // 叶子菜单项（无 children）—— 包装点击跳转
      if (!item.path) {
        return dom
      }

      return (
        <a
          onClick={() => {
            const targetPath = item.path!
            history.push(targetPath)
          }}
        >
          {dom}
        </a>
      )
    },

    // ========== 收起菜单项中告警中心带 Badge ==========
    subMenuItemRender: (item: MenuItem, dom: React.ReactNode) => {
      if (item.name === '监控告警') {
        return (
          <span className="app-menu-label-with-badge">
            {dom}
            {activeNavIncidents.length > 0 ? (
              <Badge
                count={activeNavIncidents.length}
                overflowCount={99}
                size="small"
                color={criticalNavIncidents > 0 ? '#dc2626' : '#b45309'}
                title={`${activeNavIncidents.length} 个未恢复事件，其中 ${criticalNavIncidents} 个严重事件`}
              />
            ) : null}
          </span>
        )
      }
      return dom
    },

    avatarProps: false,
    onMenuHeaderClick: () => history.push('/dashboard'),
    // 仍由 ProLayout 维护收缩状态和过渡；这里只替换官方触发器的内容。
    collapsedButtonRender: (collapsed: boolean | undefined, defaultDom: React.ReactNode) => {
      const collapseAction = isValidElement(defaultDom)
        ? (defaultDom as ReactElement<{ onClick?: React.MouseEventHandler<HTMLButtonElement> }>).props.onClick
        : undefined

      return (
        <button
          type="button"
          className={`ant-pro-sider-collapsed-button app-sider-collapse-control${collapsed ? ' app-sider-collapse-control--collapsed' : ''}`}
          aria-label={collapsed ? '展开侧边栏' : '收起侧边栏'}
          onClick={collapseAction}
        >
          {collapsed ? <DoubleRightOutlined /> : <DoubleLeftOutlined />}
          {!collapsed && <span>收起</span>}
        </button>
      )
    },

    // ========== 右侧内容区：集群选择器 + 用户头像 ==========
    rightContentRender: () => (
      <Space size={12} className="app-layout-header-actions">
        {!isAdminMode ? (
          <Select
            allowClear
            value={currentCluster?.id}
            placeholder="选择集群..."
            style={{ width: 200 }}
            options={clusters.map((cluster) => ({
              value: String(cluster.id),
              label: (
                <Space>
                  <span>{cluster.name}</span>
                  <span className="app-cluster-option__version">{cluster.k8sVersion}</span>
                  <Tag color={getClusterStatusColor(cluster.status)}>
                    {getClusterStatusText(cluster.status)}
                  </Tag>
                </Space>
              ),
            }))}
            onChange={(clusterId) => {
              if (!clusterId) {
                setCurrentCluster(null)
                history.push('/dashboard')
                return
              }
              const cluster = clusters.find((item) => String(item.id) === clusterId)
              if (cluster) {
                enterClusterWorkspace(cluster, { setCurrentCluster })
              }
            }}
          />
        ) : null}
        <Dropdown
          overlayClassName="app-sider-account-dropdown"
          menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
          placement="bottomRight"
          trigger={['click']}
        >
          <button type="button" className="app-layout-user-trigger">
            <Avatar size={28} icon={<UserOutlined />} />
            <span className="app-layout-user-trigger__name">{currentUserName}</span>
            <DownOutlined className="app-layout-user-trigger__arrow" />
          </button>
        </Dropdown>
        <ChangePasswordModal
          open={changePasswordOpen}
          onClose={() => setChangePasswordOpen(false)}
          onSuccess={handleLogout}
        />
      </Space>
    ),

    // ========== 内容区域渲染 ==========
    childrenRender: (children: React.ReactNode) => (
      <div className="app-layout-main">
        <div className="app-layout-main__body">
          <div className="app-layout-content-frame">{children}</div>
        </div>
        <div className="app-layout-footer-frame">
          <footer className="app-layout-statusbar">
            <div className="app-layout-footer-meta">
              <span className="app-layout-footer-meta__brand">AIOPS 智能运维平台</span>
              <span>© 2026 AIOPS</span>
              <a
                href="https://www.apache.org/licenses/LICENSE-2.0"
                target="_blank"
                rel="noreferrer"
              >
                Apache License 2.0
              </a>
            </div>
          </footer>
        </div>
      </div>
    ),
    // ========== 请求配置 ==========
  }
}

// ============================================================
// 请求配置
// ============================================================
export const request: RequestConfig = {
  timeout: 30_000,
  errorConfig: {},
  requestInterceptors: [
    (config: any) => {
      const token = localStorage.getItem('token')
      if (token) {
        config.headers = {
          ...config.headers,
          Authorization: `Bearer ${token}`,
        }
      }
      return config
    },
  ],
  responseInterceptors: [
    (response: any) => {
      const { status, data: payload } = response

      if (status === 401) {
        localStorage.removeItem('token')
        if (!isLoginRoute()) {
          history.replace(LOGIN_PATH)
        }
        message.error('登录已过期，请重新登录')
        throw createRequestError('登录已过期，请重新登录', 401)
      }

      const isApiEnvelope = payload && typeof payload === 'object' && 'code' in payload && 'message' in payload && 'data' in payload
      if (!isApiEnvelope) {
        return response
      }

      if (payload.code === 401 || payload.code === 1002) {
        localStorage.removeItem('token')
        if (!isLoginRoute()) {
          history.replace(LOGIN_PATH)
        }
        message.error(payload.message || '登录已过期，请重新登录')
        throw createRequestError(payload.message || '登录已过期，请重新登录', payload.code, payload.data)
      }

      if (payload.code !== 0) {
        throw createRequestError(payload.message || '请求失败', payload.code, payload.data)
      }

      response.data = payload.data
      return response
    },
  ],
}
