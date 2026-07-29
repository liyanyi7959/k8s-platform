package application

import (
	"context"
	"errors"
	"testing"
)

type podRuntimeSpy struct{ ref PodReference }

func (spy *podRuntimeSpy) List(context.Context, PodListQuery) (any, error)    { return nil, nil }
func (spy *podRuntimeSpy) Metrics(context.Context, PodListQuery) (any, error) { return nil, nil }
func (spy *podRuntimeSpy) YAML(_ context.Context, ref PodReference) (any, error) {
	spy.ref = ref
	return nil, nil
}
func (spy *podRuntimeSpy) Logs(context.Context, PodLogsInput) (any, error)  { return nil, nil }
func (spy *podRuntimeSpy) Delete(context.Context, PodReference, bool) error { return nil }
func TestPodServiceValidatesReferencesAndSessions(t *testing.T) {
	spy := &podRuntimeSpy{}
	sessions := NewPodLogSessionStore(0)
	defer sessions.Close()
	execs := NewExecSessionStore(0)
	defer execs.Close()
	service := NewPodService(spy, sessions, execs)
	if _, err := service.YAML(context.Background(), PodReference{ClusterID: 2, Namespace: " ops ", Name: " api "}); err != nil || spy.ref.Namespace != "ops" || spy.ref.Name != "api" {
		t.Fatalf("YAML() ref=%#v err=%v", spy.ref, err)
	}
	negative := int64(-1)
	if _, err := service.CreateLogSession(PodLogSessionInput{PodReference: PodReference{ClusterID: 2, Namespace: "ops", Name: "api"}, TailLines: &negative}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid session error=%v", err)
	}
	result, err := service.CreateExecSession(PodExecSessionInput{PodReference: PodReference{ClusterID: 2, Namespace: " ops ", Name: " api "}, UserID: 7, Command: []string{" sh ", " "}})
	if err != nil || result == nil || result.SessionID == "" {
		t.Fatalf("CreateExecSession() result=%#v err=%v", result, err)
	}
	session, ok := execs.Take(result.SessionID)
	if !ok || session.Namespace != "ops" || session.Pod != "api" || session.UserID != 7 || len(session.Command) != 1 || session.Command[0] != "sh" {
		t.Fatalf("session=%#v ok=%v", session, ok)
	}
}
