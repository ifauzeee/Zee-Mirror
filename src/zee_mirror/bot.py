from __future__ import annotations

from aiogram import Bot, Dispatcher, F, Router
from aiogram.client.default import DefaultBotProperties
from aiogram.client.session.aiohttp import AiohttpSession
from aiogram.client.telegram import TelegramAPIServer
from aiogram.enums import ParseMode
from aiogram.filters import Command
from aiogram.types import Message

from zee_mirror.config import Settings
from zee_mirror.service import BotService


def create_bot(settings: Settings) -> Bot:
    session = None
    if settings.telegram_api_base:
        api_server = TelegramAPIServer.from_base(settings.telegram_api_base, is_local=True)
        session = AiohttpSession(api=api_server)

    return Bot(
        token=settings.bot_token,
        session=session,
        default=DefaultBotProperties(parse_mode=ParseMode.HTML),
    )


def create_dispatcher(service: BotService) -> Dispatcher:
    dispatcher = Dispatcher()
    dispatcher.include_router(_build_router(service))
    return dispatcher


def _build_router(service: BotService) -> Router:
    router = Router()

    @router.message(Command("start"))
    async def start_handler(message: Message) -> None:
        await service.sync_user(message.from_user)
        text = (
            "<b>Zee-Mirror Python Rewrite</b>\n"
            "Fondasi Python sudah aktif.\n\n"
            "Command dasar tersedia:\n"
            "/help - lihat command\n"
            "/ping - cek respons bot\n"
            "/health - cek kesehatan service"
        )
        await message.answer(text)

    @router.message(Command("help"))
    async def help_handler(message: Message) -> None:
        await service.sync_user(message.from_user)
        text = (
            "<b>Available commands</b>\n"
            "/start - mulai bot\n"
            "/help - daftar command\n"
            "/ping - bot responsiveness\n"
            "/health - runtime health\n\n"
            "Task pipeline lanjutan masih dalam proses porting dari Go ke Python."
        )
        await message.answer(text)

    @router.message(Command("ping"))
    async def ping_handler(message: Message) -> None:
        if not await service.is_authorized(message.from_user.id if message.from_user else None):
            await message.answer("Access denied. Hubungi owner untuk mendapatkan akses.")
            return
        await service.sync_user(message.from_user)
        await message.answer("pong")

    @router.message(Command("health"))
    async def health_handler(message: Message) -> None:
        if not await service.is_authorized(message.from_user.id if message.from_user else None):
            await message.answer("Access denied. Hubungi owner untuk mendapatkan akses.")
            return
        payload = await service.get_health_payload()
        text = (
            "<b>Health</b>\n"
            f"status: {payload['status']}\n"
            f"db: {payload['db']}\n"
            f"webhook_mode: {payload['webhook_mode']}\n"
        )
        if "aria2_version" in payload:
            text += f"aria2_version: {payload['aria2_version']}\n"
        await message.answer(text)

    @router.message(F.text)
    async def fallback_handler(message: Message) -> None:
        if not await service.is_authorized(message.from_user.id if message.from_user else None):
            await message.answer("Akses belum tersedia untuk akun ini.")
            return
        await service.sync_user(message.from_user)
        await message.answer(
            "Command dan task handler lanjutan masih sedang dipindahkan ke Python. "
            "Fondasi bot, API, dan database compatibility sudah siap."
        )

    return router
