ALTER TABLE app_templates
  ADD COLUMN status VARCHAR(32) NOT NULL DEFAULT 'ready' AFTER engine,
  ADD COLUMN source_type VARCHAR(32) NOT NULL DEFAULT 'custom' AFTER source,
  ADD COLUMN source_url VARCHAR(512) NOT NULL DEFAULT '' AFTER source_type,
  ADD COLUMN source_ref JSON NULL AFTER source_url,
  ADD COLUMN chart_name VARCHAR(120) NOT NULL DEFAULT '' AFTER owner,
  ADD COLUMN app_version VARCHAR(64) NOT NULL DEFAULT '' AFTER chart_name,
  ADD COLUMN readme LONGTEXT NULL AFTER app_version,
  ADD COLUMN env_example LONGTEXT NULL AFTER readme,
  ADD COLUMN project_name_default VARCHAR(128) NOT NULL DEFAULT '' AFTER env_example,
  ADD COLUMN install_dir_default VARCHAR(256) NOT NULL DEFAULT '' AFTER project_name_default,
  ADD COLUMN extra_files JSON NULL AFTER install_dir_default,
  ADD INDEX idx_app_templates_status (status),
  ADD INDEX idx_app_templates_source_type (source_type);

ALTER TABLE app_releases
  ADD COLUMN template_engine VARCHAR(16) NOT NULL DEFAULT 'helm' AFTER name,
  ADD COLUMN target_type VARCHAR(16) NOT NULL DEFAULT '' AFTER namespace,
  ADD COLUMN target_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER target_type,
  ADD COLUMN target_name VARCHAR(120) NOT NULL DEFAULT '' AFTER target_id,
  ADD COLUMN project_name VARCHAR(128) NOT NULL DEFAULT '' AFTER target_name,
  ADD COLUMN install_dir VARCHAR(256) NOT NULL DEFAULT '' AFTER project_name,
  ADD COLUMN env_override LONGTEXT NULL AFTER install_dir,
  ADD COLUMN pull_images TINYINT(1) NOT NULL DEFAULT 1 AFTER env_override,
  ADD COLUMN auto_install_docker TINYINT(1) NOT NULL DEFAULT 1 AFTER pull_images,
  ADD COLUMN auto_install_compose TINYINT(1) NOT NULL DEFAULT 1 AFTER auto_install_docker,
  ADD INDEX idx_app_releases_target_type (target_type),
  ADD INDEX idx_app_releases_target_id (target_id);