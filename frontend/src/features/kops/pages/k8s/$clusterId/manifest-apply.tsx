import React, { useMemo, useState } from 'react'
import { Badge, Button, Card, Col, Descriptions, Empty, Input, List, Modal, Row, Select, Space, Table, Tag, Typography, message } from 'antd'
import { CheckCircleOutlined, CloudUploadOutlined, CodeOutlined, FileSearchOutlined, SearchOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AppPage, NamespaceSelector, YamlEditor } from '@/components'
import AppAlert from '@/components/AppAlert'
import { applyYaml, listManifestRecords, type ManifestApplyResult } from '@/features/kops/api/k8s'
import { useClusterId } from '@/hooks/useClusterId'
import { formatDate } from '@/utils'
import { manifestTemplates, renderManifestTemplate } from './manifestTemplates'
import type { YamlDiagnosticSummary } from '@/components/YamlEditor/kubernetesYamlLanguage'

const { Text, Title } = Typography

const ManifestApplyPage: React.FC = () => {
  const clusterId = useClusterId()
  const queryClient = useQueryClient()
  const [namespace, setNamespace] = useState('default')
  const [search, setSearch] = useState('')
  const [category, setCategory] = useState('全部')
  const [yaml, setYaml] = useState(() => renderManifestTemplate(manifestTemplates[0]!, 'default'))
  const [selectedTemplate, setSelectedTemplate] = useState(manifestTemplates[0]!.key)
  const [result, setResult] = useState<ManifestApplyResult | null>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [diagnostics, setDiagnostics] = useState<YamlDiagnosticSummary>({ errors: 0, warnings: 0 })

  const categories = ['全部', ...Array.from(new Set(manifestTemplates.map((item) => item.category)))]
  const filteredTemplates = useMemo(() => manifestTemplates.filter((item) => {
    const keyword = search.trim().toLowerCase()
    return (category === '全部' || item.category === category) &&
      (!keyword || `${item.title} ${item.description} ${item.category}`.toLowerCase().includes(keyword))
  }), [category, search])

  const recordsQuery = useQuery({
    queryKey: ['manifest-records', clusterId],
    queryFn: ({ signal }) => listManifestRecords(clusterId, { pageSize: 8 }, signal),
    enabled: !!clusterId,
  })

  const applyMutation = useMutation({
    mutationFn: (dryRun: boolean) => applyYaml(clusterId, yaml, {
      defaultNamespace: namespace,
      dryRun,
      sourceLabel: selectedTemplate ? `YAML 模板/${selectedTemplate}` : 'YAML 工作台',
    }),
    onSuccess: (response, dryRun) => {
      setResult(response)
      message.success(dryRun ? '服务端 DryRun 校验通过' : 'Manifest 已应用到集群')
      queryClient.invalidateQueries({ queryKey: ['manifest-records', clusterId] })
      if (!dryRun) setConfirmOpen(false)
    },
    onError: (error: any) => message.error(error?.message || 'YAML 处理失败'),
  })

  const selectTemplate = (key: string) => {
    const template = manifestTemplates.find((item) => item.key === key)
    if (!template) return
    setSelectedTemplate(key)
    setYaml(renderManifestTemplate(template, namespace))
    setResult(null)
  }

  const updateNamespace = (value: string) => {
    const next = value || 'default'
    const previous = namespace || 'default'
    setNamespace(next)
    setYaml((current) => current.replaceAll(`namespace: ${previous}`, `namespace: ${next}`))
  }

  return (
    <AppPage>
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Card size="small">
          <Row justify="space-between" align="middle" gutter={[12, 12]}>
            <Col>
              <Title level={4} style={{ margin: 0 }}><CodeOutlined /> YAML 部署工作台</Title>
              <Text type="secondary">模板生成、在线编辑、服务端 DryRun、变更确认与部署审计一体化</Text>
            </Col>
            <Col>
              <Space wrap>
                <NamespaceSelector clusterId={clusterId} value={namespace} onChange={updateNamespace} style={{ width: 210 }} showAll={false} allowClear={false} />
                <Button icon={<FileSearchOutlined />} loading={applyMutation.isPending} disabled={diagnostics.errors > 0} onClick={() => applyMutation.mutate(true)}>DryRun 校验</Button>
                <Button type="primary" icon={<CloudUploadOutlined />} disabled={!yaml.trim() || diagnostics.errors > 0} onClick={() => setConfirmOpen(true)}>应用到集群</Button>
              </Space>
            </Col>
          </Row>
        </Card>

        <Row gutter={12}>
          <Col xs={24} lg={6} xl={5}>
            <Card size="small" title="资源片段" extra={<Badge count={filteredTemplates.length} showZero color="#2563eb" />} styles={{ body: { padding: 12 } }}>
              <AppAlert type="info" showIcon message="快捷生成" description={<span>在编辑器空行输入 <Text keyboard>pod</Text>、<Text keyboard>deploy</Text>、<Text keyboard>svc</Text> 后按 Tab。</span>} style={{ marginBottom: 12 }} />
              <Input allowClear prefix={<SearchOutlined />} placeholder="搜索资源模板" value={search} onChange={(event) => setSearch(event.target.value)} style={{ marginBottom: 8 }} />
              <Select value={category} onChange={setCategory} options={categories.map((value) => ({ label: value === '全部' ? '全部资源类型' : value, value }))} style={{ width: '100%', marginBottom: 8 }} />
              <List
                size="small"
                dataSource={filteredTemplates}
                locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="未找到模板" /> }}
                style={{ maxHeight: 480, overflow: 'auto' }}
                renderItem={(item) => (
                  <List.Item onClick={() => selectTemplate(item.key)} style={{ cursor: 'pointer', padding: '10px 8px', borderRadius: 8, background: selectedTemplate === item.key ? '#eff6ff' : undefined, borderInlineStart: selectedTemplate === item.key ? '3px solid #2563eb' : '3px solid transparent' }}>
                    <div style={{ width: '100%', minWidth: 0 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8, alignItems: 'center' }}>
                        <Text strong ellipsis style={{ minWidth: 0 }}>{item.title}</Text>
                        <Text type="secondary" style={{ fontSize: 11, flex: 'none' }}>{item.category}</Text>
                      </div>
                      <Text type="secondary" ellipsis style={{ display: 'block', fontSize: 12, marginTop: 3 }}>{item.description}</Text>
                    </div>
                  </List.Item>
                )}
              />
            </Card>
          </Col>
          <Col xs={24} lg={18} xl={19}>
            <Card size="small" title={<Space>manifest.yaml {diagnostics.errors > 0 ? <Tag color="error">{diagnostics.errors} 个错误</Tag> : <Tag color="success">语法正常</Tag>} {diagnostics.warnings > 0 && <Tag color="warning">{diagnostics.warnings} 个警告</Tag>}</Space>} extra={<Text type="secondary">Ctrl+Space 联想 · Tab 展开片段 · 支持多文档</Text>}>
              {result && <AppAlert closable showIcon type="success" message={result.summary || (result.dry_run ? 'DryRun 校验通过' : '应用成功')} onClose={() => setResult(null)} style={{ marginBottom: 12 }} />}
              <YamlEditor value={yaml} onChange={setYaml} height={590} kubernetes onDiagnosticsChange={setDiagnostics} />
              <div style={{ borderTop: '1px solid #f0f0f0', paddingTop: 8, marginTop: 8 }}>
                <Space size={16}>
                  <Badge status={diagnostics.errors ? 'error' : 'success'} text={diagnostics.errors ? `${diagnostics.errors} 个错误，修复后才能部署` : 'YAML 基础语法与 Kubernetes 结构检查通过'} />
                  {diagnostics.warnings > 0 && <Badge status="warning" text={`${diagnostics.warnings} 个警告`} />}
                  <Text type="secondary">最终字段合法性由目标集群 DryRun 校验</Text>
                </Space>
              </div>
            </Card>
          </Col>
        </Row>

        <Card size="small" title="最近部署记录" extra={<Button type="link" onClick={() => recordsQuery.refetch()}>刷新</Button>}>
          <Table size="small" rowKey="id" loading={recordsQuery.isLoading} dataSource={recordsQuery.data?.list || []} pagination={false} columns={[
            { title: '时间', dataIndex: 'created_at', width: 180, render: (value) => value ? formatDate(value) : '-' },
            { title: '来源', dataIndex: 'source_label', ellipsis: true },
            { title: '命名空间', dataIndex: 'default_namespace', width: 140, render: (value) => value || '清单指定' },
            { title: '模式', dataIndex: 'dry_run', width: 100, render: (value) => <Tag color={value ? 'blue' : 'green'}>{value ? 'DryRun' : 'Apply'}</Tag> },
            { title: '状态', dataIndex: 'status', width: 100, render: (value) => <Tag color={value === 'success' ? 'success' : value === 'failed' ? 'error' : 'processing'}>{value}</Tag> },
            { title: '结果', dataIndex: 'summary', ellipsis: true },
          ]} />
        </Card>
      </Space>

      <Modal title="确认应用 Manifest" open={confirmOpen} onCancel={() => setConfirmOpen(false)} onOk={() => applyMutation.mutate(false)} confirmLoading={applyMutation.isPending} okText="确认应用" okButtonProps={{ danger: true }}>
        <AppAlert type="warning" showIcon message="该操作会直接创建或更新集群资源" description="建议先执行 DryRun。请再次确认目标集群、命名空间和清单内容。" style={{ marginBottom: 16 }} />
        <Descriptions size="small" column={1} bordered>
          <Descriptions.Item label="目标命名空间">{namespace}</Descriptions.Item>
          <Descriptions.Item label="模板">{manifestTemplates.find((item) => item.key === selectedTemplate)?.title || '自定义 YAML'}</Descriptions.Item>
          <Descriptions.Item label="文档数">{yaml.split(/^---\s*$/m).filter((item) => item.trim()).length}</Descriptions.Item>
          <Descriptions.Item label="校验建议"><CheckCircleOutlined style={{ color: '#16a34a' }} /> 使用 Kubernetes API DryRun</Descriptions.Item>
        </Descriptions>
      </Modal>
    </AppPage>
  )
}

export default ManifestApplyPage
