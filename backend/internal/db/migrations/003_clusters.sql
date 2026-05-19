CREATE TABLE IF NOT EXISTS clusters (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  type VARCHAR(16) NOT NULL DEFAULT 'imported',
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  kubeconfig_enc LONGTEXT NULL,
  k8s_version VARCHAR(32) NOT NULL DEFAULT '',
  node_count INT NOT NULL DEFAULT 0,
  last_health_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_clusters_name (name),
  INDEX idx_clusters_status (status),
  INDEX idx_clusters_deleted (deleted_at),
  INDEX idx_clusters_last_health_at (last_health_at)
);
