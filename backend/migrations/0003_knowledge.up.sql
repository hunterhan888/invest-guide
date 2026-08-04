CREATE TABLE knowledge_docs (
    id               TEXT PRIMARY KEY,
    title            TEXT NOT NULL,
    country          TEXT NOT NULL DEFAULT '',
    source_type      TEXT NOT NULL,
    source_url       TEXT,
    original_content TEXT,
    status           TEXT NOT NULL DEFAULT 'pending',
    error_message    TEXT,
    chunk_count      INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_knowledge_docs_country ON knowledge_docs(country);
CREATE INDEX idx_knowledge_docs_status ON knowledge_docs(status);

CREATE TABLE knowledge_chunks (
    id          TEXT PRIMARY KEY,
    doc_id      TEXT NOT NULL REFERENCES knowledge_docs(id) ON DELETE CASCADE,
    seq         INTEGER NOT NULL,
    content     TEXT NOT NULL,
    embedding   vector(1024) NOT NULL,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(doc_id, seq)
);
CREATE INDEX idx_knowledge_chunks_doc_id ON knowledge_chunks(doc_id);
