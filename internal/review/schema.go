package review

const dropReviewSchemaSQL = `
DROP TABLE IF EXISTS review_doc_snippets;
DROP TABLE IF EXISTS review_docs;
`

const createReviewSchemaSQL = `
-- Review document metadata and content
CREATE TABLE review_docs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    frontmatter_json TEXT NOT NULL DEFAULT '{}',
    indexed_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_review_docs_slug ON review_docs(slug);

-- Resolved snippet references within review docs
CREATE TABLE review_doc_snippets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    doc_id INTEGER NOT NULL REFERENCES review_docs(id) ON DELETE CASCADE,
    stub_id TEXT NOT NULL,
    directive TEXT NOT NULL,
    symbol_id TEXT,
    file_path TEXT,
    kind TEXT,
    language TEXT,
    text TEXT NOT NULL DEFAULT '',
    params_json TEXT NOT NULL DEFAULT '{}',
    start_line INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    commit_hash TEXT,
    UNIQUE(doc_id, stub_id)
);

CREATE INDEX idx_review_doc_snippets_doc ON review_doc_snippets(doc_id);
CREATE INDEX idx_review_doc_snippets_sym ON review_doc_snippets(symbol_id);
`
