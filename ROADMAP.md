# 🚀 Qwen-Claw Roadmap

**Дата:** 2026-03-22  
**Статус:** 5/5 фаз базового функционала завершено ✅

---

## ✅ Выполнено (100%)

### Фазы реализации:

| Фаза | Статус | Файлы | Строк |
|------|--------|-------|-------|
| **Gateway WebSocket** | ✅ | 3 | ~850 |
| **Session Store** | ✅ | 2 | ~650 |
| **Skills Registry** | ✅ | 2 | ~1000 |
| **Web Control UI** | ✅ | 5 | ~1900 |
| **Docker Sandbox** | ✅ | 4 | ~1150 |

**Всего:** 16 файлов, ~6500 строк кода

---

## 📋 План доработок

### 🔴 Критичные (нужно сделать срочно)

#### 1. Исправление тестов Telegram бота

**Проблема:** 4 теста failing
- `TestFormatResponse/escape_markdown`
- `TestFormatResponse/code_blocks`
- `TestEmptyAllowedUsers`
- `TestBotStop` (panic)

**Файлы:**
- `bots/telegram/telegram_test.go`
- `bots/telegram/bot.go`

**Задачи:**
- [ ] Исправить escape markdown функции
- [ ] Исправить code block formatting
- [ ] Исправить TestEmptyAllowedUsers
- [ ] Исправить Bot stop logic

---

#### 2. Исправление Skills Developer

**Проблема:** Не компилируется
```
skills/developer/developer.go: undefined: skills.Info
skills/developer/developer.go: undefined: skills.Request
skills/developer/developer.go: undefined: skills.Response
```

**Файлы:**
- `skills/developer/developer.go`
- `internal/skills/engine.go`

**Задачи:**
- [ ] Добавить missing types (Info, Request, Response)
- [ ] Исправить импорты
- [ ] Протестировать компиляцию

---

#### 3. Написание тестов для новых компонентов

**Проблема:** 0% покрытия для новых модулей

**Файлы для тестов:**
```
internal/gateway/
  ├── gateway_test.go      # 10 тестов
  ├── session_test.go      # 8 тестов
  └── config_test.go       # 5 тестов

internal/memory/
  ├── session_store_test.go # 10 тестов
  └── compaction_test.go    # 8 тестов

internal/sandbox/
  ├── sandbox_test.go       # 10 тестов
  ├── policy_test.go        # 12 тестов
  └── config_test.go        # 6 тестов

internal/web/
  └── handlers_test.go      # 15 тестов
```

**Задачи:**
- [ ] Написать тесты для Gateway
- [ ] Написать тесты для Session Store
- [ ] Написать тесты для Sandbox
- [ ] Написать тесты для Web UI handlers
- [ ] Достичь 40-50% coverage

---

### 🟡 Важные (следующая итерация)

#### 4. Интеграция Web UI с Gateway (улучшения)

**Проблема:** Частичная интеграция

**Файлы:**
- `internal/web/server.go`
- `web/static/js/app.js`

**Задачи:**
- [ ] Улучшить real-time события от Gateway
- [ ] Добавить управление сессиями Gateway из Web UI
- [ ] Добавить статистику клиентов Gateway
- [ ] Добавить WebSocket для логов Gateway

---

#### 5. Улучшение работы с конфигами

**Проблема:** Конфигурация сохраняется в отдельный файл

**Файлы:**
- `internal/web/server.go`
- `internal/config/config.go`

**Задачи:**
- [ ] Сохранение конфига в основное хранилище
- [ ] Синхронизация между config.yaml и config.web.json
- [ ] Валидация конфига перед сохранением
- [ ] История изменений конфига

---

#### 6. Улучшение логирования

**Проблема:** Простой парсинг логов

**Файлы:**
- `internal/web/server.go`
- `internal/logger/logger.go`

**Задачи:**
- [ ] Улучшить парсинг логов (regex)
- [ ] Добавить фильтрацию по времени
- [ ] Добавить поиск по логам
- [ ] Добавить экспорт логов

