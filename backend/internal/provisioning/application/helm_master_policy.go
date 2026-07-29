package application

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// HelmMasterVersionToInstall pins the Helm binary installed on managed
// control-plane hosts. Keeping this policy in Provisioning makes the runtime
// adapter responsible only for finding the host and executing the script.
const HelmMasterVersionToInstall = "v3.16.4"

// HelmMasterPreflight is the auditable outcome of preparing a managed
// control-plane host for Helm operations.
type HelmMasterPreflight struct {
	ClusterID    uint64 `json:"cluster_id"`
	MasterName   string `json:"master_name"`
	MasterIP     string `json:"master_ip"`
	HelmVersion  string `json:"helm_version"`
	InstalledNow bool   `json:"installed_now"`
	Message      string `json:"message"`
}

// HelmMasterInstallRequest is the transport-neutral chart installation input
// used by the Provisioning runtime after Kops has validated its HTTP command.
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

// HelmRepository represents the repository inventory reported by Helm on the
// target cluster Master, rather than the platform process's local cache.
type HelmRepository struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// HelmChartSearchResult is the UI-oriented subset returned by `helm search
// repo -o json` on the target Master.
type HelmChartSearchResult struct {
	Name        string `json:"name"`
	ChartName   string `json:"chart_name"`
	RepoName    string `json:"repo_name"`
	RepoURL     string `json:"repo_url"`
	Version     string `json:"version"`
	AppVersion  string `json:"app_version"`
	Description string `json:"description"`
}

// HelmMasterEnsureScript installs the pinned upstream archive only when the
// exact Helm version is not already present. It verifies the published
// checksum before replacing the binary.
func HelmMasterEnsureScript() string {
	return `set -eu
HELM_VERSION="` + HelmMasterVersionToInstall + `"
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

// HelmMasterInstallScript builds a shell-safe, atomic installation command.
// Values are transferred through base64 so user YAML is never interpreted as
// shell syntax.
func HelmMasterInstallScript(req HelmMasterInstallRequest) string {
	valuesCommand := `values_arg=""`
	if strings.TrimSpace(req.ValuesYAML) != "" {
		encodedValues := base64.StdEncoding.EncodeToString([]byte(req.ValuesYAML))
		valuesCommand = `
command -v base64 >/dev/null 2>&1 || { echo "缺少 base64，无法安全传递 values.yaml" >&2; exit 1; }
values_file="$workdir/values.yaml"
printf %s ` + shellQuote(encodedValues) + ` | base64 -d > "$values_file"
values_arg="--values $values_file"`
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

func HelmMasterAddRepositoryScript(name, repoURL string) string {
	return `set -eu
helm repo add ` + shellQuote(name) + ` ` + shellQuote(repoURL) + ` --force-update >/dev/null
helm repo update ` + shellQuote(name) + ` >/dev/null`
}

func HelmMasterDeleteRepositoryScript(name string) string {
	return `set -eu
if ! command -v helm >/dev/null 2>&1; then
  exit 0
fi
helm repo remove ` + shellQuote(name) + ` >/dev/null 2>&1 || true`
}

func HelmMasterListRepositoriesScript() string {
	return `set -eu
if ! command -v helm >/dev/null 2>&1; then
  printf '[]'
  exit 0
fi
if ! helm repo list -o json 2>/dev/null; then
  # Helm 在首次运行、尚未创建 repositories.yaml 时会返回非零；这表示空仓库，
  # 不是集群状态读取失败。
  printf '[]'
fi`
}

func HelmMasterSearchChartsScript() string {
	return `set -eu
if ! command -v helm >/dev/null 2>&1; then
  printf '[]'
  exit 0
fi
if ! helm search repo -o json 2>/dev/null; then
  # 未同步任何仓库时没有可搜索的 Chart，按空结果返回而不是将 Helm 的
  # 初始化细节暴露给界面。
  printf '[]'
fi`
}

// ParseHelmMasterPreflight accepts only the version emitted by the pinned
// bootstrap script. This prevents a partial or unrelated command output from
// being reported as a successful Helm preflight.
func ParseHelmMasterPreflight(clusterID uint64, masterName, masterIP, output string) (HelmMasterPreflight, error) {
	result := HelmMasterPreflight{
		ClusterID:  clusterID,
		MasterName: strings.TrimSpace(masterName),
		MasterIP:   strings.TrimSpace(masterIP),
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
	if result.HelmVersion != HelmMasterVersionToInstall {
		return HelmMasterPreflight{}, ErrWithMessage(ErrConflict, "Master Helm 校验未返回受支持的固定版本，已阻止部署")
	}
	if result.InstalledNow {
		result.Message = "Master Helm 已完成安装或升级，并确认计划版本可用"
	} else {
		result.Message = "Master 已安装 Helm，版本校验通过"
	}
	return result, nil
}

func ParseHelmMasterRepositories(output string) ([]HelmRepository, error) {
	repositories := make([]HelmRepository, 0)
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &repositories); err != nil {
		return nil, ErrWithMessage(ErrConflict, "Master 返回的 Helm 仓库数据无效："+err.Error())
	}
	return repositories, nil
}

// ParseHelmMasterChartSearch filters the decoded Helm index without passing
// the user's free-text keyword to a shell expression.
func ParseHelmMasterChartSearch(output, keyword string) ([]HelmChartSearchResult, error) {
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
	items := make([]HelmChartSearchResult, 0, len(raw))
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

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
