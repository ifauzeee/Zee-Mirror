from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from enum import StrEnum
from pathlib import Path


class TaskStatus(StrEnum):
    QUEUED = "queued"
    DOWNLOADING = "downloading"
    FETCHING = "fetching"
    EXTRACTING = "extracting"
    ZIPPING = "zipping"
    UPLOADING = "uploading"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"
    PROCESSING = "processing"


class TaskType(StrEnum):
    MIRROR = "mirror"
    LEECH = "leech"
    YTDLP = "ytdlp"
    YTDLP_LEECH = "ytdlp_leech"
    TORRENT = "torrent"
    CLONE = "clone"
    VIKING = "viking"
    PLAYLIST = "playlist"


@dataclass(slots=True)
class User:
    id: int
    username: str = ""
    role: str = "user"
    language: str = "id"
    max_daily_tasks: int = -1
    max_daily_bandwidth: int = -1
    created_at: datetime | None = None
    expires_at: datetime | None = None


@dataclass(slots=True)
class TaskRecord:
    id: str
    type: str
    status: str
    chat_id: int
    user_id: int
    created_at: datetime
    gid: str = ""
    url: str = ""
    file_name: str = ""
    local_path: str = ""
    remote_path: str = ""
    remote_url: str = ""
    total_size: int = 0
    downloaded_size: int = 0
    uploaded_size: int = 0
    completed_at: datetime | None = None
    zip: bool = False
    unzip: bool = False
    password: str = ""
    error: str = ""
    retries: int = 0
    quality: str = ""


@dataclass(slots=True)
class TaskCheckpoint:
    task_id: str
    downloaded_bytes: int
    total_bytes: int
    progress: float
    last_update: int


@dataclass(slots=True)
class RuntimeTask:
    id: str
    type: TaskType
    status: TaskStatus
    url: str
    chat_id: int
    user_id: int
    created_at: datetime
    file_name: str = ""
    original_name: str = ""
    quality: str = ""
    message_id: int = 0
    reply_message_id: int = 0
    total_size: int = 0
    downloaded_size: int = 0
    uploaded_size: int = 0
    progress: float = 0.0
    speed: int = 0
    remote_path: str = ""
    remote_url: str = ""
    local_path: str = ""
    error: str = ""
    gid: str = ""
    retries: int = 0
    max_retries: int = 3
    source_file_id: str = ""
    source_kind: str = "url"
    result_message_id: int = 0
    work_dir: Path | None = None
    command_text: str = ""

    def to_record(self) -> TaskRecord:
        return TaskRecord(
            id=self.id,
            gid=self.gid,
            type=self.type.value,
            status=self.status.value,
            url=self.url,
            file_name=self.file_name,
            local_path=self.local_path,
            remote_path=self.remote_path,
            remote_url=self.remote_url,
            total_size=self.total_size,
            downloaded_size=self.downloaded_size,
            uploaded_size=self.uploaded_size,
            chat_id=self.chat_id,
            user_id=self.user_id,
            created_at=self.created_at,
            completed_at=datetime.utcnow()
            if self.status in {TaskStatus.COMPLETED, TaskStatus.FAILED, TaskStatus.CANCELLED}
            else None,
            error=self.error,
            retries=self.retries,
            quality=self.quality,
        )
