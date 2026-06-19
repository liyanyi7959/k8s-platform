-- deploy_servers 表添加 credential_id 字段，支持关联已有 SSH 凭据
ALTER TABLE deploy_servers ADD COLUMN credential_id BIGINT UNSIGNED NULL AFTER auth_type;
ALTER TABLE deploy_servers ADD CONSTRAINT fk_deploy_servers_credential FOREIGN KEY (credential_id) REFERENCES ssh_credentials(id) ON DELETE SET NULL;
