# Qwen-Claw

**Qwen-Claw** — это оболочка-расширение для **Qwen Code CLI**, которая добавляет персистентную память, интеграцию с мессенджерами и другие возможности в стиле OpenCLAW.

## Архитектура

```
┌─────────────────────────────────────────────────────────────────┐
│                 Qwen-Claw (оболочка)                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Memory    │  │  Telegram   │  │   Skills    │             │
│  │   Manager   │  │    Bot      │  │   Engine    │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Qwen Code CLI (ядро)                         │
│              (выполняет все запросы к LLM)                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    LLM (Qwen / другие)                          │
└─────────────────────────────────────────────────────────────────┘
```

## Возможности

- 🖥️ **Оболочка для Qwen Code CLI** — все запросы выполняются через установленный qwen code cli
- 💾 **Персистентная память** — сохранение контекста и истории между сессиями
- 🤖 **Telegram Bot** — интеграция с мессенджерами (в разработке)
- 📝 **История разговоров** — запись и воспроизведение сессий
- 🔌 **Расширяемость** — система навыков для кастомных команд

## Требования

- **Qwen Code CLI** должен быть установлен:
  ```bash
  npm install -g @anthropic-ai/qwen-code
  ```

## Установка

```bash
cd /home/ss/qwen-claw

# Скачать зависимости
go mod tidy

# Собрать бинарник
go build -o qwen-claw ./cmd/main.go

# Установить в PATH (для запуска из любой директории)
cp qwen-claw ~/bin/
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Проверить установку
qwen-claw doctor
```

### Алиасы (опционально)

Добавьте в `~/.bashrc` для удобного запуска:

```bash
# Быстрый запуск Telegram бота
alias qwen-tg="qwen-claw telegram"

# Применить изменения
source ~/.bashrc
```

## Установка через Docker

Альтернативный способ запуска Qwen-Claw в Docker контейнере.

### Быстрый старт

```bash
# Клонировать репозиторий
cd /home/ss/qwen-claw

# Настроить конфигурацию
cp config.yaml.example config.yaml
cp .env.example .env
nano .env  # Добавьте QWEN_CLAW_TELEGRAM_TOKEN

# Запустить с docker-compose
docker-compose up -d

# Просмотр логов
docker-compose logs -f
```

**Подробнее:** см. [DOCKER.md](DOCKER.md)

## Web UI

Qwen-Claw включает встроенный веб-интерфейс для удобной работы.

### 🔐 Безопасность

Web UI защищён двухфакторной аутентификацией:

1. **Секретная фраза** — генерируется при первом запуске, сохраняется в `.web-secrets.json`
2. **JWT токен** — выдаётся после успешной аутентификации, действует 24 часа

**При сканировании порта сервис отвечает 404 Not Found** — без секретной фразы невозможно определить, что сервис работает.

### Запуск

```bash
# Запустить Web UI
./qwen-claw web

# При первом запуске будет сгенерирована секретная фраза:
# 🔐 Web UI Security Initialized
#    Secret Phrase: <ваша-фраза>
#    (saved to /home/ss/qwen-claw/.web-secrets.json)
```

### Подключение

1. Откройте браузер: `http://127.0.0.1:64656`
2. Введите секретную фразу из `.web-secrets.json`
3. После аутентификации получите доступ к интерфейсу

### API Endpoints

| Endpoint | Метод | Описание |
|----------|-------|----------|
| `/api/auth` | POST | Аутентификация (возвращает JWT токен) |
| `/api/health` | GET | Проверка здоровья (требует секрет) |
| `/api/chat` | POST | Отправить сообщение в чат (требует JWT) |
| `/api/memory` | GET | Получить записи памяти (требует JWT) |
| `/api/memory` | POST | Добавить запись (требует JWT) |
| `/api/memory` | DELETE | Очистить память (требует JWT) |
| `/api/tasks` | GET | Список задач (требует JWT) |
| `/api/tasks` | POST | Добавить задачу (требует JWT) |
| `/api/skills` | GET | Список навыков (требует JWT) |
| `/api/status` | GET | Статус системы (требует JWT) |
| `/ws` | WebSocket | Подключение для реального времени (требует токен в query) |

### Заголовки для API

```
Authorization: Bearer <ваш-JWT-токен>
X-Secret-Phrase: <ваша-секретная-фраза>
```

### Параметры для WebSocket

```
ws://127.0.0.1:64656/ws?token=<ваш-JWT-токен>
```

## Быстрый старт

### 1. Проверка установки

```bash
./qwen-claw doctor
```

### 2. Инициализация

```bash
./qwen-claw init
```

### 3. Настройка приватных данных

Скопируйте `.env.example` в `.env` и заполните значения:

```bash
cp .env.example .env
nano .env  # или ваш редактор
```

**Переменные окружения:**

