#!/bin/bash
# Запуск LLM Proxy Service

set -e

GRPC_PORT="${GRPC_PORT:-55053}"
HTTP_PORT="${HTTP_PORT:-58083}"
METRICS_PORT="${METRICS_PORT:-59093}"
HOST="${HOST:-0.0.0.0}"
LOG_LEVEL="${LOG_LEVEL:-info}"
LOG_FORMAT="${LOG_FORMAT:-json}"
OPENAI_KEY="${OPENAI_KEY:-}"
ANTHROPIC_KEY="${ANTHROPIC_KEY:-}"
DASHSCOPE_KEY="${DASHSCOPE_KEY:-}"

BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/bin"

echo "🚀 Запуск LLM Proxy Service"
echo "=================================="
echo ""
echo "gRPC Port:       $GRPC_PORT"
echo "HTTP Port:       $HTTP_PORT"
echo "Metrics Port:    $METRICS_PORT"
echo "Host:            $HOST"
echo "OpenAI Key:      ${OPENAI_KEY:-<not set>}"
echo "Anthropic Key:   ${ANTHROPIC_KEY:-<not set>}"
echo "DashScope Key:   ${DASHSCOPE_KEY:-<not set>}"
echo "Log Level:       $LOG_LEVEL"
echo "Log Format:      $LOG_FORMAT"
echo ""
echo "Нажмите Ctrl+C для остановки"
echo ""

exec "$BIN_DIR/llm-proxy" \
    --grpc-port="$GRPC_PORT" \
    --http-port="$HTTP_PORT" \
    --host="$HOST" \
    --log-level="$LOG_LEVEL" \
    --log-format="$LOG_FORMAT" \
    --metrics-port="$METRICS_PORT" \
    --openai-key="$OPENAI_KEY" \
    --anthropic-key="$ANTHROPIC_KEY" \
    --dashscope-key="$DASHSCOPE_KEY"
