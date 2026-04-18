from __future__ import annotations

import asyncio
import logging
from contextlib import suppress
from pathlib import Path

from dotenv import load_dotenv
import uvicorn

from zee_mirror.api import create_app
from zee_mirror.bot import create_bot, create_dispatcher
from zee_mirror.config import Settings
from zee_mirror.database import Database
from zee_mirror.migrations import apply_migrations
from zee_mirror.repository import Repository
from zee_mirror.service import BotService


def run() -> None:
    asyncio.run(main())


async def main() -> None:
    load_dotenv()
    settings = Settings()
    configure_logging(settings.log_level)

    migrations_dir = Path(__file__).resolve().parents[2] / "migrations"
    db_path = settings.config_path / "zee-mirror.db"
    apply_migrations(db_path, migrations_dir)

    db = Database(db_path)
    await db.connect()

    bot = create_bot(settings)
    repository = Repository(db)
    service = BotService(settings, db, repository, bot)
    await service.startup()

    dispatcher = create_dispatcher(service)
    app = create_app(settings, service, bot, dispatcher)
    server = uvicorn.Server(
        uvicorn.Config(app, host="0.0.0.0", port=settings.api_port, log_level=settings.log_level.lower())
    )

    polling_task: asyncio.Task[None] | None = None

    try:
        if settings.use_webhook and settings.webhook_url:
            webhook_url = settings.webhook_url.rstrip("/") + "/api/telegram/webhook"
            await bot.set_webhook(webhook_url, secret_token=settings.webhook_secret or None)
            await server.serve()
        else:
            polling_task = asyncio.create_task(
                dispatcher.start_polling(bot, allowed_updates=dispatcher.resolve_used_update_types())
            )
            await server.serve()
    finally:
        if polling_task is not None:
            polling_task.cancel()
            with suppress(asyncio.CancelledError):
                await polling_task
        if settings.use_webhook:
            with suppress(Exception):
                await bot.delete_webhook(drop_pending_updates=False)
        await bot.session.close()
        await db.close()


def configure_logging(log_level: str) -> None:
    level = getattr(logging, log_level.upper(), logging.INFO)
    logging.basicConfig(
        level=level,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )


if __name__ == "__main__":
    run()
