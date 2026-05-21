-- 0010 plan custom feature bullets (SQLite)
ALTER TABLE plans ADD COLUMN features_json TEXT NOT NULL DEFAULT '[]';
