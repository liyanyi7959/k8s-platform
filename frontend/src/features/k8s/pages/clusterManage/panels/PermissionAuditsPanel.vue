<template>
  <div class="permission-audit-panel">
    <div class="srv-query-bar permission-audit-toolbar">
      <div class="qb-search">
        <el-icon class="qb-search-icon"><Search /></el-icon>
        <el-input
          v-model="query.keyword"
          class="qb-keyword"
          size="default"
          placeholder="搜索分析名称或结果摘要…"
          clearable
          @keyup.enter="loadList"
        />
      </div>

      <div class="qb-filters">
        <el-select v-model="query.status" class="qb-select" size="default" clearable placeholder="状态">
          <el-option label="待执行" value="pending" />
          <el-option label="运行中" value="running" />
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
          <el-option label="部分完成" value="incomplete" />
          <el-option label="已取消" value="canceled" />
        </el-select>
        <el-select v-model="query.risk_level" class="qb-select" size="default" clearable placeholder="风险等级">
          <el-option label="Critical" value="critical" />
          <el-option label="High" value="high" />
          <el-option label="Medium" value="medium" />
          <el-option label="Low" value="low" />
        </el-select>
      </div>

      <div class="qb-actions">
        <el-button class="qb-btn" size="default" :icon="RefreshRight" :loading="loadingSummary || loadingList" @click="refreshAll">刷新</el-button>
        <el-button class="qb-btn" size="default" :icon="Document" @click="openRbacDialog">建议 RBAC</el-button>
        <el-button class="qb-btn" size="default" :icon="Connection" :loading="creatingAdhoc" @click="openAdhocDialog">临时分析</el-button>
        <el-button class="qb-btn qb-btn--primary" type="primary" size="default" :icon="Plus" :loading="creating" @click="openCreateDialog">发起分析</el-button>
      </div>
    </div>

    <el-card shadow="never" class="summary-card">
      <template #header>
        <div class="summary-card-head">
          <div>
            <div class="summary-card-title">最近一次权限分析</div>
            <div class="summary-card-sub">面向当前集群的最小权限画像与运行期 RBAC 风险结果</div>
          </div>
          <el-tag :type="statusTagType(latestAudit?.status)">{{ statusText(latestAudit?.status) }}</el-tag>
        </div>
      </template>

      <EmptyState v-if="!latestAudit && !loadingSummary" type="no-data" description="当前集群还没有权限分析记录" />

      <div v-else class="summary-grid" v-loading="loadingSummary">
        <div class="summary-grid-main">
          <div class="summary-kpi-list">
            <div class="summary-kpi">
              <span class="summary-kpi-label">资源总数</span>
              <strong class="summary-kpi-value">{{ numberText(latestAudit?.summary?.total_resources) }}</strong>
            </div>
            <div class="summary-kpi">
              <span class="summary-kpi-label">集群级资源</span>
              <strong class="summary-kpi-value">{{ numberText(latestAudit?.summary?.cluster_scoped_resources) }}</strong>
            </div>
            <div class="summary-kpi">
              <span class="summary-kpi-label">高权限工作负载</span>
              <strong class="summary-kpi-value">{{ numberText(latestAudit?.summary?.high_privilege_workloads) }}</strong>
            </div>
            <div class="summary-kpi">
              <span class="summary-kpi-label">可降权候选</span>
              <strong class="summary-kpi-value">{{ numberText(latestAudit?.summary?.namespace_only_candidate_workloads) }}</strong>
            </div>
          </div>

          <div class="summary-meta" v-if="latestAudit">
            <span>分析时间：{{ formatTime(latestAudit.updated_at || latestAudit.created_at) }}</span>
            <span v-if="latestAudit.task_id">任务 ID：{{ latestAudit.task_id }}</span>
            <span>来源：{{ sourceText(latestAudit.source_type) }}</span>
            <el-button v-if="latestAudit.status === 'success'" link type="primary" size="small" @click="openDetail(latestAudit.id)">查看详情 →</el-button>
          </div>
        </div>

        <div class="summary-risk" v-if="latestAudit?.summary?.risk || latestAudit?.stats?.risk">
          <div class="risk-chip risk-chip--critical">Critical {{ riskCount('critical', latestAudit) }}</div>
          <div class="risk-chip risk-chip--high">High {{ riskCount('high', latestAudit) }}</div>
          <div class="risk-chip risk-chip--medium">Medium {{ riskCount('medium', latestAudit) }}</div>
          <div class="risk-chip risk-chip--low">Low {{ riskCount('low', latestAudit) }}</div>
        </div>
      </div>
    </el-card>

    <EnhancedTable
      v-model:page="page"
      v-model:page-size="pageSize"
      class="audit-history-table"
      pagination
      persist-key="k8s:permission_audits:history"
      :data="list"
      :columns="columns"
      :total="total"
      :loading="loadingList"
      :row-key="'id'"
      stripe
      border
      @refresh="loadList"
      @sort-change="onSortChange"
    >
      <template #cell-display_name="{ row }">
        <div class="audit-title-cell">
          <div class="audit-title-main">{{ auditTitle(row) }}</div>
          <div class="audit-title-meta">
            <span>分析 #{{ row.id }}</span>
            <span v-if="row.task_id">任务 {{ row.task_id }}</span>
          </div>
        </div>
      </template>

      <template #cell-status="{ row }">
        <el-tag :type="statusTagType(row.status)">{{ statusText(row.status) }}</el-tag>
      </template>

      <template #cell-source_type="{ row }">
        <div class="audit-source-cell">
          <el-tag size="small" effect="plain" :type="sourceTagType(row.source_type)">{{ sourceText(row.source_type) }}</el-tag>
          <span class="audit-source-meta">{{ row.cluster_name || '当前集群' }}</span>
        </div>
      </template>

      <template #cell-summary="{ row }">
        <div class="summary-inline">
          <span class="summary-chip">资源 {{ numberText(row.summary?.total_resources) }}</span>
          <span class="summary-chip">集群级 {{ numberText(row.summary?.cluster_scoped_resources) }}</span>
          <span class="summary-chip summary-chip--high">高权限 {{ numberText(row.summary?.high_privilege_workloads) }}</span>
          <span class="summary-chip summary-chip--critical">Critical {{ riskCount('critical', row) }}</span>
        </div>
      </template>

      <template #cell-created_at="{ row }">
        <span>{{ formatTime(row.created_at) }}</span>
      </template>

      <template #cell-actions="{ row }">
        <div class="k8s-act-group">
          <el-tooltip content="查看日志" placement="top" :show-after="300">
            <button class="k8s-act-btn" @click="openLogs(row)"><el-icon><Document /></el-icon></button>
          </el-tooltip>
          <el-tooltip content="查看详情" placement="top" :show-after="300">
            <button class="k8s-act-btn k8s-act-btn--info" @click="openDetail(row.id)"><el-icon><View /></el-icon></button>
          </el-tooltip>
          <el-tooltip v-if="canCancelAudit(row)" content="取消任务" placement="top" :show-after="300">
            <button class="k8s-act-btn k8s-act-btn--danger" :disabled="cancelingAuditId === row.id" @click="cancelAudit(row)"><el-icon><CloseBold /></el-icon></button>
          </el-tooltip>
        </div>
      </template>
    </EnhancedTable>

    <el-dialog v-model="createVisible" title="发起权限分析" width="620px">
      <el-form label-width="120px" @submit.prevent>
        <el-form-item label="分析模式">
          <el-tag type="info">全集群 full</el-tag>
        </el-form-item>
        <el-form-item label="目标命名空间">
          <el-select
            v-model="createForm.namespaces"
            class="w-full"
            multiple
            collapse-tags
            collapse-tags-tooltip
            :max-collapse-tags="2"
            clearable
            filterable
            placeholder="留空表示分析全集群"
          >
            <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
          </el-select>
        </el-form-item>
        <el-form-item label="标签选择器">
          <el-input v-model="createForm.label_selector" placeholder="例如 app.kubernetes.io/instance=my-release" clearable />
        </el-form-item>
        <el-form-item label="平台映射">
          <el-switch v-model="createForm.include_platform_mapping" />
          <span class="form-tip">尝试将结果映射到平台纳管工作负载记录</span>
        </el-form-item>
        <el-form-item label="运行期 RBAC">
          <el-switch v-model="createForm.include_runtime_rbac" />
          <span class="form-tip">分析 ServiceAccount 绑定链、RBAC 写权限与高风险运行时能力</span>
        </el-form-item>
        <el-form-item label="归属识别">
          <el-switch v-model="createForm.include_ownership_detection" />
          <span class="form-tip">识别资源是否属于平台纳管工作负载、共享能力或无关资源</span>
        </el-form-item>
        <el-form-item label="资源范围">
          <el-checkbox-group v-model="createForm.resource_allowlist" class="allowlist-grid">
            <el-checkbox v-for="item in resourceAllowlistOptions" :key="item.value" :label="item.value">{{ item.label }}</el-checkbox>
          </el-checkbox-group>
          <div class="form-tip">不勾选表示扫描默认资源全集；勾选后仅扫描所选资源类型。</div>
        </el-form-item>
        <el-form-item label="默认能力">
          <div class="default-capability-list">
            <el-tag size="small" :type="createForm.include_runtime_rbac ? 'success' : 'info'">运行期 RBAC 分析</el-tag>
            <el-tag size="small" :type="createForm.include_ownership_detection ? 'success' : 'info'">资源归属识别</el-tag>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="submitCreate">开始分析</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="adhocVisible" title="临时 kubeconfig 权限分析" width="720px">
      <el-form label-width="120px" @submit.prevent>
        <el-form-item label="显示名称">
          <el-input v-model="adhocForm.display_name" placeholder="例如 cwbk-临时分析" clearable />
        </el-form-item>
        <el-form-item label="目标命名空间">
          <el-select
            v-model="adhocForm.namespaces"
            class="w-full"
            multiple
            collapse-tags
            collapse-tags-tooltip
            :max-collapse-tags="2"
            clearable
            filterable
            allow-create
            default-first-option
            placeholder="留空表示分析全集群"
          >
            <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
          </el-select>
        </el-form-item>
        <el-form-item label="标签选择器">
          <el-input v-model="adhocForm.label_selector" placeholder="例如 app=my-release" clearable />
        </el-form-item>
        <el-form-item label="平台映射">
          <el-switch v-model="adhocForm.include_platform_mapping" />
          <span class="form-tip">仅对已纳管集群效果更好，临时分析一般建议关闭</span>
        </el-form-item>
        <el-form-item label="运行期 RBAC">
          <el-switch v-model="adhocForm.include_runtime_rbac" />
        </el-form-item>
        <el-form-item label="归属识别">
          <el-switch v-model="adhocForm.include_ownership_detection" />
        </el-form-item>
        <el-form-item label="资源范围">
          <el-checkbox-group v-model="adhocForm.resource_allowlist" class="allowlist-grid">
            <el-checkbox v-for="item in resourceAllowlistOptions" :key="item.value" :label="item.value">{{ item.label }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="kubeconfig">
          <el-input v-model="adhocForm.kubeconfig" type="textarea" :rows="12" placeholder="粘贴只读或受控权限的 kubeconfig 内容" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="adhocVisible = false">取消</el-button>
        <el-button type="primary" :loading="creatingAdhoc" @click="submitAdhoc">开始分析</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" size="78%" :title="detailTitle" destroy-on-close @closed="resetDetailState">
      <div class="detail-shell" v-loading="loadingDetail">
        <template v-if="detail">
          <el-row :gutter="12" class="detail-summary-row">
            <el-col :xs="24" :sm="12" :lg="6"><div class="mini-stat"><span>资源总数</span><strong>{{ numberText(detail.summary?.total_resources) }}</strong></div></el-col>
            <el-col :xs="24" :sm="12" :lg="6"><div class="mini-stat"><span>集群级资源</span><strong>{{ numberText(detail.summary?.cluster_scoped_resources) }}</strong></div></el-col>
            <el-col :xs="24" :sm="12" :lg="6"><div class="mini-stat"><span>高权限工作负载</span><strong>{{ numberText(detail.summary?.high_privilege_workloads) }}</strong></div></el-col>
            <el-col :xs="24" :sm="12" :lg="6"><div class="mini-stat"><span>部署阻塞项</span><strong>{{ numberText(detail.stats?.blockers?.deployment_blockers) }}</strong></div></el-col>
          </el-row>

          <el-descriptions :column="2" border class="detail-descriptions">
            <el-descriptions-item label="状态"><el-tag :type="statusTagType(detail.status)">{{ statusText(detail.status) }}</el-tag></el-descriptions-item>
            <el-descriptions-item label="来源">{{ sourceText(detail.source_type) }}</el-descriptions-item>
            <el-descriptions-item label="集群">{{ detail.cluster?.name || '-' }}</el-descriptions-item>
            <el-descriptions-item label="任务 ID">{{ detail.task_id || '-' }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ formatTime(detail.created_at) }}</el-descriptions-item>
            <el-descriptions-item label="更新时间">{{ formatTime(detail.updated_at) }}</el-descriptions-item>
          </el-descriptions>

          <div class="detail-actions">
            <el-button v-if="detail?.task_id" :icon="Document" @click="openLogs(detail)">查看任务日志</el-button>
            <el-button v-if="canCancelAudit(detail)" type="danger" plain :loading="cancelingAuditId === detail.id" :icon="CloseBold" @click="cancelAudit(detail)">取消任务</el-button>
          </div>

          <el-alert
            v-if="detail.error && Object.keys(detail.error).length > 0"
            class="detail-error-alert"
            title="本次分析包含异常信息"
            type="warning"
            :closable="false"
            show-icon
          >
            <template #default>
              <div class="detail-error-text">{{ stringify(detail.error) }}</div>
            </template>
          </el-alert>

          <div class="srv-query-bar detail-filters">
            <div class="qb-search">
              <el-icon class="qb-search-icon"><Search /></el-icon>
              <el-input v-model="findingsQuery.keyword" class="qb-keyword" size="default" placeholder="搜索 findings…" clearable @keyup.enter="loadFindings" />
            </div>

            <div class="qb-filters">
              <el-select v-model="findingsQuery.finding_type" class="qb-select" size="default" clearable placeholder="类型">
                <el-option label="工作负载" value="workload" />
                <el-option label="资源" value="resource" />
                <el-option label="平台映射" value="app_release" />
                <el-option label="扫描错误" value="error" />
              </el-select>
              <el-select v-model="findingsQuery.risk_level" class="qb-select" size="default" clearable placeholder="风险等级">
                <el-option label="Critical" value="critical" />
                <el-option label="High" value="high" />
                <el-option label="Medium" value="medium" />
                <el-option label="Low" value="low" />
              </el-select>
              <el-select v-model="findingsQuery.ownership_class" class="qb-select" size="default" clearable placeholder="归属">
                <el-option label="Direct" value="direct" />
                <el-option label="Shared" value="shared" />
                <el-option label="Unrelated" value="unrelated" />
              </el-select>
              <el-select v-model="findingsQuery.privilege_class" class="qb-select" size="default" clearable placeholder="权限分类">
                <el-option label="Cluster Scoped" value="cluster_scoped" />
                <el-option label="Runtime High" value="runtime_high" />
                <el-option label="Namespace Candidate" value="namespace_only_candidate" />
                <el-option label="Shared Dependency" value="shared_cluster_dependency" />
              </el-select>
            </div>

            <div class="qb-actions">
              <el-button class="qb-btn" size="default" :icon="Switch" :loading="loadingCompare" @click="loadCompare">对比上一次</el-button>
              <el-button class="qb-btn" size="default" :icon="RefreshRight" :loading="loadingFindings" @click="loadFindings">刷新</el-button>
            </div>
          </div>

          <el-alert v-if="compareEmpty" class="detail-error-alert" title="暂无可对比的上一份分析记录" type="info" :closable="false" />

          <div v-if="compareData" class="compare-grid">
            <div class="mini-stat"><span>新增结论</span><strong>{{ compareData.summary.added_count }}</strong></div>
            <div class="mini-stat"><span>移除结论</span><strong>{{ compareData.summary.removed_count }}</strong></div>
            <div class="mini-stat"><span>变化结论</span><strong>{{ compareData.summary.changed_count }}</strong></div>
            <div class="compare-baseline">基线：{{ compareData.baseline_label }}</div>
          </div>

          <el-tabs v-if="compareData" class="compare-tabs">
            <el-tab-pane :label="`新增 (${compareData.summary.added_count})`">
              <el-table :data="compareData.added" stripe border size="small">
                <el-table-column prop="kind" label="类型" width="180" show-overflow-tooltip />
                <el-table-column prop="namespace" label="命名空间" width="160" show-overflow-tooltip />
                <el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip />
                <el-table-column prop="summary" label="摘要" min-width="280" show-overflow-tooltip />
              </el-table>
            </el-tab-pane>
            <el-tab-pane :label="`移除 (${compareData.summary.removed_count})`">
              <el-table :data="compareData.removed" stripe border size="small">
                <el-table-column prop="kind" label="类型" width="180" show-overflow-tooltip />
                <el-table-column prop="namespace" label="命名空间" width="160" show-overflow-tooltip />
                <el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip />
                <el-table-column prop="summary" label="摘要" min-width="280" show-overflow-tooltip />
              </el-table>
            </el-tab-pane>
            <el-tab-pane :label="`变化 (${compareData.summary.changed_count})`">
              <el-table :data="compareData.changed" stripe border size="small">
                <el-table-column label="类型" width="180" show-overflow-tooltip>
                  <template #default="{ row }">{{ row.current?.kind || row.baseline?.kind || '-' }}</template>
                </el-table-column>
                <el-table-column label="命名空间" width="160" show-overflow-tooltip>
                  <template #default="{ row }">{{ row.current?.namespace || row.baseline?.namespace || '-' }}</template>
                </el-table-column>
                <el-table-column label="名称" min-width="180" show-overflow-tooltip>
                  <template #default="{ row }">{{ row.current?.name || row.baseline?.name || '-' }}</template>
                </el-table-column>
                <el-table-column label="当前 / 基线" min-width="320" show-overflow-tooltip>
                  <template #default="{ row }">{{ row.current?.summary || '-' }} / {{ row.baseline?.summary || '-' }}</template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
          </el-tabs>

          <el-table :data="findings" stripe border size="small" v-loading="loadingFindings">
            <el-table-column prop="risk_level" label="风险" width="110">
              <template #default="{ row }"><el-tag :type="riskTagType(row.risk_level)">{{ riskText(row.risk_level) }}</el-tag></template>
            </el-table-column>
            <el-table-column prop="ownership_class" label="归属" width="110">
              <template #default="{ row }">{{ ownershipText(row.ownership_class) }}</template>
            </el-table-column>
            <el-table-column prop="kind" label="类型" width="150" show-overflow-tooltip />
            <el-table-column prop="namespace" label="命名空间" width="150" show-overflow-tooltip />
            <el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip />
            <el-table-column prop="summary" label="摘要" min-width="320" show-overflow-tooltip />
            <el-table-column label="操作" width="100" align="center">
              <template #default="{ row }">
                <el-button link type="primary" @click="openFinding(row)">详情</el-button>
                <el-button link type="success" @click="navigateFinding(row)">跳转</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="pager">
            <el-pagination
              v-model:current-page="findingsPage"
              v-model:page-size="findingsPageSize"
              background
              layout="total, sizes, prev, pager, next, jumper"
              :total="findingsTotal"
              :page-sizes="[10, 20, 50, 100]"
              @current-change="loadFindings"
              @size-change="loadFindings"
            />
          </div>
        </template>
      </div>
    </el-drawer>

    <el-drawer v-model="findingVisible" size="48%" title="Finding 详情" append-to-body>
      <div v-if="activeFinding" class="finding-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="风险"><el-tag :type="riskTagType(activeFinding.risk_level)">{{ riskText(activeFinding.risk_level) }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="归属">{{ ownershipText(activeFinding.ownership_class) }}</el-descriptions-item>
          <el-descriptions-item label="权限分类">{{ privilegeText(activeFinding.privilege_class) }}</el-descriptions-item>
          <el-descriptions-item label="资源">{{ [activeFinding.kind, activeFinding.namespace, activeFinding.name].filter(Boolean).join(' / ') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="摘要">{{ activeFinding.summary || '-' }}</el-descriptions-item>
          <el-descriptions-item v-if="activeFinding.detail?.service_account_name" label="ServiceAccount">
            <code>{{ activeFinding.detail.service_account_name }}</code>
          </el-descriptions-item>
        </el-descriptions>

        <!-- 原因码 -->
        <template v-if="activeFinding.detail?.reason_codes?.length">
          <div class="finding-section-title">原因码</div>
          <div class="finding-tags">
            <el-tag v-for="code in activeFinding.detail.reason_codes" :key="code" size="small" type="warning">{{ code }}</el-tag>
          </div>
        </template>

        <!-- 证据链 -->
        <template v-if="activeFinding.detail?.evidence_chain?.length">
          <div class="finding-section-title">证据链</div>
          <ol class="finding-evidence">
            <li v-for="(ev, i) in activeFinding.detail.evidence_chain" :key="i">
              <el-tag size="small" class="ev-type-tag">{{ ev.type }}</el-tag>
              <span class="ev-summary">{{ ev.summary }}</span>
            </li>
          </ol>
        </template>

        <!-- 建议操作 -->
        <template v-if="activeFinding.detail?.recommended_actions?.length">
          <div class="finding-section-title">建议操作</div>
          <ol class="finding-actions">
            <li v-for="(act, i) in activeFinding.detail.recommended_actions" :key="i">{{ act }}</li>
          </ol>
        </template>

        <div class="detail-actions">
          <el-button type="primary" plain @click="navigateFinding(activeFinding)">跳转到资源页</el-button>
        </div>

        <!-- 原始详情（折叠） -->
        <el-collapse class="finding-raw-collapse">
          <el-collapse-item title="原始详情 JSON">
            <CodeMirrorViewer :text="stringify(activeFinding.detail)" language="json" height="280px" />
          </el-collapse-item>
        </el-collapse>
      </div>
    </el-drawer>

    <el-drawer v-model="logsVisible" size="56%" title="任务日志" append-to-body>
      <div class="task-logs-shell">
        <div class="srv-query-bar detail-filters">
          <div class="qb-actions">
            <el-tag :type="statusTagType(logsStatus as any)">{{ statusText(logsStatus as any) }}</el-tag>
            <el-button class="qb-btn" size="default" :icon="RefreshRight" :loading="loadingLogs" @click="loadLogs(false)">刷新</el-button>
            <el-button v-if="logsCanCancel && activeLogsAuditId" class="qb-btn" type="danger" plain :icon="CloseBold" :loading="cancelingAuditId === activeLogsAuditId" @click="cancelAuditById(activeLogsAuditId)">取消任务</el-button>
          </div>
        </div>
        <CodeMirrorViewer :text="logsText" language="text" height="70vh" />
      </div>
    </el-drawer>

    <!-- 最小权限 RBAC 可视化配置 -->
    <RBACMatrixDrawer
      v-model="rbacMatrixVisible"
      :cluster-id="clusterId"
      :namespaces="namespaces"
      :initial-namespaces="rbacInitialNamespaces"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { CloseBold, Connection, Document, Plus, RefreshRight, Search, Switch, View } from '@element-plus/icons-vue'
import EmptyState from '@/shared/components/EmptyState.vue'
import EnhancedTable from '@/shared/components/EnhancedTable.vue'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import CodeMirrorViewer from '@/shared/components/CodeMirrorViewer.vue'
import RBACMatrixDrawer from './RBACMatrixDrawer.vue'
import * as permissionAuditApi from '@/features/k8s/api/permissionAudit'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'

const props = defineProps<{
  clusterId: number
  namespaces: string[]
}>()

const emit = defineEmits<{
  (e: 'navigate-resource', payload: { kind?: string; namespace?: string; name?: string }): void
}>()

const loadingSummary = ref(false)
const loadingList = ref(false)
const loadingDetail = ref(false)
const loadingFindings = ref(false)
const loadingLogs = ref(false)
const loadingCompare = ref(false)
const creating = ref(false)
const creatingAdhoc = ref(false)
const cancelingAuditId = ref<number | null>(null)

const latestAudit = ref<permissionAuditApi.PermissionAuditDetail | null>(null)
const list = ref<permissionAuditApi.PermissionAuditListItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const query = reactive<{
  keyword: string
  status?: permissionAuditApi.PermissionAuditStatus
  risk_level?: permissionAuditApi.PermissionAuditRiskLevel
}>({
  keyword: '',
  status: undefined,
  risk_level: undefined
})

const sortBy = ref<string | undefined>('created_at')
const order = ref<'asc' | 'desc' | undefined>('desc')

const columns: EnhancedColumn[] = [
  { key: 'display_name', label: '分析任务', prop: 'display_name', minWidth: 220, sortable: 'custom', defaultVisible: true },
  { key: 'status', label: '状态', prop: 'status', width: 120, sortable: 'custom', defaultVisible: true },
  { key: 'source_type', label: '来源', prop: 'source_type', width: 140, defaultVisible: true },
  { key: 'summary', label: '结果摘要', minWidth: 420, defaultVisible: true },
  { key: 'created_at', label: '创建时间', prop: 'created_at', minWidth: 180, sortable: 'custom', defaultVisible: true },
  { key: 'actions', label: '操作', width: 90, align: 'center', headerAlign: 'center', disableToggle: true, overflowTooltip: false, defaultVisible: true }
]

const createVisible = ref(false)
const resourceAllowlistOptions = [
  { label: 'Deployments', value: 'deployments' },
  { label: 'StatefulSets', value: 'statefulsets' },
  { label: 'DaemonSets', value: 'daemonsets' },
  { label: 'Pods', value: 'pods' },
  { label: 'Services', value: 'services' },
  { label: 'Ingresses', value: 'ingresses' },
  { label: 'ConfigMaps', value: 'configmaps' },
  { label: 'Secrets', value: 'secrets' },
  { label: 'ServiceAccounts', value: 'serviceaccounts' },
  { label: 'Roles', value: 'roles' },
  { label: 'ClusterRoles', value: 'clusterroles' },
  { label: 'RoleBindings', value: 'rolebindings' },
  { label: 'ClusterRoleBindings', value: 'clusterrolebindings' },
  { label: 'PVs', value: 'persistentvolumes' },
  { label: 'PVCs', value: 'persistentvolumeclaims' },
  { label: 'StorageClasses', value: 'storageclasses' },
  { label: 'Jobs', value: 'jobs' },
  { label: 'CronJobs', value: 'cronjobs' },
  { label: 'PDBs', value: 'poddisruptionbudgets' },
  { label: 'HPAs', value: 'horizontalpodautoscalers' },
  { label: 'Namespaces', value: 'namespaces' },
  { label: 'Nodes', value: 'nodes' }
] as const

const createForm = reactive<permissionAuditApi.PermissionAuditCreateRequest>({
  mode: 'full',
  include_runtime_rbac: true,
  include_platform_mapping: true,
  include_ownership_detection: true,
  namespaces: [],
  label_selector: '',
  resource_allowlist: []
})

const adhocVisible = ref(false)
const adhocForm = reactive<{ display_name: string; kubeconfig: string } & permissionAuditApi.PermissionAuditCreateRequest>({
  display_name: '',
  kubeconfig: '',
  mode: 'full',
  include_runtime_rbac: true,
  include_platform_mapping: false,
  include_ownership_detection: true,
  namespaces: [],
  label_selector: '',
  resource_allowlist: []
})

const detailVisible = ref(false)
const detail = ref<permissionAuditApi.PermissionAuditDetail | null>(null)
const compareData = ref<permissionAuditApi.PermissionAuditCompareResult | null>(null)
const compareEmpty = ref(false)
const findings = ref<permissionAuditApi.PermissionAuditFindingItem[]>([])
const findingsTotal = ref(0)
const findingsPage = ref(1)
const findingsPageSize = ref(20)
const findingsQuery = reactive<{
  keyword: string
  finding_type?: string
  risk_level?: permissionAuditApi.PermissionAuditRiskLevel
  ownership_class?: permissionAuditApi.PermissionAuditOwnershipClass
  privilege_class?: permissionAuditApi.PermissionAuditPrivilegeClass
}>({
  keyword: '',
  finding_type: undefined,
  risk_level: undefined,
  ownership_class: undefined,
  privilege_class: undefined
})

const findingVisible = ref(false)
const activeFinding = ref<permissionAuditApi.PermissionAuditFindingItem | null>(null)
const detailTitle = computed(() => detail.value ? `权限分析 #${detail.value.id}` : '权限分析详情')
const logsVisible = ref(false)
const activeLogsAuditId = ref<number | null>(null)
const logsLines = ref<string[]>([])
const logsStatus = ref<string>('')
const logsCanCancel = ref(false)
const logsText = computed(() => logsLines.value.length > 0 ? logsLines.value.join('\n') : '暂无日志')

let pollTimer: number | null = null

function clearPollTimer() {
  if (pollTimer != null) {
    window.clearTimeout(pollTimer)
    pollTimer = null
  }
}

function needsPolling(status?: permissionAuditApi.PermissionAuditStatus | null) {
  return status === 'pending' || status === 'running'
}

function schedulePolling() {
  clearPollTimer()
  if (!needsPolling(latestAudit.value?.status) && !needsPolling(detail.value?.status)) return
  pollTimer = window.setTimeout(async () => {
    await refreshAll(false)
    if (detailVisible.value && detail.value?.id) {
      const prevStatus = detail.value.status
      await openDetail(detail.value.id, false)
      // 分析刚完成时主动刷新 findings
      if (prevStatus !== detail.value?.status && detail.value?.status === 'success') {
        findingsPage.value = 1
        await loadFindings()
      }
    }
  }, 5000)
}

watch([latestAudit, detailVisible, detail], () => {
  schedulePolling()
}, { deep: true })

watch(() => props.clusterId, async (value) => {
  if (!value) return
  page.value = 1
  await refreshAll(false)
}, { immediate: true })

onMounted(() => {
  schedulePolling()
})

onBeforeUnmount(() => {
  clearPollTimer()
})

async function refreshAll(resetPage = false) {
  if (resetPage) page.value = 1
  await Promise.all([loadLatest(), loadList()])
}

async function loadLatest() {
  if (!props.clusterId) return
  loadingSummary.value = true
  try {
    latestAudit.value = await permissionAuditApi.getLatestClusterPermissionAudit(props.clusterId)
  } catch (e) {
    const err = e as ApiError
    if (err?.code === 2001 || err?.message?.includes('不存在')) {
      latestAudit.value = null
      return
    }
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    loadingSummary.value = false
  }
}

async function loadList() {
  if (!props.clusterId) return
  loadingList.value = true
  try {
    const data = await permissionAuditApi.listPermissionAudits({
      page: page.value,
      page_size: pageSize.value,
      cluster_id: props.clusterId,
      status: query.status,
      risk_level: query.risk_level,
      keyword: query.keyword.trim() || undefined,
      sort_by: sortBy.value,
      order: order.value
    })
    list.value = data.list ?? []
    total.value = Number(data.total ?? 0)
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    loadingList.value = false
  }
}

function onSortChange(payload: { prop?: string; order?: 'ascending' | 'descending' | null }) {
  sortBy.value = payload?.prop || 'created_at'
  order.value = payload?.order === 'ascending' ? 'asc' : 'desc'
  void loadList()
}

function openCreateDialog() {
  createForm.mode = 'full'
  createForm.include_runtime_rbac = true
  createForm.include_platform_mapping = true
  createForm.include_ownership_detection = true
  createForm.namespaces = []
  createForm.label_selector = ''
  createForm.resource_allowlist = []
  createVisible.value = true
}

function openAdhocDialog() {
  adhocForm.display_name = props.clusterId ? `cluster-${props.clusterId}-adhoc` : 'adhoc-audit'
  adhocForm.kubeconfig = ''
  adhocForm.mode = 'full'
  adhocForm.include_runtime_rbac = true
  adhocForm.include_platform_mapping = false
  adhocForm.include_ownership_detection = true
  adhocForm.namespaces = []
  adhocForm.label_selector = ''
  adhocForm.resource_allowlist = []
  adhocVisible.value = true
}

async function submitCreate() {
  if (!props.clusterId) return
  creating.value = true
  try {
    const data = await permissionAuditApi.createClusterPermissionAudit(props.clusterId, {
      mode: 'full',
      include_runtime_rbac: createForm.include_runtime_rbac !== false,
      include_platform_mapping: createForm.include_platform_mapping !== false,
      include_ownership_detection: createForm.include_ownership_detection !== false,
      namespaces: Array.isArray(createForm.namespaces) && createForm.namespaces.length > 0 ? createForm.namespaces : undefined,
      label_selector: createForm.label_selector?.trim() || undefined,
      resource_allowlist: Array.isArray(createForm.resource_allowlist) && createForm.resource_allowlist.length > 0 ? [...createForm.resource_allowlist] : undefined
    })
    createVisible.value = false
    notifySuccess(`权限分析任务已创建：${data.task_id}`)
    await refreshAll(true)
    await openDetail(data.audit_id)
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    creating.value = false
  }
}

async function submitAdhoc() {
  if (!adhocForm.kubeconfig.trim()) {
    notifyError('请先粘贴 kubeconfig')
    return
  }
  creatingAdhoc.value = true
  try {
    const data = await permissionAuditApi.createAdhocPermissionAudit({
      display_name: adhocForm.display_name.trim() || 'adhoc-audit',
      kubeconfig: adhocForm.kubeconfig,
      mode: 'full',
      include_runtime_rbac: adhocForm.include_runtime_rbac !== false,
      include_platform_mapping: adhocForm.include_platform_mapping === true,
      include_ownership_detection: adhocForm.include_ownership_detection !== false,
      namespaces: Array.isArray(adhocForm.namespaces) && adhocForm.namespaces.length > 0 ? adhocForm.namespaces : undefined,
      label_selector: adhocForm.label_selector?.trim() || undefined,
      resource_allowlist: Array.isArray(adhocForm.resource_allowlist) && adhocForm.resource_allowlist.length > 0 ? [...adhocForm.resource_allowlist] : undefined
    })
    adhocVisible.value = false
    notifySuccess(`临时权限分析任务已创建：${data.task_id}`)
    await refreshAll(true)
    await openDetail(data.audit_id)
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    creatingAdhoc.value = false
  }
}

async function openDetail(id: number, visible = true) {
  loadingDetail.value = true
  if (visible) detailVisible.value = true
  try {
    detail.value = await permissionAuditApi.getPermissionAudit(id)
    compareData.value = null
    compareEmpty.value = false
    findingsPage.value = 1
    await loadFindings()
    await loadCompare()
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    loadingDetail.value = false
  }
}

async function loadFindings() {
  if (!detail.value?.id) return
  loadingFindings.value = true
  try {
    const data = await permissionAuditApi.listPermissionAuditFindings(detail.value.id, {
      page: findingsPage.value,
      page_size: findingsPageSize.value,
      keyword: findingsQuery.keyword.trim() || undefined,
      finding_type: findingsQuery.finding_type,
      risk_level: findingsQuery.risk_level,
      ownership_class: findingsQuery.ownership_class,
      privilege_class: findingsQuery.privilege_class,
      sort_by: 'id',
      order: 'desc'
    })
    findings.value = data.list ?? []
    findingsTotal.value = Number(data.total ?? 0)
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    loadingFindings.value = false
  }
}

async function loadCompare() {
  if (!detail.value?.id) return
  loadingCompare.value = true
  compareData.value = null
  compareEmpty.value = false
  try {
    compareData.value = await permissionAuditApi.comparePermissionAudit(detail.value.id)
  } catch (e) {
    const err = e as ApiError
    if (err?.code === 2001 || err?.message?.includes('不存在')) {
      compareEmpty.value = true
      return
    }
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    loadingCompare.value = false
  }
}

function canCancelAudit(row?: { status?: permissionAuditApi.PermissionAuditStatus | null } | null) {
  return row?.status === 'pending' || row?.status === 'running'
}

async function loadLogs(visible = false) {
  if (!activeLogsAuditId.value) return
  if (visible) logsVisible.value = true
  loadingLogs.value = true
  try {
    const data = await permissionAuditApi.getPermissionAuditLogs(activeLogsAuditId.value, { offset: 0, limit: 500 })
    logsLines.value = data.lines ?? []
    logsStatus.value = data.status || ''
    logsCanCancel.value = data.can_cancel === true
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    loadingLogs.value = false
  }
}

async function openLogs(row: { id: number }) {
  activeLogsAuditId.value = row.id
  await loadLogs(true)
}

async function cancelAudit(row: { id: number }) {
  await cancelAuditById(row.id)
}

async function cancelAuditById(id: number) {
  cancelingAuditId.value = id
  try {
    await permissionAuditApi.cancelPermissionAudit(id)
    notifySuccess('任务已取消')
    await refreshAll(false)
    if (detail.value?.id === id) {
      await openDetail(id, false)
    }
    if (activeLogsAuditId.value === id) {
      await loadLogs(false)
    }
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    cancelingAuditId.value = null
  }
}

function openFinding(row: permissionAuditApi.PermissionAuditFindingItem) {
  activeFinding.value = row
  findingVisible.value = true
}

function navigateFinding(row?: permissionAuditApi.PermissionAuditFindingItem | null) {
  if (!row) return
  emit('navigate-resource', { kind: row.kind, namespace: row.namespace, name: row.name })
}

function resetDetailState() {
  detail.value = null
  compareData.value = null
  compareEmpty.value = false
  findings.value = []
  findingsTotal.value = 0
  activeFinding.value = null
  findingVisible.value = false
}

watch(logsVisible, (value) => {
  if (value) return
  activeLogsAuditId.value = null
  logsLines.value = []
  logsStatus.value = ''
  logsCanCancel.value = false
})

function numberText(value?: number) {
  const n = Number(value ?? 0)
  return Number.isFinite(n) ? String(n) : '0'
}

function formatTime(value?: string) {
  if (!value) return '-'
  const time = new Date(value).getTime()
  if (!Number.isFinite(time)) return value
  return new Date(time).toLocaleString()
}

function statusTagType(status?: permissionAuditApi.PermissionAuditStatus | null): 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'success') return 'success'
  if (status === 'running' || status === 'pending') return 'warning'
  if (status === 'failed') return 'danger'
  return 'info'
}

