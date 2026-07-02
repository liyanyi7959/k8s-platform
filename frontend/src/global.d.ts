/**
 * 全局类型声明
 */

/** 运行时配置 */
interface RuntimeConfig {
  apiBaseUrl?: string
  wsBaseUrl?: string
}

declare global {
  interface Window {
    __RUNTIME_CONFIG__?: RuntimeConfig
  }
}

export {}
