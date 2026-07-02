/**
 * 创建/编辑部署计划页
 */
import { useEffect, useMemo, useState } from 'react'
import {
  Alert, Button, Card, Checkbox, Empty, Form, Input, Radio, Select, Space, Steps, Tag, Typography, message, Divider,
} from 'antd'
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons'
import { history, useParams } from '@umijs/max'
import { useMutation, useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import TerminalCodeBlock from '@/components/TerminalCodeBlock'
import { createDeployPlan, getDeployPlanById, getServers, updateDeployPlan } from '@/services/deploy'
import type { CreateDeployPlanRequest, DeployPlan, DeployPlanStepOverride } from '@/types/deploy'

const { Text, Title } = Typography

const versionOptions = ['v1.31.0', 'v1.30.0', 'v1.29.0', 'v1.28.0']
const cniOptions = [
  { label: 'Flannel', value: 'flannel' },
  { label: 'Calico', value: 'calico' },
  { label: 'Cilium', value: 'cilium' },
]
const addonOptions = [
  { label: 'metrics-server', value: 'metrics-server' },
  { label: 'ingress-nginx', value: 'ingress-nginx' },
  { label: 'local-storage', value: 'local-storage' },
]

const stepOverrideDefinitions = [
  { stepKey: 'pre_check', stepName: '环境预检', roles: ['master', 'worker'] },
  { stepKey: 'bootstrap', stepName: '基础环境初始化', roles: ['master', 'worker'] },
  { stepKey: 'init_master', stepName: '初始化 Master', roles: ['master'] },
  { stepKey: 'join_workers', stepName: 'Worker 加入集群', roles: ['worker'] },
  { stepKey: 'install_cni', stepName: '安装网络插件', roles: ['master'] },
  { stepKey: 'register', stepName: '注册集群', roles: ['master'] },
]

type StepOverrideFormItem = {
  key: string
  stepKey: string
  nodeServerId: number
  nodeRole: 'master' | 'worker'
  commandTemplate: string
  description?: string
}

export default function CreateDeployPlanPage() {
  const params = useParams<{ id?: string }>()
  const planId = params.id ? Number(params.id) : undefined
  const isEditMode = Number.isFinite(planId)
  const [form] = Form.useForm()
  const [step, setStep] = useState(0)

  const watchedNodes = Form.useWatch('nodes', form) || []
  const watchedName = Form.useWatch('name', form)
  const watchedClusterName = Form.useWatch('clusterName', form)
  const watchedVersion = Form.useWatch('k8sVersion', form)
  const watchedCni = Form.useWatch('cniType', form)
  const watchedPodCidr = Form.useWatch('podCidr', form)
  const watchedSvcCidr = Form.useWatch('svcCidr', form)
  const watchedAddons = Form.useWatch('addons', form) || []
  const watchedStepOverrides = Form.useWatch('stepOverrides', form) || []

  const { data: serversData, isLoading: serversLoading } = useQuery({
    queryKey: ['deploy-servers-for-plan-create'],
    queryFn: () => getServers({ page: 1, pageSize: 200 }),
  })

  const { data: planDetail, isLoading: planLoading } = useQuery({
    queryKey: ['deploy-plan-detail', planId],
    queryFn: () => getDeployPlanById(planId!),
    enabled: isEditMode,
  })

  const availableServers = useMemo(
    () => (serversData?.items || []).filter((server) => ['available', 'registered'].includes(server.status)),
    [serversData],
  )

  const submitMutation = useMutation({
    mutationFn: async (data: CreateDeployPlanRequest) => {
      if (isEditMode) {
        await updateDeployPlan(planId!, data)
      } else {
        await createDeployPlan(data)
      }
    },
    onSuccess: () => {
      message.success(isEditMode ? '部署方案更新成功' : '部署方案创建成功')
      history.push('/deploy/plans')
    },
    onError: (err: any) => message.error(err?.message || (isEditMode ? '更新失败' : '创建失败')),
  })

  useEffect(() => {
    if (!planDetail) return
    const overrideItems = mapPlanOverridesToForm(planDetail)
    form.setFieldsValue({
      name: planDetail.name,
      clusterName: planDetail.clusterName,
      k8sVersion: planDetail.k8sVersion,
      podCidr: planDetail.podCidr,
      svcCidr: planDetail.svcCidr,
      cniType: planDetail.cniType,
      addons: planDetail.addons || [],
      nodes: (planDetail.nodes || []).map((node) => ({
        serverId: node.serverId,
        role: node.role,
      })),
      stepOverrides: overrideItems,
    })
  }, [form, planDetail])

  const getServerLabel = (serverId?: number) => {
    if (!serverId) return '-'
    const server = availableServers.find((item) => item.id === serverId)
    return server ? `${server.name} (${server.ip})` : `#${serverId}`
  }

  const validateNodeSelection = async () => {
    const nodes = (form.getFieldValue('nodes') || []) as Array<{ serverId?: number; role?: 'master' | 'worker' }>
    if (nodes.length === 0) {
      throw new Error('请至少添加一个部署节点')
    }
    const seen = new Set<number>()
    let masterCount = 0
    for (const node of nodes) {
      if (!node.serverId || !node.role) {
        throw new Error('请完整选择每个节点的服务器和角色')
      }
      if (seen.has(node.serverId)) {
        throw new Error('同一台服务器不能重复加入部署计划')
      }
      seen.add(node.serverId)
      if (node.role === 'master') {
        masterCount += 1
      }
    }
    if (masterCount === 0) {
      throw new Error('至少需要一个 master 节点')
    }
  }

  const handleNext = async () => {
    try {
      if (step === 0) {
        await form.validateFields(['name', 'clusterName', 'k8sVersion', 'podCidr', 'svcCidr', 'cniType'])
      }
      if (step === 1) {
        await form.validateFields(['nodes'])
        await validateNodeSelection()
      }
      setStep(step + 1)
    } catch (error) {
      if (error instanceof Error && error.message) {
        message.error(error.message)
      }
    }
  }

  const handlePrev = () => {
    setStep(step - 1)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      await validateNodeSelection()

      const data: CreateDeployPlanRequest = {
        name: values.name,
        clusterName: values.clusterName,
        k8sVersion: values.k8sVersion,
        podCidr: values.podCidr,
        svcCidr: values.svcCidr,
        cniType: values.cniType,
        cniConfig: {},
        addons: values.addons || [],
        stepOverrides: buildStepOverrides(values.stepOverrides || []),
        nodes: (values.nodes || []).map((node: { serverId: number; role: 'master' | 'worker' }, index: number) => ({
          serverId: Number(node.serverId),
          role: node.role,
          sortOrder: index + 1,
        })),
      }

      submitMutation.mutate(data)
    } catch (error) {
      if (error instanceof Error && error.message) {
        message.error(error.message)
      }
    }
  }

  const stepTitles = ['基础信息', '节点配置', '命令覆盖']
  const nodeOptions = watchedNodes
    .map((node: { serverId?: number; role?: 'master' | 'worker' }, index: number) => {
      if (!node.serverId || !node.role) return null
      return {
        label: `节点 ${index + 1} · ${getServerLabel(node.serverId)} · ${node.role}`,
        value: `${node.serverId}:${node.role}`,
        serverId: Number(node.serverId),
        role: node.role,
      }
    })
    .filter(Boolean) as Array<{ label: string; value: string; serverId: number; role: 'master' | 'worker' }>

  return (
    <AppPage>
      <Card style={{ maxWidth: 960, margin: '0 auto' }}>
        <Title level={4} style={{ marginTop: 0 }}>
          {isEditMode ? '编辑部署方案' : '创建部署方案'}
        </Title>
        <Steps current={step} items={stepTitles.map((t) => ({ title: t }))} style={{ marginBottom: 24 }} />

      <Form
        form={form}
        layout="vertical"
        initialValues={{
          k8sVersion: 'v1.31.0',
          podCidr: '10.244.0.0/16',
          svcCidr: '10.96.0.0/12',
          cniType: 'flannel',
          addons: ['metrics-server'],
          nodes: [{ role: 'master' }],
          stepOverrides: [],
        }}
      >
        {step === 0 && (
          <>
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
              message="在线部署会基于已录入的 SSH 服务器和凭据生成部署计划"
              description="建议先在“凭据管理”和“服务器管理”中完成前置资源准备，再创建部署计划。"
            />
            <Form.Item name="name" label="计划名称" rules={[{ required: true, message: '请输入计划名称' }]}> 
              <Input placeholder="例如：生产环境 K8s 集群部署" maxLength={100} />
            </Form.Item>
            <Form.Item name="clusterName" label="集群名称" rules={[{ required: true, message: '请输入集群名称' }]}> 
              <Input placeholder="例如：prod-k8s-01" maxLength={100} />
            </Form.Item>
            <Form.Item name="k8sVersion" label="K8s 版本" rules={[{ required: true, message: '请选择 K8s 版本' }]}> 
              <Select options={versionOptions.map((value) => ({ label: value, value }))} />
            </Form.Item>
            <Form.Item name="podCidr" label="Pod 网段" rules={[{ required: true, message: '请输入 Pod 网段' }]}> 
              <Input placeholder="10.244.0.0/16" />
            </Form.Item>
            <Form.Item name="svcCidr" label="Service 网段" rules={[{ required: true, message: '请输入 Service 网段' }]}> 
              <Input placeholder="10.96.0.0/12" />
            </Form.Item>
            <Form.Item name="cniType" label="CNI 类型" rules={[{ required: true, message: '请选择 CNI 类型' }]}> 
              <Radio.Group optionType="button" buttonStyle="solid" options={cniOptions} />
            </Form.Item>
          </>
        )}

        {step === 1 && (
          <>
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
              message="至少需要一个 master 节点"
              description="只有状态为“可用”或“已注册”的服务器会出现在这里。若列表为空，请先到服务器管理页完成录入和 SSH 测试。"
            />

            {availableServers.length === 0 ? (
              <Empty description="暂无可用于部署的服务器">
                <Button type="primary" onClick={() => history.push('/deploy/servers')}>前往服务器管理</Button>
              </Empty>
            ) : (
              <Form.List name="nodes">
                {(fields, { add, remove }) => (
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    {fields.map((field, index) => (
                      <Card
                        key={field.key}
                        size="small"
                        title={`节点 ${index + 1}`}
                        extra={fields.length > 1 ? <Button type="text" danger icon={<MinusCircleOutlined />} onClick={() => remove(field.name)} /> : null}
                      >
                        <Space align="start" wrap style={{ width: '100%' }}>
                          <Form.Item
                            {...field}
                            name={[field.name, 'serverId']}
                            label="服务器"
                            rules={[{ required: true, message: '请选择服务器' }]}
                            style={{ minWidth: 320, marginBottom: 0 }}
                          >
                            <Select
                              showSearch
                              optionFilterProp="label"
                              placeholder="选择服务器"
                              loading={serversLoading}
                              options={availableServers.map((server) => ({
                                label: `${server.name} (${server.ip})`,
                                value: server.id,
                              }))}
                            />
                          </Form.Item>
                          <Form.Item
                            {...field}
                            name={[field.name, 'role']}
                            label="角色"
                            rules={[{ required: true, message: '请选择角色' }]}
                            style={{ minWidth: 180, marginBottom: 0 }}
                          >
                            <Select
                              options={[
                                { label: 'Master', value: 'master' },
                                { label: 'Worker', value: 'worker' },
                              ]}
                            />
                          </Form.Item>
                        </Space>
                      </Card>
                    ))}
                    <Button icon={<PlusOutlined />} onClick={() => add({ role: 'worker' })}>添加节点</Button>
                  </Space>
                )}
              </Form.List>
            )}
          </>
        )}

        {step === 2 && (
          <>
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
              message="命令覆盖仅作用于当前部署计划"
              description="这里配置的是计划级覆盖，不会修改部署模板。你可以针对某个节点的某个步骤单独改命令。"
            />

            <Form.Item name="addons" label="附加组件">
              <Checkbox.Group options={addonOptions} />
            </Form.Item>

            <Form.List name="stepOverrides">
              {(fields, { add, remove }) => (
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  {fields.map((field) => {
                    const currentValue = form.getFieldValue(['stepOverrides', field.name]) as StepOverrideFormItem | undefined
                    const selectedDefinition = stepOverrideDefinitions.find((item) => item.stepKey === currentValue?.stepKey)
                    return (
                      <Card
                        key={field.key}
                        size="small"
                        title={currentValue?.stepKey ? `${selectedDefinition?.stepName || currentValue.stepKey} 覆盖` : '步骤命令覆盖'}
                        extra={<Button type="text" danger icon={<MinusCircleOutlined />} onClick={() => remove(field.name)} />}
                      >
                        <Space direction="vertical" size={12} style={{ width: '100%' }}>
                          <Space align="start" wrap style={{ width: '100%' }}>
                            <Form.Item
                              {...field}
                              name={[field.name, 'stepKey']}
                              label="步骤"
                              rules={[{ required: true, message: '请选择步骤' }]}
                              style={{ minWidth: 220, marginBottom: 0 }}
                            >
                              <Select
                                options={stepOverrideDefinitions.map((item) => ({ label: item.stepName, value: item.stepKey }))}
                              />
                            </Form.Item>
                            <Form.Item
                              {...field}
                              name={[field.name, 'nodeRef']}
                              label="目标节点"
                              rules={[{ required: true, message: '请选择目标节点' }]}
                              style={{ minWidth: 360, marginBottom: 0 }}
                            >
                              <Select
                                options={nodeOptions}
                                onChange={(value) => {
                                  const match = nodeOptions.find((item) => item.value === value)
                                  form.setFieldValue(['stepOverrides', field.name, 'nodeServerId'], match?.serverId)
                                  form.setFieldValue(['stepOverrides', field.name, 'nodeRole'], match?.role)
                                }}
                              />
                            </Form.Item>
                          </Space>
                          <Form.Item
                            {...field}
                            name={[field.name, 'description']}
                            label="覆盖说明"
                          >
                            <Input placeholder="例如：这台机器需要使用内网仓库或特殊依赖源" />
                          </Form.Item>
                          <Form.Item
                            {...field}
                            name={[field.name, 'commandTemplate']}
                            label="命令模板"
                            rules={[{ required: true, message: '请输入覆盖命令' }]}
                          >
                            <Input.TextArea rows={8} style={{ fontFamily: 'monospace' }} />
                          </Form.Item>
                          <TerminalCodeBlock
                            title={`${currentValue?.stepKey || 'step'}.${currentValue?.nodeRole || 'node'}.override.sh`}
                            content={currentValue?.commandTemplate}
                          />
                        </Space>
                      </Card>
                    )
                  })}
                  <Button
                    icon={<PlusOutlined />}
                    onClick={() => add({ enabled: true })}
                    disabled={nodeOptions.length === 0}
                  >
                    添加步骤命令覆盖
                  </Button>
                </Space>
              )}
            </Form.List>

            <Divider />

            <Card size="small" title="部署摘要" style={{ background: '#fafafa' }}>
              <Space direction="vertical" style={{ width: '100%' }}>
                <Text>计划名称：<Text strong>{watchedName || '-'}</Text></Text>
                <Text>集群名称：<Text strong>{watchedClusterName || '-'}</Text></Text>
                <Text>K8s 版本：<Tag color="blue">{watchedVersion || '-'}</Tag></Text>
                <Text>CNI：<Tag color="green">{watchedCni || '-'}</Tag></Text>
                <Text>Pod 网段：<Text code>{watchedPodCidr || '-'}</Text></Text>
                <Text>Service 网段：<Text code>{watchedSvcCidr || '-'}</Text></Text>
                <Text>
                  节点：
                  <Space wrap>
                    {watchedNodes.length > 0 ? watchedNodes.map((node: { serverId?: number; role?: string }, index: number) => (
                      <Tag key={`${node.serverId || 'unknown'}-${index}`} color={node.role === 'master' ? 'red' : 'blue'}>
                        {getServerLabel(node.serverId)} / {node.role || '未选角色'}
                      </Tag>
                    )) : '-'}
                  </Space>
                </Text>
                <Text>
                  附加组件：
                  <Space wrap>
                    {watchedAddons.length > 0 ? watchedAddons.map((addon: string) => <Tag key={addon}>{addon}</Tag>) : <Text type="secondary">无</Text>}
                  </Space>
                </Text>
                <Text>命令覆盖：<Text strong>{watchedStepOverrides.length || 0}</Text></Text>
              </Space>
            </Card>
          </>
        )}
      </Form>

      <div style={{ marginTop: 24, textAlign: 'right' }}>
        <Space>
          <Button onClick={() => history.push('/deploy/plans')}>取消</Button>
          {step > 0 && <Button onClick={handlePrev}>上一步</Button>}
          {step < 2 && <Button type="primary" onClick={handleNext}>下一步</Button>}
          {step === 2 && (
            <Button type="primary" loading={submitMutation.isPending || planLoading} onClick={handleSubmit}>
              {isEditMode ? '保存部署方案' : '创建部署方案'}
            </Button>
          )}
        </Space>
      </div>
      </Card>
    </AppPage>
  )
}

function buildStepOverrides(items: StepOverrideFormItem[]): Record<string, DeployPlanStepOverride> | undefined {
  if (!items.length) return undefined
  const entries = items
    .filter((item) => item.stepKey && item.nodeServerId && item.commandTemplate?.trim())
    .map((item) => ([
      `${item.stepKey}#server:${item.nodeServerId}`,
      {
        stepKey: item.stepKey,
        nodeRole: item.nodeRole,
        nodeServerId: item.nodeServerId,
        commandTemplate: item.commandTemplate,
        description: item.description,
      },
    ]))
  if (!entries.length) return undefined
  return Object.fromEntries(entries)
}

function mapPlanOverridesToForm(plan: DeployPlan): StepOverrideFormItem[] {
  if (!plan.stepOverrides) return []
  return Object.entries(plan.stepOverrides).map(([key, item]) => ({
    key,
    stepKey: item.stepKey,
    nodeServerId: item.nodeServerId || 0,
    nodeRole: (item.nodeRole as 'master' | 'worker') || 'worker',
    commandTemplate: item.commandTemplate,
    description: item.description,
    nodeRef: item.nodeServerId ? `${item.nodeServerId}:${item.nodeRole || 'worker'}` : undefined,
  })) as StepOverrideFormItem[]
}
