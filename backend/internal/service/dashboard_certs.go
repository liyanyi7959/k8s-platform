package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	neturl "net/url"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// ──────────────────────────────────────────────────────────────
// 证书风险检测
// 从 dashboard_service.go 拆分，包含集群证书到期检测相关全部逻辑。
// ──────────────────────────────────────────────────────────────

func (s *DashboardService) getClusterCertificateRisks(ctx context.Context, clusterID uint64, apiOK bool) ([]map[string]any, bool) {
	now := time.Now().UTC()
	cacheable := apiOK

	type certItem struct {
		Key        string
		Name       string
		Component  string
		Purpose    string
		NotBefore  *time.Time
		NotAfter   *time.Time
		DaysLeft   *int
		RiskStatus string
	}

	toMap := func(it certItem) map[string]any {
		out := map[string]any{
			"key":       it.Key,
			"name":      it.Name,
			"component": it.Component,
			"purpose":   it.Purpose,
			"status":    it.RiskStatus,
		}
		if it.NotBefore != nil {
			out["not_before"] = it.NotBefore.UTC().Format(time.RFC3339)
		}
		if it.NotAfter != nil {
			out["not_after"] = it.NotAfter.UTC().Format(time.RFC3339)
		}
		if it.DaysLeft != nil {
			out["days_left"] = *it.DaysLeft
		}
		return out
	}

	calcStatus := func(days *int, notAfter *time.Time) string {
		if days == nil || notAfter == nil {
			return "unknown"
		}
		if notAfter.UTC().Before(now) || *days <= 7 {
			return "critical"
		}
		if *days <= 30 {
			return "warn"
		}
		return "ok"
	}

	unknown := func(key, name, component, purpose string) certItem {
		return certItem{
			Key:        key,
			Name:       name,
			Component:  component,
			Purpose:    purpose,
			RiskStatus: "unknown",
		}
	}

	items := []certItem{
		unknown("apiserver", "API Server 证书", "API Server", "集群控制面入口（https://kube-apiserver）"),
		unknown("cluster_ca", "集群 CA 证书", "Cluster CA", "API Server 信任链根证书（kubeconfig CA）"),
		unknown("etcd", "etcd 证书", "etcd", "控制平面数据存储（etcd:2379，取最早到期）"),
		unknown("control_plane", "控制平面组件证书", "Control Plane", "controller-manager/scheduler HTTPS（10257/10259，取最早到期）"),
		unknown("kubelet", "kubelet 证书（节点）", "kubelet", "节点 kubelet HTTPS（10250，取最早到期）"),
	}

	kc, err := s.clusterReg.GetKubeconfig(ctx, clusterID)
	if err == nil && strings.TrimSpace(kc) != "" {
		if cfg, err := clientcmd.Load([]byte(kc)); err == nil && cfg != nil {
			serverURL, caData := extractKubeconfigClusterServerAndCA(cfg)
			if serverURL != "" {
				if cert, err := fetchTLSServerCertificate(ctx, serverURL, 5*time.Second); err == nil && cert != nil {
					nb := cert.NotBefore.UTC()
					na := cert.NotAfter.UTC()
					days := int(na.Sub(now).Hours() / 24)
					items[0].NotBefore = &nb
					items[0].NotAfter = &na
					items[0].DaysLeft = &days
					items[0].RiskStatus = calcStatus(&days, &na)
					items[0].Name = fmt.Sprintf("API Server 证书（%s）", cert.Subject.CommonName)
				} else {
					cacheable = false
				}
			} else {
				cacheable = false
			}

			if len(caData) > 0 {
				if cert, err := parseFirstPEMX509(caData); err == nil && cert != nil {
					nb := cert.NotBefore.UTC()
					na := cert.NotAfter.UTC()
					days := int(na.Sub(now).Hours() / 24)
					items[1].NotBefore = &nb
					items[1].NotAfter = &na
					items[1].DaysLeft = &days
					items[1].RiskStatus = calcStatus(&days, &na)
					items[1].Name = fmt.Sprintf("集群 CA 证书（%s）", cert.Subject.CommonName)
				} else {
					cacheable = false
				}
			} else {
				cacheable = false
			}
		} else {
			cacheable = false
		}
	} else {
		cacheable = false
	}

	if apiOK {
		if cert, days, which, ok := s.findEarliestControlPlaneCert(ctx, clusterID, 12); ok && cert != nil {
			nb := cert.NotBefore.UTC()
			na := cert.NotAfter.UTC()
			items[3].NotBefore = &nb
			items[3].NotAfter = &na
			items[3].DaysLeft = &days
			items[3].RiskStatus = calcStatus(&days, &na)
			if which != "" {
				items[3].Name = fmt.Sprintf("控制平面组件证书（%s）", which)
			}
		} else {
			cacheable = false
		}
		if cert, days, ok := s.findEarliestEtcdCert(ctx, clusterID, 12); ok && cert != nil {
			nb := cert.NotBefore.UTC()
			na := cert.NotAfter.UTC()
			items[2].NotBefore = &nb
			items[2].NotAfter = &na
			items[2].DaysLeft = &days
			items[2].RiskStatus = calcStatus(&days, &na)
		} else {
			cacheable = false
		}
		if cert, days, ok := s.findEarliestKubeletCert(ctx, clusterID, 20); ok && cert != nil {
			nb := cert.NotBefore.UTC()
			na := cert.NotAfter.UTC()
			items[4].NotBefore = &nb
			items[4].NotAfter = &na
			items[4].DaysLeft = &days
			items[4].RiskStatus = calcStatus(&days, &na)
		} else {
			cacheable = false
		}
	} else {
		cacheable = false
	}

	sort.SliceStable(items, func(i, j int) bool {
		score := func(s string) int {
			if s == "critical" {
				return 3
			}
			if s == "warn" {
				return 2
			}
			if s == "ok" {
				return 1
			}
			return 0
		}
		ai, aj := score(items[i].RiskStatus), score(items[j].RiskStatus)
		if ai != aj {
			return ai > aj
		}
		if items[i].DaysLeft == nil && items[j].DaysLeft != nil {
			return false
		}
		if items[i].DaysLeft != nil && items[j].DaysLeft == nil {
			return true
		}
		if items[i].DaysLeft != nil && items[j].DaysLeft != nil && *items[i].DaysLeft != *items[j].DaysLeft {
			return *items[i].DaysLeft < *items[j].DaysLeft
		}
		return items[i].Key < items[j].Key
	})

	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, toMap(it))
	}
	if ctx.Err() != nil {
		cacheable = false
	}
	return out, cacheable
}

