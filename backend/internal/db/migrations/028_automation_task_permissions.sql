-- 通用自动化任务中心权限。仅为内置管理员自动授予，其他角色须由管理员显式配置。
INSERT IGNORE INTO permissions (code, `desc`) VALUES
  ('automation:read', '自动化任务查看'),
  ('automation:execute', '自动化任务取消与执行');

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('automation:read', 'automation:execute') AND p.deleted_at IS NULL
WHERE r.deleted_at IS NULL AND (r.name = 'admin' OR r.code = 'admin');
