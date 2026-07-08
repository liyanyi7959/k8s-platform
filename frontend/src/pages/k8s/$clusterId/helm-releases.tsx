import React, { useState } from 'react'
import { Card, Input, Button, Table, Tag, Drawer, Descriptions, Space, Tooltip, Modal, Popconfirm, Form, message } from 'antd'
import { ReloadOutlined, SearchOutlined, EyeOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons'
import { useQuery, useMutation } from '@tanstack/react-query'
import { AppPage, YamlEditor } from '@/components'
import { listHelmReleases, getHelmReleaseDetail, helmInstall, helmUninstall } from '@/services/k8s'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'

// 状态颜色映射
const statusColorMap: Record<string, string> = {
  deployed: 'success',
  uninstalled: 'default',
  'pending-upgrade': 'warning',
  'pending-rollback': 'warning',
  'pending-install': 'processing',
  failed: 'error',
  superseded: 'default',
}

const HelmReleasesPage: React.FC = () => {
  const clusterId = useClusterId()
  const [search, setSearch] = useState('')
  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<any>(null)
  const [installOpen, setInstallOpen] = useState(false)
  const [valuesYaml, setValuesYaml] = useState('')
  const [installForm] = Form.useForm()

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['helm-releases', clusterId],
    queryFn: ({ signal }) => listHelmReleases(Number(clusterId), signal),
    enabled: !!clusterId,
    staleTime: 60_000,
  })

  const installMutation = useMutation({
    mutationFn: (data: any) => helmInstall(Number(clusterId), data),
    onSuccess: () => {
      message.success('安装成功')
      setInstallOpen(false)
      installForm.resetFields()
      setValuesYaml('')
      refetch()
    },
    onError: (err: any) => message.error(err?.message || '安装失败'),
  })

  const uninstallMutation = useMutation({
    mutationFn: ({ ns, name }: { ns: string; name: string }) =>
      helmUninstall(Number(clusterId), ns, name),
    onSuccess: () => {
      message.success('卸载成功')
      refetch()
    },
    onError: (err: any) => message.error(err?.message || '卸载失败'),
  })

  const handleInstall = async () => {
    try {
      const values = await installForm.validateFields()
      installMutation.mutate({ ...values, values_yaml: valuesYaml })
    } catch {
      // 校验失败
    }
  }

  const filteredData = (data?.items || []).filter((item: any) => {
    if (!search) return true
    const v = search.toLowerCase()
    return item.name?.toLowerCase().includes(v) || item.namespace?.toLowerCase().includes(v) || item.chart?.toLowerCase().includes(v)
  })

  const handleViewDetail = async (record: any) => {
    try {
      const res = await getHelmReleaseDetail(Number(clusterId), record.namespace, record.name)
      setDetail(res)
      setDetailOpen(true)
    } catch {
      message.error('获取详情失败')
    }
  }

  const columns = [
    { title: '名称', dataIndex: 'name', width: 200, ellipsis: true },
    { title: '命名空间', dataIndex: 'namespace', width: 140 },
    { title: 'Revision', dataIndex: 'revision', width: 80, align: 'center' as const },
    {
      title: '状态', dataIndex: 'status', width: 120, align: 'center' as const,
      render: (status: string) => <Tag color={statusColorMap[status] || 'default'}>{status}</Tag>,
    },
    { title: 'Chart', dataIndex: 'chart', width: 180, ellipsis: true },
    { title: '版本', dataIndex: 'chart_ver', width: 100 },
    {
      title: '更新时间', dataIndex: 'updated', width: 180,
      render: (v: string) => v ? formatDate(v) : '-',
    },
    {
      title: '操作', width: 120, align: 'center' as const,
      render: (_: any, record: any) => (
        <Space>
          <Tooltip title="查看详情">
            <a onClick={() => handleViewDetail(record)}><EyeOutlined /></a>
          </Tooltip>
          <Popconfirm
            title="确认卸载该 Release？"
            description={`将卸载 ${record.namespace}/${record.name}`}
            onConfirm={() => uninstallMutation.mutate({ ns: record.namespace, name: record.name })}
            okButtonProps={{ danger: true, loading: uninstallMutation.isPending }}
          >
            <Tooltip title="卸载">
              <a style={{ color: '#ff4d4f' }}><DeleteOutlined /></a>
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <AppPage>
      <Card>
        <Space style={{ marginBottom: 16 }}>
          <Input
            placeholder="搜索名称、命名空间或 Chart"
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ width: 280 }}
            allowClear
          />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setInstallOpen(true)}>Helm 安装</Button>
        </Space>
        <Table
          dataSource={filteredData}
          columns={columns}
          rowKey={(r) => `${r.namespace}/${r.name}`}
          loading={isLoading}
          pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
          scroll={{ x: 1000 }}
        />
      </Card>
      <Drawer
        title="Helm Release 详情"
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        width={600}
      >
        {detail && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="名称">{detail.name}</Descriptions.Item>
            <Descriptions.Item label="命名空间">{detail.namespace}</Descriptions.Item>
            <Descriptions.Item label="Revision">{detail.revision}</Descriptions.Item>
            <Descriptions.Item label="状态">{detail.status}</Descriptions.Item>
            <Descriptions.Item label="Chart">{detail.chart}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{detail.updated ? formatDate(detail.updated) : '-'}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
      <Modal
        title="Helm 安装"
        open={installOpen}
        onCancel={() => {
          setInstallOpen(false)
          installForm.resetFields()
          setValuesYaml('')
        }}
        onOk={handleInstall}
        confirmLoading={installMutation.isPending}
        width={640}
        destroyOnClose
      >
        <Form form={installForm} layout="vertical" initialValues={{ namespace: 'default' }}>
          <Form.Item name="release_name" label="Release 名称" rules={[{ required: true, message: '请输入 Release 名称' }]}>
            <Input placeholder="例如：my-redis" />
          </Form.Item>
          <Form.Item name="namespace" label="命名空间" rules={[{ required: true, message: '请输入命名空间' }]}>
            <Input placeholder="default" />
          </Form.Item>
          <Form.Item name="chart" label="Chart" rules={[{ required: true, message: '请输入 Chart 名称' }]}>
            <Input placeholder="例如：bitnami/redis" />
          </Form.Item>
          <Form.Item name="repo_name" label="仓库名称（可选）">
            <Input placeholder="例如：bitnami" />
          </Form.Item>
          <Form.Item name="repo_url" label="仓库地址（可选）">
            <Input placeholder="例如：https://charts.bitnami.com/bitnami" />
          </Form.Item>
          <Form.Item label="values.yaml（可选）">
            <YamlEditor value={valuesYaml} onChange={setValuesYaml} height={300} />
          </Form.Item>
        </Form>
      </Modal>
    </AppPage>
  )
}

export default HelmReleasesPage
