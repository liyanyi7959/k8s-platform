import { http } from '@/shared/http/http'
import type { ApiResponse } from '@/shared/types/api'

type ListParams = { sort_by?: string; order?: 'asc' | 'desc' }

async function listClusterScoped(clusterId: number, resource: string, params: ListParams = {}): Promise<{ list: any[] }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/${resource}`, { params })) as ApiResponse<{ list: any[] }>
  return resp.data
}

async function getClusterScopedYaml(clusterId: number, resource: string, name: string): Promise<{ text: string }> {
  const resp = (await http.get(`/api/v1/clusters/${clusterId}/${resource}/${encodeURIComponent(name)}/yaml`)) as ApiResponse<{ text: string }>
  return resp.data
}

async function deleteClusterScoped(clusterId: number, resource: string, name: string): Promise<void> {
  await http.delete(`/api/v1/clusters/${clusterId}/${resource}/${encodeURIComponent(name)}`)
}

async function editClusterScoped(clusterId: number, resource: string, req: { yaml: string }): Promise<void> {
  await http.patch(`/api/v1/clusters/${clusterId}/${resource}/edit`, req)
}

export const listCustomResourceDefinitions = (clusterId: number, params: ListParams = {}) => listClusterScoped(clusterId, 'customresourcedefinitions', params)
export const getCustomResourceDefinitionYaml = (clusterId: number, name: string) => getClusterScopedYaml(clusterId, 'customresourcedefinitions', name)
export const deleteCustomResourceDefinition = (clusterId: number, name: string) => deleteClusterScoped(clusterId, 'customresourcedefinitions', name)
export const editCustomResourceDefinition = (clusterId: number, req: { yaml: string }) => editClusterScoped(clusterId, 'customresourcedefinitions', req)

export const listAPIServices = (clusterId: number, params: ListParams = {}) => listClusterScoped(clusterId, 'apiservices', params)
export const getAPIServiceYaml = (clusterId: number, name: string) => getClusterScopedYaml(clusterId, 'apiservices', name)
export const deleteAPIService = (clusterId: number, name: string) => deleteClusterScoped(clusterId, 'apiservices', name)
export const editAPIService = (clusterId: number, req: { yaml: string }) => editClusterScoped(clusterId, 'apiservices', req)

export const listPriorityClasses = (clusterId: number, params: ListParams = {}) => listClusterScoped(clusterId, 'priorityclasses', params)
export const getPriorityClassYaml = (clusterId: number, name: string) => getClusterScopedYaml(clusterId, 'priorityclasses', name)
export const deletePriorityClass = (clusterId: number, name: string) => deleteClusterScoped(clusterId, 'priorityclasses', name)
export const editPriorityClass = (clusterId: number, req: { yaml: string }) => editClusterScoped(clusterId, 'priorityclasses', req)

export const listRuntimeClasses = (clusterId: number, params: ListParams = {}) => listClusterScoped(clusterId, 'runtimeclasses', params)
export const getRuntimeClassYaml = (clusterId: number, name: string) => getClusterScopedYaml(clusterId, 'runtimeclasses', name)
export const deleteRuntimeClass = (clusterId: number, name: string) => deleteClusterScoped(clusterId, 'runtimeclasses', name)
export const editRuntimeClass = (clusterId: number, req: { yaml: string }) => editClusterScoped(clusterId, 'runtimeclasses', req)

export const listCSIDrivers = (clusterId: number, params: ListParams = {}) => listClusterScoped(clusterId, 'csidrivers', params)
export const getCSIDriverYaml = (clusterId: number, name: string) => getClusterScopedYaml(clusterId, 'csidrivers', name)
export const deleteCSIDriver = (clusterId: number, name: string) => deleteClusterScoped(clusterId, 'csidrivers', name)
export const editCSIDriver = (clusterId: number, req: { yaml: string }) => editClusterScoped(clusterId, 'csidrivers', req)

export const listCSINodes = (clusterId: number, params: ListParams = {}) => listClusterScoped(clusterId, 'csinodes', params)
export const getCSINodeYaml = (clusterId: number, name: string) => getClusterScopedYaml(clusterId, 'csinodes', name)
export const deleteCSINode = (clusterId: number, name: string) => deleteClusterScoped(clusterId, 'csinodes', name)
export const editCSINode = (clusterId: number, req: { yaml: string }) => editClusterScoped(clusterId, 'csinodes', req)

export const listValidatingWebhookConfigurations = (clusterId: number, params: ListParams = {}) => listClusterScoped(clusterId, 'validatingwebhookconfigurations', params)
export const getValidatingWebhookConfigurationYaml = (clusterId: number, name: string) => getClusterScopedYaml(clusterId, 'validatingwebhookconfigurations', name)
export const deleteValidatingWebhookConfiguration = (clusterId: number, name: string) => deleteClusterScoped(clusterId, 'validatingwebhookconfigurations', name)
export const editValidatingWebhookConfiguration = (clusterId: number, req: { yaml: string }) => editClusterScoped(clusterId, 'validatingwebhookconfigurations', req)

export const listMutatingWebhookConfigurations = (clusterId: number, params: ListParams = {}) => listClusterScoped(clusterId, 'mutatingwebhookconfigurations', params)
export const getMutatingWebhookConfigurationYaml = (clusterId: number, name: string) => getClusterScopedYaml(clusterId, 'mutatingwebhookconfigurations', name)
export const deleteMutatingWebhookConfiguration = (clusterId: number, name: string) => deleteClusterScoped(clusterId, 'mutatingwebhookconfigurations', name)
export const editMutatingWebhookConfiguration = (clusterId: number, req: { yaml: string }) => editClusterScoped(clusterId, 'mutatingwebhookconfigurations', req)

export const listValidatingAdmissionPolicies = (clusterId: number, params: ListParams = {}) => listClusterScoped(clusterId, 'validatingadmissionpolicies', params)
export const getValidatingAdmissionPolicyYaml = (clusterId: number, name: string) => getClusterScopedYaml(clusterId, 'validatingadmissionpolicies', name)
export const deleteValidatingAdmissionPolicy = (clusterId: number, name: string) => deleteClusterScoped(clusterId, 'validatingadmissionpolicies', name)
export const editValidatingAdmissionPolicy = (clusterId: number, req: { yaml: string }) => editClusterScoped(clusterId, 'validatingadmissionpolicies', req)

export const listValidatingAdmissionPolicyBindings = (clusterId: number, params: ListParams = {}) => listClusterScoped(clusterId, 'validatingadmissionpolicybindings', params)
export const getValidatingAdmissionPolicyBindingYaml = (clusterId: number, name: string) => getClusterScopedYaml(clusterId, 'validatingadmissionpolicybindings', name)
export const deleteValidatingAdmissionPolicyBinding = (clusterId: number, name: string) => deleteClusterScoped(clusterId, 'validatingadmissionpolicybindings', name)
export const editValidatingAdmissionPolicyBinding = (clusterId: number, req: { yaml: string }) => editClusterScoped(clusterId, 'validatingadmissionpolicybindings', req)
