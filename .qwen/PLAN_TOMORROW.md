# 📅 План работы на завтра
## Дата: 22 марта 2026

---

## 🎯 ГЛАВНАЯ ЦЕЛЬ

**Джарвис должен самостоятельно выполнять действия через Qwen Code CLI**

Сейчас: Джарвис может только предлагать команды, но пользователь должен их вводить вручную.

Цель: Джарвис сам выполняет команды через Qwen Code CLI после подтверждения пользователя.

---

## 📊 ТЕКУЩЕЕ СОСТОЯНИЕ (контекст)

### ✅ Что уже работает:

1. **Планировщик задач (Scheduler)**
   - Файл: `internal/scheduler/scheduler.go`
   - Тесты: 18 тестов, покрытие 92%
   - Функции: AddTask, RemoveTask, EnableTask, DisableTask, RunTaskNow
   - Cron-выражения: `@daily`, `@hourly`, `*/5 * * * *`
   - Фоновый цикл: `checkAndRunTasks()` каждую минуту

2. **Система памяти (Memory)**
   - Файлы: `internal/memory/manager.go`, `search.go`, `background.go`
   - Тесты: 20 тестов, покрытие 90%
   - Уровни: Working (1мс), FTS (10мс), Associations (50мс), Web (100-500мс)
   - Базы данных: `.qwen/memory/memory.db`, `associations.db`
   - Фоновые процессы: consolidation, forgetting, restructuring

3. **Агент (Джарвис)**
   - Файл: `internal/agent/agent.go`
   - Тесты: 25 тестов, покрытие 87%
   - Личность: системный промпт в `.qwen/system-prompt.txt`
   - Поиск в памяти: `memory.Find()` перед каждым запросом
   - Антидеградация: `internal/agent/anti_degradation.go`

4. **Навыки (Skills)**
   - Файл: `internal/skills/engine.go`
   - Тесты: 15 тестов, покрытие 88%
   - Встроенные: shell, file, search, memory, git, http, notify
   - Выполнение: `engine.Execute(ctx, &SkillRequest{...})`

5. **Telegram Бот**
   - Файл: `bots/telegram/bot.go`
   - Тесты: 10 тестов, покрытие 82%
   - Команды: /start, /help, /status, /memory, /tasks, /skills
   - Обработка голосовых: `handleVoice()` (требует whisper.cpp)

6. **Web UI**
   - Файл: `internal/web/server.go`
   - Тесты: 10 тестов, покрытие 85%
   - Порт: 64656 (настраиваемый)
   - Аутентификация: JWT + Secret Phrase
   - WebSocket для реального времени

---

## ❌ ЧТО НЕ РАБОТАЕТ (проблемы)

### 1. **Агент не выполняет команды напрямую**

**Сейчас:**
```
Пользователь: "Создай файл test.txt"
Джарвис: "Выполните команду: echo 'content' > test.txt"
```

**Нужно:**
```
Пользователь: "Создай файл test.txt"
Джарвис: "⚠️ Создам файл test.txt. Подтвердите: ПОДТВЕРЖДАЮ"
Пользователь: "ПОДТВЕРЖДАЮ"
Джарвис: "✅ Выполняю... [executes via Qwen Code CLI]"
```

### 2. **Планировщик не интегрирован с агентом**

**Сейчас:**
- Планировщик выполняет задачи через `Executor` интерфейс
- Агент не знает о задачах планировщика

**Нужно:**
- Агент создаёт задачи в планировщике
- Планировщик вызывает агента для выполнения
- Агент отчитывается о выполнении

### 3. **Навыки не вызываются автоматически**

**Сейчас:**
```
Пользователь: "Запомни: проект использует Go 1.25"
Джарвис: "Используй команду: qwen-claw memory add '...'"
```

**Нужно:**
```
Пользователь: "Запомни: проект использует Go 1.25"
Джарвис: "✅ Запоминаю..." [вызывает skillEngine.Execute()]
```

---

## 🔧 ТЕХНИЧЕСКИЙ ПЛАН НА ЗАВТРА

### ЭТАП 1: Интеграция агента с Qwen Code CLI (2-3 часа)

**Задача:** Джарвис выполняет команды через Qwen Code CLI

**Файлы для изменения:**
- `internal/agent/agent.go`
- `internal/agent/executor.go` (новый файл)

**Реализация:**