func dialTimeoutFromContext(ctx context.Context, fallback time.Duration) time.Duration {
	if fallback <= 0 {
		fallback = 500 * time.Millisecond
	}
	if ctx == nil {
		return fallback
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return time.Millisecond
		}
		if remaining < fallback {
			return remaining
		}
	}
	return fallback
}

// ──────────────────────────────────────────────────────────────
// Kubeconfig 解析辅助
// ──────────────────────────────────────────────────────────────

func extractKubeconfigClusterServerAndCA(cfg *clientcmdapi.Config) (server string, caData []byte) {
	if cfg == nil {
		return "", nil
	}
	ctxName := strings.TrimSpace(cfg.CurrentContext)
	if ctxName == "" {
		for k := range cfg.Contexts {
			ctxName = k
			break
		}
	}
	if ctxName == "" {
		return "", nil
	}
	ctxObj := cfg.Contexts[ctxName]
	if ctxObj == nil {
		return "", nil
	}
	clusterName := strings.TrimSpace(ctxObj.Cluster)
	if clusterName == "" {
		return "", nil
	}
	cl := cfg.Clusters[clusterName]
	if cl == nil {
		return "", nil
	}
	return strings.TrimSpace(cl.Server), cl.CertificateAuthorityData
}

// ──────────────────────────────────────────────────────────────
// 控制面节点 IP 发现
// ──────────────────────────────────────────────────────────────

func listControlPlaneNodeIPs(ctx context.Context, cs *kubernetes.Clientset, maxNodes int) []string {
	if cs == nil {
		return nil
	}
	nodes, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil || len(nodes.Items) == 0 {
		return nil
	}
	out := make([]string, 0, len(nodes.Items))
	for i := range nodes.Items {
		if maxNodes > 0 && len(out) >= maxNodes {
			break
		}
		n := &nodes.Items[i]
		if n.Labels == nil {
			continue
		}
		if _, ok := n.Labels["node-role.kubernetes.io/control-plane"]; !ok {
			if _, ok := n.Labels["node-role.kubernetes.io/master"]; !ok {
				continue
			}
		}
		ip := ""
		for _, a := range n.Status.Addresses {
			if a.Type == corev1.NodeInternalIP && strings.TrimSpace(a.Address) != "" {
				ip = strings.TrimSpace(a.Address)
				break
			}
		}
		if ip == "" {
			for _, a := range n.Status.Addresses {
				if a.Type == corev1.NodeExternalIP && strings.TrimSpace(a.Address) != "" {
					ip = strings.TrimSpace(a.Address)
					break
				}
			}
		}
		if ip != "" {
			out = append(out, ip)
		}
	}
	return out
}

