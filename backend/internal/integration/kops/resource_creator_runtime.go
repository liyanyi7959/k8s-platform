package kops

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
	"k8s.io/client-go/kubernetes"

	service "k8s-platform-backend/internal/kops/adapters/kubernetes"
	kopsapp "k8s-platform-backend/internal/kops/application"
)

// ResourceCreatorRuntime translates Kops creation inputs directly into typed
// Kubernetes resources. K8sService is only the client transport.
type ResourceCreatorRuntime struct{ transport *service.K8sService }

func NewResourceCreatorRuntime(transport *service.K8sService) *ResourceCreatorRuntime {
	return &ResourceCreatorRuntime{transport: transport}
}

func (r *ResourceCreatorRuntime) CreateDeployment(ctx context.Context, clusterID uint64, input kopsapp.WorkloadCreateInput) error {
	client, namespace, name, replicas, labels, containers, err := r.workloadClient(ctx, clusterID, input)
	if err != nil {
		return err
	}
	_, err = client.AppsV1().Deployments(namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas, Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{Containers: containers}},
		},
	}, metav1.CreateOptions{})
	return translateKopsRuntimeError(service.NormalizeKubernetesError(err))
}

func (r *ResourceCreatorRuntime) CreateStatefulSet(ctx context.Context, clusterID uint64, input kopsapp.WorkloadCreateInput) error {
	client, namespace, name, replicas, labels, containers, err := r.workloadClient(ctx, clusterID, input)
	if err != nil {
		return err
	}
	_, err = client.AppsV1().StatefulSets(namespace).Create(ctx, &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name, Replicas: &replicas, Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{Containers: containers}},
		},
	}, metav1.CreateOptions{})
	return translateKopsRuntimeError(service.NormalizeKubernetesError(err))
}

func (r *ResourceCreatorRuntime) CreateDaemonSet(ctx context.Context, clusterID uint64, input kopsapp.WorkloadCreateInput) error {
	client, namespace, name, _, labels, containers, err := r.workloadClient(ctx, clusterID, input)
	if err != nil {
		return err
	}
	_, err = client.AppsV1().DaemonSets(namespace).Create(ctx, &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{Containers: containers}},
		},
	}, metav1.CreateOptions{})
	return translateKopsRuntimeError(service.NormalizeKubernetesError(err))
}

func (r *ResourceCreatorRuntime) CreateService(ctx context.Context, clusterID uint64, input kopsapp.ServiceCreateInput) error {
	if r == nil || r.transport == nil {
		return kopsapp.ErrConflict
	}
	namespace, name := strings.TrimSpace(input.Namespace), strings.TrimSpace(input.Name)
	if namespace == "" || name == "" || len(input.Ports) == 0 {
		return kopsapp.ErrInvalidParams
	}
	serviceType := corev1.ServiceTypeClusterIP
	switch strings.TrimSpace(input.Type) {
	case "", "ClusterIP":
	case "NodePort":
		serviceType = corev1.ServiceTypeNodePort
	case "LoadBalancer":
		serviceType = corev1.ServiceTypeLoadBalancer
	default:
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "不支持的 Service 类型")
	}
	selector := input.Selector
	if selector == nil {
		selector = map[string]string{"app": name}
	}
	ports := make([]corev1.ServicePort, 0, len(input.Ports))
	for index, port := range input.Ports {
		if port.Port <= 0 {
			return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "端口号无效")
		}
		protocol := corev1.ProtocolTCP
		if strings.EqualFold(port.Protocol, "UDP") {
			protocol = corev1.ProtocolUDP
		}
		item := corev1.ServicePort{Name: strings.TrimSpace(port.Name), Port: port.Port, Protocol: protocol}
		if item.Name == "" {
			item.Name = fmt.Sprintf("%s-%d", strings.ToLower(string(protocol)), index)
		}
		if port.TargetPort > 0 {
			item.TargetPort = intstr.FromInt32(port.TargetPort)
		} else {
			item.TargetPort = intstr.FromInt32(port.Port)
		}
		ports = append(ports, item)
	}
	client, err := r.transport.TypedClient(ctx, clusterID)
	if err != nil {
		return translateKopsRuntimeError(err)
	}
	_, err = client.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       corev1.ServiceSpec{Type: serviceType, Selector: selector, Ports: ports},
	}, metav1.CreateOptions{})
	return translateKopsRuntimeError(service.NormalizeKubernetesError(err))
}

