/**
 * ClusterManageView.utils.ts 纯函数单元测试
 * 覆盖核心工具函数的边界条件和正常路径
 */
import { describe, it, expect } from 'vitest'
import {
  filterTreeByPerms,
  filterTreeByResourceSupport,
  computeNextNamespaceSelection,
  normalizeNamespaceSelection,
  getNamespaceFilter,
  formatAgeMs,
  getWorkloadReadyText,
  getWorkloadDesired,
  getWorkloadAvailable,
  getWorkloadCurrentReplicas,
  isWorkloadProgressing,
  getWorkloadProgressText,
  getListRowSearchText,
  getRowNamespace,
  getPodReadyTextUtil,
  getPodRestartsUtil,
  getJobStatusTextLocal,
  formatJobCompletionsLocal,
  normalizeMultilineText,
  tryPrettyJson,
  decodeBase64Utf8,
  formatTs,
  asIntText,
  ownersListToText,
  getHttpStatus,
  normalizeLabelRecord,
  matchLabels,
  ingressUsesService,
  collectIngressServiceNames,
  podUsesConfigMap,
  podUsesSecret,
  ingressUsesSecret,
  formatPorts,
  formatSelector,
  getHosts,
  formatRules,
  podUsesPvc,
  formatPvcClaimRefText,
  uniq,
  collectTemplateConfigMapsSecrets,
  fmtListText,
  fmtPortsText,
  getResVal,
  buildContainerVms,
  jobPodTemplateSpec,
  cronJobPodTemplateSpec,
  isPodOwnedByJob,
  decoratePodRowUtil,
  buildStorageKey,
  nsColorIndex,
  getRestartsClass,
  getReadyNumClass,
  isNamespacedResource,
  getByPathValue,
  sortItemsByPath,
  getCreationAgeMs,
  getCreationAgeText,
  getResourceBadgeClass,
  getPodRowKey,
  getNamespacedRowKey,
} from '../ClusterManageView.utils'
import type { TreeNode, ResourceKey } from '../ClusterManageView.types'

// ─── filterTreeByPerms ───────────────────────────────────────────────────
describe('filterTreeByPerms', () => {
  const makeTree = (): TreeNode[] => [
    {
      id: 'g1',
      label: 'Group1',
      kind: 'folder',
      children: [
        { id: 'v1', label: 'View1', kind: 'view', resource: 'pods', perm: 'k8s:read' },
        { id: 'v2', label: 'View2', kind: 'view', resource: 'services', perm: 'k8s:write' },
      ],
    },
    {
      id: 'g2',
      label: 'Group2',
      kind: 'folder',
      children: [
        { id: 'v3', label: 'View3', kind: 'view', resource: 'nodes', perm: 'k8s:admin' },
      ],
    },
  ]

  it('有全部权限时返回完整树', () => {
    const result = filterTreeByPerms(makeTree(), ['k8s:read', 'k8s:write', 'k8s:admin'])
    expect(result).toHaveLength(2)
    expect(result[0].children).toHaveLength(2)
    expect(result[1].children).toHaveLength(1)
  })

  it('部分权限时过滤无权限节点', () => {
    const result = filterTreeByPerms(makeTree(), ['k8s:read'])
    expect(result).toHaveLength(1)
    expect(result[0].children).toHaveLength(1)
    expect(result[0].children![0].id).toBe('v1')
  })

  it('无权限时返回空数组', () => {
    const result = filterTreeByPerms(makeTree(), [])
    expect(result).toHaveLength(0)
  })

  it('perm 为 undefined 时默认有权限', () => {
    const tree: TreeNode[] = [
      { id: 'v1', label: 'View1', kind: 'view', resource: 'pods' },
    ]
    const result = filterTreeByPerms(tree, [])
    expect(result).toHaveLength(1)
  })

  it('perm 为数组时支持 OR 逻辑', () => {
    const tree: TreeNode[] = [
      { id: 'v1', label: 'View1', kind: 'view', resource: 'pods', perm: ['k8s:read', 'k8s:write'] },
    ]
    const result = filterTreeByPerms(tree, ['k8s:write'])
    expect(result).toHaveLength(1)
  })
})

