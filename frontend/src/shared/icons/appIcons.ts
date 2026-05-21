import { defineComponent, h, mergeProps } from 'vue'
import k8sOfficialIconUrl from '@/assets/images/k8s-official-icon.svg'

export const K8sClusterIcon = defineComponent({
  name: 'K8sClusterIcon',
  setup(_props, { attrs }) {
    return () => h(
      'img',
      mergeProps(attrs, {
        src: k8sOfficialIconUrl,
        alt: 'Kubernetes',
        draggable: false,
        style: {
          width: '1em',
          height: '1em',
          display: 'block',
          objectFit: 'contain'
        }
      })
    )
  }
})

export const PowerSwitchIcon = defineComponent({
  name: 'PowerSwitchIcon',
  setup() {
    return () => h(
      'svg',
      {
        viewBox: '0 0 24 24',
        fill: 'none',
        xmlns: 'http://www.w3.org/2000/svg',
        style: { width: '1em', height: '1em', display: 'block' }
      },
      [
        h('path', {
          d: 'M12 3.4V11.1',
          stroke: 'currentColor',
          'stroke-width': '1.9',
          'stroke-linecap': 'round'
        }),
        h('path', {
          d: 'M8 5.45C5.95 6.83 4.6 9.18 4.6 11.85C4.6 16.08 7.99 19.5 12.2 19.5C16.41 19.5 19.8 16.08 19.8 11.85C19.8 9.18 18.45 6.83 16.4 5.45',
          stroke: 'currentColor',
          'stroke-width': '1.9',
          'stroke-linecap': 'round'
        })
      ]
    )
  }
})

export const TerminalConsoleIcon = defineComponent({
  name: 'TerminalConsoleIcon',
  setup() {
    return () => h(
      'svg',
      {
        viewBox: '0 0 24 24',
        fill: 'none',
        xmlns: 'http://www.w3.org/2000/svg',
        style: { width: '1em', height: '1em', display: 'block' }
      },
      [
        h('rect', {
          x: '3.5',
          y: '5',
          width: '17',
          height: '14',
          rx: '2.5',
          stroke: 'currentColor',
          'stroke-width': '1.7'
        }),
        h('path', {
          d: 'M7.5 10L10.2 12L7.5 14',
          stroke: 'currentColor',
          'stroke-width': '1.8',
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round'
        }),
        h('path', {
          d: 'M12.8 14H16.5',
          stroke: 'currentColor',
          'stroke-width': '1.8',
          'stroke-linecap': 'round'
        })
      ]
    )
  }
})

export const SystemSettingsIcon = defineComponent({
  name: 'SystemSettingsIcon',
  setup() {
    return () => h(
      'svg',
      {
        viewBox: '0 0 24 24',
        fill: 'none',
        xmlns: 'http://www.w3.org/2000/svg',
        style: { width: '1em', height: '1em', display: 'block' }
      },
      [
        h('path', {
          d: 'M12 15a3 3 0 100-6 3 3 0 000 6z',
          stroke: 'currentColor',
          'stroke-width': '1.7',
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round'
        }),
        h('path', {
          d: 'M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 11-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 11-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 11-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 110-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 112.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 114 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 112.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 110 4h-.09a1.65 1.65 0 00-1.51 1z',
          stroke: 'currentColor',
          'stroke-width': '1.7',
          'stroke-linecap': 'round',
          'stroke-linejoin': 'round'
        })
      ]
    )
  }
})

/* ── 审计日志图标 ──────────────────────────────────────────────────────── */
export const AuditLogIcon = defineComponent({
  name: 'AuditLogIcon',
  setup() {
    return () => h('svg', {
      viewBox: '0 0 24 24', fill: 'none',
      style: { width: '1em', height: '1em', display: 'block' }
    }, [
      h('path', { d: 'M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8l-6-6z', stroke: 'currentColor', 'stroke-width': '1.7', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
      h('path', { d: 'M14 2v6h6', stroke: 'currentColor', 'stroke-width': '1.7', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
      h('path', { d: 'M9 15l2 2 4-4', stroke: 'currentColor', 'stroke-width': '1.7', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' })
    ])
  }
})

/* ── 用户管理图标 ──────────────────────────────────────────────────────── */
export const UserManageIcon = defineComponent({
  name: 'UserManageIcon',
  setup() {
    return () => h('svg', {
      viewBox: '0 0 24 24', fill: 'none',
      style: { width: '1em', height: '1em', display: 'block' }
    }, [
      h('path', { d: 'M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2', stroke: 'currentColor', 'stroke-width': '1.7', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
      h('circle', { cx: '9', cy: '7', r: '4', stroke: 'currentColor', 'stroke-width': '1.7' }),
      h('path', { d: 'M22 21v-2a4 4 0 00-3-3.87', stroke: 'currentColor', 'stroke-width': '1.7', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
      h('path', { d: 'M16 3.13a4 4 0 010 7.75', stroke: 'currentColor', 'stroke-width': '1.7', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' })
    ])
  }
})

/* ── 角色管理图标 ──────────────────────────────────────────────────────── */
export const RoleManageIcon = defineComponent({
  name: 'RoleManageIcon',
  setup() {
    return () => h('svg', {
      viewBox: '0 0 24 24', fill: 'none',
      style: { width: '1em', height: '1em', display: 'block' }
    }, [
      h('path', { d: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z', stroke: 'currentColor', 'stroke-width': '1.7', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }),
      h('path', { d: 'M9 12l2 2 4-4', stroke: 'currentColor', 'stroke-width': '1.7', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' })
    ])
  }
})
