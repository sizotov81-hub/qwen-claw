#!/bin/bash
# Запуск Qwen Wrapper Service

set -e

GRPC_PORT="${GRPC_PORT:-55052}"
HTTP_PORT="${HTTP_PORT:-58082}"
METRICS_PORT="${METRICS_PORT:-59092}"
HOST="${HOST:-0.0.0.0}"
LOG_LEVEL="${LOG_LEVEL:-info}"
LOG_FORMAT="${LOG_FORMAT:-json}"
QWEN_PATH="${QWEN_PATH:-qwen}"

BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/bin"

echo "🚀 Запуск Qwen Wrapper Service"
echo "=================================="
echo ""
echo "gRPC Port:    $GRPC_PORT"
echo "HTTP Port:    $HTTP_PORT"
echo "Metrics Port: $METRICS_PORT"
echo "Qwen Path:    $QWEN_PATH"
echo "Host:         $HOST"
echo "Log Level:    $LOG_LEVEL"
echo "Log Format:   $LOG_FORMAT"
echo ""
echo "Нажмите Ctrl+C для остановки"
echo ""

exec "$BIN_DIR/qwen-wrapper" \
    --grpc-port="$GRPC_PORT" \
    --http-port="$HTTP_PORT" \
    --host="$HOST" \
    --log-level="$LOG_LEVEL" \
    --log-format="$LOG_FORMAT" \
    --metrics-port="$METRICS_PORT" \
    --qwen-path="$QWEN_PATH"
