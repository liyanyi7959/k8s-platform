package application

import "testing"

func TestParseWorkflowDefaultsLegacyRunToBash(t *testing.T) {
	w, err := ParseWorkflow("stages:\n  - name: Build\n    steps:\n      - name: compile\n        run: go test ./...\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Stages) != 1 || w.Stages[0].Key != "build" || w.Stages[0].Steps[0].Plugin != "bash" || w.Stages[0].Steps[0].Script != "go test ./..." {
		t.Fatalf("workflow=%#v", w)
	}
}

func TestParseWorkflowPreservesPlugins(t *testing.T) {
	w, err := ParseWorkflow("stages:\n  - key: source\n    name: Source\n    steps:\n      - key: checkout\n        plugin: git\n        repository: https://example.com/app.git\n        branch: main\n")
	if err != nil {
		t.Fatal(err)
	}
	step := w.Stages[0].Steps[0]
	if step.Plugin != "git" || step.Key != "checkout" || step.Branch != "main" {
		t.Fatalf("step=%#v", step)
	}
}
