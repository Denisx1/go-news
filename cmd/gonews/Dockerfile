# syntax=docker/dockerfile:1.7

# Версия Go (посмотрите `go version` у себя или строку "go ..." в go.mod).
# Если в go.mod версия выше, Go сам докачает нужный toolchain.
ARG GO_VERSION=1.24

# ───────── этап 1: сборка ─────────
FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src
ENV CGO_ENABLED=0

# сначала зависимости: этот слой кэшируется, пока go.mod/go.sum не менялись
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/gonews ./cmd/gonews

# ───────── этап 2: запуск ─────────
FROM debian:bookworm-slim

# Chromium нужен только для источников с UseBrowser: true (сейчас MVS).
# Если такой источник убрать, chromium и fonts-liberation можно удалить:
# образ станет в разы меньше.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    chromium fonts-liberation ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

# не root: у Chrome и у сервиса нет лишних прав
RUN useradd --system --uid 1000 --create-home --home-dir /home/app app \
    && mkdir -p /app/data \
    && chown -R app:app /app

WORKDIR /app
COPY --from=build --chown=app:app /out/gonews /app/gonews

ENV TZ=Europe/Kyiv \
    HOME=/home/app \
    DATA_DIR=/app/data

USER app
CMD ["/app/gonews"]