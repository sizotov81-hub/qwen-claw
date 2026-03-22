#!/bin/bash

# Тестовый скрипт для проверки WebSocket Gateway

GATEWAY_URL="ws://127.0.0.1:18789/ws"
AUTH_TOKEN=""

echo "🧪 Qwen-Claw Gateway Test Script"
echo "================================="
echo ""

# Проверяем, запущен ли Gateway
echo "1. Проверка Gateway..."
response=$(curl -s http://127.0.0.1:18789/health 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "✅ Gateway доступен"
    echo "   Ответ: $response"
else
    echo "❌ Gateway недоступен"
    echo "   Запустите: ./qwen-claw gateway"
    exit 1
fi

# Извлекаем токен (если есть в ответе)
# AUTH_TOKEN=$(echo $response | jq -r '.auth_token' 2>/dev/null)

echo ""
echo "2. Проверка WebSocket подключения..."

# Проверяем наличие websocket клиента
if command -v websocat &> /dev/null; then
    echo "✅ websocat найден"
    
    # Тестовое подключение
    echo "   Тестовое подключение к $GATEWAY_URL..."
    
    # Отправляем сообщение аутентификации
    echo '{"type":"auth","payload":{"client_id":"test_client","client_type":"cli","token":"test"}}' | \
        websocat -n1 -q1 $GATEWAY_URL 2>&1 | head -5
    
    if [ $? -eq 0 ]; then
        echo "✅ WebSocket подключение успешно"
    else
        echo "⚠️  WebSocket подключение не удалось (это нормально если токен неверный)"
    fi
else
    echo "⚠️  websocat не найден"
    echo "   Установите: cargo install websocat"
    echo "   Или используйте другой WebSocket клиент"
fi

echo ""
echo "3. Проверка API endpoints..."

# Проверка списка сессий
echo "   Sessions API:"
curl -s http://127.0.0.1:18789/api/v1/sessions 2>/dev/null | head -1 || echo "   ❌ Недоступно"

# Проверка списка клиентов
echo ""
echo "   Clients API:"
curl -s http://127.0.0.1:18789/api/v1/clients 2>/dev/null | head -1 || echo "   ❌ Недоступно"

echo ""
echo "4. Тестовые сценарии:"
echo ""
echo "   а) Запустить Gateway:"
echo "      ./qwen-claw gateway --port 18789"
echo ""
echo "   б) Подключиться через WebSocket:"
echo "      websocat ws://127.0.0.1:18789/ws"
echo ""
echo "   в) Отправить сообщение аутентификации:"
echo '      {"type":"auth","payload":{"client_id":"test","client_type":"cli"}}'
echo ""
echo "   г) Отправить сообщение чата:"
echo '      {"type":"chat","session_id":"test","payload":{"content":"Привет!"}}'
echo ""

echo "================================="
echo "Тестирование завершено!"
