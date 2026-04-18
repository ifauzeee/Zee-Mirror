from __future__ import annotations

import logging
from datetime import datetime

import httpx
from aiogram import Bot
from aiogram.types import User as TelegramUser

from zee_mirror.config import Settings
from zee_mirror.database import Database
from zee_mirror.models import User
from zee_mirror.repository import Repository


logger = logging.getLogger(__name__)


class BotService:
    def __init__(self, settings: Settings, db: Database, repository: Repository, bot: Bot) -> None:
        self.settings = settings
        self.db = db
        self.repository = repository
        self.bot = bot
        self.bot_username: str = ""

    async def startup(self) -> None:
        self.settings.config_path.mkdir(parents=True, exist_ok=True)
        self.settings.download_path.mkdir(parents=True, exist_ok=True)
        await self.bootstrap_users()
        me = await self.bot.get_me()
        self.bot_username = me.username or ""
        logger.info("Authorized on account %s", self.bot_username or me.id)

    async def bootstrap_users(self) -> None:
        await self.repository.upsert_user(
            User(
                id=self.settings.owner_id,
                username="Owner",
                role="owner",
                language="id",
                created_at=datetime.utcnow(),
                max_daily_tasks=-1,
                max_daily_bandwidth=-1,
            )
        )

        for user_id in self.settings.authorized_users:
            await self.repository.upsert_user(
                User(
                    id=user_id,
                    username="Authorized User",
                    role="authorized",
                    language="id",
                    created_at=datetime.utcnow(),
                    max_daily_tasks=self.settings.default_max_daily_tasks,
                    max_daily_bandwidth=_parse_bandwidth(self.settings.default_max_daily_bandwidth),
                )
            )

    async def sync_user(self, tg_user: TelegramUser | None) -> None:
        if tg_user is None:
            return

        existing = await self.repository.get_user_by_id(tg_user.id)
        role = "user"
        if tg_user.id == self.settings.owner_id:
            role = "owner"
        elif tg_user.id in self.settings.authorized_users:
            role = "authorized"

        created_at = existing.created_at if existing and existing.created_at else datetime.utcnow()
        language = existing.language if existing else "id"
        max_daily_tasks = existing.max_daily_tasks if existing else self.settings.default_max_daily_tasks
        max_daily_bandwidth = (
            existing.max_daily_bandwidth
            if existing
            else _parse_bandwidth(self.settings.default_max_daily_bandwidth)
        )

        await self.repository.upsert_user(
            User(
                id=tg_user.id,
                username=tg_user.username or tg_user.full_name,
                role=existing.role if existing and existing.role in {"owner", "authorized"} else role,
                language=language,
                created_at=created_at,
                max_daily_tasks=max_daily_tasks,
                max_daily_bandwidth=max_daily_bandwidth,
            )
        )

    async def is_authorized(self, user_id: int | None) -> bool:
        if user_id is None:
            return False
        if user_id == self.settings.owner_id or user_id in self.settings.authorized_users:
            return True

        user = await self.repository.get_user_by_id(user_id)
        if user is None:
            return False

        if user.expires_at and user.expires_at < datetime.utcnow():
            return False
        return user.role in {"owner", "authorized", "user"}

    async def get_health_payload(self) -> dict[str, object]:
        payload: dict[str, object] = {
            "status": "ok",
            "bot_username": self.bot_username,
            "db": "ok" if await self.db.ping() else "error",
            "webhook_mode": self.settings.use_webhook,
        }

        aria2_version = await self.get_aria2_version()
        if aria2_version:
            payload["aria2_version"] = aria2_version

        payload["stats"] = await self.repository.get_bot_stats()
        return payload

    async def get_aria2_version(self) -> str | None:
        request_body = {
            "jsonrpc": "2.0",
            "id": "health-check",
            "method": "aria2.getVersion",
            "params": [],
        }
        if self.settings.aria2_rpc_secret:
            request_body["params"].append(f"token:{self.settings.aria2_rpc_secret}")

        try:
            async with httpx.AsyncClient(timeout=3.0) as client:
                response = await client.post(self.settings.aria2_rpc_url, json=request_body)
                response.raise_for_status()
                data = response.json()
        except Exception as exc:
            logger.debug("Aria2 health probe skipped: %s", exc)
            return None

        result = data.get("result") if isinstance(data, dict) else None
        if isinstance(result, dict):
            return result.get("version")
        return None


def _parse_bandwidth(value: str) -> int:
    cleaned = value.strip().upper()
    if cleaned in {"", "-1"}:
        return -1

    units = {
        "KB": 1024,
        "MB": 1024**2,
        "GB": 1024**3,
        "TB": 1024**4,
    }
    for suffix, multiplier in units.items():
        if cleaned.endswith(suffix):
            number = float(cleaned[: -len(suffix)])
            return int(number * multiplier)
    return int(cleaned)
