# 📋 Список фич и улучшений из OpenClaw для qwen-claw

## 🔍 Анализ OpenClaw

**OpenClaw** — это AI-ассистент с архитектурой **hub-and-spoke**, где единый **Gateway** управляет всеми каналами связи, сессиями и инструментами.

### Ключевые метрики OpenClaw:
- ⭐ **330k+ звёзд** на GitHub
- 🔌 **25+ мессенджеров** (WhatsApp, Telegram, Discord, Slack, iMessage и др.)
- 🎯 **68K+ пользователей**
- 📦 **500+ навыков** в ClawHub

---

## 🎯 Приоритетные функции для заимствования

### 🔴 Критически важные (высокий приоритет)

#### 1. **Gateway WebSocket Control Plane**
**Что это:** Единый WebSocket-сервер для управления всеми клиентами (CLI, Web UI, mobile nodes, боты).

**Зачем нужно:**
- Централизованное управление сессиями
- Real-time координация между клиентами
- Единая точка аутентификации и авторизации
- Возможность подключения нескольких клиентов одновременно

**Как реализовать:**
```
┌───────────────────────────────┐
│         Gateway               │
│    ws://127.0.0.1:18789       │
└──────────────┬────────────────┘
               │
    ┌──────────┼──────────┐
    ▼          ▼          ▼
  CLI      Telegram    Web UI
```

**Файлы для создания:**
- `internal/gateway/server.go` — WebSocket сервер
- `internal/gateway/handlers.go` — обработчики сообщений
- `internal/gateway/session.go` — управление сессиями
- `internal/gateway/auth.go` — аутентификация

---

#### 2. **Система сессий с персистентностью**
**Что это:** Append-only event log с автоматической компaction и ветвлением.

**Зачем нужно:**
- Сохранение полной истории диалогов
- Автоматическое суммаризация старых сообщений
- Поддержка нескольких независимых сессий
- Ветвление контекста для разных задач

**Как реализовать:**
```go
type Session struct {
    ID           string    `json:"id"`
    Type         string    `json:"type"` // main/dm/group
    EventLog     []Event   `json:"events"`
    MemoryIndex  int       `json:"memory_index"`
    Created      time.Time `json:"created"`
    LastAccess   time.Time `json:"last_access"`
}
```

**Файлы для обновления:**
- `internal/memory/session.go` — новая структура сессий
- `internal/memory/session_store.go` — хранилище сессий
- `internal/memory/compaction.go` — автоматическая компaction

---

#### 3. **Система навыков (Skills) с реестром**
**Что это:** ClawHub — централизованный реестр навыков с установкой в один клик.

**Зачем нужно:**
- Пользователи могут устанавливать навыки из реестра
- Навыки не инжектятся все сразу, только релевантные
- Разделение на встроенные, управляемые и пользовательские

**Как реализовать:**
```bash
# CLI команды
qwen-claw skills search <query>
qwen-claw skills install <skill-name>
qwen-claw skills list
qwen-claw skills enable <skill-name>
qwen-claw skills disable <skill-name>
```

**Структура навыка:**
```
skills/<name>/
├── skill.json      # Метаданные
├── SKILL.md        # Документация
├── prompts/        # Системные промпты
├── tools/          # Инструменты
└── tests/          # Тесты
```

**Файлы для обновления:**
- `internal/skills/registry.go` — клиент для реестра навыков
- `internal/skills/injector.go` — селективная инъекция навыков
- `cmd/skills.go` — CLI команды для управления навыками

---

#### 4. **Веб-контроль UI**
**Что это:** Браузерная панель управления с вкладками Chat, Sessions, Config, Skills, Logs.

**Зачем нужно:**
- Визуальное управление сессиями
- Редактирование конфигурации с hot reload
- Просмотр логов в реальном времени
- Установка навыков через UI

**Структура UI:**
```
┌─────────────────────────────────────────┐
│         Qwen-Claw Control UI            │
├─────────────────────────────────────────┤
│  Chat  │ Sessions │ Config │ Skills │ Logs │
└─────────────────────────────────────────┘
```

**Файлы для создания:**
- `internal/web/server.go` — расширенный веб-сервер
- `internal/web/handlers_chat.go` — чат
- `internal/web/handlers_sessions.go` — сессии
- `internal/web/handlers_config.go` — конфигурация
- `internal/web/handlers_skills.go` — навыки
- `internal/web/static/` — фронтенд

---

#### 5. **Sandbox для инструментов**
**Что это:** Docker-контейнеры для изоляции выполнения опасных команд.

**Зачем нужно:**
- Безопасное выполнение shell-команд
- Изоляция по сессиям (DM/Group — sandbox, Main — native)
- Ограничение доступа к файловой системе
- Лимиты CPU/memory

