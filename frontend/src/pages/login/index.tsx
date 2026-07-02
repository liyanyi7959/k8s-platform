import React, { useState } from 'react'
import { history, useModel } from '@umijs/max'
import { Alert, Button, Form, Input, Space, Typography, message } from 'antd'
import {
  ArrowRightOutlined,
  DeploymentUnitOutlined,
  LockOutlined,
  SafetyCertificateOutlined,
  ThunderboltOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { getCurrentUser, login } from '@/services/auth'

const capabilityCards = [
  {
    title: '集群巡检',
    description: '统一查看连接状态、节点规模和核心健康指标。',
    icon: <DeploymentUnitOutlined />,
  },
  {
    title: '安全治理',
    description: '通过权限、证书和配置可视化降低风险暴露。',
    icon: <SafetyCertificateOutlined />,
  },
  {
    title: 'AI 辅助运维',
    description: '在同一工作台里接入诊断建议、命令和排障上下文。',
    icon: <ThunderboltOutlined />,
  },
]

/** 登录页 */
const LoginPage: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const { setInitialState } = useModel('@@initialState')

  const handleSubmit = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      await login(values)
      const currentUser = await getCurrentUser()
      setInitialState({ currentUser })
      message.success('登录成功')
      history.push('/')
    } catch {
      // 错误消息已由全局响应拦截器统一展示，此处不再重复提示
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="app-login-shell">
      <div className="app-login-panel">
        <section className="app-login-brand">
          <div className="app-login-brand__badge">Kubernetes Operations Console</div>
          <Typography.Title className="app-login-brand__title">
            AIOPS 智能运维平台
          </Typography.Title>
          <Typography.Paragraph className="app-login-brand__desc">
            将集群巡检、工作负载操作、监控告警和 AI 辅助诊断整合到一张更清晰的控制台里。
          </Typography.Paragraph>

          <div className="app-login-brand__grid">
            {capabilityCards.map((item) => (
              <div key={item.title} className="app-login-highlight">
                <div className="app-login-highlight__icon">{item.icon}</div>
                <div className="app-login-highlight__title">{item.title}</div>
                <div className="app-login-highlight__desc">{item.description}</div>
              </div>
            ))}
          </div>

          <div className="app-login-trustbar">
            <div className="app-login-trustbar__item">
              <span className="app-login-trustbar__label">管理视角</span>
              <span className="app-login-trustbar__value">集群 + 部署 + 监控</span>
            </div>
            <div className="app-login-trustbar__item">
              <span className="app-login-trustbar__label">响应方式</span>
              <span className="app-login-trustbar__value">实时操作与排障</span>
            </div>
          </div>
        </section>

        <section className="app-login-form">
          <span className="app-login-form__eyebrow">Sign In</span>
          <Typography.Title level={3} style={{ marginBottom: 0 }}>
            欢迎回来
          </Typography.Title>
          <Typography.Paragraph className="app-login-form__subtitle">
            使用平台账号进入工作台，继续查看集群状态、部署计划和 AI 会话记录。
          </Typography.Paragraph>

          <Alert
            className="app-login-demo"
            type="info"
            showIcon
            message="演示账号"
            description="admin / admin123"
          />

          <Form onFinish={handleSubmit} size="large" autoComplete="off" layout="vertical">
            <Form.Item
              label="用户名"
              name="username"
              rules={[{ required: true, message: '请输入用户名' }]}
              initialValue="admin"
            >
              <Input
                prefix={<UserOutlined />}
                placeholder="请输入平台用户名"
                autoComplete="username"
                style={{ height: 50 }}
              />
            </Form.Item>

            <Form.Item
              label="密码"
              name="password"
              rules={[{ required: true, message: '请输入密码' }]}
              initialValue="admin123"
            >
              <Input.Password
                prefix={<LockOutlined />}
                placeholder="请输入登录密码"
                autoComplete="current-password"
                style={{ height: 50 }}
              />
            </Form.Item>

            <Form.Item style={{ marginBottom: 0 }}>
              <Button
                type="primary"
                htmlType="submit"
                loading={loading}
                block
                icon={<ArrowRightOutlined />}
                style={{ height: 50, fontSize: 16, fontWeight: 700 }}
              >
                进入平台
              </Button>
            </Form.Item>
          </Form>

          <Space direction="vertical" size={4} className="app-login-footer">
            <span>登录后可直接访问集群总览、在线部署、监控告警与 AI 运维助手。</span>
            <span>如果使用正式环境账号，请确保浏览器已保存最新访问地址与令牌策略。</span>
          </Space>
        </section>
      </div>
    </div>
  )
}

export default LoginPage
