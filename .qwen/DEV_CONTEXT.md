# Qwen-Claw — Контекст разработки

## Дата последней актуализации
21 марта 2026 г.

## О проекте

**Qwen-Claw** — это оболочка-расширение для **Qwen Code CLI**, написанная на Go, которая добавляет:
- Персистентную память и контекст между сессиями
- Интеграцию с мессенджерами (Telegram бот)
- Веб-интерфейс с безопасной аутентификацией
- Планировщик задач (cron-подобный)
- Расширяемую систему навыков (skills)

**Расположение проекта:** `/home/ss/qwen-claw`

---

## ✅ Выполненные задачи (100% плана)

### 1. Базовая оболочка для Qwen Code CLI
- CLI с командами: `run`, `ask`, `exec`, `chat`, `-i` (интерактив)
- Интеграция с Qwen Code CLI через exec
- Конфигурация через YAML + .env файлы
- **Файлы:** `cmd/main.go`, `internal/agent/agent.go`

### 2. Персистентная память
- JSON-хранилище в `.qwen/memory/`
- Типы записей: fact, context, conversation, task
- Поиск по памяти, добавление, очистка
- **Файлы:** `internal/memory/manager.go`

### 3. Telegram бот
- Интеграция через `go-telegram-bot-api`
- Команды: `/start`, `/help`, `/status`, `/memory`, `/clear`, `/model`, `/skills`, `/tasks`, `/addtask`
- Inline-кнопки для навигации
- Обработка документов/файлов
- Проверка доступа по ID пользователей
- **Файлы:** `bots/telegram/bot.go`

### 4. Планировщик задач (Scheduler)
- Cron-подобные выражения (`@daily`, `@hourly`, `*/5 * * * *`, etc.)
- Хранение задач в `.qwen/scheduler/tasks.json`
- CLI команды: `tasks add`, `tasks list`, `tasks run`, `tasks remove`, `tasks enable/disable`
- Интеграция с Telegram ботом
- **Файлы:** `internal/scheduler/scheduler.go`

### 5. Встроенные навыки (Skills)
- **shell** — выполнение shell команд
- **file** — операции с файлами (read/write)
- **search** — поиск по файлам (grep)
- **memory** — управление памятью
- **git** — Git операции (status, commit, push, pull)
- **http** — HTTP запросы (GET/POST через curl)
- **notify** — Desktop уведомления (notify-send/osascript/PowerShell)
- Внешние навыки из `skills/` директории (скрипты .sh, .py)
- **Файлы:** `internal/skills/engine.go`

### 6. Web UI
- HTTP сервер на порту **64656** (нестандартный)
- **Безопасность:**
  - Секретная фраза генерируется при первом запуске
  - JWT токены (24 часа)
  - При сканировании отвечает 404 без аутентификации
- **Интерфейс:**
  - 💬 Чат (WebSocket для реального времени)
  - 🧠 Память (CRUD)
  - 📋 Задачи (планировщик)
  - 🛠️ Навыки (список)
  - 📊 Статус системы
- **Файлы:** `internal/web/server.go`, `internal/web/static/*`

### 7. Docker образ
- Многоэтапная сборка (размер ~104 MB)
- Alpine 3.19 + Go 1.21
- docker-compose.yml с томами
- **Файлы:** `Dockerfile`, `docker-compose.yml`, `DOCKER.md`

---

## 📁 Структура проекта

```
qwen-claw/
├── cmd/
│   └── main.go              # Точка входа CLI
├── internal/
│   ├── agent/               # Оболочка для Qwen Code CLI
│   │   └── agent.go
│   ├── bots/
│   │   └── telegram/        # Telegram бот
│   │       └── bot.go
│   ├── config/              # Конфигурация
│   │   └── config.go
│   ├── memory/              # Менеджер памяти
│   │   └── manager.go
│   ├── scheduler/           # Планировщик задач
│   │   └── scheduler.go
│   ├── skills/              # Skill engine
│   │   └── engine.go
│   └── web/                 # Web UI
│       ├── server.go
│       └── static/
│           ├── index.html
│           ├── styles.css
│           └── app.js
├── bots/
│   └── telegram/            # (дубликат, используется internal/bots)
├── skills/
│   └── example/             # Пример навыка
│       ├── main.sh
│       └── skill.json
├── .qwen/
│   ├── memory/              # Данные памяти
│   └── scheduler/           # Данные планировщика
├── .web-secrets.json        # Секреты Web UI (не в git!)
├── .env                     # Переменные окружения (не в git!)
├── config.yaml              # Конфигурация
├── Dockerfile
├── docker-compose.yml
├── DOCKER.md
├── README.md
├── go.mod
└── go.sum
```

---

## 🔧 Команды CLI

