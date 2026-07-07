-- 项目（多租户/命名空间分组）表
CREATE TABLE IF NOT EXISTS projects (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) DEFAULT '',
    cluster_id BIGINT UNSIGNED DEFAULT 0,
    namespaces TEXT,
    quota_cpu VARCHAR(20) DEFAULT '',
    quota_memory VARCHAR(20) DEFAULT '',
    quota_pods VARCHAR(20) DEFAULT '',
    creator_id BIGINT UNSIGNED DEFAULT 0,
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    UNIQUE INDEX idx_name (name),
    INDEX idx_cluster_id (cluster_id),
    INDEX idx_creator_id (creator_id),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
