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
