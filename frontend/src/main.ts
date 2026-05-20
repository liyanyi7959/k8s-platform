import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'

/* Design Token 三层体系 — 顺序不可调整 */
import '@/styles/tokens/primitives.css'
import '@/styles/tokens/semantic.css'
import '@/styles/tokens/component.css'

/* 全局样式 — 依赖 Token */
import '@/styles/theme-overrides.css'
import '@/styles/base.css'
import '@/styles/k8s-detail.css'
import '@/styles/k8s-edit.css'
import '@/styles/k8s-panel.css'
import '@/styles/global-overrides.css'
import '@/styles/enterprise.css'       /* 企业级统一组件规范 — 最后加载以覆盖 */
import '@/styles/art-design-v2.css'    /* V2: AppShell scoped visual refresh */

import { router } from '@/app/router'
import App from '@/App.vue'

const MIN_SPLASH_DURATION = 520

function hideSplashScreen() {
  const splash = document.getElementById('app-splash')
  if (!splash) return

  splash.classList.add('fade-out')
  splash.addEventListener('transitionend', () => splash.remove(), { once: true })
  window.setTimeout(() => splash.remove(), 700)
}

async function bootstrap() {
  const startedAt = typeof performance !== 'undefined' ? performance.now() : Date.now()

  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  app.use(ElementPlus)

  await router.isReady()
  app.mount('#app')

  const finishedAt = typeof performance !== 'undefined' ? performance.now() : Date.now()
  const elapsed = finishedAt - startedAt
  const remain = Math.max(0, MIN_SPLASH_DURATION - elapsed)

  window.setTimeout(() => {
    requestAnimationFrame(() => hideSplashScreen())
  }, remain)
}

bootstrap().catch((error) => {
  console.error('应用启动失败:', error)
  hideSplashScreen()
})

