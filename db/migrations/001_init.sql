PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS words (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    word            TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    context         TEXT,
    ai_meaning      TEXT,
    ai_examples     TEXT,
    ai_scenarios    TEXT,
    ai_memory_tip   TEXT,
    ai_generated_at INTEGER,
    interval_days   INTEGER NOT NULL DEFAULT 1,
    next_review_at  INTEGER NOT NULL DEFAULT 0,
    repetitions     INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at      INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS review_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    word_id         INTEGER NOT NULL REFERENCES words(id) ON DELETE CASCADE,
    result          TEXT    NOT NULL CHECK(result IN ('correct', 'incorrect')),
    user_answer     TEXT,
    interval_before INTEGER,
    interval_after  INTEGER,
    reviewed_at     INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_words_next_review ON words(next_review_at);
CREATE INDEX IF NOT EXISTS idx_review_logs_word_id ON review_logs(word_id);
