package fleet

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net"
	neturl "net/url"
	"strings"
	"sync"
	"time"

	"k8s-platform-backend/internal/fleet/ports"
	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// DashboardRuntime is the retained Fleet infrastructure adapter.  It performs
// Kubernetes and TLS transport work only; dashboard aggregation and cache
// policy live in fleet/application.
type DashboardRuntime struct{ k8s *service.K8sService }

func NewDashboardRuntime(k8s *service.K8sService) *DashboardRuntime {
	return &DashboardRuntime{k8s: k8s}
}

func (r *DashboardRuntime) CheckDashboardAPI(ctx context.Context, clusterID uint64) bool {
	if r == nil || r.k8s == nil || clusterID == 0 {
		return false
	}
	client, err := r.k8s.TypedClient(ctx, clusterID)
	if err != nil || client == nil {
		return false
	}
	_, err = client.Discovery().ServerVersion()
	return err == nil
}

func (r *DashboardRuntime) CollectDashboard(ctx context.Context, clusterID uint64) (ports.DashboardSnapshot, error) {
	if r == nil || r.k8s == nil || clusterID == 0 {
		return ports.DashboardSnapshot{}, nil
	}
	client, err := r.k8s.TypedClient(ctx, clusterID)
	if err != nil || client == nil {
		// The legacy dashboard intentionally returns an empty offline snapshot
		// instead of failing the overview endpoint when the cluster is down.
		return ports.DashboardSnapshot{}, nil
	}

	var snapshot ports.DashboardSnapshot
	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		version, versionErr := client.Discovery().ServerVersion()
		nodes, nodesErr := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if version != nil {
			snapshot.K8sVersion = strings.TrimSpace(version.GitVersion)
		}
		if nodesErr != nil || nodes == nil {
			snapshot.APIOK = versionErr == nil
			return
		}
		snapshot.APIOK = true
		snapshot.Nodes = dashboardNodes(nodes.Items)
	}()
	go func() {
		defer wg.Done()
		snapshot.Pods = dashboardPods(ctx, client)
	}()
	go func() {
		defer wg.Done()
		snapshot.Workloads = dashboardWorkloads(ctx, client)
	}()
	go func() {
		defer wg.Done()
		snapshot.RecentEvents = dashboardEvents(ctx, client)
	}()
	go func() {
		defer wg.Done()
		values, err := r.k8s.List(ctx, clusterID, schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}, "", "", "", nil)
		if err == nil {
			snapshot.Metrics = dashboardMetrics(values)
		}
	}()
	wg.Wait()
	return snapshot, nil
}

func dashboardNodes(nodes []corev1.Node) []ports.DashboardNode {
	result := make([]ports.DashboardNode, 0, len(nodes))
	for index := range nodes {
		node := &nodes[index]
		result = append(result, ports.DashboardNode{
			Name: node.Name, Ready: nodeReady(node), IP: nodePrimaryIP(node),
			CPUAllocatableMilli: node.Status.Allocatable.Cpu().MilliValue(),
			MemoryAllocatable:   node.Status.Allocatable.Memory().Value(),
		})
	}
	return result
}

