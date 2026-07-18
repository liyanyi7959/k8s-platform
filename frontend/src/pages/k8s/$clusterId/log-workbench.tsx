import React, { useEffect, useMemo, useState } from 'react'
import { Alert, Badge, Button, Card, Checkbox, Col, Empty, Input, Row, Select, Space, Statistic, Switch, Tag, Tooltip, Typography, message } from 'antd'
import { ClearOutlined, DownloadOutlined, PauseCircleOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { AppPage, NamespaceSelector } from '@/components'
import { useClusterId } from '@/hooks/useClusterId'
import { getPodLogs, listPods } from '@/services/k8s'
import type { Pod } from '@/types'

const { Text, Title } = Typography
const MAX_SELECTED_PODS = 10

type PodLogResult = { key: string; pod: Pod; logs: string; error?: string }

const LogWorkbenchPage: React.FC = () => {
  const clusterId = useClusterId()
  const [namespace, setNamespace] = useState('')
  const [selectedPodKeys, setSelectedPodKeys] = useState<string[]>([])
  const [container, setContainer] = useState<string>()
  const [tailLines, setTailLines] = useState(500)
  const [previous, setPrevious] = useState(false)
  const [timestamps, setTimestamps] = useState(true)
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [refreshSeconds, setRefreshSeconds] = useState(5)
  const [keyword, setKeyword] = useState('')
  const [results, setResults] = useState<PodLogResult[]>([])
  const [loadingLogs, setLoadingLogs] = useState(false)
  const [lastUpdated, setLastUpdated] = useState<Date>()

  const podsQuery = useQuery({
    queryKey: ['log-workbench-pods', clusterId, namespace],
    queryFn: ({ signal }) => listPods(clusterId, { namespace: namespace || undefined, pageSize: 500 }, signal),
    enabled: !!clusterId,
    staleTime: 30_000,
  })

  const pods = podsQuery.data?.items || []
  const podMap = useMemo(() => new Map(pods.map((pod) => [`${pod.namespace}/${pod.name}`, pod])), [pods])
  const selectedPods = selectedPodKeys.map((key) => podMap.get(key)).filter(Boolean) as Pod[]
  const containerOptions = useMemo(() => {
    if (selectedPods.length !== 1) return []
    const pod = selectedPods[0]!
    return [...(pod.initContainers || []).map((item) => ({ label: `init/${item.name}`, value: item.name })), ...(pod.containers || []).map((item) => ({ label: item.name, value: item.name }))]
  }, [selectedPods])

  const loadLogs = async (silent = false) => {
    if (!selectedPods.length) {
      if (!silent) message.warning('请先选择至少一个 Pod')
      return
    }
    if (!silent) setLoadingLogs(true)
    try {
      const settled = await Promise.allSettled(selectedPods.map(async (pod) => {
        const response = await getPodLogs(clusterId, pod.namespace, pod.name, {
          tailLines,
          container: selectedPods.length === 1 ? container : undefined,
          previous,
          timestamps,
        })
        return { key: `${pod.namespace}/${pod.name}`, pod, logs: response.logs || '' }
      }))
      setResults(settled.map((item, index) => item.status === 'fulfilled' ? item.value : {
        key: selectedPodKeys[index]!,
        pod: selectedPods[index]!,
        logs: '',
        error: item.reason?.message || '获取日志失败',
      }))
      setLastUpdated(new Date())
    } finally {
      if (!silent) setLoadingLogs(false)
    }
  }

  useEffect(() => {
    if (!autoRefresh || !selectedPods.length) return
    const timer = window.setInterval(() => loadLogs(true), refreshSeconds * 1000)
    return () => window.clearInterval(timer)
  }, [autoRefresh, refreshSeconds, selectedPodKeys.join(','), container, previous, timestamps, tailLines])

  const filteredResults = useMemo(() => results.map((result) => ({
    ...result,
    lines: result.logs.split(/\r?\n/).filter((line) => !keyword || line.toLowerCase().includes(keyword.toLowerCase())),
  })), [results, keyword])
  const visibleLineCount = filteredResults.reduce((total, result) => total + result.lines.length, 0)

  const downloadLogs = () => {
    const content = results.map((result) => `===== ${result.key} =====\n${result.error || result.logs}`).join('\n\n')
    const link = document.createElement('a')
    link.href = URL.createObjectURL(new Blob([content], { type: 'text/plain;charset=utf-8' }))
    link.download = `k8s-logs-${new Date().toISOString().replaceAll(':', '-')}.log`
    link.click()
    URL.revokeObjectURL(link.href)
  }

  return (
    <AppPage>
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Card size="small">
          <Row gutter={[16, 12]} align="middle">
            <Col flex="auto">
              <Title level={4} style={{ margin: 0 }}>日志工作台</Title>
              <Text type="secondary">面向故障排查的多 Pod 聚合日志检索，单次最多选择 {MAX_SELECTED_PODS} 个 Pod</Text>
            </Col>
            <Col><Statistic title="已选 Pod" value={selectedPods.length} suffix={`/ ${MAX_SELECTED_PODS}`} valueStyle={{ fontSize: 20 }} /></Col>
            <Col><Statistic title="可见日志行" value={visibleLineCount} valueStyle={{ fontSize: 20 }} /></Col>
          </Row>
          <Row gutter={[10, 10]} style={{ marginTop: 16 }}>
            <Col xs={24} md={6} xl={4}>
              <NamespaceSelector clusterId={clusterId} value={namespace} onChange={(value) => { setNamespace(value); setSelectedPodKeys([]); setResults([]) }} style={{ width: '100%' }} />
            </Col>
            <Col xs={24} md={18} xl={10}>
              <Select mode="multiple" allowClear showSearch placeholder="选择 Pod（支持名称搜索）" loading={podsQuery.isLoading} style={{ width: '100%' }} value={selectedPodKeys} maxTagCount="responsive" onChange={(values) => {
                if (values.length > MAX_SELECTED_PODS) {
                  message.warning(`最多选择 ${MAX_SELECTED_PODS} 个 Pod`)
                  return
                }
                setSelectedPodKeys(values); setContainer(undefined); setResults([])
              }} options={pods.map((pod) => ({ label: `${pod.namespace}/${pod.name}`, value: `${pod.namespace}/${pod.name}`, title: pod.status }))} optionRender={(option) => <Space><Badge status={option.data.title === 'Running' ? 'success' : 'warning'} /><span>{option.label}</span><Tag>{option.data.title}</Tag></Space>} />
            </Col>
            <Col xs={12} md={6} xl={3}>
              <Select allowClear disabled={selectedPods.length !== 1} placeholder={selectedPods.length > 1 ? '多 Pod 使用默认容器' : '选择容器'} value={container} onChange={setContainer} options={containerOptions} style={{ width: '100%' }} />
            </Col>
            <Col xs={12} md={6} xl={3}>
              <Select value={tailLines} onChange={setTailLines} style={{ width: '100%' }} options={[100, 200, 500, 1000, 2000].map((value) => ({ label: `最近 ${value} 行`, value }))} />
            </Col>
            <Col xs={24} md={12} xl={4}>
              <Button block type="primary" icon={<ReloadOutlined />} loading={loadingLogs} onClick={() => loadLogs()}>查询日志</Button>
            </Col>
          </Row>
          <Row justify="space-between" align="middle" gutter={[12, 10]} style={{ marginTop: 12 }}>
            <Col><Space wrap><Checkbox checked={previous} onChange={(event) => setPrevious(event.target.checked)}>前一实例</Checkbox><Checkbox checked={timestamps} onChange={(event) => setTimestamps(event.target.checked)}>显示时间戳</Checkbox><Space><Switch size="small" checked={autoRefresh} onChange={setAutoRefresh} />自动刷新<Select size="small" disabled={!autoRefresh} value={refreshSeconds} onChange={setRefreshSeconds} style={{ width: 70 }} options={[3, 5, 10, 30].map((value) => ({ label: `${value}s`, value }))} /></Space></Space></Col>
            <Col><Space><Input allowClear prefix={<SearchOutlined />} placeholder="过滤已加载日志" value={keyword} onChange={(event) => setKeyword(event.target.value)} /><Tooltip title="下载原始聚合日志"><Button icon={<DownloadOutlined />} disabled={!results.length} onClick={downloadLogs} /></Tooltip><Tooltip title="清空"><Button icon={<ClearOutlined />} disabled={!results.length} onClick={() => setResults([])} /></Tooltip></Space></Col>
          </Row>
        </Card>

        <Card size="small" title={<Space>日志输出 {autoRefresh && <Tag icon={<PauseCircleOutlined />} color="processing">每 {refreshSeconds}s 刷新</Tag>}</Space>} extra={lastUpdated && <Text type="secondary">最后更新 {lastUpdated.toLocaleTimeString()}</Text>} styles={{ body: { padding: 0 } }}>
          {!results.length ? <Empty description="选择 Pod 后查询日志" style={{ padding: 80 }} /> : (
            <div style={{ background: '#0b1220', color: '#d1d5db', minHeight: 480, maxHeight: '64vh', overflow: 'auto', padding: 16, fontFamily: 'Consolas, Monaco, monospace', fontSize: 12, lineHeight: 1.65 }}>
              {filteredResults.map((result) => <section key={result.key} style={{ marginBottom: 20 }}>
                <div style={{ position: 'sticky', top: -16, zIndex: 1, margin: '0 -16px 8px', padding: '7px 16px', background: '#162033', color: '#93c5fd', fontWeight: 700 }}>{result.key} <Tag color={result.error ? 'error' : 'success'}>{result.error ? '失败' : `${result.lines.length} 行`}</Tag></div>
                {result.error ? <Alert type="error" message={result.error} /> : result.lines.length ? result.lines.map((line, index) => <div key={index} style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all', color: keyword ? '#fde68a' : undefined }}><span style={{ display: 'inline-block', width: 54, color: '#475569', userSelect: 'none' }}>{index + 1}</span>{line}</div>) : <Text style={{ color: '#64748b' }}>无匹配日志</Text>}
              </section>)}
            </div>
          )}
        </Card>
      </Space>
    </AppPage>
  )
}

export default LogWorkbenchPage
