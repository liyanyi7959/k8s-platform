-- 为 clusters 表新增 k8s_version（集群版本）和 node_count（节点数）字段。
-- 这两个字段由健康检查时自动回填，初始值分别为空字符串和 0。

ALTER TABLE clusters
  ADD COLUMN k8s_version VARCHAR(32) NOT NULL DEFAULT '' AFTER kubeconfig_enc,
  ADD COLUMN node_count  INT         NOT NULL DEFAULT 0  AFTER k8s_version;
