package application

import (
	"fmt"
	"sigs.k8s.io/yaml"
	"strings"
)

type Workflow struct {
	Stages []WorkflowStage `json:"stages" yaml:"stages"`
}
type WorkflowStage struct {
	Key   string         `json:"key" yaml:"key"`
	Name  string         `json:"name" yaml:"name"`
	Steps []WorkflowStep `json:"steps" yaml:"steps"`
}
type WorkflowStep struct {
	Key               string            `json:"key" yaml:"key"`
	Name              string            `json:"name" yaml:"name"`
	Plugin            string            `json:"plugin" yaml:"plugin"`
	Run               string            `json:"run" yaml:"run"`
	Script            string            `json:"script" yaml:"script"`
	Repository        string            `json:"repository" yaml:"repository"`
	Branch            string            `json:"branch" yaml:"branch"`
	Path              string            `json:"path" yaml:"path"`
	Image             string            `json:"image" yaml:"image"`
	Tag               string            `json:"tag" yaml:"tag"`
	Context           string            `json:"context" yaml:"context"`
	Command           string            `json:"command" yaml:"command"`
	Args              []string          `json:"args" yaml:"args"`
	CredentialsSecret string            `json:"credentials_secret" yaml:"credentials_secret"`
	Env               map[string]string `json:"env" yaml:"env"`
}

func ParseWorkflow(raw string) (Workflow, error) {
	var workflow Workflow
	if strings.TrimSpace(raw) == "" {
		return workflow, nil
	}
	if err := yaml.Unmarshal([]byte(raw), &workflow); err != nil {
		return workflow, fmt.Errorf("invalid pipeline yaml: %w", err)
	}
	for si := range workflow.Stages {
		stage := &workflow.Stages[si]
		if strings.TrimSpace(stage.Name) == "" {
			stage.Name = fmt.Sprintf("stage-%d", si+1)
		}
		if strings.TrimSpace(stage.Key) == "" {
			stage.Key = slug(stage.Name, fmt.Sprintf("stage-%d", si+1))
		}
		for wi := range stage.Steps {
			step := &stage.Steps[wi]
			if strings.TrimSpace(step.Name) == "" {
				step.Name = fmt.Sprintf("step-%d", wi+1)
			}
			if strings.TrimSpace(step.Key) == "" {
				step.Key = slug(step.Name, fmt.Sprintf("step-%d", wi+1))
			}
			if strings.TrimSpace(step.Plugin) == "" {
				step.Plugin = "bash"
			}
			step.Plugin = strings.ToLower(strings.TrimSpace(step.Plugin))
			if step.Script == "" {
				step.Script = step.Run
			}
		}
	}
	return workflow, nil
}
func slug(value, fallback string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if b.Len() > 0 {
			b.WriteByte('-')
		}
	}
	value = strings.Trim(b.String(), "-")
	if value == "" {
		return fallback
	}
	return value
}
