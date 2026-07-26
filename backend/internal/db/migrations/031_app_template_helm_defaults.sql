-- Helm 应用目录的默认来源与 values.yaml。真实仓库状态仍由各集群 Master 的 Helm 管理。
ALTER TABLE app_templates
    ADD COLUMN helm_repo_name VARCHAR(100) NOT NULL DEFAULT '' AFTER deploy_type,
    ADD COLUMN helm_repo_url VARCHAR(500) NOT NULL DEFAULT '' AFTER helm_repo_name,
    ADD COLUMN helm_chart_version VARCHAR(100) NOT NULL DEFAULT '' AFTER helm_repo_url,
    ADD COLUMN helm_values_yaml LONGTEXT AFTER helm_chart_version;
