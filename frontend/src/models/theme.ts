/**
 * 主题全局状态模型
 * 管理暗色/亮色主题切换
 */
import { useState, useCallback, useEffect } from 'react'

type ThemeMode = 'light' | 'dark'

export default function useThemeModel() {
  const [mode, setMode] = useState<ThemeMode>(() => {
    const saved = localStorage.getItem('theme-mode')
    return (saved as ThemeMode) || 'light'
  })

  useEffect(() => {
    localStorage.setItem('theme-mode', mode)
    document.documentElement.setAttribute('data-theme', mode)
  }, [mode])

  const toggleTheme = useCallback(() => {
    setMode((prev) => (prev === 'light' ? 'dark' : 'light'))
  }, [])

  const setTheme = useCallback((newMode: ThemeMode) => {
    setMode(newMode)
  }, [])

  return {
    mode,
    isDark: mode === 'dark',
    toggleTheme,
    setTheme,
  }
}
