import GenericResourceList from '@/components/GenericResourceList'

export default function PodMetricsPage() {
  return <GenericResourceList resource="podmetrics" title="PodMetrics" namespaced deletable={false} creatable={false} />
}