#!/bin/bash
# Запуск веб-интерфейса Qwen-Claw

set -e

HOST="${1:-127.0.0.1}"
PORT="${2:-64656}"
STATIC_DIR="./internal/web/static"

echo "🌐 Запуск веб-интерфейса Qwen-Claw"
echo "=================================="
echo ""
echo "URL: http://$HOST:$PORT"
echo "Static dir: $STATIC_DIR"
echo ""
echo "Нажмите Ctrl+C для остановки"
echo ""

# Проверяем наличие статических файлов
if [ ! -d "$STATIC_DIR" ]; then
    echo "❌ Директория со статикой не найдена: $STATIC_DIR"
    exit 1
fi

# Запускаем простой HTTP сервер
if command -v python3 &> /dev/null; then
    echo "🐍 Используем Python HTTP сервер..."
    cd "$STATIC_DIR"
    python3 -m http.server "$PORT" --bind "$HOST"
elif command -v python &> /dev/null; then
    echo "🐍 Используем Python HTTP сервер..."
    cd "$STATIC_DIR"
    python -m SimpleHTTPServer "$PORT"
elif command -v npx &> /dev/null; then
    echo "📦 Используем npx http-server..."
    npx http-server "$STATIC_DIR" -p "$PORT" -a "$HOST"
else
    echo "❌ Не найден HTTP сервер. Установите Python или Node.js"
    echo ""
    echo "Варианты:"
    echo "  1. Установить Python: sudo apt install python3"
    echo "  2. Установить Node.js: sudo apt install nodejs npm"
    echo "  3. Использовать Go: go run cmd/webserver/main.go"
    exit 1
fi
