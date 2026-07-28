/**
 * Pod 日志查看抽屉
 * 对标 frontend-old 的 PodLogDrawer
 * 支持实时日志流和历史日志查看
 */
import React, { useEffect, useRef, useState } from 'react'
import { Drawer, Spin, Switch, Space, Button, message } from 'antd'
import { ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { getPodLogs } from '@/features/kops/api/k8s'

interface PodLogDrawerProps {
  open: boolean
  onClose: () => void
  clusterId: number
  namespace: string
  podName: string
}

export const PodLogDrawer: React.FC<PodLogDrawerProps> = ({
  open,
  onClose,
  clusterId,
  namespace,
  podName,
}) => {
  const [autoScroll, setAutoScroll] = useState(true)
  const tailLines = 500
  const logRef = useRef<HTMLPreElement>(null)

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['pod-logs', clusterId, namespace, podName, tailLines],
    queryFn: () => getPodLogs(clusterId, namespace, podName, { tailLines }),
    enabled: open && !!clusterId,
    refetchInterval: open ? 5000 : false,
  })

  useEffect(() => {
    if (autoScroll && logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight
    }
  }, [data?.logs, autoScroll])

  const handleDownload = () => {
    if (!data?.logs) return
    const blob = new Blob([data.logs], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${podName}-logs.txt`
    a.click()
    URL.revokeObjectURL(url)
    message.success('日志已下载')
  }

  return (
    <Drawer
      title={`Pod 日志 - ${podName}`}
      open={open}
      onClose={onClose}
      width={800}
      extra={
        <Space>
          <Switch checked={autoScroll} onChange={setAutoScroll} checkedChildren="自动滚动" unCheckedChildren="手动滚动" />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>
          <Button icon={<DownloadOutlined />} onClick={handleDownload}>下载</Button>
        </Space>
      }
    >
      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="加载日志中..." /></div>
      ) : (
        <pre
          ref={logRef}
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
          {data?.logs || '暂无日志'}
        </pre>
      )}
    </Drawer>
  )
}