```bash
# Основные
./qwen-claw -i                           # Интерактивный режим
./qwen-claw run "<запрос>"               # Выполнить запрос
./qwen-claw chat                         # Чат-сессия
./qwen-claw doctor                       # Проверка установки

# Память
./qwen-claw memory                       # Показать память
./qwen-claw memory add "<текст>"         # Добавить запись
./qwen-claw memory search "<запрос>"     # Поиск
./qwen-claw memory clear                 # Очистить

# Навыки
./qwen-claw skills                       # Список навыков
./qwen-claw skills info <name>           # Информация о навыке
./qwen-claw skills run <name> [args]     # Выполнить навык

# Задачи
./qwen-claw tasks                        # Список задач
./qwen-claw tasks add <name> <schedule> <cmd>  # Добавить задачу
./qwen-claw tasks run <id>               # Выполнить задачу
./qwen-claw tasks remove <id>            # Удалить задачу

# Web UI
./qwen-claw web                          # Запустить Web UI (порт 64656)
./qwen-claw web --host 0.0.0.0 --port 8080

# Telegram
./qwen-claw telegram                     # Запустить Telegram бота
```

---

## 🔐 Безопасность Web UI

### Первый запуск Web UI:
```bash
./qwen-claw web

# Вывод:
# 🔐 Web UI Security Initialized
#    Secret Phrase: 90d68743c3f7608d682698ad0f547622173ff97c4251fe34e75666ad502d3ae4
#    (saved to /home/ss/qwen-claw/.web-secrets.json)
```

**Секретная фраза:** `90d68743c3f7608d682698ad0f547622173ff97c4251fe34e75666ad502d3ae4`

### Подключение:
1. Открыть `http://127.0.0.1:64656`
2. Ввести секретную фразу из `.web-secrets.json`
3. JWT токен сохраняется на 24 часа в localStorage

### API:
```bash
# Аутентификация
curl -X POST http://127.0.0.1:64656/api/auth \
  -H "Content-Type: application/json" \
  -d '{"secret":"ваша-фраза"}'

# Использование API
curl http://127.0.0.1:64656/api/status \
  -H "Authorization: Bearer <токен>"

# WebSocket
ws://127.0.0.1:64656/ws?token=<токен>
```

**Без аутентификации сервер отвечает 404 Not Found** (скрытие от сканирования).

---

## 📦 Зависимости (go.mod)

```
github.com/go-telegram-bot-api/telegram-bot-api/v5
github.com/spf13/cobra
github.com/spf13/viper
github.com/joho/godotenv
github.com/gorilla/websocket
github.com/golang-jwt/jwt/v5
```

---

## 🚀 Алиасы (в ~/.bashrc)

```bash
alias qwen-claw="qwen-claw"
alias qwen-tg="qwen-claw telegram"
alias qwen-web="qwen-claw web"
alias qwen-web-open="qwen-claw web --host 0.0.0.0 --port 64656"
```

**Бинарник:** `~/bin/qwen-claw.bin` (wrapper: `~/bin/qwen-claw`)

---

## 🐳 Docker

```bash
# Сборка
docker build -t qwen-claw:latest .

# Запуск
docker-compose up -d

# Логи
docker-compose logs -f
```

**Размер образа:** ~104 MB

---

## 📊 План развития (100% выполнено)

- [x] Базовая оболочка для Qwen Code CLI
- [x] Персистентная память
- [x] Полноценная интеграция Telegram бота
- [x] Web UI интерфейс
- [x] Планировщик задач (cron)
- [x] Больше встроенных навыков (git, http, notify)
- [x] Docker образ

---

## 🔑 Ключевые файлы конфигурации

### .env
```bash
QWEN_CLAW_TELEGRAM_TOKEN="бот-токен"
QWEN_CLAW_TELEGRAM_ALLOWED_USERS="289481051"
QWEN_CLAW_MODEL=""
QWEN_CLAW_APPROVAL_MODE="auto-edit"
```

### config.yaml
```yaml
base_dir: /home/ss/qwen-claw
qwen_dir: /home/ss/qwen-claw/.qwen
skills_dir: /home/ss/qwen-claw/skills
memory_dir: /home/ss/qwen-claw/.qwen/memory

telegram:
  enabled: false

llm:
  model: ""
  approval_mode: auto-edit
  debug: false
```

---

## 🛠️ Сборка и запуск

```bash
cd /home/ss/qwen-claw

# Сборка
CGO_ENABLED=0 go build -o qwen-claw ./cmd/main.go

# Установка
cp qwen-claw ~/bin/qwen-claw.bin

# Проверка
qwen-claw doctor
```

---

## 📝 Заметки

- Порт Web UI: **64656** (нестандартный)
- Секретная фраза хранится в `.web-secrets.json` (не коммитить в git!)
- Telegram токен в `.env` (не коммитить в git!)
- Все навыки работают через `skills run <name>`
- Планировщик запускается автоматически с Web UI и Telegram ботом
- WebSocket используется для реального времени в чате

---

## 📞 Контакты и ссылки

- Telegram бот: используется токен из .env
- Web UI: http://127.0.0.1:64656
- Docker: образ `qwen-claw:latest`

---

**Этот файл содержит полный контекст проекта для быстрого восстановления работы в новой сессии.**
