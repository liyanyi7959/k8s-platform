/**
 * 部署计划列表页
 * 展示 K8s 集群部署方案，支持增删改查、预览、执行、取消、重试
 */
import { useState } from 'react'
import { Card, Table, Button, Space, Tag, Popconfirm, message, Typography, Tooltip, Empty, Modal, Descriptions, Badge, Collapse } from 'antd'
import {
  PlusOutlined, DeleteOutlined, ReloadOutlined,
  EyeOutlined, PlayCircleOutlined, StopOutlined, RedoOutlined, DesktopOutlined, EditOutlined, ProfileOutlined
} from '@ant-design/icons'
import { history } from '@umijs/max'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { ColumnsType } from 'antd/es/table'
import { AppPage, ListWorkspace } from '@/components'
import { getDeployPlans, deleteDeployPlan, dryRunDeployPlan, cancelDeployPlan, retryDeployPlan } from '@/services/deploy'
import type { DeployPlan, DeployDryRunNodeFlow, DeployDryRunResult, DeployDryRunStep } from '@/types/deploy'

const { Text } = Typography
const provisionPath = '/clusters/provision'

/** 状态颜色映射 */
const statusMap: Record<string, { color: string; text: string; badge: string }> = {
  draft: { color: 'default', text: '草稿', badge: 'default' },
  running: { color: 'processing', text: '执行中', badge: 'processing' },
  success: { color: 'success', text: '成功', badge: 'success' },
  failed: { color: 'error', text: '失败', badge: 'error' },
  canceled: { color: 'warning', text: '已取消', badge: 'warning' },
  cancelled: { color: 'warning', text: '已取消', badge: 'warning' },
}

/** 角色颜色映射 */
const roleColorMap: Record<string, string> = {
  master: 'red',
  worker: 'blue',
  etcd: 'orange',
}

/** 阶段中文映射 */
const phaseMap: Record<string, string> = {
  preflight: '预检',
  install: '安装',
  init: '初始化',
  join: '加入集群',
  addon: '插件部署',
  finalize: '收尾验证',
}