**Как реализовать:**
```json
{
  "sandbox": {
    "enabled": true,
    "mode": "per-session",
    "allowed_tools": ["bash", "read", "write"],
    "denied_tools": ["browser", "nodes"],
    "resources": {
      "memory_limit": "512MB",
      "cpu_limit": "0.5"
    }
  }
}
```

**Файлы для создания:**
- `internal/sandbox/docker.go` — Docker sandbox
- `internal/sandbox/policy.go` — политика безопасности
- `internal/sandbox/resources.go` — лимиты ресурсов

---

### 🟡 Важные (средний приоритет)

#### 6. **Multi-agent routing**
**Что это:** Изолированные агенты для разных каналов/аккаунтов с индивидуальными настройками.

**Зачем нужно:**
- Разные модели для разных каналов
- Индивидуальные настройки approval mode
- Изоляция контекста между агентами

**Конфигурация:**
```json
{
  "agents": {
    "mapping": {
      "telegram:*": {
        "workspace": "~/.qwen/workspaces/telegram",
        "model": "gpt-4",
        "approval_mode": "auto-edit"
      },
      "cli": {
        "workspace": "~/.qwen/workspaces/cli",
        "model": "claude-sonnet",
        "approval_mode": "plan"
      }
    }
  }
}
```

---

#### 7. **Память с гибридным поиском**
**Что это:** Векторный поиск + BM25 keyword matching в SQLite.

**Зачем нужно:**
- Более точный поиск по памяти
- Автоматическая индексация файлов
- Daily notes + долгосрочная память

**Файлы для обновления:**
- `internal/memory/vector_store.go` — векторный поиск
- `internal/memory/bm25.go` — keyword поиск
- `internal/memory/indexer.go` — автоматическая индексация

---

#### 8. **Chat команды**
**Что это:** Слэш-команды в чате для управления сессией.

**Список команд:**
```
/status      — статус сессии (модель + токены)
/new         — новая сессия
/reset       — сброс сессии
/compact     — сжатие контекста
/think       — уровень мышления (off/minimal/low/medium/high)
/verbose     — режим подробностей
/usage       — использование токенов
/skills      — управление навыками
```

**Файлы для обновления:**
- `cmd/chat.go` — расширенная обработка команд

---

#### 9. **Browser automation**
**Что это:** Управление браузером через CDP (Chrome DevTools Protocol).

**Зачем нужно:**
- Скриншоты страниц
- Заполнение форм
- Парсинг динамического контента
- Автоматизация веб-задач

**Инструменты:**
```
browser.navigate(url)
browser.screenshot()
browser.click(selector)
browser.fill(selector, text)
browser.evaluate(js_code)
```

---

#### 10. **Voice Wake + Talk Mode**
**Что это:** Голосовое управление с wake words.

**Зачем нужно:**
- Голосовые команды без касания
- Непрерывный голосовой режим
- Интеграция с TTS (ElevenLabs + системный)

---

### 🟢 Дополнительные (низкий приоритет)

#### 11. **Canvas / A2UI (Agent-to-UI)**
**Что это:** Визуальное рабочее пространство с компонентами.

**Компоненты:**
- Текст, кнопки, списки
- Графики, таблицы
- Формы, дашборды
- Real-time мониторинг

---

#### 12. **Tailscale интеграция**
**Что это:** Удалённый доступ через Tailscale Serve/Funnel.

**Конфигурация:**
```json
{
  "gateway": {
    "tailscale": {
      "mode": "serve"  // или "funnel"
    }
  }
}
```

---

#### 13. **Device pairing**
**Что это:** Cryptographic challenge-response для подключения устройств.

**Flow:**
1. Клиент отправляет device identity
2. Gateway генерирует challenge
3. Клиент подписывает challenge
4. Gateway выдаёт device token

---

#### 14. **Session Tools (Agent-to-Agent)**
**Что это:** Инструменты для координации между агентами.

```
sessions_list       — список активных сессий
sessions_send       — отправка сообщения сессии
sessions_history    — получение истории сессии
sessions_spawn      — создание новой сессии
```

---

#### 15. **Cron + Webhooks**
**Что это:** Планировщик задач + внешние триггеры.

**Файлы для обновления:**
- `internal/scheduler/webhook.go` — вебхуки
- `internal/scheduler/cron.go` — расширенный cron

---

## 📊 Сравнение с текущим qwen-claw

