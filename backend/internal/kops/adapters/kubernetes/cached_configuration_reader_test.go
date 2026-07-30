package kubernetes

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

type cachedConfigurationTransportStub struct {
	cache map[string][]byte
}

func (s *cachedConfigurationTransportStub) TypedClient(context.Context, uint64) (*kubernetes.Clientset, error) {
	return nil, fmt.Errorf("typed client must not be called")
}

func (s *cachedConfigurationTransportStub) ObjectInformer(context.Context, uint64, string) (cache.SharedIndexInformer, error) {
	return nil, fmt.Errorf("informer must not be called")
}

func (*cachedConfigurationTransportStub) ObjectCacheEnabled() bool      { return true }
func (*cachedConfigurationTransportStub) ObjectCacheTTL() time.Duration { return time.Minute }
func (s *cachedConfigurationTransportStub) ObjectCacheGet(_ context.Context, key string) ([]byte, bool, error) {
	value, ok := s.cache[key]
	return value, ok, nil
}
func (s *cachedConfigurationTransportStub) ObjectCacheSet(_ context.Context, key string, value []byte, _ time.Duration) error {
	s.cache[key] = append([]byte(nil), value...)
	return nil
}
func (*cachedConfigurationTransportStub) NormalizeKubernetesRuntimeError(err error) error { return err }

func TestCachedConfigurationReaderMasksSecretsReadFromRedis(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "registry", Namespace: "prod"},
		Data:       map[string][]byte{"token": []byte("raw-secret")},
	}
	// Build the exact cache payload through the runtime's typed codec so the
	// test exercises the same Redis representation as production.
	encoded, err := marshalCachedConfigurationObjects(CachedSecrets, []k8sruntime.Object{secret})
	if err != nil {
		t.Fatalf("marshal cache payload: %v", err)
	}
	transport := &cachedConfigurationTransportStub{cache: map[string][]byte{
		kopsapp.ObjectListCacheKey(7, string(CachedSecrets)): encoded,
	}}
	reader := NewCachedConfigurationReader(transport)
	items, err := reader.List(context.Background(), 7, CachedSecrets, "prod", "metadata.name", "asc")
	if err != nil {
		t.Fatalf("list secrets: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("secret count = %d, want 1", len(items))
	}
	object := items[0].(map[string]any)
	data, ok := object["data"].(map[string]any)
	if !ok {
		t.Fatalf("secret data = %#v", object["data"])
	}
	if strings.Contains(fmt.Sprint(data["token"]), base64.StdEncoding.EncodeToString([]byte("raw-secret"))) {
		t.Fatalf("secret value leaked from cache: %#v", data)
	}
}

func TestCachedConfigurationReaderFiltersAndSortsSnapshots(t *testing.T) {
	objects := []k8sruntime.Object{
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "zeta", Namespace: "ops"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "alpha", Namespace: "ops"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "default"}},
	}
	filtered := filterCachedObjects(objects, "ops", nil)
	sortCachedConfigurationObjects(filtered, "metadata.name", "asc")
	if len(filtered) != 2 {
		t.Fatalf("filtered count = %d, want 2", len(filtered))
	}
	if filtered[0].(metav1.Object).GetName() != "alpha" || filtered[1].(metav1.Object).GetName() != "zeta" {
		t.Fatalf("sorted names = %q, %q", filtered[0].(metav1.Object).GetName(), filtered[1].(metav1.Object).GetName())
	}
}

func TestCachedConfigurationReaderFiltersPodSnapshotByLabel(t *testing.T) {
	objects := []k8sruntime.Object{
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "ops", Labels: map[string]string{"app": "api"}}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "worker", Namespace: "ops", Labels: map[string]string{"app": "worker"}}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api-default", Namespace: "default", Labels: map[string]string{"app": "api"}}},
	}
	payload, err := marshalCachedConfigurationObjects(CachedPods, objects)
	if err != nil {
		t.Fatalf("marshal pod cache: %v", err)
	}
	transport := &cachedConfigurationTransportStub{cache: map[string][]byte{
		kopsapp.ObjectListCacheKey(7, string(CachedPods)): payload,
	}}
	items, err := NewCachedConfigurationReader(transport).ListWithLabelSelector(context.Background(), 7, CachedPods, "ops", "metadata.name", "asc", "app=api")
	if err != nil {
		t.Fatalf("list pods: %v", err)
	}
	if len(items) != 1 || items[0].(map[string]any)["metadata"].(map[string]any)["name"] != "api" {
		t.Fatalf("filtered pod snapshot = %#v", items)
	}
}

func TestCachedConfigurationCodecsCoverServiceAccountsAndHPAs(t *testing.T) {
	for _, test := range []struct {
		name     string
		resource CachedConfigurationResource
		objects  []k8sruntime.Object
	}{
		{
			name:     "service accounts",
			resource: CachedServiceAccounts,
			objects:  []k8sruntime.Object{&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "runner", Namespace: "ops"}}},
		},
		{
			name:     "hpas",
			resource: CachedHPAs,
			objects:  []k8sruntime.Object{&autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "ops"}}},
		},
		{
			name:     "deployments",
			resource: CachedDeployments,
			objects:  []k8sruntime.Object{&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "ops"}}},
		},
		{
			name:     "statefulsets",
			resource: CachedStatefulSets,
			objects:  []k8sruntime.Object{&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "ops"}}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload, err := marshalCachedConfigurationObjects(test.resource, test.objects)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			decoded, err := unmarshalCachedConfigurationObjects(test.resource, payload)
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(decoded) != 1 || decoded[0].(metav1.Object).GetName() != test.objects[0].(metav1.Object).GetName() {
				t.Fatalf("decoded objects = %#v", decoded)
			}
		})
	}
}
