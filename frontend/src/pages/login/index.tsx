import React, { useState, useEffect, useRef, useCallback } from 'react'
import { history, useModel } from '@umijs/max'
import { Button, Form, Input, Typography, message, Checkbox, Modal } from 'antd'
import { ArrowRightOutlined, LockOutlined, UserOutlined } from '@ant-design/icons'
import { getCurrentUser, login, requestPasswordReset, confirmPasswordReset } from '@/services/auth'

const REMEMBER_USER_KEY = 'aiops_remembered_user'
const TRACK_WIDTH = 280
const SLIDER_WIDTH = 40
const TOLERANCE = 6

const LoginPage: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [remember, setRemember] = useState(false)
  const { setInitialState } = useModel('@@initialState')
  const [form] = Form.useForm()

  // 滑块验证码状态（纯前端实现，不依赖 Redis）
  const [targetX, setTargetX] = useState(0)
  const [sliderX, setSliderX] = useState(0)
  const [dragging, setDragging] = useState(false)
  const [verified, setVerified] = useState(false)
  const trackRef = useRef<HTMLDivElement>(null)

  // 找回密码弹窗
  const [resetModalOpen, setResetModalOpen] = useState(false)
  const [resetStep, setResetStep] = useState<'request' | 'confirm'>('request')
  const [resetToken, setResetToken] = useState('')
  const [resetForm] = Form.useForm()

  // 生成随机目标位置
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
      // ignore
    }
    generateCaptcha()
  }, [form, generateCaptcha])

  // 滑块拖拽
  const handleMouseDown = (e: React.MouseEvent) => {
    if (verified) return
    setDragging(true)
    e.preventDefault()
  }

  useEffect(() => {
    if (!dragging) return
    const handleMouseMove = (e: MouseEvent) => {
      if (!trackRef.current) return
      const rect = trackRef.current.getBoundingClientRect()
      const x = Math.max(0, Math.min(e.clientX - rect.left, TRACK_WIDTH - SLIDER_WIDTH))
      setSliderX(x)
    }
    const handleMouseUp = () => {
      setDragging(false)
      // 松手时校验
      const diff = Math.abs(sliderX - targetX)
      if (diff <= TOLERANCE) {
        setVerified(true)
      } else {
        message.error('验证失败，请重试')
        // 失败后重置
        setTimeout(() => generateCaptcha(), 300)
      }
    }
    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [dragging, sliderX, targetX, generateCaptcha])

  const handleSubmit = async (values: { username: string; password: string }) => {
    if (!verified) {
      message.warning('请先完成滑块验证')
      return
    }
    setLoading(true)
    try {
      await login(values)
      const currentUser = await getCurrentUser()
      setInitialState({ currentUser })
      if (remember && values.username) {
        localStorage.setItem(REMEMBER_USER_KEY, btoa(values.username))
      } else {
        localStorage.removeItem(REMEMBER_USER_KEY)
      }
      message.success('登录成功')
      history.push('/')
    } catch {
      // 登录失败，重置滑块
      generateCaptcha()
    } finally {
      setLoading(false)
    }
  }

  // ── 找回密码 ──
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
      // error handled by interceptor
    }
  }

  const handleConfirmReset = async () => {
    const values = await resetForm.validateFields()
    try {
      const res = await confirmPasswordReset(resetToken, values.newPassword)
      message.success(res.message || '密码重置成功')
      setResetModalOpen(false)
    } catch {
      // error handled by interceptor
    }
  }

  return (
    <div className="app-login-shell">
      <div className="app-login-panel">
        <section className="app-login-form">
          <span className="app-login-form__eyebrow">Sign In</span>
          <Typography.Title level={3} style={{ marginBottom: 24 }}>
            AIOPS 智能运维平台
          </Typography.Title>

          <Form
            form={form}
            onFinish={handleSubmit}
            size="large"
            autoComplete="off"
            layout="vertical"
          >
            <Form.Item
              label="用户名"
              name="username"
              rules={[
                { required: true, message: '请输入用户名' },
                { pattern: /^[a-zA-Z0-9_@.\\-]+$/, message: '用户名包含非法字符' },
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
              rules={[
                { required: true, message: '请输入密码' },
                { min: 6, message: '密码长度至少 6 位' },
              ]}
              validateTrigger="onBlur"
            >
              <Input.Password
                prefix={<LockOutlined />}
                placeholder="请输入密码"
                autoComplete="current-password"
                style={{ height: 50 }}
              />
            </Form.Item>

            {/* 滑块验证码（纯前端实现） */}
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
                {/* 进度条 */}
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
                {/* 目标区域提示（半透明虚线框） */}
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
                {/* 滑块按钮 */}
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
                  }}
                  onMouseDown={handleMouseDown}
                >
                  {verified ? '✓' : '→'}
                </div>
                {/* 提示文字 */}
                <div
                  style={{
                    position: 'absolute',
                    right: 12,
                    top: '50%',
                    transform: 'translateY(-50%)',
                    color: verified ? '#52c41a' : '#999',
                    fontSize: 13,
                    pointerEvents: 'none',
                  }}
                >
                  {verified ? '验证成功' : '拖动滑块到虚线框内'}
                </div>
              </div>
            </div>

            <div className="app-login-remember">
              <Checkbox checked={remember} onChange={(e) => setRemember(e.target.checked)}>
                记住用户名
              </Checkbox>
              <a onClick={handleOpenReset}>忘记密码？</a>
            </div>

            <Form.Item style={{ marginBottom: 0, marginTop: 16 }}>
              <Button
                type="primary"
                htmlType="submit"
                loading={loading}
                block
                icon={<ArrowRightOutlined />}
                style={{ height: 50, fontSize: 16, fontWeight: 700 }}
              >
                登录
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

      {/* 找回密码弹窗 */}
      <Modal
        title="找回密码"
        open={resetModalOpen}
        onCancel={() => setResetModalOpen(false)}
        footer={null}
        destroyOnClose
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