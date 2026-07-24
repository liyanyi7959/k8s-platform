package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
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
