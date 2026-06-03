<template>
  <div class="ai-page">
    <section class="topbar-card">
      <div class="topbar-copy">
        <div class="hero-title">
          <span class="hero-title__icon">
            <el-icon><ChatDotRound /></el-icon>
          </span>
          <div>
            <p class="eyebrow">AI Ops Desk</p>
            <h1>集群排障工作台</h1>
          </div>
        </div>
        <p>
          继续追问、采集只读证据并人工确认提案，保持一条紧凑的排障链路。
        </p>
      </div>

      <div class="topbar-metrics">
        <div class="metric-pill">
          <span class="metric-pill__icon">
            <el-icon><Monitor /></el-icon>
          </span>
          <div>
            <span>集群</span>
            <strong>{{ currentClusterLabel }}</strong>
          </div>
        </div>
        <div class="metric-pill">
          <span class="metric-pill__icon">
            <el-icon><Collection /></el-icon>
          </span>
          <div>
            <span>会话</span>
            <strong>{{ conversationResult.total }} 条</strong>
          </div>
        </div>
        <div class="metric-pill metric-pill--warn">
          <span class="metric-pill__icon">
            <el-icon><Operation /></el-icon>
          </span>
          <div>
            <span>策略</span>
            <strong>只读采集 / 人工确认</strong>
          </div>
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
            :prefix-icon="Search"
            placeholder="按标题、摘要或问题关键字过滤"
            clearable
            @keyup.enter="loadConversations"
            @clear="loadConversations"
          />
        </el-form-item>

        <div class="control-actions">
          <el-button type="primary" :icon="Plus" @click="openCreateDialog">新建会话</el-button>
          <el-button :icon="RefreshRight" :loading="loadingConversations" @click="loadConversations">刷新列表</el-button>
        </div>
      </div>
    </section>

    <section class="workspace">
      <aside class="sidebar-card">
        <div class="sidebar-head">
          <div class="panel-title panel-title--tight">
            <span class="panel-title__icon">
              <el-icon><Collection /></el-icon>
            </span>
            <div>
              <h2>会话索引</h2>
              <p>从最近处理过的问题继续追问，或者新开一个诊断上下文。</p>
            </div>
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
                      ? `${assistantModeLabel(activeConversation.assistant_mode)} · 集群 #${activeConversation.cluster_id} · ${activeConversation.status}`
                      : '直接在下方输入问题即可自动创建会话。AI 会先尝试只读采集证据，再给出结论和建议动作。'
                  }}
                </p>
              </div>
            </div>
            <div class="detail-head__actions">
              <el-button v-if="activeConversation" text :icon="RefreshRight" @click="reloadActiveConversation">刷新详情</el-button>
              <el-button v-if="activeConversation" type="primary" plain :icon="MagicStick" @click="openProposalDialog">创建提案</el-button>
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

            <section v-if="activeConversation.tool_calls.length > 0" class="section-card">
              <div class="section-head">
                <div class="panel-title panel-title--tight">
                  <span class="panel-title__icon">
                    <el-icon><DataAnalysis /></el-icon>
                  </span>
                  <div>
                    <h3>自动取证记录</h3>
                    <p>这里展示 AI 为了回答问题所触发的只读查询和取证结果。</p>
                  </div>
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
                <div class="panel-title panel-title--tight">
                  <span class="panel-title__icon panel-title__icon--primary">
                    <el-icon><ChatDotRound /></el-icon>
                  </span>
                  <div>
                    <h3>对话时间线</h3>
                    <p>保留问题、AI 结论、工具取证轨迹以及结构化建议动作。</p>
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
            <div class="panel-title panel-title--tight">
              <span class="panel-title__icon panel-title__icon--accent">
                <el-icon><MagicStick /></el-icon>
              </span>
              <div>
                <p class="detail-kicker">提问入口</p>
                <h3>{{ activeConversationId ? '继续追问当前会话' : '发起新的 AI 诊断' }}</h3>
                <p>支持带上下文提问。诊断模式会优先采集只读证据，聊天模式更适合方案讨论和文档问答。</p>
              </div>
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

          <div class="composer-shell">
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
              class="composer-input"
              type="textarea"
              :rows="6"
              resize="none"
              placeholder="例如：帮我分析 payment 命名空间最近 30 分钟 Pod 重启频繁的问题，优先查看事件、日志和工作负载状态。"
              @keyup.ctrl.enter="sendMessage"
            />

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
                <el-button v-if="activeConversationId" :icon="MagicStick" @click="openProposalDialog">手动创建提案</el-button>
                <el-button type="primary" :icon="Promotion" :loading="sendingMessage" @click="sendMessage">
                  {{ activeConversationId ? '继续追问' : '发送诊断' }}
                </el-button>
              </div>
            </div>

            <div class="composer-meta">
              <span>{{ activeConversationId ? '发送后将继续当前会话' : '发送后将自动创建一个新会话' }}</span>
              <span>{{ modelUsageHint }}</span>
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
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ChatDotRound,
  Collection,
  Cpu,
  DataAnalysis,
  Document,
  MagicStick,
  Monitor,
  Operation,
  Plus,
  Promotion,
  RefreshRight,
  Search,
  SetUp
} from '@element-plus/icons-vue'

