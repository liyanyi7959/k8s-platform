import React, { useMemo, useState } from 'react'
import { history } from '@umijs/max'
import { AppstoreOutlined, ClockCircleOutlined, SearchOutlined, SettingOutlined } from '@ant-design/icons'
import { Button, Empty, Input, Space, Tag, Typography } from 'antd'
import { AppPage } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'

const { Paragraph, Title } = Typography

type ResourceEntry = {
  group: string
  name: string
  path: string
}

const resourceEntries: ResourceEntry[] = [
  { group: '应用与运行时', name: 'Pods', path: 'pods' },
  { group: '应用与运行时', name: 'Deployments', path: 'deployments' },
  { group: '应用与运行时', name: 'StatefulSets', path: 'statefulsets' },
  { group: '应用与运行时', name: 'Jobs', path: 'jobs' },
  { group: '网络与入口', name: 'Services', path: 'services' },
  { group: '网络与入口', name: 'Ingresses', path: 'ingresses' },
  { group: '存储与配额', name: 'PVCs', path: 'pvcs' },
  { group: '存储与配额', name: 'StorageClasses', path: 'storage-classes' },
  { group: '配置与访问控制', name: 'ConfigMaps', path: 'configmaps' },
  { group: '配置与访问控制', name: 'Secrets', path: 'secrets' },
  { group: '集群与扩展治理', name: 'Namespaces', path: 'namespaces' },
  { group: '集群与扩展治理', name: 'Nodes', path: 'nodes' },
]

const recentResourcePaths = ['pods', 'deployments', 'services', 'configmaps']

const AdvancedResourcesPage: React.FC = () => {
  const clusterId = useClusterId()
  const [query, setQuery] = useState('')
  const normalizedQuery = query.trim().toLowerCase()

  const visibleResources = useMemo(
    () => resourceEntries.filter((resource) => (
      !normalizedQuery
      || resource.name.toLowerCase().includes(normalizedQuery)
      || resource.group.includes(query.trim())
    )),
    [normalizedQuery, query],
  )

  const openResource = (path: string) => history.push(`/k8s/${clusterId}/${path}`)
  const recentResources = resourceEntries.filter((resource) => recentResourcePaths.includes(resource.path))

  return (
    <AppPage>
      <div className="app-resource-explorer">
        <section className="app-resource-explorer__intro">
          <div>
            <Space size={10} align="center">
              <span className="app-resource-explorer__intro-icon"><AppstoreOutlined /></span>
              <Title level={3}>资源浏览器</Title>
              <Tag color="blue">目录模式</Tag>
            </Space>
            <Paragraph>
              通过左侧资源目录持续浏览 Kubernetes 对象；进入任意资源后，目录会保留当前分组与选中位置。
            </Paragraph>
          </div>
          <Button icon={<SettingOutlined />} onClick={() => history.push(`/k8s/${clusterId}/topology`)}>
            打开资源关系图
          </Button>
        </section>

        <section className="app-resource-explorer__workspace" aria-label="资源快速定位">
          <div className="app-resource-explorer__command">
            <span className="app-resource-explorer__command-icon"><SearchOutlined /></span>
            <Input
              allowClear
              bordered={false}
              placeholder="搜索资源，例如 Pods、Services、ConfigMaps"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <span className="app-resource-explorer__command-hint">在左侧目录中继续浏览全部资源</span>
          </div>

          <div className="app-resource-explorer__body">
            <div className="app-resource-explorer__recent">
              <div className="app-resource-explorer__section-title">
                <ClockCircleOutlined />
                <span>常用资源</span>
              </div>
              <div className="app-resource-explorer__quick-list">
                {recentResources.map((resource) => (
                  <Button key={resource.path} type="text" onClick={() => openResource(resource.path)}>
                    {resource.name}
                  </Button>
                ))}
              </div>
            </div>

            <div className="app-resource-explorer__results">
              <div className="app-resource-explorer__section-title">
                <SearchOutlined />
                <span>{normalizedQuery ? '匹配资源' : '快速定位'}</span>
                <span className="app-resource-explorer__result-count">{visibleResources.length}</span>
              </div>
              {visibleResources.length ? (
                <div className="app-resource-explorer__result-list">
                  {visibleResources.map((resource) => (
                    <button
                      key={resource.path}
                      type="button"
                      className="app-resource-explorer__result"
                      onClick={() => openResource(resource.path)}
                    >
                      <span className="app-resource-explorer__result-name">{resource.name}</span>
                      <span className="app-resource-explorer__result-group">{resource.group}</span>
                    </button>
                  ))}
                </div>
              ) : (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="未找到匹配资源" />
              )}
            </div>
          </div>
        </section>
      </div>
    </AppPage>
  )
}

export default AdvancedResourcesPage
