import React, { useEffect } from 'react'
import { history, useModel } from '@umijs/max'

const routeMap: Record<string, string> = {
  '/cluster/pods': 'pods',
  '/cluster/services': 'services',
  '/cluster/deployments': 'deployments',
  '/cluster/statefulsets': 'statefulsets',
  '/cluster/daemonsets': 'daemonsets',
  '/cluster/jobs': 'jobs',
  '/cluster/cronjobs': 'cronjobs',
  '/cluster/configmaps': 'configmaps',
  '/cluster/secrets': 'secrets',
  '/cluster/pvc': 'pvc',
  '/cluster/storage-classes': 'storage-classes',
  '/cluster/nodes': 'nodes',
  '/cluster/events': 'nodes',
  '/cluster/hpas': 'hpas',
}

const ClusterResourceRedirect: React.FC = () => {
  const { currentCluster } = useModel('cluster')

  useEffect(() => {
    if (!currentCluster) return

    const target = routeMap[history.location.pathname] || 'pods'
    history.replace(`/k8s/${currentCluster.id}/${target}?cluster=${currentCluster.id}`)
  }, [currentCluster])

  return null
}

export default ClusterResourceRedirect
