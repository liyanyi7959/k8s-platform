import React, { useState } from 'react'
import { Card, Input, Button, Table, Tag, Space, Tooltip, Modal, Popconfirm, Form, message } from 'antd'
import { ReloadOutlined, SearchOutlined, PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { listHelmRepos } from '@/services/k8s'
import { useClusterId } from '@/hooks/useClusterId'

/**
 * Helm 仓库管理页面
 * 展示已添加的 Helm 仓库列表，支持搜索
 * 添加/删除仓库功能待后端 API 就绪后实现
 */
const HelmReposPage: React.FC = () => {
  const clusterId = useClusterId()
  const [search, setSearch] = useState('')
  const [addOpen, setAddOpen] = useState(false)
  const [addForm] = Form.useForm()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['helm-repos', clusterId],
    queryFn: ({ signal }) => listHelmRepos(Number(clusterId), signal),
    enabled: !!clusterId,
    staleTime: 60_000,
  })

  // 按关键词过滤
  const filteredData = (data || []).filter((item: any) => {
    if (!search) return true
    const v = search.toLowerCase()
    return item.name?.toLowerCase().includes(v) || item.url?.toLowerCase().includes(v)
  })

  const columns = [
    { title: '仓库名称', dataIndex: 'name', width: 200, ellipsis: true },
    { title: 'URL', dataIndex: 'url', ellipsis: true },
    {
      title: '状态', dataIndex: 'status', width: 120, align: 'center' as const,
      render: (status: string) => {
        const colorMap: Record<string, string> = {
          ok: 'success',
          error: 'error',
          failed: 'error',
        }
        return <Tag color={colorMap[status] || 'default'}>{status || 'unknown'}</Tag>
      },
    },
    {
      title: '操作', width: 100, align: 'center' as const,
      render: () => (
        <Tooltip title="删除仓库功能待实现">
          <Popconfirm
            title="确认删除该仓库？"
            disabled
            onConfirm={() => message.info('删除仓库功能待后端 API 就绪后实现')}
          >
            <a style={{ color: '#dc2626', opacity: 0.5 }}>
              <DeleteOutlined />
            </a>
          </Popconfirm>
        </Tooltip>
      ),
    },
  ]

  return (
    <AppPage>
      <Card>
        <Space style={{ marginBottom: 16 }}>
          <Input
            placeholder="搜索仓库名称或 URL"
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ width: 280 }}
            allowClear
          />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>添加仓库</Button>
        </Space>
        <Table
          dataSource={filteredData}
          columns={columns}
          rowKey="name"
          loading={isLoading}
          pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
          scroll={{ x: 800 }}
        />
      </Card>
      <Modal
        title="添加 Helm 仓库"
        open={addOpen}
        onCancel={() => {
          setAddOpen(false)
          addForm.resetFields()
        }}
        onOk={() => message.info('添加仓库功能待后端 API 就绪后实现')}
        okButtonProps={{ disabled: true }}
        width={560}
        destroyOnClose
      >
        <Form form={addForm} layout="vertical">
          <Form.Item name="name" label="仓库名称" rules={[{ required: true, message: '请输入仓库名称' }]}>
            <Input placeholder="例如：bitnami" />
          </Form.Item>
          <Form.Item name="url" label="仓库地址" rules={[{ required: true, message: '请输入仓库 URL' }]}>
            <Input placeholder="例如：https://charts.bitnami.com/bitnami" />
          </Form.Item>
        </Form>
        <div style={{ color: '#999', fontSize: 12 }}>
          注：添加/删除仓库功能待后端 API 就绪后实现
        </div>
      </Modal>
    </AppPage>
  )
}

export default HelmReposPage
