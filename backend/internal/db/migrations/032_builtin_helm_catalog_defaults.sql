-- 为已存在的内置 Helm 目录补齐默认来源和 values.yaml；自定义目录与已修改的字段保持不变。
UPDATE app_templates
SET
  helm_repo_name = CASE WHEN helm_repo_name = '' THEN CASE name
    WHEN 'helm-prometheus' THEN 'prometheus-community'
    ELSE 'bitnami'
  END ELSE helm_repo_name END,
  helm_repo_url = CASE WHEN helm_repo_url = '' THEN CASE name
    WHEN 'helm-prometheus' THEN 'https://prometheus-community.github.io/helm-charts'
    ELSE 'https://charts.bitnami.com/bitnami'
  END ELSE helm_repo_url END,
  helm_values_yaml = CASE WHEN COALESCE(helm_values_yaml, '') = '' THEN CASE name
    WHEN 'helm-redis' THEN 'architecture: standalone\nauth:\n  enabled: false\n'
    WHEN 'helm-nginx' THEN 'service:\n  type: ClusterIP\n'
    WHEN 'helm-mysql' THEN 'auth:\n  rootPassword: change-me\nprimary:\n  persistence:\n    enabled: true\n'
    WHEN 'helm-mongodb' THEN 'architecture: standalone\nauth:\n  enabled: false\n'
    WHEN 'helm-prometheus' THEN 'server:\n  persistentVolume:\n    enabled: false\n'
    ELSE helm_values_yaml
  END ELSE helm_values_yaml END
WHERE deleted_at IS NULL AND is_builtin = 1 AND deploy_type = 'helm';
