from __future__ import annotations

import sqlite3
from pathlib import Path


def apply_migrations(db_path: Path, migrations_dir: Path) -> None:
    db_path.parent.mkdir(parents=True, exist_ok=True)

    with sqlite3.connect(db_path) as connection:
        connection.execute(
            """
            CREATE TABLE IF NOT EXISTS schema_migrations (
                version INTEGER NOT NULL,
                dirty BOOLEAN NOT NULL
            )
            """
        )
        connection.commit()

        row = connection.execute(
            "SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1"
        ).fetchone()
        current_version = int(row[0]) if row else 0
        dirty = bool(row[1]) if row else False
        if dirty:
            raise RuntimeError(f"Database is marked dirty at migration version {current_version}")

        for migration in sorted(migrations_dir.glob("*_*.up.sql")):
            version = int(migration.name.split("_", 1)[0])
            if version <= current_version:
                continue

            connection.execute("DELETE FROM schema_migrations")
            connection.execute(
                "INSERT INTO schema_migrations(version, dirty) VALUES (?, ?)",
                (version, True),
            )
            connection.commit()

            script = migration.read_text(encoding="utf-8")
            try:
                connection.executescript(script)
            except sqlite3.OperationalError as exc:
                message = str(exc).lower()
                duplicate_column = "duplicate column name" in message
                existing_table = "already exists" in message
                if not (duplicate_column or existing_table):
                    raise

            connection.execute("DELETE FROM schema_migrations")
            connection.execute(
                "INSERT INTO schema_migrations(version, dirty) VALUES (?, ?)",
                (version, False),
            )
            connection.commit()
