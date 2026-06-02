ALTER TABLE nodes ADD COLUMN sort INTEGER NOT NULL DEFAULT 0;
CREATE INDEX idx_nodes_sort ON nodes (sort, id);
