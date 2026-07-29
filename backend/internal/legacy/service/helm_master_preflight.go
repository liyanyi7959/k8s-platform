package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	model "k8s-platform-backend/internal/provisioning/domain"
)

const helmVersionToInstall = "v3.16.4"

// HelmMasterPreflight is the auditable result of preparing a managed cluster's
// control-plane host for Helm operations. The actual deployment still uses the
// Helm SDK, but keeping Helm on the master makes emergency investigation and
// recovery possible from the cluster itself.
type HelmMasterPreflight struct {
	ClusterID    uint64 `json:"cluster_id"`
	MasterName   string `json:"master_name"`
	MasterIP     string `json:"master_ip"`
	HelmVersion  string `json:"helm_version"`
	InstalledNow bool   `json:"installed_now"`
	Message      string `json:"message"`
}

// HelmMasterInstallRequest is the constrained input needed to run a chart
// installation on a managed cluster's control-plane host. Keeping this type
// in the service layer prevents the API controller from handling SSH details.
type HelmMasterInstallRequest struct {
	ReleaseName string
	Namespace   string
	Chart       string
	Version     string
	RepoName    string
	RepoURL     string
	ValuesYAML  string
}

type HelmMasterInstallResult struct {
	Output string `json:"output"`
}

// HelmRepository is the repository inventory reported by the Helm binary on
// the cluster Master.  It deliberately does not use the platform host's local
// Helm cache: the Master is where chart installation is executed.
type HelmRepository struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// HelmChartSearchResult is the small, UI-oriented subset returned by
// `helm search repo -o json` on the cluster Master.
type HelmChartSearchResult struct {
	Name        string `json:"name"`
	ChartName   string `json:"chart_name"`
	RepoName    string `json:"repo_name"`
	RepoURL     string `json:"repo_url"`
	Version     string `json:"version"`
	AppVersion  string `json:"app_version"`
	Description string `json:"description"`
}

// EnsureClusterMasterHelm verifies the control-plane mapping, its managed SSH
// asset and Helm binary. If Helm is absent, it downloads the pinned upstream
// archive over HTTPS, verifies its published SHA-256 checksum and installs it
// with elevated privileges before reporting success.
func (s *DeployService) EnsureClusterMasterHelm(ctx context.Context, clusterID uint64) (HelmMasterPreflight, error) {
	if s == nil || s.db == nil {
		return HelmMasterPreflight{}, fmt.Errorf("Helm 前置校验不可用：部署服务未初始化")
	}
	if clusterID == 0 {
		return HelmMasterPreflight{}, ErrInvalidParams
	}

	var plan model.DeployPlan
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND cluster_id = ?", clusterID).
		Order("id DESC").
		First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return HelmMasterPreflight{}, ErrWithMessage(ErrNotFound, "当前集群未关联平台部署计划，无法安全定位 Master 并校验 Helm；请先纳管 Master SSH 资产后再部署")
		}
		return HelmMasterPreflight{}, err
	}

	var masterNode model.DeployPlanNode
	if err := s.db.WithContext(ctx).
		Where("plan_id = ? AND role = ?", plan.ID, "master").
		Order("sort_order ASC, id ASC").
		First(&masterNode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return HelmMasterPreflight{}, ErrWithMessage(ErrNotFound, "部署计划未配置 Master 节点，无法执行 Helm 前置校验")
		}
		return HelmMasterPreflight{}, err
	}

	master, credential, authType, err := s.getServerCredentialForNode(ctx, masterNode.ServerID)
	if err != nil {
		return HelmMasterPreflight{}, ErrWithMessage(ErrNotFound, "无法读取 Master SSH 凭据："+err.Error())
	}
	if master.Status != "available" {
		return HelmMasterPreflight{}, ErrWithMessage(ErrConflict, "Master 主机当前不可用，请先在主机资源池完成 SSH 巡检后再部署")
	}
	master.AuthType = authType

	client, err := dialDeploySSH(ctx, master, credential)
	if err != nil {
		return HelmMasterPreflight{}, ErrWithMessage(ErrConflict, "Master SSH 连接失败，已阻止 Helm 部署："+err.Error())
	}
	defer client.Close()

	output, err := runPrivilegedSSHCommand(client, master, credential, helmMasterEnsureScript())
	if err != nil {
		return HelmMasterPreflight{}, ErrWithMessage(ErrConflict, "Master Helm 安装或校验失败："+err.Error())
	}
	result := HelmMasterPreflight{
		ClusterID:  clusterID,
		MasterName: master.Name,
		MasterIP:   master.IP,
	}
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "HELM_VERSION":
			result.HelmVersion = strings.TrimSpace(value)
		case "HELM_INSTALLED_NOW":
			result.InstalledNow = strings.TrimSpace(value) == "true"
		}
	}
	if result.HelmVersion == "" {
		return HelmMasterPreflight{}, ErrWithMessage(ErrConflict, "Master Helm 校验未返回有效版本，已阻止部署")
	}
	if result.InstalledNow {
		result.Message = "Master Helm 已完成安装或升级，并确认计划版本可用"
	} else {
		result.Message = "Master 已安装 Helm，版本校验通过"
	}
	return result, nil
}

