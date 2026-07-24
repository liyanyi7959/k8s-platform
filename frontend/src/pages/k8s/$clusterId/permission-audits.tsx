import React, { useState } from 'react'
import {
  Card,
  Row,
  Col,
  Select,
  Input,
  Button,
  Table,
  Checkbox,
  Tag,
  Space,
  Typography,
  message,
  type ColumnsType,
} from 'antd'
import { DownloadOutlined, ThunderboltOutlined, SafetyOutlined } from '@ant-design/icons'
import { useQuery, useMutation } from '@tanstack/react-query'
import { AppPage, YamlEditor } from '@/components'
import AppAlert from '@/components/AppAlert'
import { listNamespaces, defaultRBACMatrix, buildRBACFromMatrix } from '@/services/k8s'
import { useClusterId } from '@/hooks/useClusterId'
import type { RBACMatrixRequest, RBACMatrixRow, Namespace } from '@/types'

const { Text } = Typography

/** K8s 标准 verbs */
const VERBS = ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] as const

/** verb → 标签颜色 */
const verbLabelColor: Record<string, string> = {
  get: 'blue',
  list: 'blue',
  watch: 'cyan',
  create: 'green',
  update: 'orange',
  patch: 'gold',
  delete: 'red',
}

const PermissionAuditsPage: React.FC = () => {
  const clusterId = useClusterId()
  const [saName, setSaName] = useState('xingku-platform')
  const [saNamespace, setSaNamespace] = useState('kube-system')
  const [selectedNamespaces, setSelectedNamespaces] = useState<string[]>([])
  const [matrix, setMatrix] = useState<RBACMatrixRequest | null>(null)
  const [yamlContent, setYamlContent] = useState('')

  // 命名空间列表
  const { data: namespaces } = useQuery({
    queryKey: ['k8s-namespaces', clusterId],
    queryFn: ({ signal }) => listNamespaces(Number(clusterId), signal),
    enabled: !!clusterId,
  })

  // 加载默认权限矩阵（用户主动触发）
  const loadMutation = useMutation({
    mutationFn: () => defaultRBACMatrix(Number(clusterId), selectedNamespaces),
    onSuccess: (data) => {
      setMatrix({
        ...data,
        service_account: saName,
        sa_namespace: saNamespace,
        target_namespaces: [...selectedNamespaces],
      })
      setYamlContent('')
      message.success('权限矩阵已加载')
    },
    onError: () => message.error('加载权限矩阵失败'),
  })

  // 生成 RBAC YAML
  const generateMutation = useMutation({
    mutationFn: () => buildRBACFromMatrix(Number(clusterId), matrix!),
    onSuccess: (res) => {
      setYamlContent(res.yaml_content)
      message.success('RBAC YAML 生成成功')
    },
    onError: () => message.error('YAML 生成失败'),
  })

  // 勾选/取消勾选某个 verb
  const toggleVerb = (scope: 'cluster' | 'namespace', rowIndex: number, verb: string) => {
    if (!matrix) return
    const rows = scope === 'cluster' ? [...matrix.cluster_rows] : [...matrix.namespace_rows]
    const row = rows[rowIndex]
    if (!row) return
    const hasVerb = row.verbs.includes(verb)
    rows[rowIndex] = {
      ...row,
      verbs: hasVerb ? row.verbs.filter((v) => v !== verb) : [...row.verbs, verb],
    }
    setMatrix(
      scope === 'cluster'
        ? { ...matrix, cluster_rows: rows }
        : { ...matrix, namespace_rows: rows },
    )
    // 矩阵变更后清空旧 YAML
    setYamlContent('')
  }

  // 构建网格列定义
  const buildColumns = (scope: 'cluster' | 'namespace'): ColumnsType<RBACMatrixRow> => [
    {
      title: '资源类型',
      dataIndex: 'label',
      width: 220,
      fixed: 'left',
      render: (_, record) => (
        <div>
          <Text strong>{record.label}</Text>
          <div style={{ fontSize: 11, color: '#94a3b8' }}>
            {record.api_group || 'core'} · {record.resources.join(', ')}
          </div>
        </div>
      ),
    },
    ...VERBS.map((verb) => ({
      title: (
        <Tag color={verbLabelColor[verb]} style={{ margin: 0 }}>
          {verb}
        </Tag>
      ),
      dataIndex: verb,
      width: 80,
      align: 'center' as const,
      render: (_: unknown, record: RBACMatrixRow, rowIndex: number) => (
        <Checkbox
          checked={record.verbs.includes(verb)}
          onChange={() => toggleVerb(scope, rowIndex, verb)}
        />
      ),
    })),
  ]

  const handleLoadMatrix = () => {
    if (selectedNamespaces.length === 0) {
      message.warning('请先选择目标命名空间')
      return
    }
    loadMutation.mutate()
  }

  const handleDownload = () => {
    if (!yamlContent) return
    const blob = new Blob([yamlContent], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `rbac-${saName}.yaml`
    a.click()
    URL.revokeObjectURL(url)
  }

  const nsOptions = (Array.isArray(namespaces) ? namespaces : []).map((ns: Namespace) => ({
    label: ns.name,
    value: ns.name,
  }))

  return (
    <AppPage breadcrumbRender={false}>
      <div className="app-permission-audit">
        {/* ===== 配置区 ===== */}
        <Card bordered={false} className="app-permission-audit__config">
          <Row gutter={[16, 16]} align="bottom">
            <Col xs={24} sm={12} lg={6}>
              <div className="app-permission-audit__field">
                <Text type="secondary" className="app-permission-audit__field-label">
                  ServiceAccount 名称
                </Text>
                <Input
                  value={saName}
                  onChange={(e) => setSaName(e.target.value)}
                  placeholder="如 xingku-platform"
                />
              </div>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <div className="app-permission-audit__field">
                <Text type="secondary" className="app-permission-audit__field-label">
                  SA 所在命名空间
                </Text>
                <Input
                  value={saNamespace}
                  onChange={(e) => setSaNamespace(e.target.value)}
                  placeholder="如 kube-system"
                />
              </div>
            </Col>
            <Col xs={24} lg={8}>
              <div className="app-permission-audit__field">
                <Text type="secondary" className="app-permission-audit__field-label">
                  目标命名空间（多选）
                </Text>
                <Select
                  mode="multiple"
                  style={{ width: '100%' }}
                  placeholder="选择目标命名空间"
                  value={selectedNamespaces}
                  onChange={(val) => {
                    setSelectedNamespaces(val)
                    // 清空命名空间时同步清空矩阵和 YAML
                    if (val.length === 0) {
                      setMatrix(null)
                      setYamlContent('')
                    }
                  }}
                  options={nsOptions}
                  optionFilterProp="label"
                  maxTagCount="responsive"
                />
              </div>
            </Col>
            <Col xs={24} lg={4}>
              <Button
                type="primary"
                icon={<SafetyOutlined />}
                onClick={handleLoadMatrix}
                loading={loadMutation.isPending}
                block
              >
                加载权限矩阵
              </Button>
            </Col>
          </Row>
        </Card>

        {/* ===== 权限网格 ===== */}
        {matrix ? (
          <>
            <Card
              bordered={false}
              className="app-permission-audit__grid"
              title={
                <Space>
                  <Tag color="purple">集群级</Tag>
                  <span>权限规则</span>
                </Space>
              }
            >
              <Table
                dataSource={matrix.cluster_rows}
                columns={buildColumns('cluster')}
                rowKey={(_, i) => `cluster-${i}`}
                pagination={false}
                size="small"
                scroll={{ x: 780 }}
              />
            </Card>

            <Card
              bordered={false}
              className="app-permission-audit__grid"
              title={
                <Space>
                  <Tag color="blue">命名空间级</Tag>
                  <span>权限规则</span>
                </Space>
              }
            >
              <Table
                dataSource={matrix.namespace_rows}
                columns={buildColumns('namespace')}
                rowKey={(_, i) => `ns-${i}`}
                pagination={false}
                size="small"
                scroll={{ x: 780 }}
              />
            </Card>

            {/* ===== YAML 生成区 ===== */}
            <Card bordered={false} className="app-permission-audit__yaml">
              <Space style={{ marginBottom: 12 }}>
                <Button
                  type="primary"
                  icon={<ThunderboltOutlined />}
                  onClick={() => generateMutation.mutate()}
                  loading={generateMutation.isPending}
                >
                  生成 RBAC YAML
                </Button>
                {yamlContent ? (
                  <Button icon={<DownloadOutlined />} onClick={handleDownload}>
                    下载 YAML
                  </Button>
                ) : null}
              </Space>
              {yamlContent ? (
                <YamlEditor value={yamlContent} readOnly height={500} />
              ) : (
                <AppAlert
                  type="info"
                  showIcon
                  message="调整权限勾选后，点击「生成 RBAC YAML」生成标准 K8s RBAC 配置"
                />
              )}
            </Card>
          </>
        ) : (
          <Card bordered={false}>
            <div className="app-permission-audit__empty">
              <SafetyOutlined style={{ fontSize: 48, color: '#d6e4ff' }} />
              <div style={{ marginTop: 16, color: '#94a3b8' }}>
                选择目标命名空间并点击「加载权限矩阵」开始生成 RBAC 权限配置
              </div>
            </div>
          </Card>
        )}
      </div>
    </AppPage>
  )
}

export default PermissionAuditsPage
