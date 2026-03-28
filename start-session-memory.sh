#!/bin/bash
# Запуск Session Memory Service

set -e

GRPC_PORT="${GRPC_PORT:-55051}"
HTTP_PORT="${HTTP_PORT:-58081}"
METRICS_PORT="${METRICS_PORT:-59091}"
HOST="${HOST:-0.0.0.0}"
LOG_LEVEL="${LOG_LEVEL:-info}"
LOG_FORMAT="${LOG_FORMAT:-json}"
DATABASE_URL="${DATABASE_URL:-}"
REDIS_URL="${REDIS_URL:-}"

BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/bin"

echo "🚀 Запуск Session Memory Service"
echo "=================================="
echo ""
echo "gRPC Port:    $GRPC_PORT"
echo "HTTP Port:    $HTTP_PORT"
echo "Metrics Port: $METRICS_PORT"
echo "Host:         $HOST"
echo "Database URL: ${DATABASE_URL:-<not set>}"
echo "Redis URL:    ${REDIS_URL:-<not set>}"
echo "Log Level:    $LOG_LEVEL"
echo "Log Format:   $LOG_FORMAT"
echo ""
echo "Нажмите Ctrl+C для остановки"
echo ""

exec "$BIN_DIR/session-memory" \
    --grpc-port="$GRPC_PORT" \
    --http-port="$HTTP_PORT" \
    --host="$HOST" \
    --log-level="$LOG_LEVEL" \
    --log-format="$LOG_FORMAT" \
    --metrics-port="$METRICS_PORT" \
    --database-url="$DATABASE_URL" \
    --redis-url="$REDIS_URL"
