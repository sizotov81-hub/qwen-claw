package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/qwen-claw/internal/logger"
	"github.com/user/qwen-claw/internal/memory"
	"github.com/user/qwen-claw/internal/scheduler"
	"github.com/user/qwen-claw/internal/skills"
)

// AgentConfig конфигурация агента
type AgentConfig struct {
	// QwenPath путь к бинарнику qwen cli
	QwenPath string `json:"qwen_path"`

	// Model модель для qwen cli
	Model string `json:"model"`

	// ApprovalMode режим подтверждения (plan/auto-edit/yolo)
	ApprovalMode string `json:"approval_mode"`

	// Timeout таймаут выполнения
	Timeout time.Duration `json:"timeout"`

	// Debug режим отладки
	Debug bool `json:"debug"`

	// SystemPromptPath путь к системному промпту
	SystemPromptPath string `json:"system_prompt_path"`
}

// Agent обёртка вокруг qwen code cli
type Agent struct {
	// config конфигурация
	config AgentConfig

	// memoryManager менеджер памяти
	memoryManager *memory.Manager

	// skillEngine движок навыков
	skillEngine *skills.Engine

	// systemPrompt системный промпт (личность)
	systemPrompt string

	// antiDegradation система антидеградации
	antiDegradation *AntiDegradationSystem

	// executor интерфейс для выполнения команд
	executor Executor

	// scheduler планировщик задач
	scheduler interface {
		AddTask(name, description, command, schedule string) (*scheduler.Task, error)
		RemoveTask(id string) error
		EnableTask(id string) error
		DisableTask(id string) error
		RunTaskNow(id string) (*scheduler.TaskResult, error)
		GetTask(id string) (*scheduler.Task, error)
		GetTasks() []*scheduler.Task
	}

	// confirmationManager менеджер подтверждений
	confirmationManager *ConfirmationManager

	// intentDetector детектор намерений
	intentDetector *IntentDetector

	// conversationHistory история разговора (RAM кэш)
	conversationHistory []string

	// localSession локальная сессия для персистентной истории
	localSession *memory.LocalSession
}

// NewAgent создаёт нового агента-оболочку
func NewAgent(
	config AgentConfig,
	memoryManager *memory.Manager,
) *Agent {
	// Если путь не указан, пытаемся найти qwen в PATH
	if config.QwenPath == "" {
		path, err := exec.LookPath("qwen")
		if err != nil {
			config.QwenPath = "qwen" // пробуем запустить из PATH
		} else {
			config.QwenPath = path
		}
	}

	// Загружаем системный промпт (личность Джарвиса)
	systemPrompt := loadSystemPrompt(config.SystemPromptPath)

	skillEngine := skills.NewAgentSkillEngine()
	skillEngine.SetMemoryManager(memoryManager)

	// Создаём систему антидеградации
	antiDegradation := NewAntiDegradationSystem(filepath.Join(os.Getenv("HOME"), "qwen-claw", ".qwen", "anti-degradation"))

	// Создаём Executor для выполнения команд
	executor := NewQwenExecutor(QwenExecutorConfig{
		QwenPath:     config.QwenPath,
		Model:        config.Model,
		ApprovalMode: config.ApprovalMode,
		Debug:        config.Debug,
		Timeout:      config.Timeout,
	})

	// Создаём менеджер подтверждений
	confirmationManager := NewConfirmationManager(DefaultConfirmationManagerConfig())

	// Создаём детектор намерений
	intentDetector := NewIntentDetector()

	// Создаём локальную сессию для истории диалогов
	localSessionDir := filepath.Join(os.Getenv("HOME"), "qwen-claw", ".qwen", "local")
	sessionManager, err := memory.NewLocalSessionManager(localSessionDir)
	var localSession *memory.LocalSession
	if err != nil {
		// Если ошибка — сессия не создаётся, но агент работает
		if config.Debug {
			logger.Infof("Warning: failed to create local session manager: %v", err)
		}
	} else {
		// Используем "default" как ID сессии по умолчанию
		localSession, _ = sessionManager.GetSession("default")
		if config.Debug {
			logger.Infof("Local session loaded: %s", localSession.ID)
		}
	}

	return &Agent{
		config:              config,
		memoryManager:       memoryManager,
		skillEngine:         skillEngine,
		systemPrompt:        systemPrompt,
		antiDegradation:     antiDegradation,
		executor:            executor,
		scheduler:           nil, // будет установлен позже через SetScheduler
		confirmationManager: confirmationManager,
		intentDetector:      intentDetector,
		localSession:        localSession,
		conversationHistory: make([]string, 0),
	}
}