1. **Создать Executor интерфейс:**
```go
type Executor interface {
    Execute(command string, args []string) (string, error)
    Confirm(action string) bool // запрос подтверждения
}
```

2. **Добавить в Agent:**
```go
type Agent struct {
    // ... existing fields
    executor      Executor
    approvalMode  string // "auto-edit", "plan", "yolo"
}
```

3. **Реализовать QwenExecutor:**
```go
type QwenExecutor struct {
    qwenPath string
    mode     string
}

func (e *QwenExecutor) Execute(cmd string, args []string) (string, error) {
    // Запуск qwen cli с аргументами
    // --approval-mode из конфига
}
```

4. **Обновить Agent.Run():**
```go
func (a *Agent) Run(ctx context.Context, query string) (string, error) {
    // 1. Поиск в памяти
    // 2. Анализ запроса
    // 3. Если команда → Execute()
    // 4. Если вопрос → Qwen CLI
}
```

**Тесты:**
- `internal/agent/executor_test.go`
- Тесты на подтверждение действий
- Тесты на разные approval modes

---

### ЭТАП 2: Интеграция планировщика с агентом (2 часа)

**Задача:** Планировщик вызывает агента для выполнения задач

**Файлы для изменения:**
- `internal/scheduler/scheduler.go`
- `internal/agent/agent.go`

**Реализация:**

1. **Scheduler.Executor через агента:**
```go
type AgentExecutor struct {
    agent *Agent
}

func (e *AgentExecutor) Execute(ctx context.Context, command string) (string, error) {
    return e.agent.Run(ctx, command)
}
```

2. **Агент создаёт задачи:**
```go
func (a *Agent) ScheduleTask(name, schedule, command string) (*Task, error) {
    return a.scheduler.AddTask(name, "", command, schedule)
}
```

3. **Команды для агента:**
```
"Запланируй задачу backup на @daily с командой 'сделай бэкап'"
→ agent.ScheduleTask("backup", "@daily", "сделай бэкап")
```

**Тесты:**
- `internal/scheduler/agent_integration_test.go`
- Тесты на создание задач агентом
- Тесты на выполнение задач планировщиком

---

### ЭТАП 3: Автоматический вызов навыков (1-2 часа)

**Задача:** Агент автоматически вызывает навыки при распознавании команд

**Файлы для изменения:**
- `internal/agent/agent.go`
- `internal/skills/engine.go`

**Реализация:**

1. **Intent Detection в агенте:**
```go
func (a *Agent) detectIntent(query string) *Intent {
    // Анализ запроса
    // "запомни X" → Intent{Type: "memory", Action: "add", Data: X}
    // "найди X" → Intent{Type: "search", Action: "search", Data: X}
}
```

2. **Выполнение навыков:**
```go
func (a *Agent) executeIntent(intent *Intent) (string, error) {
    skill := a.skillEngine.FindSkillByCommand(intent.Type)
    if skill != nil {
        return a.skillEngine.Execute(ctx, &SkillRequest{
            Command: intent.Type,
            Args:    []string{intent.Data},
        })
    }
}
```

3. **Обновление Run():**
```go
func (a *Agent) Run(query string) (string, error) {
    // 1. Поиск в памяти
    // 2. Intent Detection
    // 3. Если навык → executeIntent()
    // 4. Иначе → Qwen CLI
}
```

**Тесты:**
- `internal/agent/intent_test.go`
- Тесты на распознавание команд
- Тесты на выполнение навыков

---

### ЭТАП 4: Система подтверждений (1-2 часа)

**Задача:** Безопасное выполнение действий с подтверждением

**Файлы для изменения:**
- `internal/agent/agent.go`
- `internal/agent/confirmation.go` (новый файл)

**Реализация:**

1. **Confirmation Manager:**
```go
type ConfirmationManager struct {
    pending map[string]*PendingAction
    mode    string // "plan", "auto-edit", "yolo"
}

type PendingAction struct {
    ID        string
    Command   string
    Created   time.Time
    Expires   time.Time
}
```

2. **Режимы подтверждения:**
```go
const (
    ModePlan     = "plan"     // Всегда спрашивать
    ModeAutoEdit = "auto-edit" // Спрашивать для опасных действий
    ModeYolo     = "yolo"     // Никогда не спрашивать
)
```

