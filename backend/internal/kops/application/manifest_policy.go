package application

import "strings"

var ErrInvalidManifest = ErrInvalidParams

// ManifestApplyInput is transport-neutral validation for a multi-document
// manifest application. Kubernetes execution remains an adapter concern.
type ManifestApplyInput struct {
	ClusterID        uint64
	YAML             string
	DefaultNamespace string
	DryRun           bool
	CreateOnly       bool
	SourceLabel      string
	SourceResource   string
	WorkloadKind     string
	CreatedBy        uint64
	CreatedByName    string
}

func NormalizeManifestApply(input ManifestApplyInput) (ManifestApplyInput, error) {
	input.YAML = strings.TrimSpace(input.YAML)
	input.DefaultNamespace = strings.TrimSpace(input.DefaultNamespace)
	input.SourceLabel = strings.TrimSpace(input.SourceLabel)
	input.SourceResource = strings.TrimSpace(input.SourceResource)
	input.WorkloadKind = strings.TrimSpace(input.WorkloadKind)
	input.CreatedByName = strings.TrimSpace(input.CreatedByName)
	if input.ClusterID == 0 || input.YAML == "" {
		return ManifestApplyInput{}, ErrInvalidManifest
	}
	if input.SourceLabel == "" {
		input.SourceLabel = "通用 YAML 清单"
	}
	return input, nil
}

func NormalizeManifestPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
