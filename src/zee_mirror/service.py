from __future__ import annotations

import logging
from datetime import datetime
from html import escape
from urllib.parse import urlparse

import httpx
from aiogram import Bot
from aiogram.types import Message, User as TelegramUser

from zee_mirror.config import Settings
from zee_mirror.database import Database
from zee_mirror.models import RuntimeTask, TaskStatus, TaskType, User
from zee_mirror.repository import Repository
from zee_mirror.task_manager import TaskManager
from zee_mirror.transfer import TransferManager


logger = logging.getLogger(__name__)


class BotService:
    def __init__(self, settings: Settings, db: Database, repository: Repository, bot: Bot) -> None:
        self.settings = settings
        self.db = db
        self.repository = repository
        self.bot = bot
        self.bot_username: str = ""
        self.transfer = TransferManager(settings, bot)
        self.task_manager = TaskManager(
            repository=repository,
            processor=self.process_task,
            download_dir=settings.download_path,
            max_concurrent=settings.max_concurrent_downloads,
            max_retries=3,
            owner_id=settings.owner_id,
            privileged_users=set(settings.authorized_users),
        )

    async def startup(self) -> None:
        self.settings.config_path.mkdir(parents=True, exist_ok=True)
        self.settings.download_path.mkdir(parents=True, exist_ok=True)
        await self.bootstrap_users()
        await self.task_manager.startup()
        me = await self.bot.get_me()
        self.bot_username = me.username or ""
        logger.info("Authorized on account %s", self.bot_username or me.id)

    async def shutdown(self) -> None:
        await self.task_manager.shutdown()

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
            "active_tasks": len(self.task_manager.get_active_tasks()),
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

    async def create_url_task(self, message: Message, task_type: TaskType, url: str, file_name: str = "") -> RuntimeTask:
        await self.sync_user(message.from_user)
        normalized_name = file_name or self.guess_name_from_url(url)
        task = await self.task_manager.create_task(
            task_type=task_type,
            url=url,
            file_name=normalized_name,
            chat_id=message.chat.id,
            user_id=message.from_user.id if message.from_user else 0,
            message_id=message.message_id,
            reply_message_id=message.reply_to_message.message_id if message.reply_to_message else 0,
            command_text=message.text or "",
        )
        return task

    async def create_reply_file_task(self, message: Message, task_type: TaskType) -> RuntimeTask | None:
        reply = message.reply_to_message
        if reply is None:
            return None

        file_id, file_name = self.extract_file_from_message(reply)
        if not file_id:
            return None

        await self.sync_user(message.from_user)
        return await self.task_manager.create_task(
            task_type=task_type,
            url=f"telegram://{file_id}",
            file_name=file_name or "telegram-file.bin",
            chat_id=message.chat.id,
            user_id=message.from_user.id if message.from_user else 0,
            message_id=message.message_id,
            reply_message_id=reply.message_id,
            source_kind="telegram_file",
            source_file_id=file_id,
            original_name=file_name or "",
            command_text=message.text or "",
        )

    async def process_task(self, task: RuntimeTask) -> None:
        if task.status == TaskStatus.CANCELLED:
            return

        await self.task_manager.mark_status(task, TaskStatus.DOWNLOADING)
        try:
            await self.transfer.execute(task)
        except Exception as exc:
            logger.exception("Task %s failed", task.id)
            task.error = str(exc)
            task.retries += 1
            if task.retries <= task.max_retries:
                await self.bot.send_message(
                    task.chat_id,
                    (
                        f"Task <code>{escape(task.id)}</code> gagal sementara dan akan dicoba ulang.\n"
                        f"Retry: {task.retries}/{task.max_retries}\n"
                        f"Error: <code>{escape(task.error)}</code>"
                    ),
                )
                await self.task_manager.requeue_task(task, delay_seconds=min(5 * task.retries, 30))
                return

            task.status = TaskStatus.FAILED
            await self.repository.save_task(task)
            await self.bot.send_message(
                task.chat_id,
                (
                    f"Task <code>{escape(task.id)}</code> gagal.\n"
                    f"Error: <code>{escape(task.error)}</code>"
                ),
            )
            return

        await self.repository.save_task(task)
        completion_text = (
            f"Task <code>{escape(task.id)}</code> selesai.\n"
            f"Mode: <code>{escape(task.type.value)}</code>\n"
            f"File: <code>{escape(task.file_name)}</code>"
        )
        if task.type == TaskType.MIRROR and task.remote_path:
            completion_text += f"\nRemote: <code>{escape(task.remote_path)}</code>"
        await self.bot.send_message(task.chat_id, completion_text)

    def guess_name_from_url(self, url: str) -> str:
        parsed = urlparse(url)
        name = parsed.path.rsplit("/", 1)[-1]
        return name or "download.bin"

    def extract_file_from_message(self, message: Message) -> tuple[str, str]:
        if message.document:
            return message.document.file_id, message.document.file_name or "document.bin"
        if message.video:
            return message.video.file_id, message.video.file_name or "video.mp4"
        if message.audio:
            return message.audio.file_id, message.audio.file_name or "audio.bin"
        if message.voice:
            return message.voice.file_id, "voice.ogg"
        return "", ""

    def render_task_status(self, chat_id: int) -> str:
        tasks = self.task_manager.get_active_tasks(chat_id=chat_id)
        if not tasks:
            return "Tidak ada task aktif."

        lines = ["Task aktif:"]
        for task in tasks[:10]:
            lines.append(
                f"- {task.id} | {task.type.value} | {task.status.value} | "
                f"{task.progress:.1f}% | {task.file_name or task.url}"
            )
        return "\n".join(lines)

    async def cancel_task(self, task_id: str) -> bool:
        return await self.task_manager.cancel_task(task_id)


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