func nodeReady(node *corev1.Node) bool {
	if node == nil {
		return false
	}
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func nodePrimaryIP(node *corev1.Node) string {
	if node == nil {
		return ""
	}
	for _, target := range []corev1.NodeAddressType{corev1.NodeInternalIP, corev1.NodeExternalIP} {
		for _, address := range node.Status.Addresses {
			if address.Type == target && strings.TrimSpace(address.Address) != "" {
				return strings.TrimSpace(address.Address)
			}
		}
	}
	return ""
}

func dashboardPods(ctx context.Context, client *kubernetes.Clientset) []ports.DashboardPod {
	if client == nil {
		return nil
	}
	items := make([]ports.DashboardPod, 0)
	options := metav1.ListOptions{Limit: 500}
	for {
		list, err := client.CoreV1().Pods("").List(ctx, options)
		if err != nil || list == nil {
			break
		}
		for index := range list.Items {
			items = append(items, toDashboardPod(&list.Items[index]))
		}
		if strings.TrimSpace(list.Continue) == "" {
			break
		}
		options.Continue = list.Continue
	}
	return items
}

func toDashboardPod(pod *corev1.Pod) ports.DashboardPod {
	if pod == nil {
		return ports.DashboardPod{}
	}
	item := ports.DashboardPod{Name: pod.Name, Namespace: pod.Namespace, Phase: string(pod.Status.Phase), NodeName: pod.Spec.NodeName, Deleting: pod.DeletionTimestamp != nil}
	statuses := append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
	for _, status := range statuses {
		if item.WaitingReason == "" && status.State.Waiting != nil {
			item.WaitingReason = status.State.Waiting.Reason
		}
		if item.TerminatedReason == "" && status.State.Terminated != nil {
			item.TerminatedReason, item.TerminatedExitCode = status.State.Terminated.Reason, status.State.Terminated.ExitCode
		}
	}
	return item
}

func dashboardWorkloads(ctx context.Context, client *kubernetes.Clientset) []ports.DashboardWorkload {
	if client == nil {
		return nil
	}
	result := make([]ports.DashboardWorkload, 0)
	var deployments *appsv1.DeploymentList
	var statefulsets *appsv1.StatefulSetList
	var daemonsets *appsv1.DaemonSetList
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		deployments, _ = client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	}()
	go func() {
		defer wg.Done()
		statefulsets, _ = client.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	}()
	go func() {
		defer wg.Done()
		daemonsets, _ = client.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	}()
	wg.Wait()
	if deployments != nil {
		for index := range deployments.Items {
			item := &deployments.Items[index]
			result = append(result, ports.DashboardWorkload{Name: item.Name, Namespace: item.Namespace, Kind: "Deployment", Replicas: item.Status.Replicas, Ready: item.Status.ReadyReplicas})
		}
	}
	if statefulsets != nil {
		for index := range statefulsets.Items {
			item := &statefulsets.Items[index]
			result = append(result, ports.DashboardWorkload{Name: item.Name, Namespace: item.Namespace, Kind: "StatefulSet", Replicas: item.Status.Replicas, Ready: item.Status.ReadyReplicas})
		}
	}
	if daemonsets != nil {
		for index := range daemonsets.Items {
			item := &daemonsets.Items[index]
			result = append(result, ports.DashboardWorkload{Name: item.Name, Namespace: item.Namespace, Kind: "DaemonSet", Replicas: item.Status.DesiredNumberScheduled, Ready: item.Status.NumberReady})
		}
	}
	return result
}

func dashboardEvents(ctx context.Context, client *kubernetes.Clientset) []ports.DashboardEvent {
	if client == nil {
		return nil
	}
	list, err := client.CoreV1().Events("").List(ctx, metav1.ListOptions{FieldSelector: "type=Warning", Limit: 50})
	if err != nil || list == nil {
		return nil
	}
	result := make([]ports.DashboardEvent, 0, len(list.Items))
	for index := range list.Items {
		event := &list.Items[index]
		timestamp := event.LastTimestamp.Time
		if timestamp.IsZero() {
			timestamp = event.EventTime.Time
		}
		result = append(result, ports.DashboardEvent{Type: event.Type, Reason: event.Reason, Message: event.Message, Namespace: event.Namespace, LastTimestamp: timestamp, Count: event.Count, InvolvedKind: event.InvolvedObject.Kind, InvolvedName: event.InvolvedObject.Name})
	}
	return result
}

func dashboardMetrics(values []any) []ports.DashboardNodeMetric {
	result := make([]ports.DashboardNodeMetric, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}
		metadata, metadataOK := item["metadata"].(map[string]any)
		usage, usageOK := item["usage"].(map[string]any)
		if !metadataOK || !usageOK {
			continue
		}
		name, nameOK := metadata["name"].(string)
		cpuRaw, cpuOK := usage["cpu"].(string)
		memoryRaw, memoryOK := usage["memory"].(string)
		if !nameOK || !cpuOK || !memoryOK {
			continue
		}
		cpu, cpuErr := resource.ParseQuantity(cpuRaw)
		memory, memoryErr := resource.ParseQuantity(memoryRaw)
		if cpuErr != nil || memoryErr != nil {
			continue
		}
		result = append(result, ports.DashboardNodeMetric{NodeName: name, CPUMilli: cpu.MilliValue(), MemoryBytes: memory.Value()})
	}
	return result
}