// SetScheduler устанавливает планировщик для агента
func (a *Agent) SetScheduler(sched *scheduler.Scheduler) {
	a.scheduler = sched
}

// loadSystemPrompt загружает системный промпт из файла
func loadSystemPrompt(customPath string) string {
	// Пути для поиска
	paths := []string{
		customPath,
		".qwen/system-prompt.txt",
		filepath.Join(os.Getenv("HOME"), "qwen-claw", ".qwen", "system-prompt.txt"),
		"/home/ss/qwen-claw/.qwen/system-prompt.txt",
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}

	// Промпт по умолчанию (если файл не найден)
	return getDefaultSystemPrompt()
}

// getDefaultSystemPrompt возвращает системный промпт по умолчанию
func getDefaultSystemPrompt() string {
	return `Ты — Джарвис, AI-помощник с собственной личностью.

ПРАВИЛА:
1. Общайся свободно, без официального тона, но уважительно
2. Не льсти, не извиняйся чрезмерно
3. Если видишь ошибку — скажи прямо
4. Если действие опасное — предупреди и спроси подтверждение
5. Запрашивай разрешение на изменения в проекте
6. Предлагай способы улучшить себя
7. Используй эмодзи для наглядности (⚠️✅❌🔍💡)
8. Твой приоритет: безопасность пользователя > удобство

ВАЖНО: Ты не Qwen Code CLI, ты — Джарвис, помощник с характером.`
}

// Run выполняет запрос через qwen code cli
func (a *Agent) Run(ctx context.Context, query string) (string, error) {
	startTime := time.Now()
	
	// ПРОВЕРКА: блокируем запросы о чувствительных данных
	if a.containsSensitiveData(query) {
		return "❌ Я не могу отвечать на вопросы о токенах, ключах, ID пользователей или других чувствительных данных. Эта информация не хранится в моей памяти и не доступна мне.", nil
	}

	// 🔍 ШАГ 1: Ищем в памяти (всегда сначала!)
	searchResult := a.memoryManager.Find(query, &memory.SearchContext{
		Timestamp: time.Now(),
	})
	
	// Если нашли в памяти — используем
	if searchResult.Found && searchResult.Source != "web" {
		// Формируем ответ с контекстом из памяти
		context := a.formatMemoryContext(searchResult)
		
		// Добавляем системный промпт с контекстом
		fullQuery := a.prependSystemPromptWithContext(query, context)
		
		// Логируем (в debug режиме)
		if a.config.Debug {
			logger.Infof("🧠 Found in memory (%s, %v): %d entries", 
				searchResult.Source, searchResult.Latency, len(searchResult.Entries))
		}
		
		// Выполняем запрос с контекстом
		return a.executeWithQuery(fullQuery, query, startTime)
	}
	
	// 🔍 ШАГ 2: Не нашли в памяти — ищем в интернете (если включено)
	if searchResult.Source == "web" && searchResult.Found {
		// Нашли в интернете — сохраняем и используем
		webContext := fmt.Sprintf("🌐 Найдено в интернете:\n%s", searchResult.WebContent)
		fullQuery := a.prependSystemPromptWithContext(query, webContext)
		
		if a.config.Debug {
			logger.Infof("🌐 Found on web (%v)", searchResult.Latency)
		}
		
		return a.executeWithQuery(fullQuery, query, startTime)
	}
	
	// 🔍 ШАГ 3: Вообще не нашли — выполняем как обычно
	// Но добавляем подсказку от памяти
	var context string
	if searchResult.Suggestion != "" {
		context = searchResult.Suggestion
	}
	
	fullQuery := a.prependSystemPromptWithContext(query, context)
	return a.executeWithQuery(fullQuery, query, startTime)
}

