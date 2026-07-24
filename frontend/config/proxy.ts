/**
 * 代理配置
 * 本地开发环境代理 API 请求到后端服务
 */
export default {
  '/api/v1/deploy/servers/terminal/ws': {
    target: 'ws://localhost:8080',
    ws: true,
    changeOrigin: true,
  },
  '/api': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
  '/ws': {
    target: 'ws://localhost:8080',
    ws: true,
    changeOrigin: true,
  },
}
