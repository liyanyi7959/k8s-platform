-- 036: Provisioning 部署执行登记（Durable Job：Lease + Heartbeat）
-- 部署长任务在执行过程中通过心跳续期租约；进程崩溃后租约过期，由
-- 进程内 worker 兜底标记失败并联动恢复部署计划状态。
CREATE TABLE IF NOT EXISTS provision_deploy_jobs (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  plan_id          BIGINT UNSIGNED NOT NULL,
  task_id          BIGINT UNSIGNED NOT NULL DEFAULT 0,
  job_type         VARCHAR(64) NOT NULL DEFAULT 'deploy_cluster',
  status           VARCHAR(32) NOT NULL DEFAULT 'running',
  lease_expires_at DATETIME(3) NULL,
  message          TEXT NULL,
  created_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at       DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_provision_deploy_jobs_plan (plan_id),
  INDEX idx_provision_deploy_jobs_status_lease (status, lease_expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
