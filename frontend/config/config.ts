/**
 * UmiJS Max 配置文件
 * AIOPS 智能运维平台前端配置
 */
import { defineConfig } from '@umijs/max'
import routes from './routes'

export default defineConfig({
  routes,
  conventionLayout: false,
  // 禁用 Module Federation（避免 mf-va_remoteEntry.js 加载失败）
  mf: false,
  // 当前仓库在 Windows 开发态会出现 .umi / icons 构建竞态与 OOM，先关闭 MFSU 保持稳定
  mfsu: false,
  // 生产构建开启 IIFE helper 去重，避免 esbuild helper conflict 阻断产物生成
  esbuildMinifyIIFE: true,
  antd: {
    theme: {
      token: {
        colorPrimary: '#1677ff',
        borderRadius: 6,
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
    config.performance
      .maxAssetSize(500 * 1024)
      .maxEntrypointSize(1200 * 1024)
      .hints('warning')
  },
  // 代理配置（/api/v1/ws 必须在 /api 前面，确保 WebSocket 升级请求被正确代理）
  proxy: {
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
