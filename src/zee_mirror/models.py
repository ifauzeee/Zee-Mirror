from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from enum import StrEnum


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
