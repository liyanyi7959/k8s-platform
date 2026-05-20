<template>
  <div class="k8s-shell">
    <aside class="k8s-aside">
      <div class="aside-head">
        <div class="aside-title-wrap">
          <button class="aside-back-btn" type="button" title="返回集群管理" @click="router.push('/clusters')">
            <el-icon><ArrowLeft /></el-icon>
          </button>
          <div class="aside-title">{{ clusterTitle }}</div>
        </div>
        <el-button size="small" :icon="RefreshRight" :loading="loadingTree" @click="refreshAll">刷新</el-button>
      </div>

      <el-scrollbar class="aside-body">
        <el-tree
          ref="treeRef"
          :data="tree"
          node-key="id"
          :expand-on-click-node="true"
          :highlight-current="false"
          default-expand-all
          @node-click="onTreeNodeClick"
        >
          <template #default="{ data }">
            <span :class="[
              'tree-node',
              data.kind === 'folder' ? 'tree-node--folder' : 'tree-node--leaf',
              current?.id === data.id ? 'tree-node--active' : ''
            ]">
              <img v-if="data.iconUrl" :src="data.iconUrl" :class="['tree-icon-img', data.kind === 'folder' ? 'tree-icon-img--folder' : 'tree-icon-img--leaf']" alt="">
              <el-icon v-else :class="['tree-icon', data.kind === 'folder' ? 'tree-icon--folder' : 'tree-icon--leaf']"><component :is="data.kind === 'folder' ? Collection : Document" /></el-icon>
              <span class="tree-label">{{ data.label }}</span>
            </span>
          </template>
        </el-tree>
        <EmptyState v-if="!tree.length && !loadingTree" type="no-data" description="暂无资源目录" />
      </el-scrollbar>
    </aside>

    <main class="k8s-main">
      <el-card class="page-card">
        <div v-if="current?.resource !== 'permissionaudits' && current?.resource !== 'topology'" class="srv-query-bar">
          <!-- 资源标识 -->
          <div v-if="current?.resource" class="qb-resource-meta">
            <span :class="['resource-badge', resourceBadgeClass]">{{ toolbarResourceText }}</span>
          </div>

          <!-- 筛选区 -->
          <div v-if="showNamespaceSelect" class="qb-filters">
            <el-select
              :model-value="namespace"
              placeholder="名称空间"
              class="qb-select qb-select--ns"
              size="default"
              :loading="loadingTree"
              filterable
              multiple
              collapse-tags
              collapse-tags-tooltip
              :max-collapse-tags="1"
              @update:model-value="onNamespaceSelectChange"
            >
              <el-option :label="'全部'" :value="ALL_NAMESPACE" />
              <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
            </el-select>
          </div>

          <div v-if="current?.resource === 'workloads'" class="qb-filters">
            <el-input v-model="workloadLabelSelector" class="qb-input qb-input--label" size="default" placeholder="标签 app=nginx" clearable />
          </div>

          <div v-if="current?.resource !== 'dashboard'" class="qb-search">
            <el-icon class="qb-search-icon"><Search /></el-icon>
            <el-input v-model="keywordInput" class="qb-keyword" size="default" placeholder="搜索关键字…" clearable />
          </div>

          <!-- 操作区 -->
          <div class="qb-actions">
            <el-button v-if="current?.resource === 'namespaces' && canWriteK8s" class="qb-btn qb-btn--primary" type="primary" size="default" :disabled="!clusterId" :icon="Plus" @click="openCreateNamespace">创建</el-button>
            <el-button class="qb-btn" size="default" :disabled="!clusterId || !current" :loading="current?.resource === 'dashboard' ? loadingDashboard : loadingList" :icon="RefreshRight" @click="loadCurrent" />
            <el-dropdown
              v-if="showColumnPickerBtn"
              trigger="click"
              :hide-on-click="false"
              popper-class="k8s-columns-popper"
              @visible-change="onColumnsDropdownVisibleChange"
            >
              <el-button size="small" :icon="Setting" @click="syncActiveColumns">列</el-button>
              <template #dropdown>
                <div class="columns-panel">
                  <div v-if="activeTableColumns.length === 0" class="columns-panel-empty">暂无可筛选字段</div>
                  <template v-else>
                    <div class="columns-panel-actions">
                      <el-button size="small" :icon="Check" @click="selectAllActiveColumns">全选</el-button>
                      <el-button size="small" :icon="CircleClose" @click="clearAllActiveColumns">全不选</el-button>
                      <el-button size="small" :icon="RefreshRight" @click="resetActiveColumns">重置</el-button>
                    </div>
                    <el-divider style="margin: 10px 0" />
                    <div class="columns-panel-list">
                      <el-checkbox
                        v-for="c in activeTableColumns"
                        :key="c.key"
                        v-model="activeColumnVisible[c.key]"
                        :disabled="c.disableToggle === true"
                        @change="onActiveColumnVisibleChange"
                      >
                        {{ c.label }}
                      </el-checkbox>
                    </div>
                  </template>
                </div>
              </template>
            </el-dropdown>
          </div>
        </div><!-- /srv-query-bar -->

        <ClusterDashboard
          v-if="current?.resource === 'dashboard'"
          :cluster-id="clusterId"
          :cluster-overview="clusterOverview"
          :loading-dashboard="loadingDashboard"
          :dashboard-namespaces-total="dashboardNamespacesTotal"
          :dashboard-events="dashboardEvents"
          :dashboard-last-updated-at="dashboardLastUpdatedAt"
          :dashboard-cert-risk-loading="dashboardCertRiskLoading"
          :dashboard-cert-risk-unavailable="dashboardCertRiskUnavailable"
          :dashboard-alert-counts="dashboardAlertCounts"
          :dashboard-health-score="dashboardHealthScore"
          :dashboard-auto-refresh="dashboardAutoRefresh"
          :dashboard-auto-refresh-sec="dashboardAutoRefreshSec"
          :dashboard-cpu-text="dashboardCpuText"
          :dashboard-memory-text="dashboardMemoryText"
          :dashboard-ready-text="dashboardReadyText"
          :dashboard-enabled-widgets="dashboardEnabledWidgets"
          @load-dashboard="loadDashboard"
          @navigate-resource="onDashboardNavigate"
          @navigate-object="handlePermissionAuditNavigation"
          @update:auto-refresh="dashboardAutoRefresh = $event"
          @update:auto-refresh-sec="dashboardAutoRefreshSec = $event"
        />

        <ResourceTopologyView
          v-else-if="current?.resource === 'topology'"
          :fixed-cluster-id="clusterId"
          embedded
        />

        <PermissionAuditsPanel
          v-else-if="current?.resource === 'permissionaudits'"
          :cluster-id="clusterId"
          :namespaces="namespaces"
          @navigate-resource="handlePermissionAuditNavigation"
        />

        <NamespacesPanel
          v-else-if="current?.resource === 'namespaces'"
          ref="namespacesPanelRef"
          :cluster-id="clusterId"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:namespaces`"
          :show-tools="showListTableTools"
          :open-yaml="openNamespaceYaml"
          :on-deleted="refreshAll"
          @sort-change="onSortChange"
        />

        <NodesPanel
          v-else-if="current?.resource === 'nodes'"
          ref="nodesPanelRef"
          :cluster-id="clusterId"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:nodes`"
          :show-tools="showListTableTools"
          :get-ready="getReady"
          :open-node-detail="openNodeDetail"
          :open-node-yaml="openNodeYaml"
          @sort-change="onSortChange"
          @refresh="loadCurrent"
        />

        <PodsPanel
          v-else-if="current?.resource === 'pods'"
          ref="podsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:pods`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :pod-phase-summary="podPhaseSummary"
          :active-phase-filter="podPhaseQuickFilter"
          :get-warning-event-count="getPodWarningEventCount"
          :get-pod-row-key="getPodRowKey"
          :get-pod-phase-tag-type="getPodPhaseTagType"
          :get-pod-phase-text="getPodPhaseText"
          :get-pod-ready-text="getPodReadyText"
          :get-pod-restarts="getPodRestarts"
          :get-pod-age="getPodAge"
          :open-pod-detail="openPodDetail"
          :open-pod-logs="openPodLogs"
          :open-pod-exec="openPodExec"
          :open-pod-yaml="openPodYaml"
          :delete-pod-row="deletePodRow"
          :bulk-delete-pods="bulkDeletePods"
          @sort-change="onSortChange"
          @selection-change="onPodSelectionChange"
          @phase-filter-change="onPodPhaseQuickFilterChange"
        />

        <WorkloadsPanel
          v-else-if="current?.resource === 'workloads'"
          ref="workloadsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:workloads:${workloadKind}`"
          :show-tools="showListTableTools"
          :workload-kind="workloadKind"
          :cluster-id="clusterId"
          :get-ready-text="getReadyText"
          :get-warning-event-count="getWorkloadWarningEventCount"
          :open-deployment-detail="openDeploymentDetail"
          :open-edit-deployment="openEditDeployment"
          :open-stateful-set-detail="openStatefulSetDetail"
          :open-edit-stateful-set="openEditStatefulSet"
          :open-daemon-set-detail="openDaemonSetDetail"
          :open-edit-daemon-set="openEditDaemonSet"
          :open-scale="openScale"
          :restart-workload-row="restartWorkloadRow"
          :restart-selected-workloads="restartSelectedWorkloads"
          :update-workload-image="updateWorkloadImage"
          :toggle-workload-paused="toggleWorkloadPaused"
          :open-workload-rollout="openWorkloadRollout"
          :open-workload-yaml="openWorkloadYaml"
          :delete-workload-row="deleteWorkloadRow"
          @sort-change="onSortChange"
        />

        <NamespacedGovernancePanel
          v-else-if="current?.resource === 'replicasets'"
          ref="replicaSetsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:replicasets`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :get-summary="getReplicaSetSummary"
          :open-yaml="openReplicaSetYaml"
          :open-edit="openEditReplicaSet"
          :delete-row="deleteReplicaSetRow"
          @sort-change="onSortChange"
        />

        <HPAsPanel
          v-else-if="current?.resource === 'hpas'"
          ref="hpasPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:hpas`"
          :show-tools="showListTableTools"
          :open-h-p-a-yaml="openHPAYaml"
          :open-edit-h-p-a="openEditHPA"
          :delete-h-p-a-row="deleteHPARow"
          @sort-change="onSortChange"
        />

        <PDBsPanel
          v-else-if="current?.resource === 'pdbs'"
          ref="pdbsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:pdbs`"
          :show-tools="showListTableTools"
          :open-pdb-yaml="openPdbYaml"
          :open-edit-pdb="openEditPdb"
          :delete-pdb-row="deletePdbRow"
          @sort-change="onSortChange"
        />

        <ServicesPanel
          v-else-if="current?.resource === 'services'"
          ref="servicesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:services`"
          :show-tools="showListTableTools"
          :format-ports="formatPorts"
          :format-selector="formatSelector"
          :open-service-detail="openServiceDetail"
          :open-service-yaml="openServiceYaml"
          :open-edit-service="openEditService"
          :delete-service-row="deleteServiceRow"
          @sort-change="onSortChange"
        />

        <NamespacedGovernancePanel
          v-else-if="current?.resource === 'networkpolicies'"
          ref="networkPoliciesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:networkpolicies`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :get-summary="getNetworkPolicySummary"
          :open-yaml="openNetworkPolicyYaml"
          :open-edit="openEditNetworkPolicy"
          :delete-row="deleteNetworkPolicyRow"
          @sort-change="onSortChange"
        />

        <NamespacedGovernancePanel
          v-else-if="current?.resource === 'endpoints'"
          ref="endpointsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:endpoints`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :get-summary="getEndpointsSummary"
          :open-yaml="openEndpointsYaml"
          :open-edit="openEditEndpoints"
          :delete-row="deleteEndpointsRow"
          @sort-change="onSortChange"
        />

        <NamespacedGovernancePanel
          v-else-if="current?.resource === 'endpointslices'"
          ref="endpointSlicesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:endpointslices`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :get-summary="getEndpointSliceSummary"
          :open-yaml="openEndpointSliceYaml"
          :open-edit="openEditEndpointSlice"
          :delete-row="deleteEndpointSliceRow"
          @sort-change="onSortChange"
        />

        <IngressesPanel
          v-else-if="current?.resource === 'ingresses'"
          ref="ingressesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:ingresses`"
          :show-tools="showListTableTools"
          :get-hosts="getHosts"
          :format-rules="formatRules"
          :open-ingress-detail="openIngressDetail"
          :open-ingress-yaml="openIngressYaml"
          :open-edit-ingress="openEditIngress"
          :delete-ingress-row="deleteIngressRow"
          @sort-change="onSortChange"
        />

        <IngressClassesPanel
          v-else-if="current?.resource === 'ingressclasses'"
          ref="ingressClassesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:ingressclasses`"
          :show-tools="showListTableTools"
          :open-ingress-class-detail="openIngressClassDetail"
          :open-ingress-class-yaml="openIngressClassYaml"
          :open-edit-ingress-class="openEditIngressClass"
          :delete-ingress-class-row="deleteIngressClassRow"
          @sort-change="onSortChange"
        />

        <ConfigMapsPanel
          v-else-if="current?.resource === 'configmaps'"
          ref="configMapsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:configmaps`"
          :show-tools="showListTableTools"
          :get-data-keys="getDataKeys"
          :open-config-map-detail="openConfigMapDetail"
          :open-config-map-yaml="openConfigMapYaml"
          :open-edit-config-map="openEditConfigMap"
          :delete-config-map-row="deleteConfigMapRow"
          @sort-change="onSortChange"
        />

        <SecretsPanel
          v-else-if="current?.resource === 'secrets'"
          ref="secretsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:secrets`"
          :show-tools="showListTableTools"
          :can-reveal-secret="canRevealSecret"
          :get-data-keys="getDataKeys"
          :open-secret-detail="openSecretDetail"
          :open-secret-reveal="openSecretReveal"
          :open-secret-yaml="openSecretYaml"
          :open-edit-secret="openEditSecret"
          :delete-secret-row="deleteSecretRow"
          @sort-change="onSortChange"
        />

        <ServiceAccountsPanel
          v-else-if="current?.resource === 'serviceaccounts'"
          ref="serviceAccountsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:serviceaccounts`"
          :show-tools="showListTableTools"
          :open-service-account-yaml="openServiceAccountYaml"
          :open-edit-service-account="openEditServiceAccount"
          :delete-service-account-row="deleteServiceAccountRow"
          @sort-change="onSortChange"
        />

        <RolesPanel
          v-else-if="current?.resource === 'roles'"
          ref="rolesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:roles`"
          :show-tools="showListTableTools"
          :can-write="canWriteRbac"
          :open-role-yaml="openRoleYaml"
          :open-role-edit="openEditRole"
          :delete-role-row="deleteRoleRow"
          @sort-change="onSortChange"
        />

        <ClusterRolesPanel
          v-else-if="current?.resource === 'clusterroles'"
          ref="clusterRolesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:clusterroles`"
          :show-tools="showListTableTools"
          :can-write="canWriteRbac"
          :open-cluster-role-yaml="openClusterRoleYaml"
          :open-cluster-role-edit="openEditClusterRole"
          :delete-cluster-role-row="deleteClusterRoleRow"
          @sort-change="onSortChange"
        />

        <RoleBindingsPanel
          v-else-if="current?.resource === 'rolebindings'"
          ref="roleBindingsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:rolebindings`"
          :show-tools="showListTableTools"
          :can-write="canWriteRbac"
          :open-role-binding-yaml="openRoleBindingYaml"
          :open-role-binding-edit="openEditRoleBinding"
          :delete-role-binding-row="deleteRoleBindingRow"
          @sort-change="onSortChange"
        />

        <ClusterRoleBindingsPanel
          v-else-if="current?.resource === 'clusterrolebindings'"
          ref="clusterRoleBindingsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:clusterrolebindings`"
          :show-tools="showListTableTools"
          :can-write="canWriteRbac"
          :open-cluster-role-binding-yaml="openClusterRoleBindingYaml"
          :open-cluster-role-binding-edit="openEditClusterRoleBinding"
          :delete-cluster-role-binding-row="deleteClusterRoleBindingRow"
          @sort-change="onSortChange"
        />

        <PvcsPanel
          v-else-if="current?.resource === 'pvcs'"
          ref="pvcsPanelRef"
          :cluster-id="clusterId"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:pvcs`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :namespaces="namespaces"
          :default-namespace="defaultCreatePVCNamespace"
          :open-p-v-c-detail="openPVCDetail"
          :open-p-v-c-yaml="openPVCYaml"
          :delete-p-v-c-row="deletePVCRow"
          :on-created="loadCurrent"
          @sort-change="onSortChange"
        />

        <PvsPanel
          v-else-if="current?.resource === 'pvs'"
          ref="pvsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:pvs`"
          :show-tools="showListTableTools"
          :format-claim-ref="formatClaimRef"
          :open-p-v-detail="openPVDetail"
          :open-p-v-yaml="openPVYaml"
          :delete-p-v-row="deletePVRow"
          @sort-change="onSortChange"
        />

        <StorageClassesPanel
          v-else-if="current?.resource === 'storageclasses'"
          ref="storageClassesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:storageclasses`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :open-edit-storage-class="openEditStorageClass"
          :open-storage-class-yaml="openStorageClassYaml"
          :delete-storage-class-row="deleteStorageClassRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'csidrivers'"
          ref="csiDriversPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:csidrivers`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="驱动能力"
          :get-summary="getCSIDriverSummary"
          :open-yaml="openCSIDriverYaml"
          :open-edit="openEditCSIDriver"
          :delete-row="deleteCSIDriverRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'csinodes'"
          ref="csiNodesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:csinodes`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="Node 驱动"
          :get-summary="getCSINodeSummary"
          :open-yaml="openCSINodeYaml"
          :open-edit="openEditCSINode"
          :delete-row="deleteCSINodeRow"
          @sort-change="onSortChange"
        />

        <NamespacedGovernancePanel
          v-else-if="current?.resource === 'csistoragecapacities'"
          ref="csiStorageCapacitiesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:csistoragecapacities`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :get-summary="getCSIStorageCapacitySummary"
          :open-yaml="openCSIStorageCapacityYaml"
          :open-edit="openEditCSIStorageCapacity"
          :delete-row="deleteCSIStorageCapacityRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'volumeattachments'"
          ref="volumeAttachmentsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:volumeattachments`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="节点 / 卷"
          :get-summary="getVolumeAttachmentSummary"
          :open-yaml="openVolumeAttachmentYaml"
          :open-edit="openEditVolumeAttachment"
          :delete-row="deleteVolumeAttachmentRow"
          @sort-change="onSortChange"
        />

        <NamespacedGovernancePanel
          v-else-if="current?.resource === 'volumesnapshots'"
          ref="volumeSnapshotsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:volumesnapshots`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :get-summary="getVolumeSnapshotSummary"
          :open-yaml="openVolumeSnapshotYaml"
          :open-edit="openEditVolumeSnapshot"
          :delete-row="deleteVolumeSnapshotRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'volumesnapshotclasses'"
          ref="volumeSnapshotClassesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:volumesnapshotclasses`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="驱动 / 删除策略"
          :get-summary="getVolumeSnapshotClassSummary"
          :open-yaml="openVolumeSnapshotClassYaml"
          :open-edit="openEditVolumeSnapshotClass"
          :delete-row="deleteVolumeSnapshotClassRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'volumesnapshotcontents'"
          ref="volumeSnapshotContentsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:volumesnapshotcontents`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="句柄 / 绑定"
          :get-summary="getVolumeSnapshotContentSummary"
          :open-yaml="openVolumeSnapshotContentYaml"
          :open-edit="openEditVolumeSnapshotContent"
          :delete-row="deleteVolumeSnapshotContentRow"
          @sort-change="onSortChange"
        />

        <NamespacedGovernancePanel
          v-else-if="current?.resource === 'resourcequotas'"
          ref="resourceQuotasPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:resourcequotas`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          show-resource-quota-usage
          :get-summary="getResourceQuotaSummary"
          :open-yaml="openResourceQuotaYaml"
          :open-edit="openEditResourceQuota"
          :delete-row="deleteResourceQuotaRow"
          @sort-change="onSortChange"
        />

        <NamespacedGovernancePanel
          v-else-if="current?.resource === 'limitranges'"
          ref="limitRangesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:limitranges`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :get-summary="getLimitRangeSummary"
          :open-yaml="openLimitRangeYaml"
          :open-edit="openEditLimitRange"
          :delete-row="deleteLimitRangeRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'customresourcedefinitions'"
          ref="customResourceDefinitionsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:customresourcedefinitions`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="组 / 资源"
          :get-summary="getCustomResourceDefinitionSummary"
          :open-yaml="openCustomResourceDefinitionYaml"
          :open-edit="openEditCustomResourceDefinition"
          :delete-row="deleteCustomResourceDefinitionRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'apiservices'"
          ref="apiServicesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:apiservices`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="版本 / 后端"
          :get-summary="getAPIServiceSummary"
          :open-yaml="openAPIServiceYaml"
          :open-edit="openEditAPIService"
          :delete-row="deleteAPIServiceRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'priorityclasses'"
          ref="priorityClassesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:priorityclasses`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="优先级"
          :get-summary="getPriorityClassSummary"
          :open-yaml="openPriorityClassYaml"
          :open-edit="openEditPriorityClass"
          :delete-row="deletePriorityClassRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'runtimeclasses'"
          ref="runtimeClassesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:runtimeclasses`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="Handler"
          :get-summary="getRuntimeClassSummary"
          :open-yaml="openRuntimeClassYaml"
          :open-edit="openEditRuntimeClass"
          :delete-row="deleteRuntimeClassRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'validatingwebhookconfigurations'"
          ref="validatingWebhookConfigurationsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:validatingwebhookconfigurations`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="Webhook 摘要"
          :get-summary="getWebhookConfigurationSummary"
          :open-yaml="openValidatingWebhookConfigurationYaml"
          :open-edit="openEditValidatingWebhookConfiguration"
          :delete-row="deleteValidatingWebhookConfigurationRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'mutatingwebhookconfigurations'"
          ref="mutatingWebhookConfigurationsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:mutatingwebhookconfigurations`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="Webhook 摘要"
          :get-summary="getWebhookConfigurationSummary"
          :open-yaml="openMutatingWebhookConfigurationYaml"
          :open-edit="openEditMutatingWebhookConfiguration"
          :delete-row="deleteMutatingWebhookConfigurationRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'validatingadmissionpolicies'"
          ref="validatingAdmissionPoliciesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:validatingadmissionpolicies`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="Failure Policy"
          :get-summary="getValidatingAdmissionPolicySummary"
          :open-yaml="openValidatingAdmissionPolicyYaml"
          :open-edit="openEditValidatingAdmissionPolicy"
          :delete-row="deleteValidatingAdmissionPolicyRow"
          @sort-change="onSortChange"
        />

        <ClusterScopedResourcesPanel
          v-else-if="current?.resource === 'validatingadmissionpolicybindings'"
          ref="validatingAdmissionPolicyBindingsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:validatingadmissionpolicybindings`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          summary-label="Policy"
          :get-summary="getValidatingAdmissionPolicyBindingSummary"
          :open-yaml="openValidatingAdmissionPolicyBindingYaml"
          :open-edit="openEditValidatingAdmissionPolicyBinding"
          :delete-row="deleteValidatingAdmissionPolicyBindingRow"
          @sort-change="onSortChange"
        />

        <JobsPanel
          v-else-if="current?.resource === 'jobs'"
          ref="jobsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:jobs`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :completed-count="completedJobsCount"
          :open-job-detail="openJobDetail"
          :open-edit-job="openEditJob"
          :open-job-yaml="openJobYaml"
          :delete-job-row="deleteJobRow"
          :clean-completed-jobs="cleanCompletedJobs"
          @sort-change="onSortChange"
        />

        <CronJobsPanel
          v-else-if="current?.resource === 'cronjobs'"
          ref="cronJobsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:cronjobs`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :open-cron-job-detail="openCronJobDetail"
          :open-edit-cron-job="openEditCronJob"
          :open-cron-job-yaml="openCronJobYaml"
          :delete-cron-job-row="deleteCronJobRow"
          :trigger-cron-job="triggerCronJob"
          :toggle-cron-job-suspend="toggleCronJobSuspend"
          @sort-change="onSortChange"
        />

        <EventsPanel
          v-else-if="current?.resource === 'events'"
          ref="eventsPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:events`"
          :show-tools="showListTableTools"
          :type-filter="eventTypeFilter"
          :reason-filter="eventReasonFilter"
          :type-options="eventTypeOptions"
          :reason-options="eventReasonOptions"
          @sort-change="onSortChange"
          @update:type-filter="onEventTypeFilterChange"
          @update:reason-filter="onEventReasonFilterChange"
        />

        <NamespacedGovernancePanel
          v-else-if="current?.resource === 'leases'"
          ref="leasesPanelRef"
          :data="pagedList"
          :persist-key="`k8s:cluster_manage:v2:${clusterId}:leases`"
          :show-tools="showListTableTools"
          :can-write="canWriteK8s"
          :get-summary="getLeaseSummary"
          :open-yaml="openLeaseYaml"
          :open-edit="openEditLease"
          :delete-row="deleteLeaseRow"
          @sort-change="onSortChange"
        />

        <div v-if="showPager" class="list-pager">
          <el-pagination
            :current-page="page"
            :page-size="pageSize"
            :page-sizes="pageSizeOptions"
            :total="displayedTotal"
            layout="total, sizes, prev, pager, next, jumper"
            @current-change="onPageChange"
            @size-change="onPageSizeChange"
          />
        </div>

        <div v-else-if="!current" class="k8s-welcome">
          <div class="k8s-welcome-icon">
            <el-icon :size="44"><K8sClusterIcon /></el-icon>
          </div>
          <div class="k8s-welcome-title">{{ clusterTitle || 'K8s 集群管理' }}</div>
          <div class="k8s-welcome-desc">从左侧选择一个资源类型开始管理</div>
          <div class="k8s-welcome-hints">
            <div class="k8s-hint-item" @click="quickSelect('namespaces')">
              <el-icon :size="16"><Folder /></el-icon>
              <span>Namespaces</span>
            </div>
            <div class="k8s-hint-item" @click="quickSelect('workloads')">
              <el-icon :size="16"><Box /></el-icon>
              <span>Workloads</span>
            </div>
            <div class="k8s-hint-item" @click="quickSelect('pods')">
              <el-icon :size="16"><Cpu /></el-icon>
              <span>Pods</span>
            </div>
            <div class="k8s-hint-item" @click="quickSelect('nodes')">
              <el-icon :size="16"><Platform /></el-icon>
              <span>Nodes</span>
            </div>
            <div class="k8s-hint-item" @click="quickSelect('services')">
              <el-icon :size="16"><Share /></el-icon>
              <span>Services</span>
            </div>
            <div class="k8s-hint-item" @click="quickSelect('events')">
              <el-icon :size="16"><BellFilled /></el-icon>
              <span>Events</span>
            </div>
          </div>
        </div>
      </el-card>
    </main>
  </div>

  <ClusterManageWorkbenches
    v-if="overlayLoaded.workbenches"
    ref="workbenchesRef"
    :cluster-id="clusterId"
    :editor-theme="editorTheme"
    :editor-theme-effective-dark="editorThemeEffectiveDark"
    @toggle-editor-theme="toggleEditorTheme"
    @namespace-created="refreshAll"
  />

  <ClusterManageDetailsHost
    ref="detailsHostRef"
    :cluster-id="clusterId"
    :cluster-name="clusterName"
    :editor-theme="editorTheme"
    :editor-theme-effective-dark="editorThemeEffectiveDark"
    :list="list"
    @refresh-list="loadCurrent"
    @toggle-editor-theme="toggleEditorTheme"
    @open-related-pod="openRelatedPodDetail"
    @open-topology="openResourceTopology"
    @open-pod-detail="openPodDetailFromExternal"
    @open-job-detail="openJobDetail"
    @open-related="openRelatedFromJob"
    @open-yaml="openYaml"
    @pod-detail-open-logs="onPodDetailOpenLogs"
    @pod-detail-open-exec="onPodDetailOpenExec"
    @pod-detail-open-related="onPodDetailOpenRelated"
    @pod-detail="openPodDetail"
    @pod-log="openPodLogs"
    @pod-exec="openPodExec"
  />

  <ClusterManageResourceEditors
    v-if="overlayLoaded.resourceEditors"
    ref="resourceEditorsRef"
    :cluster-id="clusterId"
    :cluster-name="clusterName"
    :editor-theme="editorTheme"
    :editor-theme-effective-dark="editorThemeEffectiveDark"
    @toggle-editor-theme="toggleEditorTheme"
    @saved="loadCurrent"
  />
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onErrorCaptured, reactive, ref, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import {
  BellFilled,
  Box,
  Check,
  CircleClose,
  Collection,
  CopyDocument,
  Cpu,
  Delete,
  Document,
  Download,
  Edit,
  Expand,
  Fold,
  Folder,
  ArrowLeft,
  Link,
  Monitor,
  Moon,
  Platform,
  Plus,
  RefreshRight,
  Search,
  Setting,
  Share,
  Sunny
} from '@element-plus/icons-vue'
import * as clustersApi from '@/features/clusters/api/clusters'
import * as k8sApi from '@/features/k8s/api/k8s'
import { K8sClusterIcon, PowerSwitchIcon } from '@/shared/icons/appIcons'
import type ClusterManageWorkbenchesComponent from './clusterManage/overlays/ClusterManageWorkbenches.vue'
import type ClusterManageResourceEditorsComponent from './clusterManage/overlays/ClusterManageResourceEditors.vue'
import EmptyState from '@/shared/components/EmptyState.vue'
import { useUserStore } from '@/app/store/user'
import { useClusterManageDeleteActions } from '@/features/k8s/composables/useClusterManageDeleteActions'
import { useClusterManageDashboard } from '@/features/k8s/composables/useClusterManageDashboard'
import { useClusterManageYamlActions } from '@/features/k8s/composables/useClusterManageYamlActions'
import { useClusterManageViewState } from '@/features/k8s/composables/useClusterManageViewState'
import { copyToClipboard, downloadBlob, sanitizeFileName } from '@/shared/utils/text'
import { notifyError, notifySuccess } from '@/shared/utils/notify'
import type { ApiError } from '@/shared/utils/error'
import type { EnhancedColumn } from '@/shared/components/EnhancedTable.vue'
import k8sLogoUrl from '@/assets/images/k8s-official-icon.svg'
import k8sIconCmUrl from '@/assets/images/k8s/cm.svg'
import k8sIconCronJobUrl from '@/assets/images/k8s/cronjob.svg'
import k8sIconDeploymentUrl from '@/assets/images/k8s/deploy.svg'
import k8sIconDaemonSetUrl from '@/assets/images/k8s/ds.svg'
import k8sIconGroupUrl from '@/assets/images/k8s/group.svg'
import k8sIconIngressUrl from '@/assets/images/k8s/ing.svg'
import k8sIconJobUrl from '@/assets/images/k8s/job.svg'
import k8sIconNodeUrl from '@/assets/images/k8s/node.svg'
import k8sIconNamespaceUrl from '@/assets/images/k8s/ns.svg'
import k8sIconPodUrl from '@/assets/images/k8s/pod.svg'
import k8sIconPvUrl from '@/assets/images/k8s/pv.svg'
import k8sIconPvcUrl from '@/assets/images/k8s/pvc.svg'
import k8sIconStorageClassUrl from '@/assets/images/k8s/sc.svg'
import k8sIconSecretUrl from '@/assets/images/k8s/secret.svg'
import k8sIconStatefulSetUrl from '@/assets/images/k8s/sts.svg'
import k8sIconServiceUrl from '@/assets/images/k8s/svc.svg'
import type { K8sLikeObject, ResourceKey, SortOrder, TreeNode, WorkloadKind } from './ClusterManageView.types'
import {
  buildStorageKey,
  buildTree,
  computeNextNamespaceSelection,
  filterTreeByPerms,
  filterTreeByResourceSupport,
  formatAgeMs,
  getCreationAgeMs,
  getCreationAgeText,
  getListRowSearchText,
  getNamespaceFilter,
  getPodRowKey,
  getResourceBadgeClass,
  getRowNamespace,
  getWorkloadAvailable,
  getWorkloadDesired,
  getWorkloadReadyText,
  isNamespacedResource,
  normalizeNamespaceSelection,
  readStorageJson,
  readStorageString,
  sortItemsByPath,
  writeStorageJson,
  writeStorageString
} from './ClusterManageView.utils'