// ──────────────────────────────────────────────────────────────
// PEM / TLS 证书获取工具
// ──────────────────────────────────────────────────────────────

func parseFirstPEMX509(b []byte) (*x509.Certificate, error) {
	rest := b
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return nil, errors.New("no pem block")
		}
		if block.Type != "CERTIFICATE" {
			if len(rest) == 0 {
				return nil, errors.New("no certificate pem block")
			}
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		return cert, nil
	}
}

func fetchTLSServerCertificate(ctx context.Context, serverURL string, timeout time.Duration) (*x509.Certificate, error) {
	u, err := neturl.Parse(strings.TrimSpace(serverURL))
	if err != nil {
		return nil, err
	}
	host := strings.TrimSpace(u.Hostname())
	port := strings.TrimSpace(u.Port())
	if host == "" {
		return nil, errors.New("invalid server host")
	}
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	addr := net.JoinHostPort(host, port)

	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: dialTimeoutFromContext(ctx, timeout)},
		Config:    &tls.Config{InsecureSkipVerify: true, ServerName: host},
	}
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	conn, ok := rawConn.(*tls.Conn)
	if !ok {
		_ = rawConn.Close()
		return nil, errors.New("unexpected tls connection type")
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(dialTimeoutFromContext(ctx, timeout)))
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, errors.New("no peer certificates")
	}
	return state.PeerCertificates[0], nil
}

func fetchTLSServerCertificateByAddr(ctx context.Context, addr string, serverName string, timeout time.Duration) (*x509.Certificate, error) {
	tryOnce := func(sni string) (*x509.Certificate, error) {
		dialer := &net.Dialer{Timeout: dialTimeoutFromContext(ctx, timeout)}
		rawConn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
		defer func() { _ = rawConn.Close() }()
		var captured *x509.Certificate
		cfg := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         sni,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				if len(rawCerts) == 0 || captured != nil {
					return nil
				}
				c, err := x509.ParseCertificate(rawCerts[0])
				if err != nil {
					return nil
				}
				captured = c
				return nil
			},
		}
		conn := tls.Client(rawConn, cfg)
		_ = conn.SetDeadline(time.Now().Add(dialTimeoutFromContext(ctx, timeout)))
		_ = conn.Handshake()
		_ = conn.Close()
		if captured != nil {
			return captured, nil
		}
		return nil, errors.New("no peer certificates")
	}

	if cert, err := tryOnce(serverName); err == nil && cert != nil {
		return cert, nil
	}
	if strings.TrimSpace(serverName) != "" {
		if cert, err := tryOnce(""); err == nil && cert != nil {
			return cert, nil
		}
	}
	return nil, errors.New("no peer certificates")
}

// ──────────────────────────────────────────────────────────────
// 各组件最早到期证书发现
// ──────────────────────────────────────────────────────────────

func findEarliestCertFromNodePorts(ctx context.Context, ips []string, port string, timeout time.Duration) (*x509.Certificate, int, bool) {
	if len(ips) == 0 {
		return nil, 0, false
	}
	type res struct {
		cert *x509.Certificate
	}
	sem := make(chan struct{}, 20)
	ch := make(chan res, len(ips))
	for _, ip := range ips {
		ip := ip
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			if ctx.Err() != nil {
				ch <- res{}
				return
			}
			addr := net.JoinHostPort(ip, port)
			cert, _ := fetchTLSServerCertificateByAddr(ctx, addr, ip, timeout)
			ch <- res{cert: cert}
		}()
	}
	var earliest *x509.Certificate
	for i := 0; i < len(ips); i++ {
		r := <-ch
		if r.cert == nil {
			continue
		}
		if earliest == nil || r.cert.NotAfter.Before(earliest.NotAfter) {
			earliest = r.cert
		}
	}
	if earliest == nil {
		return nil, 0, false
	}
	now := time.Now().UTC()
	days := int(earliest.NotAfter.UTC().Sub(now).Hours() / 24)
	return earliest, days, true
}

