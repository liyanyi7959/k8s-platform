<template>
  <div class="ai-page">
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
            :prefix-icon="Search"
            placeholder="按标题、摘要或问题关键字过滤"
            clearable
            @keyup.enter="loadConversations"
            @clear="loadConversations"
          />
        </el-form-item>

        <div class="control-actions">
          <el-tooltip content="新建会话" placement="bottom">
            <el-button type="primary" :icon="Plus" circle @click="openCreateDialog" />
          </el-tooltip>
          <el-tooltip content="刷新列表" placement="bottom">
            <el-button :icon="RefreshRight" circle :loading="loadingConversations" @click="loadConversations" />
          </el-tooltip>
        </div>
      </div>
    </section>

    <section class="workspace">
      <aside class="sidebar-card">
        <div class="sidebar-head">
          <h2>历史会话</h2>
          <el-tag type="info" effect="plain">{{ conversationResult.total }} 条</el-tag>
        </div>

        <el-scrollbar class="conversation-scroll">
          <EmptyState
            v-if="conversationResult.list.length === 0 && !loadingConversations"
            type="no-data"
            title="暂无会话"
            description=""
          >
            <el-button type="primary" :icon="Plus" circle @click="openCreateDialog" />
          </EmptyState>

          <div v-else class="conversation-list">
            <div
              v-for="item in orderedConversations"
              :key="item.id"
              class="conversation-item"
              :class="{
                'conversation-item--active': item.id === activeConversationId,
                'conversation-item--pinned': isConversationPinned(item.id)
              }"
              @click="selectConversation(item.id)"
              @contextmenu.prevent="openConversationMenu($event, item)"
            >
              <span class="conversation-dot" />
              <div class="conversation-item__main">
                <strong>{{ item.title || `会话 #${item.id}` }}</strong>
                <span>{{ formatDate(item.updated_at) }} · {{ item.message_count }} 条</span>
              </div>
              <el-icon v-if="isConversationPinned(item.id)" class="conversation-pin"><Top /></el-icon>
              <div class="conversation-item__actions">
                <el-tooltip content="更多操作" placement="top">
                  <el-button size="small" text :icon="MoreFilled" @click.stop="openConversationMenu($event, item)" />
                </el-tooltip>
              </div>
            </div>
          </div>
        </el-scrollbar>
      </aside>

      <div class="main-column">
        <section class="detail-card">
          <div class="detail-head">
            <div class="panel-title panel-title--tight">
              <span class="panel-title__icon panel-title__icon--primary">
                <el-icon><Document /></el-icon>
              </span>
              <div>
                <p class="detail-kicker">诊断会话</p>
                <h2>{{ activeConversation?.title || '开始一段新的 AI 排障对话' }}</h2>
                <p class="detail-subtitle">
                  {{
                    activeConversation
                      ? `${assistantModeLabel(activeConversation.assistant_mode)} · 集群 ${activeClusterLabel} · ${activeConversation.status}`
                      : ''
                  }}
                </p>
              </div>
            </div>
            <div class="detail-head__actions">
              <el-tooltip v-if="activeConversation" content="刷新详情" placement="bottom">
                <el-button circle text :icon="RefreshRight" @click="reloadActiveConversation" />
              </el-tooltip>
              <el-tooltip v-if="activeConversation" content="创建提案" placement="bottom">
                <el-button circle type="primary" plain :icon="MagicStick" @click="openProposalDialog" />
              </el-tooltip>
            </div>
          </div>

          <div v-if="loadingDetail" class="loading-card">
            <el-skeleton animated :rows="8" />
          </div>

          <template v-else-if="activeConversation">
            <div class="conversation-meta-line">
              <span>发起人 <strong>{{ activeConversation.created_by_name || `#${activeConversation.created_by}` }}</strong></span>
              <span>更新 <strong>{{ formatDate(activeConversation.updated_at) }}</strong></span>
              <el-popover
                v-if="activeConversation.tool_calls.length > 0"
                placement="bottom-start"
                width="460"
                trigger="click"
                popper-class="tool-call-popover"
              >
                <template #reference>
                  <button type="button" class="meta-link">
                    <el-icon><DataAnalysis /></el-icon>
                    <span>取证 <strong>{{ evidenceSummary }}</strong></span>
                  </button>
                </template>
                <div class="tool-popover-list">
                  <article v-for="tool in activeConversation.tool_calls" :key="tool.id" class="tool-popover-item">
                    <div>
                      <strong>{{ tool.tool_name }}</strong>
                      <span>{{ formatDate(tool.created_at) }}</span>
                    </div>
                    <p>{{ tool.result_summary || tool.error_message || '等待工具结果返回。' }}</p>
                  </article>
                </div>
              </el-popover>
              <span v-else>取证 <strong>{{ evidenceSummary }}</strong></span>
              <span>待确认 <strong>{{ pendingProposalCount }} 个</strong></span>
            </div>

            <div v-if="evidenceWarning" class="warning-banner">
              <strong>证据采集存在缺口</strong>
              <span>{{ evidenceWarning }}</span>
            </div>

            <section v-if="activeSuggestedActions.length > 0" class="section-card">
              <div class="section-head">
                <div class="panel-title panel-title--tight">
                  <span class="panel-title__icon panel-title__icon--accent">
                    <el-icon><MagicStick /></el-icon>
                  </span>
                  <div>
                    <h3>建议动作</h3>
                    <p>来自 AI 最近一次回复的结构化建议，可直接生成待确认提案。</p>
                  </div>
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
                <div class="panel-title panel-title--tight">
                  <span class="panel-title__icon panel-title__icon--warn">
                    <el-icon><Operation /></el-icon>
                  </span>
                  <div>
                    <h3>待确认与历史提案</h3>
                    <p>AI 只能创建建议，真正执行前仍需要人工确认。</p>
                  </div>
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

            <section class="section-card section-card--timeline">
              <div class="section-head">
                <div class="panel-title panel-title--tight">
                  <span class="panel-title__icon panel-title__icon--primary">
                    <el-icon><ChatDotRound /></el-icon>
                  </span>
                  <div>
                    <h3>对话时间线</h3>
                  </div>
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

                    <div class="message-content markdown-body" v-html="renderMarkdown(message.content)" />

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

          <div v-else class="empty-stage empty-stage--assistant">
            <h2>有什么我能帮你的吗？</h2>
          </div>
        </section>

        <section class="composer-card">
          <div class="composer-grid">
            <el-form-item label="会话模式" class="composer-field composer-field--mode">
              <el-radio-group v-model="draftAssistantMode" :disabled="Boolean(activeConversationId)">
                <el-radio-button label="diagnose">故障诊断</el-radio-button>
                <el-radio-button label="chat">通用聊天</el-radio-button>
              </el-radio-group>
            </el-form-item>

            <el-form-item label="命名空间" class="composer-field composer-field--namespace">
              <el-select
                v-model="draftNamespace"
                placeholder="全部命名空间"
                clearable
                filterable
                :loading="loadingNamespaces"
              >
                <el-option v-for="item in namespaceOptions" :key="item" :label="item" :value="item" />
              </el-select>
            </el-form-item>

            <el-form-item label="资源类型" class="composer-field composer-field--kind">
              <el-select v-model="draftResourceKind" placeholder="可选" clearable filterable>
                <el-option v-for="item in resourceKindOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>

            <el-form-item label="资源名称" class="composer-field composer-field--name">
              <el-select
                v-model="draftResourceName"
                placeholder="全部资源"
                clearable
                filterable
                :loading="loadingResourceNames"
                :disabled="!draftResourceKind"
              >
                <el-option v-for="item in resourceNameOptions" :key="item.value" :label="item.label" :value="item.value">
                  <span>{{ item.label }}</span>
                  <small v-if="item.namespace" class="resource-option-ns">{{ item.namespace }}</small>
                </el-option>
              </el-select>
            </el-form-item>
          </div>

          <div class="composer-shell">
            <el-input
              v-model="draftMessage"
              class="composer-input"
              type="textarea"
              :rows="3"
              resize="none"
              placeholder="输入问题，支持 Ctrl + Enter 发送；可直接粘贴图片"
              @paste="handleComposerPaste"
              @keyup.ctrl.enter="sendMessage"
            />

            <div v-if="pastedImages.length > 0" class="composer-images">
              <button
                v-for="image in pastedImages"
                :key="image.id"
                type="button"
                class="composer-image"
                @click="previewImage(image)"
              >
                <img :src="image.url" :alt="image.name" />
                <span>{{ image.name }}</span>
                <el-button
                  class="composer-image__remove"
                  size="small"
                  text
                  :icon="Close"
                  @click.stop="removePastedImage(image.id)"
                />
              </button>
            </div>

            <div class="composer-toolbar">
              <div class="composer-toolbar__left">
                <div class="composer-model-field">
                  <span class="composer-toolbar__label">
                    <el-icon><Cpu /></el-icon>
                    模型
                  </span>
                  <el-select
                    v-model="selectedModelId"
                    class="composer-model-select"
                    placeholder="跟随后端默认模型"
                    filterable
                    clearable
                    :loading="loadingModels"
                    :disabled="loadingModels || filteredModelOptions.length === 0"
                  >
                    <el-option
                      v-for="item in filteredModelOptions"
                      :key="item.id"
                      :label="formatModelOption(item)"
                      :value="item.id"
                    />
                  </el-select>
                </div>

                <div class="composer-scope-pill">
                  <el-icon><SetUp /></el-icon>
                  <span>{{ scopeSummary }}</span>
                </div>
              </div>

              <div class="composer-actions">
                <el-tooltip :content="activeConversationId ? '继续追问' : '发送诊断'" placement="top">
                  <el-button type="primary" circle :icon="Promotion" :loading="sendingMessage" @click="sendMessage" />
                </el-tooltip>
              </div>
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

    <el-dialog v-model="imagePreviewVisible" title="图片预览" width="720px" class="image-preview-dialog">
      <img v-if="previewingImage" :src="previewingImage.url" :alt="previewingImage.name" class="image-preview" />
    </el-dialog>

    <div
      v-if="conversationMenu.visible"
      class="conversation-context-menu"
      :style="{ left: `${conversationMenu.x}px`, top: `${conversationMenu.y}px` }"
      @click.stop
    >
      <button type="button" @click="togglePinConversationFromMenu">
        <el-icon><Top /></el-icon>
        <span>{{ conversationMenu.item && isConversationPinned(conversationMenu.item.id) ? '取消置顶' : '置顶会话' }}</span>
      </button>
      <button type="button" class="danger" @click="deleteConversationFromMenu">
        <el-icon><Delete /></el-icon>
        <span>删除会话</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ChatDotRound,
  Close,
  Cpu,
  DataAnalysis,
  Delete,
  Document,
  MagicStick,
  MoreFilled,
  Operation,
  Plus,
  Promotion,
  RefreshRight,
  Search,
  SetUp,
  Top
} from '@element-plus/icons-vue'

