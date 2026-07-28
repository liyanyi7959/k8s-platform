import { useEffect, useState } from 'react'
import { Drawer, Button, Space, message } from 'antd'
import { EditOutlined, CheckOutlined, CloseOutlined } from '@ant-design/icons'
import { getResourceYaml, applyYaml } from '@/features/kops/api/k8s'
import { YamlEditor } from '@/components/YamlEditor'

interface YamlDrawerProps {
  clusterId: number
  resourceType: string
  namespace?: string
  name?: string
  open: boolean
  onClose: () => void
}

/** 通用 YAML 查看/编辑抽屉组件 */
export default function YamlDrawer({ clusterId, resourceType, namespace, name, open, onClose }: YamlDrawerProps) {
  const [yaml, setYaml] = useState<string>('')
  const [yamlOriginal, setYamlOriginal] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [editing, setEditing] = useState(false)
  const [applying, setApplying] = useState(false)

  const loadYaml = async () => {
    if (!name) return
    setLoading(true)
    try {
      const res = await getResourceYaml(clusterId, resourceType, namespace || 'default', name)
      const yamlText = typeof res === 'string' ? res : (res.yaml || '')
      setYaml(yamlText)
      setYamlOriginal(yamlText)
    } catch {
      setYaml('获取 YAML 失败')
    } finally {
      setLoading(false)
    }
  }

  const handleApply = async () => {
    setApplying(true)
    try {
      const res = await applyYaml(clusterId, yaml)
      if (res?.success) {
        message.success('YAML 应用成功')
        setEditing(false)
        setYamlOriginal(yaml)
      } else {
        message.error(res?.message || '应用失败')
      }
    } catch {
      message.error('YAML 应用失败')
    } finally {
      setApplying(false)
    }
  }

  useEffect(() => {
    if (!open || !name) return
    setYaml('')
    setEditing(false)
    void loadYaml()
  }, [open, name, namespace, resourceType, clusterId])

  return (
    <Drawer
      title={`YAML - ${name}`}
      open={open}
      onClose={() => { setYaml(''); setEditing(false); onClose() }}
      width={800}
      extra={
        editing ? (
          <Space>
            <Button type="primary" icon={<CheckOutlined />} loading={applying} onClick={handleApply}>
              应用
            </Button>
            <Button icon={<CloseOutlined />} onClick={() => { setYaml(yamlOriginal); setEditing(false) }}>
              取消
            </Button>
          </Space>
        ) : (
          !loading && yaml && yaml !== '获取 YAML 失败' ? (
            <Button icon={<EditOutlined />} onClick={() => setEditing(true)}>编辑</Button>
          ) : null
        )
      }
    >
      {loading ? (
        <span style={{ color: '#999' }}>加载中...</span>
      ) : (
        <YamlEditor
          value={yaml}
          readOnly={!editing}
          height={Math.max(window.innerHeight - 200, 400)}
          onChange={(val) => setYaml(val)}
        />
      )}
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
