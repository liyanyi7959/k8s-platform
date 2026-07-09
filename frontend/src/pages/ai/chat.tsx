import React, { useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Collapse,
  Empty,
  Input,
  List,
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
  Upload,
  message as antdMessage,
  type UploadFile,
} from 'antd'
import {
  ClearOutlined,
  ClusterOutlined,
  CopyOutlined,
  DeleteOutlined,
  FileSearchOutlined,
  FileTextOutlined,
  MessageOutlined,
  PaperClipOutlined,
  PlusOutlined,
  ReloadOutlined,
  RobotOutlined,
  SafetyOutlined,
  SearchOutlined,
  SendOutlined,
  StopOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'
import { history, useSearchParams } from '@umijs/max'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage, MarkdownContent } from '@/components'
import { useAIChat } from '@/hooks'
import { confirmActionProposal, createConversation, getConversation, listAIModels, listConversations } from '@/services/ai'
import { listClusters } from '@/services/clusters'
import type { AIActionProposal, AIMessageAttachment, AIAssistantMode, AIModel } from '@/types'
import { formatDate } from '@/utils'

const { Paragraph, Text } = Typography
const { TextArea } = Input

const resourceKindOptions = [
  'Pod',
  'Deployment',
  'StatefulSet',
  'DaemonSet',
  'Service',
  'Ingress',
  'ConfigMap',
  'Secret',
  'Job',
  'CronJob',
  'Node',
]

function buildConversationDraftTitle(content: string) {
  const title = content.trim()
  if (!title) {
    return '新的 AI 会话'
  }
  const runes = Array.from(title)
  if (runes.length > 24) {
    return `${runes.slice(0, 24).join('')}...`
  }
  return title
}

function classifyFiles(files: UploadFile[]) {
  const nativeFiles: File[] = []
  files.forEach((file) => {
    if (file.originFileObj instanceof File) {
      nativeFiles.push(file.originFileObj)
    }
  })
  const imageFiles = nativeFiles.filter((file) => file.type.startsWith('image/'))
  const textFiles = nativeFiles.filter((file) => !file.type.startsWith('image/'))
  return { nativeFiles, imageFiles, textFiles }
}

function getAttachmentHeaders() {
  const headers = new Headers()
  const token = localStorage.getItem('token')
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  return headers
}

async function fetchAttachmentBlob(downloadUrl: string): Promise<Blob> {
  const response = await fetch(downloadUrl, {
    method: 'GET',
    credentials: 'include',
    headers: getAttachmentHeaders(),
  })
  if (!response.ok) {
    throw new Error('附件加载失败')
  }
  return response.blob()
}

const AttachmentItem: React.FC<{ attachment: AIMessageAttachment }> = ({ attachment }) => {
  const [previewUrl, setPreviewUrl] = useState('')

  useEffect(() => {
    if (attachment.fileKind !== 'image' || !attachment.downloadUrl) {
      return
    }

    let objectUrl = ''
    let disposed = false
    void fetchAttachmentBlob(attachment.downloadUrl)
      .then((blob) => {
        if (disposed) {
          return
        }
        objectUrl = URL.createObjectURL(blob)
        setPreviewUrl(objectUrl)
      })
      .catch(() => {
        setPreviewUrl('')
      })

    return () => {
      disposed = true
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl)
      }
    }
  }, [attachment.downloadUrl, attachment.fileKind])

  const handleDownload = async () => {
    if (!attachment.downloadUrl) {
      return
    }
    try {
      const blob = await fetchAttachmentBlob(attachment.downloadUrl)
      const objectUrl = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = objectUrl
      anchor.download = attachment.originalName || `attachment-${attachment.id}`
      document.body.appendChild(anchor)
      anchor.click()
      anchor.remove()
      URL.revokeObjectURL(objectUrl)
    } catch (error) {
      antdMessage.error(error instanceof Error ? error.message : '附件下载失败')
    }
  }

  if (attachment.fileKind === 'image' && previewUrl) {
    return (
      <button
        key={attachment.id}
        type="button"
        onClick={() => void handleDownload()}
        style={{ border: 'none', background: 'transparent', padding: 0, cursor: 'pointer' }}
      >
        <img
          src={previewUrl}
          alt={attachment.originalName}
          style={{
            width: 96,
            height: 96,
            objectFit: 'cover',
            borderRadius: 8,
            border: '1px solid #f0f0f0',
          }}
        />
      </button>
    )
  }

  return (
    <Tag key={attachment.id} style={{ cursor: attachment.downloadUrl ? 'pointer' : 'default' }} onClick={() => void handleDownload()}>
      {attachment.originalName || `attachment-${attachment.id}`}
    </Tag>
  )
}

