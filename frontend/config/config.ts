/**
 * UmiJS Max 配置文件
 * AIOPS 智能运维平台前端配置
 */
import { defineConfig } from '@umijs/max'
import routes from './routes'

const projectRootPattern = process.cwd()
  .replace(/\\/g, '/')
  .replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

// Watchpack normalizes paths to forward slashes before applying this regexp.
// Keep development watching inside this frontend project on Windows.
const devWatchIgnore = new RegExp(
  `^(?!${projectRootPattern}(?:/|$))|(?:^|/)(?:\\.git|node_modules|dist)(?:/|$)|(?:^|/)src/\\.umi-production(?:/|$)`,
)

export default defineConfig({
  plugins: ['./plugins/aiops-loading.ts'],
  routes,
  conventionLayout: false,
  // 禁用 Module Federation（避免 mf-va_remoteEntry.js 加载失败）
  mf: false,
  // Windows 下 MFSU 会与 .umi 临时文件生成发生竞态；默认保持稳定模式。
  // 修复上游竞态后可用 AIOPS_ENABLE_MFSU=1 进行验证。
  mfsu: process.env.AIOPS_ENABLE_MFSU === '1' ? { strategy: 'normal' } : false,
  // 生产构建开启 IIFE helper 去重，避免 esbuild helper conflict 阻断产物生成
  esbuildMinifyIIFE: true,
  antd: {
    theme: {
      token: {
        // Keep Ant Design primitives aligned with the product design tokens.
        colorPrimary: '#2563eb',
        colorInfo: '#2563eb',
        colorSuccess: '#047857',
        colorWarning: '#b45309',
        colorError: '#dc2626',
        colorTextSecondary: '#64748b',
        borderRadius: 12,
      },
    },
  },
  access: {},
  model: {},
  initialState: {},
  request: {},
  reactQuery: {},
  layout: {
    title: 'AIOPS 智能运维平台',
    locale: false,
  },
  // 包体积优化
  chainWebpack(config) {
    config.merge({
      watchOptions: {
        ignored: devWatchIgnore,
      },
    })
    config.performance
      .maxAssetSize(500 * 1024)
      .maxEntrypointSize(1200 * 1024)
      .hints('warning')
  },
  // WebSocket 路由必须在 /api 前面，确保升级请求不会落入普通 HTTP 代理。
  proxy: {
    '/api/v1/deploy/servers/terminal/ws': {
      target: 'ws://localhost:8080',
      ws: true,
      changeOrigin: true,
    },
    '/api/v1/ws': {
      target: 'ws://localhost:8080',
      ws: true,
      changeOrigin: true,
    },
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    },
  },
  // 忽略 moment 语言包
  ignoreMomentLocale: true,
  // 开启 hash 模式
  hash: true,
})