// ─── filterTreeByResourceSupport ─────────────────────────────────────────
describe('filterTreeByResourceSupport', () => {
  const makeTree = (): TreeNode[] => [
    {
      id: 'g1',
      label: 'Group1',
      kind: 'folder',
      children: [
        { id: 'v1', label: 'Pods', kind: 'view', resource: 'pods' },
        { id: 'v2', label: 'Services', kind: 'view', resource: 'services' },
        { id: 'v3', label: 'PodMetrics', kind: 'view', resource: 'podmetrics' },
      ],
    },
  ]

  it('全部支持时返回完整树', () => {
    const result = filterTreeByResourceSupport(makeTree(), { pods: true, services: true, podmetrics: true })
    expect(result[0].children).toHaveLength(3)
  })

  it('不支持的资源被过滤', () => {
    const result = filterTreeByResourceSupport(makeTree(), { pods: true, services: false, podmetrics: true })
    expect(result[0].children).toHaveLength(2)
  })

  it('podmetrics 即使不支持也保留', () => {
    const result = filterTreeByResourceSupport(makeTree(), { pods: true, services: true, podmetrics: false })
    expect(result[0].children).toHaveLength(3)
  })

  it('子节点全部被过滤时父节点也被过滤', () => {
    // podmetrics 有特殊保留逻辑（始终保留），所以需排除它才能测试"全部过滤"
    const tree: TreeNode[] = [
      {
        kind: 'folder',
        label: '工作负载',
        icon: 'box',
        children: [
          { kind: 'view', label: 'Pods', resource: 'pods' },
          { kind: 'view', label: 'Services', resource: 'services' },
        ],
      },
    ]
    const result = filterTreeByResourceSupport(tree, { pods: false, services: false })
    expect(result).toHaveLength(0)
  })
})

// ─── computeNextNamespaceSelection ──────────────────────────────────────
describe('computeNextNamespaceSelection', () => {
  const ns = ['default', 'kube-system', 'monitoring']
  const ALL = '__ALL__'

  it('选择全部时返回 [ALL]', () => {
    const result = computeNextNamespaceSelection([], [ALL], ns, ALL)
    expect(result).toEqual([ALL])
  })

  it('从全部取消选择后选择单个命名空间', () => {
    const result = computeNextNamespaceSelection([ALL], ['default'], ns, ALL)
    expect(result).toEqual(['default'])
  })

  it('选择所有命名空间时自动转为 [ALL]', () => {
    const result = computeNextNamespaceSelection([], ['default', 'kube-system', 'monitoring'], ns, ALL)
    expect(result).toEqual([ALL])
  })

  it('空选择时默认为 [ALL]', () => {
    const result = computeNextNamespaceSelection([], [], ns, ALL)
    expect(result).toEqual([ALL])
  })

  it('过滤不在允许列表中的命名空间', () => {
    const result = computeNextNamespaceSelection([], ['default', 'invalid-ns'], ns, ALL)
    expect(result).toEqual(['default'])
  })
})

// ─── normalizeNamespaceSelection ────────────────────────────────────────
describe('normalizeNamespaceSelection', () => {
  const ALL = '__ALL__'

  it('非数组输入返回 [ALL]', () => {
    expect(normalizeNamespaceSelection(null, ALL)).toEqual([ALL])
    expect(normalizeNamespaceSelection(undefined, ALL)).toEqual([ALL])
  })

  it('包含 ALL 时返回 [ALL]', () => {
    expect(normalizeNamespaceSelection([ALL, 'default'], ALL)).toEqual([ALL])
  })

  it('空数组返回 [ALL]', () => {
    expect(normalizeNamespaceSelection([], ALL)).toEqual([ALL])
  })

  it('去重并过滤空值', () => {
    expect(normalizeNamespaceSelection(['default', '', 'default', 'kube-system'], ALL)).toEqual(['default', 'kube-system'])
  })
})

// ─── getNamespaceFilter ─────────────────────────────────────────────────
describe('getNamespaceFilter', () => {
  const ALL = '__ALL__'

  it('选择 ALL 时返回 null', () => {
    expect(getNamespaceFilter([ALL], ALL)).toBeNull()
  })

  it('选择具体命名空间时返回数组', () => {
    expect(getNamespaceFilter(['default', 'kube-system'], ALL)).toEqual(['default', 'kube-system'])
  })
})

// ─── formatAgeMs ────────────────────────────────────────────────────────
describe('formatAgeMs', () => {
  it('负数返回 -', () => {
    expect(formatAgeMs(-1000)).toBe('-')
  })

  it('NaN 返回 -', () => {
    expect(formatAgeMs(NaN)).toBe('-')
  })

  it('秒级格式化', () => {
    expect(formatAgeMs(5000)).toBe('5s')
  })

  it('分钟级格式化', () => {
    expect(formatAgeMs(120000)).toBe('2m')
  })

  it('小时级格式化', () => {
    expect(formatAgeMs(3600000)).toBe('1h')
  })

  it('天级格式化', () => {
    expect(formatAgeMs(86400000)).toBe('1d')
  })

  it('0 毫秒返回 0s', () => {
    expect(formatAgeMs(0)).toBe('0s')
  })
})

// ─── getWorkloadReadyText ───────────────────────────────────────────────
describe('getWorkloadReadyText', () => {
  it('正常返回 ready/desired', () => {
    expect(getWorkloadReadyText({ spec: { replicas: 3 }, status: { readyReplicas: 2 } })).toBe('2/3')
  })

  it('无 replicas 时返回 0/0', () => {
    expect(getWorkloadReadyText({})).toBe('0/0')
  })
})

