package service

import (
	"testing"
	"time"

	"k8s-platform-backend/internal/model"
)

// ────────── normalizeAIActionType ──────────

func TestNormalizeAIActionType(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"scale_workload", aiActionTypeScaleWorkload},
		{"restart_workload", aiActionTypeRestartWorkload},
		{"delete_workload", aiActionTypeDeleteWorkload},
		{"update_workload_image", aiActionTypeUpdateWorkloadImage},
		{"apply_manifest", aiActionTypeApplyManifest},
		{"delete_resource", aiActionTypeDeleteResource},
		{"delete_pod", aiActionTypeDeletePod},
		{"cordon_node", aiActionTypeCordonNode},
		{"uncordon_node", aiActionTypeUncordonNode},
		{"drain_node", aiActionTypeDrainNode},
		{"trigger_cronjob", aiActionTypeTriggerCronJob},
		{"suspend_cronjob", aiActionTypeSuspendCronJob},
		{"delete_completed_jobs", aiActionTypeDeleteCompletedJobs},
		{"pause_workload_rollout", aiActionTypePauseWorkloadRollout},
		{"rollout_undo", aiActionTypeRolloutUndo},
		{"  SCALE_WORKLOAD  ", aiActionTypeScaleWorkload},
		{"Restart_Workload", aiActionTypeRestartWorkload},
		{"unknown_type", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeAIActionType(tt.input)
			if got != tt.expect {
				t.Fatalf("normalizeAIActionType(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

// ────────── normalizeAIActionTarget ──────────

func TestNormalizeAIActionTarget(t *testing.T) {
	tests := []struct {
		name       string
		inputKind  string
		expectKind string
	}{
		{"deployment", "deployment", "Deployment"},
		{"DEPLOYMENT", "DEPLOYMENT", "Deployment"},
		{"statefulset", "statefulset", "StatefulSet"},
		{"daemonset", "daemonset", "DaemonSet"},
		{"pod", "pod", "Pod"},
		{"node", "node", "Node"},
		{"job", "job", "Job"},
		{"cronjob", "cronjob", "CronJob"},
		{"namespace", "namespace", "Namespace"},
		{"service", "service", "Service"},
		{"ingress", "ingress", "Ingress"},
		{"configmap", "configmap", "ConfigMap"},
		{"secret", "secret", "Secret"},
		{"pvc", "pvc", "PersistentVolumeClaim"},
		{"pv", "pv", "PersistentVolume"},
		{"storageclass", "storageclass", "StorageClass"},
		{"replicaset", "replicaset", "ReplicaSet"},
		{"unknown", "unknown", "unknown"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := AIActionTargetResource{Kind: tt.inputKind, Namespace: " ns ", Name: " name "}
			got := normalizeAIActionTarget(input)
			if got.Kind != tt.expectKind {
				t.Fatalf("kind = %q, want %q", got.Kind, tt.expectKind)
			}
			if got.Namespace != "ns" {
				t.Fatalf("namespace = %q, want ns (trimmed)", got.Namespace)
			}
			if got.Name != "name" {
				t.Fatalf("name = %q, want name (trimmed)", got.Name)
			}
		})
	}
}

// ────────── aiActionWorkloadGVR ──────────

func TestAIActionWorkloadGVR(t *testing.T) {
	tests := []struct {
		kind      string
		wantGroup string
		wantRes   string
		wantOK    bool
	}{
		{"Deployment", "apps", "deployments", true},
		{"StatefulSet", "apps", "statefulsets", true},
		{"DaemonSet", "apps", "daemonsets", true},
		{"Pod", "", "", false},
		{"Job", "", "", false},
		{"Node", "", "", false},
		{"Unknown", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			gvr, ok := aiActionWorkloadGVR(tt.kind)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.wantOK {
				if gvr.Group != tt.wantGroup {
					t.Fatalf("group = %q, want %q", gvr.Group, tt.wantGroup)
				}
				if gvr.Resource != tt.wantRes {
					t.Fatalf("resource = %q, want %q", gvr.Resource, tt.wantRes)
				}
			}
		})
	}
}

// ────────── aiActionPayload ──────────

func TestAIActionPayload_WithPayloadKey(t *testing.T) {
	change := model.JSONMap{
		"payload": map[string]any{
			"replicas": 5,
			"image":    "nginx:1.25",
		},
	}
	got := aiActionPayload(change)
	if got["replicas"] != 5 {
		t.Fatalf("replicas = %v", got["replicas"])
	}
	if got["image"] != "nginx:1.25" {
		t.Fatalf("image = %v", got["image"])
	}
}

func TestAIActionPayload_NilChange(t *testing.T) {
	got := aiActionPayload(nil)
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestAIActionPayload_NoPayloadKey(t *testing.T) {
	change := model.JSONMap{"other": "value"}
	got := aiActionPayload(change)
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

// ────────── aiActionExecutionStatusLabel ──────────

func TestAIActionExecutionStatusLabel(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"succeeded", "成功"},
		{"SUCCEEDED", "成功"},
		{"  succeeded  ", "成功"},
		{"failed", "失败"},
		{"FAILED", "失败"},
		{"executing", "完成"},
		{"pending", "完成"},
		{"unknown", "完成"},
		{"", "完成"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := aiActionExecutionStatusLabel(tt.input)
			if got != tt.expect {
				t.Fatalf("label(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

// ────────── jsonIntValue ──────────

func TestJSONIntValue(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		expect int
		ok     bool
	}{
		{"int", 42, 42, true},
		{"int8", int8(3), 3, true},
		{"int16", int16(100), 100, true},
		{"int32", int32(1000), 1000, true},
		{"int64", int64(9999), 9999, true},
		{"uint", uint(7), 7, true},
		{"float32", float32(1.5), 1, true},
		{"float64", float64(3.14), 3, true},
		{"string", "hello", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := jsonIntValue(tt.input)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.expect {
				t.Fatalf("got = %d, want %d", got, tt.expect)
			}
		})
	}
}

// ────────── jsonNestedInt ──────────

func TestJSONNestedInt(t *testing.T) {
	m := map[string]any{
		"config": map[string]any{
			"timeout": 30,
			"nested": map[string]any{
				"deep": 42,
			},
		},
		"simple": 10,
	}
	if jsonNestedInt(m, "config", "timeout") != 30 {
		t.Fatal("nested int failed")
	}
	if jsonNestedInt(m, "simple") != 10 {
		t.Fatal("single key failed")
	}
	if jsonNestedInt(m, "missing", "key") != 0 {
		t.Fatal("missing path should return 0")
	}
	if jsonNestedInt(m, "config", "missing") != 0 {
		t.Fatal("missing nested key should return 0")
	}
}

// ────────── buildAIActionConfirmationText ──────────

func TestBuildAIActionConfirmationText(t *testing.T) {
	tests := []struct {
		proposalID uint64
		expect     string
	}{
		{1, "confirm-proposal-1"},
		{42, "confirm-proposal-42"},
		{0, "confirm-proposal-0"},
	}
	for _, tt := range tests {
		got := buildAIActionConfirmationText(tt.proposalID)
		if got != tt.expect {
			t.Fatalf("buildAIActionConfirmationText(%d) = %q, want %q", tt.proposalID, got, tt.expect)
		}
	}
}

// ────────── aiActionString ──────────

func TestAIActionString(t *testing.T) {
	tests := []struct {
		input  any
		expect string
	}{
		{"hello", "hello"},
		{"  spaced  ", "spaced"},
		{[]byte("bytes"), "bytes"},
		{[]byte("  bytes  "), "bytes"},
		{42, "42"},
		{nil, ""},
	}
	for _, tt := range tests {
		got := aiActionString(tt.input)
		if got != tt.expect {
			t.Fatalf("aiActionString(%v) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

// ────────── buildAIActionProposalItem ──────────

func TestBuildAIActionProposalItem(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	proposal := model.AIActionProposal{
		ID:               10,
		ConversationID:   100,
		ClusterID:        1,
		ActionType:       "scale_workload",
		TargetKind:       "Deployment",
		TargetName:       "nginx",
		TargetNamespace:  "default",
		RiskLevel:        "medium",
		ConfirmLevel:     "single",
		Status:           "pending",
		Title:            "Scale to 3",
		Summary:          "scaling up",
		ChangeJSON:       model.JSONMap{"replicas": 3},
		CreatedBy:        1,
		CreatedByName:    "admin",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	item := buildAIActionProposalItem(proposal, nil)
	if item.ID != 10 {
		t.Fatalf("id = %d, want 10", item.ID)
	}
	if item.ActionType != "scale_workload" {
		t.Fatalf("action type = %q", item.ActionType)
	}
	if item.TargetKind != "Deployment" {
		t.Fatalf("target kind = %q", item.TargetKind)
	}
	if item.RiskLevel != "medium" {
		t.Fatalf("risk level = %q", item.RiskLevel)
	}
	if item.RequiredConfirmationText != "confirm-proposal-10" {
		t.Fatalf("confirmation text = %q", item.RequiredConfirmationText)
	}
	if item.LatestExecution != nil {
		t.Fatal("latest_execution should be nil when no executions")
	}
}

func TestBuildAIActionProposalItem_WithExecutions(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	proposal := model.AIActionProposal{
		ID:        20,
		Status:    "executed",
		CreatedAt: now,
		UpdatedAt: now,
	}
	exes := []model.AIActionExecution{
		{ID: 1, ProposalID: 20, Status: "succeeded", ExecutionNo: 1, CreatedAt: now},
	}
	item := buildAIActionProposalItem(proposal, exes)
	if item.LatestExecution == nil {
		t.Fatal("latest_execution should not be nil")
	}
	if item.LatestExecution.Status != "succeeded" {
		t.Fatalf("latest status = %q", item.LatestExecution.Status)
	}
	if len(item.Executions) != 1 {
		t.Fatalf("executions len = %d, want 1", len(item.Executions))
	}
}

// ────────── buildAIActionExecutionItem ──────────

func TestBuildAIActionExecutionItem(t *testing.T) {
	now := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	exe := model.AIActionExecution{
		ID:              5,
		ProposalID:      10,
		Status:          "succeeded",
		ExecutionNo:     1,
		OperatorID:      2,
		OperatorName:    "ops",
		CommandSnapshot: `{"action":"restart"}`,
		ResultJSON:      model.JSONMap{"pods_restarted": 3},
		StartedAt:       &now,
		FinishedAt:      &now,
		CreatedAt:       now,
	}
	item := buildAIActionExecutionItem(exe)
	if item.ID != 5 {
		t.Fatalf("id = %d", item.ID)
	}
	if item.ProposalID != 10 {
		t.Fatalf("proposal_id = %d", item.ProposalID)
	}
	if item.Status != "succeeded" {
		t.Fatalf("status = %q", item.Status)
	}
	if item.OperatorName != "ops" {
		t.Fatalf("operator = %q", item.OperatorName)
	}
	if item.StartedAt == nil {
		t.Fatal("started_at should not be nil")
	}
	if item.FinishedAt == nil {
		t.Fatal("finished_at should not be nil")
	}
}

// ────────── validateAIActionConfirmation ──────────

func TestValidateAIActionConfirmation_MissingConfirmRisk(t *testing.T) {
	proposal := model.AIActionProposal{ID: 1}
	req := ConfirmAIActionProposalRequest{ConfirmRisk: false}
	err := validateAIActionConfirmation(proposal, req)
	if err == nil {
		t.Fatal("expected error for missing confirm_risk")
	}
}

func TestValidateAIActionConfirmation_MissingText(t *testing.T) {
	proposal := model.AIActionProposal{ID: 1}
	req := ConfirmAIActionProposalRequest{ConfirmRisk: true, ConfirmationText: ""}
	err := validateAIActionConfirmation(proposal, req)
	if err == nil {
		t.Fatal("expected error for empty confirmation text")
	}
}

func TestValidateAIActionConfirmation_WrongText(t *testing.T) {
	proposal := model.AIActionProposal{ID: 42}
	req := ConfirmAIActionProposalRequest{ConfirmRisk: true, ConfirmationText: "wrong-text"}
	err := validateAIActionConfirmation(proposal, req)
	if err == nil {
		t.Fatal("expected error for wrong confirmation text")
	}
}

func TestValidateAIActionConfirmation_CorrectText(t *testing.T) {
	proposal := model.AIActionProposal{ID: 42}
	req := ConfirmAIActionProposalRequest{
		ConfirmRisk:      true,
		ConfirmationText: "confirm-proposal-42",
	}
	err := validateAIActionConfirmation(proposal, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// ────────── ptrAIActionExecutionItem ──────────

func TestPtrAIActionExecutionItem(t *testing.T) {
	item := AIActionExecutionItem{ID: 99, Status: "ok"}
	ptr := ptrAIActionExecutionItem(item)
	if ptr == nil {
		t.Fatal("should not be nil")
	}
	if ptr.ID != 99 {
		t.Fatalf("id = %d", ptr.ID)
	}
}
