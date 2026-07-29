package application

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var helmName = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

type HelmInstallInput struct {
	ClusterID   uint64 `json:"-"`
	ReleaseName string `json:"release_name"`
	Namespace   string `json:"namespace"`
	Chart       string `json:"chart"`
	Version     string `json:"version"`
	RepoURL     string `json:"repo_url"`
	RepoName    string `json:"repo_name"`
	ValuesYAML  string `json:"values_yaml"`
}
type HelmRepositoryInput struct {
	ClusterID uint64 `json:"-"`
	Name      string `json:"name"`
	URL       string `json:"url"`
}
type HelmUpgradeInput struct {
	ClusterID  uint64 `json:"-"`
	Namespace  string `json:"-"`
	Name       string `json:"-"`
	Chart      string `json:"chart"`
	Version    string `json:"version"`
	ValuesYAML string `json:"values_yaml"`
	Atomic     bool   `json:"atomic"`
	Wait       bool   `json:"wait"`
	Timeout    string `json:"timeout"`
}
type HelmRollbackInput struct {
	ClusterID uint64 `json:"-"`
	Namespace string `json:"-"`
	Name      string `json:"-"`
	Revision  int    `json:"revision"`
	Wait      bool   `json:"wait"`
}
type HelmPreflightResult struct {
	ClusterVersion  string `json:"cluster_version"`
	Master          any    `json:"master"`
	ExecutionTarget string `json:"execution_target"`
}
type HelmInstallResult struct {
	Output          string `json:"output"`
	CommandOutput   string `json:"command_output"`
	ClusterVersion  string `json:"cluster_version"`
	Master          any    `json:"master"`
	ExecutionTarget string `json:"execution_target"`
}
type HelmRuntime interface {
	Preflight(context.Context, uint64) (string, any, error)
	Install(context.Context, HelmInstallInput) (string, error)
	ListRepositories(context.Context, uint64) (any, error)
	AddRepository(context.Context, HelmRepositoryInput) error
	DeleteRepository(context.Context, uint64, string) error
	Search(context.Context, uint64, string) (any, error)
	ListReleases(context.Context, uint64, string) (any, error)
	ReleaseDetail(context.Context, uint64, string, string) (any, error)
	Uninstall(context.Context, uint64, string, string) (string, error)
	Upgrade(context.Context, HelmUpgradeInput) (string, error)
	Rollback(context.Context, HelmRollbackInput) (string, error)
}
type HelmService struct{ runtime HelmRuntime }

