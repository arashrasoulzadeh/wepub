CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS ops (
  seq INTEGER PRIMARY KEY AUTOINCREMENT,
  id TEXT UNIQUE NOT NULL,
  device_id TEXT,
  hlc TEXT,
  entity TEXT,
  entity_id TEXT,
  field TEXT,
  value BLOB,
  created_at TEXT
);
