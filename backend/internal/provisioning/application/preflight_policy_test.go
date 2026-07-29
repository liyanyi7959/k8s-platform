package application

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestPlanPreflightChecksEvaluateTopologyAndNetwork(t *testing.T) {
	checks := PlanPreflightChecks("10.244.0.0/16", "10.96.0.0/12", []string{"master", "worker", "worker"})
	if len(checks) != 2 || checks[0].Status != PreflightPassed || checks[1].Status != PreflightPassed {
		t.Fatalf("valid plan checks = %#v", checks)
	}

	checks = PlanPreflightChecks("10.0.0.0/8", "10.96.0.0/12", []string{"master", "master"})
	if checks[0].Key != "plan.topology" || checks[0].Status != PreflightError || checks[1].Key != "plan.network" || checks[1].Status != PreflightError {
		t.Fatalf("unsafe plan checks = %#v", checks)
	}
	if !CIDRsOverlap("bad", "10.96.0.0/12") {
		t.Fatal("malformed persisted CIDR must block preflight")
	}
}

func TestPreflightAccumulatorAppliesOnlyAllowedIgnore(t *testing.T) {
	accumulator := NewPreflightAccumulator(time.Date(2026, 7, 29, 10, 30, 0, 0, time.FixedZone("CST", 8*60*60)), []string{"node.9.disk"})
	accumulator.Add(PreflightCheck{Key: "node.9.disk", Status: PreflightError, Message: "disk low", Ignorable: true})
	accumulator.Add(PreflightCheck{Key: "node.9.cpu", Status: PreflightError, Message: "cpu low", Ignorable: false})
	result := accumulator.Result()
	if result.Ready || result.CheckedAt != "2026-07-29T02:30:00Z" {
		t.Fatalf("aggregated result = %#v", result)
	}
	if result.Checks[0].Status != PreflightWarning || !result.Checks[0].Ignored || result.Checks[1].Status != PreflightError {
		t.Fatalf("ignore policy = %#v", result.Checks)
	}
	result.Checks[0].Message = "mutated"
	if accumulator.Result().Checks[0].Message == "mutated" {
		t.Fatal("result checks must not share their slice with the accumulator")
	}
}

func TestUpdatePreflightIgnoresValidatesNodeDiskKeysAndCopiesValues(t *testing.T) {
	existing := []string{"node.3.disk", "node.7.disk"}
	updated, err := UpdatePreflightIgnores(existing, []uint64{3, 7}, " node.7.disk ", false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := updated, []string{"node.3.disk"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("updated ignores = %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(existing, []string{"node.3.disk", "node.7.disk"}) {
		t.Fatalf("input mutation = %#v", existing)
	}
	if _, err := UpdatePreflightIgnores(existing, []uint64{3, 7}, "node.7.cpu", true); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("unsupported ignore error = %v", err)
	}
}

func TestNodeProbeChecksEvaluateSupportAndResources(t *testing.T) {
	checks := NodeProbeChecks(PreflightNodeProbe{ServerID: 12, ServerName: "node-a", OS: "Windows Server", CPUCores: 1, MemoryMB: 1024, DiskGB: 10})
	if got, want := preflightCheckStatuses(checks), []string{PreflightError, PreflightError, PreflightError, PreflightError}; !reflect.DeepEqual(got, want) {
		t.Fatalf("resource statuses = %#v, want %#v", got, want)
	}
	if !checks[3].Ignorable || checks[3].Key != "node.12.disk" || checks[3].ServerID == nil || *checks[3].ServerID != 12 {
		t.Fatalf("disk check = %#v", checks[3])
	}

	checks = NodeProbeChecks(PreflightNodeProbe{ServerID: 13, ServerName: "node-b", OS: "Rocky Linux", OSVersion: "9", CPUCores: 2, MemoryMB: 2048, DiskGB: 20})
	if got, want := preflightCheckStatuses(checks), []string{PreflightPassed, PreflightPassed, PreflightPassed, PreflightPassed}; !reflect.DeepEqual(got, want) {
		t.Fatalf("supported resource statuses = %#v, want %#v", got, want)
	}
}

func TestNodeRuntimeCheck(t *testing.T) {
	credentialFailure := NodeSSHFailureCheck(1, "node-a", "credential missing")
	if credentialFailure.Remediation != "检查服务器关联的 SSH 凭据" {
		t.Fatalf("credential remediation = %#v", credentialFailure)
	}
	connectionFailure := NodeSSHConnectionFailureCheck(1, "node-a", "ssh timeout")
	if connectionFailure.Remediation != "检查主机网络、SSH 服务、用户名和凭据" {
		t.Fatalf("connection remediation = %#v", connectionFailure)
	}
	failed := NodeRuntimeCheck(PreflightNodeRuntime{ServerID: 1, ConnectionMessage: "dial failed"})
	if failed.Status != PreflightError || failed.Message != "dial failed" {
		t.Fatalf("connection failure = %#v", failed)
	}
	passed := NodeRuntimeCheck(PreflightNodeRuntime{ServerID: 1, Python3Available: true, PrivilegeUsable: true})
	if passed.Status != PreflightPassed {
		t.Fatalf("runtime success = %#v", passed)
	}
}

func preflightCheckStatuses(checks []PreflightCheck) []string {
	result := make([]string, 0, len(checks))
	for _, check := range checks {
		result = append(result, check.Status)
	}
	return result
}
