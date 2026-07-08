import React, { useCallback, useEffect, useRef, useState } from 'react'
import { history, useModel } from '@umijs/max'
import { Alert, Button, Checkbox, Form, Input, Modal, Space, Typography, message } from 'antd'
import { ArrowRightOutlined, LockOutlined, UserOutlined } from '@ant-design/icons'
import { getCurrentUser, login, requestPasswordReset, confirmPasswordReset } from '@/services/auth'

const REMEMBER_USER_KEY = 'aiops_remembered_user'
const TRACK_WIDTH = 280
const SLIDER_WIDTH = 40
const TOLERANCE = 6
const LOGIN_FAILURE_LIMIT = 5
const LOGIN_LOCK_MINUTES = 15

type LoginFailurePayload = {
  reason?: string
  failed_attempts?: number
  remaining_attempts?: number
  max_attempts?: number
  locked?: boolean
  lock_remaining_seconds?: number
  lock_duration_seconds?: number
  can_reset_password?: boolean
  suggestions?: string[]
}

type LoginRequestError = Error & {
  code?: number
  data?: LoginFailurePayload
}

type LoginFeedback = {
  type: 'error' | 'warning' | 'info'
  title: string
  detail: string
  reason?: string
  canResetPassword?: boolean
  suggestions: string[]
  lockRemainingSeconds?: number
}

const formatRemainingTime = (seconds: number) => {
  const safeSeconds = Math.max(seconds, 0)
  const minutes = Math.floor(safeSeconds / 60)
  const remainSeconds = safeSeconds % 60
  if (minutes > 0) {
    return `${minutes} 分 ${String(remainSeconds).padStart(2, '0')} 秒`
  }
  return `${remainSeconds} 秒`
}

const buildLoginFeedback = (error: unknown): LoginFeedback => {
  const requestError = error as LoginRequestError
  const payload = requestError?.data || {}
  const suggestions = Array.isArray(payload.suggestions) ? payload.suggestions.filter(Boolean) : []
  const baseMessage = requestError?.message || '登录失败，请稍后重试'

  if (payload.reason === 'account_locked' || requestError?.code === 4103) {
    return {
      type: 'error',
      title: '账号暂时锁定',
      detail: baseMessage,
      reason: 'account_locked',
      canResetPassword: payload.can_reset_password !== false,
      suggestions: suggestions.length > 0 ? suggestions : [
        '请等待锁定结束后再试',
        '如已忘记密码，可使用忘记密码功能重置',
      ],
      lockRemainingSeconds: payload.lock_remaining_seconds || 0,
    }
  }

  if (payload.reason === 'account_disabled' || requestError?.code === 4102) {
    return {
      type: 'error',
      title: '账号已被禁用',
      detail: baseMessage,
      reason: 'account_disabled',
      suggestions: suggestions.length > 0 ? suggestions : ['请联系管理员检查账号状态'],
    }
  }

  if (payload.reason === 'captcha_invalid' || requestError?.code === 4104) {
    return {
      type: 'warning',
      title: '安全验证失败',
      detail: baseMessage,
      reason: 'captcha_invalid',
      suggestions: suggestions.length > 0 ? suggestions : ['请重新完成滑块验证后再试'],
    }
  }

  if (payload.reason === 'invalid_params' || requestError?.code === 4000) {
    return {
      type: 'warning',
      title: '请补全登录信息',
      detail: baseMessage,
      reason: 'invalid_params',
      suggestions: suggestions.length > 0 ? suggestions : ['请填写用户名和密码后重新提交'],
    }
  }

  if (payload.reason === 'invalid_credentials' || requestError?.code === 4101) {
    const remainingAttempts = Number(payload.remaining_attempts || 0)
    const maxAttempts = Number(payload.max_attempts || 0)
    const detail = remainingAttempts > 0 && maxAttempts > 0
      ? `用户名或密码不正确。当前还可尝试 ${remainingAttempts} 次，连续失败 ${maxAttempts} 次后账号会被临时锁定。`
      : baseMessage

    return {
      type: 'error',
      title: '账号或密码不正确',
      detail,
      reason: 'invalid_credentials',
      canResetPassword: payload.can_reset_password !== false,
      suggestions: suggestions.length > 0 ? suggestions : [
        '请确认用户名、密码和大小写是否正确',
        '如已忘记密码，可使用忘记密码功能重置',
      ],
    }
  }

  return {
    type: 'error',
    title: '登录失败',
    detail: baseMessage,
    suggestions: suggestions.length > 0 ? suggestions : ['请检查网络连接或稍后重试'],
  }
}

