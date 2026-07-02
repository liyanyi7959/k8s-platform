import React, { useMemo, useState } from 'react'
import { Button, Card, Input, Select, Space, Typography, message } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { AppPage, NamespaceSelector } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { getPodLogs, listPods } from '@/services/k8s'
import type { Pod } from '@/types'

const { Text, Title } = Typography
const { TextArea } = Input

const LogWorkbenchPage: React.FC = () => {
  const clusterId = useClusterId()
  const [namespace, setNamespace] = useState<string>('')
  const [selectedPodKeys, setSelectedPodKeys] = useState<string[]>([])
  const [tailLines, setTailLines] = useState<number>(200)
  const [logs, setLogs] = useState<string>('')
  const [loadingLogs, setLoadingLogs] = useState(false)

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['log-workbench-pods', clusterId, namespace],
    queryFn: () => listPods(clusterId, { namespace: namespace || undefined, pageSize: 200 }),
    enabled: !!clusterId,
  })

  const pods = data?.items || []

  const podOptions = useMemo(
    () =>
      pods.map((pod) => ({
        label: `${pod.namespace}/${pod.name} · ${pod.status}`,
        value: `${pod.namespace}/${pod.name}`,
      })),
    [pods],
  )

  const selectedPods = useMemo(() => {
    const selected = new Set(selectedPodKeys)
    return pods.filter((pod) => selected.has(`${pod.namespace}/${pod.name}`))
  }, [pods, selectedPodKeys])

  const handleLoadLogs = async () => {
    if (selectedPods.length === 0) {
      message.warning('请先选择至少一个 Pod')
      return
    }

    setLoadingLogs(true)
    try {
      const results = await Promise.all(
        selectedPods.map(async (pod: Pod) => {
          try {
            const res = await getPodLogs(clusterId, pod.namespace, pod.name, tailLines)
            return `===== ${pod.namespace}/${pod.name} =====\n${res.logs || ''}`
          } catch (error) {
            const errMsg = error instanceof Error ? error.message : '获取日志失败'
            return `===== ${pod.namespace}/${pod.name} =====\n${errMsg}`
          }
        }),
      )
      setLogs(results.join('\n\n'))
    } finally {
      setLoadingLogs(false)
    }
  }

  return (
    <AppPage>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Card>
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Title level={4} style={{ margin: 0 }}>日志工作台</Title>
            <Space wrap>
              <NamespaceSelector
                clusterId={clusterId}
                value={namespace}
                onChange={(value) => {
                  setNamespace(value)
                  setSelectedPodKeys([])
                  setLogs('')
                }}
                style={{ width: 220 }}
              />
              <Select
                mode="multiple"
                allowClear
                showSearch
                placeholder="选择一个或多个 Pod"
                loading={isLoading}
                style={{ minWidth: 420 }}
                value={selectedPodKeys}
                onChange={setSelectedPodKeys}
                options={podOptions}
                maxTagCount="responsive"
              />
              <Select
                value={tailLines}
                style={{ width: 120 }}
                onChange={setTailLines}
                options={[
                  { label: '最近 100 行', value: 100 },
                  { label: '最近 200 行', value: 200 },
                  { label: '最近 500 行', value: 500 },
                ]}
              />
              <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
                刷新 Pod
              </Button>
              <Button type="primary" loading={loadingLogs} onClick={handleLoadLogs}>
                拉取日志
              </Button>
            </Space>
            <Text type="secondary">
              这里先实现了旧版多 Pod 日志工作台的核心能力：按命名空间选择多个 Pod，聚合查看最近日志。
            </Text>
          </Space>
        </Card>

        <Card title={`日志输出${selectedPods.length > 0 ? `（${selectedPods.length} 个 Pod）` : ''}`}>
          <TextArea
            value={logs}
            readOnly
            rows={24}
            placeholder="选择 Pod 并点击“拉取日志”后，这里会显示聚合日志内容。"
            style={{ fontFamily: 'Consolas, Monaco, monospace' }}
          />
        </Card>
      </Space>
    </AppPage>
  )
}

export default LogWorkbenchPage