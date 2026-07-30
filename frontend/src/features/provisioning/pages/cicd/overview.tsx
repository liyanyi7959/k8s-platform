/**
 * CI/CD 总览 - 数据仪表盘
 * 展示流水线、执行、制品、环境等核心指标与可视化趋势
 */
import React, { useMemo, useState } from 'react'
import { history } from '@umijs/max'
import { Button, Card, Col, Row, Tag, Typography } from 'antd'
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
// 静态数据（后端 API 就绪后替换为 React Query）
// ============================================================
const SUMMARY = {
  pipelines: 12,
  runs30d: 348,
  successRate: 94.8,
  artifacts: 1256,
}

const TREND_DATA: TrendPoint[] = [
  { date: '07/13', total: 18, success: 17, failed: 1 },
  { date: '07/14', total: 22, success: 20, failed: 2 },
  { date: '07/15', total: 15, success: 15, failed: 0 },
  { date: '07/16', total: 28, success: 26, failed: 2 },
  { date: '07/17', total: 19, success: 18, failed: 1 },
  { date: '07/18', total: 12, success: 12, failed: 0 },
  { date: '07/19', total: 8, success: 7, failed: 1 },
  { date: '07/20', total: 25, success: 24, failed: 1 },
  { date: '07/21', total: 30, success: 28, failed: 2 },
  { date: '07/22', total: 27, success: 26, failed: 1 },
  { date: '07/23', total: 21, success: 20, failed: 1 },
  { date: '07/24', total: 24, success: 23, failed: 1 },
  { date: '07/25', total: 18, success: 17, failed: 1 },
  { date: '07/26', total: 11, success: 10, failed: 1 },
]

const STATUS_SLICES: StatusSlice[] = [
  { key: 'success', label: '成功', count: 320, color: DESIGN_COLORS.success },
  { key: 'failed', label: '失败', count: 18, color: DESIGN_COLORS.danger },
  { key: 'running', label: '执行中', count: 6, color: DESIGN_COLORS.primary },
  { key: 'canceled', label: '已取消', count: 4, color: DESIGN_COLORS.neutral },
]

const RECENT_RUNS: RecentRun[] = [
  { id: '1', pipeline: 'frontend-ci', trigger: 'admin', status: 'success', duration: '3m 24s', startedAt: '07-26 14:32' },
  { id: '2', pipeline: 'backend-deploy', trigger: 'schedule', status: 'failed', duration: '8m 12s', startedAt: '07-26 12:00' },
  { id: '3', pipeline: 'api-gateway-build', trigger: 'admin', status: 'success', duration: '5m 43s', startedAt: '07-26 10:15' },
  { id: '4', pipeline: 'helm-chart-release', trigger: 'admin', status: 'running', duration: '2m 18s', startedAt: '07-26 09:30' },
  { id: '5', pipeline: 'frontend-ci', trigger: 'push', status: 'success', duration: '3m 15s', startedAt: '07-25 18:22' },
]

const ENVIRONMENTS: EnvStatus[] = [
  { name: 'production', label: '生产', type: 'production', version: 'v1.2.3', status: 'deployed', lastDeploy: '07-26 14:32', deployedBy: 'admin', deployCount: 18 },
  { name: 'staging', label: '预发', type: 'staging', version: 'v1.2.4-rc', status: 'deployed', lastDeploy: '07-26 10:15', deployedBy: 'admin', deployCount: 32 },
  { name: 'development', label: '开发', type: 'development', version: 'v1.2.4-dev', status: 'failed', lastDeploy: '07-26 09:00', deployedBy: 'ci-bot', deployCount: 86 },
]

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
          const dash = (slice.count / total) * DONUT_C
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
              {((slice.count / total) * 100).toFixed(1)}%
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
  return (
    <AppPage keepHeaderTitle title="CI/CD 概览">
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 1. 核心指标卡片 */}
        <Row gutter={[16, 16]}>
          <Col xs={12} lg={6}>
            <MetricCard
              icon={<BranchesOutlined />}
              label="流水线总数"
              value={SUMMARY.pipelines}
              detail="活跃 10 / 暂停 2"
              color={DESIGN_COLORS.primary}
            />
          </Col>
          <Col xs={12} lg={6}>
            <MetricCard
              icon={<HistoryOutlined />}
              label="近 30 天执行"
              value={SUMMARY.runs30d}
              detail="日均 11.6 次"
              color={DESIGN_COLORS.dataSecondary}
            />
          </Col>
          <Col xs={12} lg={6}>
            <MetricCard
              icon={<CheckCircleOutlined />}
              label="执行成功率"
              value={`${SUMMARY.successRate}%`}
              detail="成功 330 / 失败 18"
              color={DESIGN_COLORS.success}
            />
          </Col>
          <Col xs={12} lg={6}>
            <MetricCard
              icon={<DatabaseOutlined />}
              label="制品总数"
              value={SUMMARY.artifacts.toLocaleString()}
              detail="镜像 980 / Chart 276"
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
              <TrendChart data={TREND_DATA} />
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card title="执行状态分布" extra={<Text type="secondary" style={{ fontSize: 13 }}>近 30 天</Text>}>
              <StatusDonut slices={STATUS_SLICES} total={SUMMARY.runs30d} />
            </Card>
          </Col>
        </Row>

        {/* 3. 最近执行记录 + 环境部署状态 */}
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={16}>
            <Card
              title="最近执行"
              extra={
                <Button type="link" onClick={() => history.push('/cicd/runs')}>
                  查看全部 <ArrowRightOutlined />
                </Button>
              }
            >
              <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                {RECENT_RUNS.map((run, idx) => {
                  const meta = RUN_STATUS_META[run.status] ?? RUN_STATUS_META.failed!
                  return (
                    <div
                      key={run.id}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 12,
                        padding: '10px 0',
                        borderBottom: idx < RECENT_RUNS.length - 1 ? `1px solid ${DESIGN_COLORS.grid}` : 'none',
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
                <Button type="link" onClick={() => history.push('/cicd/environments')}>
                  管理 <ArrowRightOutlined />
                </Button>
              }
            >
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                {ENVIRONMENTS.map((env) => {
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
