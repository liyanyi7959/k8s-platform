import GenericResourceList, { rawField } from '@/features/kops/components/GenericResourceList'
import { Tag, Table, Descriptions, Button, Space } from 'antd'
import { LinkOutlined } from '@ant-design/icons'
import { history } from '@umijs/max'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import type { ProColumns } from '@ant-design/pro-components'
import type { GenericResourceItem } from '@/features/kops/api/k8s'

/** 解析 CPU 用量字符串为 millicores */
function parseCpu(v: string): number {
  if (!v) return 0
  if (v.endsWith('n')) return parseInt(v) / 1_000_000
  return parseInt(v) || 0
}

/** 解析内存用量字符串为字节数 */
function parseMemory(v: string): number {
  if (!v) return 0
  const num = parseInt(v.replace(/\D/g, '')) || 0
  const unit = v.replace(/[0-9]/g, '')
  if (unit === 'Ki') return num * 1024
  if (unit === 'Mi') return num * 1024 * 1024
  if (unit === 'Gi') return num * 1024 * 1024 * 1024
  return num
}

/** 格式化 CPU 显示 */
function formatCpu(millicores: number): string {
  if (millicores >= 1000) return `${(millicores / 1000).toFixed(2)} Core`
  return `${Math.round(millicores)} m`
}

/** 格式化内存显示 */
function formatMemory(bytes: number): string {
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(2)} GB`
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(1)} MB`
  return `${bytes} B`
}

/** 汇总 Pod 所有容器的 CPU（millicores） */
function sumCpu(record: GenericResourceItem): number {
  const containers = (rawField(record, 'containers') as Array<{ usage?: { cpu?: string } }>) || []
  return containers.reduce((sum, c) => sum + parseCpu(c?.usage?.cpu || ''), 0)
}

/** 汇总 Pod 所有容器的内存（字节） */
function sumMemory(record: GenericResourceItem): number {
  const containers = (rawField(record, 'containers') as Array<{ usage?: { memory?: string } }>) || []
  return containers.reduce((sum, c) => sum + parseMemory(c?.usage?.memory || ''), 0)
}

export default function PodMetricsPage() {
  const clusterId = useClusterId()

  const extraColumns: ProColumns<GenericResourceItem>[] = [
    {
      title: '容器数',
      width: 80,
      align: 'center',
      search: false,
      render: (_, record) => {
        const containers = (rawField(record, 'containers') as Array<unknown>) || []
        return containers.length
      },
    },
    {
      title: 'CPU',
      width: 120,
      search: false,
      sorter: (a, b) => sumCpu(a) - sumCpu(b),
      render: (_, record) => {
        const total = sumCpu(record)
        if (!total) return '-'
        return <Tag color="blue">{formatCpu(total)}</Tag>
      },
    },
    {
      title: '内存',
      width: 120,
      search: false,
      sorter: (a, b) => sumMemory(a) - sumMemory(b),
      render: (_, record) => {
        const total = sumMemory(record)
        if (!total) return '-'
        return <Tag color="purple">{formatMemory(total)}</Tag>
      },
    },
    {
      title: '采集窗口',
      width: 100,
      search: false,
      render: (_, record) => {
        const w = rawField(record, 'window')
        return w ? String(w) : '-'
      },
    },
  ]

  const renderDetail = (record: GenericResourceItem) => {
    const containers =
      (rawField(record, 'containers') as Array<{
        name?: string
        container?: string
        usage?: { cpu?: string; memory?: string }
      }>) || []
    const window = rawField(record, 'window')
    const timestamp = rawField(record, 'timestamp')

    return (
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="Pod 名称">{record.name}</Descriptions.Item>
          <Descriptions.Item label="命名空间">{record.namespace || '-'}</Descriptions.Item>
          {timestamp && (
            <Descriptions.Item label="采集时间">{formatDate(String(timestamp))}</Descriptions.Item>
          )}
          {window && <Descriptions.Item label="采集窗口">{String(window)}</Descriptions.Item>}
        </Descriptions>

        <Button
          type="link"
          icon={<LinkOutlined />}
          onClick={() => history.push(`/k8s/${clusterId}/pods`)}
          style={{ padding: 0 }}
        >
          查看 Pod 列表
        </Button>

        <Table
          size="small"
          rowKey="name"
          pagination={false}
          dataSource={containers.map((c) => ({
            name: c.name || c.container || '',
            cpu: c.usage?.cpu || '-',
            memory: c.usage?.memory || '-',
          }))}
          columns={[
            { title: '容器', dataIndex: 'name', width: 120, render: (t: string) => <Tag>{t}</Tag> },
            {
              title: 'CPU 使用',
              dataIndex: 'cpu',
              render: (v: string) => {
                if (!v || v === '-') return '-'
                return <Tag color="blue">{formatCpu(parseCpu(v))}</Tag>
              },
            },
            {
              title: '内存使用',
              dataIndex: 'memory',
              render: (v: string) => {
                if (!v || v === '-') return '-'
                return <Tag color="purple">{formatMemory(parseMemory(v))}</Tag>
              },
            },
          ]}
        />
      </Space>
    )
  }

  return (
    <GenericResourceList
      resource="podmetrics"
      title="PodMetrics"
      namespaced
      deletable={false}
      creatable={false}
      refetchInterval={15_000}
      extraColumns={extraColumns}
      renderDetail={renderDetail}
    />
  )
}
