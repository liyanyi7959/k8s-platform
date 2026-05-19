<template>
  <div class="login-page" :class="{ 'is-light': effectiveTheme === 'light' }">
    <!-- Theme toggle -->
    <button class="theme-toggle" :title="themeLabel" @click="toggleTheme">
      <svg v-if="effectiveTheme === 'dark'" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      <svg v-else xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
    </button>

    <!-- ── Full-screen immersive background ── -->
    <div class="login-bg" aria-hidden="true">
      <div class="bg-orb bg-orb-1"></div>
      <div class="bg-orb bg-orb-2"></div>
      <div class="bg-orb bg-orb-3"></div>
      <div class="bg-grid"></div>
      <div class="bg-noise"></div>
    </div>

    <!-- ── Left: Brand Region (no container, breathes with bg) ── -->
    <div class="login-brand">
      <!-- 散落几何装饰 -->
      <div class="geo-decor geo-circle geo-circle-1" />
      <div class="geo-decor geo-circle geo-circle-2" />
      <div class="geo-decor geo-circle geo-circle-3" />
      <div class="geo-decor geo-diamond geo-diamond-1" />
      <div class="geo-decor geo-diamond geo-diamond-2" />
      <div class="geo-decor geo-dot geo-dot-1" />
      <div class="geo-decor geo-dot geo-dot-2" />
      <div class="geo-decor geo-dot geo-dot-3" />

      <div class="brand-inner">
        <!-- Pure CSS Logo Mark — Linux Terminal + 枢纽网络 -->
        <div class="logo-mark">
          <div class="logo-hexagon">
            <svg viewBox="0 0 80 80" class="logo-svg">
              <defs>
                <linearGradient id="logo-grad" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stop-color="#6366f1" />
                  <stop offset="50%" stop-color="#818cf8" />
                  <stop offset="100%" stop-color="#06b6d4" />
                </linearGradient>
                <linearGradient id="logo-grad2" x1="0%" y1="10%" x2="100%" y2="90%">
                  <stop offset="0%" stop-color="#818cf8" />
                  <stop offset="100%" stop-color="#22d3ee" />
                </linearGradient>
              </defs>

              <!-- 外层科技六边形 -->
              <path d="M40 4 L69 21 Q72 23 72 26.5 V53.5 Q72 57 69 59 L40 76 L11 59 Q8 57 8 53.5 V26.5 Q8 23 11 21 Z"
                    fill="url(#logo-grad)" opacity="0.1"/>
              <path d="M40 4 L69 21 Q72 23 72 26.5 V53.5 Q72 57 69 59 L40 76 L11 59 Q8 57 8 53.5 V26.5 Q8 23 11 21 Z"
                    fill="none" stroke="url(#logo-grad)" stroke-width="1" opacity="0.3"/>

              <!-- 枢纽网络：6节点 + 连线 -->
              <g opacity="0.3" stroke="url(#logo-grad2)" stroke-width="0.8">
                <line x1="40" y1="40" x2="40" y2="14"/>
                <line x1="40" y1="40" x2="62.5" y2="27"/>
                <line x1="40" y1="40" x2="62.5" y2="53"/>
                <line x1="40" y1="40" x2="40" y2="66"/>
                <line x1="40" y1="40" x2="17.5" y2="53"/>
                <line x1="40" y1="40" x2="17.5" y2="27"/>
              </g>
              <g fill="url(#logo-grad2)" opacity="0.55">
                <circle cx="40" cy="14" r="3"/>
                <circle cx="62.5" cy="27" r="3"/>
                <circle cx="62.5" cy="53" r="3"/>
                <circle cx="40" cy="66" r="3"/>
                <circle cx="17.5" cy="53" r="3"/>
                <circle cx="17.5" cy="27" r="3"/>
              </g>

              <!-- 旋转虚线环 -->
              <circle cx="40" cy="40" r="30" fill="none" stroke="url(#logo-grad)" stroke-width="1" opacity="0.15"/>
              <circle cx="40" cy="40" r="30" fill="none" stroke="url(#logo-grad)" stroke-width="1.2" opacity="0.45" stroke-dasharray="5 10" class="logo-ring-rotate"/>

              <!-- 核心：终端提示符 >_ -->
              <g transform="translate(24, 30)">
                <path d="M0 0 L10.5 8 L0 16" fill="none" stroke="url(#logo-grad)" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round"/>
                <line x1="14" y1="16" x2="30" y2="16" stroke="url(#logo-grad2)" stroke-width="3" stroke-linecap="round">
                  <animate attributeName="opacity" values="1;0.2;1" dur="1.2s" repeatCount="indefinite"/>
                </line>
              </g>

              <!-- 中心枢纽亮点 -->
              <circle cx="40" cy="40" r="3.5" fill="url(#logo-grad)" opacity="0.35"/>
            </svg>
          </div>
        </div>

        <h1 class="brand-title">星枢</h1>
        <p class="brand-subtitle">XINGSHU · K8S Management Platform</p>

        <!-- ─── 大型运维主题插画 (SVG显示器 + HTML浮动卡片) ─── -->
        <div class="brand-illustration">
          <!-- SVG: 只画显示器 -->
          <svg viewBox="0 0 480 300" fill="none" xmlns="http://www.w3.org/2000/svg" class="illo-svg">
            <defs>
              <linearGradient id="il-screen" x1="160" y1="80" x2="340" y2="220" gradientUnits="userSpaceOnUse">
                <stop offset="0%" stop-color="#1e1b4b"/>
                <stop offset="100%" stop-color="#312e81"/>
              </linearGradient>
              <filter id="il-shadow" x="-10%" y="-10%" width="120%" height="120%">
                <feDropShadow dx="0" dy="3" stdDeviation="6" flood-color="#6366f1" flood-opacity="0.13"/>
              </filter>
            </defs>
            <ellipse cx="240" cy="268" rx="140" ry="11" fill="#e0e7ff" opacity="0.45"/>
            <rect x="226" y="232" width="28" height="26" rx="4" fill="#c7d2fe"/>
            <rect x="208" y="254" width="64" height="8" rx="4" fill="#a5b4fc"/>
            <rect x="130" y="58" width="220" height="175" rx="12" fill="#e0e7ff" filter="url(#il-shadow)"/>
            <rect x="130" y="58" width="220" height="175" rx="12" fill="none" stroke="#c7d2fe" stroke-width="1.5"/>
            <rect x="140" y="68" width="200" height="148" rx="7" fill="url(#il-screen)"/>
            <rect x="148" y="76" width="184" height="130" rx="5" fill="#1e1b4b" opacity="0.9"/>
            <rect x="148" y="76" width="184" height="20" rx="5" fill="#312e81"/>
            <circle cx="160" cy="86" r="3.5" fill="#f87171" opacity="0.8"/>
            <circle cx="170" cy="86" r="3.5" fill="#fbbf24" opacity="0.8"/>
            <circle cx="180" cy="86" r="3.5" fill="#34d399" opacity="0.8"/>
            <text x="240" y="90" text-anchor="middle" font-size="8" fill="#a5b4fc" font-family="monospace" opacity="0.7">root@xingshu:~</text>
            <text x="156" y="112" font-size="9" fill="#34d399" font-family="monospace" opacity="0.9">$ kubectl get nodes</text>
            <text x="156" y="124" font-size="8" fill="#a5b4fc" font-family="monospace" opacity="0.7">NAME        STATUS   ROLES</text>
            <text x="156" y="135" font-size="8" fill="#e0e7ff" font-family="monospace" opacity="0.6">master-01   Ready    control-plane</text>
            <text x="156" y="146" font-size="8" fill="#e0e7ff" font-family="monospace" opacity="0.6">worker-01   Ready    &lt;none&gt;</text>
            <text x="156" y="157" font-size="8" fill="#e0e7ff" font-family="monospace" opacity="0.6">worker-02   Ready    &lt;none&gt;</text>
            <text x="156" y="172" font-size="9" fill="#34d399" font-family="monospace" opacity="0.9">$</text>
            <rect x="166" y="164" width="6" height="10" rx="1" fill="#34d399" opacity="0.8">
              <animate attributeName="opacity" values="0.8;0;0.8" dur="1.1s" repeatCount="indefinite"/>
            </rect>
          </svg>

          <!-- HTML 浮动卡片: 绝对定位，不受 SVG transform 影响 -->
          <div class="illo-card illo-card-1">
            <div class="illo-card-icon illo-card-icon--blue">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><rect x="1" y="1" width="12" height="12" rx="2.5" stroke="#818cf8" stroke-width="1.5"/><rect x="4" y="4" width="6" height="6" rx="1" fill="#818cf8" opacity="0.6"/></svg>
            </div>
            <div class="illo-card-text">
              <span class="illo-card-title">集群概览</span>
              <span class="illo-card-sub">3 节点 · Ready</span>
            </div>
            <div class="illo-card-bar">
              <div class="illo-card-bar-bg"><div class="illo-card-bar-fill" style="width:72%"></div></div>
              <span class="illo-card-bar-label">72%</span>
            </div>
          </div>

          <div class="illo-card illo-card-2">
            <div class="illo-card-icon illo-card-icon--green">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M3 7.5 L5.5 10 L11 4" stroke="#22c55e" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
            </div>
            <div class="illo-card-text">
              <span class="illo-card-title">集群健康</span>
              <span class="illo-card-sub">所有节点正常</span>
            </div>
            <div class="illo-card-dots">
              <span class="illo-dot illo-dot--green"></span>
              <span class="illo-dot illo-dot--green"></span>
              <span class="illo-dot illo-dot--blue"></span>
            </div>
          </div>

          <div class="illo-card illo-card-3">
            <div class="illo-card-icon illo-card-icon--amber">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M5 7 L7 5 L9 7 M7 5 V11" stroke="#f59e0b" stroke-width="1.8" stroke-linecap="round"/></svg>
            </div>
            <div class="illo-card-text">
              <span class="illo-card-title">工作负载</span>
              <span class="illo-card-sub">12 Pods · 运行中</span>
            </div>
            <div class="illo-card-bar">
              <div class="illo-card-bar-bg"><div class="illo-card-bar-fill illo-card-bar-fill--animated"></div></div>
            </div>
          </div>
        </div>

        <p class="brand-desc">聚焦集群接入、资源编排、权限审计与运行观测，面向 K8S 场景的统一管理平台。</p>

        <div class="brand-features">
          <div class="feature-pill" v-for="f in features" :key="f.label">
            <el-icon :size="14"><component :is="f.icon" /></el-icon>
            <span>{{ f.label }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ── Right: Glassmorphism Form Card ── -->
    <div class="login-form-side">
      <div class="form-card">
        <div class="form-header">
          <h2 class="form-title">欢迎使用</h2>
          <p class="form-sub">登录星枢K8S管理平台</p>
        </div>

        <el-form class="login-form" :model="form" label-position="top" @submit.prevent>
          <el-form-item>
            <el-input v-model="form.username" autocomplete="username" placeholder="账号" :prefix-icon="UserIcon" size="large" />
          </el-form-item>
          <el-form-item>
            <el-input v-model="form.password" type="password" autocomplete="current-password" show-password placeholder="密码" :prefix-icon="LockIcon" size="large" @keyup.enter="sliderSuccess && onLogin()" />
          </el-form-item>

          <div
            ref="sliderRef"
            class="slider"
            :class="{ 'is-dragging': sliderDragging, 'is-success': sliderSuccess }"
            @mousedown.prevent="onSliderDown"
            @touchstart.passive="onSliderTouchStart"
          >
            <div class="slider-track" />
            <div class="slider-progress" :style="{ width: `${sliderX + SLIDER_HANDLE_WIDTH}px` }" />
            <div class="slider-text">
              <span class="slider-text-inner">{{ sliderText }}</span>
            </div>
            <div class="slider-handle" :style="{ transform: `translateX(${sliderX}px)` }">
              <span class="slider-handle-icon">
                <template v-if="sliderSuccess">
                  <span class="shell-dollar">$</span><span class="shell-caret">_</span>
                </template>
                <template v-else>/</template>
              </span>
            </div>
          </div>

          <div class="login-options">
            <label class="remember-toggle" @click.prevent="rememberMe = !rememberMe">
              <span class="toggle-track" :class="{ 'is-on': rememberMe }">
                <span class="toggle-thumb" />
              </span>
              <span class="toggle-label">记住密码</span>
              <el-tooltip content="密码将加密存储在本地，7 天后自动过期清除" placement="top" :show-after="400">
                <el-icon class="toggle-hint"><QuestionFilled /></el-icon>
              </el-tooltip>
            </label>
            <el-link type="primary" :underline="false" @click="forgotVisible = true">忘记密码</el-link>
          </div>

          <el-button class="login-btn" type="primary" :loading="loading" :disabled="!sliderSuccess" @click="onLogin" size="large">登 录</el-button>
        </el-form>

        <div class="login-footer">
          <span>Powered by 星枢K8S管理平台 · v1.0</span>
        </div>
      </div>
    </div>
  </div>

  <el-dialog v-model="forgotVisible" title="忘记密码" class="dialog-sm" append-to-body>
    <div class="forgot-body">
      请联系系统管理员重置密码。
    </div>
    <template #footer>
      <el-button type="primary" @click="forgotVisible = false">知道了</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, reactive, ref, computed, markRaw, type Component } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { User as UserIcon, Lock as LockIcon, Monitor, SetUp, Box, Document, QuestionFilled } from '@element-plus/icons-vue'
