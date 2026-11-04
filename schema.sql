
-- ==========================================
-- BACKEND ENGINEER ASSESSMENT 2026
-- DATABASE: article_service
-- PostgreSQL
-- ==========================================

-- 1. AUTHORS TABLE
CREATE TABLE IF NOT EXISTS authors (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

-- 2. ARTICLES TABLE
CREATE TABLE IF NOT EXISTS articles (
    id TEXT PRIMARY KEY,

    author_id TEXT NOT NULL,

    title TEXT NOT NULL,
    body TEXT NOT NULL,

    created_at TIMESTAMP NOT NULL
        DEFAULT CURRENT_TIMESTAMP,

    updated_at TIMESTAMP NOT NULL
        DEFAULT CURRENT_TIMESTAMP,

    search_vector TSVECTOR
        GENERATED ALWAYS AS (
            to_tsvector(
                'simple',
                title || ' ' || body
            )
        ) STORED,

    CONSTRAINT fk_article_author
        FOREIGN KEY (author_id)
        REFERENCES authors(id)
);

-- ==========================================
-- INDEXES
-- ==========================================

-- 3. FULL-TEXT SEARCH INDEX
-- Supports keyword search in title and body
CREATE INDEX IF NOT EXISTS idx_articles_search
ON articles
USING GIN (search_vector);

-- 4. ARTICLE SORTING AND CURSOR PAGINATION
-- Supports newest-first ordering
CREATE INDEX IF NOT EXISTS idx_articles_created
ON articles (
    created_at DESC,
    id DESC
);

-- 5. AUTHOR NAME INDEX
-- Supports filtering by author name
CREATE INDEX IF NOT EXISTS idx_authors_name
ON authors (
    name,
    id
);

-- 6. AUTHOR ARTICLE INDEX
-- Supports author-based filtering and sorting
CREATE INDEX IF NOT EXISTS idx_articles_author_created
ON articles (
    author_id,
    created_at DESC,
    id DESC
);

-- ==========================================
-- OPTIONAL: UPDATE PLANNER STATISTICS
-- ==========================================

ANALYZE authors;
ANALYZE articles;