const AIChatPage: React.FC = () => {
  const [searchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const searchConversationId = Number(searchParams.get('id') || 0) || undefined

  const [activeConversationId, setActiveConversationId] = useState<number | undefined>(searchConversationId)
  const [inputValue, setInputValue] = useState('')
  const [selectedClusterId, setSelectedClusterId] = useState<number | undefined>(undefined)
  const [assistantMode, setAssistantMode] = useState<AIAssistantMode>('diagnose')
  const [selectedModelId, setSelectedModelId] = useState<number | undefined>(undefined)
  const [namespace, setNamespace] = useState('')
  const [resourceKind, setResourceKind] = useState('')
  const [resourceName, setResourceName] = useState('')
  const [pendingFiles, setPendingFiles] = useState<UploadFile[]>([])
  const messageViewportRef = useRef<HTMLDivElement | null>(null)

  // 变更提案确认状态
  const [confirmingId, setConfirmingId] = useState<number | null>(null)
  const [confirmText, setConfirmText] = useState('')
  const [confirmRisk, setConfirmRisk] = useState(false)
  const [confirmLoading, setConfirmLoading] = useState(false)

  // 会话历史搜索
  const [searchKeyword, setSearchKeyword] = useState('')

  const {
    messages,
    toolCalls,
    actionProposals,
    isStreaming,
    isResponding,
    activeModelName,
    activeProviderName,
    hydrateConversation,
    clearConversation,
    stopGeneration,
    sendMessage,
  } = useAIChat()

  useEffect(() => {
    setActiveConversationId(searchConversationId)
    if (!searchConversationId) {
      clearConversation()
    }
  }, [clearConversation, searchConversationId])

  const { data: clustersData } = useQuery({
    queryKey: ['clusters-for-ai'],
    queryFn: ({ signal }) => listClusters({ page: 1, pageSize: 200 }, signal),
  })
  const { data: models = [] } = useQuery({
    queryKey: ['ai-models'],
    queryFn: ({ signal }) => listAIModels(signal),
  })
  const { data: conversationsData } = useQuery({
    queryKey: ['ai-conversations'],
    queryFn: ({ signal }) => listConversations({ page: 1, pageSize: 100 }, signal),
  })
  const { data: conversationDetail, isFetching: isConversationLoading } = useQuery({
    queryKey: ['ai-conversation', activeConversationId],
    queryFn: ({ signal }) => getConversation(activeConversationId!, signal),
    enabled: Boolean(activeConversationId),
  })

  useEffect(() => {
    if (!conversationDetail || isStreaming) {
      return
    }
    hydrateConversation(conversationDetail)
    setSelectedClusterId(conversationDetail.clusterId)
    setAssistantMode((conversationDetail.assistantMode as AIAssistantMode) || 'diagnose')
    setSelectedModelId(conversationDetail.modelId)
  }, [conversationDetail, hydrateConversation])

  useEffect(() => {
    const viewport = messageViewportRef.current
    if (!viewport) {
      return
    }
    viewport.scrollTop = viewport.scrollHeight
  }, [actionProposals.length, messages, toolCalls.length])

  const clusters = clustersData?.items || []
  const conversations = conversationsData?.items || []
  // 按搜索关键词过滤会话列表
  const filteredConversations = conversations.filter(c => {
    if (!searchKeyword) return true
    const kw = searchKeyword.toLowerCase()
    return c.title?.toLowerCase().includes(kw) || c.summary?.toLowerCase().includes(kw)
  })
  const selectedModel = useMemo(
    () => models.find((item) => item.id === selectedModelId),
    [models, selectedModelId],
  )
  const { nativeFiles, imageFiles, textFiles } = useMemo(
    () => classifyFiles(pendingFiles),
    [pendingFiles],
  )

  const modelCapabilityWarning = useMemo(() => {
    if (!selectedModel) {
      return ''
    }
    if (imageFiles.length > 0 && !selectedModel.supportsVision) {
      return '当前模型不支持图片输入，请切换到支持视觉的模型或移除图片。'
    }
    if (textFiles.length > 0 && !selectedModel.supportsFileInput) {
      return '当前模型不支持文件输入，请切换到支持文件输入的模型或移除文本附件。'
    }
    return ''
  }, [imageFiles.length, selectedModel, textFiles.length])

  const streamEnabled = !selectedModel || selectedModel.supportsStreaming
  const clusterLocked = Boolean(activeConversationId)
  const isBusy = isStreaming || isResponding

  const handleNewChat = () => {
    clearConversation()
    setActiveConversationId(undefined)
    setInputValue('')
    setPendingFiles([])
    setNamespace('')
    setResourceKind('')
    setResourceName('')
    history.push('/ai/chat')
  }

  const handleConversationClick = (id: number) => {
    setActiveConversationId(id)
    history.push(`/ai/chat?id=${id}`)
  }

  const handleSend = async () => {
    if (!selectedClusterId) {
      antdMessage.warning('请先选择目标集群')
      return
    }
    if (!inputValue.trim()) {
      return
    }
    if (modelCapabilityWarning) {
      antdMessage.warning(modelCapabilityWarning)
      return
    }

    const messageText = inputValue.trim()
    const filesToSend = nativeFiles
    const fileListToRestore = pendingFiles
    setInputValue('')
    setPendingFiles([])
    try {
      let targetConversationId = activeConversationId

      if (!targetConversationId) {
        const created = await createConversation({
          clusterId: selectedClusterId,
          title: buildConversationDraftTitle(messageText),
          assistantMode,
          modelId: selectedModelId,
        })
        if (!created.id) {
          throw new Error('会话创建失败')
        }
        targetConversationId = created.id
        setActiveConversationId(targetConversationId)
        history.replace(`/ai/chat?id=${targetConversationId}`)
        void queryClient.invalidateQueries({ queryKey: ['ai-conversations'] })
      }

      const resultConversationId = await sendMessage(
        selectedClusterId,
        {
          conversationId: targetConversationId,
          message: messageText,
          assistantMode,
          modelId: selectedModelId,
          preferModel: selectedModel?.modelCode,
          namespace: namespace || undefined,
          resourceKind: resourceKind || undefined,
          resourceName: resourceName || undefined,
          files: filesToSend,
        },
        streamEnabled,
      )

      const finalConversationId = resultConversationId || targetConversationId
      if (finalConversationId) {
        setActiveConversationId(finalConversationId)
        history.replace(`/ai/chat?id=${finalConversationId}`)
        void Promise.all([
          queryClient.invalidateQueries({ queryKey: ['ai-conversations'] }),
          queryClient.invalidateQueries({ queryKey: ['ai-conversation', finalConversationId] }),
        ])
      }
    } catch (error) {
      setInputValue(messageText)
      setPendingFiles(fileListToRestore)
      antdMessage.error(error instanceof Error ? error.message : '会话发送失败')
    }
  }

  // 确认执行变更提案
  const handleConfirm = async (proposal: AIActionProposal) => {
    if (confirmText !== proposal.requiredConfirmationText) {
      antdMessage.error(`请输入正确的确认短语：${proposal.requiredConfirmationText}`)
      return
    }
    if (!confirmRisk) {
      antdMessage.warning('请勾选风险确认')
      return
    }
    setConfirmLoading(true)
    try {
      await confirmActionProposal(proposal.clusterId, proposal.id, {
        confirmation_text: confirmText,
        confirm_risk: confirmRisk,
      })
      antdMessage.success('提案已确认执行')
      setConfirmingId(null)
      setConfirmText('')
      setConfirmRisk(false)
      // 刷新会话
      queryClient.invalidateQueries({ queryKey: ['ai-conversation', activeConversationId] })
    } catch (err: any) {
      antdMessage.error(err?.message || '确认失败')
    } finally {
      setConfirmLoading(false)
    }
  }

  return (
    <AppPage>
      <div style={{ display: 'grid', gridTemplateColumns: '280px minmax(0, 1fr)', gap: 16, minHeight: 'calc(100vh - 220px)' }}>
        <Card
          size="small"
          title="会话历史"
          extra={<Button type="text" size="small" icon={<PlusOutlined />} onClick={handleNewChat}>新建</Button>}
          bodyStyle={{ padding: 8 }}
        >
          <Input
            placeholder="搜索会话..."
            prefix={<SearchOutlined />}
            allowClear
            size="small"
            style={{ marginBottom: 8 }}
            value={searchKeyword}
            onChange={(e) => setSearchKeyword(e.target.value)}
          />
          <List
            dataSource={filteredConversations}
            locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无会话" /> }}
            renderItem={(item) => (
              <List.Item
                style={{
                  cursor: 'pointer',
                  display: 'block',
                  borderRadius: 10,
                  padding: 10,
                  marginBottom: 8,
                  background: item.id === activeConversationId ? '#f0f7ff' : '#fff',
                  border: item.id === activeConversationId ? '1px solid #91caff' : '1px solid #f0f0f0',
                }}
                onClick={() => handleConversationClick(item.id)}
              >
                <Space direction="vertical" size={4} style={{ width: '100%' }}>
                  <Text strong ellipsis>{item.title}</Text>
                  <Space size={[4, 4]} wrap>
                    <Tag color={item.assistantMode === 'chat' ? 'blue' : 'gold'}>
                      {item.assistantMode === 'chat' ? '通用对话' : '故障诊断'}
                    </Tag>
                    <Tag>{item.messageCount} 条</Tag>
                  </Space>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {item.summary || '暂无摘要'}
                  </Text>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {formatDate(item.updatedAt)}
                  </Text>
                </Space>
              </List.Item>
            )}
          />
        </Card>

        <Card
          style={{ display: 'flex', flexDirection: 'column' }}
          bodyStyle={{ display: 'flex', flexDirection: 'column', gap: 12, minHeight: 0, flex: 1 }}
          title={
            <Space wrap>
              <RobotOutlined />
              <Text strong>对话式运维</Text>
              {activeProviderName ? <Tag>{activeProviderName}</Tag> : null}
              {activeModelName ? <Tag color="blue">{activeModelName}</Tag> : null}
              {!streamEnabled ? <Tag color="orange">非流式</Tag> : null}
            </Space>
          }
          extra={
            <Space>
              <Button
                icon={<ClearOutlined />}
                onClick={handleNewChat}
                disabled={isBusy}
              >
                清空当前
              </Button>
            </Space>
          }
        >
          <Space wrap size={12}>
            <Select
              style={{ width: 240 }}
              placeholder="选择集群"
              value={selectedClusterId}
              onChange={setSelectedClusterId}
              disabled={clusterLocked || isBusy}
              options={clusters.map((cluster) => ({ value: cluster.id, label: cluster.name }))}
              suffixIcon={<ClusterOutlined />}
            />
            <Select
              style={{ width: 160 }}
              value={assistantMode}
              onChange={(value) => setAssistantMode(value as AIAssistantMode)}
              disabled={clusterLocked || isBusy}
              options={[
                { value: 'diagnose', label: '故障诊断' },
                { value: 'chat', label: '通用对话' },
              ]}
            />
            <Select
              allowClear
              style={{ width: 360 }}
              placeholder="自动路由模型"
              value={selectedModelId}
              onChange={setSelectedModelId}
              disabled={isBusy}
              options={models
                .filter((item) => item.enabled)
                .map((item: AIModel) => ({
                  value: item.id,
                  label: `${item.providerName} / ${item.name} (${item.modelCode})`,
                }))}
            />
            {selectedModel ? (
              <Space size={[4, 4]} wrap>
                {selectedModel.supportsVision ? <Tag color="purple">视觉</Tag> : null}
                {selectedModel.supportsFileInput ? <Tag color="cyan">文件</Tag> : null}
                {selectedModel.supportsStreaming ? <Tag color="blue">流式</Tag> : null}
                {selectedModel.supportsTools ? <Tag color="green">工具</Tag> : null}
              </Space>
            ) : (
              <Tag>自动按路由配置选择模型</Tag>
            )}
          </Space>

          <Collapse
            size="small"
            items={[
              {
                key: 'scope',
                label: '高级上下文',
                children: (
                  <Space wrap style={{ width: '100%' }}>
                    <Input
                      style={{ width: 180 }}
                      placeholder="命名空间，可选"
                      value={namespace}
                      onChange={(event) => setNamespace(event.target.value)}
                    />
                    <Select
                      allowClear
                      style={{ width: 180 }}
                      placeholder="资源类型，可选"
                      value={resourceKind || undefined}
                      onChange={(value) => setResourceKind(value || '')}
                      options={resourceKindOptions.map((value) => ({ value, label: value }))}
                    />
                    <Input
                      style={{ width: 220 }}
                      placeholder="资源名称，可选"
                      value={resourceName}
                      onChange={(event) => setResourceName(event.target.value)}
                    />
                  </Space>
                ),
              },
            ]}
          />

          <Collapse ghost size="small" style={{ marginBottom: 8 }}>
            <Collapse.Panel header={<Text type="secondary" style={{ fontSize: 12 }}><SafetyOutlined /> AI 运维安全规则</Text>} key="safety">
              <ul style={{ fontSize: 12, color: '#8c8c8c', margin: 0, paddingLeft: 16 }}>
                <li>所有写操作（创建/修改/删除）必须经人工确认后才能执行</li>
                <li>高危操作（删除资源/驱逐节点）需要两位管理员双重确认</li>
                <li>AI 不会自动执行任何变更操作，仅提供提案和建议</li>
                <li>Secret 等敏感信息已脱敏处理</li>
              </ul>
            </Collapse.Panel>
          </Collapse>

          {clusterLocked ? (
            <Alert
              type="info"
              showIcon
              message="当前历史会话已绑定集群和模式。如需切换集群，请新建会话。"
            />
          ) : null}

          {modelCapabilityWarning ? (
            <Alert type="warning" showIcon message={modelCapabilityWarning} />
          ) : null}

          <div ref={messageViewportRef} style={{ flex: 1, minHeight: 320, overflow: 'auto', paddingRight: 4 }}>
            {messages.length === 0 ? (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="输入运维问题后，助手会结合集群上下文和模型能力给出回答。"
              />
            ) : (
              <List
                loading={isConversationLoading && messages.length === 0}
                dataSource={messages}
                renderItem={(item) => (
                  <List.Item key={item.localId} style={{ border: 'none', padding: '10px 0' }}>
                    <div
                      style={{
                        width: '100%',
                        display: 'flex',
                        justifyContent: item.role === 'user' ? 'flex-end' : 'flex-start',
                      }}
                    >
                      <div
                        style={{
                          maxWidth: '84%',
                          background: item.role === 'user' ? '#1677ff' : '#fafafa',
                          color: item.role === 'user' ? '#fff' : '#141414',
                          borderRadius: 14,
                          padding: '12px 14px',
                          border: item.role === 'user' ? 'none' : '1px solid #f0f0f0',
                        }}
                      >
                        <Space size={[6, 6]} wrap style={{ marginBottom: 8 }}>
                          <Tag color={item.role === 'user' ? 'geekblue' : 'default'}>
                            {item.role === 'user' ? '用户' : '助手'}
                          </Tag>
                          <Text style={{ color: item.role === 'user' ? '#dbeafe' : '#8c8c8c', fontSize: 12 }}>
                            {formatDate(item.createdAt)}
                          </Text>
                        </Space>
                        <MarkdownContent
                          content={item.content || (item.isStreaming || item.isPending ? '正在结合集群证据生成回答...' : '')}
                          tone={item.role === 'user' ? 'dark' : 'light'}
                        />
                        <Paragraph style={{ display: 'none' }}>
                          {item.content || (item.isStreaming ? '正在生成回答...' : '')}
                        </Paragraph>
                        {item.attachments?.length ? (
                          <Space size={[8, 8]} wrap style={{ marginTop: 10 }}>
                            {item.attachments.map((attachment) => (
                              <AttachmentItem key={attachment.id} attachment={attachment} />
                            ))}
                          </Space>
                        ) : null}
                        {item.role === 'assistant' && item.content ? (
                          <div style={{ display: 'flex', gap: 4, marginTop: 4 }}>
                            <Tooltip title="复制内容">
                              <Button type="text" size="small" icon={<CopyOutlined />} onClick={() => {
                                navigator.clipboard.writeText(item.content)
                                antdMessage.success('已复制')
                              }} />
                            </Tooltip>
                            <Tooltip title="重新生成回答">
                              <Button
                                type="text"
                                size="small"
                                icon={<ReloadOutlined />}
                                disabled={isBusy}
                                onClick={() => {
                                  // 找到上一条用户消息
                                  const idx = messages.findIndex((m) => m.localId === item.localId)
                                  if (idx <= 0) return
                                  let userMsg = ''
                                  for (let i = idx - 1; i >= 0; i--) {
                                    if (messages[i].role === 'user') {
                                      userMsg = messages[i].content
                                      break
                                    }
                                  }
                                  if (userMsg) {
                                    setInputValue(userMsg)
                                    void handleSend()
                                  }
                                }}
                              />
                            </Tooltip>
                          </div>
                        ) : null}
                      </div>
                    </div>
                  </List.Item>
                )}
              />
            )}
          </div>

          {toolCalls.length || actionProposals.length ? (
            <Card size="small" title="本轮上下文结果">
              {toolCalls.length ? (
                <div style={{ marginBottom: actionProposals.length ? 12 : 0 }}>
                  <Text strong>诊断工具</Text>
                  <List
                    size="small"
                    dataSource={toolCalls}
                    renderItem={(item) => (
                      <List.Item style={{ paddingInline: 0 }}>
                        <Space wrap>
                          <Tag color={item.status === 'succeeded' ? 'green' : item.status === 'failed' ? 'red' : 'blue'}>
                            {item.toolName}
                          </Tag>
                          <Text>{item.resultSummary || item.errorMessage || item.status}</Text>
                        </Space>
                      </List.Item>
                    )}
                  />
                </div>
              ) : null}

              {actionProposals.length ? (
                <div>
                  <Text strong>变更提案</Text>
                  <List
                    size="small"
                    dataSource={actionProposals}
                    renderItem={(item) => {
                      const riskColor = item.riskLevel === 'high' ? 'red' : item.riskLevel === 'medium' ? 'orange' : 'green'
                      const statusColor = item.status === 'pending_confirm' ? 'orange' : item.status === 'approved' ? 'blue' : item.status === 'succeeded' ? 'green' : item.status === 'failed' ? 'red' : 'default'
                      return (
                        <List.Item style={{ paddingInline: 0 }}>
                          <div style={{ width: '100%' }}>
                            <Space wrap>
                              <Tag color="orange">{item.actionType}</Tag>
                              <Tag color={riskColor}>{item.riskLevel}</Tag>
                              <Text strong>{item.title || item.summary}</Text>
                              <Tag color={statusColor}>{item.status}</Tag>
                            </Space>
                            <div style={{ marginTop: 4, fontSize: 12, color: '#8c8c8c' }}>
                              目标资源：{item.targetKind} / {item.targetNamespace} / {item.targetName}
                            </div>
                            {item.riskLevel === 'high' && (
                              <Alert
                                type="error"
                                showIcon
                                banner
                                message="高危操作"
                                description="此操作为高危操作，需要两位管理员分别确认后才能执行。请仔细核实操作目标和影响范围。"
                                style={{ marginTop: 8, marginBottom: 8 }}
                              />
                            )}
                            {item.status === 'pending_confirm' ? (
                              <div style={{ marginTop: 8 }}>
                                {confirmingId === item.id ? (
                                  <Space direction="vertical" size={8} style={{ width: '100%' }}>
                                    <Input
                                      placeholder={`请输入确认短语：${item.requiredConfirmationText}`}
                                      value={confirmText}
                                      onChange={(e) => setConfirmText(e.target.value)}
                                      size="small"
                                    />
                                    <Checkbox
                                      checked={confirmRisk}
                                      onChange={(e) => setConfirmRisk(e.target.checked)}
                                    >
                                      我已了解此操作的风险
                                    </Checkbox>
                                    {item.riskLevel === 'high' && item.confirmLevel === 'double' && (
                                      <div style={{ fontSize: 12, color: '#ff4d4f', marginTop: 4 }}>
                                        ⚠ 此操作需要双重确认：您确认后，还需另一位管理员进行二次确认。
                                      </div>
                                    )}
                                    <Space>
                                      <Button
                                        type="primary"
                                        size="small"
                                        loading={confirmLoading}
                                        onClick={() => void handleConfirm(item)}
                                      >
                                        确认执行
                                      </Button>
                                      <Button
                                        size="small"
                                        onClick={() => {
                                          setConfirmingId(null)
                                          setConfirmText('')
                                          setConfirmRisk(false)
                                        }}
                                      >
                                        拒绝
                                      </Button>
                                    </Space>
                                  </Space>
                                ) : (
                                  <Button
                                    type="primary"
                                    size="small"
                                    onClick={() => {
                                      setConfirmingId(item.id)
                                      setConfirmText('')
                                      setConfirmRisk(false)
                                    }}
                                  >
                                    确认
                                  </Button>
                                )}
                              </div>
                            ) : null}
                            {item.status === 'approved' ? (
                              <div style={{ marginTop: 4, color: '#1677ff', fontSize: 12 }}>
                                等待二次确认
                              </div>
                            ) : null}
                            {item.status === 'succeeded' ? (
                              <div style={{ marginTop: 4, color: '#52c41a', fontSize: 12 }}>
                                执行成功
                              </div>
                            ) : null}
                            {item.status === 'failed' ? (
                              <div style={{ marginTop: 4, color: '#ff4d4f', fontSize: 12 }}>
                                执行失败：{item.latestExecution?.errorMessage || '未知错误'}
                              </div>
                            ) : null}
                          </div>
                        </List.Item>
                      )
                    }}
                  />
                </div>
              ) : null}
            </Card>
          ) : null}

          <Card size="small" title="输入与附件">
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <Upload
                multiple
                beforeUpload={() => false}
                fileList={pendingFiles}
                onChange={({ fileList }) => setPendingFiles(fileList.slice(-6))}
              >
                <Button icon={<PaperClipOutlined />}>上传文件 / 图片</Button>
              </Upload>

              <Space size={[6, 6]} wrap>
                {imageFiles.length ? <Tag color="purple">{imageFiles.length} 张图片</Tag> : null}
                {textFiles.length ? <Tag color="cyan">{textFiles.length} 个文件</Tag> : null}
              </Space>

              <TextArea
                value={inputValue}
                onChange={(event) => setInputValue(event.target.value)}
                disabled={isBusy}
                autoSize={{ minRows: 4, maxRows: 8 }}
                placeholder="输入运维问题。可结合附件、命名空间或具体资源进行追问。"
                onPressEnter={(event) => {
                  if (!event.shiftKey) {
                    event.preventDefault()
                    void handleSend()
                  }
                }}
              />

              <Space>
                {isStreaming ? (
                  <Button icon={<StopOutlined />} onClick={stopGeneration}>
                    停止生成
                  </Button>
                ) : (
                  <Button
                    type="primary"
                    icon={<SendOutlined />}
                    loading={isResponding}
                    onClick={() => void handleSend()}
                    disabled={!selectedClusterId || !inputValue.trim() || Boolean(modelCapabilityWarning) || isResponding}
                  >
                    发送
                  </Button>
                )}
                {activeConversationId ? (
                  <Tooltip title="清空当前对话">
                    <Button
                      icon={<ClearOutlined />}
                      onClick={() => {
                        clearConversation()
                        history.replace('/ai/chat')
                        setActiveConversationId(undefined)
                      }}
                    >
                      新对话
                    </Button>
                  </Tooltip>
                ) : null}
              </Space>

              {/* 诊断模板快捷入口 */}
              <div>
                <Text type="secondary" style={{ fontSize: 12, marginRight: 8 }}>快捷模板：</Text>
                <Space size={[4, 4]} wrap>
                  <Button size="small" icon={<ThunderboltOutlined />} onClick={() => setInputValue('请帮我做一次当前范围的故障诊断')}>巡检诊断</Button>
                  <Button size="small" icon={<FileTextOutlined />} onClick={() => setInputValue('请查看最近的集群事件并分析是否有异常')}>事件分析</Button>
                  <Button size="small" icon={<FileSearchOutlined />} onClick={() => setInputValue('请查看当前范围内异常 Pod 的日志')}>日志排查</Button>
                  <Button size="small" icon={<FileTextOutlined />} onClick={() => setInputValue('请导出当前资源的 YAML 配置并分析')}>YAML 分析</Button>
                  <Button size="small" icon={<MessageOutlined />} onClick={() => setInputValue('请帮我做一次当前范围的故障诊断')}>诊断模板</Button>
                </Space>
              </div>
            </Space>
          </Card>
        </Card>
      </div>
    </AppPage>
  )
}

export default AIChatPage
