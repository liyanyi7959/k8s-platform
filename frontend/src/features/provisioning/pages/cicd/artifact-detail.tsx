/**
 * 制品详情 - 展示制品信息、拉取命令与版本历史
 */
import React, { useEffect, useState } from 'react'
import { history } from '@umijs/max'
import { Button, Card, Descriptions, Modal, Space, Table, Tag, Tooltip, Typography, message } from 'antd'
import {
  ArrowLeftOutlined,
  CopyOutlined,
  DownloadOutlined,
  DeleteOutlined,
} from '@ant-design/icons'
import { AppPage, EmptyState } from '@/components'
import { DESIGN_COLORS } from '@/theme/designTokens'
import { getArtifact, type Artifact } from '@/features/provisioning/api/cicd'

const { Text } = Typography

const TYPE_LABELS: Record<string, string> = {
  image: '容器镜像',
  helm: 'Helm Chart',
  package: '软件包',
}

const STATUS_META: Record<string, { color: string; text: string }> = {
  pushed: { color: 'success', text: '已推送' },
  active: { color: 'success', text: '可用' },
  failed: { color: 'error', text: '失败' },
  deleted: { color: 'default', text: '已删除' },
  pending: { color: 'processing', text: '处理中' },
}

const formatSize = (bytes?: number): string => {
  if (bytes === undefined || bytes === null) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1048576).toFixed(1) + ' MB'
}

const buildPullCommand = (artifact: Artifact): string => {
  const repo = artifact.repository || 'registry.example.com'
  const type = artifact.artifactType || artifact.artifact_type || 'image'
  if (type === 'helm') {
    return 'helm pull oci://' + repo + '/charts/' + artifact.name + ' --version ' + artifact.version
  }
  if (type === 'package') {
    return 'npm install ' + artifact.name + '@' + artifact.version
  }
  return 'docker pull ' + repo + '/' + artifact.name + ':' + artifact.version
}

