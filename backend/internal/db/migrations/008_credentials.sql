CREATE TABLE IF NOT EXISTS credentials (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  type VARCHAR(32) NOT NULL,
  `desc` VARCHAR(255) NULL,
  meta JSON NULL,
  data_enc LONGTEXT NOT NULL,
  created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  updated_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_credentials_type_name (type, name),
  INDEX idx_credentials_type (type),
  INDEX idx_credentials_created_by (created_by),
  INDEX idx_credentials_deleted (deleted_at)
);

