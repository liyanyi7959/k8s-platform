<template>
  <div class="ai-page">
    <section class="topbar-card">
      <div class="topbar-copy">
        <p class="eyebrow">AI Ops Desk</p>
        <h1>集群排障工作台</h1>
        <p>
          把会话、证据采集、建议动作和人工确认放到一条连续流程里。AI 只会自动执行只读诊断查询，所有变更提案仍然需要人工确认后才能落地。
        </p>
      </div>

      <div class="topbar-metrics">
        <div class="metric-pill">
          <span>当前集群</span>
          <strong>{{ currentClusterLabel }}</strong>
        </div>
        <div class="metric-pill">
          <span>会话数量</span>
          <strong>{{ conversationResult.total }}</strong>
        </div>
        <div class="metric-pill metric-pill--warn">
          <span>执行策略</span>
          <strong>只读自动采集 / 变更强制确认</strong>
        </div>
      </div>
    </section>

    <section class="control-card">
      <div class="control-row">
        <el-form-item label="目标集群" class="control-item">
          <el-select
            v-model="selectedClusterId"
            placeholder="请选择集群"
            filterable
            class="control-select"
            @change="handleClusterChange"
          >
            <el-option v-for="cluster in clusters" :key="cluster.id" :label="cluster.name" :value="cluster.id" />
          </el-select>
        </el-form-item>

        <el-form-item label="搜索会话" class="control-item control-item--grow">
          <el-input
            v-model="keyword"
            placeholder="按标题、摘要或问题关键字过滤"
            clearable
            @keyup.enter="loadConversations"
            @clear="loadConversations"
          />
        </el-form-item>

        <div class="control-actions">
          <el-button type="primary" @click="openCreateDialog">新建会话</el-button>
          <el-button :loading="loadingConversations" @click="loadConversations">刷新列表</el-button>
        </div>
      </div>
    </section>

    <section class="workspace">
      <aside class="sidebar-card">
        <div class="sidebar-head">
          <div>
            <h2>会话索引</h2>
            <p>从最近处理过的问题继续追问，或者新开一个诊断上下文。</p>
          </div>
          <el-tag type="info" effect="plain">{{ conversationResult.total }} 条</el-tag>
        </div>

        <div class="sidebar-shortcuts">
          <button
            v-for="item in quickPrompts"
            :key="item.title"
            type="button"
            class="shortcut-chip"
            @click="applyQuickPrompt(item.mode, item.prompt)"
          >
            <span>{{ item.title }}</span>
            <small>{{ item.hint }}</small>
          </button>
        </div>

        <el-scrollbar class="conversation-scroll">
          <EmptyState
            v-if="conversationResult.list.length === 0 && !loadingConversations"
            type="no-data"
            title="还没有 AI 会话"
            description="可以直接发送问题自动创建会话，也可以先点上方“新建会话”建立独立的排障上下文。"
          >
            <el-button type="primary" @click="openCreateDialog">新建会话</el-button>
          </EmptyState>

          <div v-else class="conversation-list">
            <button
              v-for="item in conversationResult.list"
              :key="item.id"
              type="button"
              class="conversation-item"
              :class="{ 'conversation-item--active': item.id === activeConversationId }"
              @click="selectConversation(item.id)"
            >
              <div class="conversation-item__head">
                <strong>{{ item.title || `会话 #${item.id}` }}</strong>
                <el-tag size="small" :type="item.assistant_mode === 'diagnose' ? 'warning' : 'success'" effect="plain">
                  {{ assistantModeLabel(item.assistant_mode) }}
                </el-tag>
              </div>
              <p>{{ item.summary || '等待 AI 生成故障摘要或下一步结论。' }}</p>
              <div class="conversation-item__meta">
                <span>{{ item.message_count }} 条消息</span>
                <span>{{ formatDate(item.updated_at) }}</span>
              </div>
            </button>
          </div>
        </el-scrollbar>
      </aside>

      <div class="main-column">
        <section class="detail-card">
          <div class="detail-head">
            <div>
              <p class="detail-kicker">诊断会话</p>
              <h2>{{ activeConversation?.title || '开始一段新的 AI 排障对话' }}</h2>
              <p class="detail-subtitle">
                {{
                  activeConversation
                    ? `${assistantModeLabel(activeConversation.assistant_mode)} · 集群 #${activeConversation.cluster_id} · ${activeConversation.status}`
                    : '直接在下方输入问题即可自动创建会话。AI 会先尝试只读采集证据，再给出结论和建议动作。'
                }}
              </p>
            </div>
            <div class="detail-head__actions">
              <el-button v-if="activeConversation" text @click="reloadActiveConversation">刷新详情</el-button>
              <el-button v-if="activeConversation" type="primary" plain @click="openProposalDialog">创建提案</el-button>
            </div>
          </div>

          <div v-if="loadingDetail" class="loading-card">
            <el-skeleton animated :rows="8" />
          </div>

          <template v-else-if="activeConversation">
            <div class="status-grid">
              <article class="status-card">
                <span>发起人</span>
                <strong>{{ activeConversation.created_by_name || `#${activeConversation.created_by}` }}</strong>
              </article>
              <article class="status-card">
                <span>最近更新</span>
                <strong>{{ formatDate(activeConversation.updated_at) }}</strong>
              </article>
              <article class="status-card">
                <span>只读证据</span>
                <strong>{{ evidenceSummary }}</strong>
              </article>
              <article class="status-card">
                <span>待确认变更</span>
                <strong>{{ pendingProposalCount }} 个</strong>
              </article>
            </div>

            <div v-if="evidenceWarning" class="warning-banner">
              <strong>证据采集存在缺口</strong>
              <span>{{ evidenceWarning }}</span>
            </div>

            <section v-if="activeSuggestedActions.length > 0" class="section-card">
              <div class="section-head">
                <div>
                  <h3>建议动作</h3>
                  <p>来自 AI 最近一次回复的结构化建议，可直接生成待确认提案。</p>
                </div>
                <el-tag type="warning" effect="plain">{{ activeSuggestedActions.length }} 条</el-tag>
              </div>

              <div class="suggestion-grid">
                <article v-for="action in activeSuggestedActions" :key="suggestedActionKey(action)" class="suggestion-card">
                  <div class="suggestion-card__head">
                    <div>
                      <strong>{{ action.title || actionTypeLabel(action.action_type) }}</strong>
                      <p>{{ actionTypeLabel(action.action_type) }}</p>
                    </div>
                    <el-tag size="small" :type="proposalRiskType(action.risk_level || 'medium')" effect="plain">
                      {{ action.risk_level || 'medium' }}
                    </el-tag>
                  </div>

                  <p class="suggestion-card__reason">{{ action.reason || 'AI 基于当前上下文给出的后续建议。' }}</p>

                  <div class="suggestion-facts">
                    <span v-if="action.target_kind">{{ action.target_kind }}</span>
                    <span v-if="action.target_namespace">{{ action.target_namespace }}</span>
                    <span v-if="action.target_name">{{ action.target_name }}</span>
                    <span v-if="typeof action.replicas === 'number'">replicas = {{ action.replicas }}</span>
                    <span v-if="action.action_type === 'apply_manifest' && action.default_namespace">
                      默认命名空间 {{ action.default_namespace }}
                    </span>
                  </div>

                  <div v-if="action.action_type === 'apply_manifest' && action.manifest_yaml" class="manifest-preview">
                    <CodeMirrorViewer
                      :text="action.manifest_yaml"
                      language="yaml"
                      height="180px"
                      :line-numbers="false"
                    />
                  </div>

                  <div class="suggestion-actions">
                    <span class="suggestion-hint">
                      {{ isSuggestedActionMaterialized(action) ? '已生成提案，等待人工确认' : '生成提案不会直接执行变更' }}
                    </span>
                    <el-button
                      type="primary"
                      size="small"
                      :disabled="isSuggestedActionMaterialized(action)"
                      :loading="creatingSuggestedActionKey === suggestedActionKey(action)"
                      @click="createProposalFromSuggestion(action)"
                    >
                      {{ isSuggestedActionMaterialized(action) ? '提案已生成' : '生成提案' }}
                    </el-button>
                  </div>
                </article>
              </div>
            </section>

            <section v-if="activeConversation.action_proposals.length > 0" class="section-card">
              <div class="section-head">
                <div>
                  <h3>待确认与历史提案</h3>
                  <p>AI 只能创建建议，真正执行前仍需要人工确认。</p>
                </div>
                <el-tag type="info" effect="plain">{{ activeConversation.action_proposals.length }} 个</el-tag>
              </div>

              <div class="proposal-grid">
                <article v-for="proposal in activeConversation.action_proposals" :key="proposal.id" class="proposal-card">
                  <div class="proposal-card__head">
                    <div>
                      <strong>{{ proposal.title }}</strong>
                      <p>{{ actionTypeLabel(proposal.action_type) }} · {{ proposal.target_kind || 'Manifest' }}</p>
                    </div>
                    <div class="proposal-tags">
                      <el-tag size="small" :type="proposalRiskType(proposal.risk_level)" effect="plain">
                        {{ proposal.risk_level }}
                      </el-tag>
                      <el-tag size="small" :type="proposalStatusType(proposal.status)" effect="plain">
                        {{ proposalStatusLabel(proposal.status) }}
                      </el-tag>
                    </div>
                  </div>

                  <p class="proposal-card__summary">{{ proposal.summary }}</p>

                  <div class="proposal-facts">
                    <span v-if="proposal.target_namespace">ns: {{ proposal.target_namespace }}</span>
                    <span v-if="proposal.target_name">name: {{ proposal.target_name }}</span>
                    <span>确认级别: {{ proposal.confirm_level === 'double' ? '双确认' : '单确认' }}</span>
                  </div>

                  <div v-if="proposalPreview(proposal)" class="proposal-preview">
                    {{ proposalPreview(proposal) }}
                  </div>

                  <div v-if="proposal.action_type === 'apply_manifest' && proposalManifestText(proposal)" class="manifest-preview">
                    <CodeMirrorViewer
                      :text="proposalManifestText(proposal)"
                      language="yaml"
                      height="180px"
                      :line-numbers="false"
                    />
                  </div>

                  <div v-if="proposal.latest_execution" class="proposal-execution">
                    <span>最近执行: {{ proposalStatusLabel(proposal.latest_execution.status) }}</span>
                    <span>{{ formatDate(proposal.latest_execution.created_at) }}</span>
                  </div>

                  <div class="proposal-actions">
                    <el-button
                      v-if="proposal.status === 'pending_confirm'"
                      type="primary"
                      size="small"
                      :loading="confirmingProposalId === proposal.id"
                      @click="confirmProposal(proposal)"
                    >
                      确认执行
                    </el-button>
                  </div>
                </article>
              </div>
            </section>

            <section v-if="activeConversation.tool_calls.length > 0" class="section-card">
              <div class="section-head">
                <div>
                  <h3>自动取证记录</h3>
                  <p>这里展示 AI 为了回答问题所触发的只读查询和取证结果。</p>
                </div>
                <el-tag type="success" effect="plain">{{ activeConversation.tool_calls.length }} 次</el-tag>
              </div>

              <div class="tool-grid">
                <article v-for="tool in activeConversation.tool_calls" :key="tool.id" class="tool-card">
                  <div class="tool-card__head">
                    <strong>{{ tool.tool_name }}</strong>
                    <el-tag size="small" :type="toolStatusType(tool.status)" effect="plain">{{ tool.status }}</el-tag>
                  </div>
                  <p>{{ tool.result_summary || tool.error_message || '等待工具结果返回。' }}</p>
                  <div class="tool-card__meta">
                    <span>{{ tool.tool_kind }} · {{ tool.risk_level }}</span>
                    <span>{{ formatDate(tool.created_at) }}</span>
                  </div>
                </article>
              </div>
            </section>

            <section class="section-card section-card--timeline">
              <div class="section-head">
                <div>
                  <h3>对话时间线</h3>
                  <p>保留问题、AI 结论、工具取证轨迹以及结构化建议动作。</p>
                </div>
                <el-tag type="info" effect="plain">{{ activeConversation.messages.length }} 条消息</el-tag>
              </div>

              <el-scrollbar class="timeline-scroll">
                <div v-if="activeConversation.messages.length === 0" class="timeline-empty">
                  <EmptyState
                    type="empty"
                    title="当前会话还没有消息"
                    description="在下方输入问题后，AI 会结合集群只读证据给出分析；如果建议变更，会先生成待确认提案。"
                  />
                </div>

                <div v-else class="timeline-list">
                  <article
                    v-for="message in activeConversation.messages"
                    :key="message.id"
                    class="message-card"
                    :class="`message-card--${message.role}`"
                  >
                    <div class="message-card__head">
                      <div class="message-role">
                        <el-tag size="small" :type="roleTagType(message.role)" effect="plain">{{ roleLabel(message.role) }}</el-tag>
                        <span>{{ formatDate(message.created_at) }}</span>
                      </div>
                      <div class="message-usage">
                        <span v-if="message.tool_call_count > 0">{{ message.tool_call_count }} 个工具调用</span>
                        <span v-if="message.token_input || message.token_output">
                          {{ message.token_input || 0 }} / {{ message.token_output || 0 }} tokens
                        </span>
                      </div>
                    </div>

                    <p class="message-content">{{ message.content }}</p>

                    <div v-if="extractSuggestedActions(message.structured).length > 0" class="message-suggestions">
                      <div
                        v-for="action in extractSuggestedActions(message.structured)"
                        :key="suggestedActionKey(action)"
                        class="message-suggestion"
                      >
                        <strong>{{ action.title || actionTypeLabel(action.action_type) }}</strong>
                        <span>{{ action.reason }}</span>
                      </div>
                    </div>
                  </article>
                </div>
              </el-scrollbar>
            </section>
          </template>

          <div v-else class="empty-stage">
            <EmptyState
              type="empty"
              title="还没有选中会话"
              description="可以从左侧进入已有诊断，也可以直接在下方发起一个新问题。为了减少来回切换，发送首条消息时会自动创建会话。"
            />
          </div>
        </section>

        <section class="composer-card">
          <div class="composer-head">
            <div>
              <p class="detail-kicker">提问入口</p>
              <h3>{{ activeConversationId ? '继续追问当前会话' : '发起新的 AI 诊断' }}</h3>
              <p>支持带上下文提问。诊断模式会优先采集只读证据，聊天模式更适合方案讨论和文档问答。</p>
            </div>
            <el-tag :type="composerModeType" effect="dark">{{ composerModeText }}</el-tag>
          </div>

          <div class="composer-grid">
            <el-form-item label="会话模式" class="composer-field composer-field--wide">
              <el-radio-group v-model="draftAssistantMode" :disabled="Boolean(activeConversationId)">
                <el-radio-button label="diagnose">故障诊断</el-radio-button>
                <el-radio-button label="chat">通用聊天</el-radio-button>
              </el-radio-group>
            </el-form-item>

            <el-form-item label="命名空间" class="composer-field">
              <el-input v-model="draftNamespace" placeholder="例如 payment" clearable />
            </el-form-item>

            <el-form-item label="资源类型" class="composer-field">
              <el-select v-model="draftResourceKind" placeholder="可选" clearable filterable>
                <el-option v-for="item in resourceKindOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>

            <el-form-item label="资源名称" class="composer-field">
              <el-input v-model="draftResourceName" placeholder="例如 payment-api-7df8d6" clearable />
            </el-form-item>
          </div>

          <div class="composer-shortcuts">
            <span>快捷提问</span>
            <button
              v-for="item in quickPrompts"
              :key="`${item.title}-composer`"
              type="button"
              class="prompt-chip"
              @click="applyQuickPrompt(item.mode, item.prompt)"
            >
              {{ item.title }}
            </button>
          </div>

          <el-input
            v-model="draftMessage"
            type="textarea"
            :rows="6"
            resize="none"
            placeholder="例如：帮我分析 payment 命名空间最近 30 分钟 Pod 重启频繁的问题，优先查看事件、日志和工作负载状态。"
            @keyup.ctrl.enter="sendMessage"
          />

          <div class="composer-footer">
            <div class="composer-hints">
              <span>{{ activeConversationId ? '发送后将继续当前会话' : '发送后将自动创建一个新会话' }}</span>
              <span>{{ scopeSummary }}</span>
            </div>

            <div class="composer-actions">
              <el-button v-if="activeConversationId" @click="openProposalDialog">手动创建提案</el-button>
              <el-button type="primary" :loading="sendingMessage" @click="sendMessage">
                {{ activeConversationId ? '继续追问' : '发送诊断' }}
              </el-button>
            </div>
          </div>
        </section>
      </div>
    </section>

    <el-dialog v-model="createDialogVisible" title="新建 AI 会话" width="620px">
      <el-form label-position="top">
        <el-form-item label="会话标题">
          <el-input v-model="createForm.title" placeholder="不填写则根据首条问题自动生成" />
        </el-form-item>
        <el-form-item label="工作模式">
          <el-radio-group v-model="createForm.assistant_mode">
            <el-radio-button label="diagnose">故障诊断</el-radio-button>
            <el-radio-button label="chat">通用聊天</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="首条问题">
          <el-input
            v-model="createForm.opening_message"
            type="textarea"
            :rows="5"
            placeholder="例如：帮我分析 payment 命名空间最近 30 分钟的事件、Pod 重启和工作负载健康状态。"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creatingConversation" @click="submitConversation">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="proposalDialogVisible" title="创建变更提案" width="760px">
      <el-form label-position="top">
        <el-form-item label="动作类型">
          <el-radio-group v-model="proposalForm.proposal_type">
            <el-radio-button label="restart_workload">滚动重启</el-radio-button>
            <el-radio-button label="scale_workload">调整副本数</el-radio-button>
            <el-radio-button label="apply_manifest">应用 Manifest</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <div v-if="proposalForm.proposal_type !== 'apply_manifest'" class="proposal-form-grid">
          <el-form-item label="资源类型">
            <el-select v-model="proposalForm.target_kind" filterable>
              <el-option label="Deployment" value="Deployment" />
              <el-option label="StatefulSet" value="StatefulSet" />
              <el-option label="DaemonSet" value="DaemonSet" />
            </el-select>
          </el-form-item>
          <el-form-item label="命名空间">
            <el-input v-model="proposalForm.target_namespace" />
          </el-form-item>
          <el-form-item label="资源名称">
            <el-input v-model="proposalForm.target_name" />
          </el-form-item>
          <el-form-item v-if="proposalForm.proposal_type === 'scale_workload'" label="目标副本数">
            <el-input-number v-model="proposalForm.replicas" :min="0" :max="999" style="width: 100%" />
          </el-form-item>
        </div>

        <template v-else>
          <div class="proposal-form-grid proposal-form-grid--manifest">
            <el-form-item label="提案标题">
              <el-input v-model="proposalForm.title" placeholder="例如：修复 payment 服务探针配置" />
            </el-form-item>
            <el-form-item label="默认命名空间">
              <el-input v-model="proposalForm.default_namespace" placeholder="Manifest 未显式声明时使用" />
            </el-form-item>
          </div>
          <el-form-item label="Manifest YAML">
            <el-input
              v-model="proposalForm.manifest_yaml"
              type="textarea"
              :rows="12"
              placeholder="请输入待应用的 YAML 内容，创建提案后仍需要人工确认才能执行。"
            />
          </el-form-item>
        </template>

        <el-form-item label="提案说明">
          <el-input
            v-model="proposalForm.reason"
            type="textarea"
            :rows="4"
            placeholder="例如：先进行低风险修复并观察服务恢复情况。"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="proposalDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creatingProposal" @click="submitProposal">生成提案</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import {
  confirmAIActionProposal,
  createAIActionProposal,
  createAIConversation,
  getAIConversationDetail,
  getAIConversations,
  sendAIChat,
  type AIActionProposalItem,
  type AIConversationDetail,
  type AIConversationItem,
  type AIMessageItem
} from '@/features/ai/api/ai'
import { listClusters, type ClusterItem } from '@/features/clusters/api/clusters'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import EmptyState from '@/shared/components/EmptyState.vue'
import type { PageResult } from '@/shared/types/api'

