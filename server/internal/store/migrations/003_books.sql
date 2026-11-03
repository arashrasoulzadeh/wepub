CREATE TABLE IF NOT EXISTS books (
  sha256 TEXT PRIMARY KEY,
  title TEXT,
  author TEXT,
  language TEXT,
  direction TEXT,
  fixed INTEGER,
  cover_sha256 TEXT,
  added_at TEXT
);
