package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Executor интерфейс для выполнения команд
type Executor interface {
	// Execute выполняет команду с аргументами
	Execute(ctx context.Context, command string, args []string) (string, error)

	// RequiresConfirmation проверяет, требует ли действие подтверждения
	RequiresConfirmation(action string) bool

	// Confirm запрашивает подтверждение действия
	Confirm(action string) bool
}

// QwenExecutor реализует Executor через qwen cli
type QwenExecutor struct {
	qwenPath   string
	model      string
	approvalMode string
	debug      bool
	timeout    time.Duration
}

// QwenExecutorConfig конфигурация для QwenExecutor
type QwenExecutorConfig struct {
	QwenPath     string        `json:"qwen_path"`
	Model        string        `json:"model"`
	ApprovalMode string        `json:"approval_mode"` // plan/auto-edit/yolo
	Debug        bool          `json:"debug"`
	Timeout      time.Duration `json:"timeout"`
}

// NewQwenExecutor создаёт новый QwenExecutor
func NewQwenExecutor(config QwenExecutorConfig) *QwenExecutor {
	// Если путь не указан, пытаемся найти qwen в PATH
	if config.QwenPath == "" {
		path, err := exec.LookPath("qwen")
		if err != nil {
			config.QwenPath = "qwen" // пробуем запустить из PATH
		} else {
			config.QwenPath = path
		}
	}

	// Таймаут по умолчанию
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}

	// Режим подтверждения по умолчанию
	if config.ApprovalMode == "" {
		config.ApprovalMode = "auto-edit"
	}

	return &QwenExecutor{
		qwenPath:     config.QwenPath,
		model:        config.Model,
		approvalMode: config.ApprovalMode,
		debug:        config.Debug,
		timeout:      config.Timeout,
	}
}

// Execute выполняет команду через qwen cli
func (e *QwenExecutor) Execute(ctx context.Context, command string, args []string) (string, error) {
	// Формируем команду для выполнения
	cmd := e.buildCommand(command, args)

	// Создаём контекст с таймаутом
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Запускаем команду
	var stdout, stderr bytes.Buffer
	qwenCmd := exec.CommandContext(ctx, e.qwenPath, cmd...)
	qwenCmd.Stdout = &stdout
	qwenCmd.Stderr = &stderr
	qwenCmd.Stdin = os.Stdin
	qwenCmd.Env = e.getCleanEnv()

	if err := qwenCmd.Run(); err != nil {
		output := strings.TrimSpace(stdout.String())
		if output == "" {
			output = strings.TrimSpace(stderr.String())
		}
		if output == "" {
			output = err.Error()
		}
		return output, fmt.Errorf("qwen cli failed: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// buildCommand строит команду для qwen cli
func (e *QwenExecutor) buildCommand(command string, args []string) []string {
	cmdArgs := []string{}

	// Добавляем модель если указана
	if e.model != "" {
		cmdArgs = append(cmdArgs, "-m", e.model)
	}

	// Добавляем режим подтверждения
	if e.approvalMode != "" {
		cmdArgs = append(cmdArgs, "--approval-mode", e.approvalMode)
	}

	// Добавляем debug режим если включён
	if e.debug {
		cmdArgs = append(cmdArgs, "--debug")
	}

	// Добавляем команду и аргументы
	cmdArgs = append(cmdArgs, command)
	cmdArgs = append(cmdArgs, args...)

	return cmdArgs
}

// RequiresConfirmation проверяет, требует ли действие подтверждения
func (e *QwenExecutor) RequiresConfirmation(action string) bool {
	switch e.approvalMode {
	case "yolo":
		// В режиме yolo никогда не спрашиваем
		return false
	case "plan":
		// В режиме plan всегда спрашиваем
		return true
	case "auto-edit":
		// В режиме auto-edit спрашиваем для опасных действий
		return e.isDangerousAction(action)
	default:
		// По умолчанию спрашиваем для опасных действий
		return e.isDangerousAction(action)
	}
}

// isDangerousAction проверяет, является ли действие опасным
func (e *QwenExecutor) isDangerousAction(action string) bool {
	action = strings.ToLower(action)

	// Опасные действия
	dangerousPatterns := []string{
		"delete",
		"remove",
		"rm ",
		"drop",
		"destroy",
		"format",
		"chmod",
		"chown",
		"sudo",
		"kill",
		"terminate",
		"overwrite",
		">>",
		"> ",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(action, pattern) {
			return true
		}
	}

	return false
}

// Confirm запрашивает подтверждение действия (всегда возвращает true для автоматического режима)
func (e *QwenExecutor) Confirm(action string) bool {
	// В автоматическом режиме просто возвращаем true
	// Подтверждение должно обрабатываться на уровне агента
	return !e.RequiresConfirmation(action)
}

// getCleanEnv возвращает окружение без чувствительных переменных
func (e *QwenExecutor) getCleanEnv() []string {
	// Переменные, которые нужно исключить
	blockedVars := map[string]bool{
		"QWEN_CLAW_TELEGRAM_TOKEN":         true,
		"QWEN_CLAW_TELEGRAM_ALLOWED_USERS": true,
		"QWEN_CLAW_API_KEY":                true,
		"TELEGRAM_TOKEN":                   true,
		"TELEGRAM_BOT_TOKEN":               true,
		"API_KEY":                          true,
		"SECRET":                           true,
		"PASSWORD":                         true,
		"TOKEN":                            true,
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

// GetApprovalMode возвращает текущий режим подтверждения
func (e *QwenExecutor) GetApprovalMode() string {
	return e.approvalMode
}

// SetApprovalMode устанавливает режим подтверждения
func (e *QwenExecutor) SetApprovalMode(mode string) {
	e.approvalMode = mode
}

// GetTimeout возвращает текущий таймаут
func (e *QwenExecutor) GetTimeout() time.Duration {
	return e.timeout
}

// SetTimeout устанавливает таймаут
func (e *QwenExecutor) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}
