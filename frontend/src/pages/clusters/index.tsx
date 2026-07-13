import React, { useMemo, useState } from 'react'
import { history, useModel } from '@umijs/max'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import {
  Badge,
  Button,
  Descriptions,
  Input,
  Popconfirm,
  Progress,
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import {
  CheckCircleOutlined,
  DeleteOutlined,
  EyeOutlined,
  HeartOutlined,
  ImportOutlined,
  ReloadOutlined,
  SearchOutlined,
  ToolOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage, EmptyState, StatusTag } from '@/components'
import { checkClusterConnection, deleteCluster, listClusters } from '@/services/clusters'
import {
  enterClusterWorkspace,
  formatDate,
  formatNumber,
  formatRelativeTime,
  isClusterHealthy as isHealthy,
  needsClusterAttention as needsAttention,
  normalizeClusterStatus,
} from '@/utils'
import type { Cluster } from '@/types'

const { Text } = Typography

type ClusterStatusFilter = 'active' | 'attention' | 'error' | undefined

function getHealthPercent(total: number, healthy: number) {
  if (total <= 0) return 0
  return Math.round((healthy / total) * 100)
}

/** 集群列表页 */
const ClusterListPage: React.FC = () => {
  const { setCurrentCluster } = useModel('cluster')
  const queryClient = useQueryClient()
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<ClusterStatusFilter>(undefined)

  const { data, isLoading, refetch, isRefetching, dataUpdatedAt } = useQuery({
    queryKey: ['clusters'],
    queryFn: () => listClusters(),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteCluster(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['clusters'] })
    },
  })

  const healthMutation = useMutation({
    mutationFn: (id: number) => checkClusterConnection(id),
    onSuccess: (result) => {
      if (result.apiOk) {
        message.success(`集群连接正常 (${result.nodeReady}/${result.nodeTotal} 节点就绪)`)
      } else {
        message.warning('集群连接异常')
      }
      queryClient.invalidateQueries({ queryKey: ['clusters'] })
    },
  })

  const filteredData = useMemo(() => {
    const items = data?.items || []

    return items.filter((item) => {
      const matchKeyword = !keyword || item.name.toLowerCase().includes(keyword.toLowerCase())
      const normalized = normalizeClusterStatus(item.status)
      const matchStatus =
        !statusFilter ||
        (statusFilter === 'active' && isHealthy(normalized)) ||
        (statusFilter === 'attention' && needsAttention(normalized)) ||
        (statusFilter === 'error' &&
          ['error', 'failed', 'disconnected', 'unhealthy', 'offline'].includes(normalized)) ||
        normalized === statusFilter

      return matchKeyword && matchStatus
    })
  }, [data?.items, keyword, statusFilter])

  const summary = useMemo(() => {
    const items = data?.items || []
    const total = items.length
    const healthy = items.filter((c) => isHealthy(c.status)).length
    const attention = items.filter((c) => needsAttention(c.status)).length
    const checked = items.filter((c) => Boolean(c.lastHealthAt)).length
    const hasUnknownNodes = items.some((c) => c.nodeCount == null || (c.nodeCount === 0 && !isHealthy(c.status)))
    const nodeCount = items.reduce((sum, c) => sum + (c.nodeCount || 0), 0)

    return {
      total,
      healthy,
      attention,
      checked,
      nodeCount: hasUnknownNodes ? null : nodeCount,
      healthPercent: getHealthPercent(total, healthy),
    }
  }, [data?.items])

  const hasFilters = Boolean(keyword || statusFilter)
  const lastUpdatedText = dataUpdatedAt ? formatDate(new Date(dataUpdatedAt), 'HH:mm:ss') : '暂无'

  const handleEnterCluster = (cluster: Cluster, targetPath?: string) => {
    enterClusterWorkspace(cluster, {
      setCurrentCluster,
      targetPath,
    })
  }

  const summaryCards = [
    {
      key: 'all',
      label: '集群总数',
      value: formatNumber(summary.total),
      hint: '纳管中的 Kubernetes 集群',
      icon: <Badge status="processing" />,
      active: !statusFilter,
      tone: 'blue',
      onClick: () => setStatusFilter(undefined),
    },
    {
      key: 'healthy',
      label: '稳定运行',
      value: formatNumber(summary.healthy),
      hint: `健康率 ${summary.healthPercent}%`,
      icon: <CheckCircleOutlined />,
      active: statusFilter === 'active',
      tone: 'success',
      onClick: () => setStatusFilter(statusFilter === 'active' ? undefined : 'active'),
    },
    {
      key: 'attention',
      label: '需要关注',
      value: formatNumber(summary.attention),
      hint: summary.attention > 0 ? '建议优先排查连接与节点状态' : '当前没有异常集群',
      icon: <WarningOutlined />,
      active: statusFilter === 'attention',
      tone: 'warning',
      onClick: () => setStatusFilter(statusFilter === 'attention' ? undefined : 'attention'),
    },
    {
      key: 'nodes',
      label: '节点总量',
      value: summary.nodeCount == null ? '—' : formatNumber(summary.nodeCount),
      hint: `已完成健康检查 ${formatNumber(summary.checked)} 个`,
      icon: <HeartOutlined />,
      active: false,
      tone: 'purple',
      onClick: undefined,
    },
  ]

  const columns: ProColumns<Cluster>[] = [
    {
      title: '集群',
      dataIndex: 'name',
      width: 280,
      render: (_, record) => (
        <div className="app-cluster-namecell">
          <Button
            type="link"
            className="app-cluster-namecell__link"
            onClick={() => handleEnterCluster(record)}
          >
            {record.name}
          </Button>
          <div className="app-cluster-namecell__meta">
            <Tag bordered={false} color="blue">
              {record.type || 'Kubernetes'}
            </Tag>
            {record.k8sVersion ? (
              <Tag bordered={false}>
                {record.k8sVersion.startsWith('v') ? record.k8sVersion : `v${record.k8sVersion}`}
              </Tag>
            ) : null}
          </div>
        </div>
      ),
    },
    {
      title: '节点规模',
      dataIndex: 'nodeCount',
      width: 150,
      align: 'center',
      render: (_, record) => {
        const nodeCount = record.nodeCount
        const isUnknown = nodeCount == null || (nodeCount === 0 && !isHealthy(record.status))
        return (
          <div className="app-cluster-metric">
            <strong className="app-cluster-metric__value">
              {isUnknown ? '—' : formatNumber(nodeCount)}
            </strong>
            <span className="app-cluster-metric__label">节点</span>
          </div>
        )
      },
    },
    {
      title: '运行状态',
      dataIndex: 'status',
      width: 180,
      render: (_, record) => <StatusTag status={record.status} />,
    },
    {
      title: '健康检查',
      dataIndex: 'lastHealthAt',
      width: 190,
      render: (_, record) =>
        record.lastHealthAt ? (
          <div className="app-cluster-timecell">
            <Text>{formatRelativeTime(record.lastHealthAt)}</Text>
            <span className="app-table-meta">{formatDate(record.lastHealthAt, 'MM-DD HH:mm')}</span>
          </div>
        ) : (
          <Tag bordered={false}>未检查</Tag>
        ),
    },
    {
      title: '接入时间',
      dataIndex: 'createdAt',
      width: 180,
      render: (_, record) =>
        record.createdAt ? (
          <div className="app-cluster-timecell">
            <Text>{formatDate(record.createdAt, 'YYYY-MM-DD')}</Text>
            <Tag bordered={false} className="app-cluster-timecell__tag">
              {formatRelativeTime(record.createdAt)}
            </Tag>
          </div>
        ) : (
          <span className="app-cluster-muted">暂无</span>
        ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 180,
      align: 'center',
      render: (_, record) => (
        <div className="app-table-actions app-table-actions--icon">
          <Tooltip title="查看详情">
            <Button
              type="text"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => history.push(`/clusters/${record.id}`)}
            />
          </Tooltip>
          <Tooltip title="执行健康检查">
            <Button
              type="text"
              size="small"
              icon={<CheckCircleOutlined />}
              loading={healthMutation.isPending && healthMutation.variables === record.id}
              onClick={() => healthMutation.mutate(record.id)}
            />
          </Tooltip>
          <Tooltip title={isHealthy(record.status) ? '进入运维' : '请检查集群健康状态'}>
            <Button
              type="text"
              size="small"
              icon={<ToolOutlined />}
              onClick={() => handleEnterCluster(record)}
            />
          </Tooltip>
          <Popconfirm title="确定删除该集群吗？" onConfirm={() => deleteMutation.mutate(record.id)}>
            <Tooltip title="删除">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </div>
      ),
    },
  ]

  return (
    <AppPage
      breadcrumbRender={false}
      content={
        <div className="app-cluster-page-header">
          <span
            className={[
              'app-console-toolbar__metric',
              summary.healthPercent >= 90
                ? 'is-high'
                : summary.healthPercent >= 70
                  ? 'is-medium'
                  : 'is-low',
            ].join(' ')}
          >
            <span>健康率</span>
            <strong>{summary.healthPercent}%</strong>
          </span>
          <Progress
            percent={summary.healthPercent}
            showInfo={false}
            strokeColor={
              summary.healthPercent >= 90
                ? '#059669'
                : summary.healthPercent >= 70
                  ? '#d97706'
                  : '#dc2626'
            }
            trailColor="rgba(148, 163, 184, 0.16)"
            size="small"
          />
          <span className="app-console-toolbar__hint">
            {formatNumber(summary.healthy)} / {formatNumber(summary.total)} 集群稳定
          </span>
        </div>
      }
      extra={
        <Space>
          <Button
            type="primary"
            icon={<ImportOutlined />}
            onClick={() => history.push('/clusters/import')}
          >
            导入新集群
          </Button>
          <Button icon={<ReloadOutlined />} loading={isRefetching} onClick={() => refetch()}>
            刷新
          </Button>
        </Space>
      }
    >
      <div className="app-page-shell app-clusters-page">
        <section className="app-cluster-summary-grid">
          {summaryCards.map((card) => (
            <button
              key={card.key}
              type="button"
              className={[
                'app-cluster-summary-card',
                card.active ? 'is-active' : '',
                card.tone ? `tone-${card.tone}` : '',
              ]
                .filter(Boolean)
                .join(' ')}
              onClick={card.onClick}
              disabled={!card.onClick}
            >
              <div className="app-cluster-summary-card__top">
                <div className="app-cluster-summary-card__title">
                  <span className="app-cluster-summary-card__icon">{card.icon}</span>
                  <span className="app-cluster-summary-card__label">{card.label}</span>
                </div>
                <strong className="app-cluster-summary-card__value">{card.value}</strong>
              </div>
              <span className="app-cluster-summary-card__hint">{card.hint}</span>
            </button>
          ))}
        </section>

        {summary.attention > 0 ? (
          <div className="app-inline-alert">
            <WarningOutlined />
            <span>当前有 {summary.attention} 个集群需要关注，建议优先执行健康检查。</span>
          </div>
        ) : null}

        <section className="app-console-filters">
          <div className="app-console-filters__left">
            <Input
              placeholder="按集群名称搜索"
              prefix={<SearchOutlined />}
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              className="app-console-filters__search"
              allowClear
            />
            <Select
              placeholder="状态筛选"
              value={statusFilter}
              onChange={setStatusFilter}
              className="app-console-filters__select"
              allowClear
              options={[
                { label: '运行稳定', value: 'active' },
                { label: '需要关注', value: 'attention' },
                { label: '异常', value: 'error' },
              ]}
            />
            <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
              刷新
            </Button>
            {hasFilters ? (
              <Button
                onClick={() => {
                  setKeyword('')
                  setStatusFilter(undefined)
                }}
              >
                重置
              </Button>
            ) : null}
          </div>
          <div className="app-console-filters__right">
            <span className="app-console-filters__meta">最近刷新 {lastUpdatedText}</span>
          </div>
        </section>

        <ProTable<Cluster>
          className="app-console-table"
          headerTitle="集群清单"
          columns={columns}
          dataSource={filteredData}
          loading={isLoading}
          rowKey="id"
          rowClassName={(record) =>
            needsAttention(record.status) ? 'app-cluster-row--attention' : ''
          }
          search={false}
          options={false}
          cardBordered={false}
          tableAlertRender={false}
          expandable={{
            expandedRowRender: (record) => (
              <Descriptions size="small" column={3} className="app-cluster-expanded">
                <Descriptions.Item label="集群 ID">{record.id}</Descriptions.Item>
                <Descriptions.Item label="接入来源">
                  {record.type || 'Kubernetes'}
                </Descriptions.Item>
                <Descriptions.Item label="K8s 版本">
                  {record.k8sVersion
                    ? record.k8sVersion.startsWith('v')
                      ? record.k8sVersion
                      : `v${record.k8sVersion}`
                    : '暂无'}
                </Descriptions.Item>
                <Descriptions.Item label="节点总数">
                  {formatNumber(record.nodeCount || 0)}
                </Descriptions.Item>
                <Descriptions.Item label="运行状态">
                  {isHealthy(record.status)
                    ? '运行稳定'
                    : needsAttention(record.status)
                      ? '建议优先检查'
                      : '状态待确认'}
                </Descriptions.Item>
                <Descriptions.Item label="最近健康检查">
                  {record.lastHealthAt ? formatDate(record.lastHealthAt) : '尚未执行'}
                </Descriptions.Item>
              </Descriptions>
            ),
          }}
          rowClassName={(record) =>
            needsAttention(record.status) ? 'app-table-row--attention' : ''
          }
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${formatNumber(total)} 条记录`,
          }}
          toolBarRender={false}
          locale={{
            emptyText: (
              <EmptyState
                description={hasFilters ? '没有匹配当前筛选条件的集群' : '还没有接入任何集群'}
                actionText={hasFilters ? '清空筛选' : '导入集群'}
                onAction={() => {
                  if (hasFilters) {
                    setKeyword('')
                    setStatusFilter(undefined)
                    return
                  }

                  history.push('/clusters/import')
                }}
              />
            ),
          }}
        />
      </div>
    </AppPage>
  )
}

export default ClusterListPage