import { useUserStore } from '@/app/store/user'
import { useTheme } from '@/app/composables/useTheme'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import { clearToken, getSessionUser, getToken, isTokenExpired } from '@/shared/utils/auth'
import type { ApiError } from '@/shared/utils/error'

const features: { label: string; icon: Component }[] = [
  { label: '集群接入', icon: markRaw(Monitor) },
  { label: '资源编排', icon: markRaw(SetUp) },
  { label: '工作负载', icon: markRaw(Box) },
  { label: '权限审计', icon: markRaw(Document) },
]

const { effectiveTheme, themeLabel, toggleTheme } = useTheme()

const userStore = useUserStore()
const router = useRouter()
const route = useRoute()

const form = reactive({ username: '', password: '' })
const loading = ref(false)

const REMEMBER_KEY = 'xshu_login_remember'
const REMEMBER_USER_KEY = 'xshu_login_username'
const REMEMBER_PASS_KEY = 'xshu_login_pass_enc'
const REMEMBER_EXPIRES_KEY = 'xshu_login_expires'
const REMEMBER_TTL_MS = 7 * 24 * 60 * 60 * 1000 // 7 天过期

// ── 简单混淆加密/解密（前端仅防止明文暴露，非安全加密）──
const SALT = 'xshu@2026#'
function encodePass(raw: string): string {
  try {
    return btoa(encodeURIComponent(SALT + raw).replace(/%([0-9A-F]{2})/g, (_, p1) => String.fromCharCode(parseInt(p1, 16))))
  } catch { return '' }
}
function decodePass(encoded: string): string {
  try {
    const decoded = decodeURIComponent(
      Array.from(atob(encoded), c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2)).join('')
    )
    return decoded.startsWith(SALT) ? decoded.slice(SALT.length) : ''
  } catch { return '' }
}
function isRememberExpired(): boolean {
  const exp = localStorage.getItem(REMEMBER_EXPIRES_KEY)
  if (!exp) return true
  return Date.now() > Number(exp)
}
function clearRemembered() {
  localStorage.removeItem(REMEMBER_PASS_KEY)
  localStorage.removeItem(REMEMBER_EXPIRES_KEY)
  localStorage.removeItem(REMEMBER_USER_KEY)
  localStorage.setItem(REMEMBER_KEY, '0')
}

