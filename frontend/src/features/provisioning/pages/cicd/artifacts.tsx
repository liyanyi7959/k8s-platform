/**
 * 制品仓库 - 构建产物管理（容器镜像、Helm Chart 等）
 */
import { useEffect, useState } from 'react'
import { history } from '@umijs/max'
import { Button, Card, Input, Select, Space, Table, Tag, Tooltip } from 'antd'
import { EyeOutlined } from '@ant-design/icons'
import { AppPage, EmptyState } from '@/components'
import type { ProColumns } from '@ant-design/pro-components'
import { getArtifacts, type Artifact } from '@/features/provisioning/api/cicd'

interface ArtifactRecord {
  id: string
  name: string
  type: 'image' | 'helm' | 'package'
  version: string
  size: string
  pushedAt: string
}

const TYPE_MAP: Record<string, { color: string; text: string }> = {
  image: { color: 'blue', text: '容器镜像' },
  helm: { color: 'green', text: 'Helm Chart' },
  package: { color: 'purple', text: '通用包' },
}

const columns: ProColumns<ArtifactRecord>[] = [
  {
    title: '制品名称',
    dataIndex: 'name',
    key: 'name',
    render: (_, record) => <strong>{record.name}</strong>,
  },
  {
    title: '类型',
    dataIndex: 'type',
    key: 'type',
    width: 120,
    render: (_, record) => {
      const cfg = TYPE_MAP[record.type] ?? TYPE_MAP.package!
      return <Tag color={cfg.color}>{cfg.text}</Tag>
    },
  },
  {
    title: '版本',
    dataIndex: 'version',
    key: 'version',
    width: 120,
    render: (_, record) => <Tag>{record.version}</Tag>,
  },
  {
    title: '大小',
    dataIndex: 'size',
    key: 'size',
    width: 100,
  },
  {
    title: '推送时间',
    dataIndex: 'pushedAt',
    key: 'pushedAt',
    width: 180,
    ellipsis: true,
  },
  {
    title: '操作',
    key: 'action',
    width: 100,
    render: (_, record) => (
      <Tooltip title="查看详情"><Button type="text" aria-label="查看详情" icon={<EyeOutlined />} onClick={(event) => { event.stopPropagation(); history.push(`/cicd/artifacts/${record.id}`) }} /></Tooltip>
    ),
  },
]

/** 格式化文件大小：< 1024 显示 "x B"，< 1048576 显示 "x.x KB"，否则 "x.x MB" */
const formatSize = (bytes?: number): string => {
  if (!bytes) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1048576).toFixed(1) + ' MB'
}

const ArtifactsPage: React.FC = () => {
  const [artifacts, setArtifacts] = useState<ArtifactRecord[]>([])
  const [keyword, setKeyword] = useState('')
  const [typeFilter, setTypeFilter] = useState<string | undefined>(undefined)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)

  const load = () => {
    getArtifacts({ page, pageSize, keyword, artifactType: typeFilter }).then((res) => {
      setArtifacts((res.list || []).map((a: Artifact) => ({
        id: String(a.id),
        name: a.name,
        type: (a.artifactType || a.artifact_type || 'package') as ArtifactRecord['type'],
        version: a.version,
        size: formatSize(a.sizeBytes || a.size_bytes),
        pushedAt: a.createdAt || a.created_at || '-',
      })))
      setTotal(res.total || 0)
    })
  }

  useEffect(() => { load() }, [page, pageSize, keyword, typeFilter])
  return (
    <AppPage>
      <Card>
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Input.Search
              placeholder="搜索制品名称"
              style={{ width: 240 }}
              allowClear
              onSearch={(value) => { setKeyword(value); setPage(1) }}
            />
            <Select
              placeholder="制品类型"
              style={{ width: 140 }}
              allowClear
              options={[
                { value: 'image', label: '容器镜像' },
                { value: 'helm', label: 'Helm Chart' },
                { value: 'package', label: '通用包' },
              ]}
              onChange={(value) => { setTypeFilter(value); setPage(1) }}
            />
          </Space>
        </div>
        <Table<ArtifactRecord>
          rowKey="id"
          columns={columns as any}
          dataSource={artifacts}
          pagination={{
            current: page,
            pageSize: pageSize,
            total: total,
            showSizeChanger: true,
            showTotal: (t) => '共 ' + t + ' 条',
            onChange: (p, ps) => { setPage(p); setPageSize(ps) },
          }}
          onRow={(record) => ({ onClick: () => history.push(`/cicd/artifacts/${record.id}`), style: { cursor: 'pointer' } })}
          locale={{
            emptyText: <EmptyState description="暂无制品，构建流水线执行后将自动归集产物" />,
          }}
        />
      </Card>
    </AppPage>
  )
}

export default ArtifactsPage
