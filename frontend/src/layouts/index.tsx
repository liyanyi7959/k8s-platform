import React from 'react'
import { Outlet } from '@umijs/max'

/**
 * 根布局组件
 *
 * 注意：ProLayout 由 @umijs/plugin-layout 提供，
 * 配置通过 app.tsx 的 layout 导出控制。
 * 本组件仅作为 Outlet 容器。
 */
const RootLayout: React.FC = () => {
  return <Outlet />
}

export default RootLayout
