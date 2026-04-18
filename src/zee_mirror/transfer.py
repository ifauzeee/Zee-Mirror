from __future__ import annotations

import asyncio
import mimetypes
import time
from pathlib import Path
from urllib.parse import urlparse

import httpx
from aiogram import Bot
from aiogram.types import FSInputFile

from zee_mirror.config import Settings
from zee_mirror.models import RuntimeTask, TaskStatus


class TransferManager:
    def __init__(self, settings: Settings, bot: Bot) -> None:
        self.settings = settings
        self.bot = bot

    async def execute(self, task: RuntimeTask) -> None:
        task.work_dir = task.work_dir or (Path(self.settings.download_dir) / task.id)
        task.work_dir.mkdir(parents=True, exist_ok=True)
        task.status = TaskStatus.DOWNLOADING

        if task.source_kind == "telegram_file":
            await self._download_telegram_file(task)
        else:
            await self._download_http_or_local(task)

        if not task.local_path:
            raise RuntimeError("Downloaded file not found")

        if task.type.value == "leech":
            await self._upload_to_telegram(task)
        else:
            await self._upload_with_rclone(task)

        task.progress = 100.0
        task.status = TaskStatus.COMPLETED

    async def _download_http_or_local(self, task: RuntimeTask) -> None:
        parsed = urlparse(task.url)
        target_name = task.file_name or self._guess_name(task.url)
        destination = task.work_dir / target_name
        task.file_name = target_name

        if parsed.scheme == "file":
            source = Path(parsed.path)
            await asyncio.to_thread(self._copy_local_file, source, destination, task)
            task.local_path = str(destination)
            return

        async with httpx.AsyncClient(follow_redirects=True, timeout=None) as client:
            async with client.stream("GET", task.url) as response:
                response.raise_for_status()
                content_length = response.headers.get("content-length")
                if content_length and content_length.isdigit():
                    task.total_size = int(content_length)

                disposition_name = self._filename_from_headers(response.headers.get("content-disposition"))
                if disposition_name:
                    destination = task.work_dir / disposition_name
                    task.file_name = disposition_name

                downloaded = 0
                started = time.monotonic()
                with destination.open("wb") as handle:
                    async for chunk in response.aiter_bytes(chunk_size=1024 * 256):
                        if task.status == TaskStatus.CANCELLED:
                            return
                        handle.write(chunk)
                        downloaded += len(chunk)
                        task.downloaded_size = downloaded
                        task.progress = (downloaded / task.total_size * 100) if task.total_size else 0.0
                        elapsed = max(time.monotonic() - started, 0.001)
                        task.speed = int(downloaded / elapsed)

        task.local_path = str(destination)
        if task.total_size == 0:
            task.total_size = destination.stat().st_size
        task.downloaded_size = task.total_size

    async def _download_telegram_file(self, task: RuntimeTask) -> None:
        file = await self.bot.get_file(task.source_file_id)
        file_name = task.file_name or task.original_name or Path(file.file_path or "").name or f"{task.id}.bin"
        destination = task.work_dir / file_name
        task.file_name = file_name
        await self.bot.download_file(file.file_path, destination=destination)
        task.local_path = str(destination)
        if destination.exists():
            task.total_size = destination.stat().st_size
            task.downloaded_size = task.total_size
            task.progress = 100.0

    async def _upload_to_telegram(self, task: RuntimeTask) -> None:
        task.status = TaskStatus.UPLOADING
        file_path = Path(task.local_path)
        if not file_path.exists():
            raise RuntimeError("Local file missing before Telegram upload")

        file_input = FSInputFile(file_path)
        mime_type, _ = mimetypes.guess_type(file_path.name)
        caption = f"File: {file_path.name}"
        if mime_type and mime_type.startswith("video/"):
            sent = await self.bot.send_video(chat_id=task.chat_id, video=file_input, caption=caption)
        else:
            sent = await self.bot.send_document(chat_id=task.chat_id, document=file_input, caption=caption)
        task.result_message_id = sent.message_id
        task.remote_path = "telegram"
        task.uploaded_size = file_path.stat().st_size

    async def _upload_with_rclone(self, task: RuntimeTask) -> None:
        task.status = TaskStatus.UPLOADING
        file_path = Path(task.local_path)
        if not file_path.exists():
            raise RuntimeError("Local file missing before rclone upload")

        config_path = Path(self.settings.config_dir) / "rclone.conf"
        if not config_path.exists():
            raise RuntimeError("rclone.conf not found in config directory")

        remote_dest = self.settings.rclone_dest
        task.remote_path = f"{remote_dest.rstrip('/')}/{file_path.name}"

        process = await asyncio.create_subprocess_exec(
            "rclone",
            "copy",
            str(file_path),
            remote_dest,
            "--config",
            str(config_path),
            "--progress",
            "--stats",
            "1s",
            "--stats-one-line",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.STDOUT,
        )
        assert process.stdout is not None

        while True:
            line = await process.stdout.readline()
            if not line:
                break
            if task.status == TaskStatus.CANCELLED:
                process.kill()
                await process.wait()
                return

        return_code = await process.wait()
        if return_code != 0:
            raise RuntimeError(f"rclone copy failed with exit code {return_code}")

        task.remote_url = ""
        task.uploaded_size = file_path.stat().st_size

    @staticmethod
    def _guess_name(url: str) -> str:
        parsed = urlparse(url)
        name = Path(parsed.path).name
        return name or "download.bin"

    @staticmethod
    def _filename_from_headers(content_disposition: str | None) -> str | None:
        if not content_disposition:
            return None
        for item in content_disposition.split(";"):
            part = item.strip()
            if part.startswith("filename="):
                return part.split("=", 1)[1].strip("\"'")
        return None

    @staticmethod
    def _copy_local_file(source: Path, destination: Path, task: RuntimeTask) -> None:
        if not source.exists():
            raise RuntimeError(f"Source file not found: {source}")
        total = source.stat().st_size
        task.total_size = total
        downloaded = 0
        started = time.monotonic()
        with source.open("rb") as reader, destination.open("wb") as writer:
            while True:
                chunk = reader.read(1024 * 256)
                if not chunk:
                    break
                writer.write(chunk)
                downloaded += len(chunk)
                task.downloaded_size = downloaded
                task.progress = (downloaded / total * 100) if total else 0.0
                elapsed = max(time.monotonic() - started, 0.001)
                task.speed = int(downloaded / elapsed)