func findEarliestControlPlaneCertFromNodeIPs(ctx context.Context, ips []string) (*x509.Certificate, int, string, bool) {
	type probe struct {
		port  string
		label string
	}
	probes := []probe{
		{port: "10257", label: "controller-manager"},
		{port: "10259", label: "scheduler"},
	}
	timeout := 2 * time.Second
	var earliest *x509.Certificate
	which := ""
	for _, p := range probes {
		cert, _, ok := findEarliestCertFromNodePorts(ctx, ips, p.port, timeout)
		if !ok || cert == nil {
			continue
		}
		if earliest == nil || cert.NotAfter.Before(earliest.NotAfter) {
			earliest = cert
			which = p.label
		}
	}
	if earliest == nil {
		return nil, 0, "", false
	}
	now := time.Now().UTC()
	days := int(earliest.NotAfter.UTC().Sub(now).Hours() / 24)
	return earliest, days, which, true
}

func (s *DashboardService) findEarliestEtcdCert(ctx context.Context, clusterID uint64, maxNodes int) (*x509.Certificate, int, bool) {
	if s.k8sSvc == nil || clusterID == 0 {
		return nil, 0, false
	}
	cs, err := s.k8sSvc.typedClient(ctx, clusterID)
	if err != nil {
		return nil, 0, false
	}
	ips := listControlPlaneNodeIPs(ctx, cs, maxNodes)
	return findEarliestCertFromNodePorts(ctx, ips, "2379", 2*time.Second)
}

func (s *DashboardService) findEarliestControlPlaneCert(ctx context.Context, clusterID uint64, maxNodes int) (*x509.Certificate, int, string, bool) {
	if s.k8sSvc == nil || clusterID == 0 {
		return nil, 0, "", false
	}
	cs, err := s.k8sSvc.typedClient(ctx, clusterID)
	if err != nil {
		return nil, 0, "", false
	}
	ips := listControlPlaneNodeIPs(ctx, cs, maxNodes)
	return findEarliestControlPlaneCertFromNodeIPs(ctx, ips)
}

func (s *DashboardService) findEarliestKubeletCert(ctx context.Context, clusterID uint64, maxNodes int) (*x509.Certificate, int, bool) {
	if s.k8sSvc == nil || clusterID == 0 {
		return nil, 0, false
	}
	cs, err := s.k8sSvc.typedClient(ctx, clusterID)
	if err != nil {
		return nil, 0, false
	}
	nodes, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil || len(nodes.Items) == 0 {
		return nil, 0, false
	}

	type res struct {
		cert *x509.Certificate
		err  error
	}

	addrs := make([]string, 0, len(nodes.Items))
	for i := range nodes.Items {
		if maxNodes > 0 && len(addrs) >= maxNodes {
			break
		}
		ip := ""
		for _, a := range nodes.Items[i].Status.Addresses {
			if a.Type == corev1.NodeInternalIP && strings.TrimSpace(a.Address) != "" {
				ip = strings.TrimSpace(a.Address)
				break
			}
		}
		if ip == "" {
			for _, a := range nodes.Items[i].Status.Addresses {
				if a.Type == corev1.NodeExternalIP && strings.TrimSpace(a.Address) != "" {
					ip = strings.TrimSpace(a.Address)
					break
				}
			}
		}
		if ip != "" {
			addrs = append(addrs, ip)
		}
	}
	if len(addrs) == 0 {
		return nil, 0, false
	}

	timeout := 2 * time.Second
	sem := make(chan struct{}, 20)
	ch := make(chan res, len(addrs))
	for _, ip := range addrs {
		ip := ip
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			if ctx.Err() != nil {
				ch <- res{err: ctx.Err()}
				return
			}
			addr := net.JoinHostPort(ip, "10250")
			cert, err := fetchTLSServerCertificateByAddr(ctx, addr, ip, timeout)
			if err != nil {
				ch <- res{err: err}
				return
			}
			ch <- res{cert: cert}
		}()
	}

	var earliest *x509.Certificate
	for i := 0; i < len(addrs); i++ {
		r := <-ch
		if r.cert == nil {
			continue
		}
		if earliest == nil || r.cert.NotAfter.Before(earliest.NotAfter) {
			earliest = r.cert
		}
	}
	if earliest == nil {
		return nil, 0, false
	}
	now := time.Now().UTC()
	days := int(earliest.NotAfter.UTC().Sub(now).Hours() / 24)
	return earliest, days, true
}
