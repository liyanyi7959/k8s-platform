-- 修复：集群软删除后，原 name 唯一索引导致无法重新导入同名集群。
-- 将唯一索引改为仅对未删除记录生效（MySQL 使用生成列实现部分唯一索引）。
ALTER TABLE clusters
  DROP INDEX uk_clusters_name,
  ADD COLUMN name_unique VARCHAR(120) AS (CASE WHEN deleted_at IS NULL THEN name ELSE NULL END) STORED,
  ADD UNIQUE KEY uk_clusters_name_active (name_unique);
