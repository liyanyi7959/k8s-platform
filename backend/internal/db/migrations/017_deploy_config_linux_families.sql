-- 统一校准部署步骤模板，补齐 Ubuntu 系列与红帽系列各自完整的 6 步命令。
-- 说明：
-- 1. 现有库不会重跑旧 migration，因此用新 migration 修正线上/本地已落库数据。
-- 2. 红帽系列统一以 os_type=centos 存储，rocky/rhel/almalinux 通过服务层回退到该模板。

-- Ubuntu / Debian 系列：环境预检
UPDATE deploy_configs
SET step_name = '环境预检',
    step_order = 1,
    command_template = '# 环境预检
set -eu
sudo -n true
test -f /etc/os-release && . /etc/os-release && echo "OS=${PRETTY_NAME:-${NAME:-unknown}}" && echo "OS_VERSION=${VERSION_ID:-unknown}" || { echo "OS=unknown"; echo "OS_VERSION=unknown"; }
echo "KERNEL=$(uname -r)"
echo "HOSTNAME=$(hostname)"
echo "CPU_CORES=$(nproc)"
echo "MEMORY_MB=$(free -m | awk ''/Mem:/{print $2}'')"
echo "DISK_GB=$(df -BG / | awk ''NR==2{gsub(/G/, "", $4); print $4}'')"
echo "PRODUCT_UUID=$(sudo cat /sys/class/dmi/id/product_uuid 2>/dev/null || echo unknown)"
echo "DEFAULT_IP=$(ip route get 1.1.1.1 2>/dev/null | awk ''{for(i=1;i<=NF;i++) if ($i=="src") {print $(i+1); exit}}'' || hostname -I | awk ''{print $1}'')"
echo "SWAP_LINES=$(swapon --show --noheadings | wc -l)"
echo "PORT_6443=$(ss -lnt sport = :6443 | tail -n +2 | wc -l)"
echo "PORT_2379=$(ss -lnt sport = :2379 | tail -n +2 | wc -l)"
echo "PORT_2380=$(ss -lnt sport = :2380 | tail -n +2 | wc -l)"
echo "PORT_10250=$(ss -lnt sport = :10250 | tail -n +2 | wc -l)"
echo "BR_NETFILTER=$(sysctl -n net.bridge.bridge-nf-call-iptables 2>/dev/null || echo 0)"
echo "IP_FORWARD=$(sysctl -n net.ipv4.ip_forward 2>/dev/null || echo 0)"
echo "NTP_SYNC=$(timedatectl show --property=NTPSynchronized --value 2>/dev/null || echo unknown)"
echo "CONTAINERD=$(command -v containerd >/dev/null 2>&1 && echo installed || echo missing)"
echo "KUBEADM=$(command -v kubeadm >/dev/null 2>&1 && kubeadm version -o short || echo missing)"',
    description = '检查 sudo、主机标识、默认路由、swap、关键端口、内核转发、NTP 与 containerd/kubeadm 安装情况',
    timeout_seconds = 120,
    retry_count = 0,
    enabled = 1
WHERE step_key = 'pre_check' AND os_type = 'ubuntu';

-- Ubuntu / Debian 系列：基础环境初始化
UPDATE deploy_configs
SET step_name = '基础环境初始化',
    step_order = 2,
    command_template = '# 基础环境初始化（Ubuntu / Debian）
set -eux

sudo swapoff -a
sudo sed -ri ''/\sswap\s/s/^#?/#/'' /etc/fstab

cat <<''EOF'' | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF
sudo modprobe overlay
sudo modprobe br_netfilter

cat <<''EOF'' | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward = 1
EOF
sudo sysctl --system

sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl gpg
sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/v{{.MinorVersion}}/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo ''deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v{{.MinorVersion}}/deb/ /'' | sudo tee /etc/apt/sources.list.d/kubernetes.list > /dev/null

