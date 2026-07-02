/**
 * 全局样式
 */
import { createStyles } from 'antd-style'

const useStyles = createStyles(() => {
  return {
    body: {
      margin: 0,
      padding: 0,
      fontFamily:
        '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
    },
  }
})

export default function GlobalStyle() {
  useStyles()
  return null
}
