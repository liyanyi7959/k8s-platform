/**
 * 创建/编辑部署计划页
 */
import { useEffect, useMemo, useState } from 'react'
import { Alert, Button, Card, Checkbox, Empty, Form, Input, Radio, Select, Space, Steps, Typography, message, Descriptions, Tag } from 'antd'
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons'
import { history, useParams } from '@umijs/max'
import { useMutation, useQuery } from '@tanstack/react-query'
import { AppPage } from '@/components'
import { createDeployPlan, getDeployPlanById, getServers, updateDeployPlan } from '@/services/deploy'
import type { CreateDeployPlanRequest } from '@/types/deploy'

const { Title } = Typography

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

export default function CreateDeployPlanPage() {
  const params = useParams<{ id?: string }>()
  const planId = params.id ? Number(params.id) : undefined
  const isEditMode = Number.isFinite(planId)
  const [form] = Form.useForm()
  const [step, setStep] = useState(0)

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
    })
  }, [form, planDetail])

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
    if (masterCount !== 1) {
      throw new Error('当前部署模式必须且只能选择一个 Master 节点')
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
        await handleSubmit()
        return
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
      // step===0 的 Form.Item 在 step===1 时已卸载，validateFields 无法获取其值
      // 改用 getFieldsValue(true) 获取所有字段（含已卸载 Form.Item 的值）
      const values = form.getFieldsValue(true)
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

  const stepTitles = ['基础信息', '节点配置']

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
              message="当前模式支持 1 个 Master 和任意数量 Worker"
              description="只有状态为“可用”或“已注册”的服务器会出现在这里。多控制平面需要 VIP/负载均衡与专用 HA 流程，本向导会拒绝多个 Master。"
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
                                label: `${server.name} (${server.ip}) · ${server.os || '未知'} · ${server.status}`,
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

            <Form.Item name="addons" label="附加组件">
              <Checkbox.Group options={addonOptions} />
            </Form.Item>

            {/* 部署摘要 */}
            <Card size="small" title="部署摘要" style={{ marginTop: 16, background: '#fafafa' }}>
              <Descriptions column={2} size="small">
                <Descriptions.Item label="K8s 版本">
                  <Tag color="blue">{form.getFieldsValue(true).k8sVersion || '-'}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="CNI">
                  <Tag color="green">{form.getFieldsValue(true).cniType || '-'}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="Pod 网段">
                  <Typography.Text code>{form.getFieldsValue(true).podCidr || '-'}</Typography.Text>
                </Descriptions.Item>
                <Descriptions.Item label="Service 网段">
                  <Typography.Text code>{form.getFieldsValue(true).svcCidr || '-'}</Typography.Text>
                </Descriptions.Item>
              </Descriptions>
            </Card>
          </>
        )}
      </Form>

      <div style={{ marginTop: 24, textAlign: 'right' }}>
        <Space>
          <Button onClick={() => history.push('/deploy/plans')}>取消</Button>
          {step > 0 && <Button onClick={handlePrev}>上一步</Button>}
          {step < 1 && <Button type="primary" onClick={handleNext}>下一步</Button>}
          {step === 1 && (
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