const ClusterManageWorkbenches = defineAsyncComponent(() => import('./clusterManage/overlays/ClusterManageWorkbenches.vue'))
const ClusterManageDetailsHost = defineAsyncComponent(() => import('./clusterManage/overlays/ClusterManageDetailsHost.vue'))
const ClusterManageResourceEditors = defineAsyncComponent(() => import('./clusterManage/overlays/ClusterManageResourceEditors.vue'))
const ClusterDashboard = defineAsyncComponent(() => import('./clusterManage/ClusterDashboard.vue'))
const ClusterScopedResourcesPanel = defineAsyncComponent(() => import('./clusterManage/panels/ClusterScopedResourcesPanel.vue'))
const ClusterRolesPanel = defineAsyncComponent(() => import('./clusterManage/panels/ClusterRolesPanel.vue'))
const ClusterRoleBindingsPanel = defineAsyncComponent(() => import('./clusterManage/panels/ClusterRoleBindingsPanel.vue'))
const ConfigMapsPanel = defineAsyncComponent(() => import('./clusterManage/panels/ConfigMapsPanel.vue'))
const CronJobsPanel = defineAsyncComponent(() => import('./clusterManage/panels/CronJobsPanel.vue'))
const EventsPanel = defineAsyncComponent(() => import('./clusterManage/panels/EventsPanel.vue'))
const HPAsPanel = defineAsyncComponent(() => import('./clusterManage/panels/HPAsPanel.vue'))
const IngressClassesPanel = defineAsyncComponent(() => import('./clusterManage/panels/IngressClassesPanel.vue'))
const IngressesPanel = defineAsyncComponent(() => import('./clusterManage/panels/IngressesPanel.vue'))
const JobsPanel = defineAsyncComponent(() => import('./clusterManage/panels/JobsPanel.vue'))
const NamespacedGovernancePanel = defineAsyncComponent(() => import('./clusterManage/panels/NamespacedGovernancePanel.vue'))
const NamespacesPanel = defineAsyncComponent(() => import('./clusterManage/panels/NamespacesPanel.vue'))
const NodesPanel = defineAsyncComponent(() => import('./clusterManage/panels/NodesPanel.vue'))
const PDBsPanel = defineAsyncComponent(() => import('./clusterManage/panels/PDBsPanel.vue'))
const PodsPanel = defineAsyncComponent(() => import('./clusterManage/panels/PodsPanel.vue'))
const PermissionAuditsPanel = defineAsyncComponent(() => import('./clusterManage/panels/PermissionAuditsPanel.vue'))
const PvcsPanel = defineAsyncComponent(() => import('./clusterManage/panels/PvcsPanel.vue'))
const PvsPanel = defineAsyncComponent(() => import('./clusterManage/panels/PvsPanel.vue'))
const ResourceTopologyView = defineAsyncComponent(() => import('./ResourceTopologyView.vue'))
const RoleBindingsPanel = defineAsyncComponent(() => import('./clusterManage/panels/RoleBindingsPanel.vue'))
const RolesPanel = defineAsyncComponent(() => import('./clusterManage/panels/RolesPanel.vue'))
const SecretsPanel = defineAsyncComponent(() => import('./clusterManage/panels/SecretsPanel.vue'))
const ServiceAccountsPanel = defineAsyncComponent(() => import('./clusterManage/panels/ServiceAccountsPanel.vue'))
const ServicesPanel = defineAsyncComponent(() => import('./clusterManage/panels/ServicesPanel.vue'))
const StorageClassesPanel = defineAsyncComponent(() => import('./clusterManage/panels/StorageClassesPanel.vue'))
const WorkloadsPanel = defineAsyncComponent(() => import('./clusterManage/panels/WorkloadsPanel.vue'))

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

