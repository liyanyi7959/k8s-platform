import { ProTable, type ProColumns } from '@ant-design/pro-components'
import {
  Button,
  Tag,
  Space,
  Popconfirm,
  message,
  Drawer,
  Descriptions,
  Tabs,
  Progress,
  Badge,
  Modal,
  Switch,
  InputNumber,
  Typography,
  Row,
  Col,
  Card,
  Alert,
  Tooltip,
  Input,
  Select,
  Table,
} from 'antd'
import {
  ReloadOutlined,
  ProfileOutlined,
  DeleteOutlined,
  StopOutlined,
  PlayCircleOutlined,
  ExclamationCircleOutlined,
  InfoCircleOutlined,
  EyeOutlined,
  CloudServerOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useMemo } from 'react'
import {
  getNodes,
  getPodEvents,
  listPods,
  cordonNode,
  uncordonNode,
  drainNode,
  deleteNode,
} from '@/services/k8s'
import { AppPage } from '@/components'
import YamlDrawer, { useYamlDrawer } from '@/components/YamlDrawer'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'

const { Text } = Typography

export default function NodesPage() {
  const clusterId = useClusterId()
  const qc = useQueryClient()
  const [searchType, setSearchType] = useState<'name' | 'status' | 'role'>('name')
  const [searchValue, setSearchValue] = useState('')

  // ═══ YAML Drawer ═══
  const yamlDrawer = useYamlDrawer()

  // ═══ Detail Drawer ═══
  const [detailDrawer, setDetailDrawer] = useState<{ open: boolean; name?: string }>({
    open: false,
  })

  // ═══ Events ═══
  const { data: eventsData, isLoading: eventsLoading } = useQuery({
    queryKey: ['node-events', clusterId, detailDrawer.name],
    queryFn: ({ signal }) => getPodEvents(clusterId, '', detailDrawer.name || '', signal),
    enabled: detailDrawer.open && !!detailDrawer.name,
  })

  // ═══ Node Pods ═══
  const { data: podsData, isLoading: podsLoading } = useQuery({
    queryKey: ['node-pods', clusterId, detailDrawer.name],
    queryFn: ({ signal }) => listPods(clusterId, {}, signal),
    enabled: detailDrawer.open && !!detailDrawer.name,
  })

  const nodePods = (podsData?.items || []).filter((p: any) => p.nodeName === detailDrawer.name)

  // ═══ Drain Modal ═══
  const [drainModal, setDrainModal] = useState<{
    open: boolean
    name?: string
    force: boolean
    timeout: number
  }>({ open: false, force: false, timeout: 300 })

  // ═══ Nodes Query ═══
  const { data: nodesData, isLoading, refetch } = useQuery({
    queryKey: ['k8s-nodes', clusterId],
    queryFn: ({ signal }) => getNodes(clusterId, signal),
    enabled: !!clusterId,
    refetchInterval: detailDrawer.open || yamlDrawer.open || drainModal.open ? false : 30_000,
  })

  // ═══ All Pods (for pod count per node) ═══
  const { data: allPodsData } = useQuery({
    queryKey: ['k8s-all-pods', clusterId],
    queryFn: ({ signal }) => listPods(clusterId, {}, signal),
    enabled: !!clusterId,
    refetchInterval: detailDrawer.open || yamlDrawer.open || drainModal.open ? false : 30_000,
  })

  const nodePodCountMap = useMemo(() => {
    const map = new Map<string, number>()
    ;(allPodsData?.items || []).forEach((pod: any) => {
      if (pod.nodeName) {
        map.set(pod.nodeName, (map.get(pod.nodeName) || 0) + 1)
      }
    })
    return map
  }, [allPodsData])

  const filteredData = (() => {
    let items = nodesData?.items || nodesData || []
    if (searchValue) {
      const v = searchValue.toLowerCase()
      items = items.filter((i: any) => {
        if (searchType === 'name') return i.name?.toLowerCase().includes(v)
        if (searchType === 'status') return i.status?.toLowerCase().includes(v)
        // role
        return Array.isArray(i.roles) && i.roles.some((r: string) => r.toLowerCase().includes(v))
      })
    }
    return items
  })()

  const detail = detailDrawer.name ? filteredData.find((n: any) => n.name === detailDrawer.name) || null : null

  // ═══ Mutations ═══
  const cordonMutation = useMutation({
    mutationFn: (name: string) => cordonNode(clusterId, name),
    onSuccess: () => {
      message.success('已停止调度')
      qc.invalidateQueries({ queryKey: ['k8s-nodes', clusterId] })
    },
    onError: () => message.error('停止调度失败'),
  })

  const uncordonMutation = useMutation({
    mutationFn: (name: string) => uncordonNode(clusterId, name),
    onSuccess: () => {
      message.success('已恢复调度')
      qc.invalidateQueries({ queryKey: ['k8s-nodes', clusterId] })
    },
    onError: () => message.error('恢复调度失败'),
  })

  const drainMutation = useMutation({
    mutationFn: ({ name, force, timeout }: { name: string; force: boolean; timeout: number }) =>
      drainNode(clusterId, name, { force, timeout_seconds: timeout, ignore_daemonsets: true }),
    onSuccess: () => {
      message.success('驱逐完成')
      qc.invalidateQueries({ queryKey: ['k8s-nodes', clusterId] })
      setDrainModal({ open: false, force: false, timeout: 300 })
    },
    onError: () => message.error('驱逐失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteNode(clusterId, name),
    onSuccess: () => {
      message.success('删除成功')
      qc.invalidateQueries({ queryKey: ['k8s-nodes', clusterId] })
    },
    onError: () => message.error('删除失败'),
  })

  // ═══ Status helpers ═══
  const isCordoned = (status: string) =>
    status === 'SchedulingDisabled' || status?.includes('SchedulingDisabled')

  const statusColor = (status: string) => {
    if (status === 'Ready') return 'success'
    if (status === 'SchedulingDisabled' || status?.includes('SchedulingDisabled')) return 'warning'
    return 'error'
  }

  // ═══ Columns ═══
  const columns: ProColumns[] = [
    { title: '名称', dataIndex: 'name', width: 200, ellipsis: true, fixed: 'left', render: (_, record) => <Text strong>{record.name}</Text> },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      render: (_, r) => (
        <Space size={4}>
          <Badge status={statusColor(r.status) as any} text={<Text style={{ fontSize: 12 }}>{r.status}</Text>} />
          {isCordoned(r.status) && <Text type="warning" style={{ fontSize: 11 }}>调度已停止</Text>}
        </Space>
      ),
      filters: true,
      onFilter: true,
      valueEnum: {
        Ready: { text: 'Ready' },
        NotReady: { text: 'NotReady' },
        SchedulingDisabled: { text: 'SchedulingDisabled' },
      },
    },
    {
      title: 'Roles',
      dataIndex: 'roles',
      width: 100,
      ellipsis: true,
      render: (_, r) => {
        const roles = Array.isArray(r.roles) && r.roles.length > 0 ? r.roles.join(', ') : 'worker'
        return <Text style={{ fontSize: 12 }}>{roles}</Text>
      },
    },
    { title: 'InternalIP', dataIndex: 'ip', width: 130 },
    {
      title: 'CPU',
      width: 80,
      align: 'center' as const,
      render: (_, r) => {
        const cap = parseInt(r.capacity?.cpu || '0')
        const alloc = parseInt(r.allocatable?.cpu || '0')
        return (
          <Tooltip title={`总量 ${cap} 核，可分配 ${alloc} 核`}>
            <Text style={{ fontSize: 12 }}>{cap || '-'} 核</Text>
          </Tooltip>
        )
      },
    },
    {
      title: '内存',
      width: 90,
      align: 'center' as const,
      render: (_, r) => {
        const fmtMem = (v?: string) => {
          if (!v) return '-'
          if (v.endsWith('Ki')) return `${(parseInt(v) / 1024 / 1024).toFixed(0)} Gi`
          if (v.endsWith('Mi')) return `${(parseInt(v) / 1024).toFixed(0)} Gi`
          if (v.endsWith('Gi')) return v
          return v
        }
        return (
          <Tooltip title={`总量 ${fmtMem(r.capacity?.memory)}，可分配 ${fmtMem(r.allocatable?.memory)}`}>
            <Text style={{ fontSize: 12 }}>{fmtMem(r.capacity?.memory)}</Text>
          </Tooltip>
        )
      },
    },
    {
      title: 'Pods',
      width: 80,
      align: 'center' as const,
      render: (_, r) => {
        const used = nodePodCountMap.get(r.name) || 0
        const total = r.capacity?.pods || '-'
        return (
          <Tooltip title={`已运行 ${used} 个，最大 ${total} 个`}>
            <Text style={{ fontSize: 12 }}>{used} / {total}</Text>
          </Tooltip>
        )
      },
    },
    {
      title: 'Taints',
      dataIndex: 'taints',
      width: 90,
      align: 'center' as const,
      render: (_, r) =>
        r.taints && r.taints.length > 0 ? (
          <Text type="warning" style={{ fontSize: 12 }}>{r.taints.length} 个</Text>
        ) : (
          <Text type="secondary" style={{ fontSize: 12 }}>无</Text>
        ),
    },
    { title: 'kubelet', dataIndex: 'kubeletVersion', width: 120, ellipsis: true },
    { title: 'OS', dataIndex: 'osImage', width: 140, ellipsis: true },
    { title: 'PodCIDR', dataIndex: 'podCIDR', width: 140, ellipsis: true, hideInSearch: true },
    { title: 'Age', dataIndex: 'age', width: 110, align: 'center' as const, ellipsis: true },
    {
      title: '操作',
      valueType: 'option',
      width: 180,
      fixed: 'right',
      align: 'center' as const,
      render: (_, record) => (
        <Space size={8} style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Tooltip title="详情">
            <a onClick={() => setDetailDrawer({ open: true, name: record.name })}>
              <EyeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => yamlDrawer.openYaml(record.name)}>
              <ProfileOutlined />
            </a>
          </Tooltip>
          {isCordoned(record.status) ? (
            <Popconfirm
              title="确定恢复该节点调度？"
              onConfirm={() => uncordonMutation.mutate(record.name)}
            >
              <Tooltip title="恢复调度">
                <a style={{ color: '#52c41a' }}>
                  <PlayCircleOutlined />
                </a>
              </Tooltip>
            </Popconfirm>
          ) : (
            <Popconfirm
              title="确定停止该节点调度？"
              onConfirm={() => cordonMutation.mutate(record.name)}
            >
              <Tooltip title="停止调度">
                <a style={{ color: '#faad14' }}>
                  <StopOutlined />
                </a>
              </Tooltip>
            </Popconfirm>
          )}
          <Popconfirm
            title="确定驱逐该节点上的所有 Pod？"
            description="将忽略 DaemonSet 管理的 Pod"
            onConfirm={() =>
              setDrainModal({ open: true, name: record.name, force: false, timeout: 300 })
            }
          >
            <Tooltip title="驱逐">
              <a style={{ color: '#ff4d4f' }}>
                <ExclamationCircleOutlined />
              </a>
            </Tooltip>
          </Popconfirm>
          <Popconfirm title="确定删除该节点？" onConfirm={() => deleteMutation.mutate(record.name)}>
            <Tooltip title="删除">
              <a style={{ color: '#ff4d4f' }}>
                <DeleteOutlined />
              </a>
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <AppPage>
      <ProTable
        headerTitle={
          <Space>
            <CloudServerOutlined />
            节点管理
          </Space>
        }
        rowKey="name"
        search={false}
        options={{ reload: false }}
        dataSource={filteredData}
        loading={isLoading}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        toolBarRender={() => [
          <Input.Search
            key="search"
            placeholder={searchType === 'name' ? '按名称搜索' : searchType === 'status' ? '按状态搜索' : '按角色搜索'}
            allowClear
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ width: 280 }}
            addonBefore={<Select value={searchType} onChange={(v) => setSearchType(v)} style={{ width: 70 }}
              options={[{ value: 'name', label: '名称' }, { value: 'status', label: '状态' }, { value: 'role', label: '角色' }]} />}
            prefix={<SearchOutlined />}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>
            刷新
          </Button>,
        ]}
        columns={columns}
        scroll={{ x: 1800 }}
      />

      {/* ═══ YAML Drawer ═══ */}
      <YamlDrawer
        clusterId={clusterId}
        resourceType="nodes"
        name={yamlDrawer.name}
        open={yamlDrawer.open}
        onClose={yamlDrawer.closeYaml}
      />

      {/* ═══ Detail Drawer ═══ */}
      <Drawer
        title={`节点详情 - ${detailDrawer.name}`}
        open={detailDrawer.open}
        onClose={() => setDetailDrawer({ open: false })}
        width={780}
      >
        {!detail ? (
          <div style={{ textAlign: 'center', padding: 40 }}>暂无数据</div>
        ) : (
          detail && (
            <Tabs
              items={[
                {
                  key: 'overview',
                  label: '概览',
                  children: (
                    <Descriptions column={2} bordered size="small">
                      <Descriptions.Item label="名称">{detail.name}</Descriptions.Item>
                      <Descriptions.Item label="状态">
                        <Badge status={statusColor(detail.status) as any} text={detail.status} />
                      </Descriptions.Item>
                      <Descriptions.Item label="Roles">
                        {Array.isArray(detail.roles) && detail.roles.length > 0
                          ? detail.roles.join(', ')
                          : 'worker'}
                      </Descriptions.Item>
                      <Descriptions.Item label="InternalIP">{detail.ip}</Descriptions.Item>
                      <Descriptions.Item label="kubelet 版本">
                        {detail.kubeletVersion || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="OS Image">
                        {detail.osImage || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="PodCIDR">{detail.podCIDR || '-'}</Descriptions.Item>
                      <Descriptions.Item label="Age">{detail.age}</Descriptions.Item>
                      <Descriptions.Item label="CPU" span={1}>
                        <Text>{detail.cpu}</Text>
                      </Descriptions.Item>
                      <Descriptions.Item label="内存" span={1}>
                        <Text>{detail.memory}</Text>
                      </Descriptions.Item>
                      <Descriptions.Item label="Taints" span={2}>
                        {detail.taints && detail.taints.length > 0 ? (
                          detail.taints.map((t: any, i: number) => (
                            <Tag key={i} color={t.effect === 'NoSchedule' ? 'error' : 'warning'}>
                              {t.key}={t.value}:{t.effect}
                            </Tag>
                          ))
                        ) : (
                          <Text type="secondary">无</Text>
                        )}
                      </Descriptions.Item>
                    </Descriptions>
                  ),
                },
                {
                  key: 'conditions',
                  label: '条件',
                  children: detail.conditions ? (
                    <Table
                      size="small"
                      tableLayout="fixed"
                      rowKey={(_, i) => String(i)}
                      pagination={false}
                      dataSource={detail.conditions}
                      columns={[
                        { title: '类型', dataIndex: 'type', width: 120, render: (v) => <Tag>{v}</Tag> },
                        { title: '状态', dataIndex: 'status', width: 80, align: 'center' as const, render: (v) => <Badge status={v === 'True' ? 'success' : 'error'} text={v} /> },
                        { title: '原因', dataIndex: 'reason', ellipsis: true },
                        { title: '信息', dataIndex: 'message', ellipsis: true },
                        { title: '上次转换', dataIndex: 'lastTransitionTime', width: 180 },
                      ]}
                    />
                  ) : (
                    <Text type="secondary">暂无条件数据</Text>
                  ),
                },
                {
                  key: 'addresses',
                  label: '地址',
                  children: detail.addresses ? (
                    <Table
                      size="small"
                      rowKey={(_, i) => String(i)}
                      pagination={false}
                      dataSource={detail.addresses}
                      columns={[
                        { title: '类型', dataIndex: 'type', width: 150, render: (v) => <Tag>{v}</Tag> },
                        { title: '地址', dataIndex: 'address' },
                      ]}
                    />
                  ) : (
                    <Text type="secondary">暂无地址数据</Text>
                  ),
                },
                {
                  key: 'capacity',
                  label: '容量',
                  children: (() => {
                    const fmt = (k: string, v: string) => {
                      if (k === 'cpu') return `${v} 核`
                      if (k === 'pods') return `${v} 个`
                      // 存储/内存类资源：统一转换为 Gi
                      if (k === 'memory' || k === 'ephemeral-storage' || k === 'storage') {
                        // 纯数字（字节）
                        if (/^\d+$/.test(v)) {
                          const bytes = parseInt(v)
                          if (bytes >= 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} Gi`
                          if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} Mi`
                          return `${v} B`
                        }
                        // 带单位的
                        if (v.endsWith('Ki')) return `${(parseInt(v) / 1024 / 1024).toFixed(1)} Gi`
                        if (v.endsWith('Mi')) return `${(parseInt(v) / 1024).toFixed(1)} Gi`
                        if (v.endsWith('Gi')) return v
                        return v
                      }
                      return v
                    }
                    const keys = Object.keys(detail.capacity || {})
                    return (
                      <Table
                        size="small"
                        rowKey="resource"
                        pagination={false}
                        dataSource={keys.map(k => ({
                          resource: k,
                          capacity: fmt(k, String(detail.capacity?.[k] || '-')),
                          allocatable: fmt(k, String(detail.allocatable?.[k] || '-')),
                        }))}
                        columns={[
                          { title: '资源', dataIndex: 'resource', width: 120 },
                          { title: 'Capacity', dataIndex: 'capacity', align: 'center' as const },
                          { title: 'Allocatable', dataIndex: 'allocatable', align: 'center' as const },
                        ]}
                      />
                    )
                  })(),
                },
                {
                  key: 'images',
                  label: '镜像',
                  children: detail.images ? (
                    <Table
                      size="small"
                      tableLayout="fixed"
                      rowKey={(_, i) => String(i)}
                      pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
                      dataSource={detail.images}
                      columns={[
                        { title: '#', width: 45, align: 'center' as const, render: (_, __, i) => i + 1 },
                        { title: '镜像名称', dataIndex: 'names', render: (v: string[]) =>
                          v?.length > 0 ? (
                            <Space size={4} wrap>
                              {v.map((n, j) => <Tooltip key={j} title={n}><Text style={{ fontSize: 11 }}>{n}</Text></Tooltip>)}
                            </Space>
                          ) : '-'
                        },
                        { title: '大小', dataIndex: 'sizeBytes', width: 90, align: 'center' as const, render: (v) => v ? `${(v / 1024 / 1024).toFixed(0)} MB` : '-' },
                      ]}
                    />
                  ) : (
                    <Text type="secondary">暂无镜像数据</Text>
                  ),
                },
                {
                  key: 'pods',
                  label: `Pods (${nodePods.length})`,
                  children: (
                    <Table
                      size="small"
                      rowKey={(r: any) => `${r.namespace}/${r.name}`}
                      loading={podsLoading}
                      pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
                      dataSource={nodePods}
                      columns={[
                        { title: 'Namespace', dataIndex: 'namespace', width: 120, ellipsis: true },
                        { title: '名称', dataIndex: 'name', ellipsis: true },
                        { title: '状态', dataIndex: 'status', width: 100, align: 'center' as const, render: (v: string) => {
                          const colorMap: Record<string, string> = { Running: 'success', Pending: 'processing', Failed: 'error', Succeeded: 'default' }
                          return <Badge status={colorMap[v] as any || 'default'} text={v} />
                        }},
                        { title: '重启', dataIndex: 'restarts', width: 70, align: 'center' as const },
                        { title: 'Age', dataIndex: 'createdAt', width: 110, align: 'center' as const, ellipsis: true, render: (v: string) => v ? formatDate(v) : '-' },
                      ]}
                    />
                  ),
                },
                {
                  key: 'labels',
                  label: '标签/注解',
                  children: (
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                      <Descriptions bordered column={1} size="small" title="Labels">
                        {detail.labels && Object.keys(detail.labels).length > 0
                          ? Object.entries(detail.labels).map(([k, v]) => (
                              <Descriptions.Item key={k} label={k}><Tag color="blue">{String(v)}</Tag></Descriptions.Item>
                            ))
                          : <Descriptions.Item label="-"><Text type="secondary">暂无标签</Text></Descriptions.Item>}
                      </Descriptions>
                      <Descriptions bordered column={1} size="small" title="Annotations">
                        {detail.annotations && Object.keys(detail.annotations).length > 0
                          ? Object.entries(detail.annotations).map(([k, v]) => (
                              <Descriptions.Item key={k} label={k}>
                                <Tooltip title={String(v)}>
                                  <Text style={{ fontSize: 12 }} ellipsis>{String(v)}</Text>
                                </Tooltip>
                              </Descriptions.Item>
                            ))
                          : <Descriptions.Item label="-"><Text type="secondary">暂无注解</Text></Descriptions.Item>}
                      </Descriptions>
                    </Space>
                  ),
                },
                {
                  key: 'events',
                  label: '事件',
                  children: (
                    <Table
                      size="small"
                      rowKey={(_, i) => String(i)}
                      loading={eventsLoading}
                      pagination={{ defaultPageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
                      dataSource={eventsData || []}
                      columns={[
                        { title: '类型', dataIndex: 'type', width: 90, render: (v) => <Tag color={v === 'Warning' ? 'warning' : 'success'}>{v || 'Normal'}</Tag> },
                        { title: '原因', dataIndex: 'reason', width: 140, ellipsis: true },
                        { title: '消息', dataIndex: 'message', ellipsis: true },
                        { title: '时间', dataIndex: 'lastTimestamp', width: 160, render: (v) => v ? formatDate(v) : '-' },
                      ]}
                    />
                  ),
                },
              ]}
            />
          )
        )}
      </Drawer>

      {/* ═══ Drain Modal ═══ */}
      <Modal
        title={`驱逐节点 - ${drainModal.name}`}
        open={drainModal.open}
        onCancel={() => setDrainModal({ open: false, force: false, timeout: 300 })}
        onOk={() =>
          drainMutation.mutate({
            name: drainModal.name!,
            force: drainModal.force,
            timeout: drainModal.timeout,
          })
        }
        confirmLoading={drainMutation.isPending}
        okText="确认驱逐"
        okButtonProps={{ danger: true }}
      >
        <Alert
          type="warning"
          showIcon
          message="驱逐将清除节点上所有非 DaemonSet Pod，请确认操作。"
          style={{ marginBottom: 16 }}
        />
        <Descriptions column={1} size="small">
          <Descriptions.Item label="强制驱逐">
            <Switch
              checked={drainModal.force}
              onChange={(v) => setDrainModal((s) => ({ ...s, force: v }))}
            />
          </Descriptions.Item>
          <Descriptions.Item label="超时时间（秒）">
            <InputNumber
              min={30}
              max={3600}
              value={drainModal.timeout}
              onChange={(v) => setDrainModal((s) => ({ ...s, timeout: v || 300 }))}
            />
          </Descriptions.Item>
        </Descriptions>
      </Modal>
    </AppPage>
  )
}
