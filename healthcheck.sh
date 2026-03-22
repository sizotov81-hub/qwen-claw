#!/bin/bash
# Docker health check для Qwen-Claw

set -e

# Проверяем что процесс запущен
if ! pgrep -x "qwen-claw" > /dev/null; then
    echo "❌ qwen-claw process not running"
    exit 1
fi

# Проверяем HTTP health endpoint
HEALTH_URL="http://localhost:18789/api/health"

response=$(curl -s -o /dev/null -w "%{http_code}" "$HEALTH_URL" 2>/dev/null || echo "000")

if [ "$response" = "200" ]; then
    echo "✅ qwen-claw is healthy (HTTP $response)"
    exit 0
elif [ "$response" = "000" ]; then
    echo "⚠️  qwen-claw not responding (connection failed)"
    exit 1
else
    echo "❌ qwen-claw unhealthy (HTTP $response)"
    exit 1
fi
