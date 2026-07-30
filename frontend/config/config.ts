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
  // 在首屏脚本加载前写入浏览器标签图标，避免依赖运行时 useEffect。
  favicons: ['/brand/aiops-mark.svg?v=2'],
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
        colorText: '#17283f',
        colorTextSecondary: '#64748b',
        colorTextTertiary: '#94a3b8',
        colorBorder: '#e2e8f0',
        colorBorderSecondary: '#edf1f6',
        colorBgLayout: '#f3f6fa',
        fontFamily: "Inter, 'SF Pro Text', 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif",
        fontSize: 14,
        controlHeight: 38,
        borderRadius: 12,
      },
    },
  },
  access: {},
  model: {},
  initialState: {},
  request: {},
  // 由 src/app.tsx 根容器提供 QueryClient，避免 Umi 运行时代码反向导入 app.tsx 造成循环依赖。
  reactQuery: { queryClient: false, devtool: false },
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
  // 流式路由必须在 /api 前面，确保升级请求不会落入普通 HTTP 代理。
  proxy: {
    '/streams/v2': {
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