// ─── getWorkloadDesired / getWorkloadAvailable ──────────────────────────
describe('getWorkloadDesired', () => {
  it('返回 spec.replicas', () => {
    expect(getWorkloadDesired({ spec: { replicas: 5 } })).toBe(5)
  })

  it('无 replicas 时返回 0', () => {
    expect(getWorkloadDesired({})).toBe(0)
  })
})

describe('getWorkloadAvailable', () => {
  it('优先返回 availableReplicas', () => {
    expect(getWorkloadAvailable({ status: { availableReplicas: 3, readyReplicas: 2 } })).toBe(3)
  })

  it('fallback 到 readyReplicas', () => {
    expect(getWorkloadAvailable({ status: { readyReplicas: 2 } })).toBe(2)
  })

  it('无状态时返回 0', () => {
    expect(getWorkloadAvailable({})).toBe(0)
  })
})

// ─── getWorkloadCurrentReplicas ─────────────────────────────────────────
describe('getWorkloadCurrentReplicas', () => {
  it('DaemonSet 返回 currentNumberScheduled', () => {
    expect(getWorkloadCurrentReplicas({ kind: 'DaemonSet', status: { currentNumberScheduled: 3 } })).toBe(3)
  })

  it('Deployment 返回 status.replicas', () => {
    expect(getWorkloadCurrentReplicas({ kind: 'Deployment', status: { replicas: 5 } })).toBe(5)
  })
})

// ─── isWorkloadProgressing / getWorkloadProgressText ────────────────────
describe('isWorkloadProgressing', () => {
  it('null 输入返回 false', () => {
    expect(isWorkloadProgressing(null)).toBe(false)
  })

  it('完全就绪返回 false', () => {
    expect(isWorkloadProgressing({
      kind: 'Deployment',
      spec: { replicas: 3 },
      status: { replicas: 3, readyReplicas: 3, availableReplicas: 3, updatedReplicas: 3, observedGeneration: 1 },
      metadata: { generation: 1 },
    })).toBe(false)
  })

  it('available < desired 时返回 true', () => {
    expect(isWorkloadProgressing({
      kind: 'Deployment',
      spec: { replicas: 3 },
      status: { replicas: 3, readyReplicas: 2, availableReplicas: 2, updatedReplicas: 3, observedGeneration: 1 },
      metadata: { generation: 1 },
    })).toBe(true)
  })
})

describe('getWorkloadProgressText', () => {
  it('非 progressing 返回空字符串', () => {
    expect(getWorkloadProgressText({
      kind: 'Deployment',
      spec: { replicas: 3 },
      status: { replicas: 3, readyReplicas: 3, availableReplicas: 3, updatedReplicas: 3, observedGeneration: 1 },
      metadata: { generation: 1 },
    })).toBe('')
  })

  it('DaemonSet 滚动中返回正确文本', () => {
    const text = getWorkloadProgressText({
      kind: 'DaemonSet',
      spec: {},
      status: { desiredNumberScheduled: 3, updatedNumberScheduled: 2, numberReady: 2, currentNumberScheduled: 3, observedGeneration: 1 },
      metadata: { generation: 1 },
    })
    expect(text).toContain('滚动中')
  })
})

// ─── getListRowSearchText ───────────────────────────────────────────────
describe('getListRowSearchText', () => {
  it('pods 资源返回完整搜索文本', () => {
    const row = {
      metadata: { namespace: 'default', name: 'test-pod' },
      status: { phase: 'Running', podIP: '10.0.0.1', hostIP: '192.168.1.1' },
      spec: { nodeName: 'node-1' },
    }
    const text = getListRowSearchText(row as any, 'pods')
    expect(text).toContain('default')
    expect(text).toContain('test-pod')
    expect(text).toContain('running')
    expect(text).toContain('10.0.0.1')
  })

  it('services 资源返回正确搜索文本', () => {
    const row = {
      metadata: { namespace: 'default', name: 'my-svc' },
      spec: { type: 'ClusterIP', clusterIP: '10.96.0.1' },
    }
    const text = getListRowSearchText(row as any, 'services')
    expect(text).toContain('clusterip')
    expect(text).toContain('10.96.0.1')
  })

  it('未知资源类型返回默认文本', () => {
    const row = { metadata: { namespace: 'default', name: 'test' }, kind: 'Unknown' }
    const text = getListRowSearchText(row as any, undefined)
    expect(text).toContain('unknown')
    expect(text).toContain('default')
    expect(text).toContain('test')
  })

  it('null row 返回空字符串', () => {
    expect(getListRowSearchText(null as any, 'pods')).toBe('')
  })
})