// InstallClusterMasterHelm runs the chart operation where Helm was prepared:
// the cluster Master. This avoids coupling application installation to the
// platform server's Internet egress, while the platform continues to use its
// kubeconfig for release inspection and lifecycle views.
func (s *DeployService) InstallClusterMasterHelm(ctx context.Context, clusterID uint64, req HelmMasterInstallRequest) (HelmMasterInstallResult, error) {
	if s == nil || s.db == nil {
		return HelmMasterInstallResult{}, fmt.Errorf("Helm 部署服务未初始化")
	}
	if clusterID == 0 {
		return HelmMasterInstallResult{}, ErrInvalidParams
	}

	output, err := s.runClusterMasterHelmScript(ctx, clusterID, helmMasterInstallScript(req), true)
	if err != nil {
		if strings.Contains(output, "AIOPS_REPOSITORY_UNAVAILABLE") {
			return HelmMasterInstallResult{}, ErrWithMessage(ErrConflict, "Master 无法访问 Helm 仓库。系统已在 Master 上重试 3 次；请检查该节点的 DNS、HTTPS 出口或改用可访问的内部仓库")
		}
		return HelmMasterInstallResult{}, ErrWithMessage(ErrConflict, "Master 执行 Helm 安装失败："+err.Error())
	}
	return HelmMasterInstallResult{Output: strings.TrimSpace(output)}, nil
}

// ListClusterMasterHelmRepos returns the repository state that Helm on the
// selected cluster can actually use.  Browsing this page never changes the
// host: if Helm has not been prepared yet, the returned inventory is simply
// empty.  The first installation still prepares Helm automatically.
func (s *DeployService) ListClusterMasterHelmRepos(ctx context.Context, clusterID uint64) ([]HelmRepository, error) {
	output, err := s.runClusterMasterHelmScript(ctx, clusterID, `set -eu
if ! command -v helm >/dev/null 2>&1; then
  printf '[]'
  exit 0
fi
if ! helm repo list -o json 2>/dev/null; then
  # Helm 在首次运行、尚未创建 repositories.yaml 时会返回非零；这表示空仓库，
  # 不是集群状态读取失败。
  printf '[]'
fi`, false)
	if err != nil {
		return nil, err
	}
	repositories := make([]HelmRepository, 0)
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &repositories); err != nil {
		return nil, ErrWithMessage(ErrConflict, "Master 返回的 Helm 仓库数据无效："+err.Error())
	}
	return repositories, nil
}

// AddClusterMasterHelmRepo adds or updates a repository on the same Master
// that performs application installation.  Helm is prepared here on demand.
func (s *DeployService) AddClusterMasterHelmRepo(ctx context.Context, clusterID uint64, name, repoURL string) error {
	_, err := s.runClusterMasterHelmScript(ctx, clusterID, helmMasterAddRepositoryScript(name, repoURL), true)
	return err
}

// DeleteClusterMasterHelmRepo removes one repository from the Master.  This
// only changes Helm's repository registry; it never uninstalls any Release.
func (s *DeployService) DeleteClusterMasterHelmRepo(ctx context.Context, clusterID uint64, name string) error {
	_, err := s.runClusterMasterHelmScript(ctx, clusterID, helmMasterDeleteRepositoryScript(name), false)
	return err
}

