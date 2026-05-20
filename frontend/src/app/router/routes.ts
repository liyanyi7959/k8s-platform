import type { RouteRecordRaw } from 'vue-router'

export interface AppRouteMeta {
  title: string
  requiresAuth?: boolean
  perm?: string | string[]
  icon?: string
  hideInMenu?: boolean
  standalone?: boolean
  fullContent?: boolean
}

declare module 'vue-router' {
  interface RouteMeta extends AppRouteMeta {}
}

export const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/features/auth/pages/LoginView.vue'),
    meta: { title: '登录', requiresAuth: false, hideInMenu: true }
  },
  {
    path: '/',
    component: () => import('@/app/layouts/AppShell.vue'),
    meta: { title: 'Home', requiresAuth: true, hideInMenu: true },
    children: [
      {
        path: '',
        redirect: '/clusters'
      },
      {
        path: 'clusters',
        name: 'Clusters',
        component: () => import('@/features/clusters/pages/ClustersView.vue'),
        meta: { title: '集群管理', requiresAuth: true, perm: 'cluster:read' }
      },
      {
        path: 'k8s/cluster/:clusterId',
        name: 'K8sClusterManage',
        component: () => import('@/features/k8s/pages/ClusterManageView.vue'),
        meta: {
          title: 'K8s 集群管理',
          requiresAuth: true,
          perm: ['cluster:read', 'k8s:read', 'k8s:rbac_read', 'k8s:permission_audit'],
          hideInMenu: true,
          fullContent: true
        }
      },
      {
        path: 'k8s/topology',
        name: 'K8sResourceTopology',
        component: () => import('@/features/k8s/pages/ResourceTopologyView.vue'),
        meta: { title: '资源关系图', requiresAuth: true, perm: ['cluster:read', 'k8s:read'], hideInMenu: true }
      }
    ]
  }
]
