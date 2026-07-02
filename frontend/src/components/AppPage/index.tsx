import React from 'react'
import { PageContainer, type PageContainerProps } from '@ant-design/pro-components'

type AppPageProps = PageContainerProps & {
  keepHeaderTitle?: boolean
}

const AppPage: React.FC<AppPageProps> = ({
  keepHeaderTitle = false,
  title,
  header,
  className,
  breadcrumbRender,
  ...rest
}) => {
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

  const mergedBreadcrumbRender =
    breadcrumbRender === false ? false : breadcrumbRender ?? ((_, defaultDom) => defaultDom)

  return (
    <PageContainer
      {...rest}
      className={['app-page-container', className].filter(Boolean).join(' ')}
      title={keepHeaderTitle ? title : false}
      header={mergedHeader}
      breadcrumbRender={mergedBreadcrumbRender}
    />
  )
}

export default AppPage