sudo apt-get install -y containerd
sudo mkdir -p /etc/containerd
sudo containerd config default | sudo tee /etc/containerd/config.toml > /dev/null
sudo sed -i ''s/SystemdCgroup = false/SystemdCgroup = true/'' /etc/containerd/config.toml
sudo systemctl restart containerd
sudo systemctl enable containerd

sudo apt-get update
sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
sudo systemctl enable --now kubelet',
    description = 'Ubuntu 系列节点关闭 swap、启用内核模块与转发、安装并配置 containerd、接入 pkgs.k8s.io apt 仓库并安装 kubelet/kubeadm/kubectl',
    timeout_seconds = 900,
    retry_count = 0,
    enabled = 1
WHERE step_key = 'bootstrap' AND os_type = 'ubuntu';

-- Ubuntu / Debian 系列：初始化控制平面
UPDATE deploy_configs
SET step_name = '初始化 Master 节点',
    step_order = 3,
    command_template = '# 初始化控制平面
set -eux
sudo kubeadm init \
  --pod-network-cidr={{.PodCIDR}} \
  --service-cidr={{.SvcCIDR}} \
  --kubernetes-version={{.K8sVersion}} \
  --apiserver-advertise-address={{.MasterIP}} \
  --upload-certs

mkdir -p "$HOME/.kube"
sudo cp -f /etc/kubernetes/admin.conf "$HOME/.kube/config"
sudo chown "$(id -u):$(id -g)" "$HOME/.kube/config"

sudo kubeadm token create --print-join-command',
    description = '在控制平面节点执行 kubeadm init，生成管理员 kubeconfig，并输出 worker 加入命令',
    timeout_seconds = 900,
    retry_count = 0,
    enabled = 1
WHERE step_key = 'init_master' AND os_type = 'ubuntu';

-- Ubuntu / Debian 系列：Worker 加入
UPDATE deploy_configs
SET step_name = 'Worker 节点加入',
    step_order = 4,
    command_template = '# Worker 节点加入集群
set -eux
sudo {{.JoinCommand}}',
    description = '在每台 Worker 节点执行 kubeadm join 命令加入集群',
    timeout_seconds = 300,
    retry_count = 0,
    enabled = 1
WHERE step_key = 'join_workers' AND os_type = 'ubuntu';

-- Ubuntu / Debian 系列：安装网络插件
UPDATE deploy_configs
SET step_name = '安装网络插件',
    step_order = 5,
    command_template = '# 安装 CNI 网络插件
set -eux
export KUBECONFIG=/etc/kubernetes/admin.conf
{{.CNICommand}}
kubectl wait --for=condition=Ready pods -n kube-system -l k8s-app=kube-dns --timeout=180s || \
kubectl wait --for=condition=Ready pods -n kube-system -l k8s-app=coredns --timeout=180s',
    description = '在控制平面节点安装选定的 CNI 插件，并等待 CoreDNS 就绪',
    timeout_seconds = 300,
    retry_count = 0,
    enabled = 1
WHERE step_key = 'install_cni' AND os_type = 'ubuntu';

-- Ubuntu / Debian 系列：注册集群
UPDATE deploy_configs
SET step_name = '注册集群到平台',
    step_order = 6,
    command_template = '# 提取 kubeconfig 供平台注册
set -eu
sudo cat /etc/kubernetes/admin.conf',
    description = '读取控制平面节点上的 admin.conf，供平台导入并注册集群',
    timeout_seconds = 120,
    retry_count = 0,
    enabled = 1
WHERE step_key = 'register' AND os_type = 'ubuntu';

-- 红帽系列：更新已存在的 bootstrap 模板
UPDATE deploy_configs
SET step_name = '基础环境初始化',
    step_order = 2,
    command_template = '# 基础环境初始化（红帽系列）
set -eux

