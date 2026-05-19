ALTER TABLE credentials
  ADD COLUMN provider VARCHAR(32) NOT NULL DEFAULT '' AFTER name,
  ADD COLUMN auth_type VARCHAR(32) NOT NULL DEFAULT '' AFTER provider;

UPDATE credentials SET provider = `type` WHERE provider = '' OR provider IS NULL;
UPDATE credentials SET auth_type = 'generic_json' WHERE auth_type = '' OR auth_type IS NULL;

ALTER TABLE credentials
  DROP INDEX uk_credentials_type_name,
  DROP INDEX idx_credentials_type,
  DROP COLUMN `type`,
  ADD UNIQUE KEY uk_credentials_provider_name (provider, name),
  ADD INDEX idx_credentials_provider (provider),
  ADD INDEX idx_credentials_auth_type (auth_type);
