CREATE TABLE IF NOT EXISTS cicd_run_steps (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  run_id BIGINT UNSIGNED NOT NULL,
  stage_key VARCHAR(128) NOT NULL,
  step_key VARCHAR(128) NOT NULL,
  name VARCHAR(128) NOT NULL,
  plugin VARCHAR(64) NOT NULL DEFAULT 'bash',
  status VARCHAR(32) NOT NULL DEFAULT 'queued',
  log LONGTEXT NOT NULL,
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  sort_order INT NOT NULL DEFAULT 0,
  UNIQUE KEY uk_cicd_run_steps_key (run_id, step_key),
  INDEX idx_cicd_run_steps_run (run_id, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
