/**
 * CI/CD 总览 - 数据仪表盘
 * 展示流水线、执行、制品、环境等核心指标与可视化趋势
 */
import React, { useEffect, useMemo, useState } from 'react'
import { history } from '@umijs/max'
import { Button, Card, Col, Row, Tag, Tooltip, Typography } from 'antd'
import {
  BranchesOutlined,
  HistoryOutlined,
  DatabaseOutlined,
  GlobalOutlined,
  ArrowRightOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  StopOutlined,
} from '@ant-design/icons'
import { AppPage } from '@/components'
import { DESIGN_COLORS } from '@/theme/designTokens'
import { getCicdSummary, getEnvironments, getRuns, type Run } from '@/features/provisioning/api/cicd'

const { Text } = Typography

// ============================================================
// 类型定义
// ============================================================
interface TrendPoint {
  date: string
  total: number
  success: number
  failed: number
}

interface StatusSlice {
  key: string
  label: string
  count: number
  color: string
}

interface RecentRun {
  id: string
  pipeline: string
  trigger: string
  status: 'success' | 'failed' | 'running' | 'canceled'
  duration: string
  startedAt: string
}

interface EnvStatus {
  name: string
  label: string
  type: 'production' | 'staging' | 'development'
  version: string
  status: 'deployed' | 'failed' | 'idle'
  lastDeploy: string
  deployedBy: string
  deployCount: number
}

// ============================================================
// 数据派生工具函数
// ============================================================

/** 计算执行时长：< 60s 显示 "x.xs"，否则 "x.xm" */
function formatDuration(startedAt?: string, finishedAt?: string): string {
  if (!startedAt || !finishedAt) return '-'
  const start = new Date(startedAt).getTime()
  const end = new Date(finishedAt).getTime()
  if (isNaN(start) || isNaN(end)) return '-'
  const diff = (end - start) / 1000
  if (diff < 0) return '-'
  if (diff < 60) return diff.toFixed(1) + 's'
  return (diff / 60).toFixed(1) + 'm'
}

/** 从执行记录中按日期分组统计，生成最近 14 天的趋势数据 */
function deriveTrendData(runs: Run[]): TrendPoint[] {
  const now = new Date()
  const days: TrendPoint[] = []
  for (let i = 13; i >= 0; i--) {
    const d = new Date(now)
    d.setDate(d.getDate() - i)
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    days.push({ date: month + '/' + day, total: 0, success: 0, failed: 0 })
  }
  const dayMap = new Map<string, TrendPoint>(days.map((d) => [d.date, d]))
  for (const run of runs) {
    if (!run.startedAt) continue
    const d = new Date(run.startedAt)
    if (isNaN(d.getTime())) continue
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    const entry = dayMap.get(month + '/' + day)
    if (!entry) continue
    entry.total++
    if (run.status === 'success') entry.success++
    else if (run.status === 'failed') entry.failed++
  }
  return days
}

/** 从执行记录中按状态统计，生成环形图分片数据 */
function deriveStatusSlices(runs: Run[]): StatusSlice[] {
  const counts = { success: 0, failed: 0, running: 0, canceled: 0 }
  for (const run of runs) {
    if (run.status === 'success') counts.success++
    else if (run.status === 'failed') counts.failed++
    else if (run.status === 'running') counts.running++
    else if (run.status === 'canceled') counts.canceled++
  }
  return [
    { key: 'success', label: '成功', count: counts.success, color: DESIGN_COLORS.success },
    { key: 'failed', label: '失败', count: counts.failed, color: DESIGN_COLORS.danger },
    { key: 'running', label: '执行中', count: counts.running, color: DESIGN_COLORS.primary },
    { key: 'canceled', label: '已取消', count: counts.canceled, color: DESIGN_COLORS.neutral },
  ]
}

const ENV_TYPE_COLOR: Record<string, string> = {
  production: DESIGN_COLORS.danger,
  staging: DESIGN_COLORS.warning,
  development: DESIGN_COLORS.success,
}