interface SuggestedAction {
  message_id?: number
  action_type: 'restart_workload' | 'scale_workload' | 'apply_manifest'
  title?: string
  reason?: string
  target_kind?: string
  target_namespace?: string
  target_name?: string
  replicas?: number
  manifest_yaml?: string
  default_namespace?: string
  risk_level?: string
  requires_confirmation?: boolean
  auto_proposal_eligible?: boolean
  proposal_created?: boolean
  proposal_id?: number
}

const clusters = ref<ClusterItem[]>([])
const selectedClusterId = ref<number>()
const keyword = ref('')
const draftMessage = ref('')
const draftNamespace = ref('')
const draftResourceKind = ref('')
const draftResourceName = ref('')
const draftAssistantMode = ref<'diagnose' | 'chat'>('diagnose')
const loadingConversations = ref(false)
const loadingDetail = ref(false)
const creatingConversation = ref(false)
const sendingMessage = ref(false)
const creatingProposal = ref(false)
const confirmingProposalId = ref<number>()
const creatingSuggestedActionKey = ref('')
const createDialogVisible = ref(false)
const proposalDialogVisible = ref(false)
const activeConversationId = ref<number>()
const activeConversation = ref<AIConversationDetail>()

const conversationResult = ref<PageResult<AIConversationItem>>({
  list: [],
  total: 0,
  page: 1,
  page_size: 20
})