import {
  confirmAIActionProposal,
  createAIActionProposal,
  createAIConversation,
  deleteAIConversation,
  getAIConversationDetail,
  getAIConversations,
  getAIModels,
  sendAIChat,
  type AIActionProposalItem,
  type AIConversationDetail,
  type AIConversationItem,
  type AIModelItem
} from '@/features/ai/api/ai'
import { listClusters, type ClusterItem } from '@/features/clusters/api/clusters'
import { listNamespaces } from '@/features/k8s/api/namespace'
import { listPods } from '@/features/k8s/api/pod'
import { listWorkloads } from '@/features/k8s/api/workload'
import { listJobs, listCronJobs } from '@/features/k8s/api/batch'
import { listServices, listIngresses } from '@/features/k8s/api/network'
import { listConfigMaps, listSecrets } from '@/features/k8s/api/config'
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

interface ResourceNameOption {
  label: string
  value: string
  namespace: string
  name: string
}

interface PastedImage {
  id: string
  name: string
  url: string
  file: File
}

const clusters = ref<ClusterItem[]>([])
const selectedClusterId = ref<number>()
const keyword = ref('')
const conversationResult = ref<PageResult<AIConversationItem>>({
  list: [],
  total: 0,
  page: 1,
  page_size: 20
})
const loadingConversations = ref(false)
const loadingDetail = ref(false)
const sendingMessage = ref(false)
const creatingConversation = ref(false)
const createDialogVisible = ref(false)
const creatingProposal = ref(false)
const proposalDialogVisible = ref(false)
const activeConversationId = ref<number>()
const activeConversation = ref<AIConversationDetail>()
const availableModels = ref<AIModelItem[]>([])
const selectedModelId = ref<number>()
const loadingModels = ref(false)
const modelLoadError = ref('')
const confirmingProposalId = ref<number>()
const creatingSuggestedActionKey = ref('')
const namespaceOptions = ref<string[]>([])
const resourceNameOptions = ref<ResourceNameOption[]>([])
const loadingNamespaces = ref(false)
const loadingResourceNames = ref(false)
const pastedImages = ref<PastedImage[]>([])
const imagePreviewVisible = ref(false)
const previewingImage = ref<PastedImage>()
const pinnedConversationKey = 'ai-assistant:pinned-conversations'
const pinnedConversationIds = ref<number[]>(loadPinnedConversationIds())
const conversationMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
  item: undefined as AIConversationItem | undefined
})

const draftAssistantMode = ref<'diagnose' | 'chat'>('diagnose')
const draftNamespace = ref('')
const draftResourceKind = ref('')
const draftResourceName = ref('')
const draftMessage = ref('')

const createForm = reactive({
  title: '',
  assistant_mode: 'diagnose' as 'diagnose' | 'chat',
  opening_message: ''
})

const proposalForm = reactive({
  proposal_type: 'restart_workload' as 'restart_workload' | 'scale_workload' | 'apply_manifest',
  title: '',
  target_kind: 'Deployment',
  target_namespace: '',
  target_name: '',
  replicas: 1,
  manifest_yaml: '',
  default_namespace: '',
  reason: ''
})

const resourceKindOptions = [
  { label: 'Pod', value: 'Pod' },
  { label: 'Deployment', value: 'Deployment' },
  { label: 'StatefulSet', value: 'StatefulSet' },
  { label: 'DaemonSet', value: 'DaemonSet' },
  { label: 'Job', value: 'Job' },
  { label: 'CronJob', value: 'CronJob' },
  { label: 'Service', value: 'Service' },
  { label: 'Ingress', value: 'Ingress' },
  { label: 'ConfigMap', value: 'ConfigMap' },
  { label: 'Secret', value: 'Secret' }
]

const currentClusterLabel = computed(() => {
  const current = clusters.value.find((item) => item.id === selectedClusterId.value)
  return current?.name ?? '未选择'
})

const activeClusterLabel = computed(() => {
  const clusterId = activeConversation.value?.cluster_id ?? selectedClusterId.value
  const current = clusters.value.find((item) => item.id === clusterId)
  return current?.name ?? (clusterId ? `#${clusterId}` : currentClusterLabel.value)
})

const effectiveAssistantMode = computed<'diagnose' | 'chat'>(() => {
  const mode = activeConversation.value?.assistant_mode ?? draftAssistantMode.value
  return mode === 'chat' ? 'chat' : 'diagnose'
})

const filteredModelOptions = computed(() => {
  const allowTypes = effectiveAssistantMode.value === 'chat'
    ? new Set(['chat', 'reasoning'])
    : new Set(['chat', 'reasoning', 'vision'])
  return availableModels.value.filter((item) => item.enabled && allowTypes.has(item.model_type))
})

const selectedModel = computed(() => availableModels.value.find((item) => item.id === selectedModelId.value))
const selectedResourceOption = computed(() => resourceNameOptions.value.find((item) => item.value === draftResourceName.value))
const resolvedNamespace = computed(() => selectedResourceOption.value?.namespace || draftNamespace.value)
const resolvedResourceName = computed(() => selectedResourceOption.value?.name || draftResourceName.value)
const orderedConversations = computed(() => {
  const pinned = new Set(pinnedConversationIds.value)
  return [...conversationResult.value.list].sort((a, b) => {
    const aPinned = pinned.has(a.id)
    const bPinned = pinned.has(b.id)
    if (aPinned !== bPinned) return aPinned ? -1 : 1
    return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
  })
})

const modelUsageHint = computed(() => {
  if (loadingModels.value) return '正在加载模型列表'
  if (selectedModel.value) {
    return `当前发送将使用 ${selectedModel.value.provider_name} / ${selectedModel.value.name}`
  }
  if (modelLoadError.value) return modelLoadError.value
  return '未手动指定模型，将沿用后端默认路由'
})

const quickPrompts = computed(() => {
  if (effectiveAssistantMode.value === 'chat') {
    return [
      {
        title: '梳理根因',
        hint: '把当前现象整理成排障结论',
        mode: 'chat' as const,
        prompt: '请基于当前上下文，帮我梳理问题现象、根因假设和下一步排查建议。'
      },
      {
        title: '制定修复方案',
        hint: '给出低风险处置步骤',
        mode: 'chat' as const,
        prompt: '请结合当前证据，给我一套低风险的修复方案，并说明每一步的预期影响。'
      },
      {
        title: '复盘摘要',
        hint: '生成适合同步的简报',
        mode: 'chat' as const,
        prompt: '请把当前对话整理成一份适合同步给团队的故障复盘摘要。'
      }
    ]
  }

  return [
    {
      title: '排查频繁重启',
      hint: '聚焦事件、探针和 Pod 状态',
      mode: 'diagnose' as const,
      prompt: '请帮我分析当前工作负载最近频繁重启的原因，优先查看事件、探针、容器退出信息和副本状态。'
    },
    {
      title: '检查命名空间健康',
      hint: '快速扫描资源异常',
      mode: 'diagnose' as const,
      prompt: '请先概览这个命名空间的健康状态，指出异常的工作负载、Pod、事件和资源告警。'
    },
    {
      title: '分析发布失败',
      hint: '关注滚动更新与事件',
      mode: 'diagnose' as const,
      prompt: '请帮我分析当前应用发布失败的问题，重点看 Deployment 状态、事件、Pod 调度和镜像拉取。'
    }
  ]
})

