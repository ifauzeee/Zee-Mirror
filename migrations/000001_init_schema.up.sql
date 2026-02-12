CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,
    username TEXT,
    role TEXT NOT NULL DEFAULT 'user',
    created_at DATETIME NOT NULL,
    max_daily_tasks INTEGER DEFAULT -1,
    max_daily_bandwidth INTEGER DEFAULT -1,
    expires_at DATETIME
);

CREATE TABLE IF NOT EXISTS tasks (
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

CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME NOT NULL
);
