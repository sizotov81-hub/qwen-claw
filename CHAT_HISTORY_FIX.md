# Исправление истории чата в qwen-claw

## 📋 Проблема

При использовании `qwen-claw chat` история диалога **не сохранялась** между сессиями и **не передавалась** в Qwen CLI.

### Симптомы из диалога:
- Пользователь спрашивает: "Ты видишь историю этого чата, что я тебе писал перед этим и твои ответы?"
- Агент отвечает, что видит только текущую сессию, но история не сохраняется
- Файл `.qwen/local/default.json` показывает `"conversation_history": []` — пустая история

## 🔍 Причины проблемы

### 1. Неправильный путь к локальной сессии
**Файл:** `internal/agent/agent.go` (строка 127)

**Было:**
```go
localSessionDir := filepath.Join(os.Getenv("HOME"), "qwen-claw", ".qwen", "local")
```

**Проблема:** Путь жёстко закодирован как `~/qwen-claw/.qwen/local`, но проект может находиться в другом месте (например, `/home/ss/qwen-claw`).

### 2. Неподдерживаемый флаг --context
**Файл:** `internal/agent/agent.go` (метод `buildCommandArgs`)

**Было:**
```go
if len(a.conversationHistory) > 0 {
    history := a.getRecentHistory(10)
    if history != "" {
        args = append(args, "--context", history)
    }
}
```

**Проблема:** Qwen Code CLI **не поддерживает** флаг `--context`. Проверка через `qwen --help` показала, что такого флага нет.

### 3. Отсутствие команды для просмотра истории
В режиме chat не было команды `/history` для просмотра текущей сессии.

## ✅ Выполненные исправления

### 1. Исправлен путь к локальной сессии

**Файл:** `internal/agent/agent.go`

**Стало:**
```go
// Используем относительный путь от рабочей директории
workingDir, _ := os.Getwd()
localSessionDir := filepath.Join(workingDir, ".qwen", "local")
sessionManager, err := memory.NewLocalSessionManager(localSessionDir)
```

**Дополнительно:** Исправлен метод `StartNewSession()` (строка 523):
```go
workingDir, _ := os.Getwd()
sessionManager, err := memory.NewLocalSessionManager(filepath.Join(workingDir, ".qwen", "local"))
```

### 2. Убран неподдерживаемый флаг --context

**Файл:** `internal/agent/agent.go` (метод `buildCommandArgs`)

**Стало:**
```go
// Qwen CLI не поддерживает --context флаг, поэтому история передаётся
// через локальную сессию и сохраняется автоматически через --chat-recording
// при использовании интерактивного режима

// Добавляем запрос как позиционный аргумент
args = append(args, query)
```

### 3. Добавлена команда /history

**Файл:** `cmd/main.go` (функция `runChat`)

**Новая команда:**
```go
case "/history":
    history := agentInstance.GetSessionHistory()
    if len(history) == 0 {
        fmt.Println("📋 История сессии пуста")
    } else {
        fmt.Printf("📋 История сессии (%d записей):\n\n", len(history))
        for i, msg := range history {
            if i >= 20 { // Показываем последние 20 сообщений
                fmt.Printf("... и ещё %d записей\n", len(history)-20)
                break
            }
            if strings.HasPrefix(msg, "User:") {
                fmt.Printf("👤 %s\n", msg[5:])
            } else if strings.HasPrefix(msg, "Assistant:") {
                fmt.Printf("🤖 %s\n", msg[10:])
            } else {
                fmt.Printf("   %s\n", msg)
            }
        }
    }
```

### 4. Улучшены сообщения в других командах

- `/clear` → "✅ История сессии очищена"
- `/memory` → "📚 Память пуста" / "📚 Память (N записей):"
- `/context` → "📭 Контекст пуст"
- `/help` → добавлена строка про `/history`

## 📁 Изменённые файлы

| Файл | Изменения |
|------|-----------|
| `internal/agent/agent.go` | Исправлен путь к сессии, убран флаг --context |
| `cmd/main.go` | Добавлена команда /history, улучшены сообщения |

## 🧪 Как тестировать

```bash
cd /home/ss/qwen-claw

# Запустить чат
./qwen-claw chat

# В чате:
🔹> привет
🔹> как дела?
🔹> /history        # Показать историю сессии
🔹> /memory         # Показать память
🔹> /context        # Показать контекст
🔹> /clear          # Очистить историю
🔹> /exit           # Выйти

# Проверить файл истории
cat .qwen/local/default.json
```

## 📊 Ожидаемый результат

После исправлений:
1. ✅ История сохраняется в `.qwen/local/default.json`
2. ✅ Команда `/history` показывает текущую сессию
3. ✅ История сохраняется между перезапусками в пределах одной сессии
4. ✅ Путь к сессии правильный (относительный от проекта)

## ⚠️ Известные ограничения

1. **Qwen CLI не поддерживает внешний контекст:** Флаг `--context` не существует в Qwen Code CLI. История сохраняется только локально в файле `.qwen/local/default.json`.

2. **История не передаётся в Qwen:** При использовании `qwen-claw chat` каждый запрос отправляется в Qwen CLI без истории диалога. Qwen CLI не "помнит" предыдущие сообщения.

3. **Для полной истории используйте Qwen CLI напрямую:** Если нужна полная история в Qwen, используйте:
   ```bash
   qwen -i  # Интерактивный режим с --chat-recording
   ```

## 🔧 Рекомендации на будущее

1. **Добавить суммаризацию истории:** При достижении лимита контекста автоматически суммаризировать старые сообщения.

2. **Экспорт/импорт истории:** Добавить команды для экспорта истории в файл и импорта из файла.

3. **Несколько сессий:** Поддержка переключения между разными сессиями (`/session new`, `/session load <name>`).

4. **Поиск по истории:** Команда `/search <query>` для поиска по истории сессии.

---

**Дата исправления:** 2026-03-22  
**Исполнитель:** Qwen Code (Jarvis persona)
