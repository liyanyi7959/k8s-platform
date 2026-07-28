/**
 * 通用 K8s 资源列表组件
 * 基于 ProTable 模板，配合 listGenericResources / deleteGenericResource / getResourceYaml
 * 后端 List 返回完整 K8s Unstructured 对象，raw 字段包含 apiVersion/kind/metadata/spec/status
 * 通过 rawField 辅助函数可从 raw 中提取任意嵌套字段用于专属列展示
 */
import React, { useMemo, useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Popconfirm, message, Drawer, Descriptions, Space, Tooltip, Typography, Button, Empty, Input } from 'antd'
import { DeleteOutlined, ProfileOutlined, EyeOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listGenericResources, deleteGenericResource, type GenericResourceItem } from '@/features/kops/api/k8s'
import { AppPage, ListWorkspace, NamespaceSelector, ManifestApplyDrawer } from '@/components'
import AppAlert from '@/components/AppAlert'
import EllipsisText from '@/components/EllipsisText'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'

const { Text } = Typography

export interface GenericResourceListProps {
  /** 资源标识（对应 listGenericResources 的 resource 参数） */
  resource: string
  /** 资源显示名（如 Endpoints） */
  title: string
  /** 是否命名空间级资源 */
  namespaced?: boolean
  /** 额外列（插入在名称/命名空间之后、创建时间之前） */
  extraColumns?: ProColumns<GenericResourceItem>[]
  /** 是否允许删除，默认 true */
  deletable?: boolean
  /** 是否允许创建，默认 true */
  creatable?: boolean
  /** 创建时预填充 YAML 模板 */
  createTemplate?: string
  /** 自动刷新间隔（毫秒），如 15000 = 15秒 */
  refetchInterval?: number
  /** 自定义详情抽屉内容 */
  renderDetail?: (record: GenericResourceItem) => React.ReactNode
}

/**
 * 从 K8s Unstructured raw 对象中按点分路径提取嵌套字段
 * 如 rawField(record, 'spec.type') → 'ClusterIP'
 */
export function rawField(record: GenericResourceItem, path: string): unknown {
  const parts = path.split('.')
  let current: unknown = record.raw
  for (const key of parts) {
    if (current == null || typeof current !== 'object') return undefined
    current = (current as Record<string, unknown>)[key]
  }
  return current
}

/** 快速构建一个从 raw 提取字段的列 */
export function rawColumn(
  title: string,
  path: string,
  opts?: {
    width?: number
    align?: 'left' | 'center' | 'right'
    sorter?: (a: GenericResourceItem, b: GenericResourceItem) => number
    render?: (val: unknown) => React.ReactNode
  },
): ProColumns<GenericResourceItem> {
  return {
    title,
    width: opts?.width,
    align: opts?.align,
    search: false,
    sorter: opts?.sorter,
    render: (_: unknown, record: GenericResourceItem) => {
      const val = rawField(record, path)
      if (val == null || val === '') return '-'
      if (opts?.render) return opts.render(val)
      if (Array.isArray(val)) return val.length > 0 ? val.join(', ') : '-'
      if (typeof val === 'object') return JSON.stringify(val)
      return String(val)
    },
  }
}

