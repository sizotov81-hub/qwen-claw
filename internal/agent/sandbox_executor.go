package agent

import (
	"context"
	"fmt"
	"github.com/user/qwen-claw/internal/sandbox"
	"strings"
	"time"
)

// SandboxExecutor Executor с sandbox поддержкой
type SandboxExecutor struct {
	// wrapped завёрнутый Executor
	wrapped Executor
	
	// sandboxManager менеджер sandbox
	sandboxManager *sandbox.Manager
	
	// defaultSecurityLevel уровень безопасности по умолчанию
	defaultSecurityLevel sandbox.SecurityLevel
	
	// useSandboxForShell использовать sandbox для shell команд
	useSandboxForShell bool
}

// SandboxExecutorConfig конфигурация SandboxExecutor
type SandboxExecutorConfig struct {
	// Wrapped завёрнутый executor
	Wrapped Executor
	
	// SandboxEnabled включён ли sandbox
	SandboxEnabled bool
	
	// SandboxDataDir директория для sandbox данных
	SandboxDataDir string
	
	// SecurityLevel уровень безопасности
	SecurityLevel sandbox.SecurityLevel
	
	// UseSandboxForShell использовать sandbox для shell команд
	UseSandboxForShell bool
}

// NewSandboxExecutor создаёт новый SandboxExecutor
func NewSandboxExecutor(config *SandboxExecutorConfig) (*SandboxExecutor, error) {
	if config == nil {
		config = &SandboxExecutorConfig{}
	}
	
	// Создаём конфиг sandbox
	sbConfig := sandbox.DefaultConfig()
	sbConfig.Enabled = config.SandboxEnabled
	
	// Создаём менеджер sandbox
	sbManager, err := sandbox.NewManager(sbConfig, config.SandboxDataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox manager: %w", err)
	}
	
	// Устанавливаем политику безопасности
	securityPolicy := sandbox.GetSecurityPolicyForLevel(config.SecurityLevel)
	sbManager.SetPolicy(securityPolicy)
	
	return &SandboxExecutor{
		wrapped:              config.Wrapped,
		sandboxManager:       sbManager,
		defaultSecurityLevel: config.SecurityLevel,
		useSandboxForShell:   config.UseSandboxForShell,
	}, nil
}

// SetPolicy устанавливает политику безопасности
func (e *SandboxExecutor) SetPolicy(policy *sandbox.SecurityPolicy) {
	if e.sandboxManager != nil {
		e.sandboxManager.SetPolicy(policy)
	}
}

// Execute выполняет команду с sandbox
func (e *SandboxExecutor) Execute(ctx context.Context, command string, args []string) (string, error) {
	fullCommand := command
	if len(args) > 0 {
		fullCommand = command + " " + strings.Join(args, " ")
	}
	
	// Проверяем, требует ли команда sandbox
	requiresSandbox := e.requiresSandbox(fullCommand)
	
	if requiresSandbox && e.sandboxManager != nil {
		return e.executeInSandbox(ctx, fullCommand)
	}
	
	// Выполняем напрямую
	return e.wrapped.Execute(ctx, command, args)
}

// requiresSandbox проверяет, требует ли команда sandbox
func (e *SandboxExecutor) requiresSandbox(command string) bool {
	if !e.useSandboxForShell {
		return false
	}
	
	// Shell команды требуют sandbox
	shellCommands := []string{
		"bash", "sh", "zsh", "fish",
		"python", "python3", "node", "ruby", "perl", "php",
	}
	
	for _, shell := range shellCommands {
		if len(command) >= len(shell) && 
		   (command == shell || strings.HasPrefix(command, shell + " ") || 
		    strings.HasPrefix(command, shell + ";") || 
		    strings.HasPrefix(command, shell + "\n")) {
			return true
		}
	}
	
	return false
}

// executeInSandbox выполняет команду в sandbox
func (e *SandboxExecutor) executeInSandbox(ctx context.Context, command string) (string, error) {
	// Создаём или получаем sandbox
	sessionID := getSessionIDFromContext(ctx)
	if sessionID == "" {
		sessionID = "default"
	}
	
	_, err := e.sandboxManager.GetSandbox(sessionID)
	if err != nil {
		// Создаём новый sandbox
		_, err = e.sandboxManager.CreateSandbox(sessionID)
		if err != nil {
			return "", fmt.Errorf("failed to create sandbox: %w", err)
		}
	}
	
	// Выполняем команду
	result, err := e.sandboxManager.Execute(ctx, sessionID, command)
	if err != nil {
		// Проверяем, не ошибка ли безопасности
		if _, ok := err.(sandbox.ErrSecurityViolation); ok {
			return "", fmt.Errorf("blocked by sandbox: %w", err)
		}
		return "", fmt.Errorf("sandbox execution failed: %w", err)
	}
	
	// Формируем вывод
	output := result.Stdout
	if result.Stderr != "" {
		output += "\n" + result.Stderr
	}
	
	if result.TimedOut {
		return output + "\n[Command timed out]", nil
	}
	
	if result.ExitCode != 0 {
		return output + fmt.Sprintf("\n[Exit code: %d]", result.ExitCode), nil
	}
	
	return strings.TrimSpace(output), nil
}

// RequiresConfirmation проверяет, требует ли действие подтверждения
func (e *SandboxExecutor) RequiresConfirmation(action string) bool {
	return e.wrapped.RequiresConfirmation(action)
}

// Confirm запрашивает подтверждение действия
func (e *SandboxExecutor) Confirm(action string) bool {
	return e.wrapped.Confirm(action)
}

// GetSandboxManager возвращает менеджер sandbox
func (e *SandboxExecutor) GetSandboxManager() *sandbox.Manager {
	return e.sandboxManager
}

// GetSandboxStats возвращает статистику sandbox
func (e *SandboxExecutor) GetSandboxStats() map[string]interface{} {
	if e.sandboxManager == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}
	
	sandboxes := e.sandboxManager.ListSandboxes()
	stats := make([]map[string]interface{}, 0, len(sandboxes))
	
	for _, sb := range sandboxes {
		stats = append(stats, sb.GetStats())
	}
	
	return map[string]interface{}{
		"enabled":    true,
		"sandboxes":  stats,
		"count":      len(sandboxes),
		"security":   string(e.defaultSecurityLevel),
	}
}

// CleanupSandbox очищает sandbox
func (e *SandboxExecutor) CleanupSandbox(maxAge time.Duration) error {
	if e.sandboxManager == nil {
		return nil
	}
	return e.sandboxManager.Cleanup(maxAge)
}

// getSessionIDFromContext получает ID сессии из контекста
func getSessionIDFromContext(ctx context.Context) string {
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		return sessionID
	}
	return ""
}
