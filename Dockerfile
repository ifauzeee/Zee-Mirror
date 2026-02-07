
FROM node:18-alpine AS dashboard-builder
WORKDIR /dashboard
COPY dashboard/package*.json ./
RUN npm install
COPY dashboard/ ./
RUN npm run build


FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o zee-mirror ./cmd/zee-mirror


FROM alpine:3.20
LABEL maintainer="Zee-Mirror Bot"
LABEL description="Telegram Mirror/Leech Bot"

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    aria2 \
    ffmpeg \
    python3 \
    py3-pip \
    p7zip \
    curl \
    unzip \
    nodejs \
    py3-cryptography \
    py3-pycryptodomex \
    gcc \
    musl-dev \
    python3-dev \
    coreutils \
    expect

# Menggunakan --no-cache-dir dan memastikan download versi terbaru setiap build
RUN pip3 install --break-system-packages --no-cache-dir -U yt-dlp speedtest-cli


COPY --from=rclone/rclone:latest /usr/local/bin/rclone /usr/bin/rclone
RUN chmod 755 /usr/bin/rclone

RUN addgroup -S botgroup && adduser -D -G botgroup botuser
RUN mkdir -p /app/downloads /app/config /home/botuser/.cache/yt-dlp && \
    chown -R botuser:botgroup /app /home/botuser

WORKDIR /app
COPY --from=builder /build/zee-mirror /app/zee-mirror
COPY --from=dashboard-builder /dist /app/dist
RUN chmod +x /app/zee-mirror

USER root

ENV BOT_TOKEN="" \
    OWNER_ID="" \
    AUTHORIZED_USERS="" \
    RCLONE_DEST="gdrive:/MirrorBot" \
    MAX_CONCURRENT_DOWNLOADS="3" \
    DOWNLOAD_DIR="/app/downloads" \
    CONFIG_DIR="/app/config" \
    HOME="/home/botuser" \
    PATH="/usr/local/bin:/usr/bin:/bin:/home/botuser/.local/bin" \
    WEB_DASHBOARD_TOKEN="zee-mirror-secret"

EXPOSE 8080

VOLUME ["/app/downloads", "/app/config"]

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

CMD ["/app/zee-mirror"]