// ─── getRowNamespace ────────────────────────────────────────────────────
describe('getRowNamespace', () => {
  it('有 namespace 时返回', () => {
    expect(getRowNamespace({ metadata: { namespace: 'default' } } as any)).toBe('default')
  })

  it('无 namespace 时返回 null', () => {
    expect(getRowNamespace({ metadata: {} } as any)).toBeNull()
  })

  it('空字符串返回 null', () => {
    expect(getRowNamespace({ metadata: { namespace: '' } } as any)).toBeNull()
  })
})

// ─── getPodReadyTextUtil ────────────────────────────────────────────────
describe('getPodReadyTextUtil', () => {
  it('正常返回 ready/total', () => {
    const row = { status: { containerStatuses: [{ ready: true }, { ready: false }] } }
    expect(getPodReadyTextUtil(row)).toBe('1/2')
  })

  it('无 containerStatuses 返回 -', () => {
    expect(getPodReadyTextUtil({})).toBe('-')
  })

  it('使用 row.ready 属性', () => {
    expect(getPodReadyTextUtil({ ready: '3/3' })).toBe('3/3')
  })
})

// ─── getPodRestartsUtil ────────────────────────────────────────────────
describe('getPodRestartsUtil', () => {
  it('累加 restartCount', () => {
    const row = { status: { containerStatuses: [{ restartCount: 3 }, { restartCount: 5 }] } }
    expect(getPodRestartsUtil(row)).toBe(8)
  })

  it('无 containerStatuses 返回 0', () => {
    expect(getPodRestartsUtil({})).toBe(0)
  })
})

// ─── getJobStatusTextLocal ──────────────────────────────────────────────
describe('getJobStatusTextLocal', () => {
  it('succeeded > 0 返回 Succeeded', () => {
    expect(getJobStatusTextLocal({ status: { succeeded: 1 } })).toBe('Succeeded')
  })

  it('failed > 0 返回 Failed', () => {
    expect(getJobStatusTextLocal({ status: { failed: 1 } })).toBe('Failed')
  })

  it('active > 0 返回 Running', () => {
    expect(getJobStatusTextLocal({ status: { active: 1 } })).toBe('Running')
  })

  it('默认返回 Pending', () => {
    expect(getJobStatusTextLocal({})).toBe('Pending')
  })
})

// ─── formatJobCompletionsLocal ──────────────────────────────────────────
describe('formatJobCompletionsLocal', () => {
  it('有 completions 时返回 succeeded/desired', () => {
    expect(formatJobCompletionsLocal({ spec: { completions: 5 }, status: { succeeded: 3 } })).toBe('3/5')
  })

  it('无 completions 时返回 succeeded', () => {
    expect(formatJobCompletionsLocal({ status: { succeeded: 2 } })).toBe('2')
  })
})

// ─── normalizeMultilineText ─────────────────────────────────────────────
describe('normalizeMultilineText', () => {
  it('空输入返回空字符串', () => {
    expect(normalizeMultilineText('')).toBe('')
  })

  it('CRLF 转 LF', () => {
    expect(normalizeMultilineText('a\r\nb')).toBe('a\nb')
  })

  it('带引号的转义换行', () => {
    expect(normalizeMultilineText('"a\\nb"')).toBe('a\nb')
  })

  it('无真实换行时转换转义字符', () => {
    expect(normalizeMultilineText('a\\nb')).toBe('a\nb')
  })
})

// ─── tryPrettyJson ──────────────────────────────────────────────────────
describe('tryPrettyJson', () => {
  it('有效 JSON 格式化', () => {
    const result = tryPrettyJson('{"a":1}')
    expect(result.ok).toBe(true)
    expect(result.text).toContain('"a": 1')
  })

  it('无效 JSON 返回原始文本', () => {
    const result = tryPrettyJson('not json')
    expect(result.ok).toBe(false)
  })

  it('空字符串返回 ok=false', () => {
    const result = tryPrettyJson('')
    expect(result.ok).toBe(false)
  })
})

// ─── decodeBase64Utf8 ──────────────────────────────────────────────────
describe('decodeBase64Utf8', () => {
  it('解码 base64', () => {
    expect(decodeBase64Utf8(btoa('hello'))).toBe('hello')
  })

  it('空输入返回空字符串', () => {
    expect(decodeBase64Utf8('')).toBe('')
  })
})

// ─── formatTs ──────────────────────────────────────────────────────────
describe('formatTs', () => {
  it('null 返回 -', () => {
    expect(formatTs(null)).toBe('-')
  })

  it('有效时间戳返回格式化字符串', () => {
    const result = formatTs('2024-01-01T00:00:00Z')
    expect(result).not.toBe('-')
  })
})

// ─── asIntText ─────────────────────────────────────────────────────────
describe('asIntText', () => {
  it('null 返回 -', () => {
    expect(asIntText(null)).toBe('-')
  })

  it('数字返回截断字符串', () => {
    expect(asIntText(3.7)).toBe('3')
  })

  it('非数字返回 -', () => {
    expect(asIntText('abc')).toBe('-')
  })
})

