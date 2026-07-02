ALTER TABLE deploy_plans
  ADD COLUMN step_overrides JSON NULL COMMENT '部署计划级步骤命令覆盖，不影响 deploy_configs 模板' AFTER addons;