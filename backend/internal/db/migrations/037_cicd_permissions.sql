INSERT IGNORE INTO permissions (code, `desc`) VALUES
  ('cicd:read', 'CI/CD pipeline, run, and artifact read access'),
  ('cicd:write', 'CI/CD pipeline and environment configuration'),
  ('cicd:execute', 'CI/CD pipeline run and cancellation');

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('cicd:read', 'cicd:write', 'cicd:execute') AND p.deleted_at IS NULL
WHERE r.deleted_at IS NULL AND (r.name = 'admin' OR r.code = 'admin');
