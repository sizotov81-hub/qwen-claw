# 📋 Session Store — Event-based система сессий

**Персистентная история диалогов с автосуммаризацией**

---

## 🎯 Обзор

Session Store — это новая система хранения истории диалогов на основе событий (Event-based), вдохновлённая архитектурой OpenClaw.

### Ключевые возможности:

- ✅ **Event-based архитектура** — каждое сообщение это событие с типом и метаданными
- ✅ **Автосуммаризация** — старые сообщения суммаризируются для экономии места
- ✅ **Персистентность** — история сохраняется на диск после каждого сообщения
- ✅ **Compaction** — автоматическое сжатие старых событий
- ✅ **Статус компaction** — визуальное отображение заполнения сессии

---

## 🏗️ Архитектура

```
┌─────────────────────────────────────────────────────┐
│                    Agent                            │
├─────────────────────────────────────────────────────┤
│  conversationHistory  → RAM кэш (быстрый доступ)    │
│  localSession         → Старая система (совместимость) │
│  eventStore           → Event-based хранилище      │
│  sessionStoreManager  → Менеджер хранилищ          │
│  compactor            → Автосуммаризация           │
└─────────────────────────────────────────────────────┘
```

---

## 📁 Типы событий

| Тип | Описание | Пример |
|-----|----------|--------|
| `user_message` | Сообщение пользователя | "привет, как дела?" |
| `assistant_reply` | Ответ ассистента | "Привет! Я хорошо..." |
| `system_message` | Системное сообщение | "Контекст сжат" |
| `tool_call` | Вызов инструмента | "execute: ls -la" |
| `tool_result` | Результат инструмента | "total: 10 files" |
| `error` | Ошибка | "Failed to execute..." |
| `summary` | Суммаризация | "📝 Summary of 50 events..." |

---

## 🔧 Использование

### В чате

```bash
./qwen-claw chat

# Команды:
🔹> /history      # Показать историю с суммаризацией
🔹> /compact      # Сжать контекст вручную
🔹> /clear        # Очистить историю
```

### Программно

```go
// Получить все события
events := agent.GetSessionEvents()

// Получить последние N событий
recent := agent.GetRecentSessionEvents(20)

// Получить с суммаризацией
summary, events := agent.GetSessionWithSummary()

// Получить статус компaction
status := agent.GetCompactionStatus()
fmt.Println(status.FormatStatus()) // "🟢 Normal (45/100 events, 45.0%)"

// Сжать сессию вручную
summary := memory.SimpleSummaryGenerator(events)
agent.CompactSession(summary)
```

---

## 📊 Compaction (автосуммаризация)

### Конфигурация по умолчанию

```go
type CompactionConfig struct {
    MaxEvents:   100,      // Макс. событий перед компaction
    KeepLastN:   20,       // Сохранять последних N событий
    AutoCompact: true,     // Автоматическая компaction
}
```

### Статусы компaction

| Статус | Процент | Описание |
|--------|---------|----------|
| 🟢 Normal | < 50% | Всё хорошо |
| 🟡 Warning | 50-75% | Рекомендуется сжатие |
| 🟠 Critical | 75-90% | Требуется сжатие |
| 🔴 Overflow | > 90% | Немедленное сжатие |

### Пример суммаризации

```
📝 Summary of 80 events from 2026-03-22 14:30:

- User messages: 40
- Assistant replies: 38
- Tool calls: 2

Total tokens: ~12500
Time range: 14:30 - 15:45
```

---

## 📁 Хранение данных

### Структура файлов

```
.qwen/
├── local/
│   └── default.json          # Старая система (совместимость)
└── sessions/
    └── default.json          # Event-based хранилище
```

### Формат хранилища

```json
{
  "session_id": "default",
  "events": [
    {
      "id": "20260322143000.000",
      "type": "user_message",
      "content": "привет!",
      "tokens": 10,
      "timestamp": "2026-03-22T14:30:00Z"
    },
    {
      "id": "20260322143001.000",
      "type": "assistant_reply",
      "content": "Привет! Чем могу помочь?",
      "tokens": 25,
      "timestamp": "2026-03-22T14:30:01Z"
    }
  ],
  "total_tokens": 35,
  "summary": "📝 Summary of 2 events...",
  "summary_from": 0,
  "created": "2026-03-22T14:30:00Z",
  "last_access": "2026-03-22T14:30:01Z"
}
```

