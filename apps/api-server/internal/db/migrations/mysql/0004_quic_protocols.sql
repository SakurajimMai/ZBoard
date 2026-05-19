-- 0004 hysteria2 / tuic fields (MySQL / MariaDB)
ALTER TABLE nodes
  ADD COLUMN obfs_password VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN congestion_control VARCHAR(32) NOT NULL DEFAULT '',
  ADD COLUMN up_mbps INT NOT NULL DEFAULT 0,
  ADD COLUMN down_mbps INT NOT NULL DEFAULT 0;