const createForm = reactive({
  title: '',
  assistant_mode: 'diagnose' as 'diagnose' | 'chat',
  opening_message: ''
})

const proposalForm = reactive({
  proposal_type: 'restart_workload' as 'restart_workload' | 'scale_workload' | 'apply_manifest',
  target_kind: 'Deployment',
  target_namespace: '',
  target_name: '',
  replicas: 1,
  title: '',
  manifest_yaml: '',
  default_namespace: '',
  reason: ''
})

const resourceKindOptions = [
  { label: 'Pod', value: 'Pod' },
  { label: 'Node', value: 'Node' },
  { label: 'Deployment', value: 'Deployment' },
  { label: 'Service', value: 'Service' },
  { label: 'Ingress', value: 'Ingress' },
  { label: 'ConfigMap', value: 'ConfigMap' },
  { label: 'Secret', value: 'Secret' },
  { label: 'StatefulSet', value: 'StatefulSet' },
  { label: 'DaemonSet', value: 'DaemonSet' }
]

const quickPrompts = computed(() => {
  const diagnosePrompts = [
    {
      title: '排查 Pod 重启',
      hint: '事件、日志、重启次数',
      mode: 'diagnose' as const,
      prompt: '帮我分析最近 30 分钟 Pod 重启异常，优先查看事件、日志和容器退出原因。'
    },
    {
      title: '分析服务不可用',
      hint: '工作负载、探针、流量入口',
      mode: 'diagnose' as const,
      prompt: '请排查服务不可用问题，优先检查工作负载状态、探针失败和入口配置。'
    },
    {
      title: '看节点健康',
      hint: '调度、资源、异常事件',
      mode: 'diagnose' as const,
      prompt: '帮我看看当前节点健康情况，关注 NotReady、资源压力和调度异常。'
    }
  ]

  const chatPrompts = [
    {
      title: '梳理处理方案',
      hint: '讨论修复路径',
      mode: 'chat' as const,
      prompt: '结合当前上下文，帮我梳理一个分阶段的排障与修复方案。'
    }
  ]

  return draftAssistantMode.value === 'chat' ? [...diagnosePrompts, ...chatPrompts] : diagnosePrompts
})