// 如果过期则自动清除
if (localStorage.getItem(REMEMBER_KEY) === '1' && isRememberExpired()) {
  clearRemembered()
}

const rememberMe = ref(localStorage.getItem(REMEMBER_KEY) === '1')
const forgotVisible = ref(false)

const sliderRef = ref<HTMLElement | null>(null)
const SLIDER_HANDLE_WIDTH = 42
const SLIDER_SUCCESS_THRESHOLD = 2
const sliderX = ref(0)
const sliderMax = ref(0)
const sliderDragging = ref(false)
const sliderSuccess = ref(false)
const sliderText = computed(() => {
  if (sliderSuccess.value) return '验证成功'
  return '按住滑块拖动'
})

let sliderStartClientX = 0
let sliderStartX = 0
let cleanupMoveListeners: (() => void) | null = null

let prevHtmlOverflow = ''
let prevBodyOverflow = ''

onUnmounted(() => {
  if (typeof document !== 'undefined') {
    document.documentElement.style.overflow = prevHtmlOverflow
    document.body.style.overflow = prevBodyOverflow
  }
  cleanupMoveListeners?.()
})

onMounted(() => {
  prevHtmlOverflow = document.documentElement.style.overflow
  prevBodyOverflow = document.body.style.overflow
  document.documentElement.style.overflow = 'hidden'
  document.body.style.overflow = 'hidden'

  if (rememberMe.value && !isRememberExpired()) {
    form.username = localStorage.getItem(REMEMBER_USER_KEY) ?? ''
    const enc = localStorage.getItem(REMEMBER_PASS_KEY) ?? ''
    form.password = enc ? decodePass(enc) : ''
  } else if (rememberMe.value) {
    // 已过期 — 清除并仅保留用户名
    const savedUser = localStorage.getItem(REMEMBER_USER_KEY) ?? ''
    clearRemembered()
    rememberMe.value = false
    form.username = savedUser
  }
})

function clamp(n: number, min: number, max: number): number {
  if (n < min) return min
  if (n > max) return max
  return n
}

function calcSliderMax() {
  const el = sliderRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  sliderMax.value = Math.max(0, Math.round(rect.width - SLIDER_HANDLE_WIDTH))
}

function setSliderXFromClientX(clientX: number) {
  const max = sliderMax.value
  const next = clamp(sliderStartX + (clientX - sliderStartClientX), 0, max)
  sliderX.value = next
}

