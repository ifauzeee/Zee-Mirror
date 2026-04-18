from __future__ import annotations

from functools import cached_property
from pathlib import Path

from pydantic import Field, field_validator, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    bot_token: str = Field(alias="BOT_TOKEN")
    owner_id: int = Field(alias="OWNER_ID")
    authorized_users: list[int] = Field(default_factory=list, alias="AUTHORIZED_USERS")
    telegram_api: str = Field(default="", alias="TELEGRAM_API")
    telegram_api_id: int = Field(default=0, alias="TELEGRAM_API_ID")
    telegram_api_hash: str = Field(default="", alias="TELEGRAM_API_HASH")
    app_id: int = Field(default=0, alias="APP_ID")
    app_hash: str = Field(default="", alias="APP_HASH")
    user_session_string: str = Field(default="", alias="USER_SESSION_STRING")
    api_port: int = Field(default=8080, alias="API_PORT")
    use_webhook: bool = Field(default=False, alias="USE_WEBHOOK")
    webhook_url: str = Field(default="", alias="WEBHOOK_URL")
    webhook_secret: str = Field(default="", alias="WEBHOOK_SECRET")
    rclone_dest: str = Field(default="gdrive:/MirrorBot", alias="RCLONE_DEST")
    download_dir: str = Field(default="/app/downloads", alias="DOWNLOAD_DIR")
    config_dir: str = Field(default="/app/config", alias="CONFIG_DIR")
    log_level: str = Field(default="info", alias="LOG_LEVEL")
    max_concurrent_downloads: int = Field(default=3, alias="MAX_CONCURRENT_DOWNLOADS")
    default_max_daily_tasks: int = Field(default=-1, alias="DEFAULT_MAX_DAILY_TASKS")
    default_max_daily_bandwidth: str = Field(default="-1", alias="DEFAULT_MAX_DAILY_BANDWIDTH")
    smart_auto_organization: bool = Field(default=False, alias="SMART_AUTO_ORGANIZATION")
    stop_duplicate: bool = Field(default=False, alias="STOP_DUPLICATE")
    index_url: str = Field(default="", alias="INDEX_URL")
    aria2_rpc_url: str = Field(default="http://localhost:6800/jsonrpc", alias="ARIA2_RPC_URL")
    aria2_rpc_secret: str = Field(default="", alias="ARIA2_RPC_SECRET")

    @field_validator("authorized_users", mode="before")
    @classmethod
    def parse_authorized_users(cls, value: object) -> list[int]:
        if value in (None, "", []):
            return []
        if isinstance(value, list):
            return [int(item) for item in value]
        if isinstance(value, str):
            return [int(item.strip()) for item in value.split(",") if item.strip()]
        raise TypeError("AUTHORIZED_USERS must be a comma-separated string or list[int]")

    @model_validator(mode="after")
    def validate_required_fields(self) -> "Settings":
        if not self.bot_token:
            raise ValueError("BOT_TOKEN is required")
        if not self.owner_id:
            raise ValueError("OWNER_ID is required")
        return self

    @cached_property
    def telegram_api_base(self) -> str:
        if not self.telegram_api:
            return ""
        base = self.telegram_api
        for suffix in ("/bot%s/%s", "/bot{token}/{method}"):
            if base.endswith(suffix):
                return base[: -len(suffix)]
        return base.rstrip("/")

    @cached_property
    def effective_app_id(self) -> int:
        return self.app_id or self.telegram_api_id

    @cached_property
    def effective_app_hash(self) -> str:
        return self.app_hash or self.telegram_api_hash

    @cached_property
    def config_path(self) -> Path:
        return Path(self.config_dir)

    @cached_property
    def download_path(self) -> Path:
        return Path(self.download_dir)