const currentClusterLabel = computed(() => {
  const current = clusters.value.find((item) => item.id === selectedClusterId.value)
  return current?.name ?? '未选择'
})

const composerModeText = computed(() => {
  const mode = activeConversation.value?.assistant_mode ?? draftAssistantMode.value
  return mode === 'diagnose' ? '只读诊断模式' : '通用聊天模式'
})

const composerModeType = computed(() => {
  const mode = activeConversation.value?.assistant_mode ?? draftAssistantMode.value
  return mode === 'diagnose' ? 'warning' : 'success'
})

const pendingProposalCount = computed(() => {
  return activeConversation.value?.action_proposals.filter((item) => item.status === 'pending_confirm').length ?? 0
})

const evidenceSummary = computed(() => {
  const toolCalls = activeConversation.value?.tool_calls ?? []
  if (toolCalls.length === 0) return '尚未采集'
  const succeeded = toolCalls.filter((item) => item.status === 'succeeded').length
  const failed = toolCalls.filter((item) => item.status === 'failed').length
  return `成功 ${succeeded} / 失败 ${failed}`
})

const evidenceWarning = computed(() => {
  const toolCalls = activeConversation.value?.tool_calls ?? []
  if (toolCalls.length === 0) return ''
  const failedItems = toolCalls.filter((item) => item.status === 'failed')
  if (failedItems.length === 0) return ''
  const networkError = failedItems.find((item) => (item.error_message || '').toLowerCase().includes('network error'))
  if (networkError) {
    return '部分集群取证请求失败，当前结论可能缺少实时证据，请结合集群连通性一起判断。'
  }
  return '部分工具调用失败，建议在确认结论前补充核实关键证据。'
})

