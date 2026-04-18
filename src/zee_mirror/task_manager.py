from __future__ import annotations

import asyncio
import logging
from datetime import datetime
from pathlib import Path
from typing import Awaitable, Callable
from uuid import uuid4

from zee_mirror.models import RuntimeTask, TaskStatus, TaskType
from zee_mirror.repository import Repository


logger = logging.getLogger(__name__)

TaskProcessor = Callable[[RuntimeTask], Awaitable[None]]


class TaskManager:
    def __init__(
        self,
        repository: Repository,
        processor: TaskProcessor,
        download_dir: Path,
        max_concurrent: int,
        max_retries: int,
        owner_id: int,
        privileged_users: set[int],
    ) -> None:
        self.repository = repository
        self.processor = processor
        self.download_dir = download_dir
        self.max_concurrent = max(1, max_concurrent)
        self.max_retries = max_retries
        self.owner_id = owner_id
        self.privileged_users = privileged_users
        self.tasks: dict[str, RuntimeTask] = {}
        self._queue: asyncio.PriorityQueue[tuple[int, float, str]] = asyncio.PriorityQueue()
        self._workers: list[asyncio.Task[None]] = []
        self._shutdown = asyncio.Event()

    async def startup(self) -> None:
        self.download_dir.mkdir(parents=True, exist_ok=True)
        await self._restore_active_tasks()
        for index in range(self.max_concurrent):
            self._workers.append(asyncio.create_task(self._worker(index)))

    async def shutdown(self) -> None:
        self._shutdown.set()
        for worker in self._workers:
            worker.cancel()
        if self._workers:
            await asyncio.gather(*self._workers, return_exceptions=True)

    async def create_task(
        self,
        *,
        task_type: TaskType,
        url: str,
        file_name: str,
        chat_id: int,
        user_id: int,
        message_id: int = 0,
        reply_message_id: int = 0,
        source_kind: str = "url",
        source_file_id: str = "",
        original_name: str = "",
        command_text: str = "",
    ) -> RuntimeTask:
        task_id = uuid4().hex[:8]
        task = RuntimeTask(
            id=task_id,
            type=task_type,
            status=TaskStatus.QUEUED,
            url=url,
            file_name=file_name,
            original_name=original_name or file_name,
            chat_id=chat_id,
            user_id=user_id,
            created_at=datetime.utcnow(),
            message_id=message_id,
            reply_message_id=reply_message_id,
            max_retries=self.max_retries,
            source_kind=source_kind,
            source_file_id=source_file_id,
            command_text=command_text,
        )
        task.work_dir = self.download_dir / task.id
        self.tasks[task.id] = task
        await self.repository.save_task(task)
        await self._enqueue(task)
        return task

    async def mark_status(self, task: RuntimeTask, status: TaskStatus, error: str = "") -> None:
        task.status = status
        if error:
            task.error = error
        await self.repository.save_task(task)

    async def update_progress(self, task: RuntimeTask) -> None:
        await self.repository.save_task(task)

    def get_task(self, task_id: str) -> RuntimeTask | None:
        return self.tasks.get(task_id)

    def get_active_tasks(self, chat_id: int | None = None) -> list[RuntimeTask]:
        active = [
            task
            for task in self.tasks.values()
            if task.status not in {TaskStatus.COMPLETED, TaskStatus.FAILED, TaskStatus.CANCELLED}
        ]
        if chat_id is not None:
            active = [task for task in active if task.chat_id == chat_id]
        return sorted(active, key=lambda item: item.created_at)

    async def cancel_task(self, task_id: str) -> bool:
        task = self.tasks.get(task_id)
        if task is None:
            return False
        if task.status in {TaskStatus.COMPLETED, TaskStatus.FAILED, TaskStatus.CANCELLED}:
            return False
        task.status = TaskStatus.CANCELLED
        task.error = "Cancelled by user"
        await self.repository.save_task(task)
        return True

    async def requeue_task(self, task: RuntimeTask, delay_seconds: int = 0) -> None:
        if delay_seconds > 0:
            await asyncio.sleep(delay_seconds)
        if task.status == TaskStatus.CANCELLED:
            return
        task.status = TaskStatus.QUEUED
        await self.repository.save_task(task)
        await self._enqueue(task)

    async def _enqueue(self, task: RuntimeTask) -> None:
        priority = 0 if task.user_id in self.privileged_users or task.user_id == self.owner_id else 10
        await self._queue.put((priority, task.created_at.timestamp(), task.id))

    async def _restore_active_tasks(self) -> None:
        active_tasks = await self.repository.get_active_tasks()
        for item in active_tasks:
            task = RuntimeTask(
                id=str(item["id"]),
                type=TaskType(str(item["type"])),
                status=TaskStatus.QUEUED,
                url=str(item["url"] or ""),
                file_name=str(item["file_name"] or ""),
                original_name=str(item["file_name"] or ""),
                chat_id=int(item["chat_id"]),
                user_id=int(item["user_id"]),
                created_at=_parse_datetime(item["created_at"]),
                local_path=str(item["local_path"] or ""),
                remote_path=str(item["remote_path"] or ""),
                remote_url=str(item["remote_url"] or ""),
                total_size=int(item["total_size"] or 0),
                downloaded_size=int(item["downloaded_size"] or 0),
                uploaded_size=int(item["uploaded_size"] or 0),
                error=str(item["error"] or ""),
                retries=int(item["retries"] or 0),
                quality=str(item["quality"] or ""),
                max_retries=self.max_retries,
            )
            task.work_dir = self.download_dir / task.id
            self.tasks[task.id] = task
            await self._enqueue(task)

    async def _worker(self, index: int) -> None:
        while not self._shutdown.is_set():
            try:
                _, _, task_id = await self._queue.get()
            except asyncio.CancelledError:
                return

            task = self.tasks.get(task_id)
            if task is None or task.status == TaskStatus.CANCELLED:
                self._queue.task_done()
                continue

            try:
                await self.processor(task)
            except asyncio.CancelledError:
                raise
            except Exception as exc:
                logger.exception("Task worker %s failed task %s", index, task.id)
                task.error = str(exc)
                task.status = TaskStatus.FAILED
                await self.repository.save_task(task)
            finally:
                self._queue.task_done()


def _parse_datetime(value: object) -> datetime:
    if isinstance(value, datetime):
        return value
    if not value:
        return datetime.utcnow()
    text = str(value)
    for fmt in ("%Y-%m-%d %H:%M:%S", "%Y-%m-%d %H:%M:%S.%f", "%Y-%m-%dT%H:%M:%S", "%Y-%m-%dT%H:%M:%S.%f"):
        try:
            return datetime.strptime(text, fmt)
        except ValueError:
            continue
    return datetime.utcnow()
