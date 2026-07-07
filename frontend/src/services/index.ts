/**
 * API 服务层统一导出
 */
export { login, logout, getCurrentUser } from './auth'
export {
  listProjects,
  createProject,
  updateProject,
  deleteProject,
} from './project'
export {
  listAppTemplates,
  getAppTemplate,
  createAppTemplate,
  updateAppTemplate,
  deleteAppTemplate,
  type AppTemplate,
} from './app-template'
export {
  listClusters,
  getClusterById,
  importCluster,
  updateCluster,
  deleteCluster,
  checkClusterConnection,
} from './clusters'
export * from './clusterContext'
export {
  listNamespaces,
  listNodes,
  listPods,
  getPod,
  getPodYaml,
  deletePod,
  listDeployments,
  deleteDeployment,
  scaleDeployment,
  rollbackDeployment,
  listServices,
  deleteService,
  listConfigMaps,
  deleteConfigMap,
  listEvents,
  getDashboardStats,
  getTopology,
} from './k8s'
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
} from './deploy'
export {
  listAlertRules,
  createAlertRule,
  updateAlertRule,
  deleteAlertRule,
  toggleAlertRule,
  listAlertEvents,
  queryMetrics,
  getMetrics,
  listEvents as listMonitorEvents,
} from './monitor'
export { getChatUrl, listConversations, getConversation, deleteConversation } from './ai'
export { fetchDashboardOverview } from './dashboard'
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
} from './system'
