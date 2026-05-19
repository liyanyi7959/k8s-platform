CREATE TABLE IF NOT EXISTS playbook_templates (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  scenario VARCHAR(32) NOT NULL DEFAULT 'service_install',
  description VARCHAR(255) NULL,
  current_version VARCHAR(32) NOT NULL DEFAULT '',
  created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  updated_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_playbook_templates_name (name),
  INDEX idx_playbook_templates_scenario (scenario),
  INDEX idx_playbook_templates_deleted (deleted_at)
);

CREATE TABLE IF NOT EXISTS playbook_template_versions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  template_id BIGINT UNSIGNED NOT NULL,
  version VARCHAR(32) NOT NULL,
  source JSON NULL,
  params_schema JSON NULL,
  defaults JSON NULL,
  created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_ptv_template_version (template_id, version),
  INDEX idx_ptv_template_id (template_id),
  INDEX idx_ptv_created_at (created_at)
);

