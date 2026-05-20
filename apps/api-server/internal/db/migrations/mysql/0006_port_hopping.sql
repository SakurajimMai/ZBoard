-- 0006 hysteria2 port hopping (MySQL / MariaDB)
ALTER TABLE nodes ADD COLUMN port_range VARCHAR(32) NOT NULL DEFAULT '';
