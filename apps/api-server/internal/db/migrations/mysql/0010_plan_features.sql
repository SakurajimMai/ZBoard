-- 0010 plan custom feature bullets (MySQL / MariaDB)
ALTER TABLE plans
  ADD COLUMN features_json VARCHAR(4096) NOT NULL DEFAULT '[]';