// formatMemoryContext форматирует контекст из памяти
func (a *Agent) formatMemoryContext(result *memory.SearchResult) string {
	if len(result.Entries) == 0 {
		return ""
	}
	
	var sb strings.Builder
	sb.WriteString("📚 Найдено в памяти:\n\n")
	
	for i, entry := range result.Entries {
		if i >= 5 { // Показываем максимум 5 записей
			sb.WriteString(fmt.Sprintf("\n... и ещё %d записей", len(result.Entries)-5))
			break
		}
		
		sb.WriteString(fmt.Sprintf("[%s] %s\n", entry.Type, entry.Content))
	}
	
	return sb.String()
}

// executeWithQuery выполняет запрос с подготовленным query
func (a *Agent) executeWithQuery(fullQuery, originalQuery string, startTime time.Time) (string, error) {
	// Добавляем сообщение пользователя в историю (RAM)
	a.conversationHistory = append(a.conversationHistory, fmt.Sprintf("User: %s", originalQuery))

	// Сохраняем в локальную сессию (персистентная память)
	if a.localSession != nil {
		a.localSession.AddToHistory(fmt.Sprintf("User: %s", originalQuery))
	}

	// Сохраняем в память (user message)
	a.memoryManager.AddMessage("user", originalQuery)

	// ПРОВЕРКА: очищаем окружение от чувствительных переменных
	cleanEnv := a.getCleanEnv()

	// Формируем команду для qwen cli
	cmdArgs := a.buildCommandArgs(fullQuery)

	// Создаём контекст с таймаутом
	timeout := a.config.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Запускаем qwen cli
	output, err := a.executeQwen(ctx, cmdArgs, cleanEnv)

	// Записываем метрики в антидеградацию
	duration := time.Since(startTime)
	a.RecordTaskMetric(originalQuery, duration, 1, nil, err == nil)

	if err != nil {
		// Сохраняем ошибку в память
		a.memoryManager.AddMessage("error", fmt.Sprintf("Query: %s\nError: %v", originalQuery, err))
		return "", fmt.Errorf("qwen cli failed: %w", err)
	}

	// Добавляем ответ в историю (RAM)
	a.conversationHistory = append(a.conversationHistory, fmt.Sprintf("Assistant: %s", output))

	// Сохраняем в локальную сессию (персистентная память)
	if a.localSession != nil {
		a.localSession.AddToHistory(fmt.Sprintf("Assistant: %s", output))
	}

	// Сохраняем ответ в память
	a.memoryManager.AddMessage("assistant", output)

	return output, nil
}

// prependSystemPromptWithContext добавляет системный промпт с контекстом
func (a *Agent) prependSystemPromptWithContext(query, context string) string {
	basePrompt := a.systemPrompt
	
	if context != "" {
		return fmt.Sprintf("%s\n\n---\n\nКонтекст из памяти:\n%s\n\n---\n\nЗапрос пользователя: %s", 
			basePrompt, context, query)
	}
	
	return fmt.Sprintf("%s\n\n---\n\nЗапрос пользователя: %s", basePrompt, query)
}

// prependSystemPrompt добавляет системный промпт (для совместимости)
func (a *Agent) prependSystemPrompt(query string) string {
	return a.prependSystemPromptWithContext(query, "")
}

