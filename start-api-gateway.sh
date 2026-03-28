#!/bin/bash
# Запуск API Gateway Service

set -e

GRPC_PORT="${GRPC_PORT:-55050}"
HTTP_PORT="${HTTP_PORT:-58080}"
METRICS_PORT="${METRICS_PORT:-59090}"
HOST="${HOST:-0.0.0.0}"
LOG_LEVEL="${LOG_LEVEL:-info}"
LOG_FORMAT="${LOG_FORMAT:-json}"

BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/bin"

echo "🚀 Запуск API Gateway Service"
echo "=================================="
echo ""
echo "gRPC Port:    $GRPC_PORT"
echo "HTTP Port:    $HTTP_PORT"
echo "Metrics Port: $METRICS_PORT"
echo "Host:         $HOST"
echo "Log Level:    $LOG_LEVEL"
echo "Log Format:   $LOG_FORMAT"
echo ""
echo "Нажмите Ctrl+C для остановки"
echo ""

exec "$BIN_DIR/api-gateway" \
    --grpc-port="$GRPC_PORT" \
    --http-port="$HTTP_PORT" \
    --host="$HOST" \
    --log-level="$LOG_LEVEL" \
    --log-format="$LOG_FORMAT" \
    --metrics-port="$METRICS_PORT"
