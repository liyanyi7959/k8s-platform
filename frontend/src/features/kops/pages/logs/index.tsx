import React, { useEffect, useMemo, useState } from 'react'
import { history } from '@umijs/max'
import {
  ClusterOutlined,
  FileSearchOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import {
  Button,
  Card,
  Input,
  Select,
  Space,
  Table,
  Typography,
  message,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { AppPage, ContextNotice, WorkspaceHeader } from '@/components'
import { listClusters } from '@/features/fleet'
import { getPodLogs, listPods } from '@/features/kops/api/k8s'

const { Text } = Typography

type LogRow = {
  key: string
  clusterId: number
  cluster: string
  namespace: string
  pod: string
  time?: string
  content: string
}

const MAX_CLUSTERS = 5
const MAX_PODS = 30

function splitLogLines(raw: string): Array<{ time?: string; content: string }> {
  return raw
    .split(/\r?\n/)
    .filter(Boolean)
    .map((line) => {
      const match = line.match(/^(\d{4}-\d{2}-\d{2}T\S+?)\s+(.*)$/)
      return match ? { time: match[1], content: match[2] } : { content: line }
    })
}

const LogsPage: React.FC = () => {
  const [clusterScope, setClusterScope] = useState<string>('all')
  const [namespace, setNamespace] = useState('')
  const [keyword, setKeyword] = useState('')
  const [tailLines, setTailLines] = useState(100)
  const [rows, setRows] = useState<LogRow[]>([])
  const [searched, setSearched] = useState(false)
  const [searching, setSearching] = useState(false)
  const [searchNote, setSearchNote] = useState('')

  const { data: clusterResult, isLoading: clustersLoading } = useQuery({
    queryKey: ['global-log-clusters'],
    queryFn: ({ signal }) => listClusters({ page: 1, pageSize: 100 }, signal),
  })
  const clusters = clusterResult?.items || []

  useEffect(() => {
    if (clusterScope === 'all' && clusters.length === 1) setClusterScope(String(clusters[0].id))
  }, [clusterScope, clusters])

  const clusterOptions = useMemo(
    () => [
      { value: 'all', label: `全部可访问集群（最多 ${MAX_CLUSTERS} 个）` },
      ...clusters.map((cluster) => ({ value: String(cluster.id), label: cluster.name })),
    ],
    [clusters],
  )

  const columns: ColumnsType<LogRow> = [
    { title: '集群', dataIndex: 'cluster', width: 160, ellipsis: true },
    { title: '命名空间', dataIndex: 'namespace', width: 140, ellipsis: true },
    {
      title: 'Pod',
      dataIndex: 'pod',
      width: 230,
      ellipsis: true,
      render: (pod: string, row) => (
        <Button
          type="link"
          size="small"
          style={{ padding: 0 }}
          onClick={() => history.push(`/k8s/${row.clusterId}/pods?namespace=${encodeURIComponent(row.namespace)}`)}
        >
          {pod}
        </Button>
      ),
    },
    { title: '时间', dataIndex: 'time', width: 210, render: (value?: string) => value || '-' },
    {
      title: '日志内容',
      dataIndex: 'content',
      render: (value: string) => (
        <Text code style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', color: '#334155' }}>
          {value}
        </Text>
      ),
    },
  ]

  const search = async () => {
    const selectedClusters =
      clusterScope === 'all' ? clusters.slice(0, MAX_CLUSTERS) : clusters.filter((item) => String(item.id) === clusterScope)
    if (!selectedClusters.length) {
      message.warning('请选择至少一个可访问集群')
      return
    }

    setSearching(true)
    setSearched(true)
    setRows([])
    try {
      const podGroups = await Promise.all(
        selectedClusters.map(async (cluster) => {
          const result = await listPods(cluster.id, { namespace: namespace.trim() || undefined })
          const ranked = [...result.items].sort((a, b) => {
            const score = (pod: (typeof result.items)[number]) =>
              (pod.status === 'Failed' ? 4 : 0) + (pod.containerReason ? 2 : 0) + Math.min(pod.restarts || 0, 1)
            return score(b) - score(a)
          })
          return { cluster, pods: ranked.slice(0, MAX_PODS) }
        }),
      )
      const podCount = podGroups.reduce((total, group) => total + group.pods.length, 0)
      const results = await Promise.all(
        podGroups.flatMap(({ cluster, pods }) =>
          pods.map(async (pod) => {
            try {
              const response = await getPodLogs(cluster.id, pod.namespace, pod.name, { tailLines, timestamps: true })
              return splitLogLines(response.logs).map((line, index) => ({
                key: `${cluster.id}-${pod.namespace}-${pod.name}-${index}-${line.content}`,
                clusterId: cluster.id,
                cluster: cluster.name,
                namespace: pod.namespace,
                pod: pod.name,
                ...line,
              }))
            } catch {
              return [] as LogRow[]
            }
          }),
        ),
      )
      const normalizedKeyword = keyword.trim().toLowerCase()
      const nextRows = results
        .flat()
        .filter((row) => !normalizedKeyword || row.content.toLowerCase().includes(normalizedKeyword))
        .slice(-5000)
      setRows(nextRows)
      setSearchNote(`已从 ${selectedClusters.length} 个集群、${podCount} 个 Pod 获取实时日志；展示最近 ${nextRows.length} 条匹配记录。`)
    } catch (error: any) {
      setSearchNote(error?.message || '日志检索失败，请检查集群连接和访问权限。')
      message.error('日志检索失败')
    } finally {
      setSearching(false)
    }
  }

  return (
    <AppPage>
      <div className="app-page-shell">
        <WorkspaceHeader
          title="全局日志分析"
          description="按范围读取已接入集群的 Pod 实时日志。"
          icon={<FileSearchOutlined />}
        />

        <Card className="app-aiops-panel" title="检索条件">
          <Space wrap size={[12, 12]}>
            <Select
              loading={clustersLoading}
              value={clusterScope}
              options={clusterOptions}
              onChange={setClusterScope}
              style={{ width: 260 }}
            />
            <Input
              allowClear
              value={namespace}
              onChange={(event) => setNamespace(event.target.value)}
              placeholder="命名空间（留空为全部）"
              style={{ width: 210 }}
            />
            <Input
              allowClear
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              onPressEnter={search}
              placeholder="日志关键字，如 error / timeout"
              style={{ width: 280 }}
            />
            <Select value={tailLines} onChange={setTailLines} style={{ width: 130 }} options={[
              { value: 50, label: '最近 50 行' },
              { value: 100, label: '最近 100 行' },
              { value: 200, label: '最近 200 行' },
            ]} />
            <Button type="primary" icon={<FileSearchOutlined />} loading={searching} onClick={search}>
              检索实时日志
            </Button>
            <Button icon={<ReloadOutlined />} disabled={searching} onClick={search}>刷新</Button>
          </Space>
          <ContextNotice
            style={{ marginTop: 16 }}
            type="info"
            showIcon
            icon={<SafetyCertificateOutlined />}
            message={`安全边界：单次最多查询 ${MAX_CLUSTERS} 个集群、每集群 ${MAX_PODS} 个 Pod，结果最多保留 5,000 行。`}
          />
        </Card>

        <Card
          className="app-aiops-panel"
          style={{ marginTop: 16 }}
          title="检索结果"
          extra={searched && <Text type="secondary">{searchNote}</Text>}
        >
          <Table<LogRow>
            rowKey="key"
            size="small"
            loading={searching}
            columns={columns}
            dataSource={rows}
            scroll={{ x: 980, y: 560 }}
            pagination={{ pageSize: 100, showSizeChanger: true, showTotal: (total) => `共 ${total} 条` }}
            locale={{ emptyText: searched ? '没有匹配的实时日志' : '设置检索范围后开始查询' }}
          />
          {searched && rows.length > 0 && (
            <Space style={{ marginTop: 12 }}>
              <Text type="secondary">发现异常后可进入集群工作台查看关联 Events、资源指标和 AI 诊断。</Text>
              <Button icon={<ClusterOutlined />} size="small" onClick={() => history.push('/clusters')}>进入集群管理</Button>
            </Space>
          )}
        </Card>
      </div>
    </AppPage>
  )
}

export default LogsPage