function onSliderDown(e: MouseEvent) {
  if (sliderSuccess.value) return
  calcSliderMax()
  sliderDragging.value = true
  sliderStartClientX = e.clientX
  sliderStartX = sliderX.value

  const onMove = (ev: MouseEvent) => setSliderXFromClientX(ev.clientX)
  const onUp = () => finishSlider()

  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp, { once: true })
  cleanupMoveListeners = () => {
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
    cleanupMoveListeners = null
  }
}

function onSliderTouchStart(e: TouchEvent) {
  if (sliderSuccess.value) return
  const touch = e.touches?.[0]
  if (!touch) return
  calcSliderMax()
  sliderDragging.value = true
  sliderStartClientX = touch.clientX
  sliderStartX = sliderX.value

  const onMove = (ev: TouchEvent) => {
    const t = ev.touches?.[0]
    if (!t) return
    setSliderXFromClientX(t.clientX)
  }
  const onEnd = () => finishSlider()

  window.addEventListener('touchmove', onMove, { passive: true })
  window.addEventListener('touchend', onEnd, { once: true })
  cleanupMoveListeners = () => {
    window.removeEventListener('touchmove', onMove)
    window.removeEventListener('touchend', onEnd)
    cleanupMoveListeners = null
  }
}

function finishSlider() {
  cleanupMoveListeners?.()
  sliderDragging.value = false
  const max = sliderMax.value
  if (max > 0 && sliderX.value >= max - SLIDER_SUCCESS_THRESHOLD) {
    sliderX.value = max
    sliderSuccess.value = true
    return
  }
  sliderX.value = 0
}

