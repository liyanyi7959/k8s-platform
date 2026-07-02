/**
 * Pod 管理页
 * 完整功能：列表+筛选+批量删除+强制删除+多Pod日志+终端+详情+YAML
 */
import React, { useState, useMemo } from 'react'
import { ProTable, type ProColumns } from '@ant-design/pro-components'
import {
  Button,
  Space,
  message,
  Popconfirm,
  Tag,
  Tooltip,
  Drawer,
  Descriptions,
  Tabs,
  Badge,
  Typography,
  Modal,
  Switch,
  Alert,
} from 'antd'
import {
  DeleteOutlined,
  CodeOutlined,
  FileTextOutlined,
  ReloadOutlined,
  DesktopOutlined,
  InfoCircleOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listPods, deletePod, getPodLogs, getPodTerminalUrl, getPodYaml, getPodInspection } from '@/services/k8s'
import { AppPage, PodStatusTag, NamespaceSelector } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import { withCenterStyleBatch } from '@/utils/fieldStyle'
import type { Pod } from '@/types'

const { Text } = Typography

const PodsPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState<string>('')
  const [selectedKeys, setSelectedKeys] = useState<string[]>([])

  // ═══ Drawers ═══
  const [logDrawer, setLogDrawer] = useState<{ open: boolean; pod?: Pod; pods?: Pod[] }>({
    open: false,
  })
  const [detailDrawer, setDetailDrawer] = useState<{ open: boolean; pod?: Pod }>({ open: false })
  const [yamlDrawer, setYamlDrawer] = useState<{
    open: boolean
    pod?: Pod
    yaml?: string
    loading: boolean
  }>({ open: false, loading: false })
  const [logs, setLogs] = useState<string>('')
  const [logsLoading, setLogsLoading] = useState(false)
  const [forceDelete, setForceDelete] = useState(false)
  const [inspectDrawer, setInspectDrawer] = useState<{
    open: boolean
    pod?: Pod
    text?: string
    loading: boolean
  }>({ open: false, loading: false })
  const [batchDeleteModal, setBatchDeleteModal] = useState(false)

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['pods', clusterId, namespace],
    queryFn: () => listPods(clusterId, { namespace: namespace || undefined }),
    enabled: !!clusterId,
    refetchInterval: 30_000,
  })

  // ═══ Mutations ═══
  const deleteMutation = useMutation({
    mutationFn: (params: { namespace: string; name: string; force?: boolean }) =>
      deletePod(clusterId, params.namespace, params.name, params.force),
    onSuccess: () => {
      message.success('Pod 已删除')
      queryClient.invalidateQueries({ queryKey: ['pods', clusterId] })
    },
  })

  const batchDeleteMutation = useMutation({
    mutationFn: async (pods: Pod[]) => {
      for (const pod of pods) {
        await deletePod(clusterId, pod.namespace, pod.name, forceDelete)
      }
    },
    onSuccess: () => {
      message.success(`已删除 ${selectedKeys.length} 个 Pod`)
      setSelectedKeys([])
      setBatchDeleteModal(false)
      queryClient.invalidateQueries({ queryKey: ['pods', clusterId] })
    },
    onError: () => message.error('批量删除失败'),
  })

  // ═══ Handlers ═══
  const handleViewLogs = async (pod: Pod) => {
    setLogDrawer({ open: true, pod })
    setLogsLoading(true)
    try {
      const res = await getPodLogs(clusterId, pod.namespace, pod.name, 200)
      setLogs(res.logs)
    } catch {
      setLogs('获取日志失败')
    } finally {
      setLogsLoading(false)
    }
  }

  const handleMultiPodLogs = async (pods: Pod[]) => {
    setLogDrawer({ open: true, pods })
    setLogsLoading(true)
    try {
      const results = await Promise.all(
        pods.map(async (p) => {
          try {
            const res = await getPodLogs(clusterId, p.namespace, p.name, 50)
            return `═══ ${p.namespace}/${p.name} ═══\n${res.logs}`
          } catch {
            return `═══ ${p.namespace}/${p.name} ═══\n获取日志失败`
          }
        }),
      )
      setLogs(results.join('\n\n'))
    } catch {
      setLogs('获取日志失败')
    } finally {
      setLogsLoading(false)
    }
  }

  const handleTerminal = async (pod: Pod) => {
    try {
      const res = await getPodTerminalUrl(clusterId, pod.namespace, pod.name)
      const token = localStorage.getItem('token') || ''
      const params = new URLSearchParams({ ws_url: res.url })
      if (token) params.set('token', token)
      window.open(`/k8s/terminal?${params.toString()}`, '_blank', 'width=900,height=600')
    } catch {
      message.error('获取终端连接失败')
    }
  }

  const handleViewYaml = async (pod: Pod) => {
    setYamlDrawer({ open: true, pod, loading: true })
    try {
      const res = await getPodYaml(clusterId, pod.namespace, pod.name)
      setYamlDrawer((prev) => ({ ...prev, yaml: res.yaml, loading: false }))
    } catch {
      setYamlDrawer((prev) => ({ ...prev, yaml: '获取 YAML 失败', loading: false }))
    }
  }

  const handleInspection = async (pod: Pod) => {
    setInspectDrawer({ open: true, pod, loading: true })
    try {
      const res = await getPodInspection(clusterId, pod.namespace, pod.name)
      setInspectDrawer((prev) => ({ ...prev, text: res.text, loading: false }))
    } catch {
      setInspectDrawer((prev) => ({ ...prev, text: '获取巡检结果失败', loading: false }))
    }
  }

  const selectedPods = useMemo(() => {
    return (data?.items || []).filter((p) => selectedKeys.includes(`${p.namespace}/${p.name}`))
  }, [data, selectedKeys])

  // ═══ Columns ═══
  const rawColumns: ProColumns<Pod>[] = [
    {
      title: '名称',
      dataIndex: 'name',
      ellipsis: true,
      render: (_, record) => (
        <a onClick={() => setDetailDrawer({ open: true, pod: record })} style={{ fontWeight: 500 }}>
          {record.name}
        </a>
      ),
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      width: 110,
      render: (t) => <Tag>{t as string}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 130,
      render: (_, record) => <PodStatusTag status={record.status} ready={record.ready} />,
    },
    {
      title: 'Ready',
      dataIndex: 'ready',
      width: 70,
      render: (t) =>
        t ? <Badge status="success" text="是" /> : <Badge status="error" text="否" />,
    },
    {
      title: '重启',
      dataIndex: 'restarts',
      width: 70,
      render: (t) => {
        const v = t as number
        return <span style={{ color: v > 5 ? '#ff4d4f' : v > 0 ? '#faad14' : '#52c41a' }}>{v}</span>
      },
    },
    {
      title: 'Pod IP',
      dataIndex: 'ip',
      width: 120,
      render: (t) => <Tag color="blue">{t as string}</Tag>,
    },
    { title: '节点', dataIndex: 'nodeName', width: 130, ellipsis: true },
    {
      title: '所属',
      width: 180,
      ellipsis: true,
      render: (_, r) => {
        if (!r.ownerName) return <Text type="secondary">-</Text>
        const full = `${r.ownerKind || 'ReplicaSet'}/${r.ownerName}`
        return (
          <Tooltip title={full}>
            <Tag
              style={{
                maxWidth: '100%',
                display: 'inline-block',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                verticalAlign: 'middle',
              }}
            >
              {full}
            </Tag>
          </Tooltip>
        )
      },
    },
    {
      title: 'QoS',
      dataIndex: 'qosClass',
      width: 90,
      render: (t) => <Tag>{(t as string) || 'BestEffort'}</Tag>,
    },
    { title: 'Age', dataIndex: 'createdAt', width: 90, render: (_, r) => formatDate(r.createdAt) },
    {
      title: '操作',
      valueType: 'option',
      width: 190,
      fixed: 'right',
      render: (_, record) => (
        <Space
          size="small"
          style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}
        >
          <Tooltip title="查看日志">
            <a onClick={() => handleViewLogs(record)}>
              <FileTextOutlined />
            </a>
          </Tooltip>
          <Tooltip title="终端">
            <a onClick={() => handleTerminal(record)}>
              <DesktopOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看详情">
            <a onClick={() => setDetailDrawer({ open: true, pod: record })}>
              <InfoCircleOutlined />
            </a>
          </Tooltip>
          <Tooltip title="查看 YAML">
            <a onClick={() => handleViewYaml(record)}>
              <CodeOutlined />
            </a>
          </Tooltip>
          <Tooltip title="巡检诊断">
            <a onClick={() => handleInspection(record)}>
              <SafetyCertificateOutlined />
            </a>
          </Tooltip>
          <Popconfirm
            title="确定删除该 Pod？"
            onConfirm={() =>
              deleteMutation.mutate({ namespace: record.namespace, name: record.name })
            }
          >
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

  // 自动为数字/状态字段添加居中样式
  const columns = withCenterStyleBatch(rawColumns as Parameters<typeof withCenterStyleBatch>[0])

  return (
    <AppPage>
      <ProTable<Pod>
        headerTitle="Pod 列表"
        columns={columns}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey={(r) => `${r.namespace}/${r.name}`}
        search={false}
        pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 个 Pod` }}
        scroll={{ x: 1400 }}
        rowSelection={{
          selectedRowKeys: selectedKeys,
          onChange: (keys) => setSelectedKeys(keys as string[]),
        }}
        tableAlertRender={({ selectedRowKeys }) => (
          <Space>
            <Text>已选 {selectedRowKeys.length} 个 Pod</Text>
            <Button
              size="small"
              icon={<FileTextOutlined />}
              onClick={() => handleMultiPodLogs(selectedPods)}
            >
              聚合日志
            </Button>
            <Button
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => setBatchDeleteModal(true)}
            >
              批量删除
            </Button>
          </Space>
        )}
        toolBarRender={() => [
          <NamespaceSelector
            key="ns"
            clusterId={clusterId}
            value={namespace}
            onChange={setNamespace}
          />,
          <Button key="refresh" icon={<ReloadOutlined />} onClick={() => refetch()}>
            刷新
          </Button>,
        ]}
      />

      {/* ═══ 日志抽屉 ═══ */}
      <Drawer
        title={
          logDrawer.pods
            ? `聚合日志 - ${logDrawer.pods.length} 个 Pod`
            : `日志 - ${logDrawer.pod?.name}`
        }
        open={logDrawer.open}
        onClose={() => setLogDrawer({ open: false })}
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
          {logsLoading ? '加载中...' : logs || '暂无日志'}
        </pre>
      </Drawer>

      {/* ═══ 详情抽屉 ═══ */}
      <Drawer
        title={`Pod 详情 - ${detailDrawer.pod?.name}`}
        open={detailDrawer.open}
        onClose={() => setDetailDrawer({ open: false })}
        width={700}
      >
        {detailDrawer.pod && (
          <Tabs
            items={[
              {
                key: 'overview',
                label: '概览',
                children: (
                  <Descriptions bordered column={2} size="small">
                    <Descriptions.Item label="名称" span={2}>
                      {detailDrawer.pod.name}
                    </Descriptions.Item>
                    <Descriptions.Item label="命名空间">
                      <Tag>{detailDrawer.pod.namespace}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="状态">
                      <PodStatusTag
                        status={detailDrawer.pod.status}
                        ready={detailDrawer.pod.ready}
                      />
                    </Descriptions.Item>
                    <Descriptions.Item label="Pod IP">
                      {detailDrawer.pod.ip || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="节点">
                      {detailDrawer.pod.nodeName || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="重启次数">
                      {detailDrawer.pod.restarts}
                    </Descriptions.Item>
                    <Descriptions.Item label="QoS">
                      {detailDrawer.pod.qosClass || 'BestEffort'}
                    </Descriptions.Item>
                    <Descriptions.Item label="所属">
                      {detailDrawer.pod.ownerKind || 'ReplicaSet'}/
                      {detailDrawer.pod.ownerName || '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="创建时间" span={2}>
                      {formatDate(detailDrawer.pod.createdAt)}
                    </Descriptions.Item>
                    {detailDrawer.pod.labels && (
                      <Descriptions.Item label="标签" span={2}>
                        {Object.entries(detailDrawer.pod.labels).map(([k, v]) => (
                          <Tag key={k} color="blue">
                            {k}={v}
                          </Tag>
                        ))}
                      </Descriptions.Item>
                    )}
                    {detailDrawer.pod.annotations && (
                      <Descriptions.Item label="注解" span={2}>
                        {Object.entries(detailDrawer.pod.annotations)
                          .slice(0, 5)
                          .map(([k, v]) => (
                            <Tooltip key={k} title={v}>
                              <Tag style={{ marginBottom: 4 }}>{k}</Tag>
                            </Tooltip>
                          ))}
                      </Descriptions.Item>
                    )}
                  </Descriptions>
                ),
              },
              {
                key: 'containers',
                label: '容器',
                children: detailDrawer.pod.containers ? (
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead>
                      <tr style={{ background: '#fafafa' }}>
                        {['名称', '镜像', '就绪', '重启', '状态'].map((h) => (
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
                      {detailDrawer.pod.containers.map((c: any, i: number) => (
                        <tr key={i}>
                          <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                            <Tag>{c.name}</Tag>
                          </td>
                          <td
                            style={{
                              padding: '8px 12px',
                              borderBottom: '1px solid #f0f0f0',
                              fontSize: 11,
                            }}
                          >
                            {c.image}
                          </td>
                          <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                            {c.ready ? (
                              <Badge status="success" text="是" />
                            ) : (
                              <Badge status="error" text="否" />
                            )}
                          </td>
                          <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                            {c.restartCount ?? 0}
                          </td>
                          <td style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
                            {c.state || '-'}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                ) : (
                  <Text type="secondary">暂无容器数据</Text>
                ),
              },
              {
                key: 'conditions',
                label: '条件',
                children: detailDrawer.pod.conditions ? (
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead>
                      <tr style={{ background: '#fafafa' }}>
                        {['类型', '状态', '原因', '信息'].map((h) => (
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
                      {detailDrawer.pod.conditions.map((c: any, i: number) => (
                        <tr key={i}>
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
                        </tr>
                      ))}
                    </tbody>
                  </table>
                ) : (
                  <Text type="secondary">暂无条件数据</Text>
                ),
              },
            ]}
          />
        )}
      </Drawer>

      {/* ═══ YAML 抽屉 ═══ */}
      <Drawer
        title={`YAML - ${yamlDrawer.pod?.name}`}
        open={yamlDrawer.open}
        onClose={() => setYamlDrawer({ open: false, loading: false })}
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
          {yamlDrawer.loading ? '加载中...' : yamlDrawer.yaml || '暂无数据'}
        </pre>
      </Drawer>

      {/* ═══ 巡检诊断抽屉 ═══ */}
      <Drawer
        title={`巡检诊断 - ${inspectDrawer.pod?.name}`}
        open={inspectDrawer.open}
        onClose={() => setInspectDrawer({ open: false, loading: false })}
        width={800}
      >
        <pre
          style={{
            background: '#1e1e1e',
            color: '#d4d4d4',
            padding: 16,
            borderRadius: 4,
            fontSize: 13,
            lineHeight: 1.6,
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-all',
          }}
        >
          {inspectDrawer.loading ? '巡检中...' : inspectDrawer.text || '暂无巡检数据'}
        </pre>
      </Drawer>

      {/* ═══ 批量删除确认 ═══ */}
      <Modal
        title="批量删除 Pod"
        open={batchDeleteModal}
        onCancel={() => setBatchDeleteModal(false)}
        onOk={() => batchDeleteMutation.mutate(selectedPods)}
        confirmLoading={batchDeleteMutation.isPending}
        okText="确认删除"
        okButtonProps={{ danger: true }}
      >
        <Alert
          type="warning"
          showIcon
          message={`确定删除选中的 ${selectedKeys.length} 个 Pod？`}
          style={{ marginBottom: 12 }}
        />
        <Descriptions column={1} size="small">
          <Descriptions.Item label="强制删除">
            <Switch checked={forceDelete} onChange={setForceDelete} />
          </Descriptions.Item>
        </Descriptions>
        <div style={{ marginTop: 12 }}>
          {selectedPods.map((p) => (
            <Tag key={`${p.namespace}/${p.name}`} color="red" style={{ marginBottom: 4 }}>
              {p.namespace}/{p.name}
            </Tag>
          ))}
        </div>
      </Modal>
    </AppPage>
  )
}

export default PodsPage
