-- 023: 扩展系统管理模块数据模型
-- - users 表新增 nickname 列
-- - roles 表新增 code 列
-- - 新增 system_settings 表存储全局配置

ALTER TABLE users ADD COLUMN nickname VARCHAR(80) NULL AFTER username;
ALTER TABLE users ADD KEY idx_users_nickname (nickname);

ALTER TABLE roles ADD COLUMN code VARCHAR(32) NULL AFTER name;
ALTER TABLE roles ADD UNIQUE KEY uk_roles_code (code);
ALTER TABLE roles ADD KEY idx_roles_code (code);

CREATE TABLE IF NOT EXISTS system_settings (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `key` VARCHAR(64) NOT NULL,
  `value` TEXT NOT NULL,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_system_settings_key (`key`),
  KEY idx_system_settings_updated (updated_at)
);