const activeSuggestedActions = computed(() => {
  const messages = activeConversation.value?.messages ?? []
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index]
    if (message.role !== 'assistant') continue
    const actions = extractSuggestedActions(message.structured)
    if (actions.length > 0) return actions
  }
  return [] as SuggestedAction[]
})

const scopeSummary = computed(() => {
  const segments = [draftNamespace.value, draftResourceKind.value, draftResourceName.value].filter(Boolean)
  return segments.length > 0 ? `当前上下文: ${segments.join(' / ')}` : '当前未限定命名空间或资源范围'
})

async function loadClusters() {
  const result = await listClusters({ page: 1, page_size: 200 })
  clusters.value = result.list
  if (!selectedClusterId.value && clusters.value.length > 0) {
    selectedClusterId.value = clusters.value[0].id
  }
}

async function loadConversations() {
  loadingConversations.value = true
  try {
    const result = await getAIConversations({
      page: 1,
      page_size: 100,
      cluster_id: selectedClusterId.value,
      keyword: keyword.value || undefined
    })
    conversationResult.value = result
    if (!result.list.some((item) => item.id === activeConversationId.value)) {
      activeConversationId.value = result.list[0]?.id
    }
    if (activeConversationId.value) {
      await selectConversation(activeConversationId.value)
    } else {
      activeConversation.value = undefined
    }
  } finally {
    loadingConversations.value = false
  }
}

async function selectConversation(id: number) {
  activeConversationId.value = id
  loadingDetail.value = true
  try {
    activeConversation.value = await getAIConversationDetail(id)
    draftAssistantMode.value = activeConversation.value.assistant_mode as 'diagnose' | 'chat'
  } finally {
    loadingDetail.value = false
  }
}

async function reloadActiveConversation() {
  if (!activeConversationId.value) return
  await selectConversation(activeConversationId.value)
}

function openCreateDialog() {
  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择目标集群')
    return
  }
  createForm.title = ''
  createForm.assistant_mode = draftAssistantMode.value
  createForm.opening_message = draftMessage.value.trim()
  createDialogVisible.value = true
}

async function submitConversation() {
  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择目标集群')
    return
  }
  creatingConversation.value = true
  try {
    const result = await createAIConversation(selectedClusterId.value, {
      title: createForm.title || undefined,
      assistant_mode: createForm.assistant_mode,
      opening_message: createForm.opening_message || undefined
    })
    createDialogVisible.value = false
    await loadConversations()
    if (result.id) {
      await selectConversation(result.id)
    }
    ElMessage.success('AI 会话已创建')
  } finally {
    creatingConversation.value = false
  }
}

async function sendMessage() {
  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择目标集群')
    return
  }
  const message = draftMessage.value.trim()
  if (!message) {
    ElMessage.warning('请输入问题后再发送')
    return
  }

  sendingMessage.value = true
  try {
    const result = await sendAIChat(selectedClusterId.value, {
      conversation_id: activeConversationId.value,
      message,
      assistant_mode: activeConversationId.value ? undefined : draftAssistantMode.value,
      namespace: draftNamespace.value || undefined,
      resource_kind: draftResourceKind.value || undefined,
      resource_name: draftResourceName.value || undefined
    })
    draftMessage.value = ''
    await loadConversations()
    await selectConversation(result.conversation_id)
    const autoProposalCount = result.action_proposals?.length ?? 0
    if (autoProposalCount > 0) {
      ElMessage.success(`AI 已回复，并自动生成 ${autoProposalCount} 个待确认变更提案`)
      return
    }
    ElMessage.success(result.tool_calls.length > 0 ? `AI 已回复，并自动采集 ${result.tool_calls.length} 条诊断上下文` : 'AI 已回复')
  } finally {
    sendingMessage.value = false
  }
}

function openProposalDialog() {
  if (!activeConversationId.value) {
    ElMessage.warning('请先选择或创建一个会话')
    return
  }
  proposalForm.proposal_type = 'restart_workload'
  proposalForm.target_kind = normalizeProposalKind(draftResourceKind.value) || 'Deployment'
  proposalForm.target_namespace = draftNamespace.value
  proposalForm.target_name = draftResourceName.value
  proposalForm.replicas = 1
  proposalForm.title = ''
  proposalForm.manifest_yaml = ''
  proposalForm.default_namespace = draftNamespace.value
  proposalForm.reason = ''
  proposalDialogVisible.value = true
}

async function submitProposal() {
  if (!selectedClusterId.value || !activeConversationId.value) {
    ElMessage.warning('请先选择集群并打开会话')
    return
  }

  if (proposalForm.proposal_type !== 'apply_manifest') {
    if (!proposalForm.target_namespace.trim() || !proposalForm.target_name.trim()) {
      ElMessage.warning('请填写命名空间和资源名称')
      return
    }
    if (proposalForm.proposal_type === 'scale_workload' && proposalForm.target_kind === 'DaemonSet') {
      ElMessage.warning('DaemonSet 不支持调整副本数')
      return
    }
  }

  if (proposalForm.proposal_type === 'apply_manifest' && !proposalForm.manifest_yaml.trim()) {
    ElMessage.warning('请填写 Manifest YAML')
    return
  }

  creatingProposal.value = true
  try {
    await createAIActionProposal(selectedClusterId.value, {
      conversation_id: activeConversationId.value,
      proposal_type: proposalForm.proposal_type,
      target_resource: {
        kind: proposalForm.proposal_type === 'apply_manifest' ? '' : proposalForm.target_kind,
        namespace: proposalForm.proposal_type === 'apply_manifest' ? '' : proposalForm.target_namespace.trim(),
        name: proposalForm.proposal_type === 'apply_manifest' ? '' : proposalForm.target_name.trim()
      },
      payload:
        proposalForm.proposal_type === 'scale_workload'
          ? { replicas: proposalForm.replicas }
          : proposalForm.proposal_type === 'apply_manifest'
            ? {
                yaml: proposalForm.manifest_yaml.trim(),
                default_namespace: proposalForm.default_namespace.trim() || undefined,
                title: proposalForm.title.trim() || undefined
              }
            : {},
      reason: proposalForm.reason.trim() || undefined
    })
    proposalDialogVisible.value = false
    await reloadActiveConversation()
    ElMessage.success('变更提案已生成，等待人工确认')
  } finally {
    creatingProposal.value = false
  }
}

