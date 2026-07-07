import React, { useState } from 'react'
import { Card, Input, Button, Table, Tag, Drawer, Descriptions, Space, Tooltip, message } from 'antd'
import { ReloadOutlined, SearchOutlined, EyeOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { listHelmReleases, getHelmReleaseDetail } from '@/services/k8s'
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

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['helm-releases', clusterId],
    queryFn: ({ signal }) => listHelmReleases(Number(clusterId), signal),
    enabled: !!clusterId,
    staleTime: 60_000,
  })

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
      title: '操作', width: 80, align: 'center' as const,
      render: (_: any, record: any) => (
        <Tooltip title="查看详情">
          <a onClick={() => handleViewDetail(record)}><EyeOutlined /></a>
        </Tooltip>
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
    </AppPage>
  )
}

export default HelmReleasesPage
