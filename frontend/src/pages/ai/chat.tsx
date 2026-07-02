import React, { useState } from 'react'
import { Button, Card, Input, List, Select, Space, Spin, Tag, Typography } from 'antd'
import {
  ClearOutlined,
  ClusterOutlined,
  MessageOutlined,
  PlusOutlined,
  SendOutlined,
  StopOutlined,
} from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from '@umijs/max'
import { AppPage } from '@/components'
import { useAIChat } from '@/hooks'
import { getConversation, listConversations } from '@/services/ai'
import { listClusters } from '@/services/clusters'
import { formatDate } from '@/utils'

const { Paragraph, Text } = Typography

const AIChatPage: React.FC = () => {
  const [searchParams] = useSearchParams()
  const conversationId = searchParams.get('id')
  const [inputValue, setInputValue] = useState('')
  const [selectedClusterId, setSelectedClusterId] = useState<number | undefined>(undefined)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const { messages, isStreaming, sendMessage, stopGeneration, clearMessages } = useAIChat(
    selectedClusterId ? String(selectedClusterId) : undefined,
  )

  const { data: clustersData } = useQuery({
    queryKey: ['clusters-for-ai'],
    queryFn: () => listClusters(),
  })

  const { data: conversationsData } = useQuery({
    queryKey: ['conversations'],
    queryFn: () => listConversations(),
  })

  useQuery({
    queryKey: ['conversation', conversationId],
    queryFn: () => getConversation(conversationId!),
    enabled: !!conversationId,
  })

  const clusters = clustersData?.items || []
  const conversations = conversationsData?.items || []

  const handleSend = () => {
    if (!inputValue.trim() || isStreaming) {
      return
    }
    sendMessage(inputValue.trim())
    setInputValue('')
  }

  const handleNewChat = () => {
    clearMessages()
    window.history.pushState({}, '', '/ai/chat')
  }

  return (
    <AppPage>
      <div style={{ display: 'flex', gap: 16, minHeight: 'calc(100vh - 240px)' }}>
        <Card
          size="small"
          style={{ width: sidebarCollapsed ? 48 : 260, flexShrink: 0, overflow: 'auto' }}
          bodyStyle={{ padding: sidebarCollapsed ? '8px' : '12px' }}
        >
          {sidebarCollapsed ? (
            <Button type="text" icon={<MessageOutlined />} onClick={() => setSidebarCollapsed(false)} />
          ) : (
            <>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}>
                <Text strong>对话列表</Text>
                <Space>
                  <Button type="text" size="small" icon={<PlusOutlined />} onClick={handleNewChat} />
                  <Button type="text" size="small" onClick={() => setSidebarCollapsed(true)}>
                    收起
                  </Button>
                </Space>
              </div>
              <List
                size="small"
                dataSource={conversations}
                renderItem={(item) => (
                  <List.Item
                    style={{
                      cursor: 'pointer',
                      padding: '8px',
                      borderRadius: 8,
                      background: String(item.id) === conversationId ? '#e6f4ff' : 'transparent',
                    }}
                    onClick={() => {
                      window.location.href = `/ai/chat?id=${item.id}`
                    }}
                  >
                    <List.Item.Meta
                      title={<Text ellipsis style={{ fontSize: 13 }}>{item.title}</Text>}
                      description={
                        <Text type="secondary" ellipsis style={{ fontSize: 11 }}>
                          {item.messageCount} 条消息 · {formatDate(item.updatedAt)}
                        </Text>
                      }
                    />
                  </List.Item>
                )}
              />
            </>
          )}
        </Card>

        <Card
          style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
          bodyStyle={{ flex: 1, overflow: 'auto', display: 'flex', flexDirection: 'column' }}
          title={
            <Space>
              <ClusterOutlined />
              <Select
                allowClear
                size="small"
                placeholder="选择集群（可选）"
                style={{ width: 220 }}
                value={selectedClusterId}
                onChange={setSelectedClusterId}
                options={clusters.map((cluster) => ({ label: cluster.name, value: cluster.id }))}
              />
            </Space>
          }
          extra={
            messages.length > 0 ? (
              <Button icon={<ClearOutlined />} onClick={clearMessages} disabled={isStreaming} size="small">
                清空
              </Button>
            ) : null
          }
        >
          <div style={{ flex: 1, overflow: 'auto', paddingBottom: 16 }}>
            {messages.length === 0 && (
              <div style={{ textAlign: 'center', padding: '80px 0', color: '#999' }}>
                <div style={{ fontSize: 40, marginBottom: 16 }}>AI</div>
                <div style={{ fontSize: 16, marginBottom: 8 }}>AI 运维助手</div>
                <div style={{ fontSize: 13 }}>输入运维问题，系统会给出诊断建议、操作思路和命令参考。</div>
                {selectedClusterId && (
                  <Tag color="blue" style={{ marginTop: 8 }}>
                    当前集群：{clusters.find((cluster) => cluster.id === selectedClusterId)?.name}
                  </Tag>
                )}
                <div style={{ marginTop: 24, display: 'flex', justifyContent: 'center', gap: 8, flexWrap: 'wrap' }}>
                  {['如何查看 Pod 日志？', 'Pod 频繁重启怎么办？', 'Deployment 滚动更新', '如何排查 OOMKilled'].map((question) => (
                    <Button key={question} size="small" onClick={() => setInputValue(question)}>
                      {question}
                    </Button>
                  ))}
                </div>
              </div>
            )}

            <List
              dataSource={messages}
              renderItem={(item) => (
                <List.Item style={{ border: 'none', padding: '8px 0' }}>
                  <div
                    style={{
                      width: '100%',
                      display: 'flex',
                      justifyContent: item.role === 'user' ? 'flex-end' : 'flex-start',
                    }}
                  >
                    <div
                      style={{
                        maxWidth: '80%',
                        padding: '12px 16px',
                        borderRadius: 10,
                        background: item.role === 'user' ? '#1677ff' : '#f5f5f5',
                        color: item.role === 'user' ? '#fff' : '#333',
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                      }}
                    >
                      <Paragraph style={{ margin: 0, color: 'inherit', whiteSpace: 'pre-wrap' }}>
                        {item.content}
                      </Paragraph>
                      {item.isStreaming && (
                        <span
                          style={{
                            display: 'inline-block',
                            width: 6,
                            height: 14,
                            marginLeft: 2,
                            background: '#1677ff',
                            animation: 'blink 1s infinite',
                          }}
                        />
                      )}
                    </div>
                  </div>
                </List.Item>
              )}
            />

            {isStreaming && messages.length > 0 && messages[messages.length - 1]?.content === '' && (
              <div style={{ textAlign: 'center', padding: 16 }}>
                <Spin tip="AI 思考中..." />
              </div>
            )}
          </div>

          <div style={{ paddingTop: 12, borderTop: '1px solid #f0f0f0' }}>
            <Input.Group compact>
              <Input
                style={{ width: 'calc(100% - 100px)' }}
                placeholder={selectedClusterId ? '输入运维问题（已绑定集群上下文）...' : '输入运维问题...'}
                value={inputValue}
                onChange={(event) => setInputValue(event.target.value)}
                onPressEnter={handleSend}
                disabled={isStreaming}
              />
              {isStreaming ? (
                <Button style={{ width: 100 }} icon={<StopOutlined />} onClick={stopGeneration}>
                  停止
                </Button>
              ) : (
                <Button
                  style={{ width: 100 }}
                  type="primary"
                  icon={<SendOutlined />}
                  onClick={handleSend}
                  disabled={!inputValue.trim()}
                >
                  发送
                </Button>
              )}
            </Input.Group>
          </div>
        </Card>
      </div>

      <style>{`
        @keyframes blink {
          0%, 50% { opacity: 1; }
          51%, 100% { opacity: 0; }
        }
      `}</style>
    </AppPage>
  )
}

export default AIChatPage
