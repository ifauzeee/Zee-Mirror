CREATE TABLE IF NOT EXISTS task_checkpoints (
    task_id TEXT PRIMARY KEY,
    downloaded_bytes BIGINT NOT NULL,
    total_bytes BIGINT NOT NULL,
    progress REAL NOT NULL,
    last_update BIGINT NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);
