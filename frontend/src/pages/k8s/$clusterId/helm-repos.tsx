import React, { useMemo, useState } from 'react'
import { Button, Card, Form, Input, Modal, Popconfirm, Space, Table, Tag, Tooltip, Typography, message } from 'antd'
import { DeleteOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { addHelmRepo, deleteHelmRepo, listHelmRepos, type HelmRepository } from '@/services/k8s'
import { useClusterId } from '@/hooks/useClusterId'

const { Text } = Typography

/** Helm 仓库与 Helm Release 一样，直接读取和维护目标集群 Master 的状态。 */
const HelmReposPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [addOpen, setAddOpen] = useState(false)
  const [addForm] = Form.useForm<HelmRepository>()

  const repositoriesQuery = useQuery({
    queryKey: ['helm-repos', clusterId],
    queryFn: ({ signal }) => listHelmRepos(clusterId, signal),
    enabled: !!clusterId,
    staleTime: 30_000,
  })
  const refresh = () => queryClient.invalidateQueries({ queryKey: ['helm-repos', clusterId] })
  const addMutation = useMutation({
    mutationFn: (values: HelmRepository) => addHelmRepo(clusterId, values),
    onSuccess: () => {
      message.success('仓库已同步到当前集群 Master')
      setAddOpen(false)
      addForm.resetFields()
      refresh()
    },
    onError: (error: any) => message.error(error?.message || '同步 Helm 仓库失败'),
  })
  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteHelmRepo(clusterId, name),
    onSuccess: () => {
      message.success('仓库已移除')
      refresh()
    },
    onError: (error: any) => message.error(error?.message || '移除 Helm 仓库失败'),
  })

  const filteredData = useMemo(() => (repositoriesQuery.data || []).filter((item) => {
    const keyword = search.trim().toLowerCase()
    return !keyword || item.name.toLowerCase().includes(keyword) || item.url.toLowerCase().includes(keyword)
  }), [repositoriesQuery.data, search])

  return <AppPage>
    <Card
      title="Helm 仓库"
      extra={<Text type="secondary">数据来自当前集群 Master</Text>}
    >
      <Space style={{ marginBottom: 16 }} wrap>
        <Input
          placeholder="搜索仓库名称或 URL"
          prefix={<SearchOutlined />}
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          style={{ width: 300 }}
          allowClear
        />
        <Button icon={<ReloadOutlined />} loading={repositoriesQuery.isFetching} onClick={() => repositoriesQuery.refetch()}>刷新</Button>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>同步仓库</Button>
      </Space>
      {repositoriesQuery.isError && <div style={{ marginBottom: 16 }}>
        <Text type="danger">无法读取当前 Master 的 Helm 仓库：{(repositoriesQuery.error as Error)?.message || '请检查 Master SSH 与 Helm 状态后重试。'}</Text>
      </div>}
      <Table
        dataSource={filteredData}
        rowKey="name"
        loading={repositoriesQuery.isLoading}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (total) => `共 ${total} 个仓库` }}
        locale={{ emptyText: repositoriesQuery.isError ? '请处理上方连接错误后重试' : '当前 Master 尚未同步 Helm 仓库' }}
        scroll={{ x: 760 }}
        columns={[
          { title: '仓库名称', dataIndex: 'name', width: 220, ellipsis: true, render: (value: string) => <Tag color="blue">{value}</Tag> },
          { title: '仓库地址', dataIndex: 'url', ellipsis: true },
          {
            title: '操作', width: 100, align: 'center' as const,
            render: (_: unknown, record: HelmRepository) => <Tooltip title="仅移除 Helm 仓库配置，不会卸载已有 Release"><Popconfirm title={`移除仓库 ${record.name}？`} description="不会影响已安装的 Release。" okText="移除" okButtonProps={{ danger: true }} onConfirm={() => deleteMutation.mutate(record.name)}><Button type="text" danger loading={deleteMutation.isPending} icon={<DeleteOutlined />} /></Popconfirm></Tooltip>,
          },
        ]}
      />
    </Card>
    <Modal
      title="同步 Helm 仓库"
      open={addOpen}
      onCancel={() => { setAddOpen(false); addForm.resetFields() }}
      onOk={async () => addMutation.mutate(await addForm.validateFields())}
      okText="同步到 Master"
      confirmLoading={addMutation.isPending}
      width={560}
      destroyOnClose
    >
      <Form form={addForm} layout="vertical">
        <Form.Item name="name" label="仓库名称" rules={[{ required: true, pattern: /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/, message: '使用小写字母、数字和连字符' }]}><Input placeholder="例如 bitnami" /></Form.Item>
        <Form.Item name="url" label="仓库地址" rules={[{ required: true, type: 'url', message: '请输入 HTTPS 仓库地址' }, { pattern: /^https:\/\//, message: '仅允许 HTTPS 仓库地址' }]}><Input placeholder="https://charts.bitnami.com/bitnami" /></Form.Item>
      </Form>
    </Modal>
  </AppPage>
}

export default HelmReposPage
