-- 0010 plan custom feature bullets (PostgreSQL)
ALTER TABLE plans
  ADD COLUMN IF NOT EXISTS features_json TEXT NOT NULL DEFAULT '[]';
