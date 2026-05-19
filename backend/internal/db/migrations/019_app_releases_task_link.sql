ALTER TABLE app_releases
  ADD COLUMN last_task_id BIGINT UNSIGNED NULL AFTER auto_install_compose,
  ADD INDEX idx_app_releases_last_task_id (last_task_id);