const RUN_STATUS_META: Record<string, { color: string; text: string; icon: React.ReactNode }> = {
  success: { color: 'success', text: '成功', icon: <CheckCircleOutlined style={{ color: DESIGN_COLORS.success }} /> },
  failed: { color: 'error', text: '失败', icon: <CloseCircleOutlined style={{ color: DESIGN_COLORS.danger }} /> },
  running: { color: 'processing', text: '执行中', icon: <LoadingOutlined style={{ color: DESIGN_COLORS.primary }} /> },
  canceled: { color: 'default', text: '已取消', icon: <StopOutlined style={{ color: DESIGN_COLORS.neutral }} /> },
}

const ENV_STATUS_META: Record<string, { color: string; text: string }> = {
  deployed: { color: 'success', text: '已部署' },
  failed: { color: 'error', text: '部署失败' },
  idle: { color: 'default', text: '未部署' },
}

// ============================================================
// 执行趋势柱状图（SVG）
// ============================================================
const W = 960
const H = 280
const PAD = { top: 24, right: 24, bottom: 42, left: 48 }
const CW = W - PAD.left - PAD.right
const CH = H - PAD.top - PAD.bottom

const TrendChart: React.FC<{ data: TrendPoint[] }> = ({ data }) => {
  const [hoverIdx, setHoverIdx] = useState<number | null>(null)
  const maxVal = useMemo(
    () => Math.ceil(Math.max(...data.map((d) => d.total), 10) / 10) * 10,
    [data],
  )
  const slot = CW / data.length
  const barW = slot * 0.45
  const ticks = [0, maxVal / 2, maxVal]

  const handleMove = (e: React.MouseEvent<SVGSVGElement>) => {
    const rect = e.currentTarget.getBoundingClientRect()
    const x = ((e.clientX - rect.left) / rect.width) * W
    const ratio = Math.max(0, Math.min(1, (x - PAD.left) / CW))
    setHoverIdx(Math.round(ratio * (data.length - 1)))
  }

  return (
    <div style={{ position: 'relative', width: '100%' }}>
      <svg
        viewBox={`0 0 ${W} ${H}`}
        preserveAspectRatio="none"
        style={{ width: '100%', height: 280, cursor: 'pointer' }}
        onMouseMove={handleMove}
        onMouseLeave={() => setHoverIdx(null)}
      >
        {/* Y 轴刻度线 */}
        {ticks.map((tick) => {
          const y = PAD.top + CH - (tick / maxVal) * CH
          return (
            <g key={tick}>
              <line x1={PAD.left} x2={W - PAD.right} y1={y} y2={y} stroke={DESIGN_COLORS.grid} strokeDasharray="4 4" />
              <text x={PAD.left - 8} y={y + 4} textAnchor="end" fill={DESIGN_COLORS.textMuted} fontSize="11">
                {tick}
              </text>
            </g>
          )
        })}

        {/* 柱子 */}
        {data.map((point, i) => {
          const x = PAD.left + i * slot + (slot - barW) / 2
          const successH = (point.success / maxVal) * CH
          const failedH = (point.failed / maxVal) * CH
          const successY = PAD.top + CH - successH
          const failedY = successY - failedH
          const isHover = hoverIdx === i
          return (
            <g key={point.date}>
              {/* hover 背景条 */}
              {isHover && (
                <rect
                  x={PAD.left + i * slot}
                  y={PAD.top}
                  width={slot}
                  height={CH}
                  fill={DESIGN_COLORS.primarySoft}
                  rx={4}
                />
              )}
              {/* 成功部分 */}
              <rect x={x} y={successY} width={barW} height={successH} fill={DESIGN_COLORS.success} rx={3} />
              {/* 失败部分 */}
              {point.failed > 0 && (
                <rect x={x} y={failedY} width={barW} height={failedH} fill={DESIGN_COLORS.danger} rx={3} />
              )}
              {/* X 轴标签 */}
              <text
                x={PAD.left + i * slot + slot / 2}
                y={H - 14}
                textAnchor="middle"
                fill={DESIGN_COLORS.textMuted}
                fontSize="11"
              >
                {point.date}
              </text>
            </g>
          )
        })}

        {/* 坐标轴 */}
        <line x1={PAD.left} x2={PAD.left} y1={PAD.top} y2={PAD.top + CH} stroke={DESIGN_COLORS.border} />
        <line x1={PAD.left} x2={W - PAD.right} y1={PAD.top + CH} y2={PAD.top + CH} stroke={DESIGN_COLORS.border} />
      </svg>

      {/* 图例 */}
      <div style={{ display: 'flex', gap: 16, fontSize: 12, color: DESIGN_COLORS.textSecondary, marginTop: 4 }}>
        <span>
          <i style={{ display: 'inline-block', width: 10, height: 10, borderRadius: 2, background: DESIGN_COLORS.success, marginRight: 5 }} />
          成功
        </span>
        <span>
          <i style={{ display: 'inline-block', width: 10, height: 10, borderRadius: 2, background: DESIGN_COLORS.danger, marginRight: 5 }} />
          失败
        </span>
      </div>

      {/* hover tooltip */}
      {hoverIdx !== null && data[hoverIdx] && (
        <div
          style={{
            position: 'absolute',
            left: `${((PAD.left + hoverIdx * slot + slot / 2) / W) * 100}%`,
            top: 8,
            transform: hoverIdx > data.length * 0.7 ? 'translateX(-100%)' : 'translateX(-50%)',
            minWidth: 120,
            padding: '8px 10px',
            borderRadius: 6,
            background: 'rgba(255,255,255,0.96)',
            border: `1px solid ${DESIGN_COLORS.border}`,
            boxShadow: '0 4px 14px rgba(15,23,42,0.14)',
            pointerEvents: 'none',
            zIndex: 2,
          }}
        >
          <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 4 }}>{data[hoverIdx].date}</div>
          <div style={{ color: DESIGN_COLORS.success, fontSize: 12 }}>成功：{data[hoverIdx].success}</div>
          <div style={{ color: DESIGN_COLORS.danger, fontSize: 12 }}>失败：{data[hoverIdx].failed}</div>
          <div style={{ color: DESIGN_COLORS.textSecondary, fontSize: 12, marginTop: 2 }}>总计：{data[hoverIdx].total}</div>
        </div>
      )}
    </div>
  )
}

