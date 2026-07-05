import { ProTable, type ActionType, type ProColumns } from '@ant-design/pro-components'
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
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import {
  getNodes,
  getNodeDetail,
  getNodeYaml,
  cordonNode,
  uncordonNode,
  drainNode,
  deleteNode,
} from '@/services/k8s'
import { AppPage } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'

const { Text } = Typography

export default function NodesPage() {
  const clusterId = useClusterId()
  const actionRef = useRef<ActionType>()
  const qc = useQueryClient()

  // ═══ YAML Drawer ═══
  const [yamlDrawer, setYamlDrawer] = useState<{ open: boolean; name?: string }>({ open: false })
  const { data: yamlData, isLoading: yamlLoading } = useQuery({
    queryKey: ['node-yaml', clusterId, yamlDrawer.name],
    queryFn: ({ signal }) => getNodeYaml(clusterId, yamlDrawer.name!, signal),
    enabled: yamlDrawer.open && !!yamlDrawer.name,
  })

  // ═══ Detail Drawer ═══
  const [detailDrawer, setDetailDrawer] = useState<{ open: boolean; name?: string }>({
    open: false,
  })
  const { data: detail, isLoading: detailLoading } = useQuery({
    queryKey: ['node-detail', clusterId, detailDrawer.name],
    queryFn: ({ signal }) => getNodeDetail(clusterId, detailDrawer.name!, signal),
    enabled: detailDrawer.open && !!detailDrawer.name,
  })

  // ═══ Drain Modal ═══
  const [drainModal, setDrainModal] = useState<{
    open: boolean
    name?: string
    force: boolean
    timeout: number
  }>({ open: false, force: false, timeout: 300 })

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
      width: 140,
      render: (_, r) => (
        <Badge
          status={statusColor(r.status) as any}
          text={
            <Space size={4}>
              <Tag color={statusColor(r.status)}>{r.status}</Tag>
              {isCordoned(r.status) && <Tag color="warning">调度已停止</Tag>}
            </Space>
          }
        />
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
      render: (_, r) => {
        const roles = Array.isArray(r.roles) && r.roles.length > 0 ? r.roles.join(', ') : 'worker'
        return <Tag>{roles}</Tag>
      },
    },
    { title: 'InternalIP', dataIndex: 'ip', width: 130 },
    {
      title: 'CPU',
      width: 140,
      render: (_, r) => (
        <Space direction="vertical" size={0} style={{ width: '100%' }}>
          <Text style={{ fontSize: 11 }}>{r.cpu || '-'}</Text>
          <Progress
            percent={r.cpuPercent || 0}
            size="small"
            showInfo={false}
            strokeColor={(r.cpuPercent || 0) > 80 ? '#ff4d4f' : '#0891b2'}
          />
          <Text type="secondary" style={{ fontSize: 10 }}>
            {r.cpuPercent || 0}%
          </Text>
        </Space>
      ),
    },
    {
      title: '内存',
      width: 140,
      render: (_, r) => (
        <Space direction="vertical" size={0} style={{ width: '100%' }}>
          <Text style={{ fontSize: 11 }}>{r.memory || '-'}</Text>
          <Progress
            percent={r.memoryPercent || 0}
            size="small"
            showInfo={false}
            strokeColor={(r.memoryPercent || 0) > 80 ? '#ff4d4f' : '#7c3aed'}
          />
          <Text type="secondary" style={{ fontSize: 10 }}>
            {r.memoryPercent || 0}%
          </Text>
        </Space>
      ),
    },
    {
      title: 'Pods',
      dataIndex: 'podCount',
      width: 70,
      align: 'center' as const,
      render: (_, r) => <Text>{r.podCount || 0}</Text>,
    },
    {
      title: 'Taints',
      dataIndex: 'taints',
      width: 90,
      align: 'center' as const,
      render: (_, r) =>
        r.taints && r.taints.length > 0 ? (
          <Tag color="warning">{r.taints.length} 个</Tag>
        ) : (
          <Text type="secondary">无</Text>
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
            <a onClick={() => setYamlDrawer({ open: true, name: record.name })}>
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
        actionRef={actionRef}
        rowKey="name"
        search={{ labelWidth: 80 }}
        options={{ reload: false }}
        toolBarRender={() => [
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() => actionRef.current?.reload()}
          >
            刷新
          </Button>,
        ]}
        request={async (params) => {
          const data = await getNodes(clusterId)
          let items = data.items || data || []
          if (params.name) items = items.filter((i: any) => i.name?.includes(params.name))
          if (params.status) items = items.filter((i: any) => i.status === params.status)
          return { data: items, success: true }
        }}
        columns={columns}
        scroll={{ x: 1800 }}
      />

      {/* ═══ YAML Drawer ═══ */}
      <Drawer
        title={`YAML - ${yamlDrawer.name}`}
        open={yamlDrawer.open}
        onClose={() => setYamlDrawer({ open: false })}
        width={800}
      >
        <pre
          style={{
            background: '#1e1e1e',
            color: '#d4d4d4',
            padding: 16,
            borderRadius: 8,
            height: 'calc(100vh - 200px)',
            overflow: 'auto',
            fontSize: 13,
            lineHeight: 1.6,
            fontFamily: 'Consolas, Monaco, monospace',
          }}
        >
          {yamlLoading
            ? '加载中...'
            : (typeof yamlData === 'string' ? yamlData : JSON.stringify(yamlData, null, 2)) ||
              '暂无数据'}
        </pre>
      </Drawer>

      {/* ═══ Detail Drawer ═══ */}
      <Drawer
        title={`节点详情 - ${detailDrawer.name}`}
        open={detailDrawer.open}
        onClose={() => setDetailDrawer({ open: false })}
        width={780}
      >
        {detailLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
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
                    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                      <thead>
                        <tr style={{ background: '#fafafa' }}>
                          {['类型', '状态', '原因', '信息', '上次转换'].map((h) => (
                            <th
                              key={h}
                              style={{
                                padding: '8px 12px',
                                borderBottom: '1px solid #f0f0f0',
                                textAlign: 'left',
                                fontSize: 12,
                              }}
                            >
                              {h}
                            </th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {detail.conditions.map((c: any, i: number) => (
                          <tr
                            key={i}
                            style={{
                              background:
                                c.status !== 'True' && c.type !== 'Ready' ? undefined : undefined,
                            }}
                          >
                            <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                              <Tag>{c.type}</Tag>
                            </td>
                            <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                              <Badge
                                status={c.status === 'True' ? 'success' : 'error'}
                                text={c.status}
                              />
                            </td>
                            <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                              {c.reason || '-'}
                            </td>
                            <td
                              style={{
                                padding: '8px 12px',
                                borderBottom: '1px solid #f0f0f0',
                                maxWidth: 200,
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                              }}
                            >
                              {c.message || '-'}
                            </td>
                            <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                              {c.lastTransitionTime || '-'}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  ) : (
                    <Text type="secondary">暂无条件数据</Text>
                  ),
                },
                {
                  key: 'addresses',
                  label: '地址',
                  children: detail.addresses ? (
                    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                      <thead>
                        <tr style={{ background: '#fafafa' }}>
                          <th style={{ padding: '8px 12px', textAlign: 'left' }}>类型</th>
                          <th style={{ padding: '8px 12px', textAlign: 'left' }}>地址</th>
                        </tr>
                      </thead>
                      <tbody>
                        {detail.addresses.map((a: any, i: number) => (
                          <tr key={i}>
                            <td style={{ padding: '8px 12px' }}>
                              <Tag>{a.type}</Tag>
                            </td>
                            <td style={{ padding: '8px 12px' }}>{a.address}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  ) : (
                    <Text type="secondary">暂无地址数据</Text>
                  ),
                },
                {
                  key: 'capacity',
                  label: '容量',
                  children: (
                    <Row gutter={16}>
                      <Col span={12}>
                        <Card title="Capacity" size="small">
                          {detail.capacity ? (
                            Object.entries(detail.capacity).map(([k, v]) => (
                              <Descriptions.Item key={k} label={k}>
                                <Text>{String(v)}</Text>
                              </Descriptions.Item>
                            ))
                          ) : (
                            <Text type="secondary">-</Text>
                          )}
                        </Card>
                      </Col>
                      <Col span={12}>
                        <Card title="Allocatable" size="small">
                          {detail.allocatable ? (
                            Object.entries(detail.allocatable).map(([k, v]) => (
                              <Descriptions.Item key={k} label={k}>
                                <Text>{String(v)}</Text>
                              </Descriptions.Item>
                            ))
                          ) : (
                            <Text type="secondary">-</Text>
                          )}
                        </Card>
                      </Col>
                    </Row>
                  ),
                },
                {
                  key: 'images',
                  label: '镜像',
                  children: detail.images ? (
                    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                      <thead>
                        <tr style={{ background: '#fafafa' }}>
                          <th style={{ padding: '8px 12px', textAlign: 'left' }}>镜像名称</th>
                          <th style={{ padding: '8px 12px', textAlign: 'left' }}>大小</th>
                        </tr>
                      </thead>
                      <tbody>
                        {detail.images.map((img: any, i: number) => (
                          <tr key={i}>
                            <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                              {img.names?.map((n: string, j: number) => (
                                <Tag key={j} style={{ marginBottom: 2 }}>
                                  {n}
                                </Tag>
                              ))}
                            </td>
                            <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                              {img.sizeBytes
                                ? `${(img.sizeBytes / 1024 / 1024).toFixed(1)} MB`
                                : '-'}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  ) : (
                    <Text type="secondary">暂无镜像数据</Text>
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
