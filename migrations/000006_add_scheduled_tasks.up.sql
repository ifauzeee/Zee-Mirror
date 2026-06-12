CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id          TEXT PRIMARY KEY,
    task_type   TEXT NOT NULL,
    url         TEXT NOT NULL,
    file_name   TEXT NOT NULL DEFAULT '',
    chat_id     INTEGER NOT NULL,
    user_id     INTEGER NOT NULL,
    zip         INTEGER DEFAULT 0,
    unzip       INTEGER DEFAULT 0,
    password    TEXT DEFAULT '',
    quality     TEXT DEFAULT '',
    scheduled_at DATETIME NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    task_id     TEXT DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