const LoginPage: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [remember, setRemember] = useState(false)
  const [loginFeedback, setLoginFeedback] = useState<LoginFeedback>()
  const { setInitialState } = useModel('@@initialState')
  const [form] = Form.useForm()

  const [targetX, setTargetX] = useState(0)
  const [sliderX, setSliderX] = useState(0)
  const [dragging, setDragging] = useState(false)
  const [verified, setVerified] = useState(false)
  const trackRef = useRef<HTMLDivElement>(null)

  const [resetModalOpen, setResetModalOpen] = useState(false)
  const [resetStep, setResetStep] = useState<'request' | 'confirm'>('request')
  const [resetToken, setResetToken] = useState('')
  const [resetForm] = Form.useForm()

  const generateCaptcha = useCallback(() => {
    const max = TRACK_WIDTH - SLIDER_WIDTH - 20
    setTargetX(20 + Math.floor(Math.random() * max))
    setSliderX(0)
    setVerified(false)
  }, [])

  useEffect(() => {
    try {
      const raw = localStorage.getItem(REMEMBER_USER_KEY)
      if (raw) {
        form.setFieldsValue({ username: atob(raw) })
        setRemember(true)
      }
    } catch {
      // ignore invalid remembered username
    }
    generateCaptcha()
  }, [form, generateCaptcha])

  useEffect(() => {
    if (loginFeedback?.reason !== 'account_locked' || !loginFeedback.lockRemainingSeconds || loginFeedback.lockRemainingSeconds <= 0) {
      return
    }

    const timer = window.setInterval(() => {
      setLoginFeedback((current) => {
        if (!current || current.reason !== 'account_locked') {
          return current
        }
        const nextSeconds = Math.max((current.lockRemainingSeconds || 0) - 1, 0)
        return { ...current, lockRemainingSeconds: nextSeconds }
      })
    }, 1000)

    return () => window.clearInterval(timer)
  }, [loginFeedback?.reason, loginFeedback?.lockRemainingSeconds])

  const handleMouseDown = (event: React.MouseEvent) => {
    if (verified) return
    setDragging(true)
    event.preventDefault()
  }

  useEffect(() => {
    if (!dragging) return

    const handleMouseMove = (event: MouseEvent) => {
      if (!trackRef.current) return
      const rect = trackRef.current.getBoundingClientRect()
      const x = Math.max(0, Math.min(event.clientX - rect.left, TRACK_WIDTH - SLIDER_WIDTH))
      setSliderX(x)
    }

    const handleMouseUp = () => {
      setDragging(false)
      const diff = Math.abs(sliderX - targetX)
      if (diff <= TOLERANCE) {
        setVerified(true)
        setLoginFeedback((current) => {
          if (!current || current.reason !== 'captcha_invalid') {
            return current
          }
          return undefined
        })
      } else {
        setLoginFeedback({
          type: 'warning',
          title: '安全验证未通过',
          detail: '请将滑块拖动到虚线框位置后再提交登录。',
          reason: 'captcha_invalid',
          suggestions: ['验证失败后已自动重置滑块，请重新完成验证'],
        })
        window.setTimeout(() => generateCaptcha(), 300)
      }
    }

    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [dragging, generateCaptcha, sliderX, targetX])

  const handleSubmit = async (values: { username: string; password: string }) => {
    if (!verified) {
      setLoginFeedback({
        type: 'warning',
        title: '请先完成安全验证',
        detail: '完成滑块验证后才能提交登录请求。',
        reason: 'captcha_invalid',
        suggestions: ['将滑块拖动到虚线框位置后再登录'],
      })
      return
    }

    setLoading(true)
    try {
      const loginResult = await login(values)
      const currentUser = loginResult.user || await getCurrentUser()
      setInitialState({ currentUser })
      if (remember && values.username) {
        localStorage.setItem(REMEMBER_USER_KEY, btoa(values.username))
      } else {
        localStorage.removeItem(REMEMBER_USER_KEY)
      }
      setLoginFeedback(undefined)
      message.success('登录成功')
      history.replace('/')
    } catch (error) {
      setLoginFeedback(buildLoginFeedback(error))
      generateCaptcha()
    } finally {
      setLoading(false)
    }
  }

  const handleOpenReset = () => {
    setResetStep('request')
    setResetToken('')
    resetForm.resetFields()
    setResetModalOpen(true)
  }

  const handleRequestReset = async () => {
    const values = await resetForm.validateFields()
    try {
      const res = await requestPasswordReset(values.identifier)
      if (res.token) {
        setResetToken(res.token)
        setResetStep('confirm')
        message.success(res.message || '重置 token 已生成')
      } else {
        message.info(res.message || '如果该账号存在，重置链接已生成')
        setResetModalOpen(false)
      }
    } catch {
      // handled by request interceptor
    }
  }

  const handleConfirmReset = async () => {
    const values = await resetForm.validateFields()
    try {
      const res = await confirmPasswordReset(resetToken, values.newPassword)
      message.success(res.message || '密码重置成功')
      setResetModalOpen(false)
    } catch {
      // handled by request interceptor
    }
  }

  const lockedSeconds = loginFeedback?.reason === 'account_locked' ? loginFeedback.lockRemainingSeconds || 0 : 0
  const submitLocked = lockedSeconds > 0
  const feedbackDetail = loginFeedback?.reason === 'account_locked'
    ? (lockedSeconds > 0
      ? `账号已被临时锁定，请 ${formatRemainingTime(lockedSeconds)} 后再试。`
      : '账号锁定已结束，请重新完成滑块验证后再试。')
    : loginFeedback?.detail

  return (
    <div className="app-login-shell">
      <div className="app-login-panel">
        <section className="app-login-form">
          <span className="app-login-form__eyebrow">Sign In</span>
          <Typography.Title level={3} style={{ marginBottom: 12 }}>
            AIOPS 智能运维平台
          </Typography.Title>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 24 }}>
            登录失败超过 {LOGIN_FAILURE_LIMIT} 次会锁定账号 {LOGIN_LOCK_MINUTES} 分钟。遇到无法恢复的情况，可以直接使用“忘记密码”。
          </Typography.Paragraph>

          {loginFeedback && (
            <Alert
              showIcon
              type={loginFeedback.type}
              style={{ marginBottom: 20 }}
              message={loginFeedback.title}
              description={(
                <Space direction="vertical" size={6} style={{ width: '100%' }}>
                  <span>{feedbackDetail}</span>
                  {loginFeedback.suggestions.map((item) => (
                    <span key={item}>- {item}</span>
                  ))}
                </Space>
              )}
            />
          )}

          <Form
            form={form}
            onFinish={handleSubmit}
            size="large"
            autoComplete="off"
            layout="vertical"
            onValuesChange={() => {
              setLoginFeedback((current) => {
                if (current?.reason === 'account_locked' && (current.lockRemainingSeconds || 0) > 0) {
                  return current
                }
                return undefined
              })
            }}
          >
            <Form.Item
              label="用户名"
              name="username"
              rules={[
                { required: true, message: '请输入用户名' },
                { pattern: /^[a-zA-Z0-9_@.\-]+$/, message: '用户名包含非法字符' },
              ]}
              validateTrigger="onBlur"
            >
              <Input
                prefix={<UserOutlined />}
                placeholder="请输入用户名"
                autoComplete="username"
                style={{ height: 50 }}
              />
            </Form.Item>

            <Form.Item
              label="密码"
              name="password"
              rules={[{ required: true, message: '请输入密码' }]}
              validateTrigger="onBlur"
            >
              <Input.Password
                prefix={<LockOutlined />}
                placeholder="请输入密码"
                autoComplete="current-password"
                style={{ height: 50 }}
              />
            </Form.Item>

            <div style={{ marginBottom: 16 }}>
              <div
                ref={trackRef}
                style={{
                  position: 'relative',
                  height: 44,
                  width: '100%',
                  background: verified ? '#f6ffed' : '#f5f5f5',
                  borderRadius: 8,
                  overflow: 'hidden',
                  border: verified ? '1px solid #52c41a' : '1px solid #d9d9d9',
                }}
              >
                <div
                  style={{
                    position: 'absolute',
                    left: 0,
                    top: 0,
                    height: '100%',
                    width: verified ? '100%' : `${(sliderX / TRACK_WIDTH) * 100}%`,
                    background: verified
                      ? 'linear-gradient(90deg, #52c41a22, #52c41a44)'
                      : 'linear-gradient(90deg, #1890ff22, #1890ff44)',
                    transition: dragging ? 'none' : 'width 0.2s',
                  }}
                />
                {!verified && (
                  <div
                    style={{
                      position: 'absolute',
                      left: targetX,
                      top: 0,
                      height: '100%',
                      width: SLIDER_WIDTH,
                      border: '2px dashed #1890ff66',
                      borderRadius: 6,
                      pointerEvents: 'none',
                    }}
                  />
                )}
                <div
                  style={{
                    position: 'absolute',
                    left: verified ? TRACK_WIDTH - SLIDER_WIDTH : sliderX,
                    top: 0,
                    height: '100%',
                    width: SLIDER_WIDTH,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    cursor: verified ? 'default' : dragging ? 'grabbing' : 'grab',
                    background: verified ? '#52c41a' : '#fff',
                    color: verified ? '#fff' : '#999',
                    border: verified ? 'none' : '1px solid #d9d9d9',
                    borderRadius: 8,
                    boxShadow: '0 2px 4px rgba(0,0,0,0.1)',
                    userSelect: 'none',
                    transition: dragging ? 'none' : 'left 0.2s, background 0.2s',
                    zIndex: 2,
                  }}
                  onMouseDown={handleMouseDown}
                >
                  {verified ? '✓' : '→'}
                </div>
                {/* 文字居中 + 对号在最右侧 */}
                <div
                  style={{
                    position: 'absolute',
                    left: 0,
                    right: verified ? SLIDER_WIDTH + 4 : 0,
                    top: 0,
                    height: '100%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: verified ? '#389e0c' : '#999',
                    fontSize: 13,
                    fontWeight: verified ? 600 : 400,
                    pointerEvents: 'none',
                    zIndex: 1,
                  }}
                >
                  {verified ? '验证成功' : '拖动滑块到虚线框内'}
                </div>
              </div>
            </div>

            <div className="app-login-remember">
              <Checkbox checked={remember} onChange={(event) => setRemember(event.target.checked)}>
                记住用户名
              </Checkbox>
              <a onClick={handleOpenReset}>忘记密码？</a>
            </div>

            <Form.Item style={{ marginBottom: 0, marginTop: 16 }}>
              <Button
                type="primary"
                htmlType="submit"
                loading={loading}
                disabled={submitLocked}
                block
                icon={<ArrowRightOutlined />}
                style={{ height: 50, fontSize: 16, fontWeight: 700 }}
              >
                {submitLocked ? `账号锁定中 (${formatRemainingTime(lockedSeconds)})` : '登录'}
              </Button>
            </Form.Item>
          </Form>

          <div className="app-login-footer">
            <span className="app-login-footer__copyright">
              AIOPS V2.1.0 | ops@aiops.company.com
            </span>
          </div>
        </section>
      </div>

      <Modal
        title="找回密码"
        open={resetModalOpen}
        onCancel={() => setResetModalOpen(false)}
        footer={null}
        destroyOnHidden
      >
        {resetStep === 'request' ? (
          <Form form={resetForm} layout="vertical" style={{ marginTop: 16 }}>
            <Form.Item
              label="用户名或邮箱"
              name="identifier"
              rules={[{ required: true, message: '请输入用户名或邮箱' }]}
            >
              <Input placeholder="请输入用户名或邮箱" />
            </Form.Item>
            <Button type="primary" block onClick={handleRequestReset}>
              获取重置码
            </Button>
          </Form>
        ) : (
          <Form form={resetForm} layout="vertical" style={{ marginTop: 16 }}>
            <Form.Item
              label="新密码"
              name="newPassword"
              rules={[
                { required: true, message: '请输入新密码' },
                { min: 6, message: '密码长度至少 6 位' },
              ]}
            >
              <Input.Password placeholder="请输入新密码" />
            </Form.Item>
            <Form.Item
              label="确认新密码"
              name="confirmPassword"
              dependencies={['newPassword']}
              rules={[
                { required: true, message: '请再次输入新密码' },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    if (!value || getFieldValue('newPassword') === value) {
                      return Promise.resolve()
                    }
                    return Promise.reject(new Error('两次输入的密码不一致'))
                  },
                }),
              ]}
            >
              <Input.Password placeholder="请再次输入新密码" />
            </Form.Item>
            <Button type="primary" block onClick={handleConfirmReset}>
              重置密码
            </Button>
          </Form>
        )}
      </Modal>
    </div>
  )
}

export default LoginPage
