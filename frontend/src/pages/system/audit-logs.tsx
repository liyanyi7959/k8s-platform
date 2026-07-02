import React, { useMemo, useState } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import { Input, Select, Tag } from 'antd'
import { SearchOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { listAuditLogs } from '@/services/system'
import { formatDate } from '@/utils'
import type { AuditLog } from '@/types'

/** 审计日志页 */
const AuditLogsPage: React.FC = () => {
  const [searchText, setSearchText] = useState('')
  const [actionFilter, setActionFilter] = useState<string | undefined>(undefined)

  const { data, isLoading } = useQuery({
    queryKey: ['audit-logs'],
    queryFn: () => listAuditLogs(),
  })

  const logs = useMemo(() => {
    const keyword = searchText.trim().toLowerCase()

    return (data?.items || []).filter((log) => {
      const matchKeyword =
        !keyword ||
        log.username?.toLowerCase().includes(keyword) ||
        log.resource?.toLowerCase().includes(keyword) ||
        log.ip?.includes(keyword) ||
        log.detail?.toLowerCase().includes(keyword)

      const matchAction = !actionFilter || log.action === actionFilter

      return matchKeyword && matchAction
    })
  }, [actionFilter, data?.items, searchText])

  const summary = useMemo(() => {
    const source = data?.items || []
    return {
      total: source.length,
      risky: source.filter((item) => ['delete', 'update'].includes(item.action)).length,
      logins: source.filter((item) => item.action === 'login').length,
    }
  }, [data?.items])

  const columns: ProColumns<AuditLog>[] = [
    {
      title: '操作者',
      dataIndex: 'username',
      width: 180,
      render: (_, record) => (
        <div className="app-table-user">
          <span className="app-table-user__name">{record.username || '未知用户'}</span>
          <span className="app-table-user__meta">{record.ip || '未知 IP'}</span>
        </div>
      ),
    },
    {
      title: '动作',
      dataIndex: 'action',
      width: 120,
      align: 'center',
      render: (_, record) => {
        const actionMap: Record<string, { label: string; color: string }> = {
          create: { label: '创建', color: 'green' },
          update: { label: '更新', color: 'blue' },
          delete: { label: '删除', color: 'red' },
          login: { label: '登录', color: 'cyan' },
          logout: { label: '登出', color: 'default' },
        }
        const action = actionMap[record.action] || { label: record.action, color: 'default' }

        return <Tag color={action.color}>{action.label}</Tag>
      },
    },
    {
      title: '资源与详情',
      dataIndex: 'resource',
      align: 'center',
      render: (_, record) => (
        <div className="app-table-stack">
          <span className="app-table-stack__main">{record.resource || '未知资源'}</span>
          <span className="app-table-stack__sub">{record.detail || '无详情说明'}</span>
        </div>
      ),
    },
    {
      title: '时间',
      dataIndex: 'timestamp',
      width: 180,
      align: 'center',
      render: (_, record) => (
        <div className="app-table-stack">
          <span className="app-table-stack__main">
            {formatDate(record.timestamp, 'YYYY-MM-DD')}
          </span>
          <span className="app-table-stack__sub">{formatDate(record.timestamp, 'HH:mm:ss')}</span>
        </div>
      ),
    },
  ]

  return (
    <AppPage>
      <div className="app-data-console">
        <section className="app-data-console__statgrid">
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">总日志数</span>
            <strong className="app-data-console__stat-value">{summary.total}</strong>
            <span className="app-data-console__stat-hint">覆盖关键操作轨迹</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">登录事件</span>
            <strong className="app-data-console__stat-value">{summary.logins}</strong>
            <span className="app-data-console__stat-hint">用于会话行为追溯</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">当前结果</span>
            <strong className="app-data-console__stat-value">{logs.length}</strong>
            <span className="app-data-console__stat-hint">按筛选条件实时收敛</span>
          </div>
        </section>

        <section className="app-data-console__filters">
          <div className="app-data-console__filters-left">
            <Input
              placeholder="搜索用户、资源、IP 或详情"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(event) => setSearchText(event.target.value)}
              style={{ width: 280 }}
              allowClear
            />
            <Select
              allowClear
              placeholder="操作类型"
              style={{ width: 140 }}
              value={actionFilter}
              onChange={setActionFilter}
              options={[
                { label: '创建', value: 'create' },
                { label: '更新', value: 'update' },
                { label: '删除', value: 'delete' },
                { label: '登录', value: 'login' },
                { label: '登出', value: 'logout' },
              ]}
            />
          </div>
          <div className="app-data-console__filters-right">
            <span className="app-data-console__meta">
              高风险动作 <strong>{summary.risky}</strong>
            </span>
            <span className="app-data-console__meta">
              当前展示 <strong>{logs.length}</strong> / {summary.total}
            </span>
          </div>
        </section>

        <ProTable<AuditLog>
          className="app-data-console__protable"
          columns={columns}
          dataSource={logs}
          loading={isLoading}
          rowKey="id"
          search={false}
          options={false}
          cardBordered
          tableAlertRender={false}
          pagination={{
            pageSize: 20,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
          toolBarRender={false}
        />
      </div>
    </AppPage>
  )
}

export default AuditLogsPage