async function createProposalFromSuggestion(action: SuggestedAction) {
  if (!selectedClusterId.value || !activeConversationId.value) {
    ElMessage.warning('请先选择集群并打开会话')
    return
  }
  const requestPayload =
    action.action_type === 'scale_workload'
      ? { replicas: action.replicas ?? 1 }
      : action.action_type === 'apply_manifest'
        ? {
            yaml: action.manifest_yaml || '',
            default_namespace: action.default_namespace || action.target_namespace || undefined,
            title: action.title || undefined
          }
        : {}

  const targetResource =
    action.action_type === 'apply_manifest'
      ? { kind: '', namespace: '', name: '' }
      : {
          kind: action.target_kind || '',
          namespace: action.target_namespace || '',
          name: action.target_name || ''
        }

  creatingSuggestedActionKey.value = suggestedActionKey(action)
  try {
    await createAIActionProposal(selectedClusterId.value, {
      conversation_id: activeConversationId.value,
      message_id: action.message_id,
      proposal_type: action.action_type,
      target_resource: targetResource,
      payload: requestPayload,
      reason: action.reason || undefined
    })
    await reloadActiveConversation()
    ElMessage.success('建议动作已转成待确认提案')
  } finally {
    creatingSuggestedActionKey.value = ''
  }
}

async function confirmProposal(proposal: AIActionProposalItem) {
  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择目标集群')
    return
  }
  const confirmText = `确认执行提案“${proposal.title}”？\n\n${proposal.summary}`
  await ElMessageBox.confirm(confirmText, '执行确认', {
    confirmButtonText: '确认执行',
    cancelButtonText: '取消',
    type: proposal.risk_level === 'high' ? 'warning' : 'info'
  })

  confirmingProposalId.value = proposal.id
  try {
    const result = await confirmAIActionProposal(selectedClusterId.value, proposal.id, {
      confirmation_text: 'confirmed',
      confirm_risk: true
    })
    await reloadActiveConversation()
    ElMessage.success(result.result_summary || '提案已执行')
  } finally {
    confirmingProposalId.value = undefined
  }
}

async function handleClusterChange() {
  activeConversationId.value = undefined
  activeConversation.value = undefined
  draftAssistantMode.value = 'diagnose'
  await loadConversations()
}

function applyQuickPrompt(mode: 'diagnose' | 'chat', prompt: string) {
  if (!activeConversationId.value) {
    draftAssistantMode.value = mode
  }
  draftMessage.value = prompt
}

function formatDate(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN')
}

function assistantModeLabel(mode: string) {
  return mode === 'chat' ? '通用聊天' : '故障诊断'
}

function roleLabel(role: string) {
  if (role === 'assistant') return 'AI'
  if (role === 'tool') return '工具'
  if (role === 'system') return '系统'
  return '用户'
}

function roleTagType(role: string) {
  if (role === 'assistant') return 'success'
  if (role === 'tool') return 'warning'
  if (role === 'system') return 'info'
  return ''
}

function toolStatusType(status: string) {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  return 'warning'
}

function proposalStatusType(status: string) {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'pending_confirm') return 'warning'
  return 'info'
}

function proposalRiskType(risk: string) {
  if (risk === 'high') return 'danger'
  if (risk === 'medium') return 'warning'
  return 'success'
}

function proposalStatusLabel(status: string) {
  switch (status) {
    case 'pending_confirm':
      return '待确认'
    case 'approved':
      return '已确认'
    case 'executing':
      return '执行中'
    case 'succeeded':
      return '已成功'
    case 'failed':
      return '已失败'
    case 'cancelled':
      return '已取消'
    default:
      return status || '未知'
  }
}

function actionTypeLabel(actionType: string) {
  switch (actionType) {
    case 'restart_workload':
      return '滚动重启'
    case 'scale_workload':
      return '调整副本数'
    case 'apply_manifest':
      return '应用 Manifest'
    default:
      return actionType || '建议动作'
  }
}

function proposalPreview(proposal: AIActionProposalItem) {
  const preview = proposal.change?.preview
  return typeof preview === 'string' ? preview : ''
}

function proposalManifestText(proposal: AIActionProposalItem) {
  const payload = proposal.change?.payload
  if (!payload || typeof payload !== 'object') return ''
  const yaml = (payload as Record<string, unknown>).yaml
  return typeof yaml === 'string' ? yaml : ''
}

function normalizeProposalKind(value: string) {
  if (value === 'Deployment' || value === 'StatefulSet' || value === 'DaemonSet') {
    return value
  }
  return ''
}

function suggestedActionKey(action: SuggestedAction) {
  return [
    action.message_id ?? 'new',
    action.action_type,
    action.target_kind ?? '-',
    action.target_namespace ?? '-',
    action.target_name ?? '-',
    action.replicas ?? '-',
    action.proposal_id ?? '-'
  ].join(':')
}

function isSuggestedActionMaterialized(action: SuggestedAction) {
  if (action.proposal_created || action.proposal_id) return true
  const proposals = activeConversation.value?.action_proposals ?? []
  return proposals.some((proposal) => {
    if (action.message_id && proposal.message_id === action.message_id) {
      if (proposal.action_type !== action.action_type) return false
      if (action.action_type === 'apply_manifest') return true
      return (
        proposal.target_kind === (action.target_kind || '') &&
        proposal.target_namespace === (action.target_namespace || '') &&
        proposal.target_name === (action.target_name || '')
      )
    }
    return false
  })
}

