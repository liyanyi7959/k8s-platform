import React, { useMemo, useState } from 'react'
import {
  ModalForm,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProTable,
  type ProColumns,
} from '@ant-design/pro-components'
import { Button, Input, message, Popconfirm, Select, Space, Tag, Tooltip } from 'antd'
import {
  DeleteOutlined,
  EditOutlined,
  KeyOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage } from '@/components'
import {
  createUser,
  deleteUser,
  listRoles,
  listUsers,
  resetPassword,
  updateUser,
} from '@/services/system'
import { userCreateSchema } from '@/schemas/system'
import { formatDate } from '@/utils'
import type { User } from '@/types'
import type { UserCreateInput } from '@/schemas/system'

type StatusFilter = 'enabled' | 'disabled' | undefined

const getRoleIds = (user?: User | null) => user?.roles?.map((role) => role.id) || []

/** 用户管理页 */
const UsersPage: React.FC = () => {
  const queryClient = useQueryClient()
  const [createVisible, setCreateVisible] = useState(false)
  const [editVisible, setEditVisible] = useState(false)
  const [resetVisible, setResetVisible] = useState(false)
  const [currentUser, setCurrentUser] = useState<User | null>(null)
  const [searchText, setSearchText] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>(undefined)
  const [roleFilter, setRoleFilter] = useState<number | undefined>(undefined)

  const { data, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ['users'],
    queryFn: () => listUsers(),
  })

  const { data: roleOptionsData = [] } = useQuery({
    queryKey: ['roles-all'],
    queryFn: async () => {
      const result = await listRoles({ page: 1, pageSize: 100 })
      return result.items || []
    },
  })

  const roleOptions = roleOptionsData.map((role) => ({
    label: role.name,
    value: role.id,
  }))

  const users = useMemo(() => {
    return (data?.items || []).filter((user) => {
      const keyword = searchText.trim().toLowerCase()
      const matchKeyword =
        !keyword ||
        user.username.toLowerCase().includes(keyword) ||
        user.nickname?.toLowerCase().includes(keyword) ||
        user.email?.toLowerCase().includes(keyword)

      const matchStatus =
        !statusFilter ||
        (statusFilter === 'enabled' && user.enabled) ||
        (statusFilter === 'disabled' && !user.enabled)

      const matchRole = !roleFilter || user.roles?.some((role) => role.id === roleFilter)

      return matchKeyword && matchStatus && matchRole
    })
  }, [data?.items, roleFilter, searchText, statusFilter])

  const summary = useMemo(() => {
    const source = data?.items || []
    return {
      total: source.length,
      enabled: source.filter((item) => item.enabled).length,
      disabled: source.filter((item) => !item.enabled).length,
      admins: source.filter((item) => item.roles?.some((role) => role.code === 'admin')).length,
    }
  }, [data?.items])

  const createMutation = useMutation({
    mutationFn: (payload: Omit<UserCreateInput, 'confirmPassword'>) => createUser(payload),
    onSuccess: () => {
      message.success('创建成功')
      setCreateVisible(false)
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
    onError: () => message.error('创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: Record<string, unknown> }) =>
      updateUser(id, payload),
    onSuccess: () => {
      message.success('更新成功')
      setEditVisible(false)
      setCurrentUser(null)
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
    onError: () => message.error('更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteUser(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
    onError: () => message.error('删除失败'),
  })

  const resetMutation = useMutation({
    mutationFn: ({ id, password }: { id: number; password: string }) => resetPassword(id, password),
    onSuccess: () => {
      message.success('密码重置成功')
      setResetVisible(false)
      setCurrentUser(null)
    },
    onError: () => message.error('密码重置失败'),
  })

  const columns: ProColumns<User>[] = [
    {
      title: '用户',
      dataIndex: 'username',
      width: 260,
      render: (_, record) => (
        <div className="app-table-user">
          <span className="app-table-user__name">{record.nickname || record.username}</span>
          <span className="app-table-user__meta">
            {record.username}
            {record.email ? ` · ${record.email}` : ''}
          </span>
        </div>
      ),
    },
    {
      title: '角色',
      dataIndex: 'roles',
      width: 220,
      align: 'center',
      render: (_, record) => (
        <Space size={[6, 6]} wrap>
          {record.roles?.length ? (
            record.roles.map((role) => (
              <Tag key={role.id} color={role.code === 'admin' ? 'processing' : 'blue'}>
                {role.name}
              </Tag>
            ))
          ) : (
            <span className="app-table-stack__sub">未分配角色</span>
          )}
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 120,
      align: 'center',
      render: (_, record) => (
        <div className="app-table-stack">
          <span className="app-table-stack__main">{record.enabled ? '启用' : '禁用'}</span>
          <span className="app-table-stack__sub">
            {record.enabled ? '可登录并执行授权操作' : '已阻止登录与变更'}
          </span>
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
      width: 132,
      align: 'center',
      render: (_, record) => (
        <div className="app-table-actions app-table-actions--icon">
          <Tooltip title="编辑">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => {
                setCurrentUser(record)
                setEditVisible(true)
              }}
            />
          </Tooltip>
          <Tooltip title="重置密码">
            <Button
              type="text"
              size="small"
              icon={<KeyOutlined />}
              onClick={() => {
                setCurrentUser(record)
                setResetVisible(true)
              }}
            />
          </Tooltip>
          <Popconfirm title="确定删除该用户？" onConfirm={() => deleteMutation.mutate(record.id)}>
            <Tooltip title="删除">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </div>
      ),
    },
  ]

  const hasFilters = Boolean(searchText || statusFilter || roleFilter)

  return (
    <AppPage>
      <div className="app-data-console">
        <section className="app-data-console__statgrid">
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">启用账号</span>
            <strong className="app-data-console__stat-value">{summary.enabled}</strong>
            <span className="app-data-console__stat-hint">正常参与平台操作</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">禁用账号</span>
            <strong className="app-data-console__stat-value">{summary.disabled}</strong>
            <span className="app-data-console__stat-hint">已阻断登录与控制动作</span>
          </div>
          <div className="app-data-console__stat">
            <span className="app-data-console__stat-label">当前结果</span>
            <strong className="app-data-console__stat-value">{users.length}</strong>
            <span className="app-data-console__stat-hint">基于当前筛选条件展示</span>
          </div>
        </section>

        <section className="app-data-console__filters">
          <div className="app-data-console__filters-left">
            <Input
              placeholder="搜索用户名、昵称或邮箱"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(event) => setSearchText(event.target.value)}
              style={{ width: 260 }}
              allowClear
            />
            <Select
              placeholder="账号状态"
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: 140 }}
              allowClear
              options={[
                { label: '启用', value: 'enabled' },
                { label: '禁用', value: 'disabled' },
              ]}
            />
            <Select
              placeholder="角色筛选"
              value={roleFilter}
              onChange={setRoleFilter}
              style={{ width: 180 }}
              allowClear
              options={roleOptions}
            />
            <Button icon={<ReloadOutlined />} loading={isRefetching} onClick={() => refetch()}>
              刷新
            </Button>
            {hasFilters ? (
              <Button
                onClick={() => {
                  setSearchText('')
                  setStatusFilter(undefined)
                  setRoleFilter(undefined)
                }}
              >
                重置
              </Button>
            ) : null}
          </div>
          <div className="app-data-console__filters-right">
            <span className="app-data-console__meta">
              管理员 <strong>{summary.admins}</strong>
            </span>
            <span className="app-data-console__meta">
              当前展示 <strong>{users.length}</strong> / {summary.total}
            </span>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
              创建用户
            </Button>
          </div>
        </section>

        <ProTable<User>
          className="app-data-console__protable"
          columns={columns}
          dataSource={users}
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
        title="创建用户"
        open={createVisible}
        onOpenChange={setCreateVisible}
        onFinish={async (values) => {
          const result = userCreateSchema.safeParse(values)
          if (!result.success) {
            message.error(result.error.issues[0]?.message || '请检查表单输入')
            return false
          }

          const { confirmPassword, ...payload } = result.data
          createMutation.mutate(payload)
          return true
        }}
        width={560}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText
          name="username"
          label="用户名"
          rules={[{ required: true }]}
          extra="以字母开头，可包含数字与下划线。"
        />
        <ProFormText name="nickname" label="昵称" />
        <ProFormText name="email" label="邮箱" />
        <ProFormSelect
          name="roleIds"
          label="角色"
          mode="multiple"
          rules={[{ required: true, message: '请选择角色' }]}
          options={roleOptions}
        />
        <ProFormSwitch name="enabled" label="启用" initialValue />
        <ProFormText.Password
          name="password"
          label="密码"
          rules={[{ required: true }]}
          extra="至少 8 位，需包含大小写字母、数字和特殊字符。"
        />
        <ProFormText.Password
          name="confirmPassword"
          label="确认密码"
          rules={[{ required: true }]}
        />
      </ModalForm>

      <ModalForm
        title={`编辑用户${currentUser ? ` - ${currentUser.username}` : ''}`}
        open={editVisible}
        onOpenChange={(visible) => {
          setEditVisible(visible)
          if (!visible) {
            setCurrentUser(null)
          }
        }}
        onFinish={async (values) => {
          if (!currentUser) {
            return false
          }

          updateMutation.mutate({
            id: currentUser.id,
            payload: values,
          })
          return true
        }}
        width={560}
        modalProps={{ destroyOnClose: true }}
        initialValues={
          currentUser
            ? {
                username: currentUser.username,
                nickname: currentUser.nickname,
                email: currentUser.email,
                enabled: currentUser.enabled,
                roleIds: getRoleIds(currentUser),
              }
            : undefined
        }
      >
        <ProFormText name="username" label="用户名" disabled />
        <ProFormText name="nickname" label="昵称" />
        <ProFormText name="email" label="邮箱" />
        <ProFormSelect name="roleIds" label="角色" mode="multiple" options={roleOptions} />
        <ProFormSwitch name="enabled" label="启用" />
      </ModalForm>

      <ModalForm
        title={`重置密码${currentUser ? ` - ${currentUser.username}` : ''}`}
        open={resetVisible}
        onOpenChange={(visible) => {
          setResetVisible(visible)
          if (!visible) {
            setCurrentUser(null)
          }
        }}
        onFinish={async (values) => {
          if (!currentUser) {
            return false
          }

          resetMutation.mutate({
            id: currentUser.id,
            password: values.newPassword,
          })
          return true
        }}
        width={420}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText.Password
          name="newPassword"
          label="新密码"
          rules={[
            { required: true, message: '请输入新密码' },
            { min: 8, message: '密码至少 8 位' },
          ]}
          extra="建议使用高强度密码。"
        />
        <ProFormText.Password
          name="confirmPassword"
          label="确认密码"
          rules={[
            { required: true, message: '请再次输入密码' },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('newPassword') === value) {
                  return Promise.resolve()
                }
                return Promise.reject(new Error('两次密码不一致'))
              },
            }),
          ]}
        />
      </ModalForm>
    </AppPage>
  )
}

export default UsersPage
