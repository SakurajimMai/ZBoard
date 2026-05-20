-- 0006 hysteria2 port hopping (PostgreSQL)
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS port_range TEXT NOT NULL DEFAULT '';