func NewHelmService(runtime HelmRuntime) *HelmService { return &HelmService{runtime: runtime} }
func (s *HelmService) Preflight(ctx context.Context, clusterID uint64) (*HelmPreflightResult, error) {
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	version, master, err := s.runtime.Preflight(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return &HelmPreflightResult{ClusterVersion: version, Master: master, ExecutionTarget: "master"}, nil
}
func (s *HelmService) Install(ctx context.Context, input HelmInstallInput) (*HelmInstallResult, error) {
	if err := normalizeHelmInstall(&input); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Minute)
	defer cancel()
	version, master, err := s.runtime.Preflight(ctx, input.ClusterID)
	if err != nil {
		return nil, err
	}
	output, err := s.runtime.Install(ctx, input)
	if err != nil {
		return nil, err
	}
	return &HelmInstallResult{Output: "Release " + input.Namespace + "/" + input.ReleaseName + " 已由 Master Helm 安装并通过就绪校验", CommandOutput: output, ClusterVersion: version, Master: master, ExecutionTarget: "master"}, nil
}
func (s *HelmService) ListRepositories(ctx context.Context, clusterID uint64) (any, error) {
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.ListRepositories(ctx, clusterID)
}
func (s *HelmService) AddRepository(ctx context.Context, input HelmRepositoryInput) error {
	if err := normalizeHelmRepository(&input); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.AddRepository(ctx, input)
}
func (s *HelmService) DeleteRepository(ctx context.Context, clusterID uint64, name string) (string, error) {
	name = strings.TrimSpace(name)
	if clusterID == 0 || !validHelmName(name) {
		return "", ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return "", ErrConflict
	}
	return name, s.runtime.DeleteRepository(ctx, clusterID, name)
}
func (s *HelmService) Search(ctx context.Context, clusterID uint64, keyword string) (any, error) {
	keyword = strings.TrimSpace(keyword)
	if clusterID == 0 || keyword == "" {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.Search(ctx, clusterID, keyword)
}
func (s *HelmService) ListReleases(ctx context.Context, clusterID uint64, namespace string) (any, error) {
	if clusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.ListReleases(ctx, clusterID, strings.TrimSpace(namespace))
}
func (s *HelmService) ReleaseDetail(ctx context.Context, clusterID uint64, namespace, name string) (any, error) {
	namespace, name = strings.TrimSpace(namespace), strings.TrimSpace(name)
	if clusterID == 0 || name == "" {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.ReleaseDetail(ctx, clusterID, namespace, name)
}
func (s *HelmService) Uninstall(ctx context.Context, clusterID uint64, namespace, name string) (string, error) {
	namespace, name = strings.TrimSpace(namespace), strings.TrimSpace(name)
	if clusterID == 0 || namespace == "" || name == "" {
		return "", ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return "", ErrConflict
	}
	return s.runtime.Uninstall(ctx, clusterID, namespace, name)
}
func (s *HelmService) Upgrade(ctx context.Context, input HelmUpgradeInput) (string, error) {
	input.Namespace, input.Name, input.Chart = strings.TrimSpace(input.Namespace), strings.TrimSpace(input.Name), strings.TrimSpace(input.Chart)
	if input.ClusterID == 0 || input.Namespace == "" || input.Name == "" || input.Chart == "" {
		return "", ErrInvalidParams
	}
	if strings.TrimSpace(input.ValuesYAML) != "" {
		var values map[string]any
		if yaml.Unmarshal([]byte(input.ValuesYAML), &values) != nil {
			return "", ErrInvalidParams
		}
	}
	if s == nil || s.runtime == nil {
		return "", ErrConflict
	}
	return s.runtime.Upgrade(ctx, input)
}
func (s *HelmService) Rollback(ctx context.Context, input HelmRollbackInput) (string, error) {
	input.Namespace, input.Name = strings.TrimSpace(input.Namespace), strings.TrimSpace(input.Name)
	if input.ClusterID == 0 || input.Namespace == "" || input.Name == "" || input.Revision <= 0 {
		return "", ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return "", ErrConflict
	}
	return s.runtime.Rollback(ctx, input)
}
func normalizeHelmInstall(input *HelmInstallInput) error {
	if input == nil {
		return ErrInvalidParams
	}
	input.ReleaseName, input.Namespace, input.Chart, input.RepoURL, input.RepoName = strings.TrimSpace(input.ReleaseName), strings.TrimSpace(input.Namespace), strings.TrimSpace(input.Chart), strings.TrimSpace(input.RepoURL), strings.TrimSpace(input.RepoName)
	if input.Namespace == "" {
		input.Namespace = "default"
	}
	if input.ClusterID == 0 || !validHelmName(input.ReleaseName) || len(input.ReleaseName) > 53 || !validHelmName(input.Namespace) || len(input.Namespace) > 63 || input.Chart == "" || strings.Contains(input.Chart, "..") || strings.Contains(input.Chart, "://") || strings.HasPrefix(input.Chart, "/") || len(input.ValuesYAML) > 1024*1024 {
		return ErrInvalidParams
	}
	if (input.RepoURL == "") != (input.RepoName == "") {
		return ErrInvalidParams
	}
	if input.RepoURL != "" {
		if !validHelmURL(input.RepoURL) || !validHelmName(input.RepoName) || len(input.RepoName) > 63 || !strings.HasPrefix(input.Chart, input.RepoName+"/") {
			return ErrInvalidParams
		}
	}
	if strings.TrimSpace(input.ValuesYAML) != "" {
		var values map[string]any
		if yaml.Unmarshal([]byte(input.ValuesYAML), &values) != nil {
			return ErrInvalidParams
		}
	}
	return nil
}
func normalizeHelmRepository(input *HelmRepositoryInput) error {
	if input == nil {
		return ErrInvalidParams
	}
	input.Name, input.URL = strings.TrimSpace(input.Name), strings.TrimSpace(input.URL)
	if input.ClusterID == 0 || !validHelmName(input.Name) || len(input.Name) > 63 || !validHelmURL(input.URL) {
		return ErrInvalidParams
	}
	return nil
}
func validHelmName(value string) bool { return value != "" && helmName.MatchString(value) }
func validHelmURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}
