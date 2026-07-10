import React from 'react'
import { Button, Card, Empty, List, message, Popconfirm, Tag, Typography } from 'antd'
import { DeleteOutlined, MessageOutlined, PlusOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@umijs/max'
import { AppPage } from '@/components'
import { deleteConversation, listConversations } from '@/services/ai'
import { formatDate } from '@/utils'

const { Text } = Typography

const AIHistoryPage: React.FC = () => {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['ai-conversations'],
    queryFn: ({ signal }) => listConversations(undefined, signal),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteConversation(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({ queryKey: ['ai-conversations'] })
    },
    onError: () => message.error('删除失败'),
  })

  const conversations = data?.items || []

  return (
    <AppPage>
      <Card
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/ai/chat')}>
            新建对话
          </Button>
        }
      >
        {!isLoading && conversations.length === 0 ? (
          <Empty description="暂无对话历史">
            <Button type="primary" onClick={() => navigate('/ai/chat')}>
              开始对话
            </Button>
          </Empty>
        ) : (
          <List
            loading={isLoading}
            dataSource={conversations}
            renderItem={(item) => (
              <List.Item
                actions={[
                  <Tag key="count" color="blue">
                    {item.messageCount} 条消息
                  </Tag>,
                  <Text key="time" type="secondary" style={{ fontSize: 12 }}>
                    {formatDate(item.updatedAt)}
                  </Text>,
                  <Popconfirm
                    key="delete"
                    title="确定删除该对话？"
                    onConfirm={() => deleteMutation.mutate(item.id)}
                  >
                    <Button type="text" danger size="small" icon={<DeleteOutlined />} />
                  </Popconfirm>,
                ]}
              >
                <List.Item.Meta
                  avatar={<MessageOutlined style={{ fontSize: 20, color: '#1677ff' }} />}
                  title={<a onClick={() => navigate(`/ai/chat?id=${item.id}`)}>{item.title}</a>}
                  description={
                    <Text type="secondary" ellipsis style={{ maxWidth: 500 }}>
                      更新于 {formatDate(item.updatedAt)}
                    </Text>
                  }
                />
              </List.Item>
            )}
          />
        )}
      </Card>
    </AppPage>
  )
}

export default AIHistoryPage
