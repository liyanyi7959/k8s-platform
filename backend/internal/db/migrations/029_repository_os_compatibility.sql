-- 标识历史内置 YUM 源的适配发行版，避免在麒麟、Rocky 等系统上误用 CentOS 目录结构。
-- mirror_of 复用为仓库适配系统标识：centos、rocky、almalinux、rhel、kylin 等。
UPDATE deploy_repositories
SET mirror_of = 'centos'
WHERE repo_type = 'yum'
  AND (mirror_of IS NULL OR mirror_of = '')
  AND LOWER(url) LIKE '%/centos%';
