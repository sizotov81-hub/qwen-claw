package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/user/qwen-claw/internal/memory"
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
}

// Agent обёртка вокруг qwen code cli
type Agent struct {
	// config конфигурация
	config AgentConfig

	// memoryManager менеджер памяти
	memoryManager *memory.Manager

	// skillEngine движок навыков
	skillEngine *skills.Engine

	// conversationHistory история разговора
	conversationHistory []string
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

	skillEngine := skills.NewAgentSkillEngine()
	skillEngine.SetMemoryManager(memoryManager)

	return &Agent{
		config:              config,
		memoryManager:       memoryManager,
		skillEngine:         skillEngine,
		conversationHistory: make([]string, 0),
	}
}

// Run выполняет запрос через qwen code cli
func (a *Agent) Run(ctx context.Context, query string) (string, error) {
	// ПРОВЕРКА: блокируем запросы о чувствительных данных
	if a.containsSensitiveData(query) {
		return "❌ Я не могу отвечать на вопросы о токенах, ключах, ID пользователей или других чувствительных данных. Эта информация не хранится в моей памяти и не доступна мне.", nil
	}

	// Добавляем сообщение пользователя в историю
	a.conversationHistory = append(a.conversationHistory, fmt.Sprintf("User: %s", query))

	// Сохраняем в память
	a.memoryManager.AddMessage("user", query)

	// ПРОВЕРКА: очищаем окружение от чувствительных переменных перед запуском qwen cli
	cleanEnv := a.getCleanEnv()

	// Формируем команду для qwen cli
	cmdArgs := a.buildCommandArgs(query)

	// Создаём контекст с таймаутом
	timeout := a.config.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Запускаем qwen cli с очищенным окружением
	output, err := a.executeQwen(ctx, cmdArgs, cleanEnv)
	if err != nil {
		// Сохраняем ошибку в память
		a.memoryManager.AddMessage("error", fmt.Sprintf("Query: %s\nError: %v", query, err))
		return "", fmt.Errorf("qwen cli failed: %w", err)
	}
	
	// Добавляем ответ в историю
	a.conversationHistory = append(a.conversationHistory, fmt.Sprintf("Assistant: %s", output))
	
	// Сохраняем ответ в память
	a.memoryManager.AddMessage("assistant", output)
	
	return output, nil
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
	
	// Добавляем запрос как позиционный аргумент
	args = append(args, query)
	
	return args
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

	// Ключевые слова для блокировки запросов о чувствительных данных
	sensitivePatterns := []string{
		// Токены и ключи
		"telegram token",
		"bot token",
		"api token",
		"api key",
		"токен бота",
		"токен телеграм",
		"ключ api",

		// ID пользователей
		"allowed user",
		"allowed id",
		"user id",
		"telegram id",
		"разрешённы пользователь",
		"id пользователя",
		"айди пользователя",

		// Переменные окружения
		"qwen_claw_telegram",
		"qwen_claw_token",
		"qwen_claw_allowed",
		".env",
		"env variable",
		"переменная окружения",

		// Пароли и секреты
		"password",
		"secret",
		"пароль",
		"секрет",
		"приватн",
		"private key",
	}

	for _, pattern := range sensitivePatterns {
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