func (r *ResourceCreatorRuntime) CreateIngress(ctx context.Context, clusterID uint64, input kopsapp.IngressCreateInput) error {
	if r == nil || r.transport == nil {
		return kopsapp.ErrConflict
	}
	namespace, name := strings.TrimSpace(input.Namespace), strings.TrimSpace(input.Name)
	if namespace == "" || name == "" || len(input.Rules) == 0 {
		return kopsapp.ErrInvalidParams
	}
	rules := make([]networkingv1.IngressRule, 0, len(input.Rules))
	hosts := make([]string, 0, len(input.Rules))
	for _, rule := range input.Rules {
		host := strings.TrimSpace(rule.Host)
		if host != "" {
			hosts = append(hosts, host)
		}
		paths := make([]networkingv1.HTTPIngressPath, 0, len(rule.Paths))
		for _, path := range rule.Paths {
			serviceName := strings.TrimSpace(path.ServiceName)
			if serviceName == "" || path.ServicePort <= 0 {
				return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "路径必须指定后端服务和端口")
			}
			pathType := networkingv1.PathTypePrefix
			switch strings.TrimSpace(path.PathType) {
			case "Exact":
				pathType = networkingv1.PathTypeExact
			case "ImplementationSpecific":
				pathType = networkingv1.PathTypeImplementationSpecific
			}
			pathValue := strings.TrimSpace(path.Path)
			if pathValue == "" {
				pathValue = "/"
			}
			paths = append(paths, networkingv1.HTTPIngressPath{
				Path: pathValue, PathType: &pathType,
				Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
					Name: serviceName, Port: networkingv1.ServiceBackendPort{Number: path.ServicePort},
				}},
			})
		}
		rules = append(rules, networkingv1.IngressRule{Host: host, IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{Paths: paths},
		}})
	}
	annotations := input.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Annotations: annotations},
		Spec:       networkingv1.IngressSpec{Rules: rules},
	}
	if class := strings.TrimSpace(input.IngressClass); class != "" {
		ingress.Spec.IngressClassName = &class
	}
	if secret := strings.TrimSpace(input.TLSSecretName); secret != "" && len(hosts) > 0 {
		ingress.Spec.TLS = []networkingv1.IngressTLS{{Hosts: hosts, SecretName: secret}}
	}
	client, err := r.transport.TypedClient(ctx, clusterID)
	if err != nil {
		return translateKopsRuntimeError(err)
	}
	_, err = client.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
	return translateKopsRuntimeError(service.NormalizeKubernetesError(err))
}

func (r *ResourceCreatorRuntime) workloadClient(ctx context.Context, clusterID uint64, input kopsapp.WorkloadCreateInput) (*kubernetes.Clientset, string, string, int32, map[string]string, []corev1.Container, error) {
	if r == nil || r.transport == nil {
		return nil, "", "", 0, nil, nil, kopsapp.ErrConflict
	}
	namespace, name, replicas, labels, containers, err := buildWorkload(input)
	if err != nil {
		return nil, "", "", 0, nil, nil, err
	}
	client, err := r.transport.TypedClient(ctx, clusterID)
	if err != nil {
		return nil, "", "", 0, nil, nil, translateKopsRuntimeError(err)
	}
	return client, namespace, name, replicas, labels, containers, nil
}

func buildWorkload(input kopsapp.WorkloadCreateInput) (string, string, int32, map[string]string, []corev1.Container, error) {
	namespace, name := strings.TrimSpace(input.Namespace), strings.TrimSpace(input.Name)
	if namespace == "" || name == "" || len(input.Containers) == 0 {
		return "", "", 0, nil, nil, kopsapp.ErrInvalidParams
	}
	replicas := input.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	labels := make(map[string]string, len(input.Labels)+1)
	for key, value := range input.Labels {
		if key = strings.TrimSpace(key); key != "" {
			labels[key] = strings.TrimSpace(value)
		}
	}
	labels["app"] = name
	containers := make([]corev1.Container, 0, len(input.Containers))
	for _, item := range input.Containers {
		containerName, image := strings.TrimSpace(item.Name), strings.TrimSpace(item.Image)
		if containerName == "" || image == "" {
			return "", "", 0, nil, nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "容器名和镜像不能为空")
		}
		container := corev1.Container{Name: containerName, Image: image}
		resources := corev1.ResourceRequirements{Requests: corev1.ResourceList{}, Limits: corev1.ResourceList{}}
		if cpu := strings.TrimSpace(item.CPU); cpu != "" {
			quantity, err := apiresource.ParseQuantity(cpu)
			if err != nil {
				return "", "", 0, nil, nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "CPU 格式无效")
			}
			resources.Requests[corev1.ResourceCPU], resources.Limits[corev1.ResourceCPU] = quantity, quantity
		}
		if memory := strings.TrimSpace(item.Memory); memory != "" {
			quantity, err := apiresource.ParseQuantity(memory)
			if err != nil {
				return "", "", 0, nil, nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "内存格式无效")
			}
			resources.Requests[corev1.ResourceMemory], resources.Limits[corev1.ResourceMemory] = quantity, quantity
		}
		if len(resources.Requests) > 0 {
			container.Resources = resources
		}
		if command := strings.TrimSpace(item.Command); command != "" {
			container.Command = strings.Fields(command)
		}
		containers = append(containers, container)
	}
	return namespace, name, replicas, labels, containers, nil
}