function statusText(status?: permissionAuditApi.PermissionAuditStatus | null) {
  if (status === 'pending') return '待执行'
  if (status === 'running') return '运行中'
  if (status === 'success') return '成功'
  if (status === 'failed') return '失败'
  if (status === 'incomplete') return '部分完成'
  if (status === 'canceled') return '已取消'
  return '暂无记录'
}

function auditTitle(row?: Partial<permissionAuditApi.PermissionAuditListItem> | null) {
  const title = String(row?.display_name ?? '').trim()
  if (title) return title
  if (row?.source_type === 'adhoc_kubeconfig') return '临时权限分析'
  return '权限分析任务'
}

function sourceTagType(source?: permissionAuditApi.PermissionAuditSourceType): 'primary' | 'success' | 'warning' | 'info' {
  if (source === 'managed_cluster') return 'primary'
  if (source === 'adhoc_kubeconfig') return 'warning'
  return 'info'
}

function sourceText(source?: permissionAuditApi.PermissionAuditSourceType) {
  if (source === 'managed_cluster') return '已纳管集群'
  if (source === 'adhoc_kubeconfig') return '临时凭据'
  return '-'
}

function riskCount(level: permissionAuditApi.PermissionAuditRiskLevel, row?: { summary?: permissionAuditApi.PermissionAuditSummary; stats?: permissionAuditApi.PermissionAuditStats } | null) {
  return String(row?.summary?.risk?.[level] ?? row?.stats?.risk?.[level] ?? 0)
}

