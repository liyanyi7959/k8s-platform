package application

import "testing"

func TestAnsiblePolicy(t *testing.T) {
	steps := ComputeEnabledAnsibleSteps("install_helm")
	if len(steps) != 3 || steps[0] != "install_helm" || steps[2] != "register" {
		t.Fatalf("enabled steps=%#v", steps)
	}
	if !HasAnsibleRecapFailure("node-a : ok=1 changed=0 unreachable=0 failed=2") {
		t.Fatal("failed recap must be detected")
	}
	if HasAnsibleRecapFailure("node-a : ok=1 changed=0 unreachable=0 failed=0") {
		t.Fatal("successful recap must not be detected")
	}
	if got := ExtractAnsiblePlayName("PLAY [环境预检 - all]"); got != "环境预检 - all" {
		t.Fatalf("play name=%q", got)
	}
}
