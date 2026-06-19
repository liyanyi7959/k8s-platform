-- 部署流程配置表：存储每个部署步骤的命令模板
CREATE TABLE IF NOT EXISTS deploy_configs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  step_key VARCHAR(64) NOT NULL COMMENT '步骤标识，如 pre_check, bootstrap, init_master 等',
  step_name VARCHAR(128) NOT NULL COMMENT '步骤显示名称',
  step_order INT NOT NULL DEFAULT 0 COMMENT '步骤排序',
  command_template TEXT NOT NULL COMMENT '命令模板，支持变量替换',
  description TEXT NULL COMMENT '步骤说明',
  enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  timeout_seconds INT NOT NULL DEFAULT 600 COMMENT '超时时间（秒）',
  retry_count INT NOT NULL DEFAULT 0 COMMENT '失败重试次数',
  created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_deploy_configs_step_key (step_key),
  INDEX idx_deploy_configs_step_order (step_order)
);

-- 部署配置版本历史表：记录每次配置修改
CREATE TABLE IF NOT EXISTS deploy_config_versions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  config_id BIGINT UNSIGNED NOT NULL COMMENT '关联的配置ID',
  step_key VARCHAR(64) NOT NULL,
  command_template TEXT NOT NULL COMMENT '修改前的命令模板',
  description TEXT NULL,
  change_type VARCHAR(16) NOT NULL DEFAULT 'update' COMMENT '变更类型：create, update, delete',
  changed_by BIGINT UNSIGNED NOT NULL COMMENT '操作人ID',
  changed_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  change_summary VARCHAR(512) NULL COMMENT '变更摘要',
  INDEX idx_deploy_config_versions_config_id (config_id),
  INDEX idx_deploy_config_versions_step_key (step_key),
  INDEX idx_deploy_config_versions_changed_at (changed_at)
);

-- 仓库配置表：存储镜像仓库和yum仓库配置
CREATE TABLE IF NOT EXISTS deploy_repositories (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  repo_type VARCHAR(32) NOT NULL COMMENT '仓库类型：yum, apt, container_mirror, registry',
  name VARCHAR(128) NOT NULL COMMENT '仓库名称',
  url VARCHAR(512) NOT NULL COMMENT '仓库地址',
  description TEXT NULL COMMENT '仓库说明',
  auth_type VARCHAR(16) NOT NULL DEFAULT 'none' COMMENT '认证类型：none, basic, token, key',
  auth_config JSON NULL COMMENT '认证配置（加密存储）',
  priority INT NOT NULL DEFAULT 100 COMMENT '优先级，数值越小优先级越高',
  enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  is_default TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否默认仓库',
  mirror_of VARCHAR(256) NULL COMMENT '镜像源对应的原始地址',
  created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  INDEX idx_deploy_repositories_type (repo_type),
  INDEX idx_deploy_repositories_priority (priority),
  INDEX idx_deploy_repositories_deleted_at (deleted_at)
);

-- 插入默认的部署流程配置
INSERT INTO deploy_configs (step_key, step_name, step_order, command_template, description, timeout_seconds) VALUES
('pre_check', '环境预检', 1, 
'# 环境预检命令
sudo -n true 2>/dev/null || echo "NO_SUDO"
swapon --show --noheadings | wc -l
ss -tlnp | grep ":6443 " | wc -l
ss -tlnp | grep ":2379 " | wc -l
ss -tlnp | grep ":2380 " | wc -l
ss -tlnp | grep ":10250 " | wc -l',
'检查SSH连接、sudo权限、swap状态和端口占用情况', 60),