PKG_MGR="$(command -v dnf || command -v yum)"
sudo swapoff -a
sudo sed -ri ''/\sswap\s/s/^#?/#/'' /etc/fstab

cat <<''EOF'' | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF
sudo modprobe overlay
sudo modprobe br_netfilter

cat <<''EOF'' | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward = 1
EOF
sudo sysctl --system

sudo setenforce 0 || true
sudo sed -ri ''s/^SELINUX=enforcing$/SELINUX=permissive/'' /etc/selinux/config || true

sudo systemctl disable --now firewalld 2>/dev/null || true

sudo "$PKG_MGR" install -y containerd containernetworking-plugins
sudo mkdir -p /etc/containerd
sudo containerd config default | sudo tee /etc/containerd/config.toml > /dev/null
sudo sed -i ''s/SystemdCgroup = false/SystemdCgroup = true/'' /etc/containerd/config.toml
sudo systemctl restart containerd
sudo systemctl enable containerd

cat <<EOF | sudo tee /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://pkgs.k8s.io/core:/stable:/v{{.MinorVersion}}/rpm/
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/v{{.MinorVersion}}/rpm/repodata/repomd.xml.key
exclude=kubelet kubeadm kubectl cri-tools kubernetes-cni
EOF

sudo "$PKG_MGR" install -y kubelet kubeadm kubectl --disableexcludes=kubernetes
sudo systemctl enable --now kubelet',
    description = '红帽系列节点关闭 swap、调整 SELinux 与 firewalld、安装并配置 containerd、接入 pkgs.k8s.io rpm 仓库并安装 kubelet/kubeadm/kubectl',
    timeout_seconds = 900,
    retry_count = 0,
    enabled = 1
WHERE step_key = 'bootstrap' AND os_type = 'centos';

-- 红帽系列：补齐缺失的 5 个步骤
INSERT INTO deploy_configs (step_key, step_name, step_order, os_type, command_template, description, enabled, timeout_seconds, retry_count, created_by)
SELECT 'pre_check', '环境预检', 1, 'centos',
'# 环境预检
set -eu
sudo -n true
test -f /etc/os-release && . /etc/os-release && echo "OS=${PRETTY_NAME:-${NAME:-unknown}}" && echo "OS_VERSION=${VERSION_ID:-unknown}" || { echo "OS=unknown"; echo "OS_VERSION=unknown"; }
echo "KERNEL=$(uname -r)"
echo "HOSTNAME=$(hostname)"
echo "CPU_CORES=$(nproc)"
echo "MEMORY_MB=$(free -m | awk ''/Mem:/{print $2}'')"
echo "DISK_GB=$(df -BG / | awk ''NR==2{gsub(/G/, "", $4); print $4}'')"
echo "PRODUCT_UUID=$(sudo cat /sys/class/dmi/id/product_uuid 2>/dev/null || echo unknown)"
echo "DEFAULT_IP=$(ip route get 1.1.1.1 2>/dev/null | awk ''{for(i=1;i<=NF;i++) if ($i=="src") {print $(i+1); exit}}'' || hostname -I | awk ''{print $1}'')"
echo "SWAP_LINES=$(swapon --show --noheadings | wc -l)"
echo "PORT_6443=$(ss -lnt sport = :6443 | tail -n +2 | wc -l)"
echo "PORT_2379=$(ss -lnt sport = :2379 | tail -n +2 | wc -l)"
echo "PORT_2380=$(ss -lnt sport = :2380 | tail -n +2 | wc -l)"
echo "PORT_10250=$(ss -lnt sport = :10250 | tail -n +2 | wc -l)"
echo "SELINUX=$(getenforce 2>/dev/null || echo unknown)"
echo "FIREWALLD=$(systemctl is-active firewalld 2>/dev/null || echo inactive)"
echo "BR_NETFILTER=$(sysctl -n net.bridge.bridge-nf-call-iptables 2>/dev/null || echo 0)"
echo "IP_FORWARD=$(sysctl -n net.ipv4.ip_forward 2>/dev/null || echo 0)"
echo "NTP_SYNC=$(timedatectl show --property=NTPSynchronized --value 2>/dev/null || echo unknown)"
echo "CONTAINERD=$(command -v containerd >/dev/null 2>&1 && echo installed || echo missing)"
echo "KUBEADM=$(command -v kubeadm >/dev/null 2>&1 && kubeadm version -o short || echo missing)"',
'检查 sudo、主机标识、默认路由、swap、关键端口、SELinux、firewalld、内核转发、NTP 与 containerd/kubeadm 安装情况', 1, 120, 0, 0
WHERE NOT EXISTS (
  SELECT 1 FROM deploy_configs WHERE step_key = 'pre_check' AND os_type = 'centos'
);

