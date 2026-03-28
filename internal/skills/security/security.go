// Package skills/security предоставляет безопасность для системы навыков
package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// IsolationLevel уровень изоляции
type IsolationLevel string

const (
	// IsolationOff без изоляции
	IsolationOff IsolationLevel = "off"
	// IsolationNonMain групповые чаты в контейнере
	IsolationNonMain IsolationLevel = "non-main"
	// IsolationAll все вызовы в контейнере
	IsolationAll IsolationLevel = "all"
)

// Permission разрешение
type Permission string

const (
	PermNetwork     Permission = "network"
	PermFilesystem  Permission = "filesystem"
	PermElevated    Permission = "elevated"
)

// FilesystemAccess доступ к файловой системе
type FilesystemAccess string

const (
	FsNone       FilesystemAccess = "none"
	FsReadonly   FilesystemAccess = "readonly"
	FsReadWrite  FilesystemAccess = "read-write"
)

// SkillPermissions разрешения навыка
type SkillPermissions struct {
	// Network доступ к сети
	Network bool `json:"network"`
	// Filesystem доступ к файловой системе
	Filesystem FilesystemAccess `json:"filesystem"`
	// Elevated повышенные привилегии
	Elevated bool `json:"elevated"`
	// Allowlist разрешённые ресурсы
	Allowlist []string `json:"allowlist,omitempty"`
	// Denylist запрещённые ресурсы
	Denylist []string `json:"denylist,omitempty"`
}

// AuditEntry запись аудита
type AuditEntry struct {
	// Timestamp время события
	Timestamp time.Time `json:"timestamp"`
	// SkillName имя навыка
	SkillName string `json:"skill_name"`
	// Command выполненная команда
	Command string `json:"command"`
	// Args аргументы
	Args []string `json:"args,omitempty"`
	// Success успешно ли выполнено
	Success bool `json:"success"`
	// Error ошибка
	Error string `json:"error,omitempty"`
	// Duration длительность выполнения
	Duration time.Duration `json:"duration"`
	// IsolationLevel уровень изоляции
	IsolationLevel IsolationLevel `json:"isolation_level"`
	// Permissions использованные разрешения
	Permissions SkillPermissions `json:"permissions"`
}

// Auditor аудитор навыков
type Auditor struct {
	// logFile файл для логирования
	logFile string
	// enabled включён ли аудит
	enabled bool
	// entries записи аудита
	entries []AuditEntry
}

// NewAuditor создаёт новый аудитор
func NewAuditor(logFile string) *Auditor {
	return &Auditor{
		logFile: logFile,
		enabled: true,
		entries: make([]AuditEntry, 0),
	}
}

// Log записывает событие аудита
func (a *Auditor) Log(entry AuditEntry) error {
	if !a.enabled {
		return nil
	}

	a.entries = append(a.entries, entry)

	// Сохраняем в файл
	return a.saveToFile(entry)
}

