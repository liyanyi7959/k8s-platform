CREATE TABLE IF NOT EXISTS monitor_alert_rules (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(160) NOT NULL,
  cluster_id BIGINT UNSIGNED NOT NULL,
  severity VARCHAR(16) NOT NULL DEFAULT 'warning',
  condition_text TEXT,
  duration VARCHAR(32) NOT NULL DEFAULT '5m',
  receivers_json JSON,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  INDEX idx_monitor_alert_rules_cluster (cluster_id),
  INDEX idx_monitor_alert_rules_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS monitor_incidents (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  fingerprint VARCHAR(160) NOT NULL,
  rule_id BIGINT UNSIGNED NULL,
  alert_name VARCHAR(255) NOT NULL,
  cluster_id BIGINT UNSIGNED NOT NULL,
  namespace VARCHAR(255) NOT NULL DEFAULT '',
  resource_kind VARCHAR(64) NOT NULL DEFAULT '',
  resource_name VARCHAR(255) NOT NULL DEFAULT '',
  severity VARCHAR(16) NOT NULL DEFAULT 'warning',
  status VARCHAR(32) NOT NULL DEFAULT 'open',
  summary VARCHAR(512) NOT NULL DEFAULT '',
  description TEXT,
  labels_json JSON,
  annotations_json JSON,
  ai_conversation_id BIGINT UNSIGNED NULL,
  ai_proposal_id BIGINT UNSIGNED NULL,
  assignee_id BIGINT UNSIGNED NULL,
  assignee_name VARCHAR(80) NOT NULL DEFAULT '',
  started_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  acknowledged_at DATETIME(3) NULL,
  resolved_at DATETIME(3) NULL,
  verified_at DATETIME(3) NULL,
  verification_note TEXT,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_monitor_incidents_fingerprint (fingerprint),
  INDEX idx_monitor_incidents_cluster_status (cluster_id, status),
  INDEX idx_monitor_incidents_started (started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS monitor_incident_timelines (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  incident_id BIGINT UNSIGNED NOT NULL,
  type VARCHAR(32) NOT NULL,
  title VARCHAR(255) NOT NULL,
  detail TEXT,
  meta_json JSON,
  operator_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  operator VARCHAR(80) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_monitor_incident_timeline (incident_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
