-- 0006 hysteria2 port hopping (SQLite)
ALTER TABLE nodes ADD COLUMN port_range TEXT NOT NULL DEFAULT '';
