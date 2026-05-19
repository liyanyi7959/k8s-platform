CREATE TABLE IF NOT EXISTS servers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  ip VARCHAR(64) NOT NULL,
  port INT NOT NULL DEFAULT 22,
  auth_type VARCHAR(16) NOT NULL DEFAULT 'password',
  username VARCHAR(80) NOT NULL,
  password_enc LONGTEXT NULL,
  private_key_enc LONGTEXT NULL,
  tags LONGTEXT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  last_check_at DATETIME(3) NULL,
  created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_servers_name (name),
  INDEX idx_servers_ip (ip),
  INDEX idx_servers_status (status),
  INDEX idx_servers_created_by (created_by),
  INDEX idx_servers_deleted (deleted_at),
  INDEX idx_servers_last_check_at (last_check_at)
);

