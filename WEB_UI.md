# 🌐 Qwen-Claw Web Control UI

**Полноценный веб-интерфейс для управления Qwen-Claw**

---

## 🎯 Обзор

Web Control UI — это современный веб-интерфейс для управления Qwen-Claw с поддержкой чата, сессий, навыков, конфигурации и логов.

### Ключевые возможности:

- ✅ **Чат с AI** — общение с агентом в реальном времени
- ✅ **Сессии** — управление сессиями (создание, просмотр, удаление)
- ✅ **Навыки** — установка, включение, отключение навыков
- ✅ **Конфигурация** — редактирование настроек
- ✅ **Логи** — просмотр логов в реальном времени
- ✅ **WebSocket** — подключение к Gateway
- ✅ **Адаптивный дизайн** — работает на десктопе и мобильных

---

## 🚀 Быстрый старт

### Запуск

```bash
# Запустить веб-интерфейс
./qwen-claw web

# С кастомными параметрами
./qwen-claw web --host 0.0.0.0 --port 64656
```

### Доступ

Откройте в браузере:
```
http://localhost:64656
```

---

## 🏗️ Архитектура

```
┌─────────────────────────────────────────────────────┐
│                   Web Browser                       │
│  ┌─────────────────────────────────────────────┐    │
│  │  HTML/CSS/JS  │  WebSocket  │  REST API     │    │
│  └─────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│                  Web Server (Go)                    │
│  ┌─────────────────────────────────────────────┐    │
│  │  Static Files  │  API Handlers  │  Auth     │    │
│  └─────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│                   Qwen-Claw Agent                   │
│  ┌─────────────────────────────────────────────┐    │
│  │  Gateway  │  Skills  │  Memory  │  Session  │    │
│  └─────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
```

---

## 📁 Структура файлов

```
web/
└── static/
    ├── index.html          # Главная страница
    ├── css/
    │   └── style.css       # Стили
    └── js/
        └── app.js          # Клиентское приложение

internal/web/
├── server.go               # Веб-сервер
└── handlers/
    ├── chat.go             # Обработчики чата
    ├── sessions.go         # Обработчики сессий
    ├── skills.go           # Обработчики навыков
    ├── config.go           # Обработчики конфигурации
    └── logs.go             # Обработчики логов
```

---

## 🎨 Интерфейс

### Sidebar (боковая панель)

```
┌─────────────────────┐
│  🦞 Qwen-Claw       │
│  Web Control UI     │
├─────────────────────┤
│  💬 Чат             │
│  📋 Сессии          │
│  🧩 Навыки          │
│  ⚙️  Конфиг         │
│  📜 Логи            │
├─────────────────────┤
│  🟢 Gateway: online │
│  v1.0.0             │
└─────────────────────┘
```

### Вкладки

#### 1. Чат (💬)

- Окно сообщений
- Выбор модели (GPT-4, Claude Sonnet, GPT-3.5)
- Отправка сообщений
- Очистка истории

#### 2. Сессии (📋)

- Список активных сессий
- Создание новой сессии
- Просмотр деталей
- Удаление сессий

#### 3. Навыки (🧩)

- Список установленных навыков
- Поиск навыков
- Включение/отключение
- Установка/удаление

#### 4. Конфиг (⚙️)

- JSON редактор конфигурации
- Сохранение настроек
- Валидация JSON

#### 5. Логи (📜)

- Поток логов в реальном времени
- Фильтрация по уровню (info/warn/error)
- Очистка логов
- Скачивание логов

---

## 🔌 API Endpoints

### Chat

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/chat` | POST | Отправить сообщение |
| `/api/chat/stream` | POST | Чат с streaming |

### Sessions

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/v1/sessions` | GET | Список сессий |
| `/api/v1/sessions` | POST | Создать сессию |
| `/api/v1/sessions?id=X` | DELETE | Удалить сессию |

### Skills

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/skills` | GET | Список навыков |
| `/api/skills` | POST | Установить навык |
| `/api/skills` | DELETE | Удалить навык |

### Config

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/v1/config` | GET | Получить конфиг |
| `/api/v1/config` | POST | Сохранить конфиг |

### Logs

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/v1/logs` | GET | Получить логи |

### WebSocket

| Endpoint | Описание |
|----------|----------|
| `/ws` | WebSocket подключение |

---

## 🔐 Безопасность

### Аутентификация

1. **JWT Token** — основной метод
2. **Secret Phrase** — fallback из `.web-secrets.json`
3. **X-Secret-Phrase header** — для API

### Файл секретов

```json
{
  "secret_phrase": "your-secret-phrase",
  "jwt_secret": "your-jwt-secret"
}
```

Расположение: `~/.qwen/.web-secrets.json`

### CORS

Веб-сервер поддерживает CORS для доступа с других доменов.

---

## 🧪 Тестирование

### Проверка работы

```bash
# Запустить веб-интерфейс
./qwen-claw web --port 64656

# Открыть в браузере
open http://localhost:64656
```

### API тесты

```bash
# Health check
curl http://localhost:64656/api/health

# Получить навыки
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:64656/api/skills

# Получить сессии
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:64656/api/v1/sessions
```

### WebSocket тест

```javascript
const ws = new WebSocket('ws://localhost:64656/ws');

ws.onopen = () => {
    console.log('Connected!');
    
    ws.send(JSON.stringify({
        type: 'auth',
        payload: {
            client_id: 'test',
            client_type: 'web',
            token: 'your_token'
        }
    }));
};

ws.onmessage = (event) => {
    console.log('Received:', JSON.parse(event.data));
};
```

---

## 🎨 Кастомизация

### Изменение темы

В `web/static/css/style.css`:

```css
:root {
    --bg-primary: #1a1a2e;      /* Основной фон */
    --bg-secondary: #16213e;    /* Фон sidebar */
    --accent: #e94560;          /* Акцентный цвет */
    --text-primary: #eee;       /* Основной текст */
}
```

### Добавление новой вкладки

1. Добавить в `index.html`:
```html
<a href="#newtab" class="nav-item" data-tab="newtab">
    <span class="icon">🆕</span>
    <span>Новая</span>
</a>
```

2. Создать контент:
```html
<div class="tab-content" id="newtab-tab">
    <div class="page-header">
        <h2>🆕 Новая вкладка</h2>
    </div>
</div>
```

3. Обработать в `app.js`:
```javascript
loadTabData(tab) {
    switch(tab) {
        case 'newtab':
            this.loadNewTab();
            break;
    }
}
```

---

## 📊 Метрики

### Производительность

| Метрика | Значение |
|---------|----------|
| Время загрузки | < 1s |
| WebSocket latency | < 50ms |
| API response time | < 100ms |

### Поддержка браузеров

- ✅ Chrome 90+
- ✅ Firefox 88+
- ✅ Safari 14+
- ✅ Edge 90+

---

## 🐛 Отладка

### Включить debug логи

```bash
./qwen-claw web --debug
```

### Логи веб-сервера

```
🔒 Web UI Security Initialized
🌐 Web UI starting at http://127.0.0.1:64656
📝 Use secret phrase for authentication
```

### Консоль браузера

Откройте DevTools (F12) → Console для просмотра логов клиента.

---

## 📚 Ссылки

- [OpenClaw Web UI](https://docs.openclaw.ai/web-ui)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)

---

**Qwen-Claw Web Control UI** — современный интерфейс для управления AI 🚀
