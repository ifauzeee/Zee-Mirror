CREATE TABLE IF NOT EXISTS task_checkpoints (
    task_id TEXT PRIMARY KEY,
    downloaded_bytes INTEGER NOT NULL,
    total_bytes INTEGER NOT NULL,
    progress REAL NOT NULL,
    last_update INTEGER NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);