type YamlLoader = () => Promise<{ text: string }>
type YamlSaver = (text: string) => Promise<void>
type ClusterManageDetailsHostExpose = {
  openScale: (payload: { kind: string; namespace: string; name: string }, desired: number, available: number) => void
  openEditDeployment: (row: any) => void
  openEditDaemonSet: (row: any) => void
  openEditStatefulSet: (row: any) => void
  openYaml: (meta: string, loader: YamlLoader, saver?: YamlSaver) => void
  openWorkloadRollout: (payload: { kind: string; namespace: string; name: string }) => void
  openServiceDetail: (row: any) => void
  openIngressDetail: (row: any) => void
  openIngressClassDetail: (row: any) => void
  openPVCDetail: (row: any) => void
  openPVDetail: (row: any) => void
  openConfigMapDetail: (row: any) => void
  openSecretDetail: (row: any) => void
  openJobDetail: (row: any) => void
  openCronJobDetail: (row: any) => void
  openPodDetail: (row: any) => void
  openNodeDetail: (row: any) => void
  openDeploymentDetail: (row: any, kind?: string) => void
  openStatefulSetDetail: (row: any) => void
}

const detailsHostRef = ref<ClusterManageDetailsHostExpose | null>(null)
const workbenchesRef = ref<InstanceType<typeof ClusterManageWorkbenchesComponent> | null>(null)
const resourceEditorsRef = ref<InstanceType<typeof ClusterManageResourceEditorsComponent> | null>(null)
const overlayLoaded = reactive({
  workbenches: false,
  resourceEditors: false
})

function runOverlayWhenReady<T>(
  key: 'workbenches' | 'resourceEditors',
  getter: () => T | null | undefined,
  runner: (target: T) => void,
  attempt = 0
) {
  if (!overlayLoaded[key]) overlayLoaded[key] = true
  void nextTick(() => {
    const target = getter()
    if (target) {
      runner(target)
      return
    }
    if (attempt >= 60) return
    window.setTimeout(() => runOverlayWhenReady(key, getter, runner, attempt + 1), 32)
  })
}

const namespacesPanelRef = ref<any>(null)
const nodesPanelRef = ref<any>(null)
const podsPanelRef = ref<any>(null)
const replicaSetsPanelRef = ref<any>(null)
const workloadsPanelRef = ref<any>(null)
const pdbsPanelRef = ref<any>(null)
const servicesPanelRef = ref<any>(null)
const networkPoliciesPanelRef = ref<any>(null)
const endpointsPanelRef = ref<any>(null)
const endpointSlicesPanelRef = ref<any>(null)
const ingressesPanelRef = ref<any>(null)
const ingressClassesPanelRef = ref<any>(null)
const configMapsPanelRef = ref<any>(null)
const secretsPanelRef = ref<any>(null)
const pvcsPanelRef = ref<any>(null)
const pvsPanelRef = ref<any>(null)
const storageClassesPanelRef = ref<any>(null)
const csiDriversPanelRef = ref<any>(null)
const csiNodesPanelRef = ref<any>(null)
const csiStorageCapacitiesPanelRef = ref<any>(null)
const volumeAttachmentsPanelRef = ref<any>(null)
const volumeSnapshotsPanelRef = ref<any>(null)
const volumeSnapshotClassesPanelRef = ref<any>(null)
const volumeSnapshotContentsPanelRef = ref<any>(null)
const resourceQuotasPanelRef = ref<any>(null)
const limitRangesPanelRef = ref<any>(null)
const customResourceDefinitionsPanelRef = ref<any>(null)
const apiServicesPanelRef = ref<any>(null)
const priorityClassesPanelRef = ref<any>(null)
const runtimeClassesPanelRef = ref<any>(null)
const validatingWebhookConfigurationsPanelRef = ref<any>(null)
const mutatingWebhookConfigurationsPanelRef = ref<any>(null)
const validatingAdmissionPoliciesPanelRef = ref<any>(null)
const validatingAdmissionPolicyBindingsPanelRef = ref<any>(null)
const jobsPanelRef = ref<any>(null)
const cronJobsPanelRef = ref<any>(null)
const eventsPanelRef = ref<any>(null)
const leasesPanelRef = ref<any>(null)
const hpasPanelRef = ref<any>(null)
const serviceAccountsPanelRef = ref<any>(null)
const rolesPanelRef = ref<any>(null)
const clusterRolesPanelRef = ref<any>(null)
const roleBindingsPanelRef = ref<any>(null)
const clusterRoleBindingsPanelRef = ref<any>(null)

const showListTableTools = true
const canReadClusterMeta = computed(() => userStore.permissions.includes('cluster:read'))
const canLoadNamespaceOptions = computed(() => userStore.permissions.includes('k8s:read') || userStore.permissions.includes('k8s:rbac_read'))
const canWriteK8s = computed(() => userStore.permissions.includes('k8s:write'))
const canRevealSecret = computed(() => userStore.permissions.includes('k8s:secret_reveal'))
const canWriteRbac = computed(() => userStore.permissions.includes('k8s:rbac_write'))

const STORAGE_PREFIX = 'k8s:cluster_manage'
function storageKey(suffix: string): string {
  return buildStorageKey(STORAGE_PREFIX, clusterId.value, suffix)
}

const clusterId = computed(() => {
  const v = Number(route.params.clusterId)
  return Number.isFinite(v) && v > 0 ? v : 0
})

const clusterDetail = ref<clustersApi.ClusterDetail | null>(null)
const clusterName = computed(() => {
  const name = String(clusterDetail.value?.name ?? '').trim()
  return name || '-'
})
const clusterTitle = computed(() => {
  if (!clusterId.value) return 'K8s 集群'
  const n = clusterDetail.value?.name ? `：${clusterDetail.value.name}` : ''
  return `K8s 集群${n}`
})

const loadingTree = ref(false)
const namespaces = ref<string[]>([])
const tree = ref<TreeNode[]>([])
const resourceSupport = ref<Partial<Record<ResourceKey, boolean>>>({})

const treeRef = ref()
const current = ref<TreeNode | null>(null)
const currentResource = computed(() => current.value?.resource)

const loadingList = ref(false)
const list = shallowRef<K8sLikeObject[]>([])

const {
  clusterOverview,
  loadingDashboard,
  loadDashboard,
  loadDashboardPrefs,
  dashboardNamespacesTotal,
  dashboardEvents,
  dashboardLastUpdatedAt,
  dashboardCertRiskLoading,
  dashboardCertRiskUnavailable,
  dashboardAlertCounts,
  dashboardHealthScore,
  dashboardAutoRefresh,
  dashboardAutoRefreshSec,
  dashboardCpuText,
  dashboardMemoryText,
  dashboardReadyText,
  dashboardPodRunningText,
  dashboardPodPendingText,
  dashboardPodFailedText,
  dashboardTopNsText,
  dashboardWorkloadsTotal,
  dashboardEnabledWidgets,
  dashboardNamespaceNames,
  setDashboardNamespaceSnapshot,
} = useClusterManageDashboard({ clusterId, currentResource })

const ALL_NAMESPACE = '__all__'
type PodPhaseQuickFilter = '' | 'Running' | 'Pending' | 'Failed' | 'Completed'
const workloadLabelSelector = ref('')
const workloadKind = computed<WorkloadKind>(() => current.value?.workloadKind ?? 'Deployment')

const {
  openNamespaceYaml,
  openNodeYaml,
  openServiceYaml,
  openNetworkPolicyYaml,
  openEndpointsYaml,
  openEndpointSliceYaml,
  openReplicaSetYaml,
  openIngressYaml,
  openConfigMapYaml,
  openSecretYaml,
  openServiceAccountYaml,
  openEditServiceAccount,
  openPdbYaml,
  openEditPdb,
  openRoleYaml,
  openClusterRoleYaml,
  openEditRole,
  openEditClusterRole,
  openRoleBindingYaml,
  openEditRoleBinding,
  openClusterRoleBindingYaml,
  openEditClusterRoleBinding,
  openHPAYaml,
  openEditHPA,
  openWorkloadYaml,
  openPodYaml,
  openIngressClassYaml,
  openPVCYaml,
  openPVYaml,
  openStorageClassYaml,
  openCSIDriverYaml,
  openCSINodeYaml,
  openCSIStorageCapacityYaml,
  openVolumeAttachmentYaml,
  openVolumeSnapshotYaml,
  openVolumeSnapshotClassYaml,
  openVolumeSnapshotContentYaml,
  openResourceQuotaYaml,
  openEditResourceQuota,
  openLimitRangeYaml,
  openEditLimitRange,
  openLeaseYaml,
  openEditLease,
  openCustomResourceDefinitionYaml,
  openEditCustomResourceDefinition,
  openAPIServiceYaml,
  openEditAPIService,
  openPriorityClassYaml,
  openRuntimeClassYaml,
  openValidatingWebhookConfigurationYaml,
  openMutatingWebhookConfigurationYaml,
  openValidatingAdmissionPolicyYaml,
  openValidatingAdmissionPolicyBindingYaml,
  openEditStorageClass,
  openEditCSIDriver,
  openEditCSINode,
  openEditCSIStorageCapacity,
  openEditVolumeSnapshot,
  openEditVolumeSnapshotClass,
  openEditVolumeSnapshotContent,
  openEditNetworkPolicy,
  openEditEndpoints,
  openEditEndpointSlice,
  openEditReplicaSet,
  openEditPriorityClass,
  openEditRuntimeClass,
  openEditValidatingWebhookConfiguration,
  openEditMutatingWebhookConfiguration,
  openEditValidatingAdmissionPolicy,
  openEditValidatingAdmissionPolicyBinding,
  openEditVolumeAttachment,
  openJobYaml,
  openCronJobYaml
} = useClusterManageYamlActions({
  clusterId,
  workloadKind,
  openYaml,
  loadCurrent
})

const {
  deleteWorkloadRow,
  deleteHPARow,
  deleteServiceRow,
  deleteNetworkPolicyRow,
  deleteIngressRow,
  deleteConfigMapRow,
  deleteSecretRow,
  deleteServiceAccountRow,
  deletePdbRow,
  deleteRoleRow,
  deleteClusterRoleRow,
  deleteRoleBindingRow,
  deleteClusterRoleBindingRow,
  deletePodRow,
  deleteNamespaceRow,
  deleteEndpointsRow,
  deleteEndpointSliceRow,
  deleteReplicaSetRow,
  deleteIngressClassRow,
  deletePVCRow,
  deletePVRow,
  deleteStorageClassRow,
  deleteCSIDriverRow,
  deleteCSINodeRow,
  deleteCSIStorageCapacityRow,
  deleteVolumeSnapshotRow,
  deleteVolumeSnapshotClassRow,
  deleteVolumeSnapshotContentRow,
  deleteResourceQuotaRow,
  deleteLimitRangeRow,
  deleteVolumeAttachmentRow,
  deleteLeaseRow,
  deleteCustomResourceDefinitionRow,
  deleteAPIServiceRow,
  deletePriorityClassRow,
  deleteRuntimeClassRow,
  deleteValidatingWebhookConfigurationRow,
  deleteMutatingWebhookConfigurationRow,
  deleteValidatingAdmissionPolicyRow,
  deleteValidatingAdmissionPolicyBindingRow,
  deleteJobRow,
  deleteCronJobRow
} = useClusterManageDeleteActions({
  clusterId,
  workloadKind,
  loadCurrent,
  refreshAll
})

const {
  sortBy,
  order,
  keywordInput,
  keyword,
  page,
  pageSize,
  pageSizeOptions,
  namespace,
  showNamespaceSelect,
  displayedList,
  displayedTotal,
  maxPage,
  pagedList,
  showPager,
  onNamespaceSelectChange,
  onPageChange,
  onPageSizeChange,
  onSortChange: onViewSortChange,
  stopTimers
} = useClusterManageViewState({
  list,
  currentResource,
  namespaces,
  allNamespace: ALL_NAMESPACE,
  clearPodSelection,
  extraFilter: computed(() => {
    const resource = currentResource.value
    if (resource === 'pods' && podPhaseQuickFilter.value) {
      return (item: any) => getPodPhaseQuickFilter(item) === podPhaseQuickFilter.value
    }
    if (resource === 'events' && (eventTypeFilter.value || eventReasonFilter.value)) {
      return (item: any) => {
        const type = String(item?.type ?? '').trim()
        const reason = String(item?.reason ?? '').trim()
        if (eventTypeFilter.value && type !== eventTypeFilter.value) return false
        if (eventReasonFilter.value && reason !== eventReasonFilter.value) return false
        return true
      }
    }
    return undefined
  })
})

const activeTableColumns = ref<EnhancedColumn[]>([])
const activeColumnVisible = reactive<Record<string, boolean>>({})

function getActiveEnhancedTable(): any | null {
  const r = current.value?.resource
  if (!r) return null
  if (r === 'namespaces') return namespacesPanelRef.value?.getTable?.() ?? null
  if (r === 'nodes') return nodesPanelRef.value?.getTable?.() ?? null
  if (r === 'pods') return podsPanelRef.value?.getTable?.() ?? null
  if (r === 'replicasets') return replicaSetsPanelRef.value?.getTable?.() ?? null
  if (r === 'workloads') return workloadsPanelRef.value?.getTable?.() ?? null
  if (r === 'pdbs') return pdbsPanelRef.value?.getTable?.() ?? null
  if (r === 'hpas') return hpasPanelRef.value?.getTable?.() ?? null
  if (r === 'services') return servicesPanelRef.value?.getTable?.() ?? null
  if (r === 'networkpolicies') return networkPoliciesPanelRef.value?.getTable?.() ?? null
  if (r === 'endpoints') return endpointsPanelRef.value?.getTable?.() ?? null
  if (r === 'endpointslices') return endpointSlicesPanelRef.value?.getTable?.() ?? null
  if (r === 'ingresses') return ingressesPanelRef.value?.getTable?.() ?? null
  if (r === 'ingressclasses') return ingressClassesPanelRef.value?.getTable?.() ?? null
  if (r === 'configmaps') return configMapsPanelRef.value?.getTable?.() ?? null
  if (r === 'secrets') return secretsPanelRef.value?.getTable?.() ?? null
  if (r === 'serviceaccounts') return serviceAccountsPanelRef.value?.getTable?.() ?? null
  if (r === 'roles') return rolesPanelRef.value?.getTable?.() ?? null
  if (r === 'clusterroles') return clusterRolesPanelRef.value?.getTable?.() ?? null
  if (r === 'rolebindings') return roleBindingsPanelRef.value?.getTable?.() ?? null
  if (r === 'clusterrolebindings') return clusterRoleBindingsPanelRef.value?.getTable?.() ?? null
  if (r === 'pvcs') return pvcsPanelRef.value?.getTable?.() ?? null
  if (r === 'pvs') return pvsPanelRef.value?.getTable?.() ?? null
  if (r === 'storageclasses') return storageClassesPanelRef.value?.getTable?.() ?? null
  if (r === 'csidrivers') return csiDriversPanelRef.value?.getTable?.() ?? null
  if (r === 'csinodes') return csiNodesPanelRef.value?.getTable?.() ?? null
  if (r === 'csistoragecapacities') return csiStorageCapacitiesPanelRef.value?.getTable?.() ?? null
  if (r === 'volumeattachments') return volumeAttachmentsPanelRef.value?.getTable?.() ?? null
  if (r === 'volumesnapshots') return volumeSnapshotsPanelRef.value?.getTable?.() ?? null
  if (r === 'volumesnapshotclasses') return volumeSnapshotClassesPanelRef.value?.getTable?.() ?? null
  if (r === 'volumesnapshotcontents') return volumeSnapshotContentsPanelRef.value?.getTable?.() ?? null
  if (r === 'resourcequotas') return resourceQuotasPanelRef.value?.getTable?.() ?? null
  if (r === 'limitranges') return limitRangesPanelRef.value?.getTable?.() ?? null
  if (r === 'customresourcedefinitions') return customResourceDefinitionsPanelRef.value?.getTable?.() ?? null
  if (r === 'apiservices') return apiServicesPanelRef.value?.getTable?.() ?? null
  if (r === 'priorityclasses') return priorityClassesPanelRef.value?.getTable?.() ?? null
  if (r === 'runtimeclasses') return runtimeClassesPanelRef.value?.getTable?.() ?? null
  if (r === 'validatingwebhookconfigurations') return validatingWebhookConfigurationsPanelRef.value?.getTable?.() ?? null
  if (r === 'mutatingwebhookconfigurations') return mutatingWebhookConfigurationsPanelRef.value?.getTable?.() ?? null
  if (r === 'validatingadmissionpolicies') return validatingAdmissionPoliciesPanelRef.value?.getTable?.() ?? null
  if (r === 'validatingadmissionpolicybindings') return validatingAdmissionPolicyBindingsPanelRef.value?.getTable?.() ?? null
  if (r === 'jobs') return jobsPanelRef.value?.getTable?.() ?? null
  if (r === 'cronjobs') return cronJobsPanelRef.value?.getTable?.() ?? null
  if (r === 'events') return eventsPanelRef.value?.getTable?.() ?? null
  if (r === 'leases') return leasesPanelRef.value?.getTable?.() ?? null
  return null
}