const ArtifactDetailPage: React.FC = () => {
  const [copied, setCopied] = useState(false)
  const [artifact, setArtifact] = useState<Artifact | null>(null)
  const id = history.location.pathname.split('/').pop() || ''

  useEffect(() => {
    if (!id) return
    getArtifact(id)
      .then(setArtifact)
      .catch((error) => message.error(error instanceof Error ? error.message : '加载制品详情失败'))
  }, [id])

  if (!artifact) {
    return (
      <AppPage keepHeaderTitle title="制品详情">
        <EmptyState description={id ? '加载中...' : '制品不存在'} />
      </AppPage>
    )
  }

  const pullCmd = buildPullCommand(artifact)
  const typeLabel = TYPE_LABELS[artifact.artifactType || artifact.artifact_type || 'image'] || artifact.artifactType || '-'
  const statusMeta = STATUS_META[artifact.status] || { color: 'default', text: artifact.status }
  const versions = (artifact as any).versions as any[] | undefined
  const versionRows = versions && versions.length
    ? versions.map((v: any) => ({
        id: v.id || v.version,
        version: v.version,
        size: v.size || formatSize(v.sizeBytes || v.size_bytes),
        pushedAt: v.pushedAt || v.pushed_at || v.createdAt || v.created_at || '-',
        pushedBy: v.pushedBy || v.pushed_by || '-',
        isCurrent: v.version === artifact.version,
      }))
    : [{
        id: artifact.id,
        version: artifact.version,
        size: formatSize(artifact.sizeBytes || artifact.size_bytes),
        pushedAt: artifact.createdAt || artifact.created_at || '-',
        pushedBy: '-',
        isCurrent: true,
      }]

  const handleCopy = () => {
    navigator.clipboard.writeText(pullCmd).then(() => {
      message.success('拉取命令已复制')
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  const handleDownload = () => {
    message.info('下载功能开发中')
  }

  const handleDelete = () => {
    Modal.confirm({
      title: '删除制品',
      content: '确定删除该制品吗？此操作不可恢复。',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: () => {
        message.info('删除功能开发中')
      },
    })
  }

  return (
    <AppPage keepHeaderTitle title={artifact.name}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 顶部操作栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Tooltip title="返回制品列表"><Button aria-label="返回制品列表" icon={<ArrowLeftOutlined />} onClick={() => history.push('/cicd/artifacts')} /></Tooltip>
            <Tag color="blue">{typeLabel}</Tag>
            <Tag>{artifact.version}</Tag>
          </Space>
          <Space>
            <Tooltip title="下载制品"><Button aria-label="下载制品" icon={<DownloadOutlined />} onClick={handleDownload} /></Tooltip>
            <Tooltip title="删除制品">
              <Button danger icon={<DeleteOutlined />} onClick={handleDelete} />
            </Tooltip>
          </Space>
        </div>

        {/* 基本信息 */}
        <Card title="基本信息">
          <Descriptions column={{ xs: 1, sm: 2, lg: 3 }}>
            <Descriptions.Item label="制品名称">{artifact.name}</Descriptions.Item>
            <Descriptions.Item label="类型">{typeLabel}</Descriptions.Item>
            <Descriptions.Item label="当前版本">
              <Tag color="blue">{artifact.version}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="大小">{formatSize(artifact.sizeBytes || artifact.size_bytes)}</Descriptions.Item>
            <Descriptions.Item label="仓库地址">{artifact.repository || '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusMeta.color}>{statusMeta.text}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">{artifact.createdAt || artifact.created_at || '-'}</Descriptions.Item>
            <Descriptions.Item label="Digest" span={2}>
              <Text code copyable style={{ fontSize: 12 }}>{artifact.digest || '-'}</Text>
            </Descriptions.Item>
          </Descriptions>
        </Card>

        {/* 拉取命令 */}
        <Card title="拉取命令">
          <div
            style={{
              display: 'flex', alignItems: 'center', gap: 12,
              padding: '12px 16px', borderRadius: 8,
              background: DESIGN_COLORS.neutral, border: '1px solid ' + DESIGN_COLORS.grid,
            }}
          >
            <Text code style={{ flex: 1, fontSize: 13, wordBreak: 'break-all' }}>
              {pullCmd}
            </Text>
            <Tooltip title={copied ? '已复制' : '复制命令'}><Button type="primary" aria-label="复制命令" icon={<CopyOutlined />} onClick={handleCopy} /></Tooltip>
          </div>
        </Card>

        {/* 构建来源 */}
        <Card title="构建来源">
          <Descriptions column={1}>
            <Descriptions.Item label="流水线">
              <a onClick={() => history.push('/cicd/pipelines/' + (artifact.pipelineId || artifact.pipeline_id || ''))}>{artifact.pipelineId || artifact.pipeline_id || '-'}</a>
            </Descriptions.Item>
            <Descriptions.Item label="执行记录">
              <a onClick={() => history.push('/cicd/runs/' + (artifact.runId || artifact.run_id || ''))}>#{artifact.runId || artifact.run_id || '-'}</a>
            </Descriptions.Item>
          </Descriptions>
        </Card>

        {/* 版本历史 */}
        <Card title="版本历史">
          <Table
            rowKey="id"
            dataSource={versionRows}
            pagination={false}
            size="small"
            columns={[
              {
                title: '版本', dataIndex: 'version', key: 'version', width: 120,
                render: (version: string, record: any) => (
                  <Space>
                    <Tag color={record.isCurrent ? 'blue' : 'default'}>{version}</Tag>
                    {record.isCurrent && <Text type="secondary" style={{ fontSize: 12 }}>当前</Text>}
                  </Space>
                ),
              },
              { title: '大小', dataIndex: 'size', key: 'size', width: 100 },
              { title: '推送时间', dataIndex: 'pushedAt', key: 'pushedAt', width: 140 },
              { title: '推送者', dataIndex: 'pushedBy', key: 'pushedBy', width: 100 },
              {
                title: '操作', key: 'action', width: 80,
                render: (_: any, record: any) => (
                  !record.isCurrent ? <a>切换到此版本</a> : <Text type="secondary">当前版本</Text>
                ),
              },
            ]}
          />
        </Card>
      </div>
    </AppPage>
  )
}

export default ArtifactDetailPage