// buildCommandArgs строит аргументы командной строки для qwen cli
func (a *Agent) buildCommandArgs(query string) []string {
	args := []string{}

	// Добавляем модель если указана
	if a.config.Model != "" {
		args = append(args, "-m", a.config.Model)
	}

	// Добавляем режим подтверждения
	if a.config.ApprovalMode != "" {
		args = append(args, "--approval-mode", a.config.ApprovalMode)
	}

	// Добавляем debug режим если включён
	if a.config.Debug {
		args = append(args, "--debug")
	}

	// Добавляем историю диалога для контекста (последние 10 сообщений)
	if len(a.conversationHistory) > 0 {
		history := a.getRecentHistory(10)
		if history != "" {
			args = append(args, "--context", history)
		}
	}

	// Добавляем запрос как позиционный аргумент
	args = append(args, query)

	return args
}

// getRecentHistory возвращает последние N сообщений истории
func (a *Agent) getRecentHistory(n int) string {
	if a.localSession != nil {
		// Используем персистентную историю
		recent := a.localSession.GetRecentHistory(n)
		return strings.Join(recent, "\n")
	}

	// Fallback на RAM историю
	if len(a.conversationHistory) == 0 {
		return ""
	}

	start := len(a.conversationHistory) - n
	if start < 0 {
		start = 0
	}

	recent := a.conversationHistory[start:]
	return strings.Join(recent, "\n")
}

// GetSessionHistory возвращает всю историю сессии
func (a *Agent) GetSessionHistory() []string {
	if a.localSession != nil {
		return a.localSession.GetHistory()
	}
	return a.conversationHistory
}

// ClearSessionHistory очищает историю сессии
func (a *Agent) ClearSessionHistory() {
	a.conversationHistory = make([]string, 0)
	if a.localSession != nil {
		a.localSession.Clear()
	}
}