func (r *DashboardRuntime) ProbeCertificates(ctx context.Context, clusterID uint64, apiOK bool) (ports.CertificateSnapshot, bool) {
	snapshot := ports.CertificateSnapshot{}
	cacheable := apiOK
	if r == nil || r.k8s == nil || clusterID == 0 {
		return snapshot, false
	}
	kubeconfig, err := r.k8s.GetKubeconfig(ctx, clusterID)
	if err != nil || strings.TrimSpace(kubeconfig) == "" {
		return snapshot, false
	}
	config, err := clientcmd.Load([]byte(kubeconfig))
	if err != nil || config == nil {
		return snapshot, false
	}
	serverURL, caData := kubeconfigServerAndCA(config)
	if serverURL == "" {
		cacheable = false
	} else if certificate, err := fetchTLSServerCertificate(ctx, serverURL, 5*time.Second); err == nil && certificate != nil {
		snapshot.APIServer = certificateObservation(certificate)
	} else {
		cacheable = false
	}
	if len(caData) == 0 {
		cacheable = false
	} else if certificate, err := parseFirstPEMCertificate(caData); err == nil && certificate != nil {
		snapshot.ClusterCA = certificateObservation(certificate)
	} else {
		cacheable = false
	}
	if !apiOK {
		return snapshot, false
	}
	client, err := r.k8s.TypedClient(ctx, clusterID)
	if err != nil || client == nil {
		return snapshot, false
	}
	controlPlaneIPs := controlPlaneNodeIPs(ctx, client, 12)
	if certificate, label, found := earliestControlPlaneCertificate(ctx, controlPlaneIPs); found {
		snapshot.ControlPlane, snapshot.ControlPlaneName = certificateObservation(certificate), label
	} else {
		cacheable = false
	}
	if certificate, found := earliestCertificateOnPort(ctx, controlPlaneIPs, "2379", 2*time.Second); found {
		snapshot.Etcd = certificateObservation(certificate)
	} else {
		cacheable = false
	}
	if certificate, found := earliestKubeletCertificate(ctx, client, 20); found {
		snapshot.Kubelet = certificateObservation(certificate)
	} else {
		cacheable = false
	}
	if ctx.Err() != nil {
		cacheable = false
	}
	return snapshot, cacheable
}

func certificateObservation(certificate *x509.Certificate) ports.CertificateObservation {
	if certificate == nil {
		return ports.CertificateObservation{}
	}
	return ports.CertificateObservation{Available: true, CommonName: certificate.Subject.CommonName, NotBefore: certificate.NotBefore.UTC(), NotAfter: certificate.NotAfter.UTC()}
}

func kubeconfigServerAndCA(config *clientcmdapi.Config) (string, []byte) {
	if config == nil {
		return "", nil
	}
	contextName := strings.TrimSpace(config.CurrentContext)
	if contextName == "" {
		for name := range config.Contexts {
			contextName = name
			break
		}
	}
	if contextName == "" || config.Contexts[contextName] == nil {
		return "", nil
	}
	clusterName := strings.TrimSpace(config.Contexts[contextName].Cluster)
	if clusterName == "" || config.Clusters[clusterName] == nil {
		return "", nil
	}
	cluster := config.Clusters[clusterName]
	return strings.TrimSpace(cluster.Server), cluster.CertificateAuthorityData
}

func parseFirstPEMCertificate(value []byte) (*x509.Certificate, error) {
	rest := value
	for {
		block, remaining := pem.Decode(rest)
		rest = remaining
		if block == nil {
			return nil, errors.New("no certificate PEM block")
		}
		if block.Type != "CERTIFICATE" {
			if len(rest) == 0 {
				return nil, errors.New("no certificate PEM block")
			}
			continue
		}
		return x509.ParseCertificate(block.Bytes)
	}
}

func fetchTLSServerCertificate(ctx context.Context, serverURL string, timeout time.Duration) (*x509.Certificate, error) {
	parsed, err := neturl.Parse(strings.TrimSpace(serverURL))
	if err != nil {
		return nil, err
	}
	host, port := strings.TrimSpace(parsed.Hostname()), strings.TrimSpace(parsed.Port())
	if host == "" {
		return nil, errors.New("invalid server host")
	}
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: dialTimeout(ctx, timeout)}, Config: &tls.Config{InsecureSkipVerify: true, ServerName: host}}
	raw, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, err
	}
	connection, ok := raw.(*tls.Conn)
	if !ok {
		_ = raw.Close()
		return nil, errors.New("unexpected TLS connection type")
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(dialTimeout(ctx, timeout)))
	state := connection.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, errors.New("no peer certificate")
	}
	return state.PeerCertificates[0], nil
}