INSERT INTO deploy_configs (step_key, step_name, step_order, os_type, command_template, description, enabled, timeout_seconds, retry_count, created_by)
SELECT 'init_master', '初始化 Master 节点', 3, 'centos',
'# 初始化控制平面
set -eux
sudo kubeadm init \
  --pod-network-cidr={{.PodCIDR}} \
  --service-cidr={{.SvcCIDR}} \
  --kubernetes-version={{.K8sVersion}} \
  --apiserver-advertise-address={{.MasterIP}} \
  --upload-certs

mkdir -p "$HOME/.kube"
sudo cp -f /etc/kubernetes/admin.conf "$HOME/.kube/config"
sudo chown "$(id -u):$(id -g)" "$HOME/.kube/config"

sudo kubeadm token create --print-join-command',
'在控制平面节点执行 kubeadm init，生成管理员 kubeconfig，并输出 worker 加入命令', 1, 900, 0, 0
WHERE NOT EXISTS (
  SELECT 1 FROM deploy_configs WHERE step_key = 'init_master' AND os_type = 'centos'
);

INSERT INTO deploy_configs (step_key, step_name, step_order, os_type, command_template, description, enabled, timeout_seconds, retry_count, created_by)
SELECT 'join_workers', 'Worker 节点加入', 4, 'centos',
'# Worker 节点加入集群
set -eux
sudo {{.JoinCommand}}',
'在每台 Worker 节点执行 kubeadm join 命令加入集群', 1, 300, 0, 0
WHERE NOT EXISTS (
  SELECT 1 FROM deploy_configs WHERE step_key = 'join_workers' AND os_type = 'centos'
);

INSERT INTO deploy_configs (step_key, step_name, step_order, os_type, command_template, description, enabled, timeout_seconds, retry_count, created_by)
SELECT 'install_cni', '安装网络插件', 5, 'centos',
'# 安装 CNI 网络插件
set -eux
export KUBECONFIG=/etc/kubernetes/admin.conf
{{.CNICommand}}
kubectl wait --for=condition=Ready pods -n kube-system -l k8s-app=kube-dns --timeout=180s || \
kubectl wait --for=condition=Ready pods -n kube-system -l k8s-app=coredns --timeout=180s',
'在控制平面节点安装选定的 CNI 插件，并等待 CoreDNS 就绪', 1, 300, 0, 0
WHERE NOT EXISTS (
  SELECT 1 FROM deploy_configs WHERE step_key = 'install_cni' AND os_type = 'centos'
);

INSERT INTO deploy_configs (step_key, step_name, step_order, os_type, command_template, description, enabled, timeout_seconds, retry_count, created_by)
SELECT 'register', '注册集群到平台', 6, 'centos',
'# 提取 kubeconfig 供平台注册
set -eu
sudo cat /etc/kubernetes/admin.conf',
'读取控制平面节点上的 admin.conf，供平台导入并注册集群', 1, 120, 0, 0
WHERE NOT EXISTS (
  SELECT 1 FROM deploy_configs WHERE step_key = 'register' AND os_type = 'centos'
);