---

### 🟢 Долгосрочные (улучшение качества)

#### 7. Multi-agent routing

**Описание:** Изолированные агенты для разных каналов

**Файлы:**
- `internal/agent/agent.go`
- `internal/agent/router.go` (новый)

**Задачи:**
- [ ] Создать роутер агентов
- [ ] Добавить маппинг каналов на агентов
- [ ] Добавить индивидуальные настройки на агента
- [ ] Добавить изоляцию контекста

---

#### 8. Tailscale интеграция

**Описание:** Удалённый доступ через Tailscale

**Файлы:**
- `internal/gateway/config.go`
- `internal/gateway/tailscale.go` (новый)

**Задачи:**
- [ ] Добавить Tailscale Serve режим
- [ ] Добавить Tailscale Funnel режим
- [ ] Добавить настройку через CLI
- [ ] Добавить документацию

---

#### 9. Векторная память (sqlite-vec)

**Описание:** Улучшенный поиск по памяти

**Файлы:**
- `internal/memory/vector.go` (новый)
- `internal/memory/search.go`

**Задачи:**
- [ ] Добавить sqlite-vec расширение
- [ ] Добавить embedding генерацию
- [ ] Добавить гибридный поиск (vector + BM25)
- [ ] Добавить кэширование embeddings

---

#### 10. Browser automation

**Описание:** Управление браузером через CDP

**Файлы:**
- `internal/tools/browser.go` (новый)
- `internal/skills/browser.go` (новый)

**Задачи:**
- [ ] Добавить CDP подключение к Chrome
- [ ] Добавить навигацию по страницам
- [ ] Добавить скриншоты
- [ ] Добавить заполнение форм
- [ ] Добавить парсинг контента

---

#### 11. Voice Wake + Talk Mode

**Описание:** Голосовое управление

**Файлы:**
- `internal/voice/wake.go` (новый)
- `internal/voice/talk.go` (новый)

**Задачи:**
- [ ] Добавить wake words detection
- [ ] Добавить TTS (ElevenLabs + системный)
- [ ] Добавить STT (whisper.cpp)
- [ ] Добавить continuous talk mode

---

#### 12. Canvas/A2UI

**Описание:** Визуальное рабочее пространство

**Файлы:**
- `internal/web/canvas.go` (новый)
- `web/static/canvas/` (новая папка)

**Задачи:**
- [ ] Добавить A2UI компоненты
- [ ] Добавить рендеринг графиков
- [ ] Добавить интерактивные элементы
- [ ] Добавить интеграцию с агентом

---

#### 13. Mobile Apps

**Описание:** iOS/Android приложения

**Платформы:**
- iOS (Swift)
- Android (Kotlin)

**Задачи:**
- [ ] Создать iOS приложение
- [ ] Создать Android приложение
- [ ] Добавить Canvas support
- [ ] Добавить Voice Wake
- [ ] Добавить камеру и экран

---

#### 14. Device Pairing

**Описание:** Cryptographic pairing для устройств

**Файлы:**
- `internal/gateway/pairing.go` (новый)
- `internal/web/pairing.go` (новый)

**Задачи:**
- [ ] Добавить challenge-response аутентификацию
- [ ] Добавить device tokens
- [ ] Добавить UI для pairing
- [ ] Добавить отзыв устройств

---

#### 15. Session Tools (Agent-to-Agent)

**Описание:** Координация между агентами

**Файлы:**
- `internal/agent/session_tools.go` (новый)

**Задачи:**
- [ ] Добавить sessions_list tool
- [ ] Добавить sessions_send tool
- [ ] Добавить sessions_history tool
- [ ] Добавить sessions_spawn tool

---

#### 16. Webhooks

**Описание:** Внешние триггеры для задач

**Файлы:**
- `internal/scheduler/webhook.go` (новый)

