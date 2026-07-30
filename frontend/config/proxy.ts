/**
 * 代理配置
 * 本地开发环境代理 API 请求到后端服务
 */
export default {
  '/streams/v2': {
    target: 'ws://localhost:8080',
    ws: true,
    changeOrigin: true,
  },
  '/api': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
}