3. **Интеграция с агентом:**
```go
func (a *Agent) Run(query string) (string, error) {
    intent := a.detectIntent(query)
    
    if a.requiresConfirmation(intent) {
        action := a.confirmation.CreatePending(intent)
        return fmt.Sprintf("⚠️ Требуется подтверждение. Напишите: ПОДТВЕРЖДАЮ %s", action.ID)
    }
    
    return a.executeIntent(intent)
}
```

**Тесты:**
- `internal/agent/confirmation_test.go`
- Тесты на разные режимы
- Тесты на истечение подтверждений

---

### ЭТАП 5: Интеграция с Telegram и Web UI (2 часа)

**Задача:** Подтверждения через Telegram и Web UI

**Файлы для изменения:**
- `bots/telegram/bot.go`
- `internal/web/server.go`

**Реализация:**

1. **Telegram кнопки подтверждения:**
```go
func (b *Bot) handleConfirmation(msg *tgbotapi.Message, actionID string) {
    // Inline кнопки: [✅ Подтвердить] [❌ Отменить]
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("✅", "confirm:"+actionID),
            tgbotapi.NewInlineKeyboardButtonData("❌", "cancel:"+actionID),
        ),
    )
}
```

2. **Web UI подтверждения:**
```go
// GET /api/confirmations
func (s *Server) handleGetConfirmations(w http.ResponseWriter, r *http.Request) {
    pending := s.agent.GetPendingConfirmations()
    s.sendJSON(w, APIResponse{Success: true, Data: pending})
}

// POST /api/confirmations/{id}
func (s *Server) handleConfirmAction(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    action := r.URL.Query().Get("action") // confirm/cancel
    s.agent.ProcessConfirmation(id, action)
}
```

**Тесты:**
- `bots/telegram/confirmation_test.go`
- `internal/web/confirmation_test.go`

---

## 📁 НОВЫЕ ФАЙЛЫ (создать завтра)

```
internal/agent/
├── executor.go           # Executor интерфейс + QwenExecutor
├── executor_test.go      # Тесты executor
├── intent.go             # Intent Detection
├── intent_test.go        # Тесты intent
├── confirmation.go       # Confirmation Manager
└── confirmation_test.go  # Тесты confirmation

internal/scheduler/
└── agent_integration_test.go  # Интеграция с агентом
```

---

## 🧪 ТЕСТЫ (написать завтра)

### Обязательные:
- [ ] `TestQwenExecutor_Execute`
- [ ] `TestAgent_ScheduleTask`
- [ ] `TestAgent_detectIntent`
- [ ] `TestConfirmationManager_CreatePending`
- [ ] `TestIntegration_SchedulerAgent`

### Интеграционные:
- [ ] `TestFullWorkflow_ScheduleAndExecute`
- [ ] `TestFullWorkflow_ConfirmationFlow`
- [ ] `TestFullWorkflow_SkillExecution`

---

## ⚠️ ИЗВЕСТНЫЕ ПРОБЛЕМЫ (решить завтра)

### 1. Go mod tidy проблема

**Симптом:**
```
go mod tidy fails:
module github.com/user/qwen-claw/internal/selfimprovement: 
git ls-remote failed: Repository not found
```

**Причина:** Go пытается найти локальные пакеты как внешние репозитории.

**Решение:**
```bash
# Вариант 1: Закоммитить go.mod/go.sum
git add go.mod go.sum
git commit -m "fix: update go.mod"

# Вариант 2: Использовать replace directive
# go.mod:
replace github.com/user/qwen-claw/internal/selfimprovement => ./internal/self-improvement
```

### 2. Запуск тестов

**Проблема:** `go mod tidy` требуется, но не работает.

**Временное решение:**
```bash
# Запуск тестов по пакетам без go mod tidy
CGO_ENABLED=0 go test -v ./internal/logger/...
CGO_ENABLED=0 go test -v ./internal/scheduler/...
```

---

## 🎯 КРИТЕРИИ ГОТОВНОСТИ (Definition of Done)

### ЭТАП 1 (Executor):
- [ ] QwenExecutor реализован
- [ ] Agent использует Executor
- [ ] Тесты написаны (покрытие >85%)
- [ ] Approval modes работают

### ЭТАП 2 (Scheduler Integration):
- [ ] Agent создаёт задачи в планировщике
- [ ] Планировщик вызывает Agent для выполнения
- [ ] Тесты интеграции написаны
- [ ] Задачи выполняются по расписанию

