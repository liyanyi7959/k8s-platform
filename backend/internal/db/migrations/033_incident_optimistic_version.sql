ALTER TABLE monitor_incidents
  ADD COLUMN version BIGINT UNSIGNED NOT NULL DEFAULT 1 AFTER verification_note;