// ─── ownersListToText ──────────────────────────────────────────────────
describe('ownersListToText', () => {
  it('正常 owners 返回格式化文本', () => {
    expect(ownersListToText([{ kind: 'Deployment', name: 'nginx' }])).toBe('Deployment/nginx')
  })

  it('多个 owners 用逗号分隔', () => {
    expect(ownersListToText([
      { kind: 'Deployment', name: 'nginx' },
      { kind: 'ReplicaSet', name: 'nginx-abc' },
    ])).toContain(',')
  })

  it('空数组返回 -', () => {
    expect(ownersListToText([])).toBe('-')
  })

  it('null 返回 -', () => {
    expect(ownersListToText(null)).toBe('-')
  })
})

// ─── getHttpStatus ─────────────────────────────────────────────────────
describe('getHttpStatus', () => {
  it('从 response.status 获取', () => {
    expect(getHttpStatus({ response: { status: 404 } })).toBe(404)
  })

  it('无 status 返回 null', () => {
    expect(getHttpStatus({})).toBeNull()
  })
})

// ─── normalizeLabelRecord ──────────────────────────────────────────────
describe('normalizeLabelRecord', () => {
  it('正常对象返回标准化结果', () => {
    expect(normalizeLabelRecord({ app: 'nginx', env: 'prod' })).toEqual({ app: 'nginx', env: 'prod' })
  })

  it('过滤空键和空值', () => {
    expect(normalizeLabelRecord({ '': 'val', key: '' })).toEqual({})
  })

  it('null 返回空对象', () => {
    expect(normalizeLabelRecord(null)).toEqual({})
  })
})

// ─── matchLabels ───────────────────────────────────────────────────────
describe('matchLabels', () => {
  it('匹配时返回 true', () => {
    expect(matchLabels({ app: 'nginx', env: 'prod' }, { app: 'nginx' })).toBe(true)
  })

  it('不匹配时返回 false', () => {
    expect(matchLabels({ app: 'nginx' }, { app: 'redis' })).toBe(false)
  })

  it('空 required 返回 false', () => {
    expect(matchLabels({ app: 'nginx' }, {})).toBe(false)
  })
})

// ─── ingressUsesService ────────────────────────────────────────────────
describe('ingressUsesService', () => {
  it('通过 defaultBackend 匹配', () => {
    const ingress = { spec: { defaultBackend: { service: { name: 'my-svc' } } } }
    expect(ingressUsesService(ingress, 'my-svc')).toBe(true)
  })

  it('通过 rules 匹配', () => {
    const ingress = { spec: { rules: [{ http: { paths: [{ backend: { service: { name: 'my-svc' } } }] } }] } }
    expect(ingressUsesService(ingress, 'my-svc')).toBe(true)
  })

  it('不匹配返回 false', () => {
    expect(ingressUsesService({ spec: {} }, 'my-svc')).toBe(false)
  })

  it('空 serviceName 返回 false', () => {
    expect(ingressUsesService({}, '')).toBe(false)
  })
})

// ─── collectIngressServiceNames ─────────────────────────────────────────
describe('collectIngressServiceNames', () => {
  it('收集所有 service 名称并排序', () => {
    const ingress = {
      spec: {
        rules: [
          { http: { paths: [{ backend: { service: { name: 'svc-b' } } }] } },
          { http: { paths: [{ backend: { service: { name: 'svc-a' } } }] } },
        ],
      },
    }
    expect(collectIngressServiceNames(ingress)).toEqual(['svc-a', 'svc-b'])
  })

  it('去重', () => {
    const ingress = {
      spec: {
        rules: [
          { http: { paths: [{ backend: { service: { name: 'svc-a' } } }] } },
          { http: { paths: [{ backend: { service: { name: 'svc-a' } } }] } },
        ],
      },
    }
    expect(collectIngressServiceNames(ingress)).toEqual(['svc-a'])
  })
})

// ─── podUsesConfigMap ──────────────────────────────────────────────────
describe('podUsesConfigMap', () => {
  it('通过 volume 匹配', () => {
    const pod = { spec: { volumes: [{ configMap: { name: 'my-cm' } }] } }
    expect(podUsesConfigMap(pod, 'my-cm')).toBe(true)
  })

  it('通过 envFrom 匹配', () => {
    const pod = { spec: { containers: [{ envFrom: [{ configMapRef: { name: 'my-cm' } }] }] } }
    expect(podUsesConfigMap(pod, 'my-cm')).toBe(true)
  })

  it('不匹配返回 false', () => {
    expect(podUsesConfigMap({ spec: {} }, 'my-cm')).toBe(false)
  })
})