// SearchClusterMasterHelmCharts searches the indexes that are already
// synchronized on the Master, then narrows the result in Go so arbitrary user
// text is never passed to a shell expression.
func (s *DeployService) SearchClusterMasterHelmCharts(ctx context.Context, clusterID uint64, keyword string) ([]HelmChartSearchResult, error) {
	output, err := s.runClusterMasterHelmScript(ctx, clusterID, `set -eu
if ! command -v helm >/dev/null 2>&1; then
  printf '[]'
  exit 0
fi
if ! helm search repo -o json 2>/dev/null; then
  # 未同步任何仓库时没有可搜索的 Chart，按空结果返回而不是把 Helm 的初始化细节暴露给界面。
  printf '[]'
fi`, false)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		AppVersion  string `json:"app_version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &raw); err != nil {
		return nil, ErrWithMessage(ErrConflict, "Master 返回的 Helm Chart 数据无效："+err.Error())
	}
	needle := strings.ToLower(strings.TrimSpace(keyword))
	items := make([]HelmChartSearchResult, 0)
	for _, item := range raw {
		if needle != "" && !strings.Contains(strings.ToLower(item.Name+" "+item.Description), needle) {
			continue
		}
		repoName, chartName, ok := strings.Cut(item.Name, "/")
		if !ok || repoName == "" || chartName == "" {
			continue
		}
		items = append(items, HelmChartSearchResult{
			Name:        item.Name,
			ChartName:   chartName,
			RepoName:    repoName,
			Version:     item.Version,
			AppVersion:  item.AppVersion,
			Description: item.Description,
		})
	}
	return items, nil
}

// runClusterMasterHelmScript executes a constrained Helm script on the
// managed control-plane host.  Keeping this in the service layer ensures the
// API never receives SSH credentials and all Helm state belongs to the target
// cluster rather than the platform process.
func (s *DeployService) runClusterMasterHelmScript(ctx context.Context, clusterID uint64, script string, ensureHelm bool) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("Helm 部署服务未初始化")
	}
	if clusterID == 0 {
		return "", ErrInvalidParams
	}
	if ensureHelm {
		if _, err := s.EnsureClusterMasterHelm(ctx, clusterID); err != nil {
			return "", err
		}
	}

	var plan model.DeployPlan
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL AND cluster_id = ?", clusterID).
		Order("id DESC").
		First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrWithMessage(ErrNotFound, "当前集群未关联平台部署计划，无法安全定位 Master")
		}
		return "", err
	}
	var masterNode model.DeployPlanNode
	if err := s.db.WithContext(ctx).
		Where("plan_id = ? AND role = ?", plan.ID, "master").
		Order("sort_order ASC, id ASC").
		First(&masterNode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrWithMessage(ErrNotFound, "部署计划未配置 Master 节点，无法执行 Helm 操作")
		}
		return "", err
	}
	master, credential, authType, err := s.getServerCredentialForNode(ctx, masterNode.ServerID)
	if err != nil {
		return "", ErrWithMessage(ErrNotFound, "无法读取 Master SSH 凭据："+err.Error())
	}
	if master.Status != "available" {
		return "", ErrWithMessage(ErrConflict, "Master 主机当前不可用，请先完成 SSH 巡检后再操作 Helm")
	}
	master.AuthType = authType
	client, err := dialDeploySSH(ctx, master, credential)
	if err != nil {
		return "", ErrWithMessage(ErrConflict, "Master SSH 连接失败，已阻止 Helm 操作："+err.Error())
	}
	defer client.Close()

	output, err := runPrivilegedSSHCommand(client, master, credential, script)
	if err != nil {
		return output, ErrWithMessage(ErrConflict, "Master 执行 Helm 命令失败："+err.Error())
	}
	return strings.TrimSpace(output), nil
}

func helmMasterInstallScript(req HelmMasterInstallRequest) string {
	valuesCommand := ""
	if strings.TrimSpace(req.ValuesYAML) != "" {
		encodedValues := base64.StdEncoding.EncodeToString([]byte(req.ValuesYAML))
		valuesCommand = `
command -v base64 >/dev/null 2>&1 || { echo "缺少 base64，无法安全传递 values.yaml" >&2; exit 1; }
values_file="$workdir/values.yaml"
printf %s ` + shellQuote(encodedValues) + ` | base64 -d > "$values_file"
values_arg="--values $values_file"`
	} else {
		valuesCommand = `values_arg=""`
	}
	versionArg := ""
	if strings.TrimSpace(req.Version) != "" {
		versionArg = " --version " + shellQuote(strings.TrimSpace(req.Version))
	}
	repositoryCommand := ""
	if req.RepoName != "" && req.RepoURL != "" {
		repositoryCommand = `
repo_synced=false
for attempt in 1 2 3; do
  if helm repo add ` + shellQuote(req.RepoName) + ` ` + shellQuote(req.RepoURL) + ` --force-update >/dev/null 2>&1 && helm repo update ` + shellQuote(req.RepoName) + ` >/dev/null 2>&1; then
    repo_synced=true
    break
  fi
  [ "$attempt" = "3" ] || sleep $((attempt * 2))
done
if [ "$repo_synced" != "true" ]; then
  echo "AIOPS_REPOSITORY_UNAVAILABLE"
  exit 21
fi`
	}

	return `set -eu
export KUBECONFIG=/etc/kubernetes/admin.conf
command -v helm >/dev/null 2>&1 || { echo "未找到 helm 命令" >&2; exit 1; }
test -r "$KUBECONFIG" || { echo "无法读取 Kubernetes 管理 kubeconfig" >&2; exit 1; }
workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT
` + valuesCommand + repositoryCommand + `
helm upgrade --install ` + shellQuote(req.ReleaseName) + ` ` + shellQuote(req.Chart) + ` --namespace ` + shellQuote(req.Namespace) + ` --create-namespace --atomic --wait --timeout 10m` + versionArg + ` $values_arg
echo "HELM_RELEASE=` + req.Namespace + `/` + req.ReleaseName + `"
echo "HELM_OPERATION=upgrade-install"`
}

func helmMasterAddRepositoryScript(name, repoURL string) string {
	return `set -eu
helm repo add ` + shellQuote(name) + ` ` + shellQuote(repoURL) + ` --force-update >/dev/null
helm repo update ` + shellQuote(name) + ` >/dev/null`
}

func helmMasterDeleteRepositoryScript(name string) string {
	return `set -eu
if ! command -v helm >/dev/null 2>&1; then
  exit 0
fi
helm repo remove ` + shellQuote(name) + ` >/dev/null 2>&1 || true`
}

func helmMasterEnsureScript() string {
	return `set -eu
HELM_VERSION="` + helmVersionToInstall + `"
if command -v helm >/dev/null 2>&1 && [ "$(helm version --template '{{ .Version }}' 2>/dev/null || true)" = "$HELM_VERSION" ]; then
  echo "HELM_VERSION=$HELM_VERSION"
  echo "HELM_INSTALLED_NOW=false"
  exit 0
fi

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "不支持的 CPU 架构：$arch" >&2; exit 1 ;;
esac
command -v tar >/dev/null 2>&1 || { echo "缺少 tar，无法安装 Helm" >&2; exit 1; }
command -v sha256sum >/dev/null 2>&1 || { echo "缺少 sha256sum，无法安全校验 Helm 安装包" >&2; exit 1; }
if command -v curl >/dev/null 2>&1; then downloader="curl -fsSLo";
elif command -v wget >/dev/null 2>&1; then downloader="wget -qO";
else echo "缺少 curl 或 wget，无法下载 Helm" >&2; exit 1; fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
archive="helm-${HELM_VERSION}-linux-${arch}.tar.gz"
url="https://get.helm.sh/${archive}"
$downloader "$tmp/$archive" "$url"
$downloader "$tmp/${archive}.sha256sum" "${url}.sha256sum"
expected_sha="$(awk '{print $1}' "$tmp/${archive}.sha256sum")"
[ "${#expected_sha}" -eq 64 ] || { echo "Helm 安装包 SHA-256 格式无效" >&2; exit 1; }
printf '%s  %s\n' "$expected_sha" "$tmp/$archive" | sha256sum -c -
tar -xzf "$tmp/$archive" -C "$tmp"
install -m 0755 "$tmp/linux-${arch}/helm" /usr/local/bin/helm
[ "$(helm version --template '{{ .Version }}')" = "$HELM_VERSION" ]
echo "HELM_VERSION=$HELM_VERSION"
echo "HELM_INSTALLED_NOW=true"`
}