async function onLogin() {
  if (!sliderSuccess.value) {
    notifyError('请先完成滑动验证')
    return
  }

  const existingToken = getToken()
  const existingUser = getSessionUser()
  if (existingToken && !isTokenExpired() && existingUser?.username && existingUser.username !== form.username) {
    try {
      await ElMessageBox.confirm(
        `当前浏览器已登录用户：${existingUser.username}。为保证同一时间段只登录同一用户，需要先退出该账号再登录 ${form.username}。`,
        '提示',
        { type: 'warning', confirmButtonText: '强制切换', cancelButtonText: '取消' }
      )
    } catch {
      return
    }
    userStore.logoutLocal()
    clearToken()
  }

  loading.value = true
  try {
    await userStore.login(form.username, form.password)
    notifySuccess('登录成功')
    localStorage.setItem(REMEMBER_KEY, rememberMe.value ? '1' : '0')
    localStorage.setItem(REMEMBER_USER_KEY, form.username)
    if (rememberMe.value) {
      localStorage.setItem(REMEMBER_PASS_KEY, encodePass(form.password))
      localStorage.setItem(REMEMBER_EXPIRES_KEY, String(Date.now() + REMEMBER_TTL_MS))
    } else {
      localStorage.removeItem(REMEMBER_PASS_KEY)
      localStorage.removeItem(REMEMBER_EXPIRES_KEY)
    }
    const redirect = (route.query.redirect as string) || '/'
    await router.push(redirect)
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* ================================================================
   World-class Login — Dark / Light dual theme via CSS variables
   Toggle controlled by useTheme composable; .is-light class on root
   ================================================================ */

/* ── Dark theme tokens (default) ── */
.login-page {
  --lp-bg:          #050a18;
  --lp-color:       #e2e8f0;
  --lp-muted:       #94a3b8;
  --lp-subtle:      #64748b;
  --lp-orb-opacity: 0.35;
  --lp-grid-line:   rgba(255,255,255,0.025);
  --lp-noise:       0.03;

  --lp-title-from:  #e0e7ff;
  --lp-title-mid:   #818cf8;
  --lp-title-to:    #06b6d4;

  --lp-pill-bg:     rgba(255,255,255,0.04);
  --lp-pill-border: rgba(255,255,255,0.06);
  --lp-pill-color:  #94a3b8;
  --lp-pill-icon:   #818cf8;
  --lp-pill-hover-bg: rgba(255,255,255,0.08);
  --lp-pill-hover-border: rgba(129,140,248,0.3);
  --lp-pill-hover-color: #c7d2fe;

  --lp-card-bg:     rgba(255,255,255,0.04);
  --lp-card-border: rgba(255,255,255,0.08);
  --lp-card-shadow: 0 0 0 1px rgba(255,255,255,0.03) inset, 0 32px 64px -16px rgba(0,0,0,0.5), 0 0 80px -20px rgba(99,102,241,0.12);

  --lp-form-title:  #f1f5f9;
  --lp-input-bg:    rgba(255,255,255,0.05);
  --lp-input-border:rgba(255,255,255,0.1);
  --lp-input-text:  #e2e8f0;
  --lp-input-focus-border: rgba(129,140,248,0.5);
  --lp-input-focus-ring: rgba(99,102,241,0.12);
  --lp-input-focus-bg: rgba(255,255,255,0.07);

  --lp-slider-track-bg: rgba(255,255,255,0.05);
  --lp-slider-track-border: rgba(255,255,255,0.1);
  --lp-handle-bg:   rgba(255,255,255,0.1);
  --lp-handle-border: rgba(255,255,255,0.15);
  --lp-handle-hover: rgba(255,255,255,0.15);
  --lp-handle-shadow: none;

  --lp-check-bg:    rgba(255,255,255,0.05);
  --lp-check-border:rgba(255,255,255,0.15);
  --lp-check-label: #94a3b8;
  --lp-link-color:  #818cf8;

  --lp-btn-shadow:  0 4px 20px rgba(99,102,241,0.3);
  --lp-btn-hover-shadow: 0 8px 32px rgba(99,102,241,0.45);

  --lp-footer-color:#475569;
  --lp-success-text:#6ee7b7;

  --lp-toggle-bg:   rgba(255,255,255,0.06);
  --lp-toggle-border: rgba(255,255,255,0.1);
  --lp-toggle-color: #94a3b8;
  --lp-toggle-hover: rgba(255,255,255,0.12);

  --lp-shine-color: rgba(255,255,255,0.5);
  --lp-geo-stroke: rgba(129,140,248,0.15);
  --lp-geo-fill: rgba(99,102,241,0.08);
}

/* ── Light theme tokens ── */
.login-page.is-light {
  --lp-bg:          #f0f2f8;
  --lp-color:       #1e293b;
  --lp-muted:       #64748b;
  --lp-subtle:      #94a3b8;
  --lp-orb-opacity: 0.12;
  --lp-grid-line:   rgba(0,0,0,0.035);
  --lp-noise:       0.015;

  --lp-title-from:  #1e293b;
  --lp-title-mid:   #6366f1;
  --lp-title-to:    #0891b2;

  --lp-pill-bg:     rgba(99,102,241,0.06);
  --lp-pill-border: rgba(99,102,241,0.12);
  --lp-pill-color:  #475569;
  --lp-pill-icon:   #6366f1;
  --lp-pill-hover-bg: rgba(99,102,241,0.1);
  --lp-pill-hover-border: rgba(99,102,241,0.25);
  --lp-pill-hover-color: #4338ca;

  --lp-card-bg:     rgba(255,255,255,0.72);
  --lp-card-border: rgba(0,0,0,0.06);
  --lp-card-shadow: 0 0 0 1px rgba(255,255,255,0.8) inset, 0 24px 48px -12px rgba(0,0,0,0.1), 0 0 60px -20px rgba(99,102,241,0.08);

  --lp-form-title:  #0f172a;
  --lp-input-bg:    rgba(255,255,255,0.8);
  --lp-input-border:rgba(0,0,0,0.1);
  --lp-input-text:  #0f172a;
  --lp-input-focus-border: rgba(99,102,241,0.5);
  --lp-input-focus-ring: rgba(99,102,241,0.08);
  --lp-input-focus-bg: #ffffff;

  --lp-slider-track-bg: rgba(0,0,0,0.04);
  --lp-slider-track-border: rgba(0,0,0,0.1);
  --lp-handle-bg:   #ffffff;
  --lp-handle-border: rgba(0,0,0,0.12);
  --lp-handle-hover: #ffffff;
  --lp-handle-shadow: 0 2px 8px rgba(0,0,0,0.1);

  --lp-check-bg:    #ffffff;
  --lp-check-border:rgba(0,0,0,0.15);
  --lp-check-label: #475569;
  --lp-link-color:  #6366f1;

  --lp-btn-shadow:  0 4px 20px rgba(99,102,241,0.2);
  --lp-btn-hover-shadow: 0 8px 32px rgba(99,102,241,0.3);

  --lp-footer-color:#94a3b8;
  --lp-success-text:#059669;

  --lp-toggle-bg:   rgba(0,0,0,0.04);
  --lp-toggle-border: rgba(0,0,0,0.1);
  --lp-toggle-color: #64748b;
  --lp-toggle-hover: rgba(0,0,0,0.08);

  --lp-shine-color: rgba(0,0,0,0.06);
  --lp-geo-stroke: rgba(99,102,241,0.12);
  --lp-geo-fill: rgba(99,102,241,0.06);
}

/* ── Page shell ── */
.login-page {
  position: relative;
  z-index: 2;
  isolation: isolate;
  display: flex;
  height: 100vh;
  overflow: hidden;
  background: var(--lp-bg);
  color: var(--lp-color);
  transition: background 0.5s ease, color 0.4s ease;
}

/* ── Theme toggle button ── */
.theme-toggle {
  position: absolute;
  top: 20px;
  right: 24px;
  z-index: 10;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--lp-toggle-bg);
  border: 1px solid var(--lp-toggle-border);
  color: var(--lp-toggle-color);
  cursor: pointer;
  backdrop-filter: blur(12px);
  transition: all 0.25s ease;
}
.theme-toggle:hover {
  background: var(--lp-toggle-hover);
  color: #818cf8;
  transform: scale(1.05);
}
.theme-toggle:active {
  transform: scale(0.95);
}

/* ── Full-screen ambient background ── */
.login-bg {
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  overflow: hidden;
  transition: opacity 0.5s ease;
}

.bg-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(100px);
  opacity: var(--lp-orb-opacity);
  will-change: transform;
  transition: opacity 0.5s ease;
}
.bg-orb-1 {
  width: 700px; height: 700px;
  background: radial-gradient(circle, rgba(99,102,241,0.7), transparent 70%);
  top: -20%; left: -10%;
  animation: orb-drift-1 14s ease-in-out infinite alternate;
}
.bg-orb-2 {
  width: 600px; height: 600px;
  background: radial-gradient(circle, rgba(6,182,212,0.5), transparent 70%);
  bottom: -25%; right: 5%;
  animation: orb-drift-2 18s ease-in-out infinite alternate;
}
.bg-orb-3 {
  width: 400px; height: 400px;
  background: radial-gradient(circle, rgba(139,92,246,0.4), transparent 70%);
  top: 50%; left: 45%;
  transform: translate(-50%, -50%);
  animation: orb-drift-3 12s ease-in-out infinite alternate;
}

.bg-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(var(--lp-grid-line) 1px, transparent 1px),
    linear-gradient(90deg, var(--lp-grid-line) 1px, transparent 1px);
  background-size: 64px 64px;
  mask-image: radial-gradient(ellipse 80% 60% at 50% 50%, black 20%, transparent 75%);
  -webkit-mask-image: radial-gradient(ellipse 80% 60% at 50% 50%, black 20%, transparent 75%);
  transition: background-image 0.5s ease;
}

.bg-noise {
  position: absolute;
  inset: 0;
  opacity: var(--lp-noise);
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
  background-repeat: repeat;
  background-size: 256px 256px;
  transition: opacity 0.5s ease;
}

/* ── Left: Brand area ── */
.login-brand {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 1;
  padding: 48px;
  animation: brand-enter 1s cubic-bezier(0.22, 1, 0.36, 1) both;
  overflow: hidden;
}

.brand-inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  max-width: 480px;
  text-align: center;
}