func fetchTLSCertificateByAddress(ctx context.Context, address, serverName string, timeout time.Duration) (*x509.Certificate, error) {
	try := func(sni string) (*x509.Certificate, error) {
		raw, err := (&net.Dialer{Timeout: dialTimeout(ctx, timeout)}).DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, err
		}
		defer raw.Close()
		var captured *x509.Certificate
		connection := tls.Client(raw, &tls.Config{InsecureSkipVerify: true, ServerName: sni, VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) > 0 && captured == nil {
				captured, _ = x509.ParseCertificate(rawCerts[0])
			}
			return nil
		}})
		_ = connection.SetDeadline(time.Now().Add(dialTimeout(ctx, timeout)))
		_ = connection.Handshake()
		_ = connection.Close()
		if captured == nil {
			return nil, errors.New("no peer certificate")
		}
		return captured, nil
	}
	if certificate, err := try(serverName); err == nil && certificate != nil {
		return certificate, nil
	}
	if strings.TrimSpace(serverName) != "" {
		return try("")
	}
	return nil, errors.New("no peer certificate")
}

func dialTimeout(ctx context.Context, fallback time.Duration) time.Duration {
	if fallback <= 0 {
		fallback = 500 * time.Millisecond
	}
	if ctx == nil {
		return fallback
	}
	if deadline, found := ctx.Deadline(); found {
		if remaining := time.Until(deadline); remaining <= 0 {
			return time.Millisecond
		} else if remaining < fallback {
			return remaining
		}
	}
	return fallback
}

func controlPlaneNodeIPs(ctx context.Context, client *kubernetes.Clientset, maxNodes int) []string {
	if client == nil {
		return nil
	}
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	result := make([]string, 0, len(nodes.Items))
	for index := range nodes.Items {
		if maxNodes > 0 && len(result) >= maxNodes {
			break
		}
		node := &nodes.Items[index]
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; !ok {
			if _, ok := node.Labels["node-role.kubernetes.io/master"]; !ok {
				continue
			}
		}
		if ip := nodePrimaryIP(node); ip != "" {
			result = append(result, ip)
		}
	}
	return result
}

func earliestCertificateOnPort(ctx context.Context, ips []string, port string, timeout time.Duration) (*x509.Certificate, bool) {
	if len(ips) == 0 {
		return nil, false
	}
	results := make(chan *x509.Certificate, len(ips))
	semaphore := make(chan struct{}, 20)
	for _, ip := range ips {
		ip := ip
		semaphore <- struct{}{}
		go func() {
			defer func() { <-semaphore }()
			if ctx.Err() != nil {
				results <- nil
				return
			}
			certificate, _ := fetchTLSCertificateByAddress(ctx, net.JoinHostPort(ip, port), ip, timeout)
			results <- certificate
		}()
	}
	var earliest *x509.Certificate
	for range ips {
		if certificate := <-results; certificate != nil && (earliest == nil || certificate.NotAfter.Before(earliest.NotAfter)) {
			earliest = certificate
		}
	}
	return earliest, earliest != nil
}

func earliestControlPlaneCertificate(ctx context.Context, ips []string) (*x509.Certificate, string, bool) {
	var earliest *x509.Certificate
	label := ""
	for _, probe := range []struct{ port, name string }{{"10257", "controller-manager"}, {"10259", "scheduler"}} {
		certificate, found := earliestCertificateOnPort(ctx, ips, probe.port, 2*time.Second)
		if found && (earliest == nil || certificate.NotAfter.Before(earliest.NotAfter)) {
			earliest, label = certificate, probe.name
		}
	}
	return earliest, label, earliest != nil
}

func earliestKubeletCertificate(ctx context.Context, client *kubernetes.Clientset, maxNodes int) (*x509.Certificate, bool) {
	if client == nil {
		return nil, false
	}
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, false
	}
	ips := make([]string, 0, len(nodes.Items))
	for index := range nodes.Items {
		if maxNodes > 0 && len(ips) >= maxNodes {
			break
		}
		if ip := nodePrimaryIP(&nodes.Items[index]); ip != "" {
			ips = append(ips, ip)
		}
	}
	return earliestCertificateOnPort(ctx, ips, "10250", 2*time.Second)
}
