-- 为任务日志增加步骤标识，支持按步骤查询日志。
ALTER TABLE task_logs ADD COLUMN step_key VARCHAR(128) NOT NULL DEFAULT '' COMMENT '关联的步骤 key';
CREATE INDEX idx_task_logs_step_key ON task_logs(task_id, step_key);