```bash
# Telegram Bot Token (от @BotFather)
QWEN_CLAW_TELEGRAM_TOKEN="123456789:ABCdefGHIjklMNOpqrsTUVwxyz"

# Telegram Allowed Users (список ID через запятую, опционально)
QWEN_CLAW_TELEGRAM_ALLOWED_USERS="123456789,987654321"

# Модель по умолчанию (опционально)
QWEN_CLAW_MODEL=""

# Режим подтверждения: plan, default, auto-edit, yolo
QWEN_CLAW_APPROVAL_MODE="auto-edit"
```

### 4. Первый запрос

```bash
./qwen-claw run "Привет! Расскажи о себе"
```

## Использование

### CLI команды

| Команда | Описание |
|---------|----------|
| `./qwen-claw -i` | Интерактивный режим |
| `./qwen-claw run <query>` | Выполнить запрос |
| `./qwen-claw ask <query>` | Задать вопрос (алиас run) |
| `./qwen-claw exec <cmd>` | Выполнить команду |
| `./qwen-claw chat` | Чат-сессия с историей |
| `./qwen-claw telegram` | Запустить Telegram бота |
| `./qwen-claw web` | **Новое** — Запустить Web UI |
| `./qwen-claw skills` | Показать навыки |
| `./qwen-claw skills run <name> [args]` | Выполнить навык |
| `./qwen-claw skills info <name>` | Информация о навыке |
| `./qwen-claw memory` | Показать память |
| `./qwen-claw memory add <text>` | Добавить в память |
| `./qwen-claw memory search <q>` | Поиск в памяти |
| `./qwen-claw memory clear` | Очистить память |
| `./qwen-claw tasks` | Показать задачи |
| `./qwen-claw tasks add <name> <schedule> <cmd>` | Добавить задачу |
| `./qwen-claw tasks run <id>` | Выполнить задачу |
| `./qwen-claw tasks remove <id>` | Удалить задачу |
| `./qwen-claw init` | Инициализировать проект |
| `./qwen-claw doctor` | Проверить установку |

### Примеры использования

```bash
# Запустить запрос
./qwen-claw run "Создай файл hello.go с программой Hello World"

# Выполнить команду
./qwen-claw exec "ls -la"

# Чат-сессия
./qwen-claw chat

# Интерактивный режим
./qwen-claw -i

# Добавить факт в память
./qwen-claw memory add "Проект использует Go 1.21"

# Поиск в памяти
./qwen-claw memory search "Go"
```

### Telegram бот через прокси

Если Telegram заблокирован, используйте прокси:

```bash
# HTTP/HTTPS прокси
export HTTPS_PROXY="http://proxy-server:port"
./qwen-claw telegram

# Или SOCKS5 прокси
export HTTPS_PROXY="socks5://proxy-server:port"
./qwen-claw telegram

# Или через переменную TELEGRAM_PROXY
export TELEGRAM_PROXY="http://127.0.0.1:8080"
./qwen-claw telegram
```

**Популярные прокси для России:**
- Используйте VPN с HTTP прокси
- Tor: `socks5://127.0.0.1:9050`
- Корпоративные прокси вашей организации

### Интерактивный режим

```
> /help     - показать помощь
> /clear    - очистить историю
> /memory   - показать память
> /exit     - выйти

Примеры запросов:
> exec ls -la          - выполнить команду
> read file.txt        - прочитать файл
> search pattern       - поиск по файлам
> remember fact        - сохранить в память
```

### Чат-режим

```
🔹> /help     - показать помощь
🔹> /clear    - очистить историю
🔹> /memory   - показать память
🔹> /context  - показать контекст
🔹> /exit     - выйти

🔹> Создай файл main.go
⏳ Thinking...
💬 [ответ от Qwen]
```

## Встроенные навыки

Qwen-Claw включает набор встроенных навыков для расширения возможностей:

### 🐚 shell
Выполнение shell команд через Qwen Code CLI.
```bash
./qwen-claw skills run shell "ls -la"
./qwen-claw skills run exec "pwd"
```

### 📁 file
Операции с файлами (чтение, запись, редактирование).
```bash
./qwen-claw skills run file read /path/to/file.txt
./qwen-claw skills run file write /path/to/file.txt "content"
```

### 🔍 search
Поиск по файлам и коду.
```bash
./qwen-claw skills run search "pattern" ./src
./qwen-claw skills run grep "func main" .
```

### 🧠 memory
Управление памятью и контекстом.
```bash
./qwen-claw skills run memory add "Проект использует Go 1.21"
./qwen-claw skills run memory search "Go"
./qwen-claw skills run memory list
```

### 🌲 git
Работа с Git репозиториями.
```bash
./qwen-claw skills run git status
./qwen-claw skills run git commit -m "Fix bug"
./qwen-claw skills run git push
./qwen-claw skills run git log --oneline
```

