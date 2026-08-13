/**
 * 环境详情 - 展示环境信息、部署历史与环境变量
 */
import React from 'react'
import { history } from '@umijs/max'
import { Button, Card, Descriptions, Space, Table, Tag, Tooltip, Typography } from 'antd'
import {
  ArrowLeftOutlined,
  RocketOutlined,
  RollbackOutlined,
  GlobalOutlined,
} from '@ant-design/icons'
import { AppPage } from '@/components'
import { DESIGN_COLORS } from '@/theme/designTokens'

const { Text } = Typography

const ENV = {
  id: '1',
  name: 'production',
  label: '生产环境',
  type: 'production',
  status: 'deployed',
  cluster: 'prod-cluster-01',
  namespace: 'default',
  version: 'v1.2.3',
  lastDeploy: '2026-07-26 14:32',
  deployedBy: 'admin',
}

const DEPLOY_HISTORY = [
  { id: 'd5', version: 'v1.2.3', status: 'success', deployedBy: 'admin', deployedAt: '07-26 14:32', duration: '2m 15s' },
  { id: 'd4', version: 'v1.2.2', status: 'success', deployedBy: 'ci-bot', deployedAt: '07-25 10:05', duration: '2m 08s' },
  { id: 'd3', version: 'v1.2.1', status: 'failed', deployedBy: 'ci-bot', deployedAt: '07-24 16:22', duration: '0m 45s' },
  { id: 'd2', version: 'v1.2.0', status: 'success', deployedBy: 'admin', deployedAt: '07-23 09:30', duration: '2m 20s' },
  { id: 'd1', version: 'v1.1.9', status: 'success', deployedBy: 'ci-bot', deployedAt: '07-22 14:10', duration: '1m 58s' },
]

const ENV_VARS = [
  { key: 'DATABASE_URL', value: 'jdbc:postgresql://prod-db:5432/app', masked: false },
  { key: 'REDIS_URL', value: 'redis://prod-redis:6379', masked: false },
  { key: 'JWT_SECRET', value: '********', masked: true },
  { key: 'LOG_LEVEL', value: 'info', masked: false },
  { key: 'API_TIMEOUT', value: '30000', masked: false },
]

const STATUS_META: Record<string, { color: string; text: string }> = {
  success: { color: 'success', text: '成功' },
  failed: { color: 'error', text: '失败' },
}

const ENV_TYPE_COLOR: Record<string, string> = {
  production: DESIGN_COLORS.danger,
  staging: DESIGN_COLORS.warning,
  development: DESIGN_COLORS.success,
}

const EnvironmentDetailPage: React.FC = () => {
  const envColor = ENV_TYPE_COLOR[ENV.type] || DESIGN_COLORS.primary

  return (
    <AppPage keepHeaderTitle title={ENV.name}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 顶部操作栏 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Tooltip title="返回环境列表"><Button aria-label="返回环境列表" icon={<ArrowLeftOutlined />} onClick={() => history.push('/cicd/environments')} /></Tooltip>
            <Tag style={{ color: envColor, borderColor: `${envColor}40`, background: `${envColor}0d` }}>{ENV.label}</Tag>
            <Tag color={ENV.status === 'deployed' ? 'success' : 'error'}>
              {ENV.status === 'deployed' ? '已部署' : '部署失败'}
            </Tag>
          </Space>
          <Space>
            <Tooltip title="部署"><Button type="primary" aria-label="部署" icon={<RocketOutlined />} /></Tooltip>
            <Tooltip title="回滚"><Button aria-label="回滚" icon={<RollbackOutlined />} /></Tooltip>
          </Space>
        </div>

        {/* 基本信息 */}
        <Card title="基本信息">
          <div style={{ borderLeft: `4px solid ${envColor}`, paddingLeft: 16, marginBottom: 16 }}>
            <Space>
              <GlobalOutlined style={{ color: envColor, fontSize: 18 }} />
              <Text strong style={{ fontSize: 16 }}>{ENV.name}</Text>
              <Text type="secondary">{ENV.label}</Text>
            </Space>
          </div>
          <Descriptions column={{ xs: 1, sm: 2, lg: 3 }}>
            <Descriptions.Item label="环境名称">{ENV.name}</Descriptions.Item>
            <Descriptions.Item label="环境类型">{ENV.label}</Descriptions.Item>
            <Descriptions.Item label="关联集群">{ENV.cluster}</Descriptions.Item>
            <Descriptions.Item label="命名空间">{ENV.namespace}</Descriptions.Item>
            <Descriptions.Item label="当前版本">
              <Tag color="blue">{ENV.version}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={ENV.status === 'deployed' ? 'success' : 'error'}>
                {ENV.status === 'deployed' ? '已部署' : '部署失败'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="最近部署">{ENV.lastDeploy}</Descriptions.Item>
            <Descriptions.Item label="部署者">{ENV.deployedBy}</Descriptions.Item>
          </Descriptions>
        </Card>

        {/* 部署历史 */}
        <Card title="部署历史">
          <Table
            rowKey="id"
            dataSource={DEPLOY_HISTORY}
            pagination={false}
            size="small"
            columns={[
              {
                title: '版本', dataIndex: 'version', key: 'version', width: 120,
                render: (version: string) => <Tag color="blue">{version}</Tag>,
              },
              {
                title: '状态', dataIndex: 'status', key: 'status', width: 80,
                render: (status: string) => <Tag color={STATUS_META[status]?.color}>{STATUS_META[status]?.text}</Tag>,
              },
              { title: '部署者', dataIndex: 'deployedBy', key: 'deployedBy', width: 100 },
              { title: '耗时', dataIndex: 'duration', key: 'duration', width: 100 },
              { title: '部署时间', dataIndex: 'deployedAt', key: 'deployedAt' },
              {
                title: '操作', key: 'action', width: 80,
                render: (_, record) => record.status === 'success' ? <a>回滚到此版本</a> : null,
              },
            ]}
          />
        </Card>

        {/* 环境变量 */}
        <Card title="环境变量">
          <Table
            rowKey="key"
            dataSource={ENV_VARS}
            pagination={false}
            size="small"
            columns={[
              { title: '变量名', dataIndex: 'key', key: 'key', width: 200, render: (key: string) => <Text code>{key}</Text> },
              {
                title: '变量值', dataIndex: 'value', key: 'value',
                render: (value: string, record) => (
                  record.masked
                    ? <Text type="secondary">{value}</Text>
                    : <Text code>{value}</Text>
                ),
              },
            ]}
          />
        </Card>
      </div>
    </AppPage>
  )
}

export default EnvironmentDetailPage