const composerModeText = computed(() => assistantModeLabel(effectiveAssistantMode.value))
const composerModeType = computed(() => (effectiveAssistantMode.value === 'diagnose' ? 'warning' : 'success'))

const pendingProposalCount = computed(() => {
  return (activeConversation.value?.action_proposals ?? []).filter((item) => {
    return item.status === 'pending_confirm' || item.status === 'await_second_confirm' || item.status === 'pending'
  }).length
})

const evidenceSummary = computed(() => {
  const toolCalls = activeConversation.value?.tool_calls ?? []
  if (toolCalls.length === 0) return '暂无取证记录'
  const successCount = toolCalls.filter((item) => item.status === 'success').length
  return `${successCount}/${toolCalls.length} 次成功`
})

const evidenceWarning = computed(() => {
  const failed = (activeConversation.value?.tool_calls ?? []).filter((item) => item.status === 'failed')
  if (failed.length === 0) return ''
  return `有 ${failed.length} 次取证失败，建议补充上下文或重试诊断。`
})

const activeSuggestedActions = computed<SuggestedAction[]>(() => {
  const messages = activeConversation.value?.messages ?? []
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index]
    if (message.role !== 'assistant') continue
    const actions = extractSuggestedActions(message.structured).map((item): SuggestedAction => ({
      ...item,
      message_id: item.message_id ?? message.id
    }))
    if (actions.length > 0) return actions
  }
  return []
})

const scopeSummary = computed(() => {
  const segments = [resolvedNamespace.value, draftResourceKind.value, resolvedResourceName.value].filter(Boolean)
  return segments.length > 0 ? segments.join(' / ') : '全部范围'
})

function assistantModeLabel(mode?: string) {
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
  return 'primary'
}

function toolStatusType(status: string) {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'running') return 'warning'
  return 'info'
}

function proposalStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: '待确认',
    pending_confirm: '待确认',
    await_second_confirm: '待二次确认',
    approved: '已批准',
    executed: '已执行',
    failed: '执行失败',
    rejected: '已拒绝',
    cancelled: '已取消'
  }
  return labels[status] ?? status
}

function proposalStatusType(status: string) {
  if (status === 'executed') return 'success'
  if (status === 'failed' || status === 'rejected') return 'danger'
  if (status === 'pending_confirm' || status === 'await_second_confirm' || status === 'pending') return 'warning'
  return 'info'
}

function proposalRiskType(riskLevel: string) {
  if (riskLevel === 'high') return 'danger'
  if (riskLevel === 'medium') return 'warning'
  if (riskLevel === 'low') return 'success'
  return 'info'
}

function actionTypeLabel(actionType: string) {
  const labels: Record<string, string> = {
    restart_workload: '滚动重启',
    scale_workload: '调整副本数',
    apply_manifest: '应用 Manifest'
  }
  return labels[actionType] ?? actionType
}

function formatDate(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

function preferredModelByMode() {
  const preferredTypes = effectiveAssistantMode.value === 'chat'
    ? ['reasoning', 'chat']
    : ['reasoning', 'vision', 'chat']
  for (const modelType of preferredTypes) {
    const matched = filteredModelOptions.value.find((item) => item.model_type === modelType)
    if (matched) return matched
  }
  return filteredModelOptions.value[0]
}

function syncSelectedModel(preferConversation = false) {
  const conversationModelId = activeConversation.value?.model_id
  if (preferConversation && conversationModelId && filteredModelOptions.value.some((item) => item.id === conversationModelId)) {
    selectedModelId.value = conversationModelId
    return
  }
  if (selectedModelId.value && filteredModelOptions.value.some((item) => item.id === selectedModelId.value)) {
    return
  }
  selectedModelId.value = preferredModelByMode()?.id
}

function resolveSelectedModel() {
  return filteredModelOptions.value.find((item) => item.id === selectedModelId.value)
}

function formatModelOption(item: AIModelItem) {
  return `${item.name} · ${item.provider_name}`
}

async function loadModels() {
  loadingModels.value = true
  modelLoadError.value = ''
  try {
    availableModels.value = await getAIModels({ enabled: true })
    syncSelectedModel(Boolean(activeConversationId.value))
  } catch {
    availableModels.value = []
    selectedModelId.value = undefined
    modelLoadError.value = '当前账号未拿到可选模型，将沿用后端默认模型'
  } finally {
    loadingModels.value = false
  }
}

async function loadClusters() {
  const result = await listClusters({ page: 1, page_size: 200 })
  clusters.value = result.list
  if (!selectedClusterId.value && clusters.value.length > 0) {
    selectedClusterId.value = clusters.value[0].id
  }
  await loadNamespaces()
}

async function loadNamespaces() {
  if (!selectedClusterId.value) {
    namespaceOptions.value = []
    draftNamespace.value = ''
    return
  }
  loadingNamespaces.value = true
  try {
    const result = await listNamespaces(selectedClusterId.value, { sort_by: 'name', order: 'asc' })
    namespaceOptions.value = result.list
      .map((item) => String(item?.metadata?.name ?? '').trim())
      .filter(Boolean)
    if (draftNamespace.value && !namespaceOptions.value.includes(draftNamespace.value)) {
      draftNamespace.value = ''
    }
  } finally {
    loadingNamespaces.value = false
  }
}

async function loadResourceNames() {
  if (!selectedClusterId.value || !draftResourceKind.value) {
    resourceNameOptions.value = []
    draftResourceName.value = ''
    return
  }
  loadingResourceNames.value = true
  try {
    const params = { namespace: draftNamespace.value || undefined, sort_by: 'name' as const, order: 'asc' as const }
    const list = await fetchResourceList(selectedClusterId.value, draftResourceKind.value, params)
    resourceNameOptions.value = list
      .map(toResourceNameOption)
      .filter((item): item is ResourceNameOption => Boolean(item))
      .sort((a, b) => a.value.localeCompare(b.value, 'zh-Hans-CN'))
    if (draftResourceName.value && !resourceNameOptions.value.some((item) => item.value === draftResourceName.value)) {
      draftResourceName.value = ''
    }
  } finally {
    loadingResourceNames.value = false
  }
}

async function fetchResourceList(
  clusterId: number,
  kind: string,
  params: { namespace?: string; sort_by?: string; order?: 'asc' | 'desc' }
) {
  if (kind === 'Pod') return (await listPods(clusterId, params)).list
  if (kind === 'Deployment' || kind === 'StatefulSet' || kind === 'DaemonSet') {
    return (await listWorkloads(clusterId, { ...params, kind })).list
  }
  if (kind === 'Job') return (await listJobs(clusterId, params)).list
  if (kind === 'CronJob') return (await listCronJobs(clusterId, params)).list
  if (kind === 'Service') return (await listServices(clusterId, params)).list
  if (kind === 'Ingress') return (await listIngresses(clusterId, params)).list
  if (kind === 'ConfigMap') return (await listConfigMaps(clusterId, params)).list
  if (kind === 'Secret') return (await listSecrets(clusterId, params)).list
  return []
}

function toResourceNameOption(item: any): ResourceNameOption | null {
  const name = String(item?.metadata?.name ?? item?.name ?? '').trim()
  const namespace = String(item?.metadata?.namespace ?? item?.namespace ?? draftNamespace.value ?? '').trim()
  if (!name) return null
  return {
    name,
    namespace,
    value: namespace ? `${namespace}/${name}` : name,
    label: draftNamespace.value || !namespace ? name : `${namespace}/${name}`
  }
}

async function loadConversations() {
  loadingConversations.value = true
  try {
    conversationResult.value = await getAIConversations({
      page: 1,
      page_size: 50,
      cluster_id: selectedClusterId.value,
      keyword: keyword.value.trim() || undefined
    })
    if (activeConversationId.value && !conversationResult.value.list.some((item) => item.id === activeConversationId.value)) {
      activeConversationId.value = undefined
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
    syncSelectedModel(true)
  } finally {
    loadingDetail.value = false
  }
}

async function reloadActiveConversation() {
  if (!activeConversationId.value) return
  await selectConversation(activeConversationId.value)
}

function openCreateDialog() {
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
  const selectedModelOption = resolveSelectedModel()
  creatingConversation.value = true
  try {
    const result = await createAIConversation(selectedClusterId.value, {
      title: createForm.title || undefined,
      assistant_mode: createForm.assistant_mode,
      provider_id: selectedModelOption?.provider_id,
      model_id: selectedModelOption?.id,
      opening_message: createForm.opening_message || undefined
    })
    createDialogVisible.value = false
    draftMessage.value = ''
    await loadConversations()
    await selectConversation(result.id)
    ElMessage.success('AI 会话已创建')
  } finally {
    creatingConversation.value = false
  }
}

function resetProposalForm() {
  Object.assign(proposalForm, {
    proposal_type: 'restart_workload',
    title: '',
    target_kind: draftResourceKind.value || 'Deployment',
    target_namespace: resolvedNamespace.value,
    target_name: resolvedResourceName.value,
    replicas: 1,
    manifest_yaml: '',
    default_namespace: resolvedNamespace.value,
    reason: ''
  })
}

function applySuggestionToForm(action: SuggestedAction) {
  Object.assign(proposalForm, {
    proposal_type: action.action_type,
    title: action.title || '',
    target_kind: action.target_kind || draftResourceKind.value || 'Deployment',
    target_namespace: action.target_namespace || resolvedNamespace.value,
    target_name: action.target_name || resolvedResourceName.value,
    replicas: action.replicas ?? 1,
    manifest_yaml: action.manifest_yaml || '',
    default_namespace: action.default_namespace || action.target_namespace || resolvedNamespace.value,
    reason: action.reason || ''
  })
}

function openProposalDialog() {
  const suggestion = activeSuggestedActions.value[0]
  if (suggestion) {
    applySuggestionToForm(suggestion)
  } else {
    resetProposalForm()
  }
  proposalDialogVisible.value = true
}

function buildProposalRequest(input: {
  proposal_type: 'restart_workload' | 'scale_workload' | 'apply_manifest'
  title?: string
  target_kind?: string
  target_namespace?: string
  target_name?: string
  replicas?: number
  manifest_yaml?: string
  default_namespace?: string
  reason?: string
  message_id?: number
}) {
  if (!activeConversationId.value) {
    throw new Error('请先创建或选中一个 AI 会话')
  }

  const payload: Record<string, unknown> = {}
  let targetKind = (input.target_kind || '').trim()
  let targetNamespace = (input.target_namespace || '').trim()
  let targetName = (input.target_name || '').trim()

  if (input.proposal_type === 'scale_workload') {
    payload.replicas = input.replicas ?? 1
  }

  if (input.proposal_type === 'apply_manifest') {
    targetKind = 'Manifest'
    targetNamespace = (input.default_namespace || input.target_namespace || '').trim()
    targetName = (input.title || 'manifest').trim()
    payload.manifest_yaml = input.manifest_yaml || ''
    if (input.default_namespace) {
      payload.default_namespace = input.default_namespace
    }
  }

  return {
    conversation_id: activeConversationId.value,
    message_id: input.message_id,
    proposal_type: input.proposal_type,
    target_resource: {
      kind: targetKind,
      namespace: targetNamespace,
      name: targetName
    },
    payload: Object.keys(payload).length > 0 ? payload : undefined,
    reason: input.reason?.trim() || undefined
  }
}

async function submitProposal() {
  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择目标集群')
    return
  }
  creatingProposal.value = true
  try {
    const request = buildProposalRequest(proposalForm)
    await createAIActionProposal(selectedClusterId.value, request)
    proposalDialogVisible.value = false
    await reloadActiveConversation()
    ElMessage.success('提案已生成，等待人工确认')
  } finally {
    creatingProposal.value = false
  }
}

async function createProposalFromSuggestion(action: SuggestedAction) {
  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择目标集群')
    return
  }
  const key = suggestedActionKey(action)
  creatingSuggestedActionKey.value = key
  try {
    const request = buildProposalRequest({
      ...action,
      proposal_type: action.action_type
    })
    await createAIActionProposal(selectedClusterId.value, request)
    await reloadActiveConversation()
    ElMessage.success('建议动作已生成提案')
  } finally {
    creatingSuggestedActionKey.value = ''
  }
}