import {
  confirmAIActionProposal,
  createAIActionProposal,
  createAIConversation,
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
  const segments = [draftNamespace.value, draftResourceKind.value, draftResourceName.value].filter(Boolean)
  return segments.length > 0 ? `当前上下文: ${segments.join(' / ')}` : '当前未限定命名空间或资源范围'
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
    target_namespace: draftNamespace.value,
    target_name: draftResourceName.value,
    replicas: 1,
    manifest_yaml: '',
    default_namespace: draftNamespace.value,
    reason: ''
  })
}

function applySuggestionToForm(action: SuggestedAction) {
  Object.assign(proposalForm, {
    proposal_type: action.action_type,
    title: action.title || '',
    target_kind: action.target_kind || draftResourceKind.value || 'Deployment',
    target_namespace: action.target_namespace || draftNamespace.value,
    target_name: action.target_name || draftResourceName.value,
    replicas: action.replicas ?? 1,
    manifest_yaml: action.manifest_yaml || '',
    default_namespace: action.default_namespace || action.target_namespace || draftNamespace.value,
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
      namespace: draftNamespace.value || undefined,
      resource_kind: draftResourceKind.value || undefined,
      resource_name: draftResourceName.value || undefined
    })
    activeConversationId.value = result.conversation_id
    draftMessage.value = ''
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
  syncSelectedModel()
  await loadConversations()
}

function applyQuickPrompt(mode: 'diagnose' | 'chat', prompt: string) {
  if (!activeConversationId.value) {
    draftAssistantMode.value = mode
  }
  draftMessage.value = prompt
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

onMounted(async () => {
  await Promise.all([loadClusters(), loadModels()])
  await loadConversations()
})
</script>

<style scoped>
.ai-page {
  --ai-bg: linear-gradient(180deg, #f8fafc 0%, #f4f7fb 100%);
  --ai-card: #ffffff;
  --ai-border: rgba(15, 23, 42, 0.08);
  --ai-text: #1e293b;
  --ai-muted: #64748b;
  --ai-primary: #409eff;
  --ai-primary-soft: rgba(64, 158, 255, 0.12);
  --ai-accent: #0f766e;
  --ai-accent-soft: rgba(15, 118, 110, 0.12);
  --ai-warm: #d97706;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 100%;
  padding: 16px;
  background:
    radial-gradient(circle at top right, rgba(191, 219, 254, 0.18), transparent 24%),
    radial-gradient(circle at top left, rgba(253, 230, 138, 0.14), transparent 22%),
    var(--ai-bg);
}

.topbar-card,
.control-card,
.sidebar-card,
.detail-card,
.composer-card {
  border: 1px solid var(--ai-border);
  background: var(--ai-card);
  border-radius: 18px;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.045);
}

.topbar-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 18px;
  background: linear-gradient(135deg, #ffffff, #fbfdff);
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
  gap: 12px;
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
  width: 34px;
  height: 34px;
  border-radius: 12px;
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
  font-size: 20px;
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
  font-size: 13px;
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
  padding: 10px 12px;
  border-radius: 14px;
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
  padding: 14px 18px;
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
  grid-template-columns: 288px minmax(0, 1fr);
  gap: 16px;
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
  padding: 16px;
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
  padding: 10px 12px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 14px;
  background: linear-gradient(180deg, #fbfcfe, #ffffff);
  text-align: left;
  cursor: pointer;
  transition: transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
}

.shortcut-chip:hover,
.conversation-item:hover,
.conversation-item--active {
  transform: translateY(-1px);
  border-color: rgba(64, 158, 255, 0.28);
  box-shadow: 0 16px 28px rgba(64, 158, 255, 0.12);
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
  padding: 12px 14px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 14px;
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
  gap: 16px;
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
  border-radius: 18px;
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

.composer-card {
  position: static;
  padding-bottom: 0;
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

.composer-shell {
  padding: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 22px;
  background: linear-gradient(180deg, rgba(248, 250, 252, 0.9), rgba(255, 255, 255, 1));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.7);
}

.composer-shortcuts {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 14px;
}

.composer-shortcuts span {
  font-size: 12px;
  color: var(--ai-muted);
  margin-right: 2px;
}

.prompt-chip {
  padding: 8px 12px;
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

.composer-input :deep(.el-textarea__inner) {
  min-height: 176px !important;
  padding: 10px 0 0;
  border: 0;
  background: transparent;
  box-shadow: none;
  color: var(--ai-text);
  font-size: 15px;
  line-height: 1.75;
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
  min-height: 42px;
  padding: 0 12px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 14px;
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
  margin-top: 12px;
  font-size: 12px;
}

.composer-actions {
  display: flex;
  gap: 10px;
  flex-shrink: 0;
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
.detail-head__actions :deep(.el-button),
.composer-actions :deep(.el-button) {
  min-height: 40px;
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
  .composer-grid,
  .proposal-form-grid,
  .proposal-form-grid--manifest {
    grid-template-columns: 1fr;
  }

  .composer-field--wide {
    grid-column: span 1;
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
