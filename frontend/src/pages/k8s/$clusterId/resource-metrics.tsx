import React, { useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Tabs, Progress, Select, Button, Typography } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { listNodeMetrics, listPodMetricsUsage } from '@/services/k8s'
import { AppPage } from '@/components'
import AppAlert from '@/components/AppAlert'
import { useClusterId } from '@/hooks/useClusterId'

const { Text } = Typography

/** 节点指标数据结构 */
interface NodeMetric {
  name: string
  cpuUsage: number
  memoryUsage: number
  cpuCapacity: number
  memoryCapacity: number
  cpuUsed: number
  memoryUsed: number
}

/** Pod 指标数据结构 */
interface PodMetric {
  name: string
  namespace: string
  cpu: number
  memory: number
}

/** 格式化 CPU（后端返回 nanocores，1 核 = 1e9 nanocores） */
function formatCPU(nanocores: number): string {
  if (!nanocores) return '0 m'
  const cores = nanocores / 1e9
  if (cores < 1) return `${Math.round(nanocores / 1e6)} m`
  return `${cores.toFixed(2)} 核`
}

/** 格式化内存（后端返回 bytes） */
function formatMemory(bytes: number): string {
  if (!bytes) return '0 Mi'
  const gi = bytes / (1024 * 1024 * 1024)
  if (gi >= 1) return `${gi.toFixed(2)} Gi`
  return `${Math.round(bytes / (1024 * 1024))} Mi`
}

/** 使用率进度条颜色：>80% 红色，>60% 橙色，其他绿色 */
function progressColor(percent: number): string {
  if (percent >= 80) return '#dc2626'
  if (percent >= 60) return '#b45309'
  return '#047857'
}

const REFRESH_OPTIONS = [
  { label: '10 秒', value: 10_000 },
  { label: '30 秒', value: 30_000 },
  { label: '60 秒', value: 60_000 },
]

const ResourceMetricsPage: React.FC = () => {
  const clusterId = useClusterId()
  const [activeTab, setActiveTab] = useState('nodes')
  const [refreshInterval, setRefreshInterval] = useState(30_000)

  // 节点资源使用率查询（自动刷新）
  const nodeQuery = useQuery({
    queryKey: ['node-metrics', clusterId],
    queryFn: ({ signal }) => listNodeMetrics(clusterId, signal),
    refetchInterval: refreshInterval,
  })

  // Pod 资源使用率查询（自动刷新）
  const podQuery = useQuery({
    queryKey: ['pod-metrics', clusterId],
    queryFn: ({ signal }) => listPodMetricsUsage(clusterId, signal),
    refetchInterval: refreshInterval,
  })

  const nodeColumns: ProColumns<NodeMetric>[] = [
    {
      title: '节点名称',
      dataIndex: 'name',
      width: 160,
      ellipsis: true,
    },
    {
      title: 'CPU 使用率',
      dataIndex: 'cpuUsage',
      width: 200,
      render: (_, r) => (
        <Progress
          percent={Number((r.cpuUsage || 0).toFixed(1))}
          size="small"
          strokeColor={progressColor(r.cpuUsage || 0)}
        />
      ),
    },
    {
      title: 'CPU 用量 / 容量',
      width: 160,
      render: (_, r) => (
        <Text>
          {formatCPU(r.cpuUsed)} / {formatCPU(r.cpuCapacity)}
        </Text>
      ),
    },
    {
      title: '内存使用率',
      dataIndex: 'memoryUsage',
      width: 200,
      render: (_, r) => (
        <Progress
          percent={Number((r.memoryUsage || 0).toFixed(1))}
          size="small"
          strokeColor={progressColor(r.memoryUsage || 0)}
        />
      ),
    },
    {
      title: '内存用量 / 容量',
      width: 160,
      render: (_, r) => (
        <Text>
          {formatMemory(r.memoryUsed)} / {formatMemory(r.memoryCapacity)}
        </Text>
      ),
    },
  ]

  const podColumns: ProColumns<PodMetric>[] = [
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 140,
      ellipsis: true,
    },
    {
      title: 'Pod 名称',
      dataIndex: 'name',
      width: 220,
      ellipsis: true,
    },
    {
      title: 'CPU 用量',
      dataIndex: 'cpu',
      width: 120,
      render: (_, r) => <Text>{formatCPU(r.cpu)}</Text>,
    },
    {
      title: '内存用量',
      dataIndex: 'memory',
      width: 120,
      render: (_, r) => <Text>{formatMemory(r.memory)}</Text>,
    },
  ]

  /** 渲染工具栏：刷新间隔选择 + 手动刷新按钮 */
  const renderToolbar = (refetch: () => void) => [
    <Select
      key="interval"
      value={refreshInterval}
      onChange={setRefreshInterval}
      options={REFRESH_OPTIONS}
      style={{ width: 100 }}
    />,
    <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>
      刷新
    </Button>,
  ]

  /** 渲染错误提示（如 metrics-server 未安装） */
  const renderError = (error: unknown) => {
    if (!error) return null
    const msg = error instanceof Error ? error.message : '获取指标数据失败'
    return (
      <AppAlert
        type="warning"
        showIcon
        message="指标数据获取失败"
        description={msg}
        style={{ marginBottom: 16 }}
      />
    )
  }

  return (
    <AppPage>
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'nodes',
            label: '节点使用率',
            children: (
              <>
                {renderError(nodeQuery.error)}
                <ProTable<NodeMetric>
                  columns={nodeColumns}
                  dataSource={nodeQuery.data || []}
                  loading={nodeQuery.isLoading}
                  rowKey="name"
                  search={false}
                  options={{ reload: false }}
                  pagination={{
                    defaultPageSize: 20,
                    showSizeChanger: true,
                    showTotal: (t) => `共 ${t} 条`,
                  }}
                  scroll={{ x: 900 }}
                  headerTitle={<Text strong>节点资源使用率</Text>}
                  toolBarRender={() => renderToolbar(nodeQuery.refetch)}
                />
              </>
            ),
          },
          {
            key: 'pods',
            label: 'Pod 使用率',
            children: (
              <>
                {renderError(podQuery.error)}
                <ProTable<PodMetric>
                  columns={podColumns}
                  dataSource={podQuery.data || []}
                  loading={podQuery.isLoading}
                  rowKey={(r) => `${r.namespace}/${r.name}`}
                  search={false}
                  options={{ reload: false }}
                  pagination={{
                    defaultPageSize: 20,
                    showSizeChanger: true,
                    showTotal: (t) => `共 ${t} 条`,
                  }}
                  scroll={{ x: 600 }}
                  headerTitle={<Text strong>Pod 资源使用率</Text>}
                  toolBarRender={() => renderToolbar(podQuery.refetch)}
                />
              </>
            ),
          },
        ]}
      />
    </AppPage>
  )
}

export default ResourceMetricsPage
