import React, { useMemo, useState } from 'react'
import {
  ModalForm,
  ProFormText,
  ProFormTextArea,
  ProTable,
  type ProColumns,
} from '@ant-design/pro-components'
import { Button, Input, message, Popconfirm, Tooltip } from 'antd'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { createRole, deleteRole, listRoles, updateRole } from '@/services/system'
import { roleCreateSchema } from '@/schemas/system'
import { formatDate } from '@/utils'
import type { Role } from '@/types'
import type { RoleCreateInput } from '@/schemas/system'

/** 角色管理页 */
const RolesPage: React.FC = () => {
  const queryClient = useQueryClient()
  const [createVisible, setCreateVisible] = useState(false)
  const [editVisible, setEditVisible] = useState(false)
  const [currentRole, setCurrentRole] = useState<Role | null>(null)
  const [searchText, setSearchText] = useState('')

  const { data, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ['roles'],
    queryFn: () => listRoles(),
  })

  const roles = useMemo(() => {
    const keyword = searchText.trim().toLowerCase()
    return (data?.items || []).filter((role) => {
      if (!keyword) {
        return true
      }
      return (
        role.name.toLowerCase().includes(keyword) ||
        role.code.toLowerCase().includes(keyword) ||
        role.description?.toLowerCase().includes(keyword)
      )
    })
  }, [data?.items, searchText])

  const summary = useMemo(() => {
    const source = data?.items || []
    return {
      total: source.length,
      withPermissions: source.filter((role) => (role.permissions?.length || 0) > 0).length,
      systemRoles: source.filter((role) => ['admin', 'operator', 'viewer'].includes(role.code))
        .length,
    }
  }, [data?.items])

  const createMutation = useMutation({
    mutationFn: (payload: RoleCreateInput) => createRole(payload),
    onSuccess: () => {
      message.success('创建成功')
      setCreateVisible(false)
      queryClient.invalidateQueries({ queryKey: ['roles'] })
    },
    onError: () => message.error('创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: Partial<RoleCreateInput> }) =>
      updateRole(id, payload),
    onSuccess: () => {
      message.success('更新成功')
      setEditVisible(false)
      setCurrentRole(null)
      queryClient.invalidateQueries({ queryKey: ['roles'] })
    },
    onError: () => message.error('更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteRole(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['roles'] })
    },
    onError: () => message.error('删除失败'),
  })

  const columns: ProColumns<Role>[] = [
    {
      title: '角色',
      dataIndex: 'name',
      width: 260,
      render: (_, record) => (
        <div className="app-table-user">
          <span className="app-table-user__name">{record.name}</span>
          <span className="app-table-user__meta">{record.code}</span>
        </div>
      ),
    },
    {
      title: '说明',
      dataIndex: 'description',
      align: 'center',
      render: (_, record) => (
        <div className="app-table-stack">
          <span className="app-table-stack__main">{record.description || '未填写描述'}</span>
          <span className="app-table-stack__sub">权限项 {record.permissions?.length || 0} 个</span>
        </div>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 180,
      align: 'center',
      render: (_, record) => (
        <div className="app-table-stack">
          <span className="app-table-stack__main">
            {formatDate(record.createdAt, 'YYYY-MM-DD')}
          </span>
          <span className="app-table-stack__sub">{formatDate(record.createdAt, 'HH:mm:ss')}</span>
        </div>
      ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 96,
      align: 'center',
      render: (_, record) => (
        <div className="app-table-actions app-table-actions--icon">
          <Tooltip title="编辑">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => {
                setCurrentRole(record)
                setEditVisible(true)
              }}
            />
          </Tooltip>
          <Popconfirm title="确定删除该角色？" onConfirm={() => deleteMutation.mutate(record.id)}>
            <Tooltip title="删除">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </div>
      ),
    },
  ]

  return (
    <AppPage>
      <div className="app-data-console">
        <section className="app-data-console__statgrid">
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">已有角色</span>
            <strong className="app-data-console__stat-value">{summary.total}</strong>
            <span className="app-data-console__stat-hint">覆盖当前权限模型</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">已配置权限</span>
            <strong className="app-data-console__stat-value">{summary.withPermissions}</strong>
            <span className="app-data-console__stat-hint">已具备权限项映射</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">当前结果</span>
            <strong className="app-data-console__stat-value">{roles.length}</strong>
            <span className="app-data-console__stat-hint">基于当前搜索条件</span>
          </div>
        </section>

        <section className="app-data-console__filters">
          <div className="app-data-console__filters-left">
            <Input
              placeholder="搜索角色名称、编码或说明"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(event) => setSearchText(event.target.value)}
              style={{ width: 280 }}
              allowClear
            />
            <Button icon={<ReloadOutlined />} loading={isRefetching} onClick={() => refetch()}>
              刷新
            </Button>
          </div>
          <div className="app-data-console__filters-right">
            <span className="app-data-console__meta">
              系统内置 <strong>{summary.systemRoles}</strong>
            </span>
            <span className="app-data-console__meta">
              当前展示 <strong>{roles.length}</strong> / {summary.total}
            </span>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
              创建角色
            </Button>
          </div>
        </section>

        <ProTable<Role>
          className="app-data-console__protable"
          columns={columns}
          dataSource={roles}
          loading={isLoading}
          rowKey="id"
          search={false}
          options={false}
          cardBordered
          tableAlertRender={false}
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
          toolBarRender={false}
        />
      </div>

      <ModalForm
        title="创建角色"
        open={createVisible}
        onOpenChange={setCreateVisible}
        onFinish={async (values) => {
          const result = roleCreateSchema.safeParse(values)
          if (!result.success) {
            message.error(result.error.issues[0]?.message || '请检查表单输入')
            return false
          }

          createMutation.mutate(result.data)
          return true
        }}
        width={560}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="name" label="角色名称" rules={[{ required: true }]} />
        <ProFormText name="code" label="角色编码" rules={[{ required: true }]} />
        <ProFormTextArea name="description" label="描述" fieldProps={{ rows: 4 }} />
      </ModalForm>

      <ModalForm
        title={`编辑角色${currentRole ? ` - ${currentRole.name}` : ''}`}
        open={editVisible}
        onOpenChange={(visible) => {
          setEditVisible(visible)
          if (!visible) {
            setCurrentRole(null)
          }
        }}
        onFinish={async (values) => {
          if (!currentRole) {
            return false
          }

          updateMutation.mutate({ id: currentRole.id, payload: values })
          return true
        }}
        width={560}
        modalProps={{ destroyOnClose: true }}
        initialValues={currentRole || undefined}
      >
        <ProFormText name="name" label="角色名称" rules={[{ required: true }]} />
        <ProFormText name="code" label="角色编码" rules={[{ required: true }]} disabled />
        <ProFormTextArea name="description" label="描述" fieldProps={{ rows: 4 }} />
      </ModalForm>
    </AppPage>
  )
}

export default RolesPage