/* ── Geometric decorations ── */
.geo-decor {
  position: absolute;
  pointer-events: none;
  transition: opacity 0.5s ease;
}
.geo-circle {
  border-radius: 50%;
  border: 2px solid;
  border-color: var(--lp-geo-stroke, rgba(99,102,241,0.12));
}
.geo-circle-1 { width: 120px; height: 120px; top: 8%; left: 8%; animation: geo-float 8s ease-in-out infinite alternate; }
.geo-circle-2 { width: 60px; height: 60px; top: 18%; right: 14%; border-style: dashed; animation: geo-float 10s ease-in-out 1s infinite alternate-reverse; }
.geo-circle-3 { width: 180px; height: 180px; bottom: 5%; right: 2%; opacity: 0.5; animation: geo-float 12s ease-in-out 2s infinite alternate; }
.geo-diamond {
  width: 22px; height: 22px;
  background: var(--lp-geo-fill, rgba(99,102,241,0.1));
  border: 1.5px solid var(--lp-geo-stroke, rgba(99,102,241,0.15));
  border-radius: 4px;
  transform: rotate(45deg);
}
.geo-diamond-1 { bottom: 22%; left: 12%; animation: geo-drift 7s ease-in-out infinite alternate; }
.geo-diamond-2 { top: 12%; right: 8%; animation: geo-drift 9s ease-in-out 1.5s infinite alternate-reverse; }
.geo-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  background: var(--lp-geo-fill, rgba(99,102,241,0.18));
}
.geo-dot-1 { top: 40%; left: 5%; }
.geo-dot-2 { bottom: 35%; right: 10%; }
.geo-dot-3 { top: 65%; left: 22%; }

/* ── Brand Illustration ── */
.brand-illustration {
  position: relative;
  width: 100%;
  max-width: 420px;
  margin: 8px auto 24px;
  animation: illo-enter 1.2s cubic-bezier(0.22, 1, 0.36, 1) 0.3s both;
}
.illo-svg {
  display: block;
  width: 100%;
  height: auto;
  filter: drop-shadow(0 8px 32px rgba(99,102,241,0.12));
}
.login-page.is-light .illo-svg {
  filter: drop-shadow(0 8px 24px rgba(99,102,241,0.1));
}

/* ── HTML 浮动卡片 ── */
.illo-card {
  position: absolute;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 10px;
  border-radius: 10px;
  background: rgba(255,255,255,0.97);
  border: 1px solid #e0e7ff;
  box-shadow: 0 4px 16px rgba(99,102,241,0.12);
  font-size: 11px;
  z-index: 2;
  pointer-events: none;
  white-space: nowrap;
}
.login-page.is-light .illo-card {
  box-shadow: 0 4px 12px rgba(99,102,241,0.08);
}
/* 卡片 1: 集群概览 — 左上 */
.illo-card-1 {
  top: 2%;
  left: 0;
  animation: card-float-1 5s ease-in-out infinite alternate;
}
/* 卡片 2: 集群健康 — 右上 */
.illo-card-2 {
  top: 0;
  right: 0;
  animation: card-float-2 6s ease-in-out 0.5s infinite alternate-reverse;
}
/* 卡片 3: 工作负载 — 右下 */
.illo-card-3 {
  bottom: 16%;
  right: 0;
  animation: card-float-3 4.5s ease-in-out 1s infinite alternate;
}

