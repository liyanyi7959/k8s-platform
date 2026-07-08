-- 020: 为 app_templates 表新增 deploy_type 列，区分 yaml 模板和 helm 模板
ALTER TABLE app_templates ADD COLUMN deploy_type VARCHAR(20) DEFAULT 'yaml' AFTER is_builtin;