function syncActiveColumns() {
  const t = getActiveEnhancedTable()
  if (!t) {
    activeTableColumns.value = []
    for (const k of Object.keys(activeColumnVisible)) delete activeColumnVisible[k]
    return
  }
  const cols = (t.getColumns?.() ?? []) as EnhancedColumn[]
  activeTableColumns.value = cols
  const m = (t.getColumnVisibleMap?.() ?? {}) as Record<string, boolean>
  for (const k of Object.keys(activeColumnVisible)) delete activeColumnVisible[k]
  for (const [k, v] of Object.entries(m)) activeColumnVisible[k] = v
}

const showColumnPickerBtn = computed(() => {
  if (!current.value?.resource) return false
  return current.value.resource !== 'dashboard'
})

async function onColumnsDropdownVisibleChange(v: boolean) {
  if (!v) return
  await nextTick()
  syncActiveColumns()
}

async function onActiveColumnVisibleChange() {
  await nextTick()
  const t = getActiveEnhancedTable()
  if (!t?.setColumnVisibleMap) return
  t.setColumnVisibleMap({ ...activeColumnVisible })
  syncActiveColumns()
}

function selectAllActiveColumns() {
  const t = getActiveEnhancedTable()
  t?.selectAllColumns?.()
  syncActiveColumns()
}

function clearAllActiveColumns() {
  const t = getActiveEnhancedTable()
  t?.clearAllColumns?.()
  syncActiveColumns()
}

function resetActiveColumns() {
  const t = getActiveEnhancedTable()
  t?.resetColumns?.()
  syncActiveColumns()
}

watch([currentResource, workloadKind], async () => {
  await nextTick()
  syncActiveColumns()
})

function openEditDeployment(row: any) {
  detailsHostRef.value?.openEditDeployment(row)
}

function openDaemonSetDetail(row: any) {
  if (!clusterId.value) return
  const ns = getRowNamespace(row)
  const name = String(row?.metadata?.name ?? '')
  if (!ns || !name) return
  detailsHostRef.value?.openDeploymentDetail(row, 'DaemonSet')
}

function openEditDaemonSet(row: any) {
  detailsHostRef.value?.openEditDaemonSet(row)
}

function openStatefulSetDetail(row: any) {
  detailsHostRef.value?.openStatefulSetDetail(row)
}

function openWorkloadRollout(row: any) {
  const kind = String(row?.kind ?? workloadKind.value ?? '').trim() || workloadKind.value
  if (kind === 'StatefulSet') {
    notifyError('当前仅支持 Deployment 版本历史')
    return
  }
  if (kind !== 'Deployment') return
  const namespace = getRowNamespace(row)
  const name = String(row?.metadata?.name ?? '').trim()
  if (!namespace || !name) return
  detailsHostRef.value?.openWorkloadRollout({ kind, namespace, name })
}

function openEditStatefulSet(row: any) {
  detailsHostRef.value?.openEditStatefulSet(row)
}

const toolbarResourceText = computed(() => {
  const r = current.value?.resource
  if (!r) return '-'
  if (r === 'workloads') return `${r}(${workloadKind.value})`
  return r
})

const resourceBadgeClass = computed(() => {
  const r = current.value?.resource
  if (!r) return 'resource-badge--default'
  return getResourceBadgeClass(r)
})

const selectedPodRows = ref<any[]>([])
const deletingPods = ref(false)
const podPhaseQuickFilter = ref<PodPhaseQuickFilter>('')
const eventTypeFilter = ref('')
const eventReasonFilter = ref('')
const warningEventSummary = ref<{ byName: Record<string, number>; byUID: Record<string, number> }>({ byName: {}, byUID: {} })
const podPhaseSummary = computed<Record<Exclude<PodPhaseQuickFilter, ''>, number>>(() => {
  const summary = {
    Running: 0,
    Pending: 0,
    Failed: 0,
    Completed: 0
  }
  for (const row of list.value) {
    const phase = getPodPhaseQuickFilter(row)
    if (phase) summary[phase] += 1
  }
  return summary
})
const completedJobsCount = computed(() => displayedList.value.filter((row) => isFinishedJob(row)).length)
const eventTypeOptions = computed(() => {
  if (currentResource.value !== 'events') return []
  return Array.from(new Set(list.value.map((row: any) => String(row?.type ?? '').trim()).filter(Boolean))).sort((a, b) => a.localeCompare(b))
})
const eventReasonOptions = computed(() => {
  if (currentResource.value !== 'events') return []
  return Array.from(new Set(list.value.map((row: any) => String(row?.reason ?? '').trim()).filter(Boolean))).sort((a, b) => a.localeCompare(b))
})
const defaultCreatePVCNamespace = computed(() => {
  const selected = getNamespaceFilter(namespace.value, ALL_NAMESPACE)
  if (selected && selected.length === 1) return selected[0]
  if (namespaces.value.includes('default')) return 'default'
  return namespaces.value[0] || ''
})

function onPodSelectionChange(rows: any[]) {
  selectedPodRows.value = Array.isArray(rows) ? rows : []
}

function onPodPhaseQuickFilterChange(next: PodPhaseQuickFilter) {
  podPhaseQuickFilter.value = next
  page.value = 1
  clearPodSelection()
}

function onEventTypeFilterChange(next: string) {
  eventTypeFilter.value = String(next ?? '')
  page.value = 1
}

function onEventReasonFilterChange(next: string) {
  eventReasonFilter.value = String(next ?? '')
  page.value = 1
}

function clearPodSelection() {
  selectedPodRows.value = []
  podsPanelRef.value?.getTable?.()?.clearSelection?.()
}

function resetWarningEventSummary() {
  warningEventSummary.value = { byName: {}, byUID: {} }
}

function getEventInvolvedObject(row: any) {
  return row?.involvedObject ?? row?.regarding ?? {}
}

function isWarningEventRow(row: any): boolean {
  return String(row?.type ?? '').trim().toLowerCase() === 'warning'
}

function buildWarningEventNameKey(kind: string, namespaceText: string, name: string): string {
  return `${kind}:${namespaceText}/${name}`
}

function getRowWarningEventCount(row: any, kind: string): number {
  const normalizedKind = String(kind ?? '').trim()
  const uid = String(row?.metadata?.uid ?? '').trim()
  if (uid) {
    const countByUID = Number(warningEventSummary.value.byUID[uid] ?? 0)
    if (Number.isFinite(countByUID) && countByUID > 0) return countByUID
  }
  const namespaceText = String(getRowNamespace(row) ?? '').trim()
  const name = String(row?.metadata?.name ?? '').trim()
  if (!normalizedKind || !namespaceText || !name) return 0
  const count = Number(warningEventSummary.value.byName[buildWarningEventNameKey(normalizedKind, namespaceText, name)] ?? 0)
  return Number.isFinite(count) && count > 0 ? count : 0
}

function getPodWarningEventCount(row: any): number {
  return getRowWarningEventCount(row, 'Pod')
}

function getWorkloadWarningEventCount(row: any): number {
  return getRowWarningEventCount(row, String(row?.kind ?? workloadKind.value ?? ''))
}

let warningEventSummarySeq = 0

async function loadWarningEventSummary() {
  const resource = currentResource.value
  if (!clusterId.value || (resource !== 'pods' && resource !== 'workloads')) {
    resetWarningEventSummary()
    return
  }

  const rows = Array.isArray(list.value) ? list.value : []
  if (rows.length === 0) {
    resetWarningEventSummary()
    return
  }

  const involvedKind = resource === 'pods' ? 'Pod' : String(workloadKind.value ?? '').trim()
  if (!involvedKind) {
    resetWarningEventSummary()
    return
  }

  const targetNamespaces = Array.from(
    new Set(
      rows
        .map((row) => String(getRowNamespace(row) ?? '').trim())
        .filter(Boolean)
    )
  )
  if (targetNamespaces.length === 0) {
    resetWarningEventSummary()
    return
  }

  const seq = ++warningEventSummarySeq
  try {
    const batches = await Promise.all(
      targetNamespaces.map((ns) =>
        k8sApi.listEvents(clusterId.value, {
          namespace: ns,
          involved_object_kind: involvedKind
        })
      )
    )
    if (seq !== warningEventSummarySeq) return

    const byName: Record<string, number> = {}
    const byUID: Record<string, number> = {}
    for (const batch of batches) {
      for (const item of batch.list ?? []) {
        if (!isWarningEventRow(item)) continue
        const involved = getEventInvolvedObject(item)
        const namespaceText = String(involved?.namespace ?? '').trim()
        const name = String(involved?.name ?? '').trim()
        const uid = String(involved?.uid ?? '').trim()
        if (uid) byUID[uid] = (byUID[uid] ?? 0) + 1
        if (!namespaceText || !name) continue
        const key = buildWarningEventNameKey(involvedKind, namespaceText, name)
        byName[key] = (byName[key] ?? 0) + 1
      }
    }
    warningEventSummary.value = { byName, byUID }
  } catch {
    if (seq !== warningEventSummarySeq) return
    resetWarningEventSummary()
  }
}

function applyPersistedViewState(node: TreeNode) {
  const r = node.resource
  if (!r) return

  const sort = readStorageJson<{ sortBy?: string; order?: SortOrder }>(storageKey(`sort:${r}`))
  sortBy.value = sort?.sortBy
  order.value = sort?.order

  const pagerMigratedKey = storageKey(`pager_migrated_default20:${r}`)
  const pagerMigrated = readStorageString(pagerMigratedKey) === '1'
  const pager = readStorageJson<{ page?: number; pageSize?: number }>(storageKey(`pager:${r}`))
  const rawPageSize = pager?.pageSize && Number.isFinite(pager.pageSize) ? Math.max(1, Number(pager.pageSize)) : 20
  const nextPageSize = !pagerMigrated && rawPageSize === 50 ? 20 : rawPageSize
  pageSize.value = nextPageSize
  page.value = pager?.page && Number.isFinite(pager.page) ? Math.max(1, Number(pager.page)) : 1
  if (!pagerMigrated && rawPageSize === 50) {
    writeStorageString(pagerMigratedKey, '1')
    writeStorageJson(storageKey(`pager:${r}`), { page: page.value, pageSize: pageSize.value })
  }

  const kw = readStorageString(storageKey(`keyword:${r}`))
  keywordInput.value = kw ?? ''
  keyword.value = ''

  if (r === 'workloads') {
    const wl = readStorageJson<{ labelSelector?: string }>(storageKey('workloads'))
    workloadLabelSelector.value = wl?.labelSelector ?? ''
  }
}

function persistViewState() {
  const r = current.value?.resource
  if (!r) return
  writeStorageJson(storageKey(`sort:${r}`), { sortBy: sortBy.value, order: order.value })
  writeStorageJson(storageKey(`pager:${r}`), { page: page.value, pageSize: pageSize.value })
  writeStorageString(storageKey(`keyword:${r}`), keywordInput.value || null)
  if (r === 'workloads') writeStorageJson(storageKey('workloads'), { labelSelector: workloadLabelSelector.value })
}

function getDefaultNamespaceSelection(resource?: ResourceKey): string[] {
  if (resource === 'leases') {
    if (namespaces.value.includes('kube-node-lease')) return ['kube-node-lease']
    return [ALL_NAMESPACE]
  }
  if (namespaces.value.includes('default')) return ['default']
  if (namespaces.value[0]) return [namespaces.value[0]]
  return [ALL_NAMESPACE]
}

function createDefaultResourceSupport(): Partial<Record<ResourceKey, boolean>> {
  return {}
}

let persistTimer: number | null = null
function schedulePersistViewState() {
  if (persistTimer != null) window.clearTimeout(persistTimer)
  persistTimer = window.setTimeout(() => {
    persistTimer = null
    persistViewState()
  }, 200)
}

let refreshAllPromise: Promise<void> | null = null
async function refreshAll() {
  if (!clusterId.value) return
  if (refreshAllPromise) return await refreshAllPromise

  refreshAllPromise = (async () => {
    loadingTree.value = true
    resourceSupport.value = createDefaultResourceSupport()

    // Fire cluster detail fetch in background — header info, NOT needed by tree or dashboard
    const clusterDetailPromise = canReadClusterMeta.value
      ? clustersApi.getCluster(clusterId.value).then(
          (d) => { clusterDetail.value = d },
          (e) => {
            const err = e as ApiError
            notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
          }
        )
      : Promise.resolve()

    const resourceSupportPromise = (userStore.permissions.includes('k8s:read') || userStore.permissions.includes('k8s:rbac_read'))
      ? k8sApi.getResourceSupport(clusterId.value).then(
          (data) => {
            resourceSupport.value = data
          },
          (e) => {
            resourceSupport.value = createDefaultResourceSupport()
            const err = e as ApiError
            notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
          }
        )
      : Promise.resolve()

    // Build tree (local, zero network) + kick off current view load immediately
    try {
      await resourceSupportPromise
      tree.value = filterTreeByResourceSupport(filterTreeByPerms(buildTree({
        k8sLogoUrl,
        k8sIconCmUrl,
        k8sIconCronJobUrl,
        k8sIconDeploymentUrl,
        k8sIconDaemonSetUrl,
        k8sIconGroupUrl,
        k8sIconIngressUrl,
        k8sIconJobUrl,
        k8sIconNodeUrl,
        k8sIconNamespaceUrl,
        k8sIconPodUrl,
        k8sIconPvUrl,
        k8sIconPvcUrl,
        k8sIconStorageClassUrl,
        k8sIconSecretUrl,
        k8sIconStatefulSetUrl,
        k8sIconServiceUrl
      }), userStore.permissions ?? []), resourceSupport.value)
      await nextTick()
      autoSelectDefault() // triggers loadCurrent (fire-and-forget → dashboard or resource)
    } catch (e) {
      const err = e as ApiError
      notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
    }

    // Sidebar namespaces — runs concurrently with clusterDetail + loadCurrent
    if (canLoadNamespaceOptions.value) {
      try {
        const data = await k8sApi.listNamespaces(clusterId.value)
        namespaces.value = data.list.map((it) => it.metadata.name)
        setDashboardNamespaceSnapshot(namespaces.value)
        const persistedNs = readStorageJson<unknown>(storageKey('namespace'))
        if (persistedNs != null) {
          onNamespaceSelectChange(persistedNs)
        } else {
          namespace.value = getDefaultNamespaceSelection(current.value?.resource)
        }
      } catch (e) {
        const err = e as ApiError
        notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
      }
    } else {
      namespaces.value = []
      namespace.value = [ALL_NAMESPACE]
    }

    // Wait for cluster detail to finish (non-blocking for UI — tree/dashboard already loading)
    await clusterDetailPromise

    loadingTree.value = false

    // autoSelectDefault may have loaded a namespaced view before persisted namespace
    // selection was restored. Reload once here so the right panel matches the final selection.
    if (current.value?.resource && isNamespacedResource(current.value.resource)) {
      await loadCurrent()
    }
  })().finally(() => {
    refreshAllPromise = null
  })

  return await refreshAllPromise
}

function autoSelectDefault() {
  const key = readStorageString(storageKey('currentKey')) || 'dashboard:overview'
  treeRef.value?.setCurrentKey?.(key)

  const find = (nodes: TreeNode[]): TreeNode | null => {
    for (const n of nodes) {
      if (n.id === key) return n
      if (n.children) {
        const hit = find(n.children)
        if (hit) return hit
      }
    }
    return null
  }
  const hit = find(tree.value)
  if (hit && hit.kind === 'view') {
    current.value = hit
    applyPersistedViewState(hit)
    void loadCurrent()
    return
  }

  const firstView = (nodes: TreeNode[]): TreeNode | null => {
    for (const n of nodes) {
      if (n.kind === 'view') return n
      const child = firstView(n.children ?? [])
      if (child) return child
    }
    return null
  }

  const fallback = firstView(tree.value)
  if (fallback) {
    treeRef.value?.setCurrentKey?.(fallback.id)
    current.value = fallback
    applyPersistedViewState(fallback)
    void loadCurrent()
  }
}

