export const CLUSTER_QUERY_KEY = 'cluster'

export const getClusterIdFromSearch = (search = '') => {
  const params = new URLSearchParams(search)
  return params.get(CLUSTER_QUERY_KEY)
}

export const buildSearchWithCluster = (search: string, clusterId: string | null) => {
  const params = new URLSearchParams(search)

  if (clusterId) {
    params.set(CLUSTER_QUERY_KEY, clusterId)
  } else {
    params.delete(CLUSTER_QUERY_KEY)
  }

  const nextSearch = params.toString()
  return nextSearch ? `?${nextSearch}` : ''
}

export const withClusterQuery = (path: string, clusterId: string) => {
  const [pathname = path, rawSearch = ''] = path.split('?')
  return `${pathname}${buildSearchWithCluster(rawSearch, clusterId)}`
}
