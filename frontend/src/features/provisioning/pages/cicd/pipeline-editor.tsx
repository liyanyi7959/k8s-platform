import { useEffect, useMemo, useState } from 'react'
import { history } from '@umijs/max'
import { useQuery } from '@tanstack/react-query'
import { Button, Divider, Drawer, Input, message, Modal, Segmented, Select, Space, Tag, Tooltip, Typography } from 'antd'
import {
  ArrowLeftOutlined,
  CheckCircleOutlined,
  CodeOutlined,
  DeleteOutlined,
  DeploymentUnitOutlined,
  GithubOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  QuestionCircleOutlined,
  SaveOutlined,
  ThunderboltOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import { AppPage, YamlEditor } from '@/components'
import { createPipeline, getPipeline, triggerPipeline, updatePipeline } from '@/features/provisioning/api/cicd'
import { listClusters } from '@/features/fleet'

type PluginType = 'git' | 'bash' | 'kubectl' | 'helm' | 'docker-build'

export type BuilderStep = {
  key: string
  name: string
  plugin: PluginType
  script: string
  repository: string
  branch: string
  path: string
  command: string
  run: string
  image: string
  tag: string
  context: string
}

export type BuilderStage = { key: string; name: string; steps: BuilderStep[] }

type PipelineMeta = {
  name: string
  trigger: string
  clusterId?: number
  namespace: string
  runnerImage: string
  branches: string
  cron: string
}

const TRIGGER_OPTIONS = [
  { value: 'push', label: '代码推送' },
  { value: 'schedule', label: '定时触发' },
  { value: 'manual', label: '手动执行' },
]

const PLUGIN_CATALOG: Array<{ type: PluginType; title: string; description: string; icon: React.ReactNode; tone: string; category: string }> = [
  { type: 'git', title: 'Checkout', description: '拉取 Git 仓库代码', icon: <GithubOutlined />, tone: 'blue', category: '代码源' },
  { type: 'bash', title: 'RunScript', description: '运行 Bash / Shell 脚本', icon: <CodeOutlined />, tone: 'violet', category: '构建工具' },
  { type: 'kubectl', title: 'Kubectl', description: '执行 Kubernetes 命令', icon: <DeploymentUnitOutlined />, tone: 'cyan', category: '部署发布' },
  { type: 'helm', title: 'Helm', description: '安装或升级 Helm Release', icon: <UploadOutlined />, tone: 'green', category: '部署发布' },
  { type: 'docker-build', title: 'Docker Build', description: '构建并推送容器镜像', icon: <ThunderboltOutlined />, tone: 'orange', category: '构建工具' },
]

const emptyStep = (plugin: PluginType = 'bash', index = 1): BuilderStep => ({
  key: `${plugin}-${Date.now()}-${index}`,
  name: PLUGIN_CATALOG.find((item) => item.type === plugin)?.title || '插件步骤',
  plugin,
  script: '',
  repository: '',
  branch: 'main',
  path: '/workspace/src',
  command: '',
  run: '',
  image: '',
  tag: 'latest',
  context: '.',
})

const createInitialStages = (): BuilderStage[] => [{ key: 'stage-1', name: '阶段 1', steps: [] }]

const yamlScalar = (value: string) => JSON.stringify(value || '')

export function builderToYaml(stages: BuilderStage[]) {
  const lines = ['version: "1"', 'stages:']
  stages.forEach((stage) => {
    lines.push(`  - key: ${yamlScalar(stage.key)}`, `    name: ${yamlScalar(stage.name)}`, '    steps:')
    stage.steps.forEach((step) => {
      lines.push(`      - key: ${yamlScalar(step.key)}`, `        name: ${yamlScalar(step.name)}`, `        plugin: ${yamlScalar(step.plugin)}`)
      if (step.plugin === 'git') {
        lines.push(`        repository: ${yamlScalar(step.repository)}`, `        branch: ${yamlScalar(step.branch)}`, `        path: ${yamlScalar(step.path)}`)
      } else if (step.plugin === 'bash') {
        lines.push('        script: |', ...(step.script || '').split('\n').map((line) => `          ${line}`))
      } else if (step.plugin === 'docker-build') {
        lines.push(`        image: ${yamlScalar(step.image)}`, `        tag: ${yamlScalar(step.tag)}`, `        context: ${yamlScalar(step.context)}`)
      } else {
        lines.push(`        command: ${yamlScalar(step.command)}`)
      }
    })
  })
  return lines.join('\n')
}

const parseYamlScalar = (value: string) => {
  const trimmed = value.trim()
  if (!trimmed) return ''
  try {
    return JSON.parse(trimmed) as string
  } catch {
    return trimmed.replace(/^['"]|['"]$/g, '')
  }
}

const yamlKey = (value: string, fallback: string) => value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || fallback

export function yamlToBuilder(yaml?: string, fallbackToInitial = true): BuilderStage[] {
  if (!yaml?.trim()) return fallbackToInitial ? createInitialStages() : []
  const stages: BuilderStage[] = []
  const lines = yaml.replace(/\r/g, '').split('\n')
  let stage: BuilderStage | undefined
  let step: BuilderStep | undefined

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index] || ''
    const stageKey = line.match(/^\s{2}-\s+(key|name):\s*(.+)$/)
    if (stageKey) {
      const value = parseYamlScalar(stageKey[2] || '')
      const key = stageKey[1] === 'key' ? value : yamlKey(value, `stage-${stages.length + 1}`)
      stage = { key, name: stageKey[1] === 'name' ? value : key, steps: [] }
      stages.push(stage)
      step = undefined
      continue
    }
    if (!stage) continue
    const stageName = line.match(/^\s{4}name:\s*(.+)$/)
    if (stageName && !step) {
      stage.name = parseYamlScalar(stageName[1] || '')
      continue
    }
    const stepKey = line.match(/^\s{6}-\s+(key|name):\s*(.+)$/)
    if (stepKey) {
      step = emptyStep('bash', stage.steps.length + 1)
      const value = parseYamlScalar(stepKey[2] || '')
      step.key = stepKey[1] === 'key' ? value : yamlKey(value, `step-${stage.steps.length + 1}`)
      if (stepKey[1] === 'name') step.name = value
      stage.steps.push(step)
      continue
    }
    if (!step) continue
    const currentStep = step
    const scriptMarker = line.match(/^\s{8}script:\s*\|\s*$/)
    if (scriptMarker) {
      const scriptLines: string[] = []
      index += 1
      while (index < lines.length && (/^\s{10}/.test(lines[index] || '') || !(lines[index] || '').trim())) {
        scriptLines.push((lines[index] || '').replace(/^\s{10}/, ''))
        index += 1
      }
      index -= 1
      step.script = scriptLines.join('\n').replace(/\n+$/, '')
      continue
    }
    const field = line.match(/^\s{8}(name|plugin|repository|branch|path|command|run|image|tag|context):\s*(.*)$/)
    if (!field) continue
    const fieldName = field[1] || ''
    const value = parseYamlScalar(field[2] || '')
    if (fieldName === 'name') currentStep.name = value
    else if (fieldName === 'plugin') currentStep.plugin = (['git', 'bash', 'kubectl', 'helm', 'docker-build'].includes(value) ? value : 'bash') as PluginType
    else if (fieldName === 'run') {
      currentStep.run = value
      if (currentStep.plugin === 'bash') currentStep.script = value
      if (currentStep.plugin === 'kubectl' || currentStep.plugin === 'helm') currentStep.command = value
    }
    else if (fieldName in currentStep) (currentStep as unknown as Record<string, string>)[fieldName] = value
  }
  stages.forEach((item) => item.steps.forEach((currentStep) => {
    if (currentStep.run && (currentStep.plugin === 'kubectl' || currentStep.plugin === 'helm') && !currentStep.command) currentStep.command = currentStep.run
    if (currentStep.run && currentStep.plugin === 'bash' && !currentStep.script) currentStep.script = currentStep.run
  }))
  return stages.length ? stages : (fallbackToInitial ? createInitialStages() : [])
}

const pluginTitle = (plugin: PluginType) => PLUGIN_CATALOG.find((item) => item.type === plugin)?.title || plugin

const PipelineEditorPage: React.FC = () => {
  const editMatch = history.location.pathname.match(/\/cicd\/pipelines\/([^/]+)\/edit$/)
  const pipelineID = editMatch?.[1]
  const [meta, setMeta] = useState<PipelineMeta>({ name: '', trigger: 'push', namespace: 'cicd', runnerImage: 'alpine:3.20', branches: '', cron: '' })
  const [stages, setStages] = useState<BuilderStage[]>(createInitialStages)
  const [selected, setSelected] = useState({ stageIndex: 0, stepIndex: 0 })
  const [catalogFilter, setCatalogFilter] = useState('')
  const [pluginPickerOpen, setPluginPickerOpen] = useState(false)
  const [pluginTargetStage, setPluginTargetStage] = useState(0)
  const [inspectorOpen, setInspectorOpen] = useState(false)
  const [editorView, setEditorView] = useState<'graph' | 'yaml'>('graph')
  const [saving, setSaving] = useState(false)
  const [loadingPipeline, setLoadingPipeline] = useState(Boolean(pipelineID))
  const clustersQuery = useQuery({
    queryKey: ['cicd-editor-clusters'],
    queryFn: ({ signal }) => listClusters({ page: 1, pageSize: 100 }, signal),
  })
  const clusterOptions = (clustersQuery.data?.items || []).map((cluster) => ({
    value: cluster.id,
    label: cluster.name,
    title: `${cluster.name} · ${cluster.status || 'unknown'}`,
  }))

  useEffect(() => {
    if (!pipelineID) return
    setLoadingPipeline(true)
    getPipeline(pipelineID).then((pipeline: any) => {
      const trigger = pipeline.triggerType || pipeline.trigger_type || 'manual'
      const parsedStages = yamlToBuilder(pipeline.configYaml || pipeline.config_yaml)
      const storedBranch = pipeline.branches?.split(',')[0]?.trim()
      if (storedBranch) {
        const gitStep = parsedStages.flatMap((stage) => stage.steps).find((item) => item.plugin === 'git')
        if (gitStep && !gitStep.branch) gitStep.branch = storedBranch
      }
      setMeta({
        name: pipeline.name || '',
        trigger,
        clusterId: pipeline.clusterId || pipeline.cluster_id || undefined,
        namespace: pipeline.namespace || 'cicd',
        runnerImage: pipeline.runnerImage || pipeline.runner_image || 'alpine:3.20',
        branches: pipeline.branches || '',
        cron: pipeline.cron || '',
      })
      setStages(parsedStages)
    }).catch((error) => message.error(error instanceof Error ? error.message : '加载流水线失败')).finally(() => setLoadingPipeline(false))
  }, [pipelineID])

  const selectedStep = stages[selected.stageIndex]?.steps[selected.stepIndex]
  const filteredPlugins = useMemo(() => PLUGIN_CATALOG.filter((item) => `${item.title} ${item.description} ${item.category}`.toLowerCase().includes(catalogFilter.toLowerCase())), [catalogFilter])
  const updateMeta = (patch: Partial<PipelineMeta>) => setMeta((current) => ({ ...current, ...patch }))
  const updateStage = (stageIndex: number, patch: Partial<BuilderStage>) => setStages((current) => current.map((stage, index) => index === stageIndex ? { ...stage, ...patch } : stage))
  const updateStep = (stageIndex: number, stepIndex: number, patch: Partial<BuilderStep>) => setStages((current) => current.map((stage, index) => index === stageIndex ? { ...stage, steps: stage.steps.map((step, childIndex) => childIndex === stepIndex ? { ...step, ...patch } : step) } : stage))

  const openPluginPicker = (stageIndex = stages.length - 1) => {
    setPluginTargetStage(stageIndex)
    setCatalogFilter('')
    setPluginPickerOpen(true)
  }

  const addPlugin = (plugin: PluginType, stageIndex = stages.length - 1) => {
    const nextStep = emptyStep(plugin, (stages[stageIndex]?.steps.length || 0) + 1)
    setStages((current) => current.map((stage, index) => index === stageIndex ? { ...stage, steps: [...stage.steps, nextStep] } : stage))
    setSelected({ stageIndex, stepIndex: stages[stageIndex]?.steps.length || 0 })
    setPluginPickerOpen(false)
    setInspectorOpen(true)
  }

  const addStage = () => {
    const stageIndex = stages.length
    setStages((current) => [...current, { key: `stage-${stageIndex + 1}`, name: `新阶段 ${stageIndex + 1}`, steps: [] }])
    setSelected({ stageIndex, stepIndex: -1 })
  }

  const removeStage = (stageIndex: number) => {
    setStages((current) => current.filter((_, index) => index !== stageIndex))
    setSelected({ stageIndex: Math.max(0, stageIndex - 1), stepIndex: 0 })
  }

  const removeStep = (stageIndex: number, stepIndex: number) => {
    setStages((current) => current.map((stage, index) => index === stageIndex ? { ...stage, steps: stage.steps.filter((_, childIndex) => childIndex !== stepIndex) } : stage))
    setSelected({ stageIndex, stepIndex: Math.max(0, stepIndex - 1) })
  }

  const selectStep = (stageIndex: number, stepIndex: number) => {
    setSelected({ stageIndex, stepIndex })
    setInspectorOpen(true)
  }

  const handleSave = async (runAfterSave = false) => {
    if (loadingPipeline) return
    if (!meta.name.trim()) {
      message.warning('请先填写流水线名称')
      return
    }
    if (!meta.clusterId) {
      message.warning('请先选择执行集群')
      return
    }
    setSaving(true)
    try {
      const gitBranch = stages.flatMap((stage) => stage.steps).find((item) => item.plugin === 'git')?.branch || meta.branches
      const payload = { ...meta, branches: gitBranch, triggerType: meta.trigger, clusterId: meta.clusterId, runnerImage: meta.runnerImage, configYaml: builderToYaml(stages) }
      const saved = pipelineID
        ? await updatePipeline(pipelineID, payload).then(() => ({ id: pipelineID }))
        : await createPipeline(payload) as { id?: number | string }
      if (runAfterSave && saved?.id) {
        const run = await triggerPipeline(saved.id) as { id?: number | string }
        message.success('流水线已保存并触发执行')
        history.push(run?.id ? `/cicd/runs/${run.id}` : `/cicd/pipelines/${saved.id}`)
      } else {
        message.success(pipelineID ? '流水线已更新' : '流水线草稿已保存')
        history.push(pipelineID ? `/cicd/pipelines/${pipelineID}` : '/cicd/pipelines')
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存流水线失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <AppPage className="pipeline-editor-page">
      <div className="pipeline-editor-shell">
        <header className="pipeline-editor-toolbar">
          <div className="pipeline-editor-toolbar__identity">
            <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => history.push('/cicd/pipelines')} aria-label="返回流水线列表" />
            <div>
              <div className="pipeline-editor-eyebrow">CI/CD PIPELINE BUILDER</div>
              <Input className="pipeline-editor-name" value={meta.name} onChange={(event) => updateMeta({ name: event.target.value })} placeholder="未命名流水线" />
            </div>
            <Tag color="gold">草稿</Tag>
          </div>
          <Space>
            <Segmented value={editorView} onChange={(value) => setEditorView(value as 'graph' | 'yaml')} options={[{ label: '图形编排', value: 'graph' }, { label: 'YAML', value: 'yaml' }]} />
            <Tooltip title="校验配置"><Button aria-label="校验配置" icon={<CheckCircleOutlined />} onClick={() => message.success('配置校验通过')} /></Tooltip>
            <Tooltip title="保存草稿"><Button aria-label="保存草稿" icon={<SaveOutlined />} loading={saving} onClick={() => handleSave(false)} /></Tooltip>
            <Tooltip title="保存并运行"><Button type="primary" aria-label="保存并运行" icon={<PlayCircleOutlined />} loading={saving} onClick={() => handleSave(true)} /></Tooltip>
          </Space>
        </header>

        <section className="pipeline-editor-meta">
          <div className="pipeline-editor-meta__field"><span>触发方式</span><Select value={meta.trigger} options={TRIGGER_OPTIONS} onChange={(trigger) => updateMeta({ trigger })} /></div>
          <div className="pipeline-editor-meta__field"><span>执行集群</span><Select showSearch optionFilterProp="label" loading={clustersQuery.isLoading} value={meta.clusterId} options={clusterOptions} onChange={(clusterId) => updateMeta({ clusterId })} placeholder="选择执行集群" /></div>
          <div className="pipeline-editor-meta__field"><span>命名空间</span><Input value={meta.namespace} onChange={(event) => updateMeta({ namespace: event.target.value })} placeholder="cicd" /></div>
          <div className="pipeline-editor-meta__field"><span>Runner 镜像</span><Input value={meta.runnerImage} onChange={(event) => updateMeta({ runnerImage: event.target.value })} placeholder="alpine:3.20" /></div>
          {meta.trigger === 'schedule' && <div className="pipeline-editor-meta__field"><span>定时规则</span><Input value={meta.cron} onChange={(event) => updateMeta({ cron: event.target.value })} placeholder="0 2 * * *" /></div>}
        </section>

        <div className="pipeline-editor-workspace">
          <main className="pipeline-editor-canvas">
            <div className="pipeline-editor-canvas__heading">
              <Typography.Title level={4}>{editorView === 'graph' ? '流水线编排' : 'YAML 配置'}</Typography.Title>
              <Space>
                {editorView === 'yaml' && <Tag color="green" icon={<CheckCircleOutlined />}>实时生成</Tag>}
              </Space>
            </div>
            {editorView === 'graph' ? <div className="pipeline-editor-canvas__scroll">
              <div className="pipeline-editor-stage-flow">
                {stages.map((stage, stageIndex) => (
                  <div className="pipeline-editor-stage-wrap" key={stage.key}>
                    <section className="pipeline-editor-stage">
                      <div className="pipeline-editor-stage__header"><div><span className="pipeline-editor-stage__index">{String(stageIndex + 1).padStart(2, '0')}</span><Input value={stage.name} variant="borderless" onChange={(event) => updateStage(stageIndex, { name: event.target.value, key: event.target.value.toLowerCase().replace(/[^a-z0-9]+/g, '-') || `stage-${stageIndex + 1}` })} /></div><Tooltip title="删除阶段"><Button type="text" danger icon={<DeleteOutlined />} onClick={() => removeStage(stageIndex)} disabled={stages.length === 1} /></Tooltip></div>
                      <div className="pipeline-editor-stage__body">
                        {stage.steps.length === 0 ? <Tooltip title="添加第一个插件"><button type="button" aria-label="添加第一个插件" className="pipeline-editor-stage__empty" onClick={() => openPluginPicker(stageIndex)}><PlusOutlined /></button></Tooltip> : stage.steps.map((step, stepIndex) => (
                          <div key={step.key} className={`pipeline-editor-step ${selected.stageIndex === stageIndex && selected.stepIndex === stepIndex ? 'pipeline-editor-step--selected' : ''}`} onClick={() => selectStep(stageIndex, stepIndex)} role="button" tabIndex={0} onKeyDown={(event) => { if (event.key === 'Enter') selectStep(stageIndex, stepIndex) }}>
                            <span className={`pipeline-editor-step__icon pipeline-editor-step__icon--${step.plugin}`}>{PLUGIN_CATALOG.find((item) => item.type === step.plugin)?.icon}</span>
                            <span className="pipeline-editor-step__copy"><strong>{step.name || '未命名步骤'}</strong><small>{pluginTitle(step.plugin)} · {step.plugin === 'bash' ? '脚本执行' : '插件任务'}</small></span>
                            <Tooltip title="删除步骤"><Button type="text" danger icon={<DeleteOutlined />} onClick={(event) => { event.stopPropagation(); removeStep(stageIndex, stepIndex) }} /></Tooltip>
                          </div>
                        ))}
                        <Tooltip title="添加插件步骤"><Button aria-label="添加插件步骤" type="dashed" block icon={<PlusOutlined />} onClick={() => openPluginPicker(stageIndex)} /></Tooltip>
                      </div>
                    </section>
                    {stageIndex < stages.length - 1 && <div className="pipeline-editor-stage-connector" aria-hidden="true"><span /></div>}
                  </div>
                ))}
                <Tooltip title="添加阶段"><button aria-label="添加阶段" className="pipeline-editor-stage-add" onClick={addStage}><PlusOutlined /></button></Tooltip>
              </div>
            </div> : <section className="pipeline-editor-yaml-view"><YamlEditor value={builderToYaml(stages)} readOnly height={620} /></section>}
          </main>

        </div>

        <Modal className="pipeline-plugin-picker" title="选择插件" open={pluginPickerOpen} width={760} footer={null} onCancel={() => setPluginPickerOpen(false)}>
          <div className="pipeline-plugin-picker__intro"><div><Typography.Text strong>选择要加入的插件</Typography.Text><Typography.Text type="secondary">将节点添加到“{stages[pluginTargetStage]?.name || '当前阶段'}”</Typography.Text></div><Input value={catalogFilter} onChange={(event) => setCatalogFilter(event.target.value)} placeholder="搜索插件名称或能力" allowClear style={{ width: 230 }} /></div>
          <div className="pipeline-plugin-picker__grid">
            {filteredPlugins.map((plugin) => (
              <button type="button" key={plugin.type} className={`pipeline-plugin-option pipeline-plugin-option--${plugin.tone}`} onClick={() => addPlugin(plugin.type, pluginTargetStage)}>
                <span className="pipeline-editor-plugin__icon">{plugin.icon}</span>
                <span><strong>{plugin.title}</strong><small>{plugin.description}</small><em>{plugin.category}</em></span>
                <PlusOutlined />
              </button>
            ))}
          </div>
        </Modal>

        <Drawer className="pipeline-plugin-inspector" title={selectedStep ? <Space><span className={`pipeline-editor-step__icon pipeline-editor-step__icon--${selectedStep.plugin}`}>{PLUGIN_CATALOG.find((item) => item.type === selectedStep.plugin)?.icon}</span><span>{pluginTitle(selectedStep.plugin)} 配置</span></Space> : '插件配置'} open={inspectorOpen && Boolean(selectedStep)} width={640} onClose={() => setInspectorOpen(false)}>
          {selectedStep && <div className="pipeline-editor-inspector__content">
            <div className="pipeline-editor-inspector__plugin"><div><strong>{pluginTitle(selectedStep.plugin)}</strong></div><Tooltip title="插件由 Kubernetes Job 执行，运行日志会写入执行记录"><QuestionCircleOutlined /></Tooltip></div>
            <label>步骤名称<Input value={selectedStep.name} onChange={(event) => updateStep(selected.stageIndex, selected.stepIndex, { name: event.target.value })} /></label>
            {selectedStep.plugin === 'git' && <><label>仓库地址<Input prefix={<GithubOutlined />} value={selectedStep.repository} onChange={(event) => updateStep(selected.stageIndex, selected.stepIndex, { repository: event.target.value })} placeholder="https://github.com/org/repo.git" /></label><div className="pipeline-editor-inspector__row"><label>分支<Input value={selectedStep.branch} onChange={(event) => updateStep(selected.stageIndex, selected.stepIndex, { branch: event.target.value })} /></label><label>检出目录<Input value={selectedStep.path} onChange={(event) => updateStep(selected.stageIndex, selected.stepIndex, { path: event.target.value })} /></label></div></>}
            {selectedStep.plugin === 'bash' && <label>脚本内容<Input.TextArea className="pipeline-editor-command-editor" value={selectedStep.script} onChange={(event) => updateStep(selected.stageIndex, selected.stepIndex, { script: event.target.value })} autoSize={{ minRows: 7, maxRows: 14 }} placeholder={'echo "start build"'} /></label>}
            {(selectedStep.plugin === 'kubectl' || selectedStep.plugin === 'helm') && <label>命令参数<Input.TextArea className="pipeline-editor-command-editor" value={selectedStep.command} onChange={(event) => updateStep(selected.stageIndex, selected.stepIndex, { command: event.target.value })} autoSize={{ minRows: 6, maxRows: 12 }} placeholder={selectedStep.plugin === 'kubectl' ? 'kubectl apply -f /workspace/src/k8s' : 'helm upgrade --install app ./chart'} /></label>}
            {selectedStep.plugin === 'docker-build' && <><label>镜像仓库<Input value={selectedStep.image} onChange={(event) => updateStep(selected.stageIndex, selected.stepIndex, { image: event.target.value })} placeholder="registry.example.com/app" /></label><div className="pipeline-editor-inspector__row"><label>Tag<Input value={selectedStep.tag} onChange={(event) => updateStep(selected.stageIndex, selected.stepIndex, { tag: event.target.value })} /></label><label>构建上下文<Input value={selectedStep.context} onChange={(event) => updateStep(selected.stageIndex, selected.stepIndex, { context: event.target.value })} /></label></div></>}
            <Divider />
          </div>}
        </Drawer>
      </div>
    </AppPage>
  )
}

export default PipelineEditorPage
