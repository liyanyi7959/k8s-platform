ALTER TABLE cicd_pipelines
  ADD COLUMN cluster_id BIGINT UNSIGNED NULL AFTER cron,
  ADD COLUMN namespace VARCHAR(255) NOT NULL DEFAULT 'cicd' AFTER cluster_id,
  ADD COLUMN runner_image VARCHAR(255) NOT NULL DEFAULT 'alpine:3.20' AFTER namespace;

ALTER TABLE cicd_runs
  ADD COLUMN executor_job_name VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN executor_namespace VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN executor_cluster_id BIGINT UNSIGNED NULL;
