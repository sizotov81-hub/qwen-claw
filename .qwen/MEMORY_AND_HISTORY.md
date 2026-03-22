# 🧠 Память и история диалогов Qwen-Claw

## 📋 Типы памяти

### 1. **Персистентная память (Memory Manager)**
- **Расположение:** `.qwen/memory/`
- **Тип:** SQLite базы данных
- **Назначение:** Долгосрочное хранение фактов, ассоциаций, контекста
- **Сохранение:** Автоматически после каждого запроса

### 2. **История диалогов (LocalSession)**
- **Расположение:** `.qwen/local/*.json`
- **Тип:** JSON файлы
- **Назначение:** История переписки в сессии
- **Сохранение:** После каждого сообщения (каждые 10 сообщений — на диск)

### 3. **Оперативная память (RAM)**
- **Расположение:** В памяти процесса
- **Тип:** Массивы Go
- **Назначение:** Быстрый доступ к последним сообщениям
- **Сохранение:** Не сохраняется (очищается при выходе)

---

## 🔧 Как это работает

### При запуске агента:

```go
// 1. Создаётся LocalSessionManager
localSessionDir := filepath.Join(os.Getenv("HOME"), "qwen-claw", ".qwen", "local")
sessionManager, _ := memory.NewLocalSessionManager(localSessionDir)

// 2. Загружается сессия "default"
localSession, _ := sessionManager.GetSession("default")

// 3. История загружается в RAM кэш
conversationHistory = localSession.GetHistory()
```

### При каждом запросе:

```go
// 1. Сохранение в RAM (быстро)
a.conversationHistory = append(..., "User: ...")

// 2. Сохранение в LocalSession (персистентно)
a.localSession.AddToHistory("User: ...")

// 3. Сохранение в Memory Manager (SQLite)
a.memoryManager.AddMessage("user", "...")
```

### При чтении истории:

```go
// Приоритет: персистентная > RAM
func (a *Agent) getRecentHistory(n int) string {
    if a.localSession != nil {
        recent := a.localSession.GetRecentHistory(n)
        return strings.Join(recent, "\n")
    }
    return strings.Join(a.conversationHistory[start:], "\n")
}
```

---

## 📁 Структура файлов

```
.qwen/
├── local/                    # ← ПРИВАТНОЕ (не в git)
│   ├── default.json          # Сессия по умолчанию
│   └── session-123.json      # Другие сессии
├── memory/                   # ← ПРИВАТНОЕ (не в git)
│   ├── memory.db             # Основная база
│   └── associations.db       # Ассоциации
└── scheduler/                # ← ПРИВАТНОЕ (не в git)
    └── tasks.json            # Задачи
```

**Все файлы в `.qwen/` исключены из git через `.gitignore`!**

---

## 🔒 Безопасность

### Что НЕ попадает в git:

```gitignore
# .gitignore
.qwen/memory/
.qwen/local/
.qwen/*.db
.qwen/scheduler/
.qwen/anti-degradation/*.json
.env
```

### Что МОЖЕТ быть в git:

```
.qwen/*.txt          # Системные промпты
.qwen/*.md           # Документация
.qwen/CONSTITUTION.md
.qwen/system-prompt.txt
```

---

## 🚀 Использование

### Получить историю сессии:

```go
history := agent.GetSessionHistory()
for _, msg := range history {
    fmt.Println(msg)
}
```

### Очистить историю:

```go
agent.ClearSessionHistory()
```

### Последние N сообщений:

```go
recent := agent.getRecentHistory(10)
// Передаётся в Qwen CLI как --context
```

---

## 📊 Сравнение

| Характеристика | Memory Manager | LocalSession | RAM |
|---------------|----------------|--------------|-----|
| **Хранение** | SQLite | JSON | Массив |
| **Персистентность** | ✅ Да | ✅ Да | ❌ Нет |
| **Скорость** | Средняя | Высокая | Очень высокая |
| **Назначение** | Факты, ассоциации | История диалогов | Кэш |
| **Размер** | Неограничен | ~1000 сообщений | ~10 сообщений |

---

## 🛠️ Администрирование

### Просмотр сессий:

```bash
ls -la .qwen/local/
cat .qwen/local/default.json | jq
```

### Очистка старых сессий:

```go
manager.Cleanup(24 * time.Hour) // Удалить старше 24 часов
```

### Резервное копирование:

```bash
# Сохранить сессии
cp -r .qwen/local/ ~/backup/qwen-sessions/

# Сохранить память
cp .qwen/memory/*.db ~/backup/qwen-memory/
```

---

## ⚠️ Важно

1. **Приватность:** Все файлы в `.qwen/local/` приватные, не коммитьте в git
2. **Производительность:** Сохранение каждые 10 сообщений для баланса скорость/надёжность
3. **Контекст:** Qwen CLI получает последние 10 сообщений через `--context`
4. **Сессии:** По умолчанию используется "default", можно создать именованные сессии

---

**Версия:** 1.0.0  
**Обновлено:** 2026-03-22