// ============================================================
// 状态分布环形图（SVG）
// ============================================================
const DONUT_R = 70
const DONUT_STROKE = 22
const DONUT_C = 2 * Math.PI * DONUT_R

const StatusDonut: React.FC<{ slices: StatusSlice[]; total: number }> = ({ slices, total }) => {
  let accOffset = 0
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 24, flexWrap: 'wrap' }}>
      <svg viewBox="0 0 200 200" style={{ width: 180, height: 180, flexShrink: 0 }}>
        {/* 背景圆 */}
        <circle cx="100" cy="100" r={DONUT_R} fill="none" stroke={DESIGN_COLORS.grid} strokeWidth={DONUT_STROKE} />
        {/* 各状态弧段 */}
        {slices.map((slice) => {
          const dash = total > 0 ? (slice.count / total) * DONUT_C : 0
          const gap = DONUT_C - dash
          const el = (
            <circle
              key={slice.key}
              cx="100"
              cy="100"
              r={DONUT_R}
              fill="none"
              stroke={slice.color}
              strokeWidth={DONUT_STROKE}
              strokeDasharray={`${dash} ${gap}`}
              strokeDashoffset={-accOffset}
              transform="rotate(-90 100 100)"
              strokeLinecap="butt"
            />
          )
          accOffset += dash
          return el
        })}
        {/* 中心文字 */}
        <text x="100" y="96" textAnchor="middle" fontSize="32" fontWeight="700" fill={DESIGN_COLORS.text}>
          {total}
        </text>
        <text x="100" y="118" textAnchor="middle" fontSize="12" fill={DESIGN_COLORS.textSecondary}>
          总执行
        </text>
      </svg>
      {/* 图例 */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {slices.map((slice) => (
          <div key={slice.key} style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13 }}>
            <i style={{ display: 'inline-block', width: 10, height: 10, borderRadius: '50%', background: slice.color }} />
            <span style={{ color: DESIGN_COLORS.textSecondary, minWidth: 48 }}>{slice.label}</span>
            <strong>{slice.count}</strong>
            <span style={{ color: DESIGN_COLORS.textMuted, fontSize: 12 }}>
              {total > 0 ? ((slice.count / total) * 100).toFixed(1) + '%' : '-'}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

// ============================================================
// 指标卡片
// ============================================================
const MetricCard: React.FC<{
  icon: React.ReactNode
  label: string
  value: React.ReactNode
  detail?: React.ReactNode
  color: string
}> = ({ icon, label, value, detail, color }) => (
  <Card bodyStyle={{ padding: 20 }}>
    <div style={{ display: 'flex', alignItems: 'flex-start', gap: 14 }}>
      <div
        style={{
          width: 44,
          height: 44,
          borderRadius: 10,
          background: `${color}14`,
          color,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 20,
          flexShrink: 0,
        }}
      >
        {icon}
      </div>
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 13, color: DESIGN_COLORS.textSecondary, marginBottom: 4 }}>{label}</div>
        <div style={{ fontSize: 28, fontWeight: 700, lineHeight: 1.1 }}>{value}</div>
        {detail && <div style={{ fontSize: 12, color: DESIGN_COLORS.textMuted, marginTop: 4 }}>{detail}</div>}
      </div>
    </div>
  </Card>
)