function riskTagType(level?: permissionAuditApi.PermissionAuditRiskLevel): 'danger' | 'warning' | 'success' | 'info' {
  if (level === 'critical' || level === 'high') return 'danger'
  if (level === 'medium') return 'warning'
  if (level === 'low') return 'success'
  return 'info'
}

function riskText(level?: permissionAuditApi.PermissionAuditRiskLevel) {
  if (level === 'critical') return 'Critical'
  if (level === 'high') return 'High'
  if (level === 'medium') return 'Medium'
  if (level === 'low') return 'Low'
  return '-'
}

function ownershipText(value?: permissionAuditApi.PermissionAuditOwnershipClass) {
  if (value === 'direct') return 'Direct'
  if (value === 'shared') return 'Shared'
  if (value === 'unrelated') return 'Unrelated'
  return '-'
}

function privilegeText(value?: permissionAuditApi.PermissionAuditPrivilegeClass) {
  if (value === 'cluster_scoped') return 'Cluster Scoped'
  if (value === 'runtime_high') return 'Runtime High'
  if (value === 'namespace_only_candidate') return 'Namespace Candidate'
  if (value === 'shared_cluster_dependency') return 'Shared Dependency'
  return '-'
}

function stringify(value: unknown) {
  if (value == null) return '{}'
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

// ── 建议 RBAC（可视化权限矩阵）──
const rbacMatrixVisible = ref(false)
const rbacInitialNamespaces = ref<string[]>([])

function openRbacDialog() {
  rbacInitialNamespaces.value = props.namespaces?.length ? [...props.namespaces] : []
  rbacMatrixVisible.value = true
}
</script>

<style scoped>
.permission-audit-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.permission-audit-toolbar {
  margin-bottom: 0;
}

.summary-card,
.audit-history-table {
  width: 100%;
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.compare-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.compare-baseline {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 72px;
  border: 1px dashed var(--el-border-color);
  border-radius: 12px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  padding: 12px;
}

.compare-tabs {
  margin-bottom: 12px;
}

.allowlist-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 8px 12px;
  width: 100%;
}

.summary-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.summary-card-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.summary-card-sub {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.summary-grid {
  display: grid;
  gap: 16px;
}

.summary-kpi-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 12px;
}

.summary-kpi,
.mini-stat {
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  padding: 14px 16px;
  background: linear-gradient(180deg, rgba(248, 250, 252, 0.96) 0%, rgba(255, 255, 255, 0.98) 100%);
}

.summary-kpi-label,
.mini-stat span {
  display: block;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.summary-kpi-value,
.mini-stat strong {
  display: block;
  margin-top: 6px;
  font-size: 24px;
  line-height: 1.1;
  color: var(--el-text-color-primary);
}

.summary-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.summary-risk {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.risk-chip {
  padding: 8px 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.risk-chip--critical,
.risk-chip--high {
  background: rgba(239, 68, 68, 0.12);
  color: #b91c1c;
}

.risk-chip--medium {
  background: rgba(245, 158, 11, 0.14);
  color: #b45309;
}

.risk-chip--low {
  background: rgba(34, 197, 94, 0.12);
  color: #15803d;
}

.summary-inline {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.summary-chip {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: rgba(59, 130, 246, 0.08);
  color: #475569;
  font-size: 12px;
  font-weight: 600;
}

.summary-chip--high {
  background: rgba(249, 115, 22, 0.1);
  color: #c2410c;
}

.summary-chip--critical {
  background: rgba(239, 68, 68, 0.1);
  color: #b91c1c;
}

.audit-title-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.audit-title-main {
  font-size: 13px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  line-height: 1.4;
  word-break: break-word;
}

.audit-title-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.audit-source-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: flex-start;
}

.audit-source-meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.form-tip {
  margin-left: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.default-capability-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.detail-shell {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-summary-row,
.detail-descriptions,
.detail-filters,
.detail-error-alert {
  width: 100%;
}

.detail-error-text {
  white-space: pre-wrap;
  word-break: break-word;
}

.finding-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

@media (max-width: 900px) {
  .summary-card-head {
    flex-direction: column;
    align-items: flex-start;
  }

  .summary-kpi-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .summary-kpi-list {
    grid-template-columns: 1fr;
  }
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}

.finding-section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin: 16px 0 8px;
}

.finding-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.finding-evidence {
  margin: 0;
  padding-left: 20px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
}

.finding-evidence li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  line-height: 1.5;
}

.ev-type-tag {
  flex-shrink: 0;
}

.ev-summary {
  color: var(--el-text-color-regular);
}

.finding-actions {
  margin: 0;
  padding-left: 20px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  line-height: 1.6;
}

.finding-raw-collapse {
  margin-top: 16px;
}

.task-logs-shell {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
