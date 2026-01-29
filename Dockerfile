# Zee-Mirror Telegram Bot
# Multi-stage Dockerfile untuk build yang optimal

# ========== Stage 1: Build ==========
FROM golang:1.22-alpine AS builder

# Install dependencies untuk build
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files terlebih dahulu untuk cache dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build aplikasi dengan optimisasi
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o zee-mirror .

# ========== Stage 2: Runtime ==========
FROM alpine:3.19

LABEL maintainer="Zee-Mirror Bot"
LABEL description="Telegram Mirror/Leech Bot"

# Install runtime dependencies
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
    && pip3 install --break-system-packages --no-cache-dir yt-dlp \
    && curl -O https://downloads.rclone.org/rclone-current-linux-amd64.zip \
    && unzip rclone-current-linux-amd64.zip \
    && cp rclone-*-linux-amd64/rclone /usr/bin/ \
    && chmod 755 /usr/bin/rclone \
    && rm -rf rclone-*

# Create non-root user
RUN addgroup -S botgroup && adduser -S botuser -G botgroup

# Create directories
RUN mkdir -p /app/downloads /app/config && \
    chown -R botuser:botgroup /app

WORKDIR /app

# Copy binary dari builder
COPY --from=builder /build/zee-mirror /app/zee-mirror

# Set permissions
RUN chmod +x /app/zee-mirror

# Switch to non-root user
USER botuser

# Environment variables dengan default values
ENV BOT_TOKEN=""
ENV OWNER_ID=""
ENV AUTHORIZED_USERS=""
ENV RCLONE_DEST="gdrive:/MirrorBot"
ENV MAX_CONCURRENT_DOWNLOADS="3"
ENV DOWNLOAD_DIR="/app/downloads"
ENV CONFIG_DIR="/app/config"

# Volumes
VOLUME ["/app/downloads", "/app/config"]

# Healthcheck
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep -x zee-mirror || exit 1

# Run bot
CMD ["/app/zee-mirror"]