// ============================================================
// 主页面
// ============================================================
export default function CicdOverviewPage() {
  const [summary, setSummary] = useState<any>({ pipelines: 0, runs30d: 0, successRate: 0, artifacts: 0 })
  const [allRuns, setAllRuns] = useState<Run[]>([])
  const [recentRuns, setRecentRuns] = useState<RecentRun[]>([])
  const [environmentRows, setEnvironmentRows] = useState<EnvStatus[]>([])
  useEffect(() => {
    getCicdSummary().then(setSummary)
    getRuns({ page: 1, pageSize: 500 }).then((res) => {
      const list = res.list || []
      setAllRuns(list)
      setRecentRuns(list.slice(0, 5).map((r: Run) => ({
        id: String(r.id),
        pipeline: r.pipelineName || r.pipeline_name || '-',
        trigger: r.triggerType || r.trigger_type || 'manual',
        status: r.status as RecentRun['status'],
        duration: formatDuration(r.startedAt, r.finishedAt),
        startedAt: r.startedAt || r.started_at || '-',
      })))
    })
    getEnvironments().then((res) => setEnvironmentRows((res.list || []).map((e: any) => ({ name: e.name, label: e.label, type: e.environmentType || e.environment_type || 'development', version: e.currentVersion || e.current_version || '-', status: e.status === 'deployed' ? 'deployed' : e.status, lastDeploy: e.lastDeployedAt || e.last_deployed_at || '-', deployedBy: e.deployedBy || e.deployed_by || '-', deployCount: e.deployCount || e.deploy_count || 0 }))))
  }, [])
  const trendData = useMemo(() => deriveTrendData(allRuns), [allRuns])
  const statusSlices = useMemo(() => deriveStatusSlices(allRuns), [allRuns])
  const runsTotal = allRuns.length
  const successCount = allRuns.filter((r) => r.status === 'success').length
  const failedCount = allRuns.filter((r) => r.status === 'failed').length
  return (
    <AppPage keepHeaderTitle title="CI/CD 概览">
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 1. 核心指标卡片 */}
        <Row gutter={[16, 16]}>
          <Col xs={12} lg={6}>
            <MetricCard
              icon={<BranchesOutlined />}
              label="流水线总数"
              value={summary.pipelines ?? 0}
              detail="-"
              color={DESIGN_COLORS.primary}
            />
          </Col>
          <Col xs={12} lg={6}>
            <MetricCard
              icon={<HistoryOutlined />}
              label="近 30 天执行"
              value={summary.runs30d ?? 0}
              detail={'日均 ' + ((summary.runs30d ?? 0) / 30).toFixed(1) + ' 次'}
              color={DESIGN_COLORS.dataSecondary}
            />
          </Col>
          <Col xs={12} lg={6}>
            <MetricCard
              icon={<CheckCircleOutlined />}
              label="执行成功率"
              value={(summary.successRate ?? summary.success_rate ?? 0) + '%'}
              detail={'成功 ' + successCount + ' / 失败 ' + failedCount}
              color={DESIGN_COLORS.success}
            />
          </Col>
          <Col xs={12} lg={6}>
            <MetricCard
              icon={<DatabaseOutlined />}
              label="制品总数"
              value={(summary.artifacts ?? 0).toLocaleString()}
              detail="-"
              color={DESIGN_COLORS.warning}
            />
          </Col>
        </Row>

        {/* 2. 执行趋势 + 状态分布 */}
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={16}>
            <Card
              title="执行趋势"
              extra={<Text type="secondary" style={{ fontSize: 13 }}>近 14 天</Text>}
            >
              <TrendChart data={trendData} />
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card title="执行状态分布" extra={<Text type="secondary" style={{ fontSize: 13 }}>近 30 天</Text>}>
              <StatusDonut slices={statusSlices} total={runsTotal} />
            </Card>
          </Col>
        </Row>

        {/* 3. 最近执行记录 + 环境部署状态 */}
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={16}>
            <Card
              title="最近执行"
              extra={
                <Tooltip title="查看全部执行记录"><Button type="text" aria-label="查看全部执行记录" icon={<ArrowRightOutlined />} onClick={() => history.push('/cicd/runs')} /></Tooltip>
              }
            >
              <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                {recentRuns.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: '24px 0', color: DESIGN_COLORS.textMuted, fontSize: 13 }}>暂无执行记录</div>
                ) : recentRuns.map((run, idx) => {
                  const meta = RUN_STATUS_META[run.status] ?? RUN_STATUS_META.failed!
                  return (
                    <div
                      key={run.id}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 12,
                        padding: '10px 0',
                        borderBottom: idx < recentRuns.length - 1 ? '1px solid ' + DESIGN_COLORS.grid : 'none',
                      }}
                    >
                      {meta.icon}
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                          <strong style={{ fontSize: 14 }}>{run.pipeline}</strong>
                          <Tag color={meta.color} style={{ margin: 0 }}>{meta.text}</Tag>
                        </div>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          触发：{run.trigger} · {run.startedAt}
                        </Text>
                      </div>
                      <Text style={{ fontSize: 13, color: DESIGN_COLORS.textSecondary, flexShrink: 0 }}>
                        {run.duration}
                      </Text>
                    </div>
                  )
                })}
              </div>
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card
              title="环境部署状态"
              extra={
                <Tooltip title="管理部署环境"><Button type="text" aria-label="管理部署环境" icon={<ArrowRightOutlined />} onClick={() => history.push('/cicd/environments')} /></Tooltip>
              }
            >
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {environmentRows.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: '24px 0', color: DESIGN_COLORS.textMuted, fontSize: 13 }}>暂无环境记录</div>
                ) : environmentRows.map((env) => {
                  const meta = ENV_STATUS_META[env.status] ?? ENV_STATUS_META.idle!
                  const envColor = ENV_TYPE_COLOR[env.type] ?? DESIGN_COLORS.primary
                  return (
                    <div
                      key={env.name}
                      style={{
                        position: 'relative',
                        padding: '14px 16px',
                        borderRadius: 8,
                        background: '#fff',
                        border: `1px solid ${DESIGN_COLORS.grid}`,
                        borderLeft: `4px solid ${envColor}`,
                      }}
                    >
                      {/* 顶部：环境名称 + 状态 */}
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                          <GlobalOutlined style={{ color: envColor, fontSize: 16 }} />
                          <strong style={{ fontSize: 14 }}>{env.name}</strong>
                          <Tag style={{ margin: 0, fontSize: 11, color: envColor, borderColor: `${envColor}40`, background: `${envColor}0d` }}>{env.label}</Tag>
                        </div>
                        <Tag color={meta.color} style={{ margin: 0 }}>{meta.text}</Tag>
                      </div>
                      {/* 版本号 */}
                      <div style={{ marginBottom: 8 }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>当前版本</Text>
                        <div style={{ fontWeight: 600, fontSize: 15 }}>{env.version}</div>
                      </div>
                      {/* 底部：部署信息 */}
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 12, color: DESIGN_COLORS.textSecondary, borderTop: `1px solid ${DESIGN_COLORS.grid}`, paddingTop: 8 }}>
                        <span>最近部署 {env.lastDeploy} · {env.deployedBy}</span>
                        <span>近30天 {env.deployCount} 次</span>
                      </div>
                    </div>
                  )
                })}
              </div>
            </Card>
          </Col>
        </Row>
      </div>
    </AppPage>
  )
}
