CREATE TABLE IF NOT EXISTS role_namespace_scopes (
  role_id BIGINT UNSIGNED NOT NULL,
  cluster_id BIGINT UNSIGNED NOT NULL,
  namespace VARCHAR(120) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (role_id, cluster_id, namespace),
  CONSTRAINT fk_role_namespace_scopes_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
  CONSTRAINT fk_role_namespace_scopes_cluster FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE,
  INDEX idx_role_namespace_scopes_cluster (cluster_id)
);