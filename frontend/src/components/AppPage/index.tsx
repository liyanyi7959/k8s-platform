import React from 'react'
import { useLocation } from '@umijs/max'
import type { BreadcrumbProps } from 'antd'
import { PageContainer, type PageContainerProps } from '@ant-design/pro-components'

type AppPageProps = PageContainerProps & {
  keepHeaderTitle?: boolean
}

/**
 * 面包屑只服务于需要“向上返回”的真实层级页面。
 * 总览、工作台、列表页已有侧栏定位，重复展示会增加视觉噪音。
 */
const getAutomaticBreadcrumb = (pathname: string): BreadcrumbProps | undefined => {
  const clusterRoot = { title: '集群管理', href: '/clusters' }

  if (pathname === '/clusters/import') {
    return { items: [clusterRoot, { title: '集群导入' }] }
  }

  if (pathname === '/clusters/provision/create') {
    return { items: [clusterRoot, { title: '部署 K8S 集群', href: '/clusters/provision' }, { title: '创建部署方案' }] }
  }

  if (/^\/clusters\/provision\/[^/]+\/edit$/.test(pathname)) {
    return { items: [clusterRoot, { title: '部署 K8S 集群', href: '/clusters/provision' }, { title: '编辑部署方案' }] }
  }

  if (/^\/clusters\/provision\/[^/]+$/.test(pathname)) {
    return { items: [clusterRoot, { title: '部署 K8S 集群', href: '/clusters/provision' }, { title: '部署方案详情' }] }
  }

  // /clusters/hosts 是独立的“服务器管理”工作台，不能因为 URL 前缀误归入集群管理。
  if (/^\/clusters\/(?!hosts$|import$|provision$)[^/]+$/.test(pathname)) {
    return { items: [clusterRoot, { title: '集群详情' }] }
  }

  // CI/CD 详情页面包屑
  const cicdRoot = { title: 'CI/CD', href: '/cicd/overview' }
  if (pathname === '/cicd/pipelines/new' || /^\/cicd\/pipelines\/[^/]+\/edit$/.test(pathname)) return undefined
  if (/^\/cicd\/pipelines\/[^/]+$/.test(pathname)) return { items: [cicdRoot, { title: '流水线', href: '/cicd/pipelines' }, { title: '详情' }] }
  if (/^\/cicd\/runs\/[^/]+$/.test(pathname)) return { items: [cicdRoot, { title: '执行记录', href: '/cicd/runs' }, { title: '详情' }] }
  if (/^\/cicd\/artifacts\/[^/]+$/.test(pathname)) return { items: [cicdRoot, { title: '制品仓库', href: '/cicd/artifacts' }, { title: '详情' }] }
  if (/^\/cicd\/environments\/[^/]+$/.test(pathname)) return { items: [cicdRoot, { title: '环境管理', href: '/cicd/environments' }, { title: '详情' }] }

  const configNames: Record<string, string> = {
    '/config/users': '用户管理',
    '/config/roles': '角色管理',
    '/config/audit-logs': '审计日志',
    '/config/credentials': '凭据库',
    '/config/deploy-assets': '部署手册',
    '/config/settings': '系统设置',
  }

  const configName = configNames[pathname]
  return configName ? { items: [{ title: '系统配置' }, { title: configName }] } : undefined
}

const AppPage: React.FC<AppPageProps> = ({
  keepHeaderTitle = false,
  title,
  header,
  className,
  breadcrumbRender,
  breadcrumb,
  ...rest
}) => {
  const location = useLocation()
  const mergedHeader =
    keepHeaderTitle || !header
      ? header
      : {
          ...header,
          title: undefined,
          subTitle: undefined,
          backIcon: false,
          onBack: undefined,
        }

  const mergedBreadcrumb = breadcrumb ?? getAutomaticBreadcrumb(location.pathname)
  const mergedBreadcrumbRender = breadcrumbRender === false
    ? false
    : breadcrumbRender ?? (mergedBreadcrumb ? ((_, defaultDom) => defaultDom) : false)

  // Most pages provide their own workspace heading.  Keeping a title-only
  // PageHeader here would still reserve its header height even though the
  // global layout intentionally hides the title, making those pages start
  // lower than pages without a title.  Retain the header only when it also
  // carries a breadcrumb (where the reserved row is meaningful).
  const pageTitle = mergedBreadcrumb && mergedBreadcrumbRender !== false && keepHeaderTitle ? title : false

  return (
    <PageContainer
      {...rest}
      className={['app-page-container', className].filter(Boolean).join(' ')}
      title={pageTitle}
      header={mergedHeader}
      breadcrumb={mergedBreadcrumb}
      breadcrumbRender={mergedBreadcrumbRender}
    />
  )
}

export default AppPage
