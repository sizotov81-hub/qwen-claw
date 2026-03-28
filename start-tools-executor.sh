#!/bin/bash
# Запуск Tools Executor Service

set -e

GRPC_PORT="${GRPC_PORT:-55054}"
HTTP_PORT="${HTTP_PORT:-58084}"
METRICS_PORT="${METRICS_PORT:-59094}"
HOST="${HOST:-0.0.0.0}"
LOG_LEVEL="${LOG_LEVEL:-info}"
LOG_FORMAT="${LOG_FORMAT:-json}"
SANDBOX_MODE="${SANDBOX_MODE:-docker}"

BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/bin"

echo "🚀 Запуск Tools Executor Service"
echo "=================================="
echo ""
echo "gRPC Port:     $GRPC_PORT"
echo "HTTP Port:     $HTTP_PORT"
echo "Metrics Port:  $METRICS_PORT"
echo "Sandbox Mode:  $SANDBOX_MODE"
echo "Host:          $HOST"
echo "Log Level:     $LOG_LEVEL"
echo "Log Format:    $LOG_FORMAT"
echo ""
echo "Нажмите Ctrl+C для остановки"
echo ""

exec "$BIN_DIR/tools-executor" \
    --grpc-port="$GRPC_PORT" \
    --http-port="$HTTP_PORT" \
    --host="$HOST" \
    --log-level="$LOG_LEVEL" \
    --log-format="$LOG_FORMAT" \
    --metrics-port="$METRICS_PORT" \
    --sandbox-mode="$SANDBOX_MODE"