function extractSuggestedActions(structured?: Record<string, unknown>) {
  const payload = structured?.suggested_actions
  if (!Array.isArray(payload)) return [] as SuggestedAction[]
  return payload
    .map((item) => normalizeSuggestedAction(item))
    .filter((item): item is SuggestedAction => Boolean(item))
}

function normalizeSuggestedAction(item: unknown) {
  if (!item || typeof item !== 'object') return null
  const row = item as Record<string, unknown>
  const actionType = typeof row.action_type === 'string' ? row.action_type : ''
  if (actionType !== 'restart_workload' && actionType !== 'scale_workload' && actionType !== 'apply_manifest') {
    return null
  }
  return {
    message_id: asNumber(row.message_id),
    action_type: actionType,
    title: asString(row.title),
    reason: asString(row.reason),
    target_kind: asString(row.target_kind),
    target_namespace: asString(row.target_namespace),
    target_name: asString(row.target_name),
    replicas: asNumber(row.replicas),
    manifest_yaml: asString(row.manifest_yaml),
    default_namespace: asString(row.default_namespace),
    risk_level: asString(row.risk_level),
    requires_confirmation: asBoolean(row.requires_confirmation),
    auto_proposal_eligible: asBoolean(row.auto_proposal_eligible),
    proposal_created: asBoolean(row.proposal_created),
    proposal_id: asNumber(row.proposal_id)
  } satisfies SuggestedAction
}

function asString(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function asNumber(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function asBoolean(value: unknown) {
  return typeof value === 'boolean' ? value : undefined
}

onMounted(async () => {
  await loadClusters()
  await loadConversations()
})
</script>

<style scoped>
.ai-page {
  --ai-bg: linear-gradient(180deg, #fffaf2 0%, #f5f7fb 38%, #eef3f8 100%);
  --ai-card: rgba(255, 255, 255, 0.82);
  --ai-border: rgba(15, 23, 42, 0.08);
  --ai-text: #132238;
  --ai-muted: #5b6b81;
  --ai-accent: #0f766e;
  --ai-accent-soft: rgba(15, 118, 110, 0.12);
  --ai-warm: #b45309;
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-height: 100%;
  padding: 18px;
  background:
    radial-gradient(circle at top left, rgba(255, 206, 138, 0.34), transparent 26%),
    radial-gradient(circle at top right, rgba(125, 211, 252, 0.24), transparent 30%),
    var(--ai-bg);
  font-family: 'Avenir Next', 'PingFang SC', 'Microsoft YaHei', sans-serif;
}

.topbar-card,
.control-card,
.sidebar-card,
.detail-card,
.composer-card {
  border: 1px solid var(--ai-border);
  background: var(--ai-card);
  border-radius: 28px;
  box-shadow: 0 22px 60px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(16px);
}

.topbar-card {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  padding: 28px;
}

.eyebrow,
.detail-kicker {
  margin: 0 0 10px;
  font-size: 12px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--ai-warm);
}

.topbar-copy h1,
.detail-head h2,
.composer-head h3 {
  margin: 0;
  color: var(--ai-text);
  font-weight: 700;
}

.topbar-copy h1 {
  font-size: 32px;
}

.topbar-copy p,
.detail-subtitle,
.sidebar-head p,
.section-head p,
.composer-head p,
.conversation-item p,
.proposal-card__summary,
.tool-card p,
.message-content,
.suggestion-card__reason,
.composer-hints,
.suggestion-hint {
  color: var(--ai-muted);
  line-height: 1.7;
}

.topbar-copy p {
  max-width: 760px;
  margin: 14px 0 0;
}

.topbar-metrics {
  display: grid;
  gap: 12px;
  min-width: 280px;
}

.metric-pill {
  padding: 16px 18px;
  border-radius: 18px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: rgba(255, 255, 255, 0.72);
}

.metric-pill span,
.status-card span,
.message-usage span,
.tool-card__meta span,
.proposal-facts span,
.conversation-item__meta span,
.suggestion-facts span {
  display: inline-flex;
  font-size: 12px;
  color: var(--ai-muted);
}

.metric-pill strong,
.status-card strong {
  display: block;
  margin-top: 6px;
  color: var(--ai-text);
}

.metric-pill--warn {
  background: rgba(255, 244, 229, 0.9);
}

.control-card {
  padding: 18px 22px 6px;
}

.control-row {
  display: flex;
  align-items: flex-end;
  gap: 16px;
}

.control-item {
  margin-bottom: 12px;
}

.control-item--grow {
  flex: 1;
}

.control-select {
  width: 260px;
}

.control-actions {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
}

.workspace {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 18px;
  min-height: 0;
  flex: 1;
}

.sidebar-card,
.detail-card,
.composer-card {
  padding: 22px;
}

.sidebar-card {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.sidebar-head,
.detail-head,
.detail-head__actions,
.section-head,
.proposal-card__head,
.tool-card__head,
.message-card__head,
.suggestion-card__head,
.suggestion-actions,
.composer-head,
.composer-footer,
.proposal-execution,
.conversation-item__head,
.conversation-item__meta {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.sidebar-head h2,
.section-head h3 {
  margin: 0;
  color: var(--ai-text);
}

.sidebar-shortcuts {
  display: grid;
  gap: 10px;
  margin: 18px 0;
}

.shortcut-chip {
  padding: 14px 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(247, 250, 252, 0.98), rgba(255, 255, 255, 0.86));
  text-align: left;
  cursor: pointer;
  transition:
    transform 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}

.shortcut-chip:hover,
.conversation-item:hover,
.conversation-item--active {
  transform: translateY(-1px);
  border-color: rgba(15, 118, 110, 0.28);
  box-shadow: 0 18px 38px rgba(15, 118, 110, 0.12);
}

.shortcut-chip span {
  display: block;
  font-weight: 600;
  color: var(--ai-text);
}

.shortcut-chip small {
  display: block;
  margin-top: 6px;
  color: var(--ai-muted);
}

.conversation-scroll {
  min-height: 0;
  flex: 1;
}

.conversation-list {
  display: grid;
  gap: 12px;
}

.conversation-item {
  width: 100%;
  padding: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.8);
  cursor: pointer;
  text-align: left;
  transition:
    transform 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}

.conversation-item__head strong {
  color: var(--ai-text);
}

.conversation-item p {
  margin: 10px 0 12px;
}

.main-column {
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
  gap: 18px;
  min-height: 0;
}

.detail-card {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.detail-kicker {
  margin-bottom: 8px;
}

.detail-head {
  margin-bottom: 18px;
}

.detail-head__actions {
  flex-wrap: wrap;
}

.status-grid,
.suggestion-grid,
.proposal-grid,
.tool-grid {
  display: grid;
  gap: 12px;
}

.status-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-bottom: 16px;
}

.status-card,
.section-card,
.loading-card,
.empty-stage,
.warning-banner {
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 22px;
  background: rgba(255, 255, 255, 0.78);
}

.status-card {
  padding: 18px;
}

.warning-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 18px;
  margin-bottom: 16px;
  color: #92400e;
  background: rgba(255, 245, 230, 0.92);
}

.section-card {
  padding: 18px;
}

.section-card + .section-card {
  margin-top: 16px;
}

.section-card--timeline {
  display: flex;
  flex-direction: column;
  min-height: 0;
  flex: 1;
}

.section-head {
  margin-bottom: 14px;
}

.suggestion-grid,
.proposal-grid,
.tool-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.suggestion-card,
.proposal-card,
.tool-card,
.message-card {
  padding: 18px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.8);
}

