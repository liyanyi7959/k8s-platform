-- 重命名凭据表：ssh_credentials -> credentials
-- 凭据管理已扩展支持多种类型（ssh/kubeconfig/git/docker-registry/token），表名不再局限于 SSH
RENAME TABLE ssh_credentials TO credentials;