---

## 🎛️ API

### SessionEventStore

| Метод | Описание |
|-------|----------|
| `AddEvent(type, content, metadata, tokens)` | Добавить событие |
| `GetEvents()` | Получить все события |
| `GetRecentEvents(n)` | Получить последние N событий |
| `GetEventsWithSummary()` | Получить с суммаризацией |
| `Compact(summary, keepLastN)` | Выполнить компaction |
| `Clear()` | Очистить хранилище |
| `GetTokenCount()` | Получить количество токенов |
| `GetEventCount()` | Получить количество событий |

### SessionStoreManager

| Метод | Описание |
|-------|----------|
| `GetStore(sessionID)` | Получить хранилище |
| `CreateStore(sessionID)` | Создать хранилище |
| `RemoveStore(sessionID)` | Удалить хранилище |
| `ListStores()` | Список всех хранилищ |
| `Cleanup(maxAge)` | Удалить старые хранилища |

### Compactor

| Метод | Описание |
|-------|----------|
| `ShouldCompact(store)` | Проверить необходимость |
| `Compact(store, generator)` | Выполнить компaction |
| `GetCompactionStatus(store)` | Получить статус |
| `AutoCompactWithTimer(store, generator, interval)` | Авто по таймеру |

---

## 🔧 Настройка

### Изменение параметров компaction

```go
// В agent.go
compactor := memory.NewCompactor(&memory.CompactionConfig{
    MaxEvents:   200,      // Увеличить лимит
    KeepLastN:   50,       // Сохранять больше событий
    AutoCompact: true,
})
```

### Отключение автосуммаризации

```go
compactor := memory.NewCompactor(&memory.CompactionConfig{
    AutoCompact: false,  // Отключить автоматическую компaction
})
```

### Кастомный генератор суммаризации

```go
func CustomSummaryGenerator(events []memory.Event) string {
    var sb strings.Builder
    sb.WriteString("📝 Custom Summary:\n\n")
    
    // Ваша логика суммаризации
    for _, event := range events {
        sb.WriteString(fmt.Sprintf("- %s: %s\n", event.Type, event.Content))
    }
    
    return sb.String()
}

// Использование
compactor.Compact(store, CustomSummaryGenerator)
```

---

## 🧪 Тестирование

### Проверка работы

```bash
# Запустить чат
./qwen-claw chat

# Отправить несколько сообщений
🔹> привет
🔹> как дела?
🔹> что ты умеешь?

# Проверить историю
🔹> /history

# Сжать вручную
🔹> /compact

# Очистить
🔹> /clear
```

### Проверка файлов

```bash
# Посмотреть хранилище
cat .qwen/sessions/default.json

# Проверить размер
ls -lh .qwen/sessions/
```

---

## 📈 Сравнение со старой системой

| Характеристика | Старая (localSession) | Новая (eventStore) |
|----------------|----------------------|-------------------|
| **Структура** | Массив строк | События с типами |
| **Метаданные** | Нет | Есть (tokens, timestamp) |
| **Суммаризация** | Нет | Автоматическая |
| **Компaction** | Нет | Есть |
| **Статус** | Нет | Визуальный (🟢🟡🟠🔴) |
| **Совместимость** | ✅ Сохранена | ✅ Fallback |

---

## 🐛 Отладка

### Включить debug логи

```bash
./qwen-claw chat --debug
```

### Проверить статус

```bash
# В чате
🔹> /history  # Показывает статус компaction
```

### Логи компaction

```
Compaction warning: failed to compact
✅ Session compacted successfully
```

---

## 📚 Ссылки

- [OpenClaw Session Architecture](https://github.com/openclaw/openclaw)
- [OpenClaw Memory System](https://docs.openclaw.ai/memory)

---

**Session Store** — персистентная история с автосуммаризацией 🚀
