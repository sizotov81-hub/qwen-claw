# 🤖 Qwen-Claw

**AI-помощник с персистентной памятью, планировщиком задач и интеграцией с мессенджерами**

[![Build Status](https://img.shields.io/github/actions/workflow/status/sizotov81-hub/qwen-claw/build.yml)](https://github.com/sizotov81-hub/qwen-claw/actions)
[![Coverage Status](https://img.shields.io/badge/coverage-85%25-brightgreen)](https://github.com/sizotov81-hub/qwen-claw)
[![Go Version](https://img.shields.io/github/go-mod/go-version/sizotov81-hub/qwen-claw)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## 🚀 Возможности

### 🧠 Умная память
- **4 уровня памяти**: Working (1мс), FTS (10мс), Associations (50мс), Web (100-500мс)
- **Персистентная история диалогов** — помнит контекст между сессиями
- **Inverted Index** — поиск быстрее в 10-100x (O(1) вместо O(n))
- **LRU кэш конфигурации** — загрузка быстрее в 50-100x

### 📋 Планировщик задач
- **Cron-расписания**: `@daily`, `@hourly`, `*/5 * * * *`
- **Автоматическое выполнение** через агента
- **Приоритеты и подтверждения** для опасных задач

### 💬 Интеграция с мессенджерами
- **Telegram бот** с inline кнопками подтверждения
- **Web UI** с WebSocket и SSE streaming
- **Потоковый вывод** с эффектом печатной машинки

### 🛡️ Безопасность
- **3 режима подтверждений**: `plan` / `auto-edit` / `yolo`
- **Rate limiting**: 100 сообщений/минуту
- **WebSocket защита**: 1MB лимит, ping/pong keepalive
- **Хранение секретов**: только в `.env` (исключён из git)

### 🔧 Расширяемость
- **Система навыков (Skills)**: shell, file, search, memory, git, http, notify
- **Intent Detection**: автоматическое распознавание команд
- **Скил разработчика** с правилами безопасности

---

## 📦 Установка

### Из исходников

```bash
git clone https://github.com/sizotov81-hub/qwen-claw.git
cd qwen-claw
go build -o qwen-claw ./cmd/main.go
```

### Docker

```bash
docker build -t qwen-claw .
docker run -d \
  -v $(pwd)/.qwen:/app/.qwen \
  -e QWEN_CLAW_TELEGRAM_TOKEN=your_token \
  qwen-claw
```

---

## 🎮 Использование

### CLI

```bash
# Интерактивный режим
./qwen-claw -i

# Единичный запрос
./qwen-claw run "запомни: проект использует Go 1.25"

# Чат-сессия
./qwen-claw chat

# Проверка установки
./qwen-claw doctor
```

### Telegram бот

```bash
export QWEN_CLAW_TELEGRAM_TOKEN="your_bot_token"
export QWEN_CLAW_TELEGRAM_ALLOWED_USERS="123,456"
./qwen-claw telegram
```

### Web UI

```bash
./qwen-claw web --host 0.0.0.0 --port 64656
```

Откройте http://localhost:64656

---

## 📁 Структура проекта

```
qwen-claw/
├── cmd/
│   └── main.go              # Точка входа CLI
├── internal/
│   ├── agent/               # AI агент (личность, память, антидеградация)
│   │   ├── agent.go
│   │   ├── executor.go      # Executor интерфейс
│   │   ├── confirmation.go  # Менеджер подтверждений
│   │   ├── intent.go        # Intent Detection
│   │   ├── stream.go        # Streaming executor
│   │   └── agent_executor.go # Адаптер для планировщика
│   ├── memory/              # Система памяти (4 уровня)
│   │   ├── manager.go
│   │   ├── search.go
│   │   ├── inverted_index.go # Inverted Index для поиска
│   │   └── local_session.go  # Персистентная история
│   ├── scheduler/           # Планировщик задач
│   │   └── scheduler.go
│   ├── config/              # Конфигурация
│   │   ├── config.go
│   │   └── cache.go         # LRU кэш
│   ├── web/                 # Web UI сервер
│   │   ├── server.go
│   │   └── static/          # Frontend
│   ├── skills/              # Система навыков
│   └── logger/              # Логирование
├── bots/
│   └── telegram/            # Telegram бот
├── skills/
│   └── developer/           # Скил разработчика
├── .qwen/
│   ├── memory/              # Базы данных памяти
│   ├── local/               # Персистентная история (приватно)
│   └── scheduler/           # Задачи планировщика (приватно)
└── .env.example             # Пример переменных окружения
```

---

## ⚙️ Конфигурация

### Переменные окружения

```bash
# Telegram
QWEN_CLAW_TELEGRAM_TOKEN="бот токен"
QWEN_CLAW_TELEGRAM_ALLOWED_USERS="123,456"

# LLM
QWEN_CLAW_MODEL="gpt-4"
QWEN_CLAW_APPROVAL_MODE="auto-edit"  # plan/auto-edit/yolo
QWEN_CLAW_DEBUG="false"

# Web UI
QWEN_CLAW_WEB_HOST="127.0.0.1"
QWEN_CLAW_WEB_PORT="64656"
```

### Контекст диалога

| Статус | Заполнение | Действие |
|--------|------------|----------|
| 🟢 Normal | < 50% | Нет действий |
| 🟡 Warning | 50-75% | Предупреждение |
| 🟠 Critical | 75-90% | Рекомендация сжатия |
| 🔴 Overflow | > 90% | Автосжатие/новая сессия |

---

## 🛡️ Безопасность

### ✅ ДЕЛАЙТЕ:

1. **Храните секреты в `.env`**
2. **Используйте `QWEN_CLAW_TELEGRAM_ALLOWED_USERS`**
3. **Проверяйте `.gitignore` перед коммитом**
4. **Используйте режим `auto-edit`**

### ❌ НЕ ДЕЛАЙТЕ:

1. **Не коммитьте `.env` в git**
2. **Не передавайте токены в аргументах**
3. **Не отключайте подтверждение для `rm -rf`**
4. **Не логируйте секреты**

---

## 🧪 Тестирование

```bash
# Запуск всех тестов
go test ./...

# Integration тесты
go test -run Integration ./...

# E2E тесты (требуют TELEGRAM_TOKEN)
TELEGRAM_TOKEN=xxx go test -run E2E ./bots/telegram/...

# Покрытие
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Покрытие тестами

| Пакет | Покрытие | Статус |
|-------|----------|--------|
| internal/agent | 88-95% | ✅ |
| internal/memory | 90% | ✅ |
| internal/scheduler | 92% | ✅ |
| internal/web | 85% | ✅ |
| bots/telegram | 82% | ✅ |

---

## 📈 Производительность

| Операция | До оптимизации | После | Улучшение |
|----------|----------------|-------|-----------|
| config.Load() (повторный) | 5-10ms | 0.1ms | **50-100x** |
| memory.Find() (index hit) | 10-50ms | 0.5-2ms | **10-100x** |
| WebSocket сообщения | Без лимита | 100/min | Защита DoS |

---

## 🔧 API

### Web UI API

| Endpoint | Метод | Описание |
|----------|-------|----------|
| `/api/health` | GET | Health check |
| `/api/auth` | POST | Аутентификация |
| `/api/chat` | POST | Чат (JSON) |
| `/api/chat/stream` | POST | Чат (SSE streaming) |
| `/api/memory` | GET/POST | Управление памятью |
| `/api/tasks` | GET/POST | Задачи планировщика |
| `/api/confirmations` | GET | Ожидающие подтверждения |
| `/api/confirm` | POST | Подтвердить действие |

### SSE Streaming

```javascript
const evtSource = new EventSource('/api/chat/stream');
evtSource.addEventListener('token', (e) => {
    const {token} = JSON.parse(e.data);
    appendToChat(token);
});
evtSource.addEventListener('complete', (e) => {
    const {response} = JSON.parse(e.data);
    console.log('Complete:', response);
});
```

---

## 📚 Документация

- [CONTRIBUTING.md](CONTRIBUTING.md) — руководство по внесению изменений
- [SECURITY.md](SECURITY.md) — политика безопасности
- [TESTING.md](TESTING.md) — руководство по тестированию
- [DOCKER.md](DOCKER.md) — Docker инструкция
- [.qwen/CONTEXT_MANAGEMENT.md](.qwen/CONTEXT_MANAGEMENT.md) — управление контекстом
- [.qwen/MEMORY_AND_HISTORY.md](.qwen/MEMORY_AND_HISTORY.md) — память и история
- [skills/developer/README.md](skills/developer/README.md) — скил разработчика

---

## 🎯 Планы развития

- [ ] Автоматическое суммаризация диалогов
- [ ] Интеграция с LLM для умного сжатия контекста
- [ ] Export/import памяти
- [ ] Skill "analyzer" (анализ кода)
- [ ] E2E тесты для Telegram бота
- [ ] Покрытие тестами 90%+

---

## 🤝 Вклад в проект

1. Fork репозиторий
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменения (`git commit -m 'Add amazing feature'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

---

## 📄 Лицензия

MIT License — см. [LICENSE](LICENSE) файл

---

## 👥 Авторы

- [@sizotov81-hub](https://github.com/sizotov81-hub)

---

## 🙏 Благодарности

- [Qwen Code CLI](https://github.com/anthropics/qwen-code) — основа для агента
- [Telegram Bot API](https://github.com/go-telegram-bot-api/telegram-bot-api) — Telegram интеграция
- [Gorilla WebSocket](https://github.com/gorilla/websocket) — WebSocket поддержка

---

**Qwen-Claw** — умный AI-помощник с памятью и планировщиком 🚀
