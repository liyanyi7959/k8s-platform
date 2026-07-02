import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'

import { useUserStore } from '@/app/store/user'
import {
  AiAssistantIcon,
  AuditLogIcon,
  K8sClusterIcon,
  ModelHubIcon,
  PowerSwitchIcon,
  RoleManageIcon,
  SystemSettingsIcon,
  TerminalConsoleIcon,
  UserManageIcon
} from '@/shared/icons/appIcons'

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

/* ── 系统管理模式（全局单例） ───────────────────────────────────────── */
const systemMode = ref(false)

/** 进入系统管理模式（侧边栏仅显示系统管理菜单） */
export function enterSystemMode() {
  systemMode.value = true
}

/** 退出系统管理模式（恢复默认侧边栏） */
export function exitSystemMode() {
  systemMode.value = false
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
      title: 'K8S 管理',
      icon: K8sClusterIcon,
      path: '/clusters',
      children: [
        {
          title: '集群管理',
          desc: '导入、查看并进入 K8s 集群',
          path: '/clusters',
          icon: K8sClusterIcon,
          perm: 'cluster:read'
        },
        {
          title: '在线部署',
          desc: '部署计划与执行任务管理',
          path: '/deploy/online',
          icon: TerminalConsoleIcon,
          perm: ['deploy:server_read', 'deploy:plan_read']
        }
      ]
    },
    {
      key: 'ai',
      title: 'AI 助手',
      icon: AiAssistantIcon,
      path: '/ai/assistant',
      children: [
        {
          title: '排障助手',
          desc: '面向故障排查、诊断和会话沉淀',
          path: '/ai/assistant',
          icon: AiAssistantIcon,
          perm: ['ai:chat', 'ai:diagnose']
        }
      ]
    },
    {
      key: 'system',
      title: '系统管理',
      icon: SystemSettingsIcon,
      path: '/system/audit-logs',
      children: [
        { title: '操作审计', path: '/system/audit-logs', icon: AuditLogIcon, perm: 'user:read' },
        { title: '用户管理', path: '/system/users', icon: UserManageIcon, perm: 'user:write' },
        { title: '角色管理', path: '/system/roles', icon: RoleManageIcon, perm: 'user:write' },
        { title: '凭据管理', desc: 'SSH 凭据集中管理', path: '/system/credentials', icon: PowerSwitchIcon, perm: 'deploy:server_read' },
        { title: '部署配置', desc: '部署流程命令与仓库配置', path: '/system/deploy-config', icon: SystemSettingsIcon, perm: 'deploy:plan_read' },
        { title: '模型配置', desc: '管理多模型与多提供商接入', path: '/system/ai-settings', icon: ModelHubIcon, perm: 'ai:model_admin' }
      ]
    }
  ]

  const visibleGroups = computed<NavGroup[]>(() => {
    const filtered = allGroups
      .filter((group) => !group.children || group.children.some((child) => hasPerm(child.perm)))
      .map((group) => ({
        ...group,
        children: group.children?.filter((child) => hasPerm(child.perm)) || []
      }))

    // 系统管理模式下只显示系统管理分组
    if (systemMode.value) {
      return filtered.filter((g) => g.key === 'system')
    }
    // 默认模式下隐藏系统管理分组（移至用户下拉菜单）
    return filtered.filter((g) => g.key !== 'system')
  })

  const activeGroup = computed<NavGroup | undefined>(() => {
    const path = route.path

    // 系统管理模式下始终返回系统分组
    if (systemMode.value) {
      return allGroups.find((group) => group.key === 'system')
    }

    if (path.startsWith('/system')) {
      return allGroups.find((group) => group.key === 'system')
    }
    if (path.startsWith('/clusters') || path.startsWith('/k8s') || path.startsWith('/deploy')) {
      return visibleGroups.value.find((group) => group.key === 'k8s')
    }
    if (path.startsWith('/ai')) {
      return visibleGroups.value.find((group) => group.key === 'ai')
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
    hasPerm,
    systemMode
  }
}
