-- deploy_configs 表增加 os_type 字段，支持按操作系统配置不同命令
-- 步骤1：添加 os_type 列（默认 ubuntu，兼容已有数据）
ALTER TABLE deploy_configs ADD COLUMN os_type VARCHAR(32) NOT NULL DEFAULT 'ubuntu' COMMENT '操作系统类型：ubuntu, centos, rocky, debian' AFTER step_key;

-- 步骤2：删除旧的唯一键，创建新的联合唯一键
ALTER TABLE deploy_configs DROP INDEX uk_deploy_configs_step_key;
ALTER TABLE deploy_configs ADD UNIQUE KEY uk_step_os (step_key, os_type);

-- 步骤3：将现有数据标记为 ubuntu（已有数据默认值已是 ubuntu，无需额外操作）

-- 步骤4：插入 CentOS/Rocky 版本的命令模板（仅 bootstrap 步骤差异较大）
INSERT INTO deploy_configs (step_key, step_name, step_order, os_type, command_template, description, timeout_seconds) VALUES
('bootstrap', '基础环境初始化', 2, 'centos',
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

# 关闭SELinux
sudo setenforce 0 || true
sudo sed -i ''s/^SELINUX=enforcing/SELINUX=disabled/'' /etc/selinux/config || true

# 关闭防火墙
sudo systemctl stop firewalld 2>/dev/null || true
sudo systemctl disable firewalld 2>/dev/null || true

# 安装containerd
sudo yum install -y yum-utils device-mapper-persistent-data lvm2
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo || true
sudo yum install -y containerd.io
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml > /dev/null
sudo sed -i ''s/SystemdCgroup = false/SystemdCgroup = true/'' /etc/containerd/config.toml
sudo systemctl restart containerd && sudo systemctl enable containerd

# 安装kubeadm/kubelet/kubectl
cat <<EOF | sudo tee /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://pkgs.k8s.io/core:/stable:/v{{.MinorVersion}}/rpm/
enabled=1
gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/v{{.MinorVersion}}/rpm/repodata/repomd.xml.key
EOF
sudo yum install -y kubelet kubeadm kubectl
sudo systemctl enable kubelet',
'CentOS/Rocky: 关闭swap/SELinux/防火墙、加载内核模块、安装containerd和kubeadm', 600);

-- CentOS 的其他步骤（pre_check, init_master, join_workers, install_cni, register）与 Ubuntu 命令兼容，无需重复插入
-- 仅 bootstrap 步骤因包管理器（apt vs yum）和 SELinux 差异需要单独配置
