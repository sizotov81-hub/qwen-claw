#!/bin/bash
# Запуск Telegram бота через прокси

# Примеры прокси (раскомментируйте нужный):
# export HTTPS_PROXY="http://proxy-server:port"
# export HTTPS_PROXY="socks5://proxy-server:port"
# export TELEGRAM_PROXY="http://127.0.0.1:8080"

# Для Tor:
# export HTTPS_PROXY="socks5://127.0.0.1:9050"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🤖 Qwen-Claw Telegram Bot"
echo "========================="
echo ""

if [ -n "$HTTPS_PROXY" ] || [ -n "$TELEGRAM_PROXY" ]; then
    echo "✅ Proxy configured:"
    echo "   HTTPS_PROXY=${HTTPS_PROXY:-not set}"
    echo "   TELEGRAM_PROXY=${TELEGRAM_PROXY:-not set}"
    echo ""
else
    echo "⚠️  No proxy configured!"
    echo ""
    echo "If Telegram is blocked, set proxy:"
    echo "  export HTTPS_PROXY=\"http://proxy:port\""
    echo "  Or use Tor: export HTTPS_PROXY=\"socks5://127.0.0.1:9050\""
    echo ""
fi

echo "Starting bot..."
echo "Press Ctrl+C to stop"
echo ""

cd "$SCRIPT_DIR"
exec ./qwen-claw telegram "$@"