function onTreeNodeClick(node: TreeNode) {
  if (node.kind !== 'view') return
  clearPodSelection()
  current.value = node
  writeStorageString(storageKey('currentKey'), node.id)
  applyPersistedViewState(node)
  if (readStorageJson<unknown>(storageKey('namespace')) == null && node.resource && isNamespacedResource(node.resource)) {
    namespace.value = getDefaultNamespaceSelection(node.resource)
  }
  scheduleLoadCurrent()
}

/** Dashboard KPI / table link clicked → navigate to the target resource tree node */
function onDashboardNavigate(treeNodeId: string) {
  const find = (nodes: TreeNode[]): TreeNode | null => {
    for (const n of nodes) {
      if (n.id === treeNodeId) return n
      if (n.children) { const hit = find(n.children); if (hit) return hit }
    }
    return null
  }
  const node = find(tree.value)
  if (node && node.kind === 'view') {
    treeRef.value?.setCurrentKey?.(node.id)
    onTreeNodeClick(node)
  }
}

function permissionAuditFindingNodeId(payload: { kind?: string; namespace?: string }): string | null {
  const kind = String(payload.kind ?? '').trim()
  if (!kind) return null
  if (kind === 'Deployment') return 'workloads:deployments'
  if (kind === 'StatefulSet') return 'workloads:statefulsets'
  if (kind === 'DaemonSet') return 'workloads:daemonsets'
  if (kind === 'Pod') return 'workloads:pods'
  if (kind === 'ServiceAccount') return 'config:serviceaccounts'
  if (kind === 'RoleBinding') return 'auth:rolebindings'
  if (kind === 'Role') return 'auth:roles'
  if (kind === 'ClusterRoleBinding') return 'auth:clusterrolebindings'
  if (kind === 'ClusterRole') return 'auth:clusterroles'
  if (kind === 'Service') return 'network:services'
  if (kind === 'Endpoints') return 'network:endpoints'
  if (kind === 'EndpointSlice') return 'network:endpointslices'
  if (kind === 'NetworkPolicy') return 'network:networkpolicies'
  if (kind === 'Ingress') return 'network:ingresses'
  if (kind === 'IngressClass') return 'network:ingressclasses'
  if (kind === 'ConfigMap') return 'config:configmaps'
  if (kind === 'Secret') return 'config:secrets'
  if (kind === 'ServiceAccount') return 'config:serviceaccounts'
  if (kind === 'VolumeSnapshot') return 'storage:volumesnapshots'
  if (kind === 'VolumeSnapshotClass') return 'storage:volumesnapshotclasses'
  if (kind === 'VolumeSnapshotContent') return 'storage:volumesnapshotcontents'
  if (kind === 'Role') return 'auth:roles'
  if (kind === 'ClusterRole') return 'auth:clusterroles'
  if (kind === 'RoleBinding') return 'auth:rolebindings'
  if (kind === 'ClusterRoleBinding') return 'auth:clusterrolebindings'
  if (kind === 'PersistentVolumeClaim') return 'storage:pvcs'
  if (kind === 'PersistentVolume') return 'storage:pvs'
  if (kind === 'StorageClass') return 'storage:storageclasses'
  if (kind === 'ResourceQuota') return 'storage:resourcequotas'
  if (kind === 'LimitRange') return 'storage:limitranges'
  if (kind === 'CustomResourceDefinition') return 'extensions:crds'
  if (kind === 'APIService') return 'extensions:apiservices'
  if (kind === 'PriorityClass') return 'extensions:priorityclasses'
  if (kind === 'ValidatingWebhookConfiguration') return 'extensions:validatingwebhooks'
  if (kind === 'MutatingWebhookConfiguration') return 'extensions:mutatingwebhooks'
  if (kind === 'Namespace') return 'cluster:namespaces'
  if (kind === 'Node') return 'cluster:nodes'
  if (kind === 'Lease') return 'cluster:leases'
  if (kind === 'Job') return 'jobs:jobs'
  if (kind === 'CronJob') return 'jobs:cronjobs'
  if (kind === 'PodDisruptionBudget') return 'workloads:pdbs'
  if (kind === 'HorizontalPodAutoscaler') return 'workloads:hpas'
  return null
}

function handlePermissionAuditNavigation(payload: { kind?: string; namespace?: string; name?: string }) {
  const nodeId = permissionAuditFindingNodeId(payload)
  if (!nodeId) {
    notifyError('当前 finding 暂未支持跳转到资源页')
    return
  }

  onDashboardNavigate(nodeId)

  if (payload.namespace) {
    namespace.value = [payload.namespace]
  } else if (!showNamespaceSelect.value) {
    namespace.value = [ALL_NAMESPACE]
  }
  keywordInput.value = String(payload.name ?? '').trim()
}

/** 欢迎页快捷按钮 → 跳转到指定资源 */
const quickSelectMap: Record<string, string> = {
  namespaces: 'cluster:namespaces',
  workloads: 'workloads:deployments',
  pods: 'workloads:pods',
  nodes: 'cluster:nodes',
  services: 'network:services',
  events: 'misc:events',
}
function quickSelect(key: string) {
  const nodeId = quickSelectMap[key]
  if (nodeId) onDashboardNavigate(nodeId)
}

function onSortChange(v: { prop?: string; order?: 'ascending' | 'descending' | null }) {
  onViewSortChange(v)
  schedulePersistViewState()
  scheduleLoadCurrent()
}

watch(
  () => namespace.value.slice().sort().join('|'),
  () => {
    writeStorageJson(storageKey('namespace'), namespace.value)
    schedulePersistViewState()
  }
)

watch(
  () => [page.value, pageSize.value, current.value?.resource].join('|'),
  () => {
    schedulePersistViewState()
  }
)

let workloadReloadTimer: number | null = null
watch(
  () => workloadLabelSelector.value,
  () => {
    schedulePersistViewState()
    if (workloadReloadTimer != null) window.clearTimeout(workloadReloadTimer)
    workloadReloadTimer = window.setTimeout(() => {
      workloadReloadTimer = null
      if (current.value?.resource === 'workloads') scheduleLoadCurrent()
    }, 260)
  }
)

let loadCurrentSeq = 0
let loadCurrentTimer: number | null = null
function scheduleLoadCurrent() {
  if (loadCurrentTimer != null) window.clearTimeout(loadCurrentTimer)
  loadCurrentTimer = window.setTimeout(() => {
    loadCurrentTimer = null
    void loadCurrent()
  }, 80)
}

