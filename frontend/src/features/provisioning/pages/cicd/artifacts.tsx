/**
 * 制品仓库 - 构建产物管理（容器镜像、Helm Chart 等）
 */
import { history } from '@umijs/max'
import { Card, Input, Select, Space, Table, Tag } from 'antd'
import { EyeOutlined } from '@ant-design/icons'
import { AppPage, EmptyState } from '@/components'
import type { ProColumns } from '@ant-design/pro-components'

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
    render: () => (
      <Space>
        <a><EyeOutlined /> 详情</a>
      </Space>
    ),
  },
]

const MOCK_ARTIFACTS: ArtifactRecord[] = [
  { id: '1', name: 'frontend', type: 'image', version: 'v1.2.3', size: '45.2 MB', pushedAt: '07-26 14:35' },
  { id: '2', name: 'backend', type: 'image', version: 'v2.0.1', size: '78.5 MB', pushedAt: '07-26 12:05' },
  { id: '3', name: 'api-gateway', type: 'image', version: 'v1.5.0', size: '52.1 MB', pushedAt: '07-26 10:20' },
  { id: '4', name: 'app-chart', type: 'helm', version: '0.8.2', size: '12.3 KB', pushedAt: '07-25 18:25' },
  { id: '5', name: 'nginx-config', type: 'helm', version: '1.1.0', size: '8.5 KB', pushedAt: '07-25 10:08' },
  { id: '6', name: 'shared-ui', type: 'package', version: '3.2.1', size: '2.1 MB', pushedAt: '07-24 16:30' },
  { id: '7', name: 'utils-lib', type: 'package', version: '1.0.5', size: '340 KB', pushedAt: '07-24 14:15' },
]

const ArtifactsPage: React.FC = () => {
  return (
    <AppPage>
      <Card>
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Input.Search placeholder="搜索制品名称" style={{ width: 240 }} allowClear />
            <Select
              placeholder="制品类型"
              style={{ width: 140 }}
              allowClear
              options={[
                { value: 'image', label: '容器镜像' },
                { value: 'helm', label: 'Helm Chart' },
                { value: 'package', label: '通用包' },
              ]}
            />
          </Space>
        </div>
        <Table<ArtifactRecord>
          rowKey="id"
          columns={columns as any}
          dataSource={MOCK_ARTIFACTS}
          pagination={false}
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