### 🌐 http
HTTP запросы (GET, POST, PUT, DELETE).
```bash
./qwen-claw skills run http get https://api.example.com/data
./qwen-claw skills run http post https://api.example.com/users '{"name":"John"}'
./qwen-claw skills run curl https://api.example.com
```

### 🔔 notify
Отправка уведомлений (desktop notifications).
```bash
./qwen-claw skills run notify "Задача выполнена!"
./qwen-claw skills run notification "Build failed"
```

**Примечание:** Для работы `notify` требуется одна из систем:
- Linux: `notify-send`
- macOS: `osascript`
- Windows: PowerShell

Если система уведомлений не найдена, уведомления сохраняются в лог-файл.

## Конфигурация

Файл `config.yaml`:

```yaml
# Базовая директория
base_dir: /home/user/qwen-claw

# Настройки Qwen Code CLI
llm:
  # Модель (оставьте пустым для модели по умолчанию)
  model: ""
  
  # Режим подтверждения: plan, default, auto-edit, yolo
  approval_mode: auto-edit
  
  # Режим отладки
  debug: false

# Telegram бот (опционально)
telegram:
  enabled: false
  
  # Токен бота от @BotFather
  token: "YOUR_BOT_TOKEN"
  
  # Разрешённые пользователи (пустой список = все)
  # allowed_users: [123456789]
```

### Настройка Telegram бота

1. Создайте бота через [@BotFather](https://t.me/botfather):
   - Отправьте `/newbot`
   - Следуйте инструкциям
   - Скопируйте полученный токен

2. Добавьте токен в `.env` файл:
```bash
QWEN_CLAW_TELEGRAM_TOKEN="123456789:ABCdefGHIjklMNOpqrsTUVwxyz"

# Опционально: ограничить доступ по ID пользователей
QWEN_CLAW_TELEGRAM_ALLOWED_USERS="123456789,987654321"
```

3. Включите бота в `config.yaml`:
```yaml
telegram:
  enabled: true
```

4. Запустите бота:
```bash
./qwen-claw telegram
```

5. Команды бота:
   - `/start` - Запустить бота
   - `/help` - Показать справку
   - `/status` - Статус бота
   - `/memory` - Показать память
   - `/clear` - Очистить историю
   - `/model` - Текущая модель

## Как это работает

1. **Пользователь отправляет запрос** через CLI или Telegram
2. **Qwen-Claw сохраняет запрос** в локальную память
3. **Запрос передаётся в Qwen Code CLI** с соответствующими аргументами
4. **Qwen Code CLI выполняет запрос** и возвращает результат
5. **Qwen-Claw сохраняет ответ** в память и возвращает пользователю

## Сравнение с OpenCLAW

| Характеристика | OpenCLAW | Qwen-Claw |
|----------------|----------|-----------|
| Ядро | Собственный агент | Qwen Code CLI |
| Язык | TypeScript | Go |
| LLM | Kimi, OpenAI, Claude | Через Qwen Code CLI |
| Память | Локальная | Локальная + контекст CLI |
| Навыки | ClawHub | Встроенные + кастомные |
| Мессенджеры | WhatsApp, Telegram, Discord | Telegram |
| Лицензия | Open-source | Open-source |

## Структура проекта

```
qwen-claw/
├── cmd/
│   └── main.go           # Точка входа CLI
├── internal/
│   ├── agent/            # Оболочка для Qwen Code CLI
│   ├── config/           # Конфигурация
│   ├── memory/           # Менеджер памяти
│   └── skills/           # Skill engine (для будущих расширений)
├── bots/
│   └── telegram/         # Telegram бот
├── skills/
│   └── example/          # Пример навыка
├── .qwen/
│   └── memory/           # Данные памяти
├── config.yaml.example   # Пример конфигурации
├── README.md
├── DOCKER.md             # Документация Docker
├── Dockerfile            # Docker образ
├── docker-compose.yml    # Docker Compose
├── go.mod
└── go.sum
```

## Развитие

### План развития

- [x] Базовая оболочка для Qwen Code CLI
- [x] Персистентная память
- [x] Полноценная интеграция Telegram бота
- [x] Web UI интерфейс
- [x] Планировщик задач (cron)
- [x] Больше встроенных навыков (git, http, notify)
- [x] Docker образ

### Вклад в проект

1. Fork репозиторий
2. Создайте ветку (`git checkout -b feature/amazing`)
3. Закоммитьте изменения (`git commit -m 'Add amazing feature'`)
4. Push (`git push origin feature/amazing`)
5. Откройте Pull Request

## Лицензия

MIT License

## Ссылки

- [Qwen Code CLI](https://github.com/anthropics/qwen-code) — оригинальный проект
- [OpenCLAW](https://openclaw.ai) — вдохновитель проекта
- [Telegram Bot API](https://core.telegram.org/bots/api) — документация Telegram
