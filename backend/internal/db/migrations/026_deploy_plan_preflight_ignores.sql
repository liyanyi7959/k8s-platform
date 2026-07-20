ALTER TABLE deploy_plans
  ADD COLUMN preflight_ignores JSON NULL COMMENT '人工确认忽略的可忽略预检项' AFTER step_overrides;