**Задачи:**
- [ ] Добавить webhook endpoints
- [ ] Добавить аутентификацию вебхуков
- [ ] Добавить payload валидацию
- [ ] Добавить retry logic

---

## 📊 Приоритеты

### Спринт 1 (1-2 недели)
1. Исправление тестов Telegram бота 🔴
2. Исправление Skills Developer 🔴
3. Тесты для Gateway 🔴
4. Тесты для Session Store 🔴
5. Тесты для Sandbox 🔴

### Спринт 2 (2-3 недели)
6. Интеграция Web UI с Gateway 🟡
7. Улучшение работы с конфигами 🟡
8. Улучшение логирования 🟡
9. Integration тесты 🟡

### Спринт 3 (1-2 месяца)
10. Multi-agent routing 🟢
11. Tailscale интеграция 🟢
12. Векторная память 🟢

### Спринт 4 (2-3 месяца)
13. Browser automation 🟢
14. Voice Wake 🟢
15. Canvas/A2UI 🟢

### Спринт 5 (3-6 месяцев)
16. Mobile Apps 🟢
17. Device Pairing 🟢
18. Session Tools 🟢
19. Webhooks 🟢

---

## 📈 Метрики качества

| Метрика | Текущая | Цель (спринт 1) | Цель (спринт 2) |
|---------|---------|-----------------|-----------------|
| **Test Coverage** | ~15% | 40% | 60% |
| **Passing Tests** | 12/16 | 16/16 | 20/20 |
| **Build Success** | 85% | 100% | 100% |
| **Integration Tests** | 5 | 10 | 20 |
| **E2E Tests** | 0 | 0 | 5 |

---

## 🎯 Definition of Done

### Для критичных задач:
- [ ] Код написан
- [ ] Тесты написаны (покрытие >85%)
- [ ] Все тесты проходят
- [ ] Документация обновлена
- [ ] Коммит запушен

### Для важных задач:
- [ ] Код написан
- [ ] Тесты написаны
- [ ] Интеграция работает
- [ ] Документация обновлена

### Для долгосрочных:
- [ ] Код написан
- [ ] Тесты написаны
- [ ] Функциональность работает
- [ ] Документация обновлена
- [ ] Примеры использования добавлены

---

## 🔧 Быстрый старт

### Для новых разработчиков:

```bash
# 1. Клонировать репозиторий
git clone https://github.com/sizotov81-hub/qwen-claw.git
cd qwen-claw

# 2. Переключиться на dev ветку
git checkout dev

# 3. Установить зависимости
go mod download

# 4. Собрать проект
CGO_ENABLED=0 go build -o qwen-claw ./cmd/main.go

# 5. Запустить тесты
CGO_ENABLED=0 go test ./... -v

# 6. Запустить проект
./qwen-claw -i
```

### Для начала работы над задачами:

```bash
# Выбрать задачу из списка выше
# Создать ветку
git checkout -b feature/TASK_NAME

# Сделать изменения
# ...

# Закоммитить
git add .
git commit -m "feat: TASK_NAME description"

# Запушить
git push origin feature/TASK_NAME
```

---

## 📚 Ресурсы

### Документация:
- `GATEWAY.md` — WebSocket сервер
- `SESSION_STORE.md` — Event-based сессии
- `SKILLS_REGISTRY.md` — Система навыков
- `WEB_UI.md` — Веб-интерфейс
- `DOCKER_SANDBOX.md` — Docker sandbox
- `TESTING_REPORT.md` — Тестирование

### Контекст:
- `OPENCLAW_FEATURES_TO_BORROW.md` — Сравнение с OpenClaw
- `README.md` — Основная документация
- `SECURITY.md` — Безопасность

---

## 🎉 Итоги

**Выполнено:** 5/5 фаз базового функционала (100%)  
**Осталось:** 19 задач доработок  
**Критичных:** 3 задачи  
**Важных:** 3 задачи  
**Долгосрочных:** 13 задач

**Следующий шаг:** Начать со спринта 1 (критичные задачи)