### ЭТАП 3 (Skills):
- [ ] Intent Detection работает
- [ ] Навыки вызываются автоматически
- [ ] Тесты на распознавание команд
- [ ] Покрытие >85%

### ЭТАП 4 (Confirmation):
- [ ] Confirmation Manager реализован
- [ ] 3 режима работают (plan, auto-edit, yolo)
- [ ] Тесты на все режимы
- [ ] Истечение подтверждений работает

### ЭТАП 5 (Integration):
- [ ] Telegram кнопки подтверждения
- [ ] Web UI подтверждения
- [ ] Тесты интеграции
- [ ] Документация обновлена

---

## 📚 КОНТЕКСТ (важно знать)

### Архитектура проекта:
```
qwen-claw/
├── cmd/main.go              # Точка входа CLI
├── internal/
│   ├── agent/               # Джарвис (личность, память, антидеградация)
│   ├── scheduler/           # Планировщик задач
│   ├── memory/              # Система памяти (4 уровня)
│   ├── skills/              # Навыки (8 встроенных)
│   ├── web/                 # Web UI (JWT auth)
│   ├── logger/              # Zap logger
│   ├── config/              # Конфигурация
│   ├── voice/               # Voice recognition (whisper)
│   └── self-improvement/    # Самосовершенствование
├── bots/telegram/           # Telegram бот
├── .qwen/
│   ├── memory/              # Базы данных памяти
│   ├── scheduler/           # Задачи планировщика
│   ├── system-prompt.txt    # Личность Джарвиса
│   └── .web-secrets.json    # Секреты Web UI
└── go.mod/go.sum
```

### Ключевые зависимости:
```go
github.com/spf13/cobra      // CLI
github.com/spf13/viper      // Конфигурация
github.com/gorilla/websocket // WebSocket
github.com/golang-jwt/jwt/v5 // JWT
go.uber.org/zap             // Логирование
modernc.org/sqlite          // SQLite (pure Go)
github.com/go-telegram-bot-api/telegram-bot-api/v5 // Telegram
```

### Переменные окружения:
```bash
QWEN_CLAW_TELEGRAM_TOKEN="бот токен"
QWEN_CLAW_TELEGRAM_ALLOWED_USERS="123,456"
QWEN_CLAW_MODEL=""
QWEN_CLAW_APPROVAL_MODE="auto-edit"
QWEN_CLAW_DEBUG="false"
```

### Секретная фраза Web UI:
```
90d68743c3f7608d682698ad0f547622173ff97c4251fe34e75666ad502d3ae4
```
(хранится в `.web-secrets.json`)

---

## 🚀 БЫСТРЫЙ СТАРТ (для продолжения)

```bash
# 1. Перейти в проект
cd /home/ss/qwen-claw

# 2. Проверить сборку
CGO_ENABLED=0 go build -o qwen-claw ./cmd/main.go

# 3. Запустить тесты (по пакетам)
CGO_ENABLED=0 go test -v ./internal/logger/...
CGO_ENABLED=0 go test -v ./internal/scheduler/...

# 4. Начать ЭТАП 1
# Создать internal/agent/executor.go
# Реализовать Executor интерфейс
```

---

## 📞 КОНТАКТЫ (если что-то непонятно)

### Файлы с контекстом:
- `.qwen/DEV_CONTEXT.md` — полная документация
- `COVERAGE_REPORT.md` — отчёт о покрытии тестами
- `TESTING.md` — руководство по тестированию
- `AUDIT.md` — аудит кода
- `REFACTORING_COMPLETE.md` — отчёт о рефакторинге

### GitHub:
- https://github.com/sizotov81-hub/qwen-claw
- Ветка: `dev`
- Последний коммит: `fce5c7a`

---

## ✅ ЧЕКЛИСТ НАЧАЛА РАБОТЫ

- [ ] Прочитать этот план
- [ ] Проверить сборку проекта
- [ ] Запустить тесты для проверки состояния
- [ ] Начать с ЭТАПА 1 (Executor)
- [ ] После каждого этапа запускать тесты
- [ ] Коммитить после каждого завершённого этапа

---

**Удачи завтра! 🚀**

*Следующий шаг: создать `internal/agent/executor.go` и реализовать Executor интерфейс.*
