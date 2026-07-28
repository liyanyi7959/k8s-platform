package service

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ──────────────────────────────────────────────────────────
// Create Deployment
// ──────────────────────────────────────────────────────────

type CreateDeploymentContainer struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	CPU     string `json:"cpu"`     // e.g. "100m"
	Memory  string `json:"memory"`  // e.g. "128Mi"
	Command string `json:"command"` // optional, space-separated
}

type CreateDeploymentInput struct {
	Namespace  string                      `json:"namespace"`
	Name       string                      `json:"name"`
	Replicas   int32                       `json:"replicas"`
	Containers []CreateDeploymentContainer `json:"containers"`
	Labels     map[string]string           `json:"labels"`
}

func buildCreateWorkloadSpec(input CreateDeploymentInput) (string, string, int32, map[string]string, []corev1.Container, error) {
	namespace := strings.TrimSpace(input.Namespace)
	name := strings.TrimSpace(input.Name)
	if namespace == "" || name == "" || len(input.Containers) == 0 {
		return "", "", 0, nil, nil, ErrInvalidParams
	}

	replicas := input.Replicas
	if replicas <= 0 {
		replicas = 1
	}

	labels := map[string]string{}
	for key, value := range input.Labels {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		labels[trimmedKey] = strings.TrimSpace(value)
	}
	labels["app"] = name

	containers := make([]corev1.Container, 0, len(input.Containers))
	for _, c := range input.Containers {
		cName := strings.TrimSpace(c.Name)
		cImage := strings.TrimSpace(c.Image)
		if cName == "" || cImage == "" {
			return "", "", 0, nil, nil, ErrWithMessage(ErrInvalidParams, "容器名和镜像不能为空")
		}
		container := corev1.Container{
			Name:  cName,
			Image: cImage,
		}
		resources := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{},
			Limits:   corev1.ResourceList{},
		}
		if cpu := strings.TrimSpace(c.CPU); cpu != "" {
			q, err := apiresource.ParseQuantity(cpu)
			if err != nil {
				return "", "", 0, nil, nil, ErrWithMessage(ErrInvalidParams, "CPU 格式无效")
			}
			resources.Requests[corev1.ResourceCPU] = q
			resources.Limits[corev1.ResourceCPU] = q
		}
		if mem := strings.TrimSpace(c.Memory); mem != "" {
			q, err := apiresource.ParseQuantity(mem)
			if err != nil {
				return "", "", 0, nil, nil, ErrWithMessage(ErrInvalidParams, "内存格式无效")
			}
			resources.Requests[corev1.ResourceMemory] = q
			resources.Limits[corev1.ResourceMemory] = q
		}
		if len(resources.Requests) > 0 {
			container.Resources = resources
		}
		if cmd := strings.TrimSpace(c.Command); cmd != "" {
			container.Command = strings.Fields(cmd)
		}
		containers = append(containers, container)
	}

	return namespace, name, replicas, labels, containers, nil
}

func (s *K8sService) CreateDeployment(ctx context.Context, clusterID uint64, input CreateDeploymentInput) error {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}

	namespace, name, replicas, labels, containers, err := buildCreateWorkloadSpec(input)
	if err != nil {
		return err
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		},
	}

	_, err = cs.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	return normalizeK8sErr(err)
}

func (s *K8sService) CreateStatefulSet(ctx context.Context, clusterID uint64, input CreateDeploymentInput) error {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}

	namespace, name, replicas, labels, containers, err := buildCreateWorkloadSpec(input)
	if err != nil {
		return err
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name,
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		},
	}

	_, err = cs.AppsV1().StatefulSets(namespace).Create(ctx, statefulSet, metav1.CreateOptions{})
	return normalizeK8sErr(err)
}

func (s *K8sService) CreateDaemonSet(ctx context.Context, clusterID uint64, input CreateDeploymentInput) error {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}

	namespace, name, _, labels, containers, err := buildCreateWorkloadSpec(input)
	if err != nil {
		return err
	}

	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		},
	}

	_, err = cs.AppsV1().DaemonSets(namespace).Create(ctx, daemonSet, metav1.CreateOptions{})
	return normalizeK8sErr(err)
}

// ──────────────────────────────────────────────────────────
// Create Service
// ──────────────────────────────────────────────────────────

type CreateServicePort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort int32  `json:"target_port"`
	Protocol   string `json:"protocol"` // TCP / UDP
}

type CreateServiceInput struct {
	Namespace string              `json:"namespace"`
	Name      string              `json:"name"`
	Type      string              `json:"type"` // ClusterIP / NodePort / LoadBalancer
	Selector  map[string]string   `json:"selector"`
	Ports     []CreateServicePort `json:"ports"`
}

