from __future__ import annotations

import os
from pathlib import Path

from dotenv import load_dotenv
from telethon import TelegramClient
from telethon.sessions import StringSession


def main() -> None:
    load_dotenv()

    app_id = input(f"Enter API ID [{os.getenv('APP_ID') or os.getenv('TELEGRAM_API_ID') or ''}]: ").strip()
    if not app_id:
        app_id = os.getenv("APP_ID") or os.getenv("TELEGRAM_API_ID") or ""

    app_hash = input(f"Enter API HASH [{os.getenv('APP_HASH') or os.getenv('TELEGRAM_API_HASH') or ''}]: ").strip()
    if not app_hash:
        app_hash = os.getenv("APP_HASH") or os.getenv("TELEGRAM_API_HASH") or ""

    if not app_id or not app_hash:
        raise SystemExit("APP_ID and APP_HASH are required")

    with TelegramClient(StringSession(), int(app_id), app_hash) as client:
        session_string = client.session.save()
        _update_env_file(
            Path(".env"),
            {
                "APP_ID": str(app_id),
                "APP_HASH": app_hash,
                "USER_SESSION_STRING": session_string,
            },
        )
        print("USER_SESSION_STRING generated and stored in .env")


def _update_env_file(env_path: Path, updates: dict[str, str]) -> None:
    lines: list[str] = []
    existing = env_path.read_text(encoding="utf-8").splitlines() if env_path.exists() else []
    touched: set[str] = set()

    for line in existing:
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or "=" not in stripped:
            lines.append(line)
            continue

        key, _ = stripped.split("=", 1)
        if key in updates:
            lines.append(f"{key}={updates[key]}")
            touched.add(key)
        else:
            lines.append(line)

    for key, value in updates.items():
        if key not in touched:
            lines.append(f"{key}={value}")

    env_path.write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")