// ─── podUsesSecret ─────────────────────────────────────────────────────
describe('podUsesSecret', () => {
  it('通过 volume 匹配', () => {
    const pod = { spec: { volumes: [{ secret: { secretName: 'my-secret' } }] } }
    expect(podUsesSecret(pod, 'my-secret')).toBe(true)
  })

  it('通过 imagePullSecrets 匹配', () => {
    const pod = { spec: { imagePullSecrets: [{ name: 'my-secret' }] } }
    expect(podUsesSecret(pod, 'my-secret')).toBe(true)
  })
})

// ─── ingressUsesSecret ─────────────────────────────────────────────────
describe('ingressUsesSecret', () => {
  it('通过 TLS 匹配', () => {
    const ingress = { spec: { tls: [{ secretName: 'tls-secret' }] } }
    expect(ingressUsesSecret(ingress, 'tls-secret')).toBe(true)
  })

  it('不匹配返回 false', () => {
    expect(ingressUsesSecret({ spec: {} }, 'tls-secret')).toBe(false)
  })
})

// ─── formatPorts ───────────────────────────────────────────────────────
describe('formatPorts', () => {
  it('格式化端口列表', () => {
    const result = formatPorts([{ port: 80, targetPort: 8080, protocol: 'TCP' }])
    expect(result).toBe('80->8080/TCP')
  })

  it('带名称的端口', () => {
    const result = formatPorts([{ name: 'http', port: 80, protocol: 'TCP' }])
    expect(result).toBe('http:80/TCP')
  })
})

// ─── formatSelector ────────────────────────────────────────────────────
describe('formatSelector', () => {
  it('正常格式化', () => {
    expect(formatSelector({ app: 'nginx', env: 'prod' })).toBe('app=nginx, env=prod')
  })

  it('空对象返回 -', () => {
    expect(formatSelector({})).toBe('-')
  })
})

// ─── getHosts ──────────────────────────────────────────────────────────
describe('getHosts', () => {
  it('提取 host 列表', () => {
    expect(getHosts({ spec: { rules: [{ host: 'a.com' }, { host: 'b.com' }] } })).toEqual(['a.com', 'b.com'])
  })

  it('过滤空 host', () => {
    expect(getHosts({ spec: { rules: [{ host: '' }, { host: 'a.com' }] } })).toEqual(['a.com'])
  })
})

// ─── formatRules ───────────────────────────────────────────────────────
describe('formatRules', () => {
  it('格式化规则', () => {
    const row = { spec: { rules: [{ host: 'a.com', http: { paths: [{ path: '/', backend: { service: { name: 'svc', port: { number: 80 } } } }] } }] } }
    expect(formatRules(row)).toContain('a.com')
    expect(formatRules(row)).toContain('svc:80')
  })

  it('无规则返回 -', () => {
    expect(formatRules({ spec: {} })).toBe('-')
  })
})

// ─── podUsesPvc ────────────────────────────────────────────────────────
describe('podUsesPvc', () => {
  it('匹配 PVC', () => {
    const pod = { spec: { volumes: [{ persistentVolumeClaim: { claimName: 'my-pvc' } }] } }
    expect(podUsesPvc(pod, 'my-pvc')).toBe(true)
  })

  it('不匹配返回 false', () => {
    expect(podUsesPvc({ spec: {} }, 'my-pvc')).toBe(false)
  })
})

// ─── formatPvcClaimRefText ─────────────────────────────────────────────
describe('formatPvcClaimRefText', () => {
  it('有 claimRef 返回 ns/name', () => {
    expect(formatPvcClaimRefText({ spec: { claimRef: { namespace: 'default', name: 'pvc-1' } } })).toBe('default/pvc-1')
  })

  it('无 claimRef 返回 -', () => {
    expect(formatPvcClaimRefText({ spec: {} })).toBe('-')
  })
})

// ─── uniq ──────────────────────────────────────────────────────────────
describe('uniq', () => {
  it('去重', () => {
    expect(uniq(['a', 'b', 'a', 'c'])).toEqual(['a', 'b', 'c'])
  })

  it('过滤空值', () => {
    expect(uniq(['a', '', 'b', ' '])).toEqual(['a', 'b'])
  })
})

// ─── collectTemplateConfigMapsSecrets ──────────────────────────────────
describe('collectTemplateConfigMapsSecrets', () => {
  it('收集 volumes 中的 CM 和 Secret', () => {
    const spec = {
      volumes: [
        { configMap: { name: 'cm-1' } },
        { secret: { secretName: 'sec-1' } },
      ],
    }
    const result = collectTemplateConfigMapsSecrets(spec)
    expect(result.configMaps).toContain('cm-1')
    expect(result.secrets).toContain('sec-1')
  })

  it('收集 envFrom 中的 CM 和 Secret', () => {
    const spec = {
      containers: [{ envFrom: [{ configMapRef: { name: 'cm-2' } }, { secretRef: { name: 'sec-2' } }] }],
    }
    const result = collectTemplateConfigMapsSecrets(spec)
    expect(result.configMaps).toContain('cm-2')
    expect(result.secrets).toContain('sec-2')
  })
})

