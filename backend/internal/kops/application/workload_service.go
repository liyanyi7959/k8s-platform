package application

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type WorkloadKind string

const (
	WorkloadDeployment  WorkloadKind = "Deployment"
	WorkloadStatefulSet WorkloadKind = "StatefulSet"
	WorkloadDaemonSet   WorkloadKind = "DaemonSet"
)

type WorkloadQuery struct {
	ClusterID                               uint64
	Kind                                    WorkloadKind
	Namespace, LabelSelector, SortBy, Order string
}

type WorkloadRef struct {
	ClusterID       uint64
	Kind            WorkloadKind
	Namespace, Name string
}

type WorkloadScale struct {
	WorkloadRef
	Replicas int
}

type WorkloadImage struct {
	ClusterID uint64       `json:"-"`
	Kind      WorkloadKind `json:"kind"`
	Namespace string       `json:"namespace"`
	Name      string       `json:"name"`
	Container string       `json:"container"`
	Image     string       `json:"image"`
}

type WorkloadPause struct {
	ClusterID uint64       `json:"-"`
	Kind      WorkloadKind `json:"kind"`
	Namespace string       `json:"namespace"`
	Name      string       `json:"name"`
	Paused    bool         `json:"paused"`
}

type WorkloadYAMLEdit struct {
	ClusterID uint64       `json:"-"`
	Kind      WorkloadKind `json:"kind"`
	Namespace string       `json:"namespace"`
	YAML      string       `json:"yaml"`
}

type WorkloadEditInput struct {
	ClusterID uint64       `json:"-"`
	Kind      WorkloadKind `json:"-"`
	Namespace string       `json:"namespace"`
	Name      string       `json:"name"`

	Replicas       *int                    `json:"replicas"`
	Labels         map[string]string       `json:"labels"`
	Tolerations    []WorkloadToleration    `json:"tolerations"`
	Containers     []WorkloadEditContainer `json:"containers"`
	InitContainers []WorkloadEditContainer `json:"initContainers"`
	Strategy       *WorkloadStrategy       `json:"strategy"`
	Volumes        []map[string]any        `json:"volumes"`
}

type WorkloadStrategy struct {
	Type           string `json:"type"`
	MaxSurge       string `json:"maxSurge"`
	MaxUnavailable string `json:"maxUnavailable"`
}

type WorkloadToleration struct {
	Key               *string `json:"key"`
	Operator          *string `json:"operator"`
	Value             *string `json:"value"`
	Effect            *string `json:"effect"`
	TolerationSeconds *int64  `json:"tolerationSeconds"`
}

type WorkloadEditContainer struct {
	Name            string                   `json:"name"`
	Image           *string                  `json:"image"`
	ImagePullPolicy *string                  `json:"imagePullPolicy"`
	Resources       *WorkloadResources       `json:"resources"`
	Probes          *WorkloadContainerProbes `json:"probes"`
	Env             []map[string]any         `json:"env"`
	EnvFrom         []map[string]any         `json:"envFrom"`
	VolumeMounts    []map[string]any         `json:"volumeMounts"`
}

type WorkloadResources struct {
	Requests map[string]string `json:"requests"`
	Limits   map[string]string `json:"limits"`
}

type WorkloadContainerProbes struct {
	Liveness  *WorkloadProbeTiming `json:"liveness"`
	Readiness *WorkloadProbeTiming `json:"readiness"`
	Startup   *WorkloadProbeTiming `json:"startup"`
}

type WorkloadProbeTiming struct {
	InitialDelaySeconds *int32 `json:"initialDelaySeconds"`
	TimeoutSeconds      *int32 `json:"timeoutSeconds"`
	PeriodSeconds       *int32 `json:"periodSeconds"`
	SuccessThreshold    *int32 `json:"successThreshold"`
	FailureThreshold    *int32 `json:"failureThreshold"`
}

