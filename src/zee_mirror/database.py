from __future__ import annotations

import asyncio
from pathlib import Path
from typing import Any

import aiosqlite


class Database:
    def __init__(self, db_path: Path) -> None:
        self.db_path = db_path
        self._connection: aiosqlite.Connection | None = None
        self._lock = asyncio.Lock()

    async def connect(self) -> None:
        self.db_path.parent.mkdir(parents=True, exist_ok=True)
        self._connection = await aiosqlite.connect(self.db_path)
        self._connection.row_factory = aiosqlite.Row
        await self._connection.execute("PRAGMA journal_mode=WAL")
        await self._connection.execute("PRAGMA busy_timeout=5000")
        await self._connection.commit()

    async def close(self) -> None:
        if self._connection is not None:
            await self._connection.close()
            self._connection = None

    async def ping(self) -> bool:
        row = await self.fetchone("SELECT 1 AS ok")
        return bool(row and row["ok"] == 1)

    async def execute(self, query: str, params: tuple[Any, ...] = ()) -> None:
        async with self._lock:
            connection = self._require_connection()
            await connection.execute(query, params)
            await connection.commit()

    async def fetchone(self, query: str, params: tuple[Any, ...] = ()) -> aiosqlite.Row | None:
        async with self._lock:
            connection = self._require_connection()
            async with connection.execute(query, params) as cursor:
                return await cursor.fetchone()

    async def fetchall(self, query: str, params: tuple[Any, ...] = ()) -> list[aiosqlite.Row]:
        async with self._lock:
            connection = self._require_connection()
            async with connection.execute(query, params) as cursor:
                return await cursor.fetchall()

    def _require_connection(self) -> aiosqlite.Connection:
        if self._connection is None:
            raise RuntimeError("Database connection has not been initialized")
        return self._connection