async function confirmProposal(proposal: AIActionProposalItem) {
  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择目标集群')
    return
  }
  try {
    await ElMessageBox.confirm(
      `即将执行提案“${proposal.title}”，该操作会对集群产生写入影响。`,
      '确认执行',
      {
        type: 'warning',
        confirmButtonText: '确认执行',
        cancelButtonText: '取消'
      }
    )
  } catch {
    return
  }

  confirmingProposalId.value = proposal.id
  try {
    await confirmAIActionProposal(selectedClusterId.value, proposal.id, {
      confirmation_text: 'operator-confirmed',
      confirm_risk: true
    })
    await reloadActiveConversation()
    ElMessage.success('提案已提交执行')
  } finally {
    confirmingProposalId.value = undefined
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
  const selectedModelOption = resolveSelectedModel()

  sendingMessage.value = true
  try {
    const result = await sendAIChat(selectedClusterId.value, {
      conversation_id: activeConversationId.value,
      message,
      assistant_mode: activeConversationId.value ? undefined : draftAssistantMode.value,
      provider_id: selectedModelOption?.provider_id,
      model_id: selectedModelOption?.id,
      prefer_model: selectedModelOption?.model_code,
      namespace: resolvedNamespace.value || undefined,
      resource_kind: draftResourceKind.value || undefined,
      resource_name: resolvedResourceName.value || undefined
    })
    activeConversationId.value = result.conversation_id
    draftMessage.value = ''
    clearPastedImages()
    await loadConversations()
    await selectConversation(result.conversation_id)
  } finally {
    sendingMessage.value = false
  }
}

async function handleClusterChange() {
  activeConversationId.value = undefined
  activeConversation.value = undefined
  draftAssistantMode.value = 'diagnose'
  draftNamespace.value = ''
  draftResourceKind.value = ''
  draftResourceName.value = ''
  resourceNameOptions.value = []
  syncSelectedModel()
  await Promise.all([loadNamespaces(), loadConversations()])
}

function applyQuickPrompt(mode: 'diagnose' | 'chat', prompt: string) {
  if (!activeConversationId.value) {
    draftAssistantMode.value = mode
  }
  draftMessage.value = prompt
}

function handleComposerPaste(event: ClipboardEvent) {
  const items = Array.from(event.clipboardData?.items ?? [])
  const imageFiles = items
    .filter((item) => item.kind === 'file' && item.type.startsWith('image/'))
    .map((item) => item.getAsFile())
    .filter((file): file is File => Boolean(file))

  if (imageFiles.length === 0) return
  event.preventDefault()
  const nextImages = imageFiles.map((file, index): PastedImage => ({
    id: `${Date.now()}-${index}-${file.name || 'paste'}`,
    name: file.name || `粘贴图片 ${pastedImages.value.length + index + 1}`,
    file,
    url: URL.createObjectURL(file)
  }))
  pastedImages.value = [...pastedImages.value, ...nextImages].slice(-6)
}

function previewImage(image: PastedImage) {
  previewingImage.value = image
  imagePreviewVisible.value = true
}

function removePastedImage(id: string) {
  const image = pastedImages.value.find((item) => item.id === id)
  if (image) URL.revokeObjectURL(image.url)
  pastedImages.value = pastedImages.value.filter((item) => item.id !== id)
}

function clearPastedImages() {
  pastedImages.value.forEach((item) => URL.revokeObjectURL(item.url))
  pastedImages.value = []
}

function loadPinnedConversationIds() {
  try {
    const raw = window.localStorage.getItem(pinnedConversationKey)
    const parsed = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? parsed.filter((item): item is number => typeof item === 'number') : []
  } catch {
    return []
  }
}

function persistPinnedConversationIds() {
  window.localStorage.setItem(pinnedConversationKey, JSON.stringify(pinnedConversationIds.value))
}

function isConversationPinned(id: number) {
  return pinnedConversationIds.value.includes(id)
}

function openConversationMenu(event: MouseEvent, item: AIConversationItem) {
  conversationMenu.visible = true
  conversationMenu.x = event.clientX
  conversationMenu.y = event.clientY
  conversationMenu.item = item
}

function closeConversationMenu() {
  conversationMenu.visible = false
}

function togglePinConversationFromMenu() {
  const item = conversationMenu.item
  if (!item) return
  if (isConversationPinned(item.id)) {
    pinnedConversationIds.value = pinnedConversationIds.value.filter((id) => id !== item.id)
  } else {
    pinnedConversationIds.value = [item.id, ...pinnedConversationIds.value]
  }
  persistPinnedConversationIds()
  closeConversationMenu()
}

