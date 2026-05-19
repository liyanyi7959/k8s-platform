CREATE TABLE IF NOT EXISTS server_terminal_favorites (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  server_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_terminal_favorites_user_server (user_id, server_id),
  INDEX idx_terminal_favorites_user (user_id),
  INDEX idx_terminal_favorites_server (server_id)
);

CREATE TABLE IF NOT EXISTS server_terminal_audits (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  server_id BIGINT UNSIGNED NOT NULL,
  server_name VARCHAR(120) NOT NULL DEFAULT '',
  server_ip VARCHAR(64) NOT NULL DEFAULT '',
  status VARCHAR(24) NOT NULL DEFAULT 'running',
  close_reason VARCHAR(255) NULL,
  risk_level VARCHAR(24) NOT NULL DEFAULT 'none',
  risk_count INT NOT NULL DEFAULT 0,
  last_command VARCHAR(255) NULL,
  started_at DATETIME(3) NOT NULL,
  last_active_at DATETIME(3) NOT NULL,
  ended_at DATETIME(3) NULL,
  idle_timeout_sec INT NOT NULL DEFAULT 900,
  absolute_timeout_sec INT NOT NULL DEFAULT 14400,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_terminal_audits_session (session_id),
  INDEX idx_terminal_audits_user (user_id),
  INDEX idx_terminal_audits_server (server_id),
  INDEX idx_terminal_audits_status (status),
  INDEX idx_terminal_audits_started_at (started_at)
);