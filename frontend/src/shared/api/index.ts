/**
 * API 服务层统一导出
 */
export { login, logout, getCurrentUser } from '@/features/iam/api'
export {
  listProjects,
  createProject,
  updateProject,
  deleteProject,
} from '@/features/workspace/api'
export {
  listAppTemplates,
  getAppTemplate,
  createAppTemplate,
  updateAppTemplate,
  deleteAppTemplate,
  type AppTemplate,
} from '@/features/provisioning/api/app-template'
export {
  listClusters,
  getClusterById,
  importCluster,
  updateCluster,
  deleteCluster,
  checkClusterConnection,
} from '@/features/fleet/api/clusters'
export * from '@/features/fleet/api/cluster-context'
export {
  listNamespaces,
  listNodes,
  listPods,
  getPod,
  getPodYaml,
  deletePod,
  listWorkloads,
  deleteWorkload,
  scaleWorkload,
  rollbackDeployment,
  listServices,
  deleteService,
  listConfigMaps,
  deleteConfigMap,
  listEvents,
  getDashboardStats,
  getTopology,
} from '@/features/kops/api/k8s'
export {
  getDeployPlans,
  getDeployPlanById,
  createDeployPlan,
  executeDeployPlan,
  cancelDeployPlan,
  deleteDeployPlan,
  getCredentials,
  createCredential,
  updateCredential,
  deleteCredential,
  batchDeleteCredentials,
} from '@/features/provisioning/api/deploy'
export {
  listAlertRules,
  createAlertRule,
  updateAlertRule,
  deleteAlertRule,
  toggleAlertRule,
  queryMetrics,
  getMetrics,
} from '@/features/incident/api'
export { getChatUrl, listConversations, getConversation, deleteConversation } from '@/features/ai/api'
export { fetchDashboardOverview } from '@/features/fleet/api/dashboard'
export {
  listUsers,
  getUserById,
  createUser,
  updateUser,
  deleteUser,
  resetPassword,
  listRoles,
  listAllRoles,
  createRole,
  updateRole,
  deleteRole,
  listAuditLogs,
  getSystemSettings,
  updateSystemSettings,
} from '@/features/platform/api'
