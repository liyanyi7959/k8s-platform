-- 应用商店模板表
CREATE TABLE IF NOT EXISTS app_templates (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200) DEFAULT '',
    description TEXT,
    category VARCHAR(50) DEFAULT '',
    icon VARCHAR(500) DEFAULT '',
    template LONGTEXT,
    variables TEXT,
    is_builtin TINYINT(1) DEFAULT 0,
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    UNIQUE INDEX idx_name (name),
    INDEX idx_category (category),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
