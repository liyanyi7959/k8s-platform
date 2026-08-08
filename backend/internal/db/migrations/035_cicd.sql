CREATE TABLE IF NOT EXISTS cicd_pipelines (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(512) NOT NULL DEFAULT '',
  trigger_type VARCHAR(32) NOT NULL DEFAULT 'manual',
  branches VARCHAR(512) NOT NULL DEFAULT '',
  cron VARCHAR(128) NOT NULL DEFAULT '',
  config_yaml LONGTEXT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'idle',
  last_run_id BIGINT UNSIGNED NULL,
  created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cicd_pipelines_name (name),
  INDEX idx_cicd_pipelines_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cicd_runs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  pipeline_id BIGINT UNSIGNED NOT NULL,
  trigger_type VARCHAR(32) NOT NULL DEFAULT 'manual',
  commit_sha VARCHAR(128) NOT NULL DEFAULT '',
  commit_message VARCHAR(512) NOT NULL DEFAULT '',
  branch VARCHAR(255) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'queued',
  error_message TEXT NULL,
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_cicd_runs_pipeline_created (pipeline_id, created_at),
  INDEX idx_cicd_runs_status_created (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cicd_run_stages (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  run_id BIGINT UNSIGNED NOT NULL,
  stage_key VARCHAR(128) NOT NULL,
  name VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'queued',
  log LONGTEXT NOT NULL,
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  sort_order INT NOT NULL DEFAULT 0,
  INDEX idx_cicd_run_stages_run (run_id, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cicd_artifacts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  run_id BIGINT UNSIGNED NULL,
  pipeline_id BIGINT UNSIGNED NULL,
  name VARCHAR(255) NOT NULL,
  artifact_type VARCHAR(32) NOT NULL DEFAULT 'image',
  version VARCHAR(128) NOT NULL DEFAULT '',
  repository VARCHAR(512) NOT NULL DEFAULT '',
  size_bytes BIGINT UNSIGNED NOT NULL DEFAULT 0,
  digest VARCHAR(255) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'available',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_cicd_artifacts_created (created_at),
  INDEX idx_cicd_artifacts_type (artifact_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cicd_environments (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  label VARCHAR(128) NOT NULL DEFAULT '',
  environment_type VARCHAR(32) NOT NULL DEFAULT 'development',
  cluster_id BIGINT UNSIGNED NULL,
  namespace VARCHAR(255) NOT NULL DEFAULT 'default',
  current_version VARCHAR(128) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'idle',
  last_run_id BIGINT UNSIGNED NULL,
  deployed_by VARCHAR(128) NOT NULL DEFAULT '',
  deploy_count INT NOT NULL DEFAULT 0,
  last_deployed_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_cicd_environments_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
