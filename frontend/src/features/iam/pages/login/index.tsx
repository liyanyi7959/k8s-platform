import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Checkbox, Form, Input, Modal, Space, Tooltip, Typography, message } from 'antd'
import {
  ArrowRightOutlined,
  ClusterOutlined,
  InfoCircleOutlined,
  LockOutlined,
  SafetyCertificateOutlined,
  ThunderboltOutlined,
  UserOutlined,
} from '@ant-design/icons'
import {
  getCaptcha,
  login,
  requestPasswordReset,
  confirmPasswordReset,
} from '@/features/iam/api'

const REMEMBER_USER_KEY = 'aiops_remembered_user'
const TRACK_WIDTH = 280
const SLIDER_WIDTH = 40
const TOLERANCE = 6
const LOGIN_FAILURE_LIMIT = 5
const LOGIN_LOCK_MINUTES = 15
const BRAND_LOGO_SRC = '/brand/aiops-mark.svg'

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
  remainingAttempts?: number
  maxAttempts?: number
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
      suggestions:
        suggestions.length > 0
          ? suggestions
          : ['请等待锁定结束后再试', '如已忘记密码，可使用忘记密码功能重置'],
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
    const detail =
      remainingAttempts > 0 && maxAttempts > 0
        ? `用户名或密码不正确。当前还可尝试 ${remainingAttempts} 次，连续失败 ${maxAttempts} 次后账号会被临时锁定。`
        : baseMessage

    return {
      type: 'error',
      title: '账号或密码不正确',
      detail,
      reason: 'invalid_credentials',
      canResetPassword: payload.can_reset_password !== false,
      remainingAttempts,
      maxAttempts,
      suggestions:
        suggestions.length > 0
          ? suggestions
          : ['请确认用户名、密码和大小写是否正确', '如已忘记密码，可使用忘记密码功能重置'],
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
  const [captchaError, setCaptchaError] = useState<string>('')
  const [form] = Form.useForm()

  const [targetX, setTargetX] = useState(0)
  const [trackWidth, setTrackWidth] = useState(TRACK_WIDTH)
  const [trackPixelWidth, setTrackPixelWidth] = useState(TRACK_WIDTH)
  const [sliderX, setSliderX] = useState(0)
  const [dragging, setDragging] = useState(false)
  const [verified, setVerified] = useState(false)
  const [captchaEnabled, setCaptchaEnabled] = useState(false)
  const [captchaToken, setCaptchaToken] = useState('')
  const trackRef = useRef<HTMLDivElement>(null)
  const sliderXRef = useRef(0)
  const dragStateRef = useRef<{ pointerId: number; startClientX: number; startSliderX: number } | null>(null)

  const [resetModalOpen, setResetModalOpen] = useState(false)
  const [resetStep, setResetStep] = useState<'request' | 'confirm'>('request')
  const [resetToken, setResetToken] = useState('')
  const [resetForm] = Form.useForm()

  const sourceSliderRange = Math.max(1, trackWidth - SLIDER_WIDTH)
  const maxSliderX = Math.max(0, trackPixelWidth - SLIDER_WIDTH)
  const targetSliderX = Math.max(0, Math.min(maxSliderX, (targetX / sourceSliderRange) * maxSliderX))
  const tolerancePixels = Math.max(4, (TOLERANCE / sourceSliderRange) * maxSliderX)
  const captchaSubmissionX = maxSliderX > 0
    ? Math.round((sliderX / maxSliderX) * sourceSliderRange)
    : 0

  const setSliderPosition = useCallback((position: number) => {
    const next = Math.max(0, Math.min(position, maxSliderX))
    sliderXRef.current = next
    setSliderX(next)
  }, [maxSliderX])

  const resetSliderPosition = useCallback(() => {
    sliderXRef.current = 0
    setSliderX(0)
  }, [])

  useEffect(() => {
    const updateTrackWidth = () => {
      const width = trackRef.current?.getBoundingClientRect().width
      if (width) setTrackPixelWidth(width)
    }
    updateTrackWidth()
    const observer = typeof ResizeObserver === 'undefined' || !trackRef.current
      ? undefined
      : new ResizeObserver(updateTrackWidth)
    if (observer && trackRef.current) observer.observe(trackRef.current)
    return () => observer?.disconnect()
  }, [])

  const generateLocalCaptcha = useCallback(() => {
    const minTarget = 24
    const maxTarget = TRACK_WIDTH - SLIDER_WIDTH - 24
    setCaptchaEnabled(false)
    setCaptchaToken('')
    setTrackWidth(TRACK_WIDTH)
    setTargetX(minTarget + Math.floor(Math.random() * (maxTarget - minTarget + 1)))
    resetSliderPosition()
    setVerified(false)
  }, [resetSliderPosition])

  const refreshCaptcha = useCallback(async () => {
    try {
      const challenge = await getCaptcha()
      if (challenge?.enabled && challenge.token && typeof challenge.target_x === 'number') {
        setCaptchaEnabled(true)
        setCaptchaToken(challenge.token)
        setTrackWidth(challenge.track_width || TRACK_WIDTH)
        setTargetX(challenge.target_x)
        resetSliderPosition()
        setVerified(false)
        return
      }
    } catch {
      // fallback to local-only captcha when backend challenge is unavailable
    }
    generateLocalCaptcha()
  }, [generateLocalCaptcha, resetSliderPosition])

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
    void refreshCaptcha()
  }, [form, refreshCaptcha])

  useEffect(() => {
    if (
      loginFeedback?.reason !== 'account_locked' ||
      !loginFeedback.lockRemainingSeconds ||
      loginFeedback.lockRemainingSeconds <= 0
    ) {
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

  const completeCaptcha = useCallback(() => {
    const diff = Math.abs(sliderXRef.current - targetSliderX)
    if (diff <= tolerancePixels) {
      setVerified(true)
      setCaptchaError('')
      setLoginFeedback((current) => current?.reason === 'captcha_invalid' ? undefined : current)
      return
    }
    setCaptchaError('验证失败，请将滑块拖到虚线目标框内')
    window.setTimeout(() => {
      void refreshCaptcha()
    }, 300)
  }, [refreshCaptcha, targetSliderX, tolerancePixels])

  const handleSliderPointerDown = (event: React.PointerEvent<HTMLButtonElement>) => {
    if (verified) return
    event.preventDefault()
    event.currentTarget.setPointerCapture(event.pointerId)
    dragStateRef.current = {
      pointerId: event.pointerId,
      startClientX: event.clientX,
      startSliderX: sliderXRef.current,
    }
    setDragging(true)
  }

  const handleSliderPointerMove = (event: React.PointerEvent<HTMLButtonElement>) => {
    const dragState = dragStateRef.current
    if (!dragState || dragState.pointerId !== event.pointerId) return
    setSliderPosition(dragState.startSliderX + event.clientX - dragState.startClientX)
  }

  const handleSliderPointerUp = (event: React.PointerEvent<HTMLButtonElement>) => {
    const dragState = dragStateRef.current
    if (!dragState || dragState.pointerId !== event.pointerId) return
    dragStateRef.current = null
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    setDragging(false)
    completeCaptcha()
  }

  const handleSliderPointerCancel = () => {
    dragStateRef.current = null
    setDragging(false)
    resetSliderPosition()
  }

  const handleSliderKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    if (verified) return
    const step = event.shiftKey ? 24 : 8
    if (event.key === 'ArrowRight') {
      event.preventDefault()
      setSliderPosition(sliderXRef.current + step)
    } else if (event.key === 'ArrowLeft') {
      event.preventDefault()
      setSliderPosition(sliderXRef.current - step)
    } else if (event.key === 'Home') {
      event.preventDefault()
      setSliderPosition(0)
    } else if (event.key === 'End') {
      event.preventDefault()
      setSliderPosition(maxSliderX)
    } else if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      completeCaptcha()
    }
  }

  const handleSubmit = async (values: { username: string; password: string }) => {
    if (!verified) {
      setCaptchaError('请先完成滑块验证')
      return
    }

    if (captchaEnabled && !captchaToken) {
      setCaptchaError('验证码已过期，请重新完成验证')
      void refreshCaptcha()
      return
    }

    setLoading(true)
    try {
      await login({
        ...values,
        captchaToken: captchaEnabled ? captchaToken : undefined,
        captchaX: captchaEnabled ? captchaSubmissionX : undefined,
      })
      if (remember && values.username) {
        localStorage.setItem(REMEMBER_USER_KEY, btoa(values.username))
      } else {
        localStorage.removeItem(REMEMBER_USER_KEY)
      }
      setLoginFeedback(undefined)
      setCaptchaError('')
      message.success('登录成功')
      // 登录页运行在初始状态 Provider 外；完整跳转能让首页重新拉取用户信息，避免读取空 model Context。
      window.location.replace('/')
    } catch (error) {
      setLoginFeedback(buildLoginFeedback(error))
      void refreshCaptcha()
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

  const lockedSeconds =
    loginFeedback?.reason === 'account_locked' ? loginFeedback.lockRemainingSeconds || 0 : 0
  const submitLocked = lockedSeconds > 0
  const feedbackDetail =
    loginFeedback?.reason === 'account_locked'
      ? lockedSeconds > 0
        ? `账号已被临时锁定，请 ${formatRemainingTime(lockedSeconds)} 后再试。`
        : '账号锁定已结束，请重新完成滑块验证后再试。'
      : loginFeedback?.detail
  const captchaHint = captchaError || (verified ? '安全验证已通过，可以提交登录。' : '拖动滑块完成安全校验。')
  const captchaHintTone = captchaError ? 'error' : verified ? 'success' : 'default'
  const feedbackMessage =
    loginFeedback?.reason === 'invalid_credentials' &&
    (loginFeedback.remainingAttempts || 0) > 0 &&
    (loginFeedback.maxAttempts || 0) > 0
      ? `账号或密码不正确，还可尝试 ${loginFeedback.remainingAttempts} 次。`
      : feedbackDetail
  const feedbackAssist = loginFeedback?.suggestions[0]

  return (
    <div className="app-login-shell">
      <aside className="app-login-console" aria-label="AIOPS 平台概览">
        <div className="app-login-console__topline">
          <div className="app-login-console__identity">
            <img src={BRAND_LOGO_SRC} alt="" draggable={false} />
            <span>AIOPS</span>
          </div>
          <span className="app-login-console__environment">CONTROL PLANE</span>
        </div>

        <div className="app-login-console__hero">
          <span className="app-login-console__eyebrow">AIOPS / CONTROL PLANE</span>
          <h1>智能运维控制平台</h1>
          <p>集中管理集群、告警与自动化任务，让运维工作保持清晰、可靠和可追溯。</p>
        </div>

        <div className="app-login-console__activity">
          <span className="app-login-console__activity-mark" aria-hidden="true">
            <i />
            <b />
            <em />
          </span>
          <span>
            <strong>实时态势感知</strong>
            <small>持续汇聚运行信号与关键事件</small>
          </span>
        </div>

        <div className="app-login-console__telemetry" aria-hidden="true">
          <div className="app-login-console__telemetry-head">
            <span>PLATFORM TELEMETRY</span>
            <i />
            <b>LIVE</b>
          </div>
          <div className="app-login-console__signal">
            <span className="app-login-console__signal-node app-login-console__signal-node--core" />
            <span className="app-login-console__signal-node app-login-console__signal-node--one" />
            <span className="app-login-console__signal-node app-login-console__signal-node--two" />
            <span className="app-login-console__signal-node app-login-console__signal-node--three" />
            <span className="app-login-console__signal-line app-login-console__signal-line--one" />
            <span className="app-login-console__signal-line app-login-console__signal-line--two" />
            <span className="app-login-console__signal-line app-login-console__signal-line--three" />
          </div>
        </div>

        <div className="app-login-console__facts">
          <div>
            <ClusterOutlined />
            <span>资源统一接入</span>
            <strong>集群与工作负载</strong>
          </div>
          <div>
            <ThunderboltOutlined />
            <span>自动化闭环</span>
            <strong>检测 · 研判 · 执行</strong>
          </div>
          <div>
            <SafetyCertificateOutlined />
            <span>安全访问控制</span>
            <strong>身份与权限审计</strong>
          </div>
        </div>

        <div className="app-login-console__foot">AIOPS / OPERATIONS INTELLIGENCE</div>
      </aside>
      <div className="app-login-panel">
        <section className="app-login-form">
          <div className="app-login-form__surface">
          <div className="app-login-form__brand">
            <div className="app-login-form__brand-top">
              <img
                className="app-login-form__logo"
                src={BRAND_LOGO_SRC}
                alt="AIOPS platform logo"
                draggable={false}
              />
              <span className="app-login-form__eyebrow">AIOPS Control Plane</span>
            </div>
            <Typography.Title level={3} className="app-login-form__title">
              AIOPS 智能运维平台
            </Typography.Title>
          </div>
          <Typography.Paragraph className="app-login-form__subtitle">
            统一进入告警、日志、集群与自动化协同的运维控制台。
          </Typography.Paragraph>

          <Form
            form={form}
            onFinish={handleSubmit}
            size="large"
            autoComplete="off"
            layout="vertical"
            onValuesChange={() => {
              setLoginFeedback((current) => {
                if (
                  current?.reason === 'account_locked' &&
                  (current.lockRemainingSeconds || 0) > 0
                ) {
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

            <div className="app-login-captcha">
              <div
                ref={trackRef}
                className={`app-login-captcha__track${dragging ? ' app-login-captcha__track--dragging' : ''}${verified ? ' app-login-captcha__track--success' : ''}${captchaError ? ' app-login-captcha__track--error' : ''}`}
              >
                {!verified && (
                  <span
                    className="app-login-captcha__target"
                    style={{ transform: `translateX(${targetSliderX}px)` }}
                    aria-hidden="true"
                  />
                )}
                <span className="app-login-captcha__label" aria-live="polite">
                  {verified ? '安全验证通过' : '拖动滑块至目标位置'}
                </span>
                <button
                  type="button"
                  className={`app-login-captcha__handle${dragging ? ' app-login-captcha__handle--dragging' : ''}${verified ? ' app-login-captcha__handle--success' : ''}`}
                  style={{ transform: `translateX(${verified ? maxSliderX : sliderX}px)` }}
                  onPointerDown={handleSliderPointerDown}
                  onPointerMove={handleSliderPointerMove}
                  onPointerUp={handleSliderPointerUp}
                  onPointerCancel={handleSliderPointerCancel}
                  onKeyDown={handleSliderKeyDown}
                  aria-label={verified ? '安全验证已通过' : '拖动滑块完成安全验证'}
                  aria-valuemin={0}
                  aria-valuemax={Math.round(maxSliderX)}
                  aria-valuenow={Math.round(verified ? maxSliderX : sliderX)}
                  aria-valuetext={verified ? '安全验证已通过' : '使用左右方向键移动，按回车确认'}
                  role="slider"
                  disabled={verified}
                >
                  <span aria-hidden="true">{verified ? '✓' : '→'}</span>
                </button>
              </div>
              <div className={`app-login-captcha__hint app-login-captcha__hint--${captchaHintTone}`}>
                {captchaHint}
              </div>
            </div>

            <div className="app-login-remember">
              <Checkbox checked={remember} onChange={(event) => setRemember(event.target.checked)}>
                记住用户名
              </Checkbox>
              <Space>
                <a onClick={handleOpenReset}>忘记密码？</a>
                <Tooltip
                  title={`连续失败 ${LOGIN_FAILURE_LIMIT} 次后将锁定 ${LOGIN_LOCK_MINUTES} 分钟，可通过“忘记密码”重置。`}
                >
                  <InfoCircleOutlined style={{ color: '#bfbfbf', fontSize: 12, cursor: 'help' }} />
                </Tooltip>
              </Space>
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

            {loginFeedback && (
              <div className={`app-login-feedback app-login-feedback--${loginFeedback.type}`}>
                <span className="app-login-feedback__text">{feedbackMessage}</span>
                {feedbackAssist ? (
                  <Tooltip title={feedbackAssist}>
                    <InfoCircleOutlined className="app-login-feedback__icon" />
                  </Tooltip>
                ) : null}
              </div>
            )}
          </Form>
          </div>

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
