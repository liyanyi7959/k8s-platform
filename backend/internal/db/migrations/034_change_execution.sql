-- 034: Change 领域执行登记与提案修订历史
-- 说明：Change 上下文补齐 Execution（持久化执行登记 + 幂等键）与 Revision（状态变更审计）。
--       AI 提案确认通过后登记 execution，AI 同步执行完成/失败后回填终态。

CREATE TABLE IF NOT EXISTS change_executions (
  id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  proposal_id       BIGINT UNSIGNED NOT NULL,
  cluster_id        BIGINT UNSIGNED NOT NULL DEFAULT 0,
  execution_no      INT NOT NULL DEFAULT 1,
  status            VARCHAR(32) NOT NULL DEFAULT 'pending',
  idempotency_key   VARCHAR(128) NULL,
  operator_id       BIGINT UNSIGNED NOT NULL DEFAULT 0,
  operator_name     VARCHAR(80) NOT NULL DEFAULT '',
  command_snapshot  TEXT NULL,
  result_json       JSON NULL,
  error_message     TEXT NULL,
  started_at        DATETIME(3) NULL,
  finished_at       DATETIME(3) NULL,
  created_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at        DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_change_executions_idempotency (idempotency_key),
  INDEX idx_change_executions_proposal_status (proposal_id, status),
  INDEX idx_change_executions_status_created (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS change_proposal_revisions (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  proposal_id  BIGINT UNSIGNED NOT NULL,
  cluster_id   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  from_status  VARCHAR(32) NOT NULL DEFAULT '',
  to_status    VARCHAR(32) NOT NULL DEFAULT '',
  actor_id     BIGINT UNSIGNED NOT NULL DEFAULT 0,
  actor_name   VARCHAR(80) NOT NULL DEFAULT '',
  decision     VARCHAR(64) NOT NULL DEFAULT '',
  created_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_change_proposal_revisions_proposal (proposal_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
