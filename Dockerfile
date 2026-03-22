# Qwen-Claw Docker Image
# Многоэтапная сборка для минимизации размера образа

# ============================================
# Этап 1: Сборка Go приложения
# ============================================
FROM golang:1.21-alpine AS builder

# Устанавливаем зависимости для сборки
RUN apk add --no-cache git ca-certificates

# Устанавливаем рабочую директорию
WORKDIR /build

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY bots/ ./bots/
COPY skills/ ./skills/

# Собираем приложение с оптимизациями
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /build/qwen-claw \
    ./cmd/main.go

# Копируем файлы Web UI для сборки
COPY internal/web/ /build/internal/web/

# ============================================
# Этап 2: Финальный образ
# ============================================
FROM alpine:3.19

# Устанавливаем runtime зависимости
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    git \
    openssh-client \
    nodejs \
    npm \
    && npm install -g qwen-code-cli 2>/dev/null || true

# Создаём пользователя для безопасности
RUN addgroup -g 1000 qwenclaw && \
    adduser -D -u 1000 -G qwenclaw qwenclaw

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем бинарник из builder
COPY --from=builder /build/qwen-claw /app/qwen-claw

# Копируем статические файлы Web UI
COPY --from=builder /build/internal/web/static /app/internal/web/static

# Копируем примеры конфигурации
COPY config.yaml.example /app/config.yaml.example
COPY .env.example /app/.env.example

# Создаём директорию для данных
RUN mkdir -p /app/.qwen/memory /app/.qwen/scheduler && \
    chown -R qwenclaw:qwenclaw /app

# Копируем health check скрипт
COPY --chown=qwenclaw:qwenclaw healthcheck.sh /app/healthcheck.sh
RUN chmod +x /app/healthcheck.sh

# Переключаемся на пользователя без root
USER qwenclaw

# Экспортируем порт (для будущего Web UI)
EXPOSE 18789

# Переменные окружения
ENV QWEN_CLAW_BASE_DIR=/app
ENV QWEN_CLAW_QWEN_DIR=/app/.qwen
ENV QWEN_CLAW_MEMORY_DIR=/app/.qwen/memory
ENV TZ=UTC

# Health check через HTTP endpoint
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD /app/healthcheck.sh

# Запуск приложения
ENTRYPOINT ["/app/qwen-claw"]

# Команда по умолчанию - запуск Telegram бота
CMD ["telegram"]
