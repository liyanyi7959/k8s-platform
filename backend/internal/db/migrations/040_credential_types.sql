-- 扩展凭据表支持多种凭据类型（SSH/Git/Docker Registry/Kubeconfig/Token）
-- 向后兼容：现有数据 type 默认为 'ssh'，auth_type 字段保留用于 SSH 子类型

ALTER TABLE ssh_credentials
  ADD COLUMN type VARCHAR(32) NOT NULL DEFAULT 'ssh' AFTER name;

-- 将现有数据根据 auth_type 推导 type
UPDATE ssh_credentials SET type = 'ssh' WHERE type = '' OR type IS NULL;
