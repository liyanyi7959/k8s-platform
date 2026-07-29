package ports

import (
	"context"
	"time"
)

// DashboardRuntime is the infrastructure boundary for a Fleet dashboard.
// It deliberately exposes normalized snapshots instead of Kubernetes clients,
// cache implementations, or legacy service types.  The application layer owns
// the aggregation, risk classification, trend policy, and cache lifetime.
type DashboardRuntime interface {
	CollectDashboard(context.Context, uint64) (DashboardSnapshot, error)
	CheckDashboardAPI(context.Context, uint64) bool
	ProbeCertificates(context.Context, uint64, bool) (CertificateSnapshot, bool)
}

// DashboardCache is a small cache boundary.  Cache failures must never make a
// dashboard request fail, so the application treats it as best-effort.
type DashboardCache interface {
	Enabled() bool
	Get(context.Context, string) ([]byte, bool, error)
	Set(context.Context, string, []byte, time.Duration) error
}

type DashboardSnapshot struct {
	APIOK       bool
	K8sVersion  string
	Nodes       []DashboardNode
	Pods        []DashboardPod
	Workloads   []DashboardWorkload
	Metrics     []DashboardNodeMetric
	RecentEvents []DashboardEvent
}

type DashboardNode struct {
	Name               string
	Ready              bool
	IP                 string
	CPUAllocatableMilli int64
	MemoryAllocatable   int64
}

type DashboardPod struct {
	Name          string
	Namespace     string
	Phase         string
	NodeName      string
	Deleting      bool
	WaitingReason string
	TerminatedReason string
	TerminatedExitCode int32
}

type DashboardWorkload struct {
	Name      string
	Namespace string
	Kind      string
	Replicas  int32
	Ready     int32
}

type DashboardNodeMetric struct {
	NodeName    string
	CPUMilli    int64
	MemoryBytes int64
}

type DashboardEvent struct {
	Type           string
	Reason         string
	Message        string
	Namespace      string
	LastTimestamp  time.Time
	Count          int32
	InvolvedKind   string
	InvolvedName   string
}

type CertificateSnapshot struct {
	APIServer        CertificateObservation
	ClusterCA        CertificateObservation
	Etcd             CertificateObservation
	ControlPlane     CertificateObservation
	ControlPlaneName string
	Kubelet          CertificateObservation
}

type CertificateObservation struct {
	Available  bool
	CommonName string
	NotBefore  time.Time
	NotAfter   time.Time
}
