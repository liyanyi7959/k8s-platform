import { computed } from 'vue'
import { useRoute } from 'vue-router'

import { useUserStore } from '@/app/store/user'
import { K8sClusterIcon } from '@/shared/icons/appIcons'

export interface MenuItem {
  title: string
  path: string
  icon?: any
  perm?: string | string[]
  children?: MenuItem[]
  desc?: string
}

export interface NavGroup {
  key: string
  title: string
  icon: any
  path?: string
  children?: MenuItem[]
}

export function useMenu() {
  const route = useRoute()
  const userStore = useUserStore()

  const perms = computed(() => userStore.permissions)

  const hasPerm = (need?: string | string[]) => {
    if (!need) return true
    if (Array.isArray(need)) return need.some((p) => perms.value.includes(p))
    return perms.value.includes(need)
  }

  const allGroups: NavGroup[] = [
    {
      key: 'k8s',
      title: 'K8s 管理',
      icon: K8sClusterIcon,
      path: '/clusters',
      children: [
        {
          title: '集群管理',
          desc: '导入、查看并进入 K8s 集群',
          path: '/clusters',
          icon: K8sClusterIcon,
          perm: 'cluster:read'
        }
      ]
    }
  ]

  const visibleGroups = computed<NavGroup[]>(() => {
    return allGroups
      .filter((group) => !group.children || group.children.some((child) => hasPerm(child.perm)))
      .map((group) => ({
        ...group,
        children: group.children?.filter((child) => hasPerm(child.perm)) || []
      }))
  })

  const activeGroup = computed<NavGroup | undefined>(() => {
    const path = route.path
    if (path.startsWith('/clusters') || path.startsWith('/k8s')) {
      return visibleGroups.value.find((group) => group.key === 'k8s')
    }
    return visibleGroups.value[0]
  })

  const activeGroupKey = computed(() => activeGroup.value?.key ?? '')
  const sidebarItems = computed<MenuItem[]>(() => activeGroup.value?.children || [])
  const activeMenuPath = computed(() => route.path)
  const hasSidebarItems = computed(() => (activeGroup.value?.children?.length ?? 0) > 0)

  return {
    railGroups: visibleGroups,
    activeGroup,
    activeGroupKey,
    sidebarItems,
    hasSidebarItems,
    activeMenuPath,
    hasPerm
  }
}
