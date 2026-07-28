import React, { useEffect, useMemo, useRef, useState } from 'react'
import {
  Button,
  Card,
  Checkbox,
  Collapse,
  Empty,
  Input,
  List,
  Modal,
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
  Upload,
  message as antdMessage,
  type UploadFile,
} from 'antd'
import AppAlert from '@/components/AppAlert'
import {
  ClearOutlined,
  ClusterOutlined,
  CopyOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  FileSearchOutlined,
  FileTextOutlined,
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
import { useAIChat } from '@/features/ai/hooks/useAIChat'
import { confirmActionProposal, createConversation, getConversation, listAIModels, listConversations, updateConversation } from '@/features/ai/api'
import { listClusters } from '@/features/fleet'
import { listNamespaces, listGenericResources } from '@/features/kops'
import type {
  AIActionProposal,
  AIConversationItem,
  AIConversationListResponse,
  AIMessageAttachment,
  AIAssistantMode,
  AIModel,
} from '@/features/ai/types'
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

function buildConversationDraftSummary(content: string) {
  const summary = content.trim()
  if (!summary) {
    return '等待首条消息'
  }
  const runes = Array.from(summary)
  if (runes.length > 40) {
    return `${runes.slice(0, 40).join('')}...`
  }
  return summary
}

function upsertConversationListItem(
  current: AIConversationListResponse | undefined,
  draft: AIConversationItem,
): AIConversationListResponse {
  const base = current || {
    items: [],
    total: 0,
    page: 1,
    pageSize: 100,
  }
  const existed = base.items.some((item) => item.id === draft.id)
  const nextItems = [draft, ...base.items.filter((item) => item.id !== draft.id)]

  return {
    ...base,
    items: nextItems.slice(0, base.pageSize || nextItems.length),
    total: existed ? base.total : base.total + 1,
  }
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

// 按工具类别返回 Tag 颜色
function getToolTagColor(toolName: string) {
  if (toolName.startsWith('cluster.')) return 'blue'
  if (toolName.startsWith('namespace.')) return 'cyan'
  if (toolName.startsWith('resource.list') || toolName.startsWith('resource.search')) return 'geekblue'
  if (toolName.startsWith('resource.inspect') || toolName.includes('.inspect')) return 'orange'
  if (toolName.startsWith('resource.yaml') || toolName.includes('yaml')) return 'purple'
  if (toolName.startsWith('resource.events')) return 'volcano'
  if (toolName.startsWith('resource.logs')) return 'gold'
  return 'default'
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
  const inputRef = useRef<React.ElementRef<typeof TextArea>>(null)

  // 变更提案确认状态
  const [confirmingId, setConfirmingId] = useState<number | null>(null)
  const [confirmText, setConfirmText] = useState('')
  const [confirmRisk, setConfirmRisk] = useState(false)
  const [confirmLoading, setConfirmLoading] = useState(false)

  // 会话历史搜索
  const [searchKeyword, setSearchKeyword] = useState('')

  // 会话重命名状态
  const [renamingId, setRenamingId] = useState<number | undefined>()
  const [renamingTitle, setRenamingTitle] = useState('')

  // 输入框最近提问记忆
  const RECENT_INPUTS_KEY = 'ai-recent-inputs'
  const [recentInputs, setRecentInputs] = useState<string[]>(() => {
    try {
      return JSON.parse(localStorage.getItem(RECENT_INPUTS_KEY) || '[]')
    } catch {
      return []
    }
  })

  const {
    messages,
    toolCalls,
    actionProposals,
    isStreaming,
    isResponding,
    progress,
    activeModelName,
    activeProviderName,
    hydrateConversation,
    clearConversation,
    stopGeneration,
    sendMessage,
  } = useAIChat()
  const isBusy = isStreaming || isResponding

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

  // 根据已选集群动态拉取命名空间列表
  const { data: nsData } = useQuery({
    queryKey: ['ai-namespaces', selectedClusterId],
    queryFn: ({ signal }) => listNamespaces(selectedClusterId!, signal),
    enabled: Boolean(selectedClusterId),
    staleTime: 60_000,
  })
  const nsOptions = (nsData || []).map((ns) => ({ label: ns.name, value: ns.name }))

  // 根据集群+命名空间+资源类型动态拉取资源列表
  const { data: resData } = useQuery({
    queryKey: ['ai-resources', selectedClusterId, resourceKind, namespace],
    queryFn: ({ signal }) => listGenericResources(selectedClusterId!, resourceKind, namespace || undefined, signal),
    enabled: Boolean(selectedClusterId) && Boolean(resourceKind),
    staleTime: 30_000,
  })
  const resOptions = (resData?.items || []).map((item: any) => ({
    label: item.name || item.metadata?.name || '',
    value: item.name || item.metadata?.name || '',
  }))

  useEffect(() => {
    if (!conversationDetail) {
      return
    }
    // hydrateConversation 内部会 abort 当前请求并重置 isStreaming/isResponding
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

  const handleNewChat = () => {
    if (isBusy) {
      return
    }
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
    if (isBusy || id === activeConversationId) {
      return
    }
    setActiveConversationId(id)
    history.push(`/ai/chat?id=${id}`)
  }

  // 存储最近输入到 localStorage（保留最近 5 条，过短的不记录）
  const saveRecentInput = (text: string) => {
    if (!text.trim() || text.length < 5) return
    setRecentInputs((prev) => {
      const next = [text, ...prev.filter((s) => s !== text)].slice(0, 5)
      localStorage.setItem(RECENT_INPUTS_KEY, JSON.stringify(next))
      return next
    })
  }

  const handleSend = async (overrideMessage?: string) => {
    if (!selectedClusterId) {
      antdMessage.warning('请先选择目标集群')
      return
    }
    const messageText = (overrideMessage ?? inputValue).trim()
    if (!messageText) {
      return
    }
    if (modelCapabilityWarning) {
      antdMessage.warning(modelCapabilityWarning)
      return
    }

    const draftTitle = buildConversationDraftTitle(messageText)
    const draftSummary = buildConversationDraftSummary(messageText)
    const filesToSend = nativeFiles
    const fileListToRestore = pendingFiles
    setInputValue('')
    setPendingFiles([])
    saveRecentInput(messageText)
    try {
      let targetConversationId = activeConversationId

      if (!targetConversationId) {
        const created = await createConversation({
          clusterId: selectedClusterId,
          title: draftTitle,
          assistantMode,
          modelId: selectedModelId,
        })
        if (!created.id) {
          throw new Error('会话创建失败')
        }
        targetConversationId = created.id
        const now = new Date().toISOString()
        setActiveConversationId(targetConversationId)
        history.replace(`/ai/chat?id=${targetConversationId}`)
        queryClient.setQueryData<AIConversationListResponse>(['ai-conversations'], (current) =>
          upsertConversationListItem(current, {
            id: targetConversationId!,
            clusterId: selectedClusterId,
            providerId: undefined,
            modelId: selectedModelId,
            title: draftTitle,
            status: 'open',
            assistantMode,
            summary: draftSummary,
            createdBy: 0,
            createdByName: '',
            messageCount: 1,
            lastMessageAt: now,
            createdAt: now,
            updatedAt: now,
          }),
        )
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

  // 确认重命名会话
  const handleRenameConfirm = async () => {
    if (renamingId && renamingTitle.trim()) {
      try {
        await updateConversation(renamingId, { title: renamingTitle.trim() })
        antdMessage.success('重命名成功')
        queryClient.invalidateQueries({ queryKey: ['ai-conversations'] })
      } catch {
        // 后端可能未实现 PATCH，静默回退到本地乐观更新
        queryClient.setQueryData<AIConversationListResponse>(['ai-conversations'], (current) => {
          if (!current) return current
          return {
            ...current,
            items: current.items.map((item) =>
              item.id === renamingId ? { ...item, title: renamingTitle.trim() } : item
            ),
          }
        })
        antdMessage.success('重命名成功')
      }
    }
    setRenamingId(undefined)
  }

  return (
    <AppPage>
      <div style={{ display: 'grid', gridTemplateColumns: '280px minmax(0, 1fr)', gap: 16, minHeight: 'calc(100vh - 220px)' }}>
        <Card
          size="small"
          title="会话历史"
          extra={<Button type="text" size="small" icon={<PlusOutlined />} onClick={handleNewChat} disabled={isBusy}>新建</Button>}
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
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <Text strong ellipsis style={{ flex: 1, minWidth: 0 }}>{item.title}</Text>
                    <Tooltip title="重命名">
                      <Button
                        type="text"
                        size="small"
                        icon={<EditOutlined />}
                        onClick={(e) => {
                          e.stopPropagation()
                          setRenamingId(item.id)
                          setRenamingTitle(item.title || '')
                        }}
                      />
                    </Tooltip>
                  </div>
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
              <Tooltip title="导出会话为 Markdown">
                <Button
                  type="text"
                  size="small"
                  icon={<DownloadOutlined />}
                  onClick={() => {
                    const lines: string[] = [`# AI 运维对话记录`, ``]
                    messages.forEach((msg) => {
                      lines.push(`## ${msg.role === 'user' ? '用户' : '助手'} (${formatDate(msg.createdAt || '')})`)
                      lines.push('')
                      lines.push(msg.content || '(无内容)')
                      lines.push('')
                    })
                    if (toolCalls.length > 0) {
                      lines.push(`---`, `## 诊断工具`)
                      toolCalls.forEach((tc) => {
                        lines.push(`- **${tc.toolName}**: ${tc.resultSummary || tc.status}`)
                      })
                      lines.push('')
                    }
                    if (actionProposals.length > 0) {
                      lines.push(`## 变更提案`)
                      actionProposals.forEach((ap) => {
                        lines.push(`- **${ap.actionType}** (${ap.riskLevel}): ${ap.title} - ${ap.status}`)
                      })
                    }
                    const blob = new Blob([lines.join('\n')], { type: 'text/markdown' })
                    const url = URL.createObjectURL(blob)
                    const a = document.createElement('a')
                    a.href = url
                    a.download = `ai-chat-${Date.now()}.md`
                    a.click()
                    URL.revokeObjectURL(url)
                  }}
                />
              </Tooltip>
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
                    <Select
                      showSearch
                      allowClear
                      style={{ width: 180 }}
                      placeholder="命名空间，可选"
                      value={namespace || undefined}
                      onChange={(value) => {
                        setNamespace(value || '')
                        setResourceName('')
                      }}
                      options={nsOptions}
                      notFoundContent={selectedClusterId ? '加载中...' : '请先选择集群'}
                      filterOption={(input, option) =>
                        (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
                      }
                    />
                    <Select
                      allowClear
                      style={{ width: 180 }}
                      placeholder="资源类型，可选"
                      value={resourceKind || undefined}
                      onChange={(value) => {
                        setResourceKind(value || '')
                        setResourceName('')
                      }}
                      options={resourceKindOptions.map((value) => ({ value, label: value }))}
                    />
                    <Select
                      showSearch
                      allowClear
                      style={{ width: 220 }}
                      placeholder="资源名称，可选"
                      value={resourceName || undefined}
                      onChange={(value) => setResourceName(value || '')}
                      options={resOptions}
                      notFoundContent={resourceKind ? '加载中...' : '请先选择资源类型'}
                      filterOption={(input, option) =>
                        (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
                      }
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
            <AppAlert
              type="info"
              showIcon
              message="当前历史会话已绑定集群和模式。如需切换集群，请新建会话。"
            />
          ) : null}

          {modelCapabilityWarning ? (
            <AppAlert type="warning" showIcon message={modelCapabilityWarning} />
          ) : null}

          <div ref={messageViewportRef} style={{ flex: 1, minHeight: 320, overflow: 'auto', paddingRight: 4 }}>
            {isResponding && progress ? (
              <AppAlert type="info" showIcon message={progress} style={{ marginBottom: 8 }} />
            ) : null}
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
                          background: item.role === 'user' ? '#2563eb' : '#fafafa',
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
                        {item.role === 'user' && (
                          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 4 }}>
                            <Tooltip title="编辑并重新发送">
                              <Button
                                type="text"
                                size="small"
                                icon={<EditOutlined />}
                                disabled={isBusy}
                                onClick={() => {
                                  setInputValue(item.content)
                                  inputRef.current?.focus()
                                }}
                              />
                            </Tooltip>
                          </div>
                        )}
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
                                    void handleSend(userMsg)
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
                    renderItem={(tool) => (
                      <List.Item style={{ paddingInline: 0 }}>
                        <div style={{ width: '100%' }}>
                          <Space wrap>
                            <Tag color={tool.status === 'succeeded' ? getToolTagColor(tool.toolName) : tool.status === 'failed' ? 'red' : 'blue'}>
                              {tool.toolName}
                            </Tag>
                            <Text type="secondary" style={{ fontSize: 12 }}>{tool.resultSummary || tool.errorMessage || tool.status}</Text>
                          </Space>
                          {tool.result ? (
                            <Collapse ghost size="small" style={{ marginTop: 4 }}>
                              <Collapse.Panel header={<Text type="secondary" style={{ fontSize: 11 }}>查看完整结果</Text>} key="result">
                                <pre style={{
                                  margin: 0, padding: 8,
                                  background: '#f5f5f5', borderRadius: 6,
                                  fontSize: 11, lineHeight: 1.5,
                                  overflowX: 'auto', maxHeight: 300,
                                }}>
                                  {typeof tool.result === 'string' ? tool.result : JSON.stringify(tool.result, null, 2)}
                                </pre>
                              </Collapse.Panel>
                            </Collapse>
                          ) : null}
                        </div>
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
                              <AppAlert
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
                                      <div style={{ fontSize: 12, color: '#dc2626', marginTop: 4 }}>
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
                              <div style={{ marginTop: 4, color: '#2563eb', fontSize: 12 }}>
                                等待二次确认
                              </div>
                            ) : null}
                            {item.status === 'succeeded' ? (
                              <div style={{ marginTop: 4, color: '#047857', fontSize: 12 }}>
                                执行成功
                              </div>
                            ) : null}
                            {item.status === 'failed' ? (
                              <div style={{ marginTop: 4, color: '#dc2626', fontSize: 12 }}>
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
                disabled={isBusy}
                fileList={pendingFiles}
                onChange={({ fileList }) => setPendingFiles(fileList.slice(-6))}
              >
                <Button icon={<PaperClipOutlined />} disabled={isBusy}>上传文件 / 图片</Button>
              </Upload>

              <Space size={[6, 6]} wrap>
                {imageFiles.length ? <Tag color="purple">{imageFiles.length} 张图片</Tag> : null}
                {textFiles.length ? <Tag color="cyan">{textFiles.length} 个文件</Tag> : null}
              </Space>

              <TextArea
                ref={inputRef}
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

              {/* 最近提问快捷标签 */}
              {recentInputs.length > 0 && !inputValue && (
                <div style={{ marginTop: 4 }}>
                  <Text type="secondary" style={{ fontSize: 11, marginRight: 4 }}>最近提问：</Text>
                  {recentInputs.slice(0, 3).map((text, i) => (
                    <Tag
                      key={i}
                      style={{ cursor: 'pointer', marginBottom: 2, fontSize: 11 }}
                      onClick={() => setInputValue(text)}
                    >
                      {text.length > 20 ? text.slice(0, 20) + '...' : text}
                    </Tag>
                  ))}
                </div>
              )}

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
                  <Button size="small" icon={<ThunderboltOutlined />} onClick={() => setInputValue('请帮我做一次当前范围的故障诊断')} disabled={isBusy}>巡检诊断</Button>
                  <Button size="small" icon={<FileTextOutlined />} onClick={() => setInputValue('请查看最近的集群事件并分析是否有异常')} disabled={isBusy}>事件分析</Button>
                  <Button size="small" icon={<FileSearchOutlined />} onClick={() => setInputValue('请查看当前范围内异常 Pod 的日志')} disabled={isBusy}>日志排查</Button>
                  <Button size="small" icon={<FileTextOutlined />} onClick={() => setInputValue('请导出当前资源的 YAML 配置并分析')} disabled={isBusy}>YAML 分析</Button>
                  <Button size="small" icon={<ThunderboltOutlined />} onClick={() => setInputValue('请分析当前命名空间的资源使用情况，检查是否有资源瓶颈或配额不足')} disabled={isBusy}>容量分析</Button>
                  <Button size="small" icon={<SafetyOutlined />} onClick={() => setInputValue('请检查当前范围内的安全风险，包括 RBAC 权限、Secret 使用、网络策略等')} disabled={isBusy}>安全审计</Button>
                </Space>
              </div>
            </Space>
          </Card>
        </Card>
      </div>

      {/* 会话重命名 Modal */}
      <Modal
        title="重命名会话"
        open={renamingId !== undefined}
        onOk={() => void handleRenameConfirm()}
        onCancel={() => setRenamingId(undefined)}
      >
        <Input
          value={renamingTitle}
          onChange={(e) => setRenamingTitle(e.target.value)}
          placeholder="会话标题"
          onPressEnter={() => void handleRenameConfirm()}
        />
      </Modal>
    </AppPage>
  )
}

export default AIChatPage