.illo-card-icon {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.illo-card-icon--blue { background: #dbeafe; }
.illo-card-icon--green { background: #dcfce7; }
.illo-card-icon--amber { background: #fef3c7; }

.illo-card-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.illo-card-title {
  font-weight: 600;
  font-size: 11px;
  color: #1e293b;
  line-height: 1.2;
}
.login-page:not(.is-light) .illo-card-title { color: #1e293b; }
.illo-card-sub {
  font-size: 9px;
  color: #94a3b8;
  line-height: 1.2;
}

/* 进度条 */
.illo-card-bar {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 2px;
}
.illo-card-bar-bg {
  flex: 1;
  height: 4px;
  background: #e0e7ff;
  border-radius: 2px;
  overflow: hidden;
  min-width: 50px;
}
.illo-card-bar-fill {
  height: 100%;
  border-radius: 2px;
  background: linear-gradient(90deg, #818cf8, #06b6d4);
}
.illo-card-bar-fill--animated {
  width: 55%;
  animation: bar-pulse 3s ease-in-out infinite;
}
.illo-card-bar-label {
  font-size: 9px;
  font-weight: 500;
  color: #818cf8;
}

/* 状态点 */
.illo-card-dots {
  display: flex;
  gap: 4px;
  margin-top: 2px;
}
.illo-dot {
  width: 18px;
  height: 4px;
  border-radius: 2px;
}
.illo-dot--green { background: #86efac; }
.illo-dot--blue { background: #93c5fd; }

/* ── Logo Mark ── */
.logo-mark {
  margin-bottom: 16px;
}
.logo-hexagon {
  width: 64px;
  height: 64px;
}
.logo-svg {
  width: 100%;
  height: 100%;
  filter: drop-shadow(0 0 20px rgba(99,102,241,0.4));
}
.login-page.is-light .logo-svg {
  filter: drop-shadow(0 0 16px rgba(99,102,241,0.25));
}
.logo-ring-rotate {
  transform-origin: 40px 40px;
  animation: ring-spin 20s linear infinite;
}

.brand-title {
  font-size: 44px;
  font-weight: 900;
  letter-spacing: -1px;
  line-height: 1.1;
  background: linear-gradient(135deg, var(--lp-title-from) 0%, var(--lp-title-mid) 50%, var(--lp-title-to) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin: 0 0 6px;
  transition: background 0.4s ease;
}

.brand-subtitle {
  font-size: 12px;
  font-weight: 500;
  color: var(--lp-subtle);
  letter-spacing: 3.5px;
  text-transform: uppercase;
  margin: 0 0 8px;
  transition: color 0.4s ease;
}

.brand-desc {
  font-size: 14.5px;
  line-height: 1.7;
  color: var(--lp-muted);
  margin: 0 0 28px;
  max-width: 400px;
  transition: color 0.4s ease;
}

/* Feature pills */
.brand-features {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.feature-pill {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 8px 16px;
  border-radius: 100px;
  font-size: 13px;
  font-weight: 500;
  color: var(--lp-pill-color);
  background: var(--lp-pill-bg);
  border: 1px solid var(--lp-pill-border);
  backdrop-filter: blur(8px);
  transition: all 0.3s ease;
  cursor: default;
}
.feature-pill:hover {
  background: var(--lp-pill-hover-bg);
  border-color: var(--lp-pill-hover-border);
  color: var(--lp-pill-hover-color);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(99,102,241,0.1);
}
.feature-pill .el-icon {
  color: var(--lp-pill-icon);
  transition: color 0.3s ease;
}

/* ── Right: Glass form card ── */
.login-form-side {
  width: 520px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 48px;
  position: relative;
  z-index: 1;
  animation: form-enter 1s cubic-bezier(0.22, 1, 0.36, 1) 0.15s both;
}

.form-card {
  width: 100%;
  max-width: 420px;
  padding: 48px 44px 40px;
  border-radius: 28px;
  background: var(--lp-card-bg);
  border: 1px solid var(--lp-card-border);
  backdrop-filter: blur(40px) saturate(1.4);
  -webkit-backdrop-filter: blur(40px) saturate(1.4);
  box-shadow: var(--lp-card-shadow);
  transition: background 0.4s ease, border-color 0.4s ease, box-shadow 0.4s ease;
}

.form-header {
  margin-bottom: 32px;
  text-align: left;
}

.form-title {
  font-size: 28px;
  font-weight: 800;
  color: var(--lp-form-title);
  margin: 0 0 8px;
  letter-spacing: 0.3px;
  transition: color 0.4s ease;
}

.form-sub {
  font-size: 14px;
  color: var(--lp-subtle);
  margin: 0;
  transition: color 0.4s ease;
}

.login-form {
  margin-top: 0;
}

/* ── Input styling ── */
.login-form :deep(.el-form-item) {
  margin-bottom: 20px;
}

.login-form :deep(.el-input__wrapper) {
  border-radius: 14px;
  min-height: 48px;
  background: var(--lp-input-bg);
  border: 1px solid var(--lp-input-border);
  box-shadow: none;
  transition: all 0.25s ease;
  --el-input-bg-color: transparent;
  --el-input-border-color: transparent;
  --el-input-text-color: var(--lp-input-text);
  --el-input-placeholder-color: var(--lp-subtle);
}
.login-form :deep(.el-input__wrapper.is-focus) {
  border-color: var(--lp-input-focus-border);
  box-shadow: 0 0 0 3px var(--lp-input-focus-ring);
  background: var(--lp-input-focus-bg);
}
.login-form :deep(.el-input__inner) {
  color: var(--lp-input-text);
  font-size: 14.5px;
}
.login-form :deep(.el-input__inner::placeholder) {
  color: var(--lp-subtle);
}
.login-form :deep(.el-input__prefix .el-icon) {
  color: var(--lp-muted);
  font-size: 17px;
  transition: color 0.25s;
}
.login-form :deep(.el-input__wrapper.is-focus .el-input__prefix .el-icon) {
  color: #818cf8;
}
/* Password eye icon */
.login-form :deep(.el-input__suffix .el-icon) {
  color: var(--lp-subtle);
}

/* ── Slider ── */
.slider {
  position: relative;
  width: 100%;
  height: 44px;
  border-radius: 14px;
  user-select: none;
  touch-action: pan-y;
  margin-bottom: 18px;
}

.slider-track {
  position: absolute;
  inset: 0;
  border-radius: 14px;
  background: var(--lp-slider-track-bg);
  border: 1px solid var(--lp-slider-track-border);
  overflow: hidden;
  transition: background 0.4s ease, border-color 0.4s ease;
}

.slider-progress {
  position: absolute;
  left: 0; top: 0;
  height: 100%;
  border-radius: 14px;
  width: 42px;
  background: linear-gradient(90deg, #6366f1, #818cf8);
  transition: width 160ms ease;
}
.slider.is-dragging .slider-progress { transition: none; }
.slider.is-success .slider-progress {
  background: linear-gradient(90deg, #10b981, #34d399);
  animation: slider-success-pulse 0.5s ease-out;
}

.slider-text {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 500;
  color: var(--lp-subtle);
  pointer-events: none;
  transition: color 0.4s ease;
}
.slider.is-success .slider-text { color: var(--lp-success-text); }

.slider-text-inner {
  position: relative;
  padding: 0 10px;
  display: inline-block;
  line-height: 1;
  overflow: hidden;
}

.slider:not(.is-success):not(.is-dragging) .slider-text-inner::after {
  content: '';
  position: absolute;
  top: 50%; left: -40%;
  width: 3.2em; height: 1.15em;
  background: linear-gradient(90deg, transparent, var(--lp-shine-color), transparent);
  transform: translateY(-50%) skewX(-18deg);
  animation: slider-shine 2s ease-in-out infinite;
  pointer-events: none;
}

.slider-handle {
  position: absolute;
  top: 1px; left: 1px;
  width: 42px; height: 42px;
  border-radius: 13px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--lp-handle-bg);
  border: 1px solid var(--lp-handle-border);
  box-shadow: var(--lp-handle-shadow);
  backdrop-filter: blur(8px);
  transition: transform 160ms ease, background 0.25s ease, border-color 0.25s ease, box-shadow 0.25s ease;
}
.slider-handle:hover {
  background: var(--lp-handle-hover);
}
.slider.is-dragging .slider-handle { transition: none; }
.slider.is-success .slider-handle {
  border-color: rgba(110, 231, 183, 0.4);
  background: rgba(16, 185, 129, 0.1);
}

.slider-handle-icon {
  font-size: 16px;
  font-weight: 900;
  line-height: 1;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  color: var(--lp-muted);
  display: inline-flex;
  align-items: center;
  transition: color 0.3s ease;
}
.slider.is-dragging:not(.is-success) .slider-handle-icon {
  animation: slider-slash-spin 520ms linear infinite;
}
.shell-caret { display: inline-block; }
.slider.is-success .shell-caret { animation: shell-caret-blink 900ms step-end infinite; }
.slider.is-success .slider-handle-icon { color: var(--lp-success-text); }

/* ── Options row ── */
.login-options {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 4px 2px 20px;
}
.login-options :deep(.el-link) { font-size: 13px; font-weight: 500; }
.login-options :deep(.el-link .el-link__inner) { color: var(--lp-link-color); transition: color 0.3s; }

/* ── Custom Toggle Switch ── */
.remember-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
}
.toggle-track {
  position: relative;
  width: 36px;
  height: 20px;
  border-radius: 10px;
  background: var(--lp-check-border, rgba(0,0,0,0.15));
  transition: background 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  flex-shrink: 0;
}
.toggle-track.is-on {
  background: linear-gradient(135deg, #6366f1 0%, #818cf8 100%);
}
.toggle-thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #ffffff;
  box-shadow: 0 1px 3px rgba(0,0,0,0.15);
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.3s;
}
.toggle-track.is-on .toggle-thumb {
  transform: translateX(16px);
  box-shadow: 0 1px 4px rgba(99,102,241,0.35);
}
.toggle-label {
  font-size: 13px;
  color: var(--lp-check-label, #94a3b8);
  transition: color 0.3s;
  line-height: 1;
}
.toggle-hint {
  font-size: 14px;
  color: var(--lp-check-label, #94a3b8);
  opacity: 0.6;
  transition: opacity 0.2s, color 0.3s;
  cursor: help;
}
.remember-toggle:hover .toggle-hint {
  opacity: 1;
}

/* ── Login button ── */
.login-btn {
  width: 100%;
  height: 50px;
  border-radius: 14px;
  font-weight: 700;
  font-size: 15.5px;
  letter-spacing: 3px;
  color: #ffffff;
  background: linear-gradient(135deg, #6366f1 0%, #818cf8 50%, #06b6d4 100%);
  background-size: 200% 200%;
  border: none;
  box-shadow: var(--lp-btn-shadow);
  transition: all 0.3s ease;
  animation: btn-shimmer 4s ease-in-out infinite;
}
.login-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: var(--lp-btn-hover-shadow);
}
.login-btn:active:not(:disabled) {
  transform: translateY(0);
}

/* ── Footer ── */
.login-footer {
  margin-top: 28px;
  padding-top: 20px;
  border-top: 1px solid var(--lp-card-border);
  text-align: center;
  font-size: 11.5px;
  letter-spacing: 0.3px;
  color: var(--lp-footer-color);
  transition: color 0.4s ease, border-color 0.4s ease;
}

/* ── Forgot dialog ── */
.forgot-body { color: #475569; line-height: 1.6; }
:global(html.dark) .forgot-body { color: #cbd5e1; }

/* ========== Keyframes ========== */
@keyframes orb-drift-1 {
  from { transform: translate(0, 0) scale(1); }
  to   { transform: translate(60px, 40px) scale(1.15); }
}
@keyframes orb-drift-2 {
  from { transform: translate(0, 0) scale(1); }
  to   { transform: translate(-70px, -50px) scale(1.1); }
}
@keyframes orb-drift-3 {
  from { transform: translate(-50%, -50%) scale(1); }
  to   { transform: translate(calc(-50% + 30px), calc(-50% - 20px)) scale(1.2); }
}

@keyframes ring-spin {
  to { transform: rotate(360deg); }
}

@keyframes brand-enter {
  from { opacity: 0; transform: translateY(20px); }
  to   { opacity: 1; transform: translateY(0); }
}
@keyframes form-enter {
  from { opacity: 0; transform: translateY(20px) scale(0.98); }
  to   { opacity: 1; transform: translateY(0) scale(1); }
}
@keyframes illo-enter {
  from { opacity: 0; transform: translateY(30px) scale(0.95); }
  to   { opacity: 1; transform: translateY(0) scale(1); }
}
@keyframes geo-float {
  from { transform: translate(0, 0); }
  to   { transform: translate(8px, -12px); }
}
@keyframes geo-drift {
  from { transform: rotate(45deg) translate(0, 0); }
  to   { transform: rotate(45deg) translate(6px, -8px); }
}
@keyframes card-float-1 {
  from { transform: translateY(0); }
  to   { transform: translateY(-7px); }
}
@keyframes card-float-2 {
  from { transform: translateY(0); }
  to   { transform: translateY(-6px); }
}
@keyframes card-float-3 {
  from { transform: translateY(0); }
  to   { transform: translateY(-8px); }
}
@keyframes bar-pulse {
  0%, 100% { width: 30%; }
  50% { width: 65%; }
}

@keyframes btn-shimmer {
  0%, 100% { background-position: 0% 50%; }
  50%      { background-position: 100% 50%; }
}

@keyframes slider-shine {
  0%   { left: -50%; opacity: 0; }
  15%  { opacity: 1; }
  60%  { left: 110%; opacity: 0; }
  100% { left: 110%; opacity: 0; }
}
@keyframes slider-slash-spin {
  to { transform: rotate(360deg); }
}
@keyframes shell-caret-blink {
  0%, 49% { opacity: 1; }
  50%, 100% { opacity: 0; }
}
@keyframes slider-success-pulse {
  0%   { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4); }
  100% { box-shadow: 0 0 0 8px rgba(16, 185, 129, 0); }
}

/* ========== Responsive ========== */
@media (max-width: 960px) {
  .login-brand { display: none; }
  .login-form-side { width: 100%; flex: 1; padding: 24px; }
  .form-card { max-width: 440px; }
}
@media (max-width: 480px) {
  .form-card { padding: 32px 22px 28px; border-radius: 22px; }
  .form-title { font-size: 22px; }
}

@media (prefers-reduced-motion: reduce) {
  .login-brand, .login-form-side { animation: none; }
  .bg-orb { animation: none; }
  .logo-ring-rotate { animation: none; }
  .login-btn { animation: none; }
  .slider:not(.is-success):not(.is-dragging) .slider-text-inner::after { animation: none; }
  .slider.is-dragging:not(.is-success) .slider-handle-icon { animation: none; }
  .geo-decor { animation: none; }
  .illo-card-1, .illo-card-2, .illo-card-3, .illo-card-bar-fill--animated { animation: none; }
  .brand-illustration { animation: none; }
}
</style>