// saveToFile сохраняет запись в файл
func (a *Auditor) saveToFile(entry AuditEntry) error {
	dir := filepath.Dir(a.logFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create audit dir: %w", err)
	}

	f, err := os.OpenFile(a.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	_, err = f.WriteString(string(data) + "\n")
	return err
}

// GetEntries возвращает записи аудита
func (a *Auditor) GetEntries() []AuditEntry {
	return a.entries
}

// Clear очищает записи аудита
func (a *Auditor) Clear() {
	a.entries = make([]AuditEntry, 0)
}

// Sandbox песочница для навыков
type Sandbox struct {
	// enabled включена ли песочница
	enabled bool
	// isolation уровень изоляции
	isolation IsolationLevel
	// workDir рабочая директория
	workDir string
	// permissions разрешения
	permissions SkillPermissions
}

// SandboxConfig конфигурация песочницы
type SandboxConfig struct {
	Enabled     bool             `json:"enabled"`
	Isolation   IsolationLevel   `json:"isolation"`
	WorkDir     string           `json:"work_dir"`
	Permissions SkillPermissions `json:"permissions"`
}

// NewSandbox создаёт новую песочницу
func NewSandbox(config SandboxConfig) *Sandbox {
	return &Sandbox{
		enabled:     config.Enabled,
		isolation:   config.Isolation,
		workDir:     config.WorkDir,
		permissions: config.Permissions,
	}
}

// DefaultSandboxConfig конфигурация по умолчанию
func DefaultSandboxConfig() SandboxConfig {
	return SandboxConfig{
		Enabled:   true,
		Isolation: IsolationNonMain,
		WorkDir:   "/tmp/qwen-claw-sandbox",
		Permissions: SkillPermissions{
			Network:    false,
			Filesystem: FsReadonly,
			Elevated:   false,
		},
	}
}

// Run выполняет команду в песочнице
func (s *Sandbox) Run(ctx context.Context, cmd string, args []string, env map[string]string) (*SandboxResult, error) {
	if !s.enabled || s.isolation == IsolationOff {
		return s.runNative(ctx, cmd, args, env)
	}

	// Проверяем Docker
	if !s.isDockerAvailable() {
		return nil, fmt.Errorf("Docker not available, cannot run in sandbox")
	}

	return s.runInDocker(ctx, cmd, args, env)
}

// SandboxResult результат выполнения в песочнице
type SandboxResult struct {
	ExitCode   int    `json:"exit_code"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
}

// runNative выполняет команду нативно (без изоляции)
func (s *Sandbox) runNative(ctx context.Context, cmd string, args []string, env map[string]string) (*SandboxResult, error) {
	command := exec.CommandContext(ctx, cmd, args...)

	// Устанавливаем переменные окружения
	for k, v := range env {
		command.Env = append(command.Env, fmt.Sprintf("%s=%s", k, v))
	}

	start := time.Now()
	output, err := command.CombinedOutput()
	duration := time.Since(start)

	result := &SandboxResult{
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = string(exitErr.Stderr)
		} else {
			result.ExitCode = -1
			result.Error = err.Error()
		}
	} else {
		result.ExitCode = 0
	}

	result.Output = string(output)
	return result, nil
}

// runInDocker выполняет команду в Docker контейнере
func (s *Sandbox) runInDocker(ctx context.Context, cmd string, args []string, env map[string]string) (*SandboxResult, error) {
	// Создаём рабочую директорию
	if err := os.MkdirAll(s.workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}

	// Формируем docker run команду
	dockerArgs := []string{
		"run", "--rm",
		"--network=none", // Нет доступа к сети по умолчанию
	}

	// Добавляем доступ к сети если разрешено
	if s.permissions.Network {
		dockerArgs = append(dockerArgs, "--network=bridge")
	}

	// Монтируем рабочую директорию
	mountMode := "ro"
	if s.permissions.Filesystem == FsReadWrite {
		mountMode = "rw"
	}
	if s.permissions.Filesystem != FsNone {
		dockerArgs = append(dockerArgs,
			"-v", fmt.Sprintf("%s:/workspace:%s", s.workDir, mountMode))
		dockerArgs = append(dockerArgs, "-w", "/workspace")
	}

	// Добавляем переменные окружения
	for k, v := range env {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Используем минимальный образ
	dockerArgs = append(dockerArgs, "alpine:latest")

	// Добавляем команду
	dockerArgs = append(dockerArgs, cmd)
	dockerArgs = append(dockerArgs, args...)

	command := exec.CommandContext(ctx, "docker", dockerArgs...)

	start := time.Now()
	output, err := command.CombinedOutput()
	duration := time.Since(start)

	result := &SandboxResult{
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = string(exitErr.Stderr)
		} else {
			result.ExitCode = -1
			result.Error = err.Error()
		}
	} else {
		result.ExitCode = 0
	}

	result.Output = string(output)
	return result, nil
}

// isDockerAvailable проверяет доступность Docker
func (s *Sandbox) isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	return cmd.Run() == nil
}

// CheckPermissions проверяет разрешения
func (s *Sandbox) CheckPermissions(resource string) error {
	// Проверяем denylist
	for _, denied := range s.permissions.Denylist {
		if strings.Contains(resource, denied) {
			return fmt.Errorf("resource denied by denylist: %s", resource)
		}
	}

	// Проверяем allowlist
	if len(s.permissions.Allowlist) > 0 {
		allowed := false
		for _, allowedRes := range s.permissions.Allowlist {
			if strings.Contains(resource, allowedRes) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("resource not in allowlist: %s", resource)
		}
	}

	return nil
}

// ValidateSkill валидирует безопасность навыка
func ValidateSkill(skillPath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Passed: true,
		Issues: make([]string, 0),
	}

	// Проверяем наличие SKILL.md
	skillFile := filepath.Join(skillPath, "SKILL.md")
	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		result.Passed = false
		result.Issues = append(result.Issues, "SKILL.md not found")
		return result, nil
	}

	// Проверяем наличие entrypoint
	// Читаем SKILL.md для получения entrypoint
	data, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	// Простой парсинг YAML frontmatter
	content := string(data)
	var entrypoint string
	if strings.Contains(content, "entrypoint:") {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "entrypoint:") {
				entrypoint = strings.TrimSpace(strings.TrimPrefix(line, "entrypoint:"))
				break
			}
		}
	}

	if entrypoint == "" {
		result.Passed = false
		result.Issues = append(result.Issues, "entrypoint not specified")
		return result, nil
	}

	// Проверяем наличие файла entrypoint
	entryPath := filepath.Join(skillPath, entrypoint)
	if _, err := os.Stat(entryPath); os.IsNotExist(err) {
		result.Passed = false
		result.Issues = append(result.Issues, fmt.Sprintf("entrypoint not found: %s", entrypoint))
		return result, nil
	}

	// Проверяем наличие разрешений
	if !strings.Contains(content, "permissions:") {
		result.Issues = append(result.Issues, "permissions not specified (recommended)")
	}

	return result, nil
}

// ValidationResult результат валидации
type ValidationResult struct {
	Passed bool     `json:"passed"`
	Issues []string `json:"issues,omitempty"`
}
