# 🔌 Qwen-Claw Gateway

**WebSocket сервер для управления сессиями и клиентами**

---

## 📋 Обзор

Gateway — это центральный WebSocket сервер для координации всех клиентов Qwen-Claw:
- CLI клиенты
- Telegram боты
- Web UI
- Мобильные приложения

---

## 🚀 Быстрый старт

### Запуск Gateway

```bash
# Базовый запуск
./qwen-claw gateway

# С кастомными параметрами
./qwen-claw gateway --port 18789 --bind 0.0.0.0 --debug

# В фоновом режиме
./qwen-claw gateway &
```

### Проверка статуса

```bash
# Health check
curl http://127.0.0.1:18789/health

# Список сессий
curl http://127.0.0.1:18789/api/v1/sessions

# Список клиентов
curl http://127.0.0.1:18789/api/v1/clients
```

---

## 🔌 WebSocket API

### Подключение

```
ws://127.0.0.1:18789/ws
```

### Формат сообщений

Все сообщения — JSON следующего формата:

```json
{
  "type": "<message_type>",
  "id": "<unique_id>",
  "session_id": "<session_id>",
  "payload": {},
  "timestamp": "2026-03-22T12:00:00Z"
}
```

### Типы сообщений

#### Клиент → Сервер

| Тип | Описание | Пример payload |
|-----|----------|---------------|
| `auth` | Аутентификация | `{"client_id": "cli", "client_type": "cli", "token": "xxx"}` |
| `chat` | Сообщение чата | `{"content": "Привет!", "model": "gpt-4"}` |
| `command` | Команда | `{"command": "run", "args": {"code": "ls -la"}}` |
| `subscribe` | Подписка на канал | `{"channel": "sessions"}` |
| `unsubscribe` | Отписка от канала | `{"channel": "sessions"}` |

#### Сервер → Клиент

| Тип | Описание | Пример payload |
|-----|----------|---------------|
| `auth_response` | Ответ аутентификации | `{"success": true, "session_id": "xxx"}` |
| `chat_response` | Ответ чата | `{"content": "Ответ...", "tokens": 100}` |
| `command_response` | Ответ команды | `{"success": true, "output": "..."}` |
| `error` | Ошибка | `{"code": 400, "message": "Bad request"}` |
| `event` | Событие | `{"event": "client_connected", "data": {...}}` |
| `presence` | Статус присутствия | `{"event": "client_disconnected", "total": 5}` |

---

## 🔐 Аутентификация

### Режимы

| Режим | Описание |
|-------|----------|
| `none` | Без аутентификации |
| `token` | Bearer токен |
| `password` | Пароль |

### Пример аутентификации

```json
{
  "type": "auth",
  "payload": {
    "client_id": "my_client",
    "client_type": "cli",
    "token": "your_auth_token"
  }
}
```

### Ответ

```json
{
  "type": "auth_response",
  "payload": {
    "success": true,
    "session_id": "session_123"
  }
}
```

---

## 📁 Конфигурация

### Файл конфигурации

`~/.qwen/gateway.json`

```json
{
  "enabled": true,
  "port": 18789,
  "bind": "127.0.0.1",
  "max_clients": 100,
  "debug": false,
  "auth": {
    "mode": "token",
    "token": "auto_generated",
    "password": "",
    "allow_list": []
  },
  "tailscale": {
    "mode": "none",
    "hostname": ""
  }
}
```

### Опции

| Опция | Описание | По умолчанию |
|-------|----------|--------------|
| `enabled` | Включить Gateway | `true` |
| `port` | Порт для прослушивания | `18789` |
| `bind` | Адрес привязки | `127.0.0.1` |
| `max_clients` | Макс. клиентов | `100` |
| `debug` | Режим отладки | `false` |
| `auth.mode` | Режим аутентификации | `token` |
| `auth.token` | Токен доступа | авто |
| `tailscale.mode` | Tailscale режим | `none` |

---

## 🧪 Примеры использования

### 1. Подключение через CLI

```bash
# websocat
websocat ws://127.0.0.1:18789/ws

# Отправка аутентификации
echo '{"type":"auth","payload":{"client_id":"cli","client_type":"cli"}}' | websocat ws://127.0.0.1:18789/ws
```

