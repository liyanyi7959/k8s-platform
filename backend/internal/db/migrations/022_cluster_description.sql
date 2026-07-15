ALTER TABLE clusters
  ADD COLUMN description VARCHAR(500) NOT NULL DEFAULT '' AFTER k8s_version;
