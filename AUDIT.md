# Code Audit Report

## Дата: 21 марта 2026

---

## ✅ Что соответствует стандартам:

1. **Структура проекта** - чистая архитектура (internal/, cmd/, bots/)
2. **Разделение ответственности** - каждый пакет отвечает за своё
3. **Интерфейсы** - используются где нужно

---

## ❌ Проблемы:

### 1. Логирование (КРИТИЧНО)

**Проблема:** Используется стандартный `log` вместо `zap`

**Файлы:**
- cmd/main.go (исправлено ✓)
- bots/telegram/bot.go
- internal/agent/agent.go
- internal/scheduler/scheduler.go

**Решение:** Заменить на `logger` пакет

---

### 2. Обработка ошибок

**Проблема:** `fmt.Errorf` без `%w` для обёртывания

**Примеры:**
```go
// ПЛОХО
return fmt.Errorf("task not found: %s", id)

// ХОРОШО
return fmt.Errorf("task not found: %w", err)
```

**Файлы для исправления:**
- internal/scheduler/scheduler.go (5 мест)
- internal/self-improvement/engine.go (4 места)
- internal/memory/search.go (1 место)
- cmd/main.go (多处)

---

### 3. Лишние комментарии

**Проблема:** Комментарии очевидного кода

**Примеры:**
```go
// Создаём директорию  ← очевидно из кода
os.MkdirAll(dir, 0755)

// Загружаем конфигурацию  ← очевидно из функции
cfg, err := config.Load(cfgFile)
```

**Удалить в:**
- internal/memory/manager.go
- internal/scheduler/scheduler.go
- internal/agent/agent.go

---

### 4. Файлы миграций

**Проблема:** Нет миграций для SQLite

**Решение:** Создать файлы миграций с naming:
```
migrations/
├── 20260321190000_create_memory_entries.sql
├── 20260321190001_create_chunks.sql
├── 20260321190002_create_nodes.sql
└── 20260321190003_create_edges.sql
```

---

### 5. Конфигурация

**Проблема:** Нет единого конфига для всех компонентов

**Решение:** Создать `config.yaml` с секциями:
```yaml
logger:
  level: info
  format: json

database:
  memory: .qwen/memory/memory.db
  associations: .qwen/memory/associations.db

telegram:
  enabled: true
  token: "..."
  
web:
  enabled: true
  host: 127.0.0.1
  port: 64656
```

---

## 📋 План исправлений:

### Приоритет 1 (Критично):
1. ✅ Добавить zap logger
2. ⏳ Исправить ошибки с %w
3. ⏳ Удалить лишние комментарии

### Приоритет 2 (Важно):
4. ⏳ Создать миграции БД
5. ⏳ Обновить конфигурацию

### Приоритет 3 (Желательно):
6. ⏳ Добавить линтеры (golangci-lint)
7. ⏳ Настроить pre-commit hooks

---

## 🎯 Стандарты для будущего кода:

### Логирование:
```go
import "github.com/user/qwen-claw/internal/logger"

logger.Infof("User %s logged in", userID)
logger.Errorf("Failed to process request: %w", err)
logger.Debugw("Request details", "method", r.Method, "path", r.URL.Path)
```

### Ошибки:
```go
// Обёртывание
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// Проверка
if errors.Is(err, sql.ErrNoRows) {
    return ErrNotFound
}

// Свои ошибки
var ErrNotFound = errors.New("not found")
```

### Комментарии:
```go
// ПЛОХО:
// Устанавливаем соединение  ← очевидно
db, err := sql.Open("sqlite", path)

// ХОРОШО:
// Используем connection pool для предотвращения утечек
// См. https://github.com/golang/go/issues/20103
db.SetMaxOpenConns(25)
```

### Имена файлов:
```
✅ 20260321190000_create_memory_entries.sql
❌ create_memory.sql
❌ 001_create_memory.sql
```
