from __future__ import annotations

from typing import Any

from aiogram import Bot, Dispatcher
from aiogram.types import Update
from fastapi import FastAPI, HTTPException, Request, Response
from prometheus_client import CONTENT_TYPE_LATEST, Gauge, generate_latest

from zee_mirror.config import Settings
from zee_mirror.service import BotService


db_status_gauge = Gauge("zee_mirror_db_status", "Database connectivity status")


def create_app(settings: Settings, service: BotService, bot: Bot, dispatcher: Dispatcher) -> FastAPI:
    app = FastAPI(title="Zee-Mirror Python", version="0.1.0")

    @app.get("/api/health")
    async def health() -> dict[str, Any]:
        payload = await service.get_health_payload()
        db_status_gauge.set(1 if payload["db"] == "ok" else 0)
        return payload

    @app.get("/metrics")
    async def metrics() -> Response:
        return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)

    @app.post("/api/telegram/webhook")
    async def telegram_webhook(request: Request) -> dict[str, bool]:
        if not settings.use_webhook:
            raise HTTPException(status_code=404, detail="Webhook mode is disabled")

        if settings.webhook_secret:
            provided = request.headers.get("X-Telegram-Bot-Api-Secret-Token", "")
            if provided != settings.webhook_secret:
                raise HTTPException(status_code=403, detail="Invalid webhook secret")

        payload = await request.json()
        update = Update.model_validate(payload)
        await dispatcher.feed_update(bot, update)
        return {"ok": True}

    return app