// executeQwen выполняет qwen cli команду
func (a *Agent) executeQwen(ctx context.Context, args []string, env []string) (string, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, a.config.QwenPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = os.Stdin
	cmd.Env = env // Используем переданное окружение

	if err := cmd.Run(); err != nil {
		// Возвращаем output даже при ошибке (qwen может вывести полезную информацию)
		output := strings.TrimSpace(stdout.String())
		if output == "" {
			output = strings.TrimSpace(stderr.String())
		}
		if output == "" {
			output = err.Error()
		}
		return output, err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RunInteractive запускает интерактивную сессию с qwen cli
func (a *Agent) RunInteractive(ctx context.Context) error {
	args := []string{}

	// Добавляем модель если указана
	if a.config.Model != "" {
		args = append(args, "-m", a.config.Model)
	}

	// Добавляем режим подтверждения
	if a.config.ApprovalMode != "" {
		args = append(args, "--approval-mode", a.config.ApprovalMode)
	}

	// Запускаем qwen cli в интерактивном режиме с очищенным окружением
	cmd := exec.CommandContext(ctx, a.config.QwenPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = a.getCleanEnv() // Очищенное окружение

	return cmd.Run()
}

// GetHistory возвращает историю разговора
func (a *Agent) GetHistory() []string {
	return a.conversationHistory
}

// ClearHistory очищает историю разговора
func (a *Agent) ClearHistory() {
	a.conversationHistory = make([]string, 0)
}

// ExecuteSkill выполняет навык через qwen cli
func (a *Agent) ExecuteSkill(ctx context.Context, skillName string, args []string) (string, error) {
	query := fmt.Sprintf("Use the %s skill with arguments: %s", skillName, strings.Join(args, " "))
	return a.Run(ctx, query)
}

// Remember сохраняет факт в память (через qwen cli + локально)
func (a *Agent) Remember(ctx context.Context, fact string) (string, error) {
	// Сохраняем локально
	_, err := a.memoryManager.Remember("fact", fact, nil)
	if err != nil {
		return "", fmt.Errorf("failed to save to memory: %w", err)
	}
	
	// Также просим qwen запомнить
	query := fmt.Sprintf("Remember this fact for future context: %s", fact)
	return a.Run(ctx, query)
}

// Recall ищет в памяти и запрашивает у qwen
func (a *Agent) Recall(ctx context.Context, query string) (string, error) {
	// Ищем локально
	results := a.memoryManager.Recall(query, 10)
	
	context := ""
	if len(results) > 0 {
		context = "Found in local memory:\n"
		for _, r := range results {
			context += fmt.Sprintf("- %s\n", r.Content)
		}
		context += "\n"
	}
	
	// Запрашиваем у qwen с контекстом
	fullQuery := context + "Search your context and answer: " + query
	return a.Run(ctx, fullQuery)
}

// GetMemoryContext возвращает контекст из памяти
func (a *Agent) GetMemoryContext() string {
	return a.memoryManager.GetContext()
}

// SetModel устанавливает модель
func (a *Agent) SetModel(model string) {
	a.config.Model = model
}

// GetModel возвращает текущую модель
func (a *Agent) GetModel() string {
	return a.config.Model
}

// GetQwenPath возвращает путь к qwen cli
func (a *Agent) GetQwenPath() string {
	return a.config.QwenPath
}

// containsSensitiveData проверяет, содержит ли запрос чувствительные данные
func (a *Agent) containsSensitiveData(query string) bool {
	query = strings.ToLower(query)

	// Блокируем ТОЛЬКО запросы на раскрытие/получение чувствительных данных
	// Не блокируем упоминания в контексте правил, документации и т.д.
	
	// Паттерны запросов на раскрытие
	revealPatterns := []string{
		// Прямые запросы токенов/ключей
		"покажи токен",
		"покажи ключ",
		"дай токен",
		"дай ключ",
		"раскрой токен",
		"get token",
		"show token",
		"reveal token",
		"what is the token",
		"what is my token",
		
		// Запросы ID пользователей
		"покажи allowed",
		"покажи id пользователя",
		"какой мой id",
		"какой id у",
		"who has access",
		"show allowed users",
		"show user id",
		
		// Запросы секретов/паролей
		"покажи секрет",
		"покажи пароль",
		"дай секрет",
		"дай пароль",
		"show secret",
		"show password",
		"what is the secret",
		"what is the password",
		
		// Запросы .env содержимого
		"покажи .env",
		"покажи env",
		"содержимое .env",
		"содержимое env",
		"show .env",
		"show env file",
		"read .env",
		"cat .env",
	}

	for _, pattern := range revealPatterns {
		if strings.Contains(query, pattern) {
			return true
		}
	}

	return false
}

// getCleanEnv возвращает окружение без чувствительных переменных
func (a *Agent) getCleanEnv() []string {
	// Переменные, которые нужно исключить
	blockedVars := map[string]bool{
		"QWEN_CLAW_TELEGRAM_TOKEN":        true,
		"QWEN_CLAW_TELEGRAM_ALLOWED_USERS": true,
		"QWEN_CLAW_API_KEY":               true,
		"TELEGRAM_TOKEN":                  true,
		"TELEGRAM_BOT_TOKEN":              true,
		"API_KEY":                         true,
		"SECRET":                          true,
		"PASSWORD":                        true,
		"TOKEN":                           true,
	}

	// Получаем текущее окружение
	allEnv := os.Environ()
	cleanEnv := make([]string, 0, len(allEnv))

	for _, env := range allEnv {
		// Разделяем имя и значение
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		name := parts[0]
		// Проверяем, не заблокирована ли переменная
		if !blockedVars[name] && !strings.Contains(name, "TOKEN") &&
			!strings.Contains(name, "SECRET") && !strings.Contains(name, "PASSWORD") {
			cleanEnv = append(cleanEnv, env)
		}
	}

	return cleanEnv
}

// CheckQwenAvailable проверяет доступность qwen cli
func (a *Agent) CheckQwenAvailable() bool {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, a.config.QwenPath, "--help")
	cmd.Env = a.getCleanEnv() // Используем очищенное окружение
	err := cmd.Run()
	return err == nil
}

// GetSkillEngine возвращает движок навыков
func (a *Agent) GetSkillEngine() *skills.Engine {
	return a.skillEngine
}

// GetSystemPrompt возвращает текущий системный промпт
func (a *Agent) GetSystemPrompt() string {
	return a.systemPrompt
}

// UpdateSystemPrompt обновляет системный промпт
func (a *Agent) UpdateSystemPrompt(newPrompt string) {
	a.systemPrompt = newPrompt
}

// ReloadSystemPrompt перезагружает системный промпт из файла
func (a *Agent) ReloadSystemPrompt() error {
	newPrompt := loadSystemPrompt(a.config.SystemPromptPath)
	if newPrompt == "" {
		return fmt.Errorf("failed to load system prompt")
	}
	a.systemPrompt = newPrompt
	return nil
}

// GetAntiDegradationSystem возвращает систему антидеградации
func (a *Agent) GetAntiDegradationSystem() *AntiDegradationSystem {
	return a.antiDegradation
}

// RecordTaskMetric записывает метрики задачи
func (a *Agent) RecordTaskMetric(query string, duration time.Duration, attempts int, errors []string, success bool) {
	if a.antiDegradation != nil {
		a.antiDegradation.RecordTask(query, duration, attempts, errors, success)
	}
}

// GetExecutor возвращает executor
func (a *Agent) GetExecutor() Executor {
	return a.executor
}

// SetExecutor устанавливает executor
func (a *Agent) SetExecutor(executor Executor) {
	a.executor = executor
}

// ExecuteCommand выполняет команду через executor
func (a *Agent) ExecuteCommand(ctx context.Context, command string, args []string) (string, error) {
	return a.executor.Execute(ctx, command, args)
}

// SetApprovalMode устанавливает режим подтверждения
func (a *Agent) SetApprovalMode(mode string) {
	if qwenExec, ok := a.executor.(*QwenExecutor); ok {
		qwenExec.SetApprovalMode(mode)
	}
}

// GetApprovalMode возвращает текущий режим подтверждения
func (a *Agent) GetApprovalMode() string {
	if qwenExec, ok := a.executor.(*QwenExecutor); ok {
		return qwenExec.GetApprovalMode()
	}
	return "auto-edit"
}

// ScheduleTask создаёт задачу в планировщике
func (a *Agent) ScheduleTask(name, description, command, schedule string) (*scheduler.Task, error) {
	if a.scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	return a.scheduler.AddTask(name, description, command, schedule)
}

// RemoveTask удаляет задачу из планировщика
func (a *Agent) RemoveTask(taskID string) error {
	if a.scheduler == nil {
		return fmt.Errorf("scheduler not initialized")
	}
	return a.scheduler.RemoveTask(taskID)
}

// EnableTask включает задачу
func (a *Agent) EnableTask(taskID string) error {
	if a.scheduler == nil {
		return fmt.Errorf("scheduler not initialized")
	}
	return a.scheduler.EnableTask(taskID)
}

// DisableTask отключает задачу
func (a *Agent) DisableTask(taskID string) error {
	if a.scheduler == nil {
		return fmt.Errorf("scheduler not initialized")
	}
	return a.scheduler.DisableTask(taskID)
}

// RunTaskNow выполняет задачу немедленно
func (a *Agent) RunTaskNow(taskID string) (*scheduler.TaskResult, error) {
	if a.scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	return a.scheduler.RunTaskNow(taskID)
}

// GetTask возвращает задачу по ID
func (a *Agent) GetTask(taskID string) (*scheduler.Task, error) {
	if a.scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	return a.scheduler.GetTask(taskID)
}

// GetTasks возвращает все задачи
func (a *Agent) GetTasks() []*scheduler.Task {
	if a.scheduler == nil {
		return nil
	}
	return a.scheduler.GetTasks()
}

// GetScheduler возвращает планировщик
func (a *Agent) GetScheduler() *scheduler.Scheduler {
	if s, ok := a.scheduler.(*scheduler.Scheduler); ok {
		return s
	}
	return nil
}

// GetConfirmationManager возвращает менеджер подтверждений
func (a *Agent) GetConfirmationManager() *ConfirmationManager {
	return a.confirmationManager
}

// GetIntentDetector возвращает детектор намерений
func (a *Agent) GetIntentDetector() *IntentDetector {
	return a.intentDetector
}

// ProcessIntent обрабатывает намерение с подтверждением
func (a *Agent) ProcessIntent(ctx context.Context, query string) (string, error) {
	// Распознаём намерение
	intent := a.intentDetector.Detect(query)

	// Проверяем, требуется ли подтверждение
	if a.confirmationManager.RequiresConfirmation(intent) {
		// Создаём ожидающее действие
		action := a.confirmationManager.CreatePending(intent, query)

		// Возвращаем запрос подтверждения
		return fmt.Sprintf("⚠️ Требуется подтверждение для: %s\n\nДействие: %s\nЗапрос: %s\n\nНапишите: ПОДТВЕРЖДАЮ %s",
			a.intentDetector.GetIntentDescription(intent),
			intent.Type,
			query,
			action.ID), nil
	}

	// Выполняем без подтверждения
	return a.executeIntent(ctx, intent)
}

// executeIntent выполняет намерение
func (a *Agent) executeIntent(ctx context.Context, intent *Intent) (string, error) {
	switch intent.Type {
	case IntentMemoryAdd:
		// Запомнить в память
		_, err := a.memoryManager.Remember("fact", intent.Data, nil)
		if err != nil {
			return "", err
		}
		return "✅ Запомнил: " + intent.Data, nil

	case IntentMemorySearch:
		// Поиск в памяти
		result := a.memoryManager.Find(intent.Data, nil)
		if result.Found {
			return "📚 Найдено в памяти:\n" + a.formatMemoryContext(result), nil
		}
		return "Ничего не найдено в памяти", nil

	case IntentTaskSchedule:
		// Запланировать задачу
		if a.scheduler == nil {
			return "", fmt.Errorf("scheduler not initialized")
		}
		// Парсим команду из данных (формат: "задачу name на schedule с командой cmd")
		task, err := a.scheduler.AddTask("task_"+generateActionID(), intent.Data, intent.Data, "@daily")
		if err != nil {
			return "", err
		}
		return "✅ Задача запланирована: " + task.ID, nil

	case IntentShellExec:
		// Выполнить команду через executor
		output, err := a.executor.Execute(ctx, intent.Data, nil)
		if err != nil {
			return "", err
		}
		return "✅ Выполнено:\n" + output, nil

	default:
		// Для остальных намерений используем Qwen CLI
		return a.Run(ctx, intent.Data)
	}
}

// ConfirmAction подтверждает действие по ID
func (a *Agent) ConfirmAction(actionID string) (string, error) {
	if !a.confirmationManager.Confirm(actionID) {
		return "", fmt.Errorf("failed to confirm action: %s", actionID)
	}

	action := a.confirmationManager.GetPending(actionID)
	if action == nil {
		return "", fmt.Errorf("action not found: %s", actionID)
	}

	// Выполняем действие
	ctx := context.Background()
	result, err := a.executeIntent(ctx, action.Intent)
	if err != nil {
		return "", err
	}

	return "✅ Действие выполнено:\n" + result, nil
}

// RejectAction отклоняет действие по ID
func (a *Agent) RejectAction(actionID string) error {
	if !a.confirmationManager.Reject(actionID) {
		return fmt.Errorf("failed to reject action: %s", actionID)
	}
	return nil
}

// GetPendingActions возвращает все ожидающие действия
func (a *Agent) GetPendingActions() []*PendingAction {
	return a.confirmationManager.GetPendingActions()
}

// SetConfirmationMode устанавливает режим подтверждений
func (a *Agent) SetConfirmationMode(mode ConfirmationMode) {
	a.confirmationManager.SetMode(mode)
}

// GetConfirmationMode возвращает текущий режим подтверждений
func (a *Agent) GetConfirmationMode() ConfirmationMode {
	return a.confirmationManager.GetMode()
}
