import { useEffect, useState } from 'react'
import { Drawer } from 'antd'
import { getResourceYaml } from '@/services/k8s'

interface YamlDrawerProps {
  clusterId: number
  resourceType: string
  namespace?: string
  name?: string
  open: boolean
  onClose: () => void
}

/** 通用 YAML 查看抽屉组件 */
export default function YamlDrawer({ clusterId, resourceType, namespace, name, open, onClose }: YamlDrawerProps) {
  const [yaml, setYaml] = useState<string>('')
  const [loading, setLoading] = useState(false)

  const loadYaml = async () => {
    if (!name) return
    setLoading(true)
    try {
      const res = await getResourceYaml(clusterId, resourceType, namespace || 'default', name)
      setYaml(typeof res === 'string' ? res : (res.yaml || ''))
    } catch {
      setYaml('获取 YAML 失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (!open || !name) return
    setYaml('')
    void loadYaml()
  }, [open, name, namespace, resourceType, clusterId])

  return (
    <Drawer title={`YAML - ${name}`} open={open} onClose={() => { setYaml(''); onClose() }} width={800}>
      <pre style={{
        background: '#1e1e1e', color: '#d4d4d4', padding: 16, borderRadius: 8,
        height: 'calc(100vh - 200px)', overflow: 'auto', fontSize: 13, lineHeight: 1.6,
        fontFamily: 'Consolas, Monaco, monospace',
      }}>
        {loading ? '加载中...' : yaml || '暂无数据'}
      </pre>
    </Drawer>
  )
}

/** 管理 YAML 抽屉状态的 Hook */
export function useYamlDrawer() {
  const [state, setState] = useState<{ open: boolean; name?: string; namespace?: string }>({ open: false })

  const openYaml = (name: string, namespace?: string) => setState({ open: true, name, namespace })
  const closeYaml = () => setState({ open: false })

  return { ...state, openYaml, closeYaml }
}
