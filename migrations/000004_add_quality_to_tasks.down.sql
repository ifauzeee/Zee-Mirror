-- Migration to remove quality column from tasks table
-- SQLite doesn't support DROP COLUMN easily in older versions, but for simple schemas we usually don't need it or use a temp table.
-- Given it's SQLite, we can just leave it or use the standard approach.
-- For now, just a placeholder as Up is what matters mostly.
PRAGMA foreign_keys=off;
BEGIN TRANSACTION;
CREATE TABLE tasks_new (
    id TEXT PRIMARY KEY,
    gid TEXT,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    url TEXT,
    file_name TEXT,
    local_path TEXT,
    remote_path TEXT,
    remote_url TEXT,
    total_size INTEGER DEFAULT 0,
    downloaded_size INTEGER DEFAULT 0,
    uploaded_size INTEGER DEFAULT 0,
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    completed_at DATETIME,
    zip BOOLEAN DEFAULT FALSE,
    unzip BOOLEAN DEFAULT FALSE,
    password TEXT,
    error TEXT,
    retries INTEGER DEFAULT 0
);
INSERT INTO tasks_new SELECT id, gid, type, status, url, file_name, local_path, remote_path, remote_url, total_size, downloaded_size, uploaded_size, chat_id, user_id, created_at, completed_at, zip, unzip, password, error, retries FROM tasks;
DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;
CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
COMMIT;
PRAGMA foreign_keys=on;