### 2. Подключение через JavaScript

```javascript
const ws = new WebSocket('ws://127.0.0.1:18789/ws');

ws.onopen = () => {
    console.log('Connected!');
    
    // Аутентификация
    ws.send(JSON.stringify({
        type: 'auth',
        payload: {
            client_id: 'web_client',
            client_type: 'web',
            token: 'your_token'
        }
    }));
};

ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    console.log('Received:', msg);
};

// Отправка сообщения чата
ws.send(JSON.stringify({
    type: 'chat',
    payload: {
        content: 'Привет, Gateway!',
        model: 'gpt-4'
    }
}));
```

### 3. Python клиент

```python
import websocket
import json

def on_open(ws):
    print("Connected!")
    ws.send(json.dumps({
        "type": "auth",
        "payload": {
            "client_id": "python_client",
            "client_type": "cli",
            "token": "your_token"
        }
    }))

def on_message(ws, message):
    print("Received:", message)

ws = websocket.WebSocketApp(
    "ws://127.0.0.1:18789/ws",
    on_open=on_open,
    on_message=on_message
)

ws.run_forever()
```

---

## 📊 API Endpoints

### GET /health

Health check.

**Ответ:**
```json
{
  "status": "ok",
  "clients": 5,
  "auth_token": "abc12345...",
  "uptime": 1234567890
}
```

### GET /api/v1/sessions

Список всех сессий.

**Ответ:**
```json
{
  "sessions": [...],
  "total": 10
}
```

### GET /api/v1/clients

Список подключённых клиентов.

**Ответ:**
```json
{
  "clients": [
    {
      "id": "client_123",
      "type": "cli",
      "session": "session_456",
      "active": true
    }
  ],
  "total": 5
}
```

### POST /api/v1/broadcast

Рассылка события всем клиентам.

**Запрос:**
```json
{
  "event": "notification",
  "data": {
    "message": "Hello all!"
  }
}
```

**Ответ:**
```json
{
  "success": true
}
```

---

## 🛡️ Безопасность

### Рекомендации

1. **Используйте loopback** по умолчанию
   ```bash
   ./qwen-claw gateway --bind 127.0.0.1
   ```

2. **Включите аутентификацию**
   ```json
   {
     "auth": {
       "mode": "token"
     }
   }
   ```

3. **Для удалённого доступа используйте SSH tunnel**
   ```bash
   ssh -L 18789:localhost:18789 user@remote
   ```

4. **Или Tailscale**
   ```json
   {
     "gateway": {
       "tailscale": {
         "mode": "serve"
       }
     }
   }
   ```

---

## 🧩 Интеграция с другими компонентами

### Telegram бот

```go
// Подключение бота к Gateway
gw, _ := gateway.NewGateway(config)
gw.Broadcast("telegram_message", map[string]interface{}{
    "chat_id": chatID,
    "text": "Новое сообщение",
})
```

### Web UI

```javascript
// Подключение Web UI
const ws = new WebSocket('ws://127.0.0.1:18789/ws');
ws.send(JSON.stringify({
    type: 'subscribe',
    payload: { channel: 'chat_updates' }
}));
```

### CLI

```go
// Отправка команды из CLI
msg := gateway.Message{
    Type: gateway.MessageTypeCommand,
    Payload: json.RawMessage(`{"command": "run", "args": {"code": "ls -la"}}`),
}
```

---

## 🐛 Отладка

### Включить debug режим

```bash
./qwen-claw gateway --debug
```

### Логи

Логи пишутся в стандартный вывод и файл:
```
~/.qwen/logs/gateway.log
```

### Проверка подключения

```bash
# WebSocket
websocat ws://127.0.0.1:18789/ws

# HTTP API
curl http://127.0.0.1:18789/health
curl http://127.0.0.1:18789/api/v1/sessions
curl http://127.0.0.1:18789/api/v1/clients
```

---

## 📚 Ссылки

- [OpenClaw Gateway Architecture](https://github.com/openclaw/openclaw)
- [WebSocket Protocol](https://datatracker.ietf.org/doc/html/rfc6455)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)

---

**Qwen-Claw Gateway** — центр управления всеми клиентами 🚀