async function loadCurrent() {
  if (!clusterId.value || !current.value?.resource) return
  if (current.value.resource === 'dashboard') {
    resetWarningEventSummary()
    await loadDashboard()
    return
  }

  const seq = ++loadCurrentSeq
  loadingList.value = true
  try {
    const nsFilter = getNamespaceFilter(namespace.value, ALL_NAMESPACE)

    const listByNamespaces = async (loader: (ns?: string) => Promise<any[]>): Promise<any[]> => {
      if (!nsFilter) return await loader(undefined)
      if (nsFilter.length <= 1) return await loader(nsFilter[0])
      const results = await Promise.all(nsFilter.map((ns) => loader(ns)))
      return sortItemsByPath(results.flat(), sortBy.value, order.value)
    }

    const resource = current.value.resource
    if (resource === 'permissionaudits') {
      list.value = []
      return
    }

    const loaders: Partial<Record<string, () => Promise<void>>> = {
      namespaces: async () => {
        const data = await k8sApi.listNamespaces(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      nodes: async () => {
        const data = await k8sApi.listNodes(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      pods: async () => {
        const backendSortable = sortBy.value === 'metadata.name' || sortBy.value === 'metadata.namespace'
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listPods(clusterId.value, {
            namespace: ns,
            sort_by: backendSortable ? sortBy.value : undefined,
            order: backendSortable ? order.value : undefined
          })
          return (data.list ?? []).map((it: any) => decoratePodRow(it))
        })
        if (seq !== loadCurrentSeq) return
        list.value = sortItemsByPath(merged, sortBy.value, order.value)
        clearPodSelection()
      },
      workloads: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listWorkloads(clusterId.value, {
            namespace: ns,
            kind: workloadKind.value,
            label_selector: workloadLabelSelector.value.trim() || undefined,
            sort_by: sortBy.value,
            order: order.value
          })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      pdbs: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listPDBs(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      hpas: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listHPAs(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      services: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listServices(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      endpoints: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listEndpoints(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      endpointslices: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listEndpointSlices(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      replicasets: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listReplicaSets(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      networkpolicies: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listNetworkPolicies(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      ingresses: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listIngresses(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      configmaps: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listConfigMaps(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      secrets: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listSecrets(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      serviceaccounts: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listServiceAccounts(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      roles: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listRoles(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      clusterroles: async () => {
        const data = await k8sApi.listClusterRoles(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      rolebindings: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listRoleBindings(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      clusterrolebindings: async () => {
        const data = await k8sApi.listClusterRoleBindings(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      events: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listEvents(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      leases: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listLeases(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      ingressclasses: async () => {
        const data = await k8sApi.listIngressClasses(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      pvcs: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listPVCs(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      volumesnapshots: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listVolumeSnapshots(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      pvs: async () => {
        const data = await k8sApi.listPVs(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      storageclasses: async () => {
        const data = await k8sApi.listStorageClasses(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      csidrivers: async () => {
        const data = await k8sApi.listCSIDrivers(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      csinodes: async () => {
        const data = await k8sApi.listCSINodes(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      csistoragecapacities: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listCSIStorageCapacities(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      volumesnapshotclasses: async () => {
        const data = await k8sApi.listVolumeSnapshotClasses(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      volumesnapshotcontents: async () => {
        const data = await k8sApi.listVolumeSnapshotContents(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      volumeattachments: async () => {
        const data = await k8sApi.listVolumeAttachments(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      resourcequotas: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listResourceQuotas(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      limitranges: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listLimitRanges(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      customresourcedefinitions: async () => {
        const data = await k8sApi.listCustomResourceDefinitions(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      apiservices: async () => {
        const data = await k8sApi.listAPIServices(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      priorityclasses: async () => {
        const data = await k8sApi.listPriorityClasses(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      runtimeclasses: async () => {
        const data = await k8sApi.listRuntimeClasses(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      validatingwebhookconfigurations: async () => {
        const data = await k8sApi.listValidatingWebhookConfigurations(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      mutatingwebhookconfigurations: async () => {
        const data = await k8sApi.listMutatingWebhookConfigurations(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      validatingadmissionpolicies: async () => {
        const data = await k8sApi.listValidatingAdmissionPolicies(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      validatingadmissionpolicybindings: async () => {
        const data = await k8sApi.listValidatingAdmissionPolicyBindings(clusterId.value, { sort_by: sortBy.value, order: order.value })
        if (seq !== loadCurrentSeq) return
        list.value = data.list
      },
      jobs: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listJobs(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      },
      cronjobs: async () => {
        const merged = await listByNamespaces(async (ns) => {
          const data = await k8sApi.listCronJobs(clusterId.value, { namespace: ns, sort_by: sortBy.value, order: order.value })
          return data.list
        })
        if (seq !== loadCurrentSeq) return
        list.value = merged
      }
    }

    await loaders[resource]?.()
    if (seq === loadCurrentSeq) {
      await loadWarningEventSummary()
    }
  } catch (e) {
    if (seq === loadCurrentSeq) {
      list.value = []
      resetWarningEventSummary()
    }
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    if (seq === loadCurrentSeq) loadingList.value = false
  }
}

onErrorCaptured((err) => {
  const msg = err instanceof Error ? err.message : String(err ?? 'unknown error')
  notifyError(msg)
  return false
})

function getReady(row: any): boolean {
  const conds: any[] = row?.status?.conditions ?? []
  const c = conds.find((it) => it?.type === 'Ready')
  return String(c?.status ?? '') === 'True'
}

function getDesired(row: any): number {
  return getWorkloadDesired(row)
}

function getReadyText(row: any): string {
  return getWorkloadReadyText(row)
}

function getPodReadyText(row: any): string {
  if (row?.ready != null) return String(row.ready)
  const css: any[] = Array.isArray(row?.status?.containerStatuses) ? row.status.containerStatuses : []
  const total = css.length
  if (total <= 0) return '-'
  const ready = css.reduce((sum, it) => sum + (it?.ready ? 1 : 0), 0)
  return `${ready}/${total}`
}

function getPodRestarts(row: any): number {
  if (row?.restarts != null) return Number(row.restarts ?? 0)
  const css: any[] = Array.isArray(row?.status?.containerStatuses) ? row.status.containerStatuses : []
  return css.reduce((sum, it) => sum + Number(it?.restartCount ?? 0), 0)
}

function getPodAgeMs(row: any): number | null {
  return getCreationAgeMs(row)
}

function getPodAge(row: any): string {
  return getCreationAgeText(row)
}

function decoratePodRow(row: any) {
  if (!row || typeof row !== 'object') return row
  const restarts = getPodRestarts(row)
  const ageMs = getPodAgeMs(row)
  row.restarts = restarts
  row.ageMs = ageMs ?? undefined
  const ns = getRowNamespace(row) ?? ''
  const name = String(row?.metadata?.name ?? '')
  const phase = String(row?.status?.phase ?? '')
  const node = String(row?.spec?.nodeName ?? '')
  const podIP = String(row?.status?.podIP ?? '')
  const hostIP = String(row?.status?.hostIP ?? '')
  row.__search = `${ns} ${name} ${phase} ${node} ${podIP} ${hostIP}`.toLowerCase()
  return row
}

function getPodPhaseText(row: any): string {
  const v = row?.status?.phase != null ? String(row.status.phase) : ''
  return v || '-'
}

function getPodPhaseQuickFilter(row: any): PodPhaseQuickFilter {
  const phase = getPodPhaseText(row)
  if (phase === 'Succeeded') return 'Completed'
  if (phase === 'Running' || phase === 'Pending' || phase === 'Failed') return phase
  return ''
}

function getPodPhaseTagType(row: any): 'success' | 'warning' | 'danger' | 'info' {
  const phase = getPodPhaseText(row)
  if (phase === 'Running') return 'success'
  if (phase === 'Pending') return 'warning'
  if (phase === 'Failed') return 'danger'
  if (phase === 'Succeeded') return 'info'
  return 'info'
}

watch(currentResource, (value) => {
  if (value !== 'pods' && podPhaseQuickFilter.value) {
    podPhaseQuickFilter.value = ''
  }
})

async function bulkDeletePods(rowsInput: any[] = selectedPodRows.value, options: { force?: boolean } = {}) {
  if (!clusterId.value || !canWriteK8s.value) return
  const rows = Array.isArray(rowsInput) ? [...rowsInput] : []
  if (rows.length === 0) return
  const force = options.force === true
  const podNames = rows
    .map((row) => {
      const ns = getRowNamespace(row)
      const name = String(row?.metadata?.name ?? '').trim()
      return ns && name ? `${ns}/${name}` : ''
    })
    .filter(Boolean)
  const previewNames = podNames.slice(0, 10)
  const hasMore = podNames.length > previewNames.length
  try {
    await ElMessageBox.confirm(
      `${force ? '将以 gracePeriodSeconds=0 立即删除，可能中断正在运行的进程。' : '确认删除以下 Pod？'}\n\n${previewNames.join('\n')}${hasMore ? `\n... 另外 ${podNames.length - previewNames.length} 个` : ''}`,
      force ? '强制删除 Pod' : '删除 Pod',
      {
        type: force ? 'error' : 'warning',
        confirmButtonText: force ? '确认强制删除' : '确认删除',
        cancelButtonText: '取消'
      }
    )
    deletingPods.value = true
    const results = await Promise.allSettled(
      rows.map((r) => {
        const ns = getRowNamespace(r)
        const name = String(r?.metadata?.name ?? '')
        if (!ns || !name) return Promise.resolve()
        return k8sApi.deletePod(clusterId.value!, ns, name, { force })
      })
    )

    const failed = results.filter((result): result is PromiseRejectedResult => result.status === 'rejected')
    const successCount = results.length - failed.length

    clearPodSelection()
    await loadCurrent()

    if (failed.length === 0) {
      notifySuccess(successCount > 0 ? `${force ? '已强制删除' : '已删除'} ${successCount} 个 Pod` : force ? '已强制删除' : '已删除')
      return
    }

    const firstError = failed[0]?.reason as ApiError | undefined
    const firstMessage = firstError?.message || `部分 Pod${force ? '强制' : ''}删除失败`
    if (successCount > 0) {
      notifyError(`${firstMessage}；${force ? '已强制删除' : '已删除'} ${successCount} 个 Pod，失败 ${failed.length} 个`)
      return
    }

    notifyError(firstError?.requestId ? `${firstMessage} (request_id=${firstError.requestId})` : firstMessage)
  } catch (e) {
    if (e === 'cancel') return
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  } finally {
    deletingPods.value = false
  }
}

function formatPorts(ports: any[]): string {
  return ports
    .map((p) => {
      const port = Number(p?.port ?? 0)
      const targetPort = p?.targetPort != null ? String(p.targetPort) : ''
      const protocol = String(p?.protocol ?? 'TCP')
      const name = p?.name ? String(p.name) : ''
      const suffix = targetPort ? `->${targetPort}` : ''
      return `${name ? `${name}:` : ''}${port}${suffix}/${protocol}`
    })
    .join(', ')
}

function formatSelector(sel: Record<string, string>): string {
  const entries = Object.entries(sel ?? {})
  if (entries.length === 0) return '-'
  return entries.map(([k, v]) => `${k}=${v}`).join(', ')
}

function getHosts(row: any): string[] {
  const rules: any[] = row?.spec?.rules ?? []
  return rules.map((r) => String(r?.host ?? '')).filter((h) => h)
}

function formatRules(row: any): string {
  const rules: any[] = row?.spec?.rules ?? []
  const parts: string[] = []
  for (const r of rules) {
    const host = String(r?.host ?? '')
    const paths: any[] = r?.http?.paths ?? []
    for (const p of paths) {
      const path = String(p?.path ?? '/')
      const svc = p?.backend?.service?.name ? String(p.backend.service.name) : '-'
      const port = p?.backend?.service?.port?.number != null ? String(p.backend.service.port.number) : '-'
      parts.push(`${host || '*'} ${path} -> ${svc}:${port}`)
    }
  }
  return parts.join(' | ') || '-'
}

function getDataKeys(row: any): string[] {
  return Object.keys(row?.data ?? {})
}

function formatClaimRef(row: any): string {
  const ref = row?.spec?.claimRef
  const ns = ref?.namespace != null ? String(ref.namespace) : ''
  const name = ref?.name != null ? String(ref.name) : ''
  if (!ns && !name) return '-'
  if (ns && name) return `${ns}/${name}`
  return ns || name
}

function getCustomResourceDefinitionSummary(row: any): string {
  const group = String(row?.spec?.group ?? '')
  const plural = String(row?.spec?.names?.plural ?? '')
  const versions = Array.isArray(row?.spec?.versions) ? row.spec.versions.map((it: any) => String(it?.name ?? '')).filter(Boolean).join(', ') : ''
  return [group && `${group}/${plural || '-'}`, versions && `versions: ${versions}`].filter(Boolean).join(' | ') || '-'
}

function getAPIServiceSummary(row: any): string {
  const group = String(row?.spec?.group ?? '')
  const version = String(row?.spec?.version ?? '')
  const svcNs = String(row?.spec?.service?.namespace ?? '')
  const svcName = String(row?.spec?.service?.name ?? '')
  const backend = svcName ? `${svcNs}/${svcName}` : 'local'
  return [`${group || 'core'}/${version || '-'}`, backend].join(' | ')
}

function getPriorityClassSummary(row: any): string {
  const value = row?.value != null ? String(row.value) : '-'
  const defaultFlag = row?.globalDefault ? 'default' : 'non-default'
  const policy = String(row?.preemptionPolicy ?? 'PreemptLowerPriority')
  return `value=${value} | ${defaultFlag} | ${policy}`
}

function getRuntimeClassSummary(row: any): string {
  return `handler=${String(row?.handler ?? '-')} | overhead=${Object.keys(row?.overhead?.podFixed ?? {}).length}`
}

function getCSIDriverSummary(row: any): string {
  const attachRequired = row?.spec?.attachRequired === true ? 'attach' : 'no-attach'
  const podInfo = row?.spec?.podInfoOnMount === true ? 'podInfo' : 'no-podInfo'
  const modes = Array.isArray(row?.spec?.volumeLifecycleModes) ? row.spec.volumeLifecycleModes.join(',') : '-'
  return `${attachRequired} | ${podInfo} | lifecycle=${modes}`
}

function getCSINodeSummary(row: any): string {
  const drivers: any[] = Array.isArray(row?.spec?.drivers) ? row.spec.drivers : []
  const first = drivers[0]
  return `drivers=${drivers.length} | first=${String(first?.name ?? '-')} | topo=${Object.keys(first?.topologyKeys ?? {}).length}`
}

function getCSIStorageCapacitySummary(row: any): string {
  return `class=${String(row?.storageClassName ?? '-')} | capacity=${String(row?.capacity ?? '-')} | max=${String(row?.maximumVolumeSize ?? '-')}`
}

function getWebhookConfigurationSummary(row: any): string {
  const webhooks: any[] = Array.isArray(row?.webhooks) ? row.webhooks : []
  const first = webhooks[0]
  const rules = Array.isArray(first?.rules) ? first.rules.length : 0
  const preview = first?.name ? String(first.name) : '-'
  return `webhooks=${webhooks.length} | first=${preview} | rules=${rules}`
}

function getValidatingAdmissionPolicySummary(row: any): string {
  const failure = String(row?.spec?.failurePolicy ?? '-')
  const params = String(row?.spec?.paramKind?.kind ?? '-')
  const validations = Array.isArray(row?.spec?.validations) ? row.spec.validations.length : 0
  return `failure=${failure} | params=${params} | validations=${validations}`
}

function getValidatingAdmissionPolicyBindingSummary(row: any): string {
  const policyName = String(row?.spec?.policyName ?? '-')
  const paramKind = String(row?.spec?.paramRef?.name ?? '-')
  const actions = Array.isArray(row?.spec?.validationActions) ? row.spec.validationActions.join(',') : '-'
  return `policy=${policyName} | param=${paramKind} | actions=${actions}`
}

function getNetworkPolicySummary(row: any): string {
  const types = Array.isArray(row?.spec?.policyTypes) ? row.spec.policyTypes.join(', ') : '-'
  const podSelector = formatSelector(row?.spec?.podSelector?.matchLabels ?? {})
  return `types=${types} | selector=${podSelector}`
}

function getEndpointsSummary(row: any): string {
  const subsets: any[] = Array.isArray(row?.subsets) ? row.subsets : []
  const addresses = subsets.reduce((sum, subset) => sum + (Array.isArray(subset?.addresses) ? subset.addresses.length : 0), 0)
  const notReady = subsets.reduce((sum, subset) => sum + (Array.isArray(subset?.notReadyAddresses) ? subset.notReadyAddresses.length : 0), 0)
  const ports = subsets.reduce((sum, subset) => sum + (Array.isArray(subset?.ports) ? subset.ports.length : 0), 0)
  return `ready=${addresses} | notReady=${notReady} | ports=${ports}`
}

function getEndpointSliceSummary(row: any): string {
  const endpoints: any[] = Array.isArray(row?.endpoints) ? row.endpoints : []
  const ports: any[] = Array.isArray(row?.ports) ? row.ports : []
  const serviceName = String(row?.metadata?.labels?.['kubernetes.io/service-name'] ?? '-')
  const addressType = String(row?.addressType ?? '-')
  return `service=${serviceName} | type=${addressType} | endpoints=${endpoints.length} | ports=${ports.length}`
}

function getReplicaSetSummary(row: any): string {
  const desired = Number(row?.status?.replicas ?? row?.spec?.replicas ?? 0)
  const ready = Number(row?.status?.readyReplicas ?? 0)
  const available = Number(row?.status?.availableReplicas ?? 0)
  return `ready=${ready}/${desired} | available=${available} | selector=${formatSelector(row?.spec?.selector?.matchLabels ?? {})}`
}

function getResourceQuotaSummary(row: any): string {
  const hard = row?.status?.hard ?? row?.spec?.hard ?? {}
  const keys = Object.keys(hard)
  return keys.length > 0 ? keys.slice(0, 4).join(', ') + (keys.length > 4 ? ` +${keys.length - 4}` : '') : '-'
}

function getVolumeSnapshotSummary(row: any): string {
  const spec = row?.spec ?? {}
  const status = row?.status ?? {}
  const pvc = String(spec?.source?.persistentVolumeClaimName ?? '-')
  const content = String(status?.boundVolumeSnapshotContentName ?? '-')
  const ready = status?.readyToUse === true ? 'yes' : status?.readyToUse === false ? 'no' : '-'
  const snapshotClass = String(spec?.volumeSnapshotClassName ?? '-')
  return `pvc=${pvc} | class=${snapshotClass} | ready=${ready} | content=${content}`
}

function getVolumeSnapshotClassSummary(row: any): string {
  const driver = String(row?.driver ?? '-')
  const deletionPolicy = String(row?.deletionPolicy ?? '-')
  return `driver=${driver} | policy=${deletionPolicy}`
}

function getVolumeSnapshotContentSummary(row: any): string {
  const spec = row?.spec ?? {}
  const handle = String(spec?.source?.snapshotHandle ?? '-')
  const snapshotRefNs = String(spec?.volumeSnapshotRef?.namespace ?? '-')
  const snapshotRefName = String(spec?.volumeSnapshotRef?.name ?? '-')
  const deletionPolicy = String(spec?.deletionPolicy ?? '-')
  return `handle=${handle} | snapshot=${snapshotRefNs}/${snapshotRefName} | policy=${deletionPolicy}`
}

function getLimitRangeSummary(row: any): string {
  const limits: any[] = Array.isArray(row?.spec?.limits) ? row.spec.limits : []
  const first = limits[0]
  const limitType = String(first?.type ?? '-')
  const defaults = Object.keys(first?.default ?? {})
  return `entries=${limits.length} | first=${limitType} | default=${defaults.join(', ') || '-'}`
}

function getVolumeAttachmentSummary(row: any): string {
  const nodeName = String(row?.spec?.nodeName ?? '-')
  const pvName = String(row?.spec?.source?.persistentVolumeName ?? '-')
  const attached = row?.status?.attached === true ? 'attached' : 'detached'
  return `${nodeName} | ${pvName} | ${attached}`
}

function getLeaseSummary(row: any): string {
  const spec = row?.spec ?? {}
  const holder = String(spec?.holderIdentity ?? '-')
  const duration = Number(spec?.leaseDurationSeconds ?? 0)
  const renew = String(spec?.renewTime ?? '-')
  return `holder=${holder} | duration=${duration || '-'} | renew=${renew}`
}

function openScale(row: any) {
  const ns = getRowNamespace(row)
  if (!clusterId.value || !ns) return
  const kind = String(row?.kind ?? workloadKind.value)
  const name = String(row?.metadata?.name ?? '')
  if (!name) return
  const desired = getDesired(row)
  const available = getWorkloadAvailable(row)
  detailsHostRef.value?.openScale({ kind, namespace: ns, name }, desired, available)
}

async function restartWorkloadRow(row: any) {
  const ns = getRowNamespace(row)
  if (!clusterId.value || !ns) return
  const req = {
    kind: String(row?.kind ?? workloadKind.value),
    namespace: ns,
    name: String(row?.metadata?.name ?? '')
  }
  if (!req.name) return
  try {
    await ElMessageBox.confirm(
      `确认滚动重启 ${req.kind} ${req.namespace}/${req.name}？\n\n重启将触发 Pod 滚动更新，可能会短暂影响服务可用性。`,
      '提示',
      { type: 'warning', confirmButtonText: '确认重启', cancelButtonText: '取消' }
    )
    await k8sApi.restartWorkload(clusterId.value, req)
    notifySuccess('已提交重启')
    // 自动刷新列表以跟踪状态变化
    setTimeout(() => void loadCurrent(), 1500)
  } catch (e) {
    if (e === 'cancel') return
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  }
}

async function restartSelectedWorkloads(rows: any[]) {
  if (!clusterId.value) return
  const requests = (Array.isArray(rows) ? rows : [])
    .map((row) => {
      const namespace = getRowNamespace(row)
      const name = String(row?.metadata?.name ?? '').trim()
      const kind = String(row?.kind ?? workloadKind.value ?? '').trim() || workloadKind.value
      if (!namespace || !name || !kind) return null
      return { kind, namespace, name }
    })
    .filter((item): item is { kind: string; namespace: string; name: string } => Boolean(item))

  if (requests.length === 0) return

  const preview = requests.slice(0, 8).map((item) => `${item.kind} ${item.namespace}/${item.name}`).join('\n')
  const suffix = requests.length > 8 ? `\n... 以及其余 ${requests.length - 8} 项` : ''

  try {
    await ElMessageBox.confirm(
      `确认批量滚动重启以下 ${requests.length} 个工作负载？\n\n${preview}${suffix}`,
      '批量重启确认',
      { type: 'warning', confirmButtonText: '确认批量重启', cancelButtonText: '取消' }
    )
  } catch {
    return
  }

  const results = await Promise.allSettled(requests.map((req) => k8sApi.restartWorkload(clusterId.value!, req)))
  const failed = results.filter((result) => result.status === 'rejected')
  const succeeded = requests.length - failed.length

  workloadsPanelRef.value?.clearSelection?.()

  if (failed.length === 0) {
    notifySuccess(`已提交 ${succeeded} 个工作负载重启`)
  } else {
    notifyError(`已提交 ${succeeded} 个工作负载重启，失败 ${failed.length} 个`)
  }

  setTimeout(() => void loadCurrent(), 1500)
}

async function updateWorkloadImage(payload: { kind: string; namespace: string; name: string; container: string; image: string }) {
  if (!clusterId.value) return
  try {
    await k8sApi.updateWorkloadImage(clusterId.value, payload)
    notifySuccess(`已提交 ${payload.name} 镜像更新`)
    setTimeout(() => void loadCurrent(), 1500)
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
    throw err
  }
}

async function toggleWorkloadPaused(row: any) {
  if (!clusterId.value) return
  const namespace = getRowNamespace(row)
  const name = String(row?.metadata?.name ?? '').trim()
  if (!namespace || !name) return
  const paused = !Boolean(row?.spec?.paused)
  const actionText = paused ? '暂停' : '恢复'
  try {
    await ElMessageBox.confirm(
      `确认${actionText} Deployment ${namespace}/${name} 的 Rollout？`,
      '提示',
      { type: 'warning', confirmButtonText: `确认${actionText}`, cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  try {
    await k8sApi.updateWorkloadPaused(clusterId.value, { kind: 'Deployment', namespace, name, paused })
    notifySuccess(paused ? '已暂停 Rollout' : '已恢复 Rollout')
    setTimeout(() => void loadCurrent(), 1200)
  } catch (e) {
    const err = e as ApiError
    notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
  }
}

type EditorTheme = 'auto' | 'light' | 'dark'
const EDITOR_THEME_KEY = 'k8s-platform:viewer-theme:cluster-manage'
const editorTheme = ref<EditorTheme>((localStorage.getItem(EDITOR_THEME_KEY) as EditorTheme) || 'auto')
const editorThemeEffectiveDark = computed(() => {
  if (editorTheme.value === 'dark') return true
  if (editorTheme.value === 'light') return false
  return document.documentElement.classList.contains('dark')
})

function toggleEditorTheme() {
  editorTheme.value = editorThemeEffectiveDark.value ? 'light' : 'dark'
  localStorage.setItem(EDITOR_THEME_KEY, editorTheme.value)
}

function openYaml(meta: string, loader: () => Promise<{ text: string }>, saver?: (text: string) => Promise<void>) {
  detailsHostRef.value?.openYaml(meta, loader, saver)
}

function openCreateNamespace() {
  runOverlayWhenReady('workbenches', () => workbenchesRef.value, (target) => target.openCreateNamespace())
}

function openServiceDetail(row: any) {
  detailsHostRef.value?.openServiceDetail(row)
}

function openIngressDetail(row: any) {
  detailsHostRef.value?.openIngressDetail(row)
}

function openIngressClassDetail(row: any) {
  detailsHostRef.value?.openIngressClassDetail(row)
}

function openPVCDetail(row: any) {
  detailsHostRef.value?.openPVCDetail(row)
}

function openPVDetail(row: any) {
  detailsHostRef.value?.openPVDetail(row)
}

function openRelatedPodDetail(row: any) {
  detailsHostRef.value?.openPodDetail(decoratePodRow(row))
}

function openPodDetailFromExternal(row: any) {
  if (row) {
    detailsHostRef.value?.openPodDetail(decoratePodRow(row))
  }
}

function openRelatedFromJob(payload: { action: string; kind?: string; name: string; namespace?: string }) {
  const kind = payload.kind ?? ''
  const ns = payload.namespace ?? ''
  const name = payload.name
  if (kind === 'ConfigMap') {
    const row = findListRowByNsName(ns, name)
    if (row) detailsHostRef.value?.openConfigMapDetail(row)
  } else if (kind === 'Secret') {
    const row = findListRowByNsName(ns, name)
    if (row) detailsHostRef.value?.openSecretDetail(row)
  }
}

function openConfigMapDetail(row: any) {
  detailsHostRef.value?.openConfigMapDetail(row)
}

function openSecretDetail(row: any) {
  detailsHostRef.value?.openSecretDetail(row)
}

async function openSecretReveal(row: any) {
  if (!clusterId.value) return
  const ns = getRowNamespace(row)
  const name = String(row?.metadata?.name ?? '').trim()
  if (!ns || !name) return
  try {
    await ElMessageBox.confirm(
      `即将查看 Secret 明文内容，并写入审计日志。\n\n目标：${ns}/${name}`,
      '查看内容确认',
      {
        confirmButtonText: '确认查看',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }
  openYaml(`cluster=${clusterId.value}  Secret 明文  ${ns}/${name}`, () => k8sApi.getSecretReveal(clusterId.value, ns, name))
}

function openJobDetail(row: any) {
  detailsHostRef.value?.openJobDetail(row)
}

function openCronJobDetail(row: any) {
  detailsHostRef.value?.openCronJobDetail(row)
}

function getNamespacedBatchMeta(row: any): { namespace: string; name: string } | null {
  const namespaceText = getRowNamespace(row)
  const nameText = String(row?.metadata?.name ?? '').trim()
  if (!namespaceText || !nameText) return null
  return { namespace: namespaceText, name: nameText }
}

function isFinishedJob(row: any): boolean {
  const active = Number(row?.status?.active ?? 0)
  const succeeded = Number(row?.status?.succeeded ?? 0)
  const failed = Number(row?.status?.failed ?? 0)
  if (active > 0) return false
  if (succeeded > 0 || failed > 0) return true
  const conditions = Array.isArray(row?.status?.conditions) ? row.status.conditions : []
  return conditions.some((cond: any) => {
    const type = String(cond?.type ?? '')
    return (type === 'Complete' || type === 'Failed') && String(cond?.status ?? '') === 'True'
  })
}

function notifyActionError(error: unknown) {
  if (error === 'cancel') return
  const err = error as ApiError
  notifyError(err.requestId ? `${err.message} (request_id=${err.requestId})` : err.message)
}

async function triggerCronJob(row: any) {
  if (!clusterId.value) return
  const meta = getNamespacedBatchMeta(row)
  if (!meta) return
  try {
    await ElMessageBox.confirm(`确认立即执行 CronJob ${meta.namespace}/${meta.name}？`, '立即执行', {
      type: 'warning',
      confirmButtonText: '立即执行',
      cancelButtonText: '取消'
    })
    const result = await k8sApi.triggerCronJob(clusterId.value, meta.namespace, meta.name)
    notifySuccess(`已创建 Job ${meta.namespace}/${result.job_name}`)
    await loadCurrent()
  } catch (error) {
    notifyActionError(error)
  }
}

async function toggleCronJobSuspend(row: any) {
  if (!clusterId.value) return
  const meta = getNamespacedBatchMeta(row)
  if (!meta) return
  const suspend = !Boolean(row?.spec?.suspend)
  try {
    await ElMessageBox.confirm(
      suspend ? `确认暂停 CronJob ${meta.namespace}/${meta.name} 的调度？` : `确认恢复 CronJob ${meta.namespace}/${meta.name} 的调度？`,
      suspend ? '暂停调度' : '恢复调度',
      {
        type: suspend ? 'warning' : 'info',
        confirmButtonText: suspend ? '确认暂停' : '确认恢复',
        cancelButtonText: '取消'
      }
    )
    await k8sApi.suspendCronJob(clusterId.value, meta.namespace, meta.name, suspend)
    notifySuccess(suspend ? '已暂停 CronJob 调度' : '已恢复 CronJob 调度')
    await loadCurrent()
  } catch (error) {
    notifyActionError(error)
  }
}

async function cleanCompletedJobs() {
  if (!clusterId.value) return
  const selectedNamespaces = getNamespaceFilter(namespace.value, ALL_NAMESPACE)
  if (selectedNamespaces && selectedNamespaces.length > 1) {
    notifyError('批量清理仅支持单个 namespace 或全部 namespace 视图')
    return
  }
  try {
    const { value } = await ElMessageBox.prompt(
      '请输入保留小时数，系统会删除早于该阈值的已完成 Job。输入 0 表示立即清理所有已完成 Job。',
      '批量清理已完成 Job',
      {
        inputValue: '24',
        inputPattern: /^(0|[1-9]\d{0,3})$/,
        inputErrorMessage: '请输入 0 到 9999 之间的整数',
        confirmButtonText: '开始清理',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    const olderThanHours = Number(value)
    const namespaceFilter = selectedNamespaces && selectedNamespaces.length === 1 ? selectedNamespaces[0] : undefined
    const result = await k8sApi.cleanCompletedJobs(clusterId.value, {
      namespace: namespaceFilter,
      older_than_hours: olderThanHours
    })
    notifySuccess(`已清理 ${result.deleted_count} 个 Job`)
    await loadCurrent()
  } catch (error) {
    notifyActionError(error)
  }
}

function openResourceTopology(payload: { mode: string; namespace?: string; name: string }) {
  if (!clusterId.value || !payload?.name) return
  router.push({
    name: 'K8sResourceTopology',
    query: {
      clusterId: String(clusterId.value),
      mode: payload.mode,
      namespace: payload.namespace,
      name: payload.name
    }
  })
}

function topologyTreeNodeId(kind?: string): string | null {
  if (!kind) return null
  if (kind === 'Pod') return 'workloads:pods'
  if (kind === 'Service') return 'network:services'
  if (kind === 'Ingress') return 'network:ingresses'
  if (kind === 'EndpointSlice') return 'network:endpointslices'
  if (kind === 'Endpoints') return 'network:endpoints'
  if (kind === 'PVC' || kind === 'PersistentVolumeClaim') return 'storage:pvcs'
  if (kind === 'PV' || kind === 'PersistentVolume') return 'storage:pvs'
  if (kind === 'StorageClass') return 'storage:storageclasses'
  if (kind === 'VolumeAttachment') return 'storage:volumeattachments'
  if (kind === 'Node') return 'cluster:nodes'
  if (kind === 'ReplicaSet') return 'workloads:replicasets'
  if (kind === 'Deployment') return 'workloads:deployments'
  if (kind === 'ConfigMap') return 'config:configmaps'
  if (kind === 'Secret') return 'config:secrets'
  if (kind === 'CSINode') return 'storage:csinodes'
  return null
}

async function handleTopologyNavigationQuery() {
  const targetKind = typeof route.query.targetKind === 'string' ? route.query.targetKind : ''
  const targetName = typeof route.query.targetName === 'string' ? route.query.targetName : ''
  const targetNamespace = typeof route.query.targetNamespace === 'string' ? route.query.targetNamespace : ''
  if (!targetKind || !targetName) return
  const nodeId = topologyTreeNodeId(targetKind)
  if (!nodeId) return
  if (targetNamespace) namespace.value = [targetNamespace]
  keywordInput.value = targetName
  onDashboardNavigate(nodeId)
  await nextTick()
  const row = list.value.find((it: any) => {
    const name = String(it?.metadata?.name ?? '')
    const ns = String(it?.metadata?.namespace ?? '')
    return name === targetName && (!targetNamespace || ns === targetNamespace)
  })
  if (row) {
    if (targetKind === 'Pod') openPodDetail(row)
    else if (targetKind === 'Service') openServiceDetail(row)
    else if (targetKind === 'Ingress') openIngressDetail(row)
    else if (targetKind === 'PVC' || targetKind === 'PersistentVolumeClaim') openPVCDetail(row)
    else if (targetKind === 'PV' || targetKind === 'PersistentVolume') openPVDetail(row)
    else if (targetKind === 'Node') openNodeDetail(row)
    else if (targetKind === 'ConfigMap') openConfigMapDetail(row)
    else if (targetKind === 'Secret') openSecretDetail(row)
    else if (targetKind === 'Deployment') openDeploymentDetail(row)
    else if (targetKind === 'ReplicaSet') openYaml(`cluster=${clusterId.value}  ${targetNamespace}/${targetName}`, () => k8sApi.getReplicaSetYaml(clusterId.value, targetNamespace, targetName))
    else if (targetKind === 'ServiceAccount') openYaml(`cluster=${clusterId.value}  ${targetNamespace}/${targetName}`, () => k8sApi.getServiceAccountYaml(clusterId.value, targetNamespace, targetName))
    else if (targetKind === 'StorageClass') openYaml(`cluster=${clusterId.value}  ${targetName}`, () => k8sApi.getStorageClassYaml(clusterId.value, targetName))
    else if (targetKind === 'VolumeAttachment') openYaml(`cluster=${clusterId.value}  ${targetName}`, () => k8sApi.getVolumeAttachmentYaml(clusterId.value, targetName))
    else if (targetKind === 'EndpointSlice') openYaml(`cluster=${clusterId.value}  ${targetNamespace}/${targetName}`, () => k8sApi.getEndpointSliceYaml(clusterId.value, targetNamespace, targetName))
    else if (targetKind === 'Endpoints') openYaml(`cluster=${clusterId.value}  ${targetNamespace}/${targetName}`, () => k8sApi.getEndpointsYaml(clusterId.value, targetNamespace, targetName))
    else if (targetKind === 'CSINode') openYaml(`cluster=${clusterId.value}  ${targetName}`, () => k8sApi.getCSINodeYaml(clusterId.value, targetName))
    else if (targetKind === 'RoleBinding') openYaml(`cluster=${clusterId.value}  ${targetNamespace}/${targetName}`, () => k8sApi.getRoleBindingYaml(clusterId.value, targetNamespace, targetName))
    else if (targetKind === 'Role') openYaml(`cluster=${clusterId.value}  ${targetNamespace}/${targetName}`, () => k8sApi.getRoleYaml(clusterId.value, targetNamespace, targetName))
    else if (targetKind === 'ClusterRoleBinding') openYaml(`cluster=${clusterId.value}  ${targetName}`, () => k8sApi.getClusterRoleBindingYaml(clusterId.value, targetName))
    else if (targetKind === 'ClusterRole') openYaml(`cluster=${clusterId.value}  ${targetName}`, () => k8sApi.getClusterRoleYaml(clusterId.value, targetName))
  }
  const nextQuery = { ...route.query }
  delete nextQuery.targetKind
  delete nextQuery.targetName
  delete nextQuery.targetNamespace
  await router.replace({ query: nextQuery })
}


function openPodExec(row: any) {
  runOverlayWhenReady('workbenches', () => workbenchesRef.value, (target) => target.openPodExec(row))
}

function openPodLogs(row: any) {
  runOverlayWhenReady('workbenches', () => workbenchesRef.value, (target) => target.openPodLogs(row))
}

function openEditJob(row: any) {
  runOverlayWhenReady('resourceEditors', () => resourceEditorsRef.value, (target) => target.openEditJob(row))
}

function openEditCronJob(row: any) {
  runOverlayWhenReady('resourceEditors', () => resourceEditorsRef.value, (target) => target.openEditCronJob(row))
}

function openEditService(row: any) {
  runOverlayWhenReady('resourceEditors', () => resourceEditorsRef.value, (target) => target.openEditService(row))
}

function openEditIngress(row: any) {
  runOverlayWhenReady('resourceEditors', () => resourceEditorsRef.value, (target) => target.openEditIngress(row))
}

function openEditIngressClass(row: any) {
  runOverlayWhenReady('resourceEditors', () => resourceEditorsRef.value, (target) => target.openEditIngressClass(row))
}

function openEditConfigMap(row: any) {
  runOverlayWhenReady('resourceEditors', () => resourceEditorsRef.value, (target) => target.openEditConfigMap(row))
}

function openEditSecret(row: any) {
  runOverlayWhenReady('resourceEditors', () => resourceEditorsRef.value, (target) => target.openEditSecret(row))
}

function findListRowByNsName(ns: string, name: string): any | null {
  const n = String(ns ?? '').trim()
  const nm = String(name ?? '').trim()
  if (!n || !nm) return null
  const arr = (list.value ?? []) as any[]
  for (const it of arr) {
    const itNs = getRowNamespace(it)
    const itName = String(it?.metadata?.name ?? '')
    if (itNs === n && itName === nm) return it
  }
  return null
}

function onPodDetailOpenLogs(v: { ns: string; name: string; container?: string }) {
  workbenchesRef.value?.openPodLogsTarget(v)
}

function onPodDetailOpenExec(v: { row: any; container?: string }) {
  workbenchesRef.value?.openPodExec(v.row, v.container)
}

function onPodDetailOpenRelated(v: { action: string; kind?: string; name: string; namespace?: string }) {
  if (!clusterId.value) return
  const action = v?.action
  const kind = String(v?.kind ?? '').trim()
  const name = String(v?.name ?? '').trim()
  const ns = v?.namespace ? String(v.namespace).trim() : ''
  if (!action || !name) return

  if (action === 'owner') {
    if (!kind || !ns) return
    if (kind === 'ReplicaSet') {
      openYaml(`cluster=${clusterId.value}  ${kind}  ${ns}/${name}`, () => k8sApi.getReplicaSetYaml(clusterId.value, ns, name))
      return
    }
    if (kind === 'Job') {
      openYaml(`cluster=${clusterId.value}  ${kind}  ${ns}/${name}`, () => k8sApi.getJobYaml(clusterId.value, ns, name))
      return
    }
    if (kind === 'CronJob') {
      openYaml(`cluster=${clusterId.value}  ${kind}  ${ns}/${name}`, () => k8sApi.getCronJobYaml(clusterId.value, ns, name))
      return
    }
    openYaml(`cluster=${clusterId.value}  ${kind}  ${ns}/${name}`, () =>
      k8sApi.getWorkloadYaml(clusterId.value, { kind, namespace: ns, name })
    )
    return
  }
  if (action === 'configmap') {
    if (!ns) return
    openYaml(`cluster=${clusterId.value}  ${ns}/${name}`, () => k8sApi.getConfigMapYaml(clusterId.value, ns, name))
    return
  }
  if (action === 'secret') {
    if (!ns) return
    openYaml(`cluster=${clusterId.value}  ${ns}/${name}`, () => k8sApi.getSecretYaml(clusterId.value, ns, name))
    return
  }
  if (action === 'service') {
    if (!ns) return
    openYaml(`cluster=${clusterId.value}  ${ns}/${name}`, () => k8sApi.getServiceYaml(clusterId.value, ns, name))
    return
  }
  if (action === 'ingress') {
    if (!ns) return
    openYaml(`cluster=${clusterId.value}  ${ns}/${name}`, () => k8sApi.getIngressYaml(clusterId.value, ns, name))
    return
  }
  if (action === 'pvc') {
    if (!ns) return
    openYaml(`cluster=${clusterId.value}  ${ns}/${name}`, () => k8sApi.getPVCYaml(clusterId.value, ns, name))
    return
  }
  if (action === 'pv') {
    openYaml(`cluster=${clusterId.value}  ${name}`, () => k8sApi.getPVYaml(clusterId.value, name))
  }
}

function openPodDetail(row: any) {
  if (!clusterId.value) return
  const ns = getRowNamespace(row)
  const name = String(row?.metadata?.name ?? '')
  if (!ns || !name) return
  detailsHostRef.value?.openPodDetail(decoratePodRow(row))
}

function openNodeDetail(row: any) {
  if (!clusterId.value) return
  const name = String(row?.metadata?.name ?? '')
  if (!name) return
  detailsHostRef.value?.openNodeDetail(row)
}

function openDeploymentDetail(row: any) {
  if (!clusterId.value) return
  const ns = getRowNamespace(row)
  const name = String(row?.metadata?.name ?? '')
  if (!ns || !name) return
  detailsHostRef.value?.openDeploymentDetail(row, 'Deployment')
}

function normalizeMultilineText(input: string): string {
  let text = String(input ?? '')
  if (!text) return ''
  text = text.replace(/\r\n/g, '\n')
  const quoted = (text.startsWith('"') && text.endsWith('"')) || (text.startsWith("'") && text.endsWith("'"))
  if (quoted && text.includes('\\n')) {
    text = text.slice(1, -1)
  }
  const hasRealNewline = text.includes('\n')
  const hasEscapedNewline = text.includes('\\n')
  if (!hasRealNewline && hasEscapedNewline) {
    text = text.replace(/\\r\\n/g, '\n').replace(/\\n/g, '\n').replace(/\\t/g, '\t')
  }
  return text
}

watch(
  () => clusterId.value,
  async (v) => {
    if (!v) {
      await router.replace('/clusters')
      return
    }
    loadDashboardPrefs()
    await refreshAll()
    await handleTopologyNavigationQuery()
  },
  { immediate: true }
)

watch(
  () => namespace.value.slice().sort().join('|'),
  () => {
    if (loadingTree.value) return
    if (!showNamespaceSelect.value) return
    void loadCurrent()
  }
)

onBeforeUnmount(() => {
  stopTimers()
})
</script>

<style scoped>
.k8s-shell {
  display: flex;
  height: 100%;
  min-height: 0;
  width: 100%;
  gap: 12px;
  padding: 12px;
  background: var(--color-bg-page, #f8fafc);
}
:global(html.dark) .k8s-shell {
  background: var(--color-bg-page, #0f172a);
}

.k8s-aside {
  display: flex;
  width: 236px;
  flex-direction: column;
  overflow: hidden;
  border-radius: 12px;
  border: 1px solid var(--color-border-subtle);
  background: var(--color-bg-card);
  backdrop-filter: blur(20px);
  --k8s-tree-item-height: 34px;
  --k8s-tree-arrow-size: 14px;
  --k8s-tree-icon-box: 18px;
  --el-tree-node-content-height: var(--k8s-tree-item-height);
  flex-shrink: 0;
}
:global(html.dark) .k8s-aside {
  background: rgba(15, 23, 42, 0.65);
  border-color: var(--color-border-default);
}

.aside-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--color-border-subtle);
}
:global(html.dark) .aside-head {
  border-bottom-color: var(--color-border-default);
}

.aside-title-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.aside-back-btn {
  display: inline-flex;
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.aside-back-btn:hover {
  background: rgba(59, 130, 246, 0.08);
  color: var(--color-accent-primary);
}

.aside-title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-primary);
  padding-left: 2px;
}

.aside-body {
  flex: 1;
  min-height: 0;
  padding: 6px;
}

.tree-node {
  display: flex;
  min-width: 0;
  min-height: var(--k8s-tree-item-height);
  align-items: center;
  gap: 6px;
  width: 100%;
  padding-right: 6px;
  font-size: 13px;
  line-height: 1;
  letter-spacing: 0.01em;
  color: var(--color-text-secondary);
}

.tree-node--folder {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.tree-node--leaf {
  color: var(--color-text-secondary);
}

.tree-icon {
  color: var(--color-text-muted);
  font-size: 16px;
  width: var(--k8s-tree-icon-box);
  height: var(--k8s-tree-icon-box);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.tree-icon--folder {
  font-size: 16px;
  color: var(--color-accent-primary);
}

.tree-icon--leaf {
  opacity: 0.88;
}

.tree-icon-img {
  width: var(--k8s-tree-icon-box);
  height: var(--k8s-tree-icon-box);
  display: inline-block;
  flex-shrink: 0;
  object-fit: contain;
  vertical-align: middle;
  opacity: 0.85;
}

.tree-icon-img--folder {
  opacity: 0.96;
}

.tree-icon-img--leaf {
  opacity: 0.82;
}

.tree-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.2;
}

/* sidebar tree item styling */
.k8s-aside :deep(.el-tree-node__content) {
  min-height: var(--k8s-tree-item-height);
  border-radius: 6px;
  margin: 0;
  padding-left: 6px !important;
  display: flex;
  align-items: center;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}

.k8s-aside :deep(.el-tree > .el-tree-node > .el-tree-node__content) {
  margin: 2px 0;
  border-radius: 8px;
  background: rgba(59, 130, 246, 0.045);
}

.k8s-aside :deep(.el-tree-node__children) {
  margin: 0 0 4px 12px;
  padding-left: 8px;
  border-left: 1px solid rgba(148, 163, 184, 0.18);
}

.k8s-aside :deep(.el-tree-node__children .el-tree-node__content) {
  min-height: var(--k8s-tree-item-height);
  padding-left: 4px !important;
}

.k8s-aside :deep(.el-tree-node__expand-icon) {
  width: var(--k8s-tree-arrow-size);
  min-width: var(--k8s-tree-arrow-size);
  height: var(--k8s-tree-arrow-size);
  margin-right: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: 12px;
  transform: none;
}

.k8s-aside :deep(.el-tree-node__expand-icon.expanded) {
  color: var(--color-accent-primary);
  transform: rotate(90deg);
}

.k8s-aside :deep(.el-tree-node__expand-icon.is-leaf) {
  opacity: 0;
}

.k8s-aside :deep(.el-tree-node__content:hover) {
  background: rgba(59, 130, 246, 0.06);
}

.tree-node--active {
  color: var(--color-accent-primary);
  font-weight: 600;
}

.tree-node--leaf.tree-node--active {
  background: transparent;
  padding: 0;
}

.tree-node--folder.tree-node--active {
  min-height: calc(var(--k8s-tree-item-height) - 4px);
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(59, 130, 246, 0.09);
}

:global(html.dark) .tree-node--folder.tree-node--active {
  background: rgba(59, 130, 246, 0.16);
}

:global(html.dark) .k8s-aside :deep(.el-tree > .el-tree-node > .el-tree-node__content) {
  background: rgba(59, 130, 246, 0.08);
}

:global(html.dark) .k8s-aside :deep(.el-tree-node__children) {
  border-left-color: rgba(148, 163, 184, 0.2);
}

.k8s-main {
  min-width: 0;
  flex: 1;
}

.page-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 12px;
  border-color: var(--color-border-subtle);
  box-shadow: none;
}

.page-card :deep(> .el-card__body) {
  flex: 1;
  min-height: 0;
  overflow: auto;
  overflow-x: hidden;
  padding: 16px;
}

.page-card :deep(> .el-card__header) {
  padding: 14px 16px;
}

/* ── Table header: unified style ── */
.page-card :deep(.el-table__header-wrapper),
.page-card :deep(.el-table__fixed-header-wrapper) {
  background: var(--table-header-bg, #e2e8f0) !important;
}

:global(html.dark) .page-card :deep(.el-table__header-wrapper),
:global(html.dark) .page-card :deep(.el-table__fixed-header-wrapper) {
  background: var(--table-header-bg, rgba(15, 23, 42, 0.6));
}

.page-card :deep(.el-table__header-wrapper th.el-table__cell) {
  background: transparent !important;
  border-bottom: none !important;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--table-header-text, #1e293b) !important;
  padding: 0 12px;
  height: 38px;
  line-height: 38px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Header cell inner .cell — keep default display, just control overflow */
.page-card :deep(.el-table__header-wrapper th .cell) {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0;
  line-height: 38px;
  vertical-align: middle;
}

/* Sort caret wrapper — compact, vertically centered */
.page-card :deep(.el-table__header-wrapper .caret-wrapper) {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 14px;
  width: 16px;
  vertical-align: middle;
  position: relative;
  overflow: visible;
  cursor: pointer;
  margin-left: 4px;
}

/* Override default absolute positioning — use tight spacing */
.page-card :deep(.el-table__header-wrapper .sort-caret) {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  width: 0;
  height: 0;
  border: solid transparent;
}

.page-card :deep(.el-table__header-wrapper .sort-caret.ascending) {
  top: 0;
  border-width: 0 4px 5px;
  border-bottom-color: rgba(59, 130, 246, 0.3);
}

.page-card :deep(.el-table__header-wrapper .sort-caret.descending) {
  bottom: 0;
  border-width: 5px 4px 0;
  border-top-color: rgba(59, 130, 246, 0.3);
}

/* Active sort state — solid blue */
.page-card :deep(.el-table__header-wrapper .ascending .sort-caret.ascending) {
  border-bottom-color: #3b82f6;
}

.page-card :deep(.el-table__header-wrapper .descending .sort-caret.descending) {
  border-top-color: #3b82f6;
}

.page-card :deep(.el-table__header-wrapper tr) {
  background: transparent !important;
}

:global(html.dark) .page-card :deep(.el-table__header-wrapper th.el-table__cell) {
  background: transparent !important;
  color: var(--table-header-text, #e2e8f0) !important;
}

/* Fixed-left / fixed-right header: use unified token */
.page-card :deep(.el-table__fixed-left .el-table__header-wrapper),
.page-card :deep(.el-table__fixed-right .el-table__header-wrapper) {
  background: var(--table-header-bg, #eef2f7);
}

:global(html.dark) .page-card :deep(.el-table__fixed-left .el-table__header-wrapper),
:global(html.dark) .page-card :deep(.el-table__fixed-right .el-table__header-wrapper) {
  background: var(--table-header-bg, rgba(15, 23, 42, 0.6));
}

/* Fixed-right-patch (corner fill) */
.page-card :deep(.el-table__fixed-right-patch) {
  background: var(--table-header-bg, #eef2f7) !important;
  border-bottom: none !important;
}

:global(html.dark) .page-card :deep(.el-table__fixed-right-patch) {
  background: var(--table-header-bg, rgba(15, 23, 42, 0.6)) !important;
  border-bottom: none !important;
}

/* Ensure fixed header cells inherit unified style */
.page-card :deep(.el-table__fixed-right .el-table__header-wrapper th.el-table__cell),
.page-card :deep(.el-table__fixed-left .el-table__header-wrapper th.el-table__cell) {
  background: transparent !important;
  border-bottom: none !important;
  color: var(--table-header-text, #1e293b) !important;
}

:global(html.dark) .page-card :deep(.el-table__fixed-right .el-table__header-wrapper th.el-table__cell),
:global(html.dark) .page-card :deep(.el-table__fixed-left .el-table__header-wrapper th.el-table__cell) {
  color: var(--table-header-text, #e2e8f0) !important;
}

/* Header bottom line — removed */
.page-card :deep(.el-table__header-wrapper) {
  border-bottom: none;
}

:global(html.dark) .page-card :deep(.el-table__header-wrapper) {
  border-bottom: none;
}

/* Gutter cell (scrollbar placeholder in header) */
.page-card :deep(.el-table__header-wrapper th.el-table__cell.gutter) {
  background: var(--table-header-bg, #eef2f7) !important;
  border-bottom: none !important;
}

:global(html.dark) .page-card :deep(.el-table__header-wrapper th.el-table__cell.gutter) {
  background: rgba(25,35,50,0.95) !important;
}

/* Eliminate any default Element Plus header border-bottom on table */
.page-card :deep(.el-table__header-wrapper .el-table__header) {
  border-bottom: none;
}

.page-card :deep(.el-table--border .el-table__header-wrapper th.el-table__cell) {
  border-right-color: rgba(0, 0, 0, 0.06);
}

/* ── Table row: clear hover, readable rows ── */
.page-card :deep(.el-table) {
  --el-table-tr-bg-color: #fff;
  --el-table-row-hover-bg-color: rgba(59, 130, 246, 0.07);
  --el-table-current-row-bg-color: rgba(59, 130, 246, 0.09);
}

/* disable stripe — use alternating borders instead */
.page-card :deep(.el-table--striped .el-table__body tr.el-table__row--striped td.el-table__cell) {
  background: var(--table-stripe-bg, #f8fafc);
}

.page-card :deep(.el-table__body tr) {
  transition: background-color 0.15s ease;
}

/* visible row dividers */
.page-card :deep(.el-table td.el-table__cell) {
  border-bottom: 1px solid rgba(0, 0, 0, 0.07);
}

:global(html.dark) .page-card :deep(.el-table td.el-table__cell) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

:global(html.dark) .page-card :deep(.el-table) {
  --el-table-tr-bg-color: rgba(2, 6, 23, 0.92);
  --el-table-row-hover-bg-color: rgba(59, 130, 246, 0.1);
  --el-table-current-row-bg-color: rgba(59, 130, 246, 0.13);
}

:global(html.dark) .page-card :deep(.el-table--striped .el-table__body tr.el-table__row--striped td.el-table__cell) {
  background: var(--table-stripe-bg, rgba(255, 255, 255, 0.03));
}

.page-card :deep(.el-table__body tr > td.el-table__cell),
.page-card :deep(.el-table__fixed-body-wrapper tr > td.el-table__cell) {
  background-color: var(--el-table-tr-bg-color);
}

.page-card :deep(.el-table__body tr:hover > td.el-table__cell),
.page-card :deep(.el-table__fixed-body-wrapper tr:hover > td.el-table__cell) {
  background-color: var(--el-table-row-hover-bg-color);
}

.page-card :deep(.el-table__body-wrapper) {
  background: #fff;
}

:global(html.dark) .page-card :deep(.el-table__body-wrapper) {
  background: rgba(2, 6, 23, 0.92);
}

.page-card :deep(.el-table__fixed-left),
.page-card :deep(.el-table__fixed-right) {
  background: #fff;
}

:global(html.dark) .page-card :deep(.el-table__fixed-left),
:global(html.dark) .page-card :deep(.el-table__fixed-right) {
  background: rgba(2, 6, 23, 0.92);
}

.page-card :deep(.el-table__fixed-left .el-table__fixed-body-wrapper),
.page-card :deep(.el-table__fixed-right .el-table__fixed-body-wrapper) {
  background: #fff;
}

:global(html.dark) .page-card :deep(.el-table__fixed-left .el-table__fixed-body-wrapper),
:global(html.dark) .page-card :deep(.el-table__fixed-right .el-table__fixed-body-wrapper) {
  background: rgba(2, 6, 23, 0.92);
}

.columns-panel {
  width: 260px;
  padding: 12px;
}

.columns-panel-empty {
  padding-top: 4px;
  padding-bottom: 4px;
  font-size: 12px;
  color: #64748b;
}

:global(.k8s-columns-popper) {
  z-index: 4000 !important;
}

.columns-panel-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.columns-panel-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* 查询栏样式由 enterprise.css 全局提供 (.srv-query-bar / .qb-*) */

/* K8s 资源管理专有 */
.qb-resource-meta {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.qb-select--ns {
  width: 180px;
}

.resource-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px 4px 8px;
  border-radius: 6px;
  border: 1px solid color-mix(in srgb, var(--resource-badge-color, rgba(59, 130, 246, 0.75)) 20%, transparent);
  background: color-mix(in srgb, var(--resource-badge-color, rgba(59, 130, 246, 0.75)) 14%, transparent);
  font-size: 13px;
  font-weight: 700;
  color: var(--app-title);
  height: 32px;
  letter-spacing: 0.01em;
}

.resource-badge::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--resource-badge-color, rgba(59, 130, 246, 0.75));
  flex-shrink: 0;
}


:global(html.dark) .resource-badge {
  background: color-mix(in srgb, var(--resource-badge-color, rgba(59, 130, 246, 0.75)) 14%, transparent);
  color: var(--app-title);
}

.resource-badge--default {
  --resource-badge-color: rgba(59, 130, 246, 0.78);
}

.resource-badge--pods {
  --resource-badge-color: rgba(34, 211, 238, 0.9);
}

.resource-badge--workloads {
  --resource-badge-color: rgba(59, 130, 246, 0.9);
}

.resource-badge--services {
  --resource-badge-color: rgba(16, 185, 129, 0.9);
}

.resource-badge--ingress {
  --resource-badge-color: rgba(168, 85, 247, 0.9);
}

.resource-badge--config {
  --resource-badge-color: rgba(245, 158, 11, 0.9);
}

.resource-badge--storage {
  --resource-badge-color: rgba(20, 184, 166, 0.9);
}

.resource-badge--nodes {
  --resource-badge-color: rgba(99, 102, 241, 0.9);
}

.resource-badge--namespaces {
  --resource-badge-color: rgba(239, 68, 68, 0.85);
}

.resource-badge--events {
  --resource-badge-color: rgba(148, 163, 184, 0.95);
}

.resource-badge--jobs {
  --resource-badge-color: rgba(244, 63, 94, 0.85);
}

/* namespace select tags 溢出处理 */
.qb-select--ns :deep(.el-select__selection) {
  flex-wrap: nowrap;
  overflow: hidden;
  align-items: center;
}

.qb-select--ns :deep(.el-select__tags) {
  flex-wrap: nowrap;
  max-width: 100%;
  overflow: hidden;
}

.qb-select--ns :deep(.el-tag) {
  max-width: 120px;
}

.qb-select--ns :deep(.el-tag__content) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  opacity: 0.9;
}

.phase-tag {
  min-width: 86px;
  justify-content: center;
}

.logs-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 10px;
  border: 1px solid var(--k8s-card-border);
  background: rgba(255, 255, 255, 0.88);
  box-shadow: 0 10px 30px rgba(2, 6, 23, 0.06);
}

.logs-toolbar-left {
  min-width: 0;
}
.logs-toolbar-right {
  flex: 0 0 auto;
}

.logs-meta {
  font-size: 12px;
  color: var(--app-muted);
}

.logs-pane {
  position: relative;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow: hidden;
}

:global(html.dark) .logs-toolbar {
  border-color: rgba(226, 232, 240, 0.08);
  background: rgba(2, 6, 23, 0.65);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.25);
}

.logs-meta-badge {
  display: inline-flex;
  align-items: center;
  max-width: 520px;
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.1);
  border: 1px solid rgba(37, 99, 235, 0.22);
  color: rgba(30, 58, 138, 0.92);
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
:global(html.dark) .logs-meta-badge {
  background: rgba(59, 130, 246, 0.16);
  border-color: rgba(96, 165, 250, 0.22);
  color: rgba(226, 232, 240, 0.92);
}

.detail-box {
  flex: 1;
  min-height: 0;
  padding: 10px 12px;
  border-radius: 12px;
  background: rgba(2, 6, 23, 0.88);
  color: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(255, 255, 255, 0.12);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.45;
  white-space: pre;
  tab-size: 2;
  overflow: auto;
}

.detail-box-wrap {
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.pod-term-toolbar {
  margin-bottom: 10px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}
.pod-term-meta {
  font-size: 12px;
  color: var(--app-muted);
  padding-top: 6px;
}

.pod-exec-terminal-host {
  height: 62vh;
  border-radius: 12px;
  overflow: hidden;
}

/* ── Welcome / empty state ─────────────────────── */
.k8s-welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  height: 100%;
  min-height: 320px;
  padding: 48px 24px;
  user-select: none;
}

.k8s-welcome-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: 20px;
  background: rgba(59, 130, 246, 0.08);
  color: rgba(59, 130, 246, 0.65);
}
:global(html.dark) .k8s-welcome-icon {
  background: rgba(59, 130, 246, 0.12);
  color: rgba(96, 165, 250, 0.7);
}

.k8s-welcome-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--color-text-primary);
  letter-spacing: 0.01em;
}

.k8s-welcome-desc {
  font-size: 13px;
  color: var(--color-text-muted);
  margin-top: -4px;
}

.k8s-welcome-hints {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: center;
  margin-top: 12px;
  max-width: 460px;
}

.k8s-hint-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 8px;
  border: 1px solid var(--color-border-subtle);
  background: var(--color-bg-card);
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.k8s-hint-item:hover {
  border-color: rgba(59, 130, 246, 0.35);
  background: rgba(59, 130, 246, 0.06);
  color: var(--color-accent-primary);
}

:global(html.dark) .k8s-hint-item {
  background: rgba(255, 255, 255, 0.03);
  border-color: var(--color-border-default);
}

:global(html.dark) .k8s-hint-item:hover {
  border-color: rgba(96, 165, 250, 0.3);
  background: rgba(59, 130, 246, 0.1);
  color: rgba(96, 165, 250, 0.9);
}
</style>
