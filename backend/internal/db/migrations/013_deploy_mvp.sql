CREATE TABLE IF NOT EXISTS deploy_servers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  ip VARCHAR(64) NOT NULL,
  ssh_port INT NOT NULL DEFAULT 22,
  user VARCHAR(64) NOT NULL DEFAULT 'root',
  auth_type VARCHAR(16) NOT NULL DEFAULT 'password',
  credential_enc TEXT NOT NULL,
  os VARCHAR(64) NULL,
  os_version VARCHAR(64) NULL,
  kernel VARCHAR(64) NULL,
  cpu_cores INT UNSIGNED NULL,
  memory_mb BIGINT UNSIGNED NULL,
  disk_gb BIGINT UNSIGNED NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'registered',
  labels JSON NULL,
  remark VARCHAR(512) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  INDEX idx_deploy_servers_status (status),
  INDEX idx_deploy_servers_deleted_at (deleted_at),
  INDEX idx_deploy_servers_ip_port (ip, ssh_port)
);

CREATE TABLE IF NOT EXISTS ssh_credentials (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  auth_type VARCHAR(16) NOT NULL DEFAULT 'password',
  username VARCHAR(64) NOT NULL DEFAULT 'root',
  credential_enc TEXT NOT NULL,
  remark VARCHAR(512) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  INDEX idx_ssh_credentials_deleted_at (deleted_at),
  INDEX idx_ssh_credentials_auth_type (auth_type)
);

CREATE TABLE IF NOT EXISTS deploy_plans (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  cluster_name VARCHAR(120) NOT NULL,
  k8s_version VARCHAR(32) NOT NULL,
  pod_cidr VARCHAR(64) NOT NULL DEFAULT '10.244.0.0/16',
  svc_cidr VARCHAR(64) NOT NULL DEFAULT '10.96.0.0/12',
  cni_type VARCHAR(32) NOT NULL DEFAULT 'flannel',
  cni_config JSON NULL,
  addons JSON NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  task_id BIGINT UNSIGNED NULL,
  cluster_id BIGINT UNSIGNED NULL,
  created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  INDEX idx_deploy_plans_status (status),
  INDEX idx_deploy_plans_task_id (task_id),
  INDEX idx_deploy_plans_cluster_id (cluster_id),
  INDEX idx_deploy_plans_deleted_at (deleted_at),
  INDEX idx_deploy_plans_cluster_name (cluster_name)
);

CREATE TABLE IF NOT EXISTS deploy_plan_nodes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  plan_id BIGINT UNSIGNED NOT NULL,
  server_id BIGINT UNSIGNED NOT NULL,
  role VARCHAR(16) NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_deploy_plan_nodes_plan_server (plan_id, server_id),
  INDEX idx_deploy_plan_nodes_plan_id (plan_id),
  INDEX idx_deploy_plan_nodes_server_id (server_id)
);

CREATE TABLE IF NOT EXISTS deploy_steps (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  plan_id BIGINT UNSIGNED NOT NULL,
  step_name VARCHAR(64) NOT NULL,
  step_order INT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  error_msg TEXT NULL,
  INDEX idx_deploy_steps_plan_order (plan_id, step_order)
);

CREATE TABLE IF NOT EXISTS deploy_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  plan_id BIGINT UNSIGNED NOT NULL,
  task_id BIGINT UNSIGNED NULL,
  step_name VARCHAR(64) NULL,
  server_ip VARCHAR(64) NULL,
  level VARCHAR(16) NOT NULL DEFAULT 'info',
  message TEXT NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_deploy_logs_plan_time (plan_id, created_at),
  INDEX idx_deploy_logs_task_id (task_id)
);