const GenericResourceList: React.FC<GenericResourceListProps> = ({
  resource,
  title,
  namespaced = false,
  extraColumns = [],
  deletable = true,
  creatable = true,
  createTemplate,
  refetchInterval,
  renderDetail,
}) => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [keyword, setKeyword] = useState<string>('')
  const [detail, setDetail] = useState<GenericResourceItem | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const yamlDrawer = useYamlDrawer()

  const queryKey = ['k8s-generic', resource, clusterId, namespace]

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey,
    queryFn: ({ signal }) =>
      listGenericResources(clusterId, resource, namespaced ? namespace || undefined : undefined, signal),
    enabled: !!clusterId,
    refetchInterval,
  })

  const deleteMutation = useMutation({
    mutationFn: (record: GenericResourceItem) =>
      deleteGenericResource(clusterId, resource, record.name, record.namespace),
    onSuccess: () => {
      message.success(`${title} 已删除`)
      queryClient.invalidateQueries({ queryKey })
    },
    onError: () => message.error('删除失败'),
  })

  // 客户端按名称过滤
  const filteredItems = useMemo(() => {
    const items = data?.items || []
    if (!keyword.trim()) return items
    return items.filter((item) => item.name.toLowerCase().includes(keyword.toLowerCase()))
  }, [data?.items, keyword])

  const columns = useMemo<ProColumns<GenericResourceItem>[]>(() => {
    const base: ProColumns<GenericResourceItem>[] = []

    if (namespaced) {
      base.push({
        title: 'Namespace',
        dataIndex: 'namespace',
        width: 140,
        search: false,
        ellipsis: true,
        render: (_, r) => <EllipsisText text={r.namespace} tag />,
      })
    }

    base.push({
      title: '名称',
      dataIndex: 'name',
      width: 160,
      ellipsis: true,
      copyable: true,
      render: (_, r) => <Text strong>{r.name}</Text>,
    })

    base.push(...extraColumns)

    base.push({
      title: 'Age',
      dataIndex: 'createdAt',
      width: 110,
      align: 'center' as const,
      search: false,
      ellipsis: true,
      render: (_, record) => (record.createdAt ? formatDate(record.createdAt) : '-'),
    })

    base.push({
      title: '操作',
      valueType: 'option',
      width: 120,
      fixed: 'right',
      align: 'center',
      render: (_, record) => (
        <Space size="small" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Tooltip title="详情">
            <a onClick={() => setDetail(record)}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name, record.namespace)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          {deletable && (
            <Popconfirm title={`确定删除该 ${title}？`} onConfirm={() => deleteMutation.mutate(record)}>
              <Tooltip title="删除">
                <a style={{ color: '#dc2626' }}>
                  <DeleteOutlined />
                </a>
              </Tooltip>
            </Popconfirm>
          )}
        </Space>
      ),
    })

    return base
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [namespaced, extraColumns, deletable, title])

  return (
    <AppPage>
      <ListWorkspace
        summary={<>当前共 {filteredItems.length} 个 {title}</>}
        actions={(
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>
            {creatable ? (
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>创建</Button>
            ) : null}
          </Space>
        )}
        filters={(
          <Space wrap>
            <Input
              allowClear
              placeholder="搜索名称..."
              prefix={<SearchOutlined />}
              style={{ width: 180 }}
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
            {namespaced ? (
              <NamespaceSelector
                clusterId={clusterId}
                value={namespace}
                onChange={setNamespace}
                style={{ width: 180 }}
              />
            ) : null}
          </Space>
        )}
      >
        {isError ? (
          <AppAlert
            type="error"
            showIcon
            message="数据加载失败"
            description={`无法获取 ${title} 列表，请检查集群连接状态或稍后重试。`}
            action={<Button size="small" onClick={() => refetch()}>重试</Button>}
          />
        ) : null}
        <ProTable<GenericResourceItem>
        columns={columns}
        dataSource={filteredItems}
        loading={isLoading}
        rowKey={(r) => `${r.namespace || ''}/${r.name}`}
        search={false}
        options={{ reload: false }}
        pagination={{
          defaultPageSize: 20,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
        scroll={{ x: 800 }}
        locale={{
          emptyText: <Empty description={`暂无 ${title} 数据`} />,
        }}
          toolBarRender={false}
          headerTitle={false}
        />
      </ListWorkspace>

      {creatable && (
        <ManifestApplyDrawer
          clusterId={clusterId}
          open={createOpen}
          onClose={() => setCreateOpen(false)}
          title={`创建 ${title}`}
          initialYaml={createTemplate}
        />
      )}

      <YamlDrawer
        clusterId={clusterId}
        resourceType={resource}
        namespace={yamlDrawer.namespace}
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />

      <Drawer
        title={`${title} 详情`}
        open={!!detail}
        onClose={() => setDetail(null)}
        width={720}
      >
        {detail && (
          renderDetail ? (
            renderDetail(detail)
          ) : (
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="名称">{detail.name}</Descriptions.Item>
              {namespaced && (
                <Descriptions.Item label="命名空间">{detail.namespace || '-'}</Descriptions.Item>
              )}
              {detail.kind && <Descriptions.Item label="Kind">{detail.kind}</Descriptions.Item>}
              {detail.status && <Descriptions.Item label="状态">{detail.status}</Descriptions.Item>}
              <Descriptions.Item label="创建时间">
                {detail.createdAt ? formatDate(detail.createdAt) : '-'}
              </Descriptions.Item>
            </Descriptions>
          )
        )}
      </Drawer>
    </AppPage>
  )
}

export default GenericResourceList
