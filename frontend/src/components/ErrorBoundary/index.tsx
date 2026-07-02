/**
 * 错误边界组件
 * 捕获渲染错误，显示友好错误页面
 */
import React from 'react'
import { Result, Button } from 'antd'

interface Props {
  children: React.ReactNode
}

interface State {
  hasError: boolean
  error?: Error
}

export class ErrorBoundary extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('ErrorBoundary caught:', error, errorInfo)
  }

  handleReset = () => {
    this.setState({ hasError: false, error: undefined })
  }

  render() {
    if (this.state.hasError) {
      return (
        <Result
          status="error"
          title="页面渲染异常"
          subTitle={this.state.error?.message || '未知错误'}
          extra={[
            <Button key="reset" type="primary" onClick={this.handleReset}>
              重新加载
            </Button>,
            <Button key="home" onClick={() => (window.location.href = '/')}>
              返回首页
            </Button>,
          ]}
        />
      )
    }

    return this.props.children
  }
}