export default function DeployPlansPage() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [previewId, setPreviewId] = useState<number | null>(null)
  const [previewData, setPreviewData] = useState<DeployDryRunResult | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['deploy-plans', page, pageSize],
    queryFn: () => getDeployPlans({ page, pageSize }),
    refetchInterval: (query) => {
      // 有运行中的计划时自动刷新
      const items = query.state.data?.items || []
      return items.some((p) => p.status === 'running') ? 5000 : false
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteDeployPlan(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['deploy-plans'] })
    },
    onError: () => message.error('删除失败'),
  })

  const cancelMutation = useMutation({
    mutationFn: (id: number) => cancelDeployPlan(id),
    onSuccess: () => {
      message.success('取消成功')
      queryClient.invalidateQueries({ queryKey: ['deploy-plans'] })
    },
    onError: () => message.error('取消失败'),
  })

  const retryMutation = useMutation({
    mutationFn: (id: number) => retryDeployPlan(id),
    onSuccess: () => {
      message.success('重试已启动')
      queryClient.invalidateQueries({ queryKey: ['deploy-plans'] })
    },
    onError: () => message.error('重试失败'),
  })

  // 执行部署：直接跳转详情页，由详情页自动触发执行
  const handleExecute = (id: number) => {
    history.push(`${provisionPath}/${id}?execute=1`)
  }

  // 重试部署并跳转详情页
  const handleRetry = async (id: number) => {
    try {
      await retryMutation.mutateAsync(id)
      history.push(`${provisionPath}/${id}`)
    } catch {
      // onError 已提示
    }
  }

  /** 预览部署计划 - 使用 dry-run 接口 */
  const handlePreview = async (id: number) => {
    setPreviewId(id)
    setPreviewLoading(true)
    try {
      const data = await dryRunDeployPlan(id)
      setPreviewData(data)
    } catch {
      message.error('获取预览失败')
    } finally {
      setPreviewLoading(false)
    }
  }

  const columns: ColumnsType<DeployPlan> = [
    {
      title: '方案名称',
      dataIndex: 'name',
      width: 180,
      render: (name: string) => <Text strong>{name}</Text>,
    },
    {
      title: '集群名称',
      dataIndex: 'clusterName',
      width: 150,
      render: (v: string) => v || '-',
    },
    {
      title: 'K8s 版本',
      dataIndex: 'k8sVersion',
      width: 120,
      align: 'center',
      render: (v: string) => v ? <Tag color="blue">{v}</Tag> : '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      align: 'center',
      render: (status: string) => {
        const item = statusMap[status] || { color: 'default', text: status || '-', badge: 'default' }
        return <Badge status={item.badge as any} text={item.text} />
      },
    },
    {
      title: 'CNI 类型',
      dataIndex: 'cniType',
      width: 100,
      align: 'center',
      render: (v: string) => v || '-',
    },
    {
      title: '节点数',
      dataIndex: 'nodes',
      width: 80,
      align: 'center',
      render: (nodes: any[]) => nodes?.length || 0,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 170,
      render: (v: string) => v ? new Date(v).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      width: 220,
      align: 'center',
      fixed: 'right',
      render: (_: unknown, record: DeployPlan) => (
        <Space size={8} style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Tooltip title="查看详情">
            <a onClick={() => history.push(`${provisionPath}/${record.id}`)}><ProfileOutlined /></a>
          </Tooltip>
          <Tooltip title="预览">
            <a onClick={() => handlePreview(record.id)}><EyeOutlined /></a>
          </Tooltip>
          {['draft', 'failed', 'cancelled', 'canceled'].includes(record.status) && (
            <Tooltip title="编辑">
              <a onClick={() => history.push(`${provisionPath}/${record.id}/edit`)}><EditOutlined /></a>
            </Tooltip>
          )}
          {record.status === 'draft' && (
            <Popconfirm title="确认执行该部署方案？" onConfirm={() => handleExecute(record.id)}>
              <Tooltip title="执行">
                <a style={{ color: '#047857' }}><PlayCircleOutlined /></a>
              </Tooltip>
            </Popconfirm>
          )}
          {record.status === 'running' && (
            <Popconfirm title="确认取消该部署任务？" onConfirm={() => cancelMutation.mutate(record.id)}>
              <Tooltip title="取消">
                <a style={{ color: '#b45309' }}><StopOutlined /></a>
              </Tooltip>
            </Popconfirm>
          )}
          {['failed', 'cancelled', 'canceled'].includes(record.status) && (
            <Popconfirm title="将重新检查部署条件，并从上次失败位置继续，确认重试？" onConfirm={() => handleRetry(record.id)}>
              <Tooltip title="从失败处重试">
                <a style={{ color: '#2563eb' }}><RedoOutlined /></a>
              </Tooltip>
            </Popconfirm>
          )}
          <Popconfirm title="确认删除该部署方案？" onConfirm={() => deleteMutation.mutate(record.id)}>
            <Tooltip title="删除">
              <a style={{ color: '#dc2626' }}><DeleteOutlined /></a>
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <AppPage>
      <ListWorkspace
        summary={<>当前共 {data?.total || 0} 个部署方案</>}
        actions={
          <Space>
            <Button onClick={() => history.push('/clusters/hosts')}>管理主机资源池</Button>
            <Button onClick={() => history.push('/config/credentials')}>管理凭据库</Button>
            <Button icon={<ReloadOutlined />} onClick={() => queryClient.invalidateQueries({ queryKey: ['deploy-plans'] })}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => history.push(`${provisionPath}/create`)}>
              部署K8S集群
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={data?.items || []}
          loading={isLoading}
          scroll={{ x: 1200 }}
          pagination={{
            current: page,
            pageSize,
            total: data?.total || 0,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (p, ps) => { setPage(p); setPageSize(ps) },
          }}
          locale={{ emptyText: <Empty description="暂无部署方案，点击「创建部署方案」开始" /> }}
        />
      </ListWorkspace>

      {/* 预览弹窗 - 展示每台机器要执行的命令 */}
      <Modal
        title={
          <Space>
            <EyeOutlined />
            <span>部署计划预览</span>
          </Space>
        }
        open={previewId !== null}
        onCancel={() => { setPreviewId(null); setPreviewData(null) }}
        footer={null}
        width={900}
        loading={previewLoading}
      >
        {previewData && (
          <>
            <Descriptions bordered column={2} size="small" style={{ marginBottom: 16 }}>
              <Descriptions.Item label="方案名称">{previewData.planName || '-'}</Descriptions.Item>
              <Descriptions.Item label="集群名称">{previewData.clusterName || '-'}</Descriptions.Item>
              <Descriptions.Item label="K8s 版本">{previewData.k8sVersion || '-'}</Descriptions.Item>
              <Descriptions.Item label="CNI 类型">{previewData.cniType || '-'}</Descriptions.Item>
            </Descriptions>


            {previewData.nodes && previewData.nodes.length > 0 && (
              <>
                <Text strong style={{ display: 'block', marginBottom: 12, fontSize: 14 }}>
                  <DesktopOutlined style={{ marginRight: 8 }} />
                  节点部署流程（共 {previewData.nodes.length} 台机器）
                </Text>
                <Collapse
                  accordion
                  items={previewData.nodes.map((node: DeployDryRunNodeFlow, index: number) => ({
                    key: String(index),
                    label: (
                      <Space>
                        <Tag color={roleColorMap[node.role] || 'default'}>{node.role?.toUpperCase()}</Tag>
                        <Text strong>{node.serverName || `节点-${index + 1}`}</Text>
                        <Text type="secondary">{node.ip}</Text>
                      </Space>
                    ),
                    children: (
                      <div>
                        {node.steps && node.steps.map((step: DeployDryRunStep, stepIndex: number) => (
                          <Card
                            key={step.key || stepIndex}
                            size="small"
                            style={{ marginBottom: 12 }}
                            title={
                              <Space>
                                <Tag color="blue">{phaseMap[step.phase] || step.phase}</Tag>
                                <Text strong>{step.title || `步骤 ${stepIndex + 1}`}</Text>
                                {step.appliesTo && (
                                  <Tag color={step.appliesTo === 'master' ? 'red' : step.appliesTo === 'worker' ? 'blue' : 'default'}>
                                    {step.appliesTo === 'all' ? '所有节点' : step.appliesTo === 'master' ? 'Master' : 'Worker'}
                                  </Tag>
                                )}
                              </Space>
                            }
                            extra={<Text type="secondary" style={{ fontSize: 12 }}>{step.description}</Text>}
                          >
                            {step.tasks && step.tasks.length > 0 ? (
                              <ul style={{ margin: 0, paddingLeft: 20, fontSize: 13, lineHeight: '1.8' }}>
                                {step.tasks.map((task, taskIdx) => (
                                  <li key={taskIdx}>{task}</li>
                                ))}
                              </ul>
                            ) : (
                              <Text type="secondary">无任务</Text>
                            )}
                            {step.dependsOn && step.dependsOn.length > 0 && (
                              <div style={{ marginTop: 8, fontSize: 12, color: '#999' }}>
                                依赖: {step.dependsOn.map((d) => <Tag key={d} style={{ fontSize: 11 }}>{d}</Tag>)}
                              </div>
                            )}
                          </Card>
                        ))}
                      </div>
                    ),
                  }))}
                />
              </>
            )}

            {previewData.summary && Object.keys(previewData.summary).length > 0 && (
              <Descriptions bordered column={4} size="small" style={{ marginTop: 16 }}>
                <Descriptions.Item label="总节点数">{previewData.nodes?.length || 0}</Descriptions.Item>
                {Object.entries(previewData.summary).map(([phase, count]) => (
                  <Descriptions.Item key={phase} label={`${phaseMap[phase] || phase}步骤`}>{count as number}</Descriptions.Item>
                ))}
              </Descriptions>
            )}
          </>
        )}
      </Modal>
    </AppPage>
  )
}
