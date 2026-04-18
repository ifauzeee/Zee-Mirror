from __future__ import annotations

from datetime import datetime

from zee_mirror.database import Database
from zee_mirror.models import User


class Repository:
    def __init__(self, db: Database) -> None:
        self.db = db

    async def get_user_by_id(self, user_id: int) -> User | None:
        row = await self.db.fetchone(
            """
            SELECT id, username, role, language, max_daily_tasks, max_daily_bandwidth, created_at, expires_at
            FROM users
            WHERE id = ?
            """,
            (user_id,),
        )
        if row is None:
            return None

        return User(
            id=row["id"],
            username=row["username"] or "",
            role=row["role"] or "user",
            language=row["language"] or "id",
            max_daily_tasks=row["max_daily_tasks"] if row["max_daily_tasks"] is not None else -1,
            max_daily_bandwidth=row["max_daily_bandwidth"] if row["max_daily_bandwidth"] is not None else -1,
            created_at=_parse_datetime(row["created_at"]),
            expires_at=_parse_datetime(row["expires_at"]),
        )

    async def upsert_user(self, user: User) -> None:
        created_at = (user.created_at or datetime.utcnow()).isoformat(sep=" ", timespec="seconds")
        expires_at = (
            user.expires_at.isoformat(sep=" ", timespec="seconds")
            if user.expires_at is not None
            else None
        )
        await self.db.execute(
            """
            INSERT INTO users (id, username, role, language, created_at, max_daily_tasks, max_daily_bandwidth, expires_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(id) DO UPDATE SET
                username = excluded.username,
                role = CASE WHEN users.role = 'owner' THEN 'owner' ELSE excluded.role END,
                language = excluded.language,
                max_daily_tasks = excluded.max_daily_tasks,
                max_daily_bandwidth = excluded.max_daily_bandwidth,
                expires_at = excluded.expires_at
            """,
            (
                user.id,
                user.username,
                user.role,
                user.language,
                created_at,
                user.max_daily_tasks,
                user.max_daily_bandwidth,
                expires_at,
            ),
        )

    async def get_setting(self, key: str) -> str | None:
        row = await self.db.fetchone("SELECT value FROM settings WHERE key = ?", (key,))
        return None if row is None else row["value"]

    async def set_setting(self, key: str, value: str) -> None:
        await self.db.execute(
            """
            INSERT INTO settings (key, value, updated_at)
            VALUES (?, ?, ?)
            ON CONFLICT(key) DO UPDATE SET
                value = excluded.value,
                updated_at = excluded.updated_at
            """,
            (key, value, datetime.utcnow().isoformat(sep=" ", timespec="seconds")),
        )

    async def get_bot_stats(self) -> dict[str, int]:
        total_row = await self.db.fetchone("SELECT COUNT(*) AS count FROM tasks")
        completed_row = await self.db.fetchone("SELECT COUNT(*) AS count FROM tasks WHERE status = 'completed'")
        failed_row = await self.db.fetchone("SELECT COUNT(*) AS count FROM tasks WHERE status = 'failed'")
        bandwidth_row = await self.db.fetchone(
            "SELECT COALESCE(SUM(total_size), 0) AS total_bandwidth FROM tasks WHERE status = 'completed'"
        )
        return {
            "total_tasks": int(total_row["count"]) if total_row else 0,
            "completed_tasks": int(completed_row["count"]) if completed_row else 0,
            "failed_tasks": int(failed_row["count"]) if failed_row else 0,
            "total_bandwidth": int(bandwidth_row["total_bandwidth"]) if bandwidth_row else 0,
        }


def _parse_datetime(value: str | None) -> datetime | None:
    if not value:
        return None
    for fmt in ("%Y-%m-%d %H:%M:%S", "%Y-%m-%d %H:%M:%S.%f", "%Y-%m-%dT%H:%M:%S", "%Y-%m-%dT%H:%M:%S.%f"):
        try:
            return datetime.strptime(value, fmt)
        except ValueError:
            continue
    return None
