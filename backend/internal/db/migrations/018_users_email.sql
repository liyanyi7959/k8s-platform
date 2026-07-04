-- 018: 为 users 表新增 email 列，用于找回密码功能
-- 注意：MySQL 的 ALTER TABLE ADD COLUMN 不支持 IF NOT EXISTS，
-- 但 GORM migration 框架通过 schema_migrations 表保证幂等（失败的 migration 不会记录，重试时重新执行）。
-- 如果 email 列已存在（上次部分执行成功），需手动删除后再执行，或直接在数据库中执行后续语句。
ALTER TABLE users ADD COLUMN email VARCHAR(120) NULL AFTER username;
ALTER TABLE users ADD UNIQUE KEY uk_users_email (email);
