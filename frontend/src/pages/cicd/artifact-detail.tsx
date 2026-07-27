/**
 * 制品详情 - 展示制品信息、拉取命令与版本历史
 */
import React, { useState } from 'react'
import { useParams, history } from '@umijs/max'
import { Button, Card, Descriptions, Space, Table, Tag, Tooltip, Typography, message } from 'antd'
import {
  ArrowLeftOutlined,
  CopyOutlined,
  DownloadOutlined,
  DeleteOutlined,
} from '@ant-design/icons'
import { AppPage } from '@/components'
import { DESIGN_COLORS } from '@/theme/designTokens'

const { Text, Paragraph } = Typography

const ARTIFACT = {
  id: '1',
  name: 'frontend',
  type: 'image',
  typeLabel: '容器镜像',
  version: 'v1.2.3',
  size: '45.2 MB',
  digest: 'sha256:abc123def456ghi789jkl012mno345pqr678stu901vwx234yz',
  pushedAt: '2026-07-26 14:35',
  pushedBy: 'admin',
  pipeline: 'frontend-ci',
  runId: 'r128',
}

const VERSIONS = [
  { id: 'v1', version: 'v1.2.3', size: '45.2 MB', pushedAt: '07-26 14:35', pushedBy: 'admin', isCurrent: true },
  { id: 'v2', version: 'v1.2.2', size: '44.8 MB', pushedAt: '07-25 10:05', pushedBy: 'ci-bot', isCurrent: false },
  { id: 'v3', version: 'v1.2.1', size: '44.5 MB', pushedAt: '07-24 16:22', pushedBy: 'ci-bot', isCurrent: false },
  { id: 'v4', version: 'v1.2.0', size: '43.9 MB', pushedAt: '07-23 09:30', pushedBy: 'admin', isCurrent: false },
  { id: 'v5', version: 'v1.1.9', size: '43.2 MB', pushedAt: '07-22 14:10', pushedBy: 'ci-bot', isCurrent: false },
]

const PULL_COMMANDS: Record<string, string> = {
  image: 'docker pull registry.example.com/frontend:v1.2.3',
  helm: 'helm pull oci://registry.example.com/charts/frontend --version 1.2.3',
  package: 'npm install frontend@1.2.3',
}

const ArtifactDetailPage: React.FC = () => {
  const params = useParams()
  const [copied, setCopied] = useState(false)
  const pullCmd = PULL_COMMANDS[ARTIFACT.type] || PULL_COMMANDS.image

  const handleCopy = () => {
    navigator.clipboard.writeText(pullCmd).then(() => {
      message.success('拉取命令已复制')
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <AppPage keepHeaderTitle title={ARTIFACT.name}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 顶部操作栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => history.push('/cicd/artifacts')}>返回</Button>
            <Tag color="blue">{ARTIFACT.typeLabel}</Tag>
            <Tag>{ARTIFACT.version}</Tag>
          </Space>
          <Space>
            <Button icon={<DownloadOutlined />}>下载</Button>
            <Tooltip title="删除制品">
              <Button danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Space>
        </div>

        {/* 基本信息 */}
        <Card title="基本信息">
          <Descriptions column={{ xs: 1, sm: 2, lg: 3 }}>
            <Descriptions.Item label="制品名称">{ARTIFACT.name}</Descriptions.Item>
            <Descriptions.Item label="类型">{ARTIFACT.typeLabel}</Descriptions.Item>
            <Descriptions.Item label="当前版本">
              <Tag color="blue">{ARTIFACT.version}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="大小">{ARTIFACT.size}</Descriptions.Item>
            <Descriptions.Item label="推送时间">{ARTIFACT.pushedAt}</Descriptions.Item>
            <Descriptions.Item label="推送者">{ARTIFACT.pushedBy}</Descriptions.Item>
            <Descriptions.Item label="Digest" span={3}>
              <Text code copyable style={{ fontSize: 12 }}>{ARTIFACT.digest}</Text>
            </Descriptions.Item>
          </Descriptions>
        </Card>

        {/* 拉取命令 */}
        <Card title="拉取命令">
          <div
            style={{
              display: 'flex', alignItems: 'center', gap: 12,
              padding: '12px 16px', borderRadius: 8,
              background: DESIGN_COLORS.neutral, border: `1px solid ${DESIGN_COLORS.grid}`,
            }}
          >
            <Text code style={{ flex: 1, fontSize: 13, wordBreak: 'break-all' }}>
              {pullCmd}
            </Text>
            <Button type="primary" icon={<CopyOutlined />} onClick={handleCopy}>
              {copied ? '已复制' : '复制'}
            </Button>
          </div>
        </Card>

        {/* 构建来源 */}
        <Card title="构建来源">
          <Descriptions column={1}>
            <Descriptions.Item label="流水线">
              <a onClick={() => history.push('/cicd/pipelines/1')}>{ARTIFACT.pipeline}</a>
            </Descriptions.Item>
            <Descriptions.Item label="执行记录">
              <a onClick={() => history.push(`/cicd/runs/${ARTIFACT.runId}`)}>#{ARTIFACT.runId}</a>
            </Descriptions.Item>
          </Descriptions>
        </Card>

        {/* 版本历史 */}
        <Card title="版本历史">
          <Table
            rowKey="id"
            dataSource={VERSIONS}
            pagination={false}
            size="small"
            columns={[
              {
                title: '版本', dataIndex: 'version', key: 'version', width: 120,
                render: (version: string, record) => (
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
                render: (_, record) => (
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
