# Zee-Mirror Python Rewrite

This directory set introduces a Python rewrite foundation for Zee-Mirror.

Current scope:
- FastAPI application with `/api/health`, `/metrics`, and Telegram webhook endpoint
- aiogram bot bootstrap with basic commands (`/start`, `/help`, `/ping`, `/health`)
- SQLite compatibility layer that reuses the existing SQL migrations in `migrations/`
- Telethon-based session generator to replace the old Go helper over time
- Docker and compose files for running the Python application in parallel with the Go codebase during migration

This is intentionally a safe migration base, not yet a full feature-parity replacement for:
- aria2 task orchestration
- rclone upload flow
- media processing pipeline
- torrent selection UI
- userbot download flow
- advanced dashboard and analytics handlers

Recommended migration path:
1. Run the Python app against the existing SQLite database and environment variables.
2. Port task manager, downloader, uploader, and media services one module at a time.
3. Switch the primary deployment from Go to Python after command and background-task parity is reached.