| Функция | OpenClaw | qwen-claw | Приоритет |
|---------|----------|-----------|-----------|
| Gateway WebSocket | ✅ | ❌ | 🔴 Высокий |
| Сессии с компaction | ✅ | ⚠️ Частично | 🔴 Высокий |
| Skills реестр | ✅ (ClawHub) | ❌ | 🔴 Высокий |
| Web Control UI | ✅ | ⚠️ Базовый | 🔴 Высокий |
| Sandbox (Docker) | ✅ | ❌ | 🔴 Высокий |
| Multi-agent routing | ✅ | ❌ | 🟡 Средний |
| Векторная память | ✅ (SQLite-vec) | ❌ | 🟡 Средний |
| Chat команды | ✅ (10+) | ⚠️ (4) | 🟡 Средний |
| Browser automation | ✅ | ❌ | 🟡 Средний |
| Voice Wake | ✅ | ❌ | 🟢 Низкий |
| Canvas/A2UI | ✅ | ❌ | 🟢 Низкий |
| Tailscale | ✅ | ❌ | 🟢 Низкий |
| Device pairing | ✅ | ❌ | 🟢 Низкий |
| Session tools | ✅ | ❌ | 🟢 Низкий |
| Webhooks | ✅ | ❌ | 🟢 Низкий |

---

## 🗺️ Дорожная карта внедрения

### Фаза 1: Фундамент (1-2 месяца)
1. ✅ Gateway WebSocket сервер
2. ✅ Система сессий с персистентностью
3. ✅ Селективная инъекция навыков

### Фаза 2: Интерфейс (1 месяц)
4. ✅ Web Control UI
5. ✅ Chat команды (расширение)

### Фаза 3: Безопасность (1 месяц)
6. ✅ Docker sandbox для инструментов
7. ✅ Access control (allowlists, pairing)

### Фаза 4: Расширения (2-3 месяца)
8. ✅ Multi-agent routing
9. ✅ Векторная память с гибридным поиском
10. ✅ Browser automation

### Фаза 5: Продвинутые функции (3-6 месяцев)
11. ✅ Canvas/A2UI
12. ✅ Voice Wake
13. ✅ Tailscale интеграция

---

## 📁 Рекомендуемая структура проекта

```
qwen-claw/
├── cmd/
│   └── main.go
├── internal/
│   ├── agent/          # AI агент
│   ├── gateway/        # 🔥 Новый: WebSocket Gateway
│   │   ├── server.go
│   │   ├── handlers.go
│   │   ├── session.go
│   │   └── auth.go
│   ├── memory/         # Система памяти
│   │   ├── manager.go
│   │   ├── session.go  # 🔥 Обновить
│   │   ├── vector.go   # 🔥 Новый
│   │   └── compaction.go # 🔥 Новый
│   ├── skills/         # Система навыков
│   │   ├── engine.go
│   │   ├── registry.go # 🔥 Новый: клиент реестра
│   │   └── injector.go # 🔥 Новый: селективная инъекция
│   ├── sandbox/        # 🔥 Новый: Sandbox
│   │   ├── docker.go
│   │   └── policy.go
│   ├── web/            # Web UI
│   │   ├── server.go
│   │   └── handlers_*.go
│   └── config/         # Конфигурация
├── skills/             # Навыки
├── web/                # Фронтенд
│   └── static/
└── .qwen/              # Данные
    ├── sessions/       # 🔥 Новое имя вместо local/
    ├── memory/
    └── skills/
```

---

## 🎯 Быстрые победы (можно сделать за 1-2 недели)

### 1. Расширить chat команды
Добавить: `/status`, `/new`, `/reset`, `/compact`, `/usage`

### 2. Улучшить систему навыков
- Селективная инъекция (только релевантные навыки)
- Структура skill.json с метаданными

### 3. Улучшить сессии
- Append-only event log
- Автосохранение после каждого сообщения
- Команда `/history` с поиском

### 4. Улучшить Web UI
- Вкладка Sessions с управлением
- Вкладка Skills с установкой
- Real-time логи

---

## ⚠️ Риски и предостережения

1. **Не раздувать ядро** — следовать принципу OpenClaw: минималистичное ядро, расширения через навыки
2. **Безопасность по умолчанию** — сильные настройки без потери функциональности
3. **TypeScript vs Go** — OpenClaw использует TypeScript для быстрой итерации, qwen-claw использует Go для производительности
4. **Не копировать слепо** — адаптировать под архитектуру qwen-claw

---

## 📚 Источники

- [OpenClaw GitHub](https://github.com/openclaw/openclaw)
- [OpenClaw Docs](https://docs.openclaw.ai)
- [OpenClaw Security Architecture](https://github.com/openclaw/openclaw/blob/main/SECURITY.md)
- [OpenClaw System Architecture Overview](https://ppaolo.substack.com/p/openclaw-system-architecture-overview)
- [OpenClaw Web UI and Canvas Guide](https://www.ququ123.top/en/2026/02/openclaw-web-ui/)

---

**Дата анализа:** 2026-03-22  
**Аналитик:** Qwen Code (Jarvis persona)
