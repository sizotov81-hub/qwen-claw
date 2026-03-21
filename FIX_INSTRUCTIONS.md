# Инструкция по исправлению кода

## ✅ Выполнено:

1. **Добавлен Zap logger** - `internal/logger/logger.go`
2. **Обновлён cmd/main.go** - использует logger вместо log

---

## 🔧 Требуется исправить:

### 1. Заменить log на logger в файлах:

```bash
# bots/telegram/bot.go
sed -i 's/"log"/"github.com\/user\/qwen-claw\/internal\/logger"/g' bots/telegram/bot.go
sed -i 's/log\.Printf/logger\.Infof/g' bots/telegram/bot.go
sed -i 's/log\.Println/logger\.Info/g' bots/telegram/bot.go

# internal/agent/agent.go
sed -i 's/"log"/"github.com\/user\/qwen-claw\/internal\/logger"/g' internal/agent/agent.go
sed -i 's/log\.Printf/logger\.Infof/g' internal/agent/agent.go

# internal/scheduler/scheduler.go
sed -i 's/"log"/"github.com\/user\/qwen-claw\/internal\/logger"/g' internal/scheduler/scheduler.go
sed -i 's/log\.Printf/logger\.Infof/g' internal/scheduler/scheduler.go
```

### 2. Исправить ошибки с %w:

**internal/scheduler/scheduler.go:**
```go
// Было:
return fmt.Errorf("task not found: %s", id)

// Стало:
return fmt.Errorf("remove task: %w", ErrTaskNotFound)
```

**internal/self-improvement/engine.go:**
```go
// Было:
return fmt.Errorf("change %s not found", changeID)

// Стало:
return fmt.Errorf("find change: %w", ErrChangeNotFound)
```

**internal/memory/search.go:**
```go
// Было:
return "", fmt.Errorf("no results found")

// Стало:
return "", fmt.Errorf("extract content: %w", ErrNoResults)
```

### 3. Удалить лишние комментарии:

```bash
# Найти комментарии-дубликаты
grep -rn "// [Зз]агружаем\|// [Сс]оздаём\|// [Пп]роверяем" --include="*.go" | head -20
```

**Примеры для удаления:**
```go
// Загружаем конфигурацию  ← УДАЛИТЬ
cfg, err := config.Load(cfgFile)

// Создаём директорию  ← УДАЛИТЬ
os.MkdirAll(dir, 0755)

// Проверяем токен  ← УДАЛИТЬ
if config.Token == "" {
```

### 4. Создать миграции:

```bash
mkdir -p /home/ss/qwen-claw/migrations
```

**20260321190000_create_memory_entries.sql:**
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS memory_entries (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    category TEXT,
    content TEXT NOT NULL,
    metadata JSON,
    importance REAL DEFAULT 0.5,
    retention REAL DEFAULT 1.0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    last_accessed INTEGER,
    access_count INTEGER DEFAULT 0,
    repetition_count INTEGER DEFAULT 0,
    half_life INTEGER DEFAULT 300,
    chunk_id TEXT,
    embeddings BLOB
);

-- +goose Down
DROP TABLE IF EXISTS memory_entries;
```

**20260321190001_create_chunks.sql:**
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS chunks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    entry_count INTEGER DEFAULT 0,
    last_used INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS chunks;
```

**20260321190002_create_nodes.sql:**
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    activation REAL DEFAULT 0.0,
    last_activated INTEGER
);

-- +goose Down
DROP TABLE IF EXISTS nodes;
```

**20260321190003_create_edges.sql:**
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS edges (
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    weight REAL DEFAULT 0.0,
    type TEXT NOT NULL,
    co_activation_count INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (source_id, target_id)
);

-- +goose Down
DROP TABLE IF EXISTS edges;
```

---

## 📋 Чеклист для будущего кода:

### Перед коммитом:

- [ ] Используется `logger` вместо `log`
- [ ] Ошибки обёрнуты с `%w`
- [ ] Нет очевидных комментариев
- [ ] Свои ошибки вынесены в `var Err...`
- [ ] Миграции названы по стандарту `YYYYMMDDHHMMSS_description.sql`

### Пример правильного кода:

```go
package user

import (
    "errors"
    "fmt"
    
    "github.com/user/qwen-claw/internal/logger"
)

var (
    ErrUserNotFound = errors.New("user not found")
    ErrUserExists   = errors.New("user already exists")
)

func (s *Service) GetUser(id string) (*User, error) {
    user, err := s.repo.Get(id)
    if err != nil {
        return nil, fmt.Errorf("get user from repo: %w", err)
    }
    
    if user == nil {
        return nil, ErrUserNotFound
    }
    
    logger.Debugw("user retrieved", "id", id, "name", user.Name)
    
    return user, nil
}
```

---

## 🚀 Автоматизация:

Создать `.golangci.yml`:
```yaml
linters:
  enable:
    - govet
    - staticcheck
    - errcheck
    - gosec
    - goconst
    - misspell
    - unconvert
    
linters-settings:
  govet:
    check-shadowing: true
  goconst:
    min-len: 3
    min-occurrences: 3
```

Запустить:
```bash
golangci-lint run ./...
```