('bootstrap', '基础环境初始化', 2,
'# 关闭swap
sudo swapoff -a && sudo sed -i ''/ swap / s/^/#/'' /etc/fstab

# 加载内核模块
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF
sudo modprobe overlay && sudo modprobe br_netfilter

# 配置sysctl
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sudo sysctl --system

# 安装containerd
sudo apt-get update -y && sudo apt-get install -y containerd
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml > /dev/null
sudo sed -i ''s/SystemdCgroup = false/SystemdCgroup = true/'' /etc/containerd/config.toml
sudo systemctl restart containerd && sudo systemctl enable containerd

# 安装kubeadm/kubelet/kubectl
sudo apt-get install -y apt-transport-https ca-certificates curl gpg
curl -fsSL https://pkgs.k8s.io/core:/stable:/v{{.MinorVersion}}/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo ''deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v{{.MinorVersion}}/deb/ /'' | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update -y && sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl',
'关闭swap、加载内核模块、配置sysctl参数、安装containerd和kubeadm', 600),

('init_master', '初始化Master节点', 3,
'# 初始化K8s Master
sudo kubeadm init \
  --pod-network-cidr={{.PodCIDR}} \
  --service-cidr={{.SvcCIDR}} \
  --kubernetes-version={{.K8sVersion}} \
  --apiserver-advertise-address={{.MasterIP}} \
  --upload-certs

# 配置kubectl
mkdir -p $HOME/.kube
sudo cp -f /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# 获取join命令
sudo kubeadm token create --print-join-command',
'执行kubeadm init初始化Master节点，配置kubectl，获取join token', 300),

('join_workers', 'Worker节点加入', 4,
'# Worker节点加入集群
{{.JoinCommand}}',
'使用join命令将Worker节点加入集群', 120),

('install_cni', '安装网络插件', 5,
'# 安装CNI网络插件
{{.CNICommand}}

# 等待CoreDNS就绪
kubectl wait --for=condition=Ready pods -l k8s-app=kube-dns -n kube-system --timeout=120s',
'安装CNI网络插件（flannel/calico/cilium）并等待CoreDNS就绪', 180),

('register', '注册集群到平台', 6,
'# 提取kubeconfig
sudo cat /etc/kubernetes/admin.conf',
'提取kubeconfig并注册集群到管理平台', 60);

-- 插入默认仓库配置
INSERT INTO deploy_repositories (repo_type, name, url, description, priority, is_default, mirror_of) VALUES
('container_mirror', '阿里云容器镜像加速', 'https://registry.aliyuncs.com/google_containers', '阿里云提供的K8s镜像加速服务', 10, 1, 'registry.k8s.io'),
('container_mirror', '中科大镜像源', 'https://docker.mirrors.ustc.edu.cn', '中国科学技术大学Docker镜像源', 20, 0, NULL),
('container_mirror', '腾讯云镜像加速', 'https://mirror.ccs.tencentyun.com', '腾讯云容器镜像加速', 30, 0, NULL),
('registry', 'Docker Hub', 'https://hub.docker.com', 'Docker官方镜像仓库', 100, 0, NULL),
('registry', '阿里云容器镜像服务', 'https://cr.console.aliyun.com', '阿里云容器镜像服务（ACR）', 50, 0, NULL),
('yum', '阿里云CentOS镜像', 'https://mirrors.aliyun.com/centos/', '阿里云CentOS yum镜像源', 10, 1, NULL),
('yum', '清华CentOS镜像', 'https://mirrors.tuna.tsinghua.edu.cn/centos/', '清华大学CentOS yum镜像源', 20, 0, NULL),
('yum', '华为云CentOS镜像', 'https://mirrors.huaweicloud.com/centos/', '华为云CentOS yum镜像源', 30, 0, NULL),
('apt', '阿里云Debian镜像', 'https://mirrors.aliyun.com/debian/', '阿里云Debian apt镜像源', 10, 1, NULL),
('apt', '清华Debian镜像', 'https://mirrors.tuna.tsinghua.edu.cn/debian/', '清华大学Debian apt镜像源', 20, 0, NULL),
('apt', 'K8s官方仓库', 'https://pkgs.k8s.io/core:/stable:', 'Kubernetes官方apt仓库', 10, 1, NULL);