// ─── fmtListText / fmtPortsText / getResVal ────────────────────────────
describe('fmtListText', () => {
  it('连接数组', () => {
    expect(fmtListText(['a', 'b', 'c'])).toBe('a b c')
  })

  it('非数组返回空', () => {
    expect(fmtListText(null)).toBe('')
  })
})

describe('fmtPortsText', () => {
  it('格式化端口', () => {
    expect(fmtPortsText([{ containerPort: 80, protocol: 'TCP' }])).toBe('80/TCP')
  })

  it('带名称', () => {
    expect(fmtPortsText([{ name: 'http', containerPort: 80, protocol: 'TCP' }])).toBe('http=80/TCP')
  })
})

describe('getResVal', () => {
  it('正常返回值', () => {
    expect(getResVal({ cpu: '100m' }, 'cpu')).toBe('100m')
  })

  it('无值返回 -', () => {
    expect(getResVal({}, 'cpu')).toBe('-')
  })
})

// ─── buildContainerVms ─────────────────────────────────────────────────
describe('buildContainerVms', () => {
  it('构建容器视图模型', () => {
    const spec = {
      containers: [{ name: 'app', image: 'nginx', resources: { requests: { cpu: '100m' } } }],
      initContainers: [{ name: 'init', image: 'busybox' }],
    }
    const result = buildContainerVms(spec)
    expect(result.options).toHaveLength(2)
    expect(result.rows).toHaveLength(2)
    expect(result.map.size).toBe(2)
  })

  it('null spec 返回空结果', () => {
    const result = buildContainerVms(null)
    expect(result.options).toHaveLength(0)
  })
})

// ─── jobPodTemplateSpec / cronJobPodTemplateSpec ────────────────────────
describe('jobPodTemplateSpec', () => {
  it('返回 Job 的 pod spec', () => {
    const row = { spec: { template: { spec: { containers: [] } } } }
    expect(jobPodTemplateSpec(row)).toEqual({ containers: [] })
  })

  it('无 template 返回 null', () => {
    expect(jobPodTemplateSpec({})).toBeNull()
  })
})

describe('cronJobPodTemplateSpec', () => {
  it('返回 CronJob 的 pod spec', () => {
    const row = { spec: { jobTemplate: { spec: { template: { spec: { containers: [] } } } } } }
    expect(cronJobPodTemplateSpec(row)).toEqual({ containers: [] })
  })
})

// ─── isPodOwnedByJob ───────────────────────────────────────────────────
describe('isPodOwnedByJob', () => {
  it('通过 labels 匹配', () => {
    const pod = { metadata: { labels: { 'job-name': 'my-job' } } }
    expect(isPodOwnedByJob(pod, 'my-job')).toBe(true)
  })

  it('通过 ownerReferences 匹配', () => {
    const pod = { metadata: { ownerReferences: [{ kind: 'Job', name: 'my-job' }] } }
    expect(isPodOwnedByJob(pod, 'my-job')).toBe(true)
  })

  it('不匹配返回 false', () => {
    expect(isPodOwnedByJob({ metadata: {} }, 'my-job')).toBe(false)
  })
})

// ─── decoratePodRowUtil ────────────────────────────────────────────────
describe('decoratePodRowUtil', () => {
  it('装饰 Pod 行', () => {
    const row = {
      metadata: { namespace: 'default', name: 'pod-1' },
      status: { containerStatuses: [{ ready: true }], phase: 'Running' },
      spec: { nodeName: 'node-1' },
    }
    const result = decoratePodRowUtil(row)
    expect(result.restarts).toBe(0)
    expect(result.__search).toContain('default')
    expect(result.__search).toContain('pod-1')
  })

  it('null 输入返回 null', () => {
    expect(decoratePodRowUtil(null)).toBeNull()
  })
})

// ─── buildStorageKey ──────────────────────────────────────────────────
describe('buildStorageKey', () => {
  it('构建存储键', () => {
    expect(buildStorageKey('ns', 1, 'default')).toBe('ns:1:default')
  })
})

// ─── nsColorIndex ──────────────────────────────────────────────────────
describe('nsColorIndex', () => {
  it('返回 0-11 范围内的索引', () => {
    const idx = nsColorIndex('default')
    expect(idx).toBeGreaterThanOrEqual(0)
    expect(idx).toBeLessThan(12)
  })

  it('空字符串返回 0', () => {
    expect(nsColorIndex('')).toBe(0)
  })

  it('相同命名空间返回相同索引', () => {
    expect(nsColorIndex('default')).toBe(nsColorIndex('default'))
  })
})