async function deleteConversationFromMenu() {
  const item = conversationMenu.item
  if (!item) return
  try {
    await ElMessageBox.confirm(`删除会话“${item.title || `#${item.id}`}”？删除后不会在历史列表展示。`, '删除会话', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    closeConversationMenu()
    return
  }

  await deleteAIConversation(item.id)
  pinnedConversationIds.value = pinnedConversationIds.value.filter((id) => id !== item.id)
  persistPinnedConversationIds()
  if (activeConversationId.value === item.id) {
    activeConversationId.value = undefined
    activeConversation.value = undefined
  }
  closeConversationMenu()
  await loadConversations()
  ElMessage.success('会话已删除')
}

function escapeHtml(input: string) {
  return input
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function renderInlineMarkdown(input: string) {
  return input
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
}

function isMarkdownTableRow(line: string) {
  return line.includes('|') && line.replace(/\|/g, '').trim().length > 0
}

function isMarkdownTableSeparator(line: string) {
  if (!isMarkdownTableRow(line)) return false
  return splitMarkdownTableRow(line).every((cell) => /^:?-{3,}:?$/.test(cell))
}

function splitMarkdownTableRow(line: string) {
  return line
    .trim()
    .replace(/^\|/, '')
    .replace(/\|$/, '')
    .split('|')
    .map((cell) => cell.trim())
}

function renderMarkdownTable(headers: string[], rows: string[][]) {
  const head = headers.map((cell) => `<th>${renderInlineMarkdown(cell)}</th>`).join('')
  const body = rows
    .map((row) => {
      const cells = headers.map((_, index) => `<td>${renderInlineMarkdown(row[index] ?? '')}</td>`).join('')
      return `<tr>${cells}</tr>`
    })
    .join('')
  return `<table><thead><tr>${head}</tr></thead><tbody>${body}</tbody></table>`
}

function renderMarkdown(input: string) {
  const lines = escapeHtml(input || '').split(/\r?\n/)
  const html: string[] = []
  let inCode = false
  let listType: 'ul' | 'ol' | '' = ''
  const paragraph: string[] = []

  const flushParagraph = () => {
    if (paragraph.length === 0) return
    html.push(`<p>${renderInlineMarkdown(paragraph.join('<br>'))}</p>`)
    paragraph.length = 0
  }

  const closeList = () => {
    if (!listType) return
    html.push(`</${listType}>`)
    listType = ''
  }

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index]
    const trimmed = line.trim()
    if (trimmed.startsWith('```')) {
      flushParagraph()
      closeList()
      if (inCode) {
        html.push('</code></pre>')
      } else {
        html.push('<pre><code>')
      }
      inCode = !inCode
      continue
    }

    if (inCode) {
      html.push(`${line}\n`)
      continue
    }

    if (!trimmed) {
      flushParagraph()
      closeList()
      continue
    }

    if (/^(-{3,}|\*{3,}|_{3,})$/.test(trimmed)) {
      flushParagraph()
      closeList()
      html.push('<hr>')
      continue
    }

    if (isMarkdownTableRow(trimmed) && index + 1 < lines.length && isMarkdownTableSeparator(lines[index + 1].trim())) {
      flushParagraph()
      closeList()
      const headers = splitMarkdownTableRow(trimmed)
      const rows: string[][] = []
      index += 2
      while (index < lines.length) {
        const rowLine = lines[index].trim()
        if (!isMarkdownTableRow(rowLine) || isMarkdownTableSeparator(rowLine)) break
        rows.push(splitMarkdownTableRow(rowLine))
        index += 1
      }
      index -= 1
      html.push(renderMarkdownTable(headers, rows))
      continue
    }

    const heading = trimmed.match(/^(#{1,4})\s+(.+)$/)
    if (heading) {
      flushParagraph()
      closeList()
      const level = heading[1].length + 1
      html.push(`<h${level}>${renderInlineMarkdown(heading[2])}</h${level}>`)
      continue
    }

    const unordered = trimmed.match(/^[-*]\s+(.+)$/)
    if (unordered) {
      flushParagraph()
      if (listType !== 'ul') {
        closeList()
        html.push('<ul>')
        listType = 'ul'
      }
      html.push(`<li>${renderInlineMarkdown(unordered[1])}</li>`)
      continue
    }

    const ordered = trimmed.match(/^\d+[.)]\s+(.+)$/)
    if (ordered) {
      flushParagraph()
      if (listType !== 'ol') {
        closeList()
        html.push('<ol>')
        listType = 'ol'
      }
      html.push(`<li>${renderInlineMarkdown(ordered[1])}</li>`)
      continue
    }

    closeList()
    paragraph.push(trimmed)
  }

  flushParagraph()
  closeList()
  if (inCode) html.push('</code></pre>')
  return html.join('')
}

function suggestedActionKey(action: SuggestedAction) {
  return [
    action.message_id ?? 'message',
    action.action_type,
    action.target_kind ?? 'kind',
    action.target_namespace ?? 'ns',
    action.target_name ?? 'name'
  ].join(':')
}

function isSuggestedActionMaterialized(action: SuggestedAction) {
  if (action.proposal_created || action.proposal_id) return true
  return (activeConversation.value?.action_proposals ?? []).some((proposal) => {
    return proposal.message_id === action.message_id
      && proposal.action_type === action.action_type
      && proposal.target_kind === (action.target_kind || 'Manifest')
      && proposal.target_namespace === (action.target_namespace || action.default_namespace || '')
      && proposal.target_name === (action.target_name || action.title || 'manifest')
  })
}

function proposalPreview(proposal: AIActionProposalItem) {
  const change = proposal.change ?? {}
  if (proposal.action_type === 'scale_workload') {
    const replicas = asNumber(change.replicas)
    if (typeof replicas === 'number') return `目标副本数: ${replicas}`
  }
  if (proposal.action_type === 'restart_workload') {
    return `${proposal.target_kind}/${proposal.target_name}`
  }
  if (proposal.action_type === 'apply_manifest') {
    return asString(change.summary) ?? asString(change.preview) ?? asString(change.title) ?? ''
  }
  return ''
}

function proposalManifestText(proposal: AIActionProposalItem) {
  const change = proposal.change ?? {}
  return asString(change.manifest_yaml) ?? asString(change.manifest) ?? asString(change.yaml) ?? ''
}

function extractSuggestedActions(structured?: Record<string, unknown>): SuggestedAction[] {
  if (!structured || typeof structured !== 'object') return []
  const candidates = [
    (structured as Record<string, unknown>).suggested_actions,
    (structured as Record<string, unknown>).suggestedActions,
    (structured as Record<string, unknown>).action_proposals,
    (structured as Record<string, unknown>).actions,
    (structured as Record<string, unknown>).recommended_actions,
    (structured as Record<string, unknown>).next_actions
  ]

  const normalized: SuggestedAction[] = candidates.flatMap((candidate) => {
    if (!Array.isArray(candidate)) return [] as SuggestedAction[]
    return candidate.map(normalizeSuggestedAction).filter(isSuggestedAction)
  })

  const seen = new Set<string>()
  return normalized.filter((item) => {
    const key = suggestedActionKey(item)
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}

function normalizeSuggestedAction(value: unknown): SuggestedAction | null {
  if (!value || typeof value !== 'object') return null
  const record = value as Record<string, unknown>
  const targetResource = record.target_resource && typeof record.target_resource === 'object'
    ? record.target_resource as Record<string, unknown>
    : undefined

  const actionType = asString(record.action_type)
  if (actionType !== 'restart_workload' && actionType !== 'scale_workload' && actionType !== 'apply_manifest') {
    return null
  }

  return {
    message_id: asNumber(record.message_id) ?? asNumber(record.source_message_id),
    action_type: actionType,
    title: asString(record.title),
    reason: asString(record.reason) ?? asString(record.summary),
    target_kind: asString(record.target_kind) ?? asString(targetResource?.kind),
    target_namespace: asString(record.target_namespace) ?? asString(targetResource?.namespace),
    target_name: asString(record.target_name) ?? asString(targetResource?.name),
    replicas: asNumber(record.replicas) ?? asNumber(record.target_replicas),
    manifest_yaml: asString(record.manifest_yaml) ?? asString(record.manifest) ?? asString(record.yaml),
    default_namespace: asString(record.default_namespace),
    risk_level: asString(record.risk_level),
    requires_confirmation: asBoolean(record.requires_confirmation),
    auto_proposal_eligible: asBoolean(record.auto_proposal_eligible),
    proposal_created: asBoolean(record.proposal_created),
    proposal_id: asNumber(record.proposal_id) ?? asNumber(record.created_proposal_id)
  }
}

function isSuggestedAction(value: SuggestedAction | null): value is SuggestedAction {
  return Boolean(value)
}

function asString(value: unknown) {
  return typeof value === 'string' && value.trim() ? value : undefined
}

function asNumber(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function asBoolean(value: unknown) {
  return typeof value === 'boolean' ? value : undefined
}

watch(() => draftAssistantMode.value, () => {
  if (!activeConversationId.value) {
    syncSelectedModel()
  }
})

watch(() => draftResourceKind.value, () => {
  draftResourceName.value = ''
  void loadResourceNames()
})

watch(() => draftNamespace.value, () => {
  draftResourceName.value = ''
  void loadResourceNames()
})

onMounted(async () => {
  window.addEventListener('click', closeConversationMenu)
  await Promise.all([loadClusters(), loadModels()])
  await loadConversations()
})

onBeforeUnmount(() => {
  window.removeEventListener('click', closeConversationMenu)
  clearPastedImages()
})
</script>

<style scoped>
.ai-page {
  --ai-bg: #f5f7fb;
  --ai-card: #ffffff;
  --ai-border: rgba(15, 23, 42, 0.075);
  --ai-text: #1e293b;
  --ai-muted: #64748b;
  --ai-primary: #409eff;
  --ai-primary-soft: rgba(64, 158, 255, 0.12);
  --ai-accent: #0f766e;
  --ai-accent-soft: rgba(15, 118, 110, 0.12);
  --ai-warm: #d97706;
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 100%;
  padding: 14px;
  background: var(--ai-bg);
}

.topbar-card,
.control-card,
.sidebar-card,
.detail-card,
.composer-card {
  border: 1px solid var(--ai-border);
  background: var(--ai-card);
  border-radius: 16px;
  box-shadow: 0 8px 22px rgba(15, 23, 42, 0.04);
}

.topbar-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 12px 16px;
  background: #ffffff;
}

.eyebrow,
.detail-kicker {
  margin: 0 0 4px;
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--ai-warm);
}

.hero-title,
.panel-title {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.hero-title__icon,
.panel-title__icon,
.metric-pill__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.hero-title__icon {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: linear-gradient(135deg, #60a5fa, #2563eb);
  color: #ffffff;
  font-size: 18px;
  box-shadow: 0 10px 18px rgba(37, 99, 235, 0.18);
}

.panel-title__icon {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  background: #eff6ff;
  color: var(--ai-primary);
}

.panel-title__icon--primary {
  background: #e0f2fe;
  color: #0284c7;
}

.panel-title__icon--accent {
  background: rgba(15, 118, 110, 0.12);
  color: #0f766e;
}

.panel-title__icon--warn {
  background: rgba(245, 158, 11, 0.12);
  color: #d97706;
}

.panel-title--tight {
  align-items: center;
}

.topbar-copy h1,
.detail-head h2,
.composer-head h3 {
  margin: 0;
  color: var(--ai-text);
  font-weight: 700;
}

.topbar-copy h1 {
  font-size: 19px;
  line-height: 1.3;
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
.suggestion-hint,
.composer-meta {
  color: var(--ai-muted);
  line-height: 1.7;
}

.topbar-copy {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.topbar-copy p {
  max-width: 720px;
  margin: 0;
  font-size: 12px;
  line-height: 1.55;
}

.topbar-metrics {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  min-width: 0;
}

.metric-pill {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 11px;
  border-radius: 12px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: #f8fafc;
  min-height: 0;
}

.metric-pill__icon {
  width: 30px;
  height: 30px;
  border-radius: 10px;
  background: var(--ai-primary-soft);
  color: var(--ai-primary);
  font-size: 14px;
}

.metric-pill span,
.status-card span,
.message-usage span,
.tool-card__meta span,
.proposal-facts span,
.conversation-item__meta span,
.suggestion-facts span {
  display: inline-flex;
  font-size: 11px;
  color: var(--ai-muted);
}

.metric-pill strong,
.status-card strong {
  display: block;
  margin-top: 1px;
  color: var(--ai-text);
  line-height: 1.35;
}

.metric-pill > div {
  min-width: 0;
}

.metric-pill--warn {
  background: rgba(255, 250, 240, 0.96);
}

.metric-pill--warn .metric-pill__icon {
  background: rgba(245, 158, 11, 0.14);
  color: #d97706;
}

.control-card {
  padding: 12px 14px;
}

.control-row {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  flex-wrap: wrap;
}

.control-item {
  margin-bottom: 0;
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
  flex-wrap: wrap;
}

.workspace {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  gap: 12px;
  min-height: 0;
  flex: 1;
}

.sidebar-card,
.detail-card,
.composer-card {
  padding: 20px;
}

.sidebar-card {
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 12px;
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
.proposal-execution,
.conversation-item__head,
.conversation-item__meta,
.composer-toolbar {
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

.sidebar-head h2 {
  font-size: 16px;
  line-height: 1.35;
}

.sidebar-head p {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 1.55;
}

.sidebar-shortcuts {
  display: grid;
  gap: 8px;
  margin: 12px 0 10px;
}

.shortcut-chip {
  width: 100%;
  padding: 9px 11px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 12px;
  background: #ffffff;
  text-align: left;
  cursor: pointer;
  transition: transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
}

.shortcut-chip:hover,
.conversation-item:hover,
.conversation-item--active {
  transform: translateY(-1px);
  border-color: rgba(64, 158, 255, 0.3);
  box-shadow: 0 10px 22px rgba(64, 158, 255, 0.1);
}

.shortcut-chip span {
  display: block;
  font-weight: 600;
  color: var(--ai-text);
  font-size: 14px;
  line-height: 1.4;
}

.shortcut-chip small {
  display: block;
  margin-top: 3px;
  color: var(--ai-muted);
  font-size: 12px;
  line-height: 1.45;
}

.conversation-scroll {
  min-height: 0;
  flex: 1;
}

.conversation-list {
  display: grid;
  gap: 8px;
}

.conversation-item {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 12px;
  background: #ffffff;
  cursor: pointer;
  text-align: left;
  position: relative;
  transition: transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
}

.conversation-item::before {
  content: '';
  position: absolute;
  top: 12px;
  left: 12px;
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: rgba(64, 158, 255, 0.18);
}

.conversation-item--active::before,
.conversation-item:hover::before {
  background: var(--ai-primary);
}

.conversation-item__head strong {
  color: var(--ai-text);
  padding-left: 12px;
  font-size: 14px;
  line-height: 1.4;
}

.conversation-item p {
  margin: 8px 0 10px;
  min-height: 0;
  font-size: 13px;
  line-height: 1.5;
  display: -webkit-box;
  line-clamp: 2;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.conversation-item__meta {
  font-size: 11px;
}

.main-column {
  display: flex;
  flex-direction: column;
  gap: 12px;
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
  margin-bottom: 14px;
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
  border-radius: 16px;
  background: #ffffff;
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
  background: rgba(255, 247, 237, 0.96);
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
  border-radius: 18px;
  background: #ffffff;
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
  background: rgba(64, 158, 255, 0.1);
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
  background: #f8fafc;
  color: #475569;
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
  background: linear-gradient(180deg, rgba(239, 246, 255, 0.98), rgba(255, 255, 255, 1));
}

.message-card--user {
  background: linear-gradient(180deg, rgba(248, 250, 255, 0.98), rgba(255, 255, 255, 1));
}

.message-card--tool {
  background: linear-gradient(180deg, rgba(255, 247, 237, 0.98), rgba(255, 255, 255, 1));
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
  background: rgba(64, 158, 255, 0.1);
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

.empty-stage--assistant {
  display: grid;
  place-items: center;
  min-height: 430px;
}

.empty-stage--assistant h2 {
  margin: 0;
  color: #0f172a;
  font-size: 28px;
  line-height: 1.25;
  font-weight: 800;
}

.composer-card {
  position: static;
  padding: 24px 26px 16px;
}

.composer-head {
  align-items: center;
  margin-bottom: 22px;
}

.composer-head .panel-title__icon {
  width: 42px;
  height: 42px;
  border-radius: 14px;
  background: rgba(15, 118, 110, 0.1);
}

.composer-head h3 {
  font-size: 18px;
  line-height: 1.35;
}

.composer-head p {
  margin: 7px 0 0;
  font-size: 13px;
  line-height: 1.55;
}

.composer-grid {
  display: grid;
  grid-template-columns: minmax(430px, 1.1fr) minmax(260px, 0.7fr) minmax(240px, 0.6fr);
  gap: 18px 22px;
  align-items: center;
  margin-bottom: 18px;
}

.proposal-form-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.proposal-form-grid--manifest {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.composer-field {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 0;
}

.composer-field :deep(.el-form-item__label) {
  flex: 0 0 auto;
  height: auto;
  margin: 0 4px 0 0;
  padding: 0;
  color: #334155;
  font-size: 16px;
  font-weight: 700;
  line-height: 1.25;
}

.composer-field :deep(.el-form-item__content) {
  flex: 1;
  min-width: 0;
}

.composer-field :deep(.el-input__wrapper),
.composer-field :deep(.el-select__wrapper) {
  min-height: 48px;
  padding: 0 16px;
  border-radius: 14px;
  background: #ffffff;
  box-shadow: 0 0 0 1px rgba(15, 23, 42, 0.08);
}

.composer-field :deep(.el-input__wrapper:hover),
.composer-field :deep(.el-select__wrapper:hover) {
  box-shadow: 0 0 0 1px rgba(64, 158, 255, 0.22);
}

.composer-field--mode :deep(.el-radio-group) {
  display: grid;
  grid-template-columns: repeat(2, minmax(116px, 1fr));
  width: 282px;
  padding: 2px;
  border: 1px solid rgba(15, 23, 42, 0.09);
  border-radius: 15px;
  background: #ffffff;
}

.composer-field--mode :deep(.el-radio-button) {
  min-width: 0;
}

.composer-field--mode :deep(.el-radio-button__inner) {
  width: 100%;
  height: 42px;
  border: 0;
  border-radius: 12px;
  box-shadow: none;
  color: #475569;
  font-size: 14px;
  font-weight: 700;
  line-height: 42px;
  padding: 0 18px;
}

.composer-field--mode :deep(.el-radio-button.is-active .el-radio-button__inner) {
  color: #ffffff;
  background: var(--ai-primary);
}

.composer-field--name {
  grid-column: 1;
  max-width: 430px;
}

.composer-shell {
  overflow: hidden;
  padding: 0;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 18px;
  background: #ffffff;
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.055);
}

.composer-shortcuts {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin: 0;
  padding: 16px 20px 12px;
}

.composer-shortcuts span {
  font-size: 12px;
  color: var(--ai-muted);
  margin-right: 2px;
}

.prompt-chip {
  padding: 8px 14px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 999px;
  background: #ffffff;
  color: var(--ai-text);
  cursor: pointer;
  transition: transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
}

.prompt-chip:hover {
  transform: translateY(-1px);
  border-color: rgba(64, 158, 255, 0.24);
  box-shadow: 0 10px 18px rgba(64, 158, 255, 0.12);
}

.composer-input {
  display: block;
  padding: 0 20px;
}

.composer-input :deep(.el-textarea__inner) {
  min-height: 210px !important;
  padding: 16px;
  border: 0;
  border-radius: 0;
  background: #ffffff;
  box-shadow: inset 0 0 0 1px rgba(64, 158, 255, 0.18);
  color: var(--ai-text);
  font-size: 15px;
  line-height: 1.7;
}

.composer-toolbar {
  align-items: center;
  padding: 10px 20px 12px;
}

.composer-toolbar__left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.composer-toolbar__label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--ai-muted);
  white-space: nowrap;
}

.composer-model-field,
.composer-scope-pill {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  padding: 0 12px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 12px;
  background: #ffffff;
}

.composer-model-select {
  width: 220px;
}

.composer-model-select :deep(.el-select__wrapper) {
  min-height: 28px;
  padding: 0;
  background: transparent;
  box-shadow: none !important;
}

.composer-scope-pill {
  max-width: 360px;
  color: var(--ai-muted);
  font-size: 12px;
}

.composer-scope-pill .el-icon {
  color: var(--ai-primary);
}

.composer-meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 0;
  padding: 0 20px 14px;
  font-size: 12px;
}

.composer-actions {
  display: flex;
  gap: 10px;
  flex-shrink: 0;
}

.composer-actions :deep(.el-button) {
  min-width: 128px;
  min-height: 46px;
  padding: 0 20px;
  font-size: 15px;
  font-weight: 700;
}

.ai-page :deep(.el-input__wrapper),
.ai-page :deep(.el-select__wrapper),
.ai-page :deep(.el-input-number) {
  border-radius: 12px;
}

.ai-page :deep(.el-button) {
  border-radius: 12px;
}

.ai-page :deep(.el-button--primary) {
  box-shadow: 0 10px 18px rgba(64, 158, 255, 0.18);
}

.control-actions :deep(.el-button),
.detail-head__actions :deep(.el-button) {
  min-height: 40px;
}

.ai-page {
  height: calc(100vh - 88px);
  min-height: 560px;
  overflow: hidden;
  gap: 8px;
  padding: 10px;
  color: #172033;
  font-size: 13px;
}

.control-card {
  flex: 0 0 auto;
  padding: 8px 10px;
  border-radius: 12px;
}

.control-row {
  align-items: center;
  gap: 10px;
}

.control-item :deep(.el-form-item__label) {
  height: 32px;
  padding-right: 8px;
  color: #26364f;
  font-size: 13px;
  font-weight: 700;
  line-height: 32px;
}

.control-item :deep(.el-input__wrapper),
.control-item :deep(.el-select__wrapper) {
  min-height: 34px;
  border-radius: 10px;
}

.control-select {
  width: 220px;
}

.control-actions {
  gap: 8px;
}

.control-actions :deep(.el-button.is-circle) {
  width: 34px;
  height: 34px;
  min-height: 34px;
  padding: 0;
}

.workspace {
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 10px;
  min-height: 0;
  flex: 1;
}

.sidebar-card {
  min-height: 0;
  padding: 10px;
  border-radius: 12px;
}

.sidebar-head {
  align-items: center;
  min-height: 30px;
  margin-bottom: 8px;
}

.sidebar-head h2 {
  font-size: 14px;
}

.sidebar-head > span {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: #eef5ff;
  color: #2563eb;
  font-size: 12px;
  font-weight: 700;
}

.conversation-list {
  gap: 3px;
}

.conversation-item {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 6px;
  min-height: 34px;
  padding: 4px 6px;
  border: 0;
  border-radius: 8px;
  background: transparent;
}

.conversation-item::before {
  display: none;
}

.conversation-item:hover,
.conversation-item--active {
  transform: none;
  background: #f3f7ff;
  border-color: transparent;
  box-shadow: none;
}

.conversation-dot {
  width: 5px;
  height: 5px;
  border-radius: 999px;
  background: #cfe5ff;
}

.conversation-item--active .conversation-dot {
  background: var(--ai-primary);
}

.conversation-item__main {
  min-width: 0;
}

.conversation-item__main strong {
  display: block;
  overflow: hidden;
  color: #172033;
  font-size: 13px;
  font-weight: 600;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conversation-item__main span,
.conversation-item__meta {
  display: none;
}

.conversation-item__actions {
  opacity: 0;
}

.conversation-item:hover .conversation-item__actions,
.conversation-item--active .conversation-item__actions {
  opacity: 1;
}

.conversation-item__actions :deep(.el-button) {
  width: 24px;
  height: 24px;
  min-height: 24px;
  padding: 0;
}

.main-column {
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
  gap: 10px;
  min-height: 0;
}

.detail-card {
  min-height: 0;
  overflow: auto;
  padding: 14px 16px;
  border-radius: 12px;
}

.detail-head {
  align-items: center;
  margin-bottom: 10px;
}

.detail-head .panel-title__icon {
  width: 28px;
  height: 28px;
  border-radius: 8px;
}

.detail-kicker {
  margin-bottom: 2px;
  font-size: 11px;
  letter-spacing: 0;
}

.detail-head h2 {
  font-size: 17px;
  line-height: 1.3;
}

.detail-subtitle {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 1.45;
}

.detail-head__actions :deep(.el-button) {
  min-height: 30px;
  padding: 6px 10px;
}

.detail-head__actions :deep(.el-button.is-circle) {
  width: 34px;
  height: 34px;
  min-height: 34px;
  padding: 0;
}

.status-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 10px;
}

.status-card,
.section-card,
.suggestion-card,
.proposal-card,
.tool-card,
.message-card {
  border-radius: 10px;
}

.status-card {
  padding: 10px;
}

.section-card {
  padding: 12px;
}

.section-card + .section-card {
  margin-top: 10px;
}

.empty-stage--assistant {
  min-height: 0;
  height: 100%;
}

.empty-stage--assistant h2 {
  font-size: 22px;
}

.composer-card {
  flex: 0 0 auto;
  padding: 10px;
  border-radius: 12px;
}

.composer-grid {
  grid-template-columns: 180px minmax(150px, 0.7fr) 150px minmax(180px, 1fr);
  gap: 8px;
  margin-bottom: 8px;
}

.composer-field {
  gap: 6px;
}

.composer-field :deep(.el-form-item__label) {
  margin: 0;
  color: #26364f;
  font-size: 12px;
  font-weight: 700;
}

.composer-field :deep(.el-input__wrapper),
.composer-field :deep(.el-select__wrapper) {
  min-height: 32px;
  padding: 0 10px;
  border-radius: 9px;
}

.composer-field--mode :deep(.el-radio-group) {
  width: 126px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  padding: 1px;
  border-radius: 10px;
}

.composer-field--mode :deep(.el-radio-button__inner) {
  height: 28px;
  padding: 0 9px;
  border-radius: 8px;
  font-size: 12px;
  line-height: 28px;
}

.composer-field--name {
  grid-column: auto;
  max-width: none;
}

.resource-option-ns {
  float: right;
  margin-left: 12px;
  color: #94a3b8;
  font-size: 12px;
}

.composer-shell {
  border-radius: 10px;
  box-shadow: none;
}

.composer-input {
  padding: 0;
}

.composer-input :deep(.el-textarea__inner) {
  min-height: 86px !important;
  padding: 10px 12px;
  border-radius: 10px 10px 0 0;
  box-shadow: none;
  font-size: 13px;
  line-height: 1.55;
}

.composer-images {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 6px 8px;
  border-top: 1px solid #eef2f7;
  background: #fbfdff;
}

.composer-image {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 160px;
  height: 36px;
  padding: 3px 24px 3px 4px;
  border: 1px solid #dbe7f5;
  border-radius: 8px;
  background: #ffffff;
  color: #475569;
  cursor: pointer;
}

.composer-image img {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  object-fit: cover;
}

.composer-image span {
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.composer-image__remove {
  position: absolute;
  top: 3px;
  right: 2px;
}

.composer-toolbar {
  align-items: center;
  padding: 8px;
  border-top: 1px solid #eef2f7;
}

.composer-model-field,
.composer-scope-pill {
  min-height: 30px;
  padding: 0 8px;
  border-radius: 8px;
  font-size: 12px;
}

.composer-model-select {
  width: 190px;
}

.composer-actions :deep(.el-button) {
  min-width: auto;
  min-height: 34px;
  padding: 0 14px;
  border-radius: 9px;
  font-size: 13px;
}

.composer-actions :deep(.el-button.is-circle) {
  width: 42px;
  height: 42px;
  min-height: 42px;
  padding: 0;
  border-radius: 10px;
  font-size: 16px;
}

.image-preview {
  display: block;
  max-width: 100%;
  max-height: 70vh;
  margin: 0 auto;
  border-radius: 10px;
}

.conversation-pin {
  color: #2563eb;
  font-size: 13px;
}

.conversation-context-menu {
  position: fixed;
  z-index: 3000;
  min-width: 136px;
  padding: 5px;
  border: 1px solid rgba(15, 23, 42, 0.1);
  border-radius: 8px;
  background: #ffffff;
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.14);
}

.conversation-context-menu button {
  display: flex;
  align-items: center;
  width: 100%;
  gap: 8px;
  padding: 7px 9px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #243247;
  font-size: 13px;
  text-align: left;
  cursor: pointer;
}

.conversation-context-menu button:hover {
  background: #f3f7ff;
}

.conversation-context-menu button.danger {
  color: #dc2626;
}

.conversation-meta-line {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  margin: -2px 0 8px;
  padding: 0 2px 8px;
  border-bottom: 1px solid #eef2f7;
  color: #64748b;
  font-size: 12px;
  line-height: 1.4;
}

.conversation-meta-line span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}

.conversation-meta-line strong {
  color: #172033;
  font-weight: 650;
}

.meta-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  border: 0;
  background: transparent;
  color: #64748b;
  font: inherit;
  cursor: pointer;
}

.meta-link:hover {
  color: #2563eb;
}

.meta-link .el-icon {
  font-size: 13px;
}

:global(.tool-call-popover) {
  padding: 8px !important;
  border-radius: 10px !important;
}

:global(.tool-call-popover .tool-popover-list) {
  display: grid;
  gap: 6px;
  max-height: 360px;
  overflow: auto;
}

:global(.tool-call-popover .tool-popover-item) {
  padding: 8px;
  border: 1px solid #eef2f7;
  border-radius: 8px;
  background: #ffffff;
}

:global(.tool-call-popover .tool-popover-item > div) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: #172033;
  font-size: 12px;
}

:global(.tool-call-popover .tool-popover-item span) {
  color: #64748b;
  white-space: nowrap;
}

:global(.tool-call-popover .tool-popover-item p) {
  margin: 5px 0 0;
  color: #475569;
  font-size: 12px;
  line-height: 1.5;
}

.section-card {
  border-radius: 10px;
}

.section-head {
  align-items: center;
  margin-bottom: 10px;
}

.section-head h3 {
  font-size: 14px;
  line-height: 1.3;
}

.section-head p {
  display: none;
}

.tool-grid {
  margin-top: 6px;
}

.section-card--timeline {
  overflow: hidden;
}

.timeline-scroll {
  min-height: 0;
  overflow: hidden;
}

.timeline-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 0 4px 4px;
}

.message-card {
  width: fit-content;
  max-width: min(780px, 78%);
  padding: 10px 12px;
  border-radius: 12px;
  overflow: hidden;
}

.message-card--user {
  align-self: flex-end;
  background: #edf5ff;
  border-color: #cfe2ff;
}

.message-card--assistant,
.message-card--tool,
.message-card--system {
  align-self: flex-start;
}

.message-card--assistant {
  background: #ffffff;
}

.message-card--tool {
  background: #fff7ed;
}

.message-card__head {
  align-items: center;
  gap: 10px;
}

.message-role,
.message-usage {
  gap: 6px;
}

.message-role span,
.message-usage span {
  font-size: 11px;
}

.message-content {
  margin: 8px 0 0;
  color: #172033;
  font-size: 13px;
  line-height: 1.65;
  overflow-wrap: anywhere;
  white-space: normal;
  word-break: break-word;
}

.markdown-body :deep(p) {
  margin: 0 0 8px;
}

.markdown-body :deep(p:last-child),
.markdown-body :deep(ul:last-child),
.markdown-body :deep(ol:last-child),
.markdown-body :deep(pre:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4),
.markdown-body :deep(h5) {
  margin: 10px 0 6px;
  color: #111827;
  font-size: 14px;
  line-height: 1.45;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  margin: 6px 0 8px;
  padding-left: 18px;
}

.markdown-body :deep(hr) {
  height: 1px;
  margin: 10px 0;
  border: 0;
  background: #e5e7eb;
}

.markdown-body :deep(table) {
  width: 100%;
  margin: 8px 0 10px;
  border-collapse: collapse;
  overflow: hidden;
  border: 1px solid #dbe5f1;
  border-radius: 8px;
  font-size: 12px;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  padding: 7px 9px;
  border: 1px solid #dbe5f1;
  text-align: left;
  vertical-align: top;
}

.markdown-body :deep(th) {
  background: #f5f8fc;
  color: #172033;
  font-weight: 700;
}

.markdown-body :deep(td) {
  background: #ffffff;
}

.markdown-body :deep(li + li) {
  margin-top: 3px;
}

.markdown-body :deep(code) {
  padding: 1px 4px;
  border-radius: 5px;
  background: #eef2f7;
  color: #0f172a;
  font-family: Consolas, Monaco, monospace;
  font-size: 12px;
}

.markdown-body :deep(pre) {
  max-width: 100%;
  margin: 8px 0;
  padding: 9px 10px;
  overflow: auto;
  border-radius: 8px;
  background: #0f172a;
  color: #e5e7eb;
}

.markdown-body :deep(pre code) {
  padding: 0;
  background: transparent;
  color: inherit;
  white-space: pre;
}

@media (max-width: 1440px) {
  .status-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 1280px) {
  .topbar-card {
    flex-direction: column;
    align-items: stretch;
  }

  .workspace {
    grid-template-columns: 1fr;
  }

  .sidebar-card {
    min-height: 320px;
  }

  .suggestion-grid,
  .proposal-grid,
  .tool-grid,
  .proposal-form-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .composer-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .composer-field--mode,
  .composer-field--name {
    grid-column: span 1;
    max-width: none;
  }
}

@media (max-width: 900px) {
  .ai-page {
    padding: 14px;
  }

  .topbar-card,
  .control-row,
  .composer-toolbar,
  .detail-head,
  .warning-banner {
    flex-direction: column;
  }

  .topbar-card {
    padding: 14px;
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

  .composer-toolbar__left,
  .composer-model-field,
  .composer-scope-pill,
  .composer-model-select {
    width: 100%;
    max-width: none;
  }

  .composer-model-field {
    flex-wrap: wrap;
    justify-content: flex-start;
    padding: 10px 12px;
  }

  .composer-meta {
    flex-direction: column;
  }

  .status-grid,
  .suggestion-grid,
  .proposal-grid,
  .tool-grid,
  .proposal-form-grid,
  .proposal-form-grid--manifest {
    grid-template-columns: 1fr;
  }

  .composer-grid {
    grid-template-columns: 1fr;
  }

  .composer-field {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
  }

  .composer-field--mode :deep(.el-radio-group) {
    width: 100%;
  }

  .sidebar-head,
  .message-card__head,
  .tool-card__meta,
  .section-head,
  .suggestion-actions,
  .proposal-card__head {
    flex-direction: column;
  }
}

@media (max-width: 640px) {
  .topbar-copy h1 {
    font-size: 18px;
  }

  .conversation-item__head,
  .tool-card__meta,
  .message-card__head,
  .section-head,
  .suggestion-actions,
  .proposal-card__head,
  .hero-title {
    flex-direction: column;
  }
}
</style>
