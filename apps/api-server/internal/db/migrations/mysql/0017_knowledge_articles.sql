-- 0017 user-facing knowledge base articles (MySQL/MariaDB)
CREATE TABLE IF NOT EXISTS knowledge_articles (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  slug VARCHAR(96) NOT NULL UNIQUE,
  title VARCHAR(200) NOT NULL,
  category VARCHAR(100) NOT NULL DEFAULT '',
  summary TEXT NOT NULL,
  content MEDIUMTEXT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_knowledge_active (status, category, sort_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