// ─── getRestartsClass ──────────────────────────────────────────────────
describe('getRestartsClass', () => {
  it('0 返回 zero', () => {
    expect(getRestartsClass(0)).toContain('zero')
  })

  it('1-5 返回 low', () => {
    expect(getRestartsClass(3)).toContain('low')
  })

  it('>5 返回 high', () => {
    expect(getRestartsClass(10)).toContain('high')
  })
})

// ─── getReadyNumClass ──────────────────────────────────────────────────
describe('getReadyNumClass', () => {
  it('current >= desired 返回 ok', () => {
    expect(getReadyNumClass(3, 3)).toContain('ok')
  })

  it('current > 0 且 < desired 返回 partial', () => {
    expect(getReadyNumClass(1, 3)).toContain('partial')
  })

  it('current = 0 返回 bad', () => {
    expect(getReadyNumClass(0, 3)).toContain('bad')
  })

  it('都为 0 返回默认', () => {
    expect(getReadyNumClass(0, 0)).toBe('k8s-ready-num')
  })
})

// ─── isNamespacedResource ──────────────────────────────────────────────
describe('isNamespacedResource', () => {
  it('namespaced 资源返回 true', () => {
    expect(isNamespacedResource('pods')).toBe(true)
    expect(isNamespacedResource('services')).toBe(true)
  })

  it('集群级资源返回 false', () => {
    expect(isNamespacedResource('nodes')).toBe(false)
    expect(isNamespacedResource('namespaces')).toBe(false)
  })
})

// ─── getByPathValue ────────────────────────────────────────────────────
describe('getByPathValue', () => {
  it('获取嵌套值', () => {
    expect(getByPathValue({ a: { b: { c: 1 } } }, 'a.b.c')).toBe(1)
  })

  it('路径不存在返回 undefined', () => {
    expect(getByPathValue({}, 'a.b.c')).toBeUndefined()
  })
})

// ─── sortItemsByPath ──────────────────────────────────────────────────
describe('sortItemsByPath', () => {
  it('按升序排序', () => {
    const items = [{ name: 'b' }, { name: 'a' }, { name: 'c' }]
    const sorted = sortItemsByPath(items, 'name', 'asc')
    expect(sorted.map((i) => i.name)).toEqual(['a', 'b', 'c'])
  })

  it('按降序排序', () => {
    const items = [{ name: 'b' }, { name: 'a' }, { name: 'c' }]
    const sorted = sortItemsByPath(items, 'name', 'desc')
    expect(sorted.map((i) => i.name)).toEqual(['c', 'b', 'a'])
  })

  it('无 prop 或 dir 返回原数组', () => {
    const items = [{ name: 'b' }, { name: 'a' }]
    expect(sortItemsByPath(items, undefined, undefined)).toEqual(items)
  })
})

// ─── getCreationAgeMs / getCreationAgeText ─────────────────────────────
describe('getCreationAgeMs', () => {
  it('使用 ageMs 属性', () => {
    expect(getCreationAgeMs({ ageMs: 5000 } as any)).toBe(5000)
  })

  it('使用 creationTimestamp', () => {
    const ts = new Date(Date.now() - 60000).toISOString()
    const ms = getCreationAgeMs({ metadata: { creationTimestamp: ts } } as any)
    expect(ms).toBeGreaterThanOrEqual(59000)
    expect(ms).toBeLessThanOrEqual(61000)
  })

  it('无时间信息返回 null', () => {
    expect(getCreationAgeMs({ metadata: {} } as any)).toBeNull()
  })
})

describe('getCreationAgeText', () => {
  it('使用 age 属性', () => {
    expect(getCreationAgeText({ age: '5d' } as any)).toBe('5d')
  })

  it('无 age 时计算', () => {
    const ts = new Date(Date.now() - 3600000).toISOString()
    expect(getCreationAgeText({ metadata: { creationTimestamp: ts } } as any)).toBe('1h')
  })
})

// ─── getResourceBadgeClass ─────────────────────────────────────────────
describe('getResourceBadgeClass', () => {
  it('pods 返回 pods badge', () => {
    expect(getResourceBadgeClass('pods')).toContain('pods')
  })

  it('services 返回 services badge', () => {
    expect(getResourceBadgeClass('services')).toContain('services')
  })

  it('未知资源返回 default', () => {
    expect(getResourceBadgeClass('unknown' as ResourceKey)).toContain('default')
  })
})

// ─── getPodRowKey / getNamespacedRowKey ─────────────────────────────────
describe('getPodRowKey', () => {
  it('返回 ns/name', () => {
    expect(getPodRowKey({ metadata: { namespace: 'default', name: 'pod-1' } } as any)).toBe('default/pod-1')
  })
})

describe('getNamespacedRowKey', () => {
  it('返回 ns/name', () => {
    expect(getNamespacedRowKey({ metadata: { namespace: 'kube-system', name: 'svc-1' } } as any)).toBe('kube-system/svc-1')
  })
})