type WorkloadRuntime interface {
	List(context.Context, WorkloadQuery) (any, error)
	History(context.Context, WorkloadRef) (any, error)
	Undo(context.Context, WorkloadRef, int) error
	Patch(context.Context, WorkloadRef, map[string]any) error
	Image(context.Context, WorkloadImage) error
	Pause(context.Context, WorkloadRef, bool) error
	Object(context.Context, WorkloadRef) (map[string]any, error)
	ApplyYAML(context.Context, WorkloadYAMLEdit) error
	Delete(context.Context, WorkloadRef) error
	YAML(context.Context, WorkloadRef) (string, error)
}

type WorkloadService struct{ runtime WorkloadRuntime }

func NewWorkloadService(runtime WorkloadRuntime) *WorkloadService {
	return &WorkloadService{runtime: runtime}
}

func (s *WorkloadService) List(ctx context.Context, q WorkloadQuery) (any, error) {
	if q.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	if q.Kind != "" {
		kind, ok := canonicalWorkloadKind(q.Kind)
		if !ok {
			return nil, ErrInvalidParams
		}
		q.Kind = kind
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	q.Namespace = strings.TrimSpace(q.Namespace)
	q.LabelSelector = strings.TrimSpace(q.LabelSelector)
	q.SortBy = strings.TrimSpace(q.SortBy)
	q.Order = strings.TrimSpace(q.Order)
	return s.runtime.List(ctx, q)
}

func (s *WorkloadService) History(ctx context.Context, ref WorkloadRef) (any, error) {
	if err := validWorkloadRef(ref); err != nil || ref.Kind != WorkloadDeployment {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.History(ctx, normalizeWorkloadRef(ref))
}

func (s *WorkloadService) Undo(ctx context.Context, ref WorkloadRef, revision int) error {
	if err := validWorkloadRef(ref); err != nil || ref.Kind != WorkloadDeployment || revision < 0 {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Undo(ctx, normalizeWorkloadRef(ref), revision)
}

func (s *WorkloadService) Scale(ctx context.Context, input WorkloadScale) error {
	if err := validWorkloadRef(input.WorkloadRef); err != nil || input.Kind == WorkloadDaemonSet || input.Replicas < 0 {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Patch(ctx, normalizeWorkloadRef(input.WorkloadRef), map[string]any{"spec": map[string]any{"replicas": input.Replicas}})
}

func (s *WorkloadService) Restart(ctx context.Context, ref WorkloadRef) error {
	if err := validWorkloadRef(ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Patch(ctx, normalizeWorkloadRef(ref), map[string]any{"spec": map[string]any{"template": map[string]any{"metadata": map[string]any{"annotations": map[string]any{"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339)}}}}})
}

func (s *WorkloadService) Image(ctx context.Context, input WorkloadImage) error {
	ref := WorkloadRef{ClusterID: input.ClusterID, Kind: input.Kind, Namespace: input.Namespace, Name: input.Name}
	if err := validWorkloadRef(ref); err != nil || strings.TrimSpace(input.Container) == "" || strings.TrimSpace(input.Image) == "" {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	input.ClusterID = ref.ClusterID
	input.Kind = normalizeWorkloadRef(ref).Kind
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.Name = strings.TrimSpace(input.Name)
	input.Container = strings.TrimSpace(input.Container)
	input.Image = strings.TrimSpace(input.Image)
	return s.runtime.Image(ctx, input)
}

func (s *WorkloadService) Pause(ctx context.Context, input WorkloadPause) error {
	ref := WorkloadRef{ClusterID: input.ClusterID, Kind: input.Kind, Namespace: input.Namespace, Name: input.Name}
	if err := validWorkloadRef(ref); err != nil || ref.Kind != WorkloadDeployment {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Pause(ctx, normalizeWorkloadRef(ref), input.Paused)
}

func (s *WorkloadService) Edit(ctx context.Context, input WorkloadEditInput) error {
	ref := WorkloadRef{ClusterID: input.ClusterID, Kind: input.Kind, Namespace: input.Namespace, Name: input.Name}
	if err := validWorkloadRef(ref); err != nil {
		return err
	}
	if input.Replicas != nil && *input.Replicas < 0 {
		return ErrInvalidParams
	}
	if ref.Kind == WorkloadDaemonSet && input.Replicas != nil {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	ref = normalizeWorkloadRef(ref)
	input.ClusterID, input.Kind, input.Namespace, input.Name = ref.ClusterID, ref.Kind, ref.Namespace, ref.Name

	object, err := s.runtime.Object(ctx, ref)
	if err != nil {
		return err
	}
	patch, err := buildWorkloadEditPatch(object, input)
	if err != nil {
		return err
	}
	if len(patch) == 0 {
		return nil
	}
	return s.runtime.Patch(ctx, ref, map[string]any{"spec": patch})
}

func (s *WorkloadService) ApplyYAML(ctx context.Context, input WorkloadYAMLEdit) error {
	if input.ClusterID == 0 || !validWorkloadKind(input.Kind) || strings.TrimSpace(input.YAML) == "" {
		return ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	input.Kind = normalizeWorkloadKind(input.Kind)
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.YAML = strings.TrimSpace(input.YAML)
	return s.runtime.ApplyYAML(ctx, input)
}

func (s *WorkloadService) Delete(ctx context.Context, ref WorkloadRef) error {
	if err := validWorkloadRef(ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Delete(ctx, normalizeWorkloadRef(ref))
}

func (s *WorkloadService) YAML(ctx context.Context, ref WorkloadRef) (string, error) {
	if err := validWorkloadRef(ref); err != nil {
		return "", err
	}
	if s == nil || s.runtime == nil {
		return "", ErrConflict
	}
	return s.runtime.YAML(ctx, normalizeWorkloadRef(ref))
}

func buildWorkloadEditPatch(object map[string]any, input WorkloadEditInput) (map[string]any, error) {
	spec, _ := object["spec"].(map[string]any)
	if spec == nil {
		return nil, ErrInvalidParams
	}
	selectorLabels := workloadSelectorLabels(spec)
	template, _ := spec["template"].(map[string]any)
	templateSpec, _ := template["spec"].(map[string]any)
	containers, _ := templateSpec["containers"].([]any)
	if len(containers) == 0 {
		return nil, ErrInvalidParams
	}
	initContainers, _ := templateSpec["initContainers"].([]any)
	if len(input.InitContainers) > 0 && len(initContainers) == 0 {
		return nil, ErrInvalidParams
	}
	if err := applyWorkloadContainerUpdates(containers, input.Containers); err != nil {
		return nil, err
	}
	if err := applyWorkloadContainerUpdates(initContainers, input.InitContainers); err != nil {
		return nil, err
	}
	strategyKey := "updateStrategy"
	if input.Kind == WorkloadDeployment {
		strategyKey = "strategy"
	}
	return workloadPatchSpec(input, selectorLabels, containers, initContainers, strategyKey)
}

func workloadSelectorLabels(spec map[string]any) map[string]string {
	result := map[string]string{}
	selector, _ := spec["selector"].(map[string]any)
	matchLabels, _ := selector["matchLabels"].(map[string]any)
	for key, value := range matchLabels {
		key = strings.TrimSpace(key)
		if key != "" {
			result[key] = workloadString(value)
		}
	}
	return result
}

func applyWorkloadContainerUpdates(containers []any, updates []WorkloadEditContainer) error {
	byName := make(map[string]map[string]any, len(containers))
	for _, value := range containers {
		container, _ := value.(map[string]any)
		if container == nil {
			continue
		}
		if name := strings.TrimSpace(workloadString(container["name"])); name != "" {
			byName[name] = container
		}
	}
	for _, update := range updates {
		name := strings.TrimSpace(update.Name)
		container := byName[name]
		if name == "" || container == nil {
			return ErrInvalidParams
		}
		if update.Image != nil {
			container["image"] = strings.TrimSpace(*update.Image)
		}
		if update.ImagePullPolicy != nil {
			value := strings.TrimSpace(*update.ImagePullPolicy)
			if value == "" {
				delete(container, "imagePullPolicy")
			} else {
				container["imagePullPolicy"] = value
			}
		}
		ensureWorkloadResources(container, update.Resources)
		if update.Probes != nil {
			applyWorkloadProbeTiming(container, "livenessProbe", update.Probes.Liveness)
			applyWorkloadProbeTiming(container, "readinessProbe", update.Probes.Readiness)
			applyWorkloadProbeTiming(container, "startupProbe", update.Probes.Startup)
		}
		if update.Env != nil {
			container["env"] = mapsToAny(update.Env)
		}
		if update.EnvFrom != nil {
			container["envFrom"] = mapsToAny(update.EnvFrom)
		}
		if update.VolumeMounts != nil {
			container["volumeMounts"] = mapsToAny(update.VolumeMounts)
		}
	}
	return nil
}

func workloadPatchSpec(input WorkloadEditInput, selectorLabels map[string]string, containers, initContainers []any, strategyKey string) (map[string]any, error) {
	patch := map[string]any{}
	if input.Replicas != nil {
		patch["replicas"] = *input.Replicas
	}
	if input.Labels != nil {
		labels := map[string]any{}
		for key, value := range input.Labels {
			if key = strings.TrimSpace(key); key != "" {
				labels[key] = strings.TrimSpace(value)
			}
		}
		for key, value := range selectorLabels {
			if workloadString(labels[key]) != value {
				return nil, ErrInvalidParams
			}
		}
		patch["template"] = mergeWorkloadMap(patch["template"], map[string]any{"metadata": map[string]any{"labels": labels}})
	}
	if input.Tolerations != nil {
		tolerations := make([]any, 0, len(input.Tolerations))
		for _, inputToleration := range input.Tolerations {
			toleration, err := workloadToleration(inputToleration)
			if err != nil {
				return nil, err
			}
			tolerations = append(tolerations, toleration)
		}
		patch["template"] = mergeWorkloadMap(patch["template"], map[string]any{"spec": map[string]any{"tolerations": tolerations}})
	}
	if len(input.Containers) > 0 {
		patch["template"] = mergeWorkloadMap(patch["template"], map[string]any{"spec": map[string]any{"containers": containers}})
	}
	if len(input.InitContainers) > 0 {
		patch["template"] = mergeWorkloadMap(patch["template"], map[string]any{"spec": map[string]any{"initContainers": initContainers}})
	}
	if input.Strategy != nil {
		strategy := map[string]any{"type": input.Strategy.Type}
		if input.Strategy.Type == "RollingUpdate" {
			rolling := map[string]any{}
			if input.Strategy.MaxSurge != "" {
				rolling["maxSurge"] = input.Strategy.MaxSurge
			}
			if input.Strategy.MaxUnavailable != "" {
				rolling["maxUnavailable"] = input.Strategy.MaxUnavailable
			}
			if len(rolling) > 0 {
				strategy["rollingUpdate"] = rolling
			}
		}
		patch[strategyKey] = strategy
	}
	if input.Volumes != nil {
		patch["template"] = mergeWorkloadMap(patch["template"], map[string]any{"spec": map[string]any{"volumes": mapsToAny(input.Volumes)}})
	}
	return patch, nil
}

func workloadToleration(input WorkloadToleration) (map[string]any, error) {
	result := map[string]any{}
	if input.Key != nil {
		result["key"] = strings.TrimSpace(*input.Key)
	}
	if input.Operator != nil {
		result["operator"] = strings.TrimSpace(*input.Operator)
	}
	if input.Value != nil {
		result["value"] = strings.TrimSpace(*input.Value)
	}
	if input.Effect != nil {
		result["effect"] = strings.TrimSpace(*input.Effect)
	}
	if input.TolerationSeconds != nil {
		if *input.TolerationSeconds < 0 {
			return nil, ErrInvalidParams
		}
		result["tolerationSeconds"] = *input.TolerationSeconds
	}
	return result, nil
}

func ensureWorkloadResources(container map[string]any, resources *WorkloadResources) {
	if container == nil || resources == nil {
		return
	}
	result, _ := container["resources"].(map[string]any)
	if result == nil {
		result = map[string]any{}
		container["resources"] = result
	}
	if requests := workloadResourceValues(resources.Requests); len(requests) > 0 {
		result["requests"] = requests
	}
	if limits := workloadResourceValues(resources.Limits); len(limits) > 0 {
		result["limits"] = limits
	}
}

func workloadResourceValues(values map[string]string) map[string]any {
	result := map[string]any{}
	for key, value := range values {
		if key, value = strings.TrimSpace(key), strings.TrimSpace(value); key != "" && value != "" {
			result[key] = value
		}
	}
	return result
}

func applyWorkloadProbeTiming(container map[string]any, key string, timing *WorkloadProbeTiming) {
	if container == nil || timing == nil || (timing.InitialDelaySeconds == nil && timing.TimeoutSeconds == nil && timing.PeriodSeconds == nil && timing.SuccessThreshold == nil && timing.FailureThreshold == nil) {
		return
	}
	probe, _ := container[key].(map[string]any)
	if probe == nil {
		probe = map[string]any{}
		container[key] = probe
	}
	if timing.InitialDelaySeconds != nil {
		probe["initialDelaySeconds"] = *timing.InitialDelaySeconds
	}
	if timing.TimeoutSeconds != nil {
		probe["timeoutSeconds"] = *timing.TimeoutSeconds
	}
	if timing.PeriodSeconds != nil {
		probe["periodSeconds"] = *timing.PeriodSeconds
	}
	if timing.SuccessThreshold != nil {
		probe["successThreshold"] = *timing.SuccessThreshold
	}
	if timing.FailureThreshold != nil {
		probe["failureThreshold"] = *timing.FailureThreshold
	}
}

func mapsToAny(values []map[string]any) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}

func mergeWorkloadMap(current any, added map[string]any) map[string]any {
	result, _ := current.(map[string]any)
	if result == nil {
		result = map[string]any{}
	}
	for key, value := range added {
		if nested, ok := value.(map[string]any); ok {
			result[key] = mergeWorkloadMap(result[key], nested)
			continue
		}
		result[key] = value
	}
	return result
}

func validWorkloadKind(kind WorkloadKind) bool {
	_, ok := canonicalWorkloadKind(kind)
	return ok
}

func canonicalWorkloadKind(kind WorkloadKind) (WorkloadKind, bool) {
	switch strings.ToLower(strings.TrimSpace(string(kind))) {
	case "deployment", "deployments":
		return WorkloadDeployment, true
	case "statefulset", "statefulsets":
		return WorkloadStatefulSet, true
	case "daemonset", "daemonsets":
		return WorkloadDaemonSet, true
	default:
		return "", false
	}
}

func normalizeWorkloadKind(kind WorkloadKind) WorkloadKind {
	normalized, _ := canonicalWorkloadKind(kind)
	return normalized
}

func validWorkloadRef(ref WorkloadRef) error {
	if ref.ClusterID == 0 || !validWorkloadKind(ref.Kind) || strings.TrimSpace(ref.Namespace) == "" || strings.TrimSpace(ref.Name) == "" {
		return ErrInvalidParams
	}
	return nil
}

func normalizeWorkloadRef(ref WorkloadRef) WorkloadRef {
	ref.Kind = normalizeWorkloadKind(ref.Kind)
	ref.Namespace = strings.TrimSpace(ref.Namespace)
	ref.Name = strings.TrimSpace(ref.Name)
	return ref
}

func workloadString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