.suggestion-card__head strong,
.proposal-card__head strong,
.tool-card__head strong {
  color: var(--ai-text);
}

.suggestion-card__head p,
.proposal-card__head p {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--ai-muted);
}

.suggestion-card__reason,
.proposal-card__summary {
  margin: 12px 0 0;
}

.suggestion-facts,
.proposal-facts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.suggestion-facts span,
.proposal-facts span {
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(15, 118, 110, 0.08);
}

.manifest-preview {
  margin-top: 14px;
  overflow: hidden;
  border-radius: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
}

.suggestion-actions,
.proposal-actions {
  margin-top: 14px;
}

.suggestion-hint {
  font-size: 12px;
}

.proposal-tags {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.proposal-preview {
  margin-top: 12px;
  padding: 12px 14px;
  border-radius: 16px;
  background: rgba(255, 248, 235, 0.92);
  color: #8a5a15;
  white-space: pre-wrap;
  word-break: break-word;
}

.tool-card p {
  margin: 12px 0 0;
}

.tool-card__meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-top: 14px;
}

.timeline-scroll {
  min-height: 340px;
  flex: 1;
}

.timeline-list {
  display: grid;
  gap: 14px;
}

.message-card {
  position: relative;
}

.message-card--assistant {
  background: linear-gradient(180deg, rgba(239, 252, 249, 0.96), rgba(255, 255, 255, 0.84));
}

.message-card--user {
  background: linear-gradient(180deg, rgba(248, 250, 255, 0.96), rgba(255, 255, 255, 0.84));
}

.message-card--tool {
  background: linear-gradient(180deg, rgba(255, 248, 237, 0.96), rgba(255, 255, 255, 0.84));
}

.message-role,
.message-usage {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.message-usage {
  font-size: 12px;
}

.message-content {
  margin: 14px 0 0;
  white-space: pre-wrap;
  color: var(--ai-text);
}

.message-suggestions {
  display: grid;
  gap: 10px;
  margin-top: 16px;
}

.message-suggestion {
  padding: 12px 14px;
  border-radius: 16px;
  background: rgba(15, 118, 110, 0.08);
}

.message-suggestion strong {
  display: block;
  color: var(--ai-text);
}

.message-suggestion span {
  display: block;
  margin-top: 6px;
  color: var(--ai-muted);
  line-height: 1.6;
}

.timeline-empty,
.empty-stage,
.loading-card {
  padding: 12px;
}

.composer-card {
  position: sticky;
  bottom: 0;
}

.composer-head {
  margin-bottom: 18px;
}

.composer-grid,
.proposal-form-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.proposal-form-grid--manifest {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.composer-field {
  margin-bottom: 0;
}

.composer-field--wide {
  grid-column: span 2;
}

.composer-shortcuts {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin: 18px 0 14px;
}

.composer-shortcuts span {
  font-size: 12px;
  color: var(--ai-muted);
}

.prompt-chip {
  padding: 8px 12px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.92);
  color: var(--ai-text);
  cursor: pointer;
  transition:
    transform 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}

.prompt-chip:hover {
  transform: translateY(-1px);
  border-color: rgba(15, 118, 110, 0.24);
  box-shadow: 0 12px 24px rgba(15, 118, 110, 0.12);
}

.composer-footer {
  align-items: center;
  margin-top: 16px;
}

.composer-hints {
  display: grid;
  gap: 6px;
  font-size: 12px;
}

.composer-actions {
  display: flex;
  gap: 10px;
  flex-shrink: 0;
}

@media (max-width: 1440px) {
  .status-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 1280px) {
  .workspace {
    grid-template-columns: 1fr;
  }

  .sidebar-card {
    min-height: 320px;
  }

  .suggestion-grid,
  .proposal-grid,
  .tool-grid,
  .composer-grid,
  .proposal-form-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .ai-page {
    padding: 14px;
  }

  .topbar-card,
  .control-row,
  .composer-footer,
  .detail-head,
  .warning-banner {
    flex-direction: column;
  }

  .topbar-card {
    padding: 22px;
  }

  .topbar-metrics,
  .control-item--grow,
  .control-select {
    width: 100%;
  }

  .control-actions,
  .composer-actions,
  .detail-head__actions {
    width: 100%;
  }

  .control-actions :deep(.el-button),
  .composer-actions :deep(.el-button) {
    flex: 1;
  }

  .status-grid,
  .suggestion-grid,
  .proposal-grid,
  .tool-grid,
  .composer-grid,
  .proposal-form-grid,
  .proposal-form-grid--manifest {
    grid-template-columns: 1fr;
  }

  .composer-field--wide {
    grid-column: span 1;
  }

  .tool-card__meta,
  .message-card__head,
  .section-head,
  .suggestion-actions,
  .proposal-card__head {
    flex-direction: column;
  }
}
</style>
