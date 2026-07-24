-- 部署计划新增 Helm 安装选项
ALTER TABLE deploy_plans
  ADD COLUMN helm_install TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否安装 Helm' AFTER addons;