func (s *K8sService) CreateService(ctx context.Context, clusterID uint64, input CreateServiceInput) error {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}

	namespace := strings.TrimSpace(input.Namespace)
	name := strings.TrimSpace(input.Name)
	if namespace == "" || name == "" || len(input.Ports) == 0 {
		return ErrInvalidParams
	}

	svcType := corev1.ServiceTypeClusterIP
	switch strings.TrimSpace(input.Type) {
	case "NodePort":
		svcType = corev1.ServiceTypeNodePort
	case "LoadBalancer":
		svcType = corev1.ServiceTypeLoadBalancer
	case "ClusterIP", "":
		svcType = corev1.ServiceTypeClusterIP
	default:
		return ErrWithMessage(ErrInvalidParams, "不支持的 Service 类型")
	}

	selector := input.Selector
	if selector == nil {
		selector = map[string]string{"app": name}
	}

	ports := make([]corev1.ServicePort, 0, len(input.Ports))
	for i, p := range input.Ports {
		if p.Port <= 0 {
			return ErrWithMessage(ErrInvalidParams, "端口号无效")
		}
		protocol := corev1.ProtocolTCP
		if strings.EqualFold(p.Protocol, "UDP") {
			protocol = corev1.ProtocolUDP
		}
		sp := corev1.ServicePort{
			Name:     strings.TrimSpace(p.Name),
			Port:     p.Port,
			Protocol: protocol,
		}
		if p.TargetPort > 0 {
			sp.TargetPort = intstr.FromInt32(p.TargetPort)
		} else {
			sp.TargetPort = intstr.FromInt32(p.Port)
		}
		if sp.Name == "" {
			sp.Name = fmt.Sprintf("%s-%d", strings.ToLower(string(protocol)), i)
		}
		ports = append(ports, sp)
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:     svcType,
			Selector: selector,
			Ports:    ports,
		},
	}

	_, err = cs.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	return normalizeK8sErr(err)
}

// ──────────────────────────────────────────────────────────
// Create Ingress
// ──────────────────────────────────────────────────────────

type CreateIngressRule struct {
	Host  string              `json:"host"`
	Paths []CreateIngressPath `json:"paths"`
}

type CreateIngressPath struct {
	Path        string `json:"path"`
	PathType    string `json:"path_type"` // Prefix / Exact / ImplementationSpecific
	ServiceName string `json:"service_name"`
	ServicePort int32  `json:"service_port"`
}

type CreateIngressInput struct {
	Namespace     string              `json:"namespace"`
	Name          string              `json:"name"`
	IngressClass  string              `json:"ingress_class"`
	Rules         []CreateIngressRule `json:"rules"`
	TLSSecretName string              `json:"tls_secret_name"`
	Annotations   map[string]string   `json:"annotations"`
}

func (s *K8sService) CreateIngress(ctx context.Context, clusterID uint64, input CreateIngressInput) error {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}

	namespace := strings.TrimSpace(input.Namespace)
	name := strings.TrimSpace(input.Name)
	if namespace == "" || name == "" || len(input.Rules) == 0 {
		return ErrInvalidParams
	}

	rules := make([]networkingv1.IngressRule, 0, len(input.Rules))
	var tlsHosts []string
	for _, r := range input.Rules {
		host := strings.TrimSpace(r.Host)
		if host != "" {
			tlsHosts = append(tlsHosts, host)
		}
		paths := make([]networkingv1.HTTPIngressPath, 0, len(r.Paths))
		for _, p := range r.Paths {
			svcName := strings.TrimSpace(p.ServiceName)
			if svcName == "" || p.ServicePort <= 0 {
				return ErrWithMessage(ErrInvalidParams, "路径必须指定后端服务和端口")
			}
			pathType := networkingv1.PathTypePrefix
			switch strings.TrimSpace(p.PathType) {
			case "Exact":
				pathType = networkingv1.PathTypeExact
			case "ImplementationSpecific":
				pathType = networkingv1.PathTypeImplementationSpecific
			}
			pathStr := strings.TrimSpace(p.Path)
			if pathStr == "" {
				pathStr = "/"
			}
			paths = append(paths, networkingv1.HTTPIngressPath{
				Path:     pathStr,
				PathType: &pathType,
				Backend: networkingv1.IngressBackend{
					Service: &networkingv1.IngressServiceBackend{
						Name: svcName,
						Port: networkingv1.ServiceBackendPort{
							Number: p.ServicePort,
						},
					},
				},
			})
		}
		rule := networkingv1.IngressRule{
			Host: host,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: paths,
				},
			},
		}
		rules = append(rules, rule)
	}

	annotations := input.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: networkingv1.IngressSpec{
			Rules: rules,
		},
	}

	if class := strings.TrimSpace(input.IngressClass); class != "" {
		ingress.Spec.IngressClassName = &class
	}

	if secret := strings.TrimSpace(input.TLSSecretName); secret != "" && len(tlsHosts) > 0 {
		ingress.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      tlsHosts,
				SecretName: secret,
			},
		}
	}

	_, err = cs.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
	return normalizeK8sErr(err)
}
