-- 巡检模板
CREATE TABLE IF NOT EXISTS inspection_templates (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name        VARCHAR(120) NOT NULL,
  description VARCHAR(512) DEFAULT NULL,
  category    VARCHAR(32) NOT NULL DEFAULT 'baseline',
  version     VARCHAR(32) NOT NULL DEFAULT 'v1.0',
  tags        JSON DEFAULT NULL,
  recommended TINYINT(1) NOT NULL DEFAULT 0,
  checks      LONGTEXT DEFAULT NULL,
  created_by  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  updated_by  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at  DATETIME(3) DEFAULT NULL,
  INDEX idx_inspection_templates_deleted (deleted_at),
  INDEX idx_inspection_templates_category (category)
);

-- 定时巡检计划
CREATE TABLE IF NOT EXISTS inspection_schedules (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name            VARCHAR(160) NOT NULL,
  template_id     BIGINT UNSIGNED NOT NULL,
  cron            VARCHAR(64) NOT NULL,
  status          VARCHAR(16) NOT NULL DEFAULT 'enabled',
  server_ids      JSON DEFAULT NULL,
  target_count    INT NOT NULL DEFAULT 0,
  last_run_status VARCHAR(16) DEFAULT NULL,
  last_run_at     DATETIME(3) DEFAULT NULL,
  next_run_at     DATETIME(3) DEFAULT NULL,
  created_by      BIGINT UNSIGNED NOT NULL DEFAULT 0,
  updated_by      BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at      DATETIME(3) DEFAULT NULL,
  INDEX idx_inspection_schedules_template (template_id),
  INDEX idx_inspection_schedules_status (status),
  INDEX idx_inspection_schedules_deleted (deleted_at)
);

-- 巡检报告
CREATE TABLE IF NOT EXISTS inspection_reports (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  report_no      VARCHAR(64) NOT NULL,
  template_id    BIGINT UNSIGNED NOT NULL,
  template_name  VARCHAR(120) NOT NULL DEFAULT '',
  scope_label    VARCHAR(255) NOT NULL DEFAULT '',
  target_count   INT NOT NULL DEFAULT 0,
  health_score   INT NOT NULL DEFAULT 100,
  abnormal_count INT NOT NULL DEFAULT 0,
  high_risk_count INT NOT NULL DEFAULT 0,
  risk_level     VARCHAR(8) NOT NULL DEFAULT 'p3',
  status         VARCHAR(16) NOT NULL DEFAULT 'success',
  top_issues     JSON DEFAULT NULL,
  task_id        BIGINT UNSIGNED DEFAULT NULL,
  hosts_json     LONGTEXT DEFAULT NULL,
  issues_json    LONGTEXT DEFAULT NULL,
  generated_at   DATETIME(3) DEFAULT NULL,
  created_by     VARCHAR(64) NOT NULL DEFAULT '',
  created_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  INDEX idx_inspection_reports_template (template_id),
  INDEX idx_inspection_reports_risk (risk_level),
  INDEX idx_inspection_reports_created (created_at)
);
