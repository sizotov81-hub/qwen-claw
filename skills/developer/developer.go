// Package developer предоставляет навык разработчика для qwen-claw
package developer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/user/qwen-claw/internal/skills"
)

// DeveloperSkill навык разработчика
type DeveloperSkill struct {
	projectRoot string
}

// NewDeveloperSkill создаёт новый скил разработчика
func NewDeveloperSkill() *DeveloperSkill {
	return &DeveloperSkill{
		projectRoot: detectProjectRoot(),
	}
}

// detectProjectRoot находит корень проекта
func detectProjectRoot() string {
	// Ищем go.mod
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "/home/ss/qwen-claw"
}

// GetInfo возвращает информацию о скиле
func (s *DeveloperSkill) GetInfo() *skills.Info {
	return &skills.Info{
		Name:        "developer",
		Description: "Навык разработчика для работы с кодом проекта qwen-claw",
		Version:     "1.0.0",
		Enabled:     true,
		Type:        "system",
		Commands:    []string{"dev", "develop", "code", "refactor"},
		EntryPoint:  "developer.go",
	}
}

// Execute выполняет команду разработчика
func (s *DeveloperSkill) Execute(ctx context.Context, req *skills.Request) (*skills.Response, error) {
	if len(req.Args) == 0 {
		return &skills.Response{
			Output: s.getHelp(),
		}, nil
	}

	command := strings.ToLower(req.Args[0])

	switch command {
	case "rules":
		return s.getSecurityRules()
	case "check":
		return s.checkSecurity()
	case "init":
		return s.initProject()
	default:
		return &skills.Response{
			Output: fmt.Sprintf("Unknown command: %s\n\n%s", command, s.getHelp()),
		}, nil
	}
}

// getHelp возвращает справку
func (s *DeveloperSkill) getHelp() string {
	return `🛠️ Developer Skill - Навык разработчика

Команды:
  dev rules     - Показать правила безопасности
  dev check     - Проверить проект на нарушения
  dev init      - Инициализировать проект

Правила безопасности:
  ✅ Храните секреты только в .env
  ✅ Используйте QWEN_CLAW_TELEGRAM_ALLOWED_USERS
  ✅ Проверяйте .gitignore перед коммитом
  ✅ Используйте режим auto-edit

Запрещено:
  ❌ Не коммитьте .env в git
  ❌ Не передавайте токены в аргументах
  ❌ Не отключайте подтверждение для rm -rf
  ❌ Не логируйте секреты
`
}

// getSecurityRules возвращает правила безопасности
func (s *DeveloperSkill) getSecurityRules() (*skills.Response, error) {
	rulesFile := filepath.Join(s.projectRoot, "skills/developer/SECURITY_RULES.md")

	if _, err := os.Stat(rulesFile); err != nil {
		return &skills.Response{
			Output: "❌ Rules file not found. Run 'dev init' to create it.",
		}, nil
	}

	data, err := os.ReadFile(rulesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules: %w", err)
	}

	return &skills.Response{
		Output: string(data),
	}, nil
}

// checkSecurity проверяет проект на нарушения
func (s *DeveloperSkill) checkSecurity() (*skills.Response, error) {
	var issues []string

	// 1. Проверка .env в git
	envInGit := s.checkEnvInGit()
	if envInGit {
		issues = append(issues, "❌ .env found in git repository!")
	}

	// 2. Проверка .gitignore
	hasGitignore := s.checkGitignore()
	if !hasGitignore {
		issues = append(issues, "❌ .gitignore not found or incomplete")
	}

	// 3. Проверка логов на секреты
	logIssues := s.checkLogsForSecrets()
	if len(logIssues) > 0 {
		issues = append(issues, logIssues...)
	}

	if len(issues) == 0 {
		return &skills.Response{
			Output: "✅ All security checks passed!",
		}, nil
	}

	return &skills.Response{
		Output: fmt.Sprintf("⚠️ Security issues found:\n\n%s", strings.Join(issues, "\n")),
	}, nil
}

// checkEnvInGit проверяет есть ли .env в git
func (s *DeveloperSkill) checkEnvInGit() bool {
	gitDir := filepath.Join(s.projectRoot, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return false // Not a git repo
	}

	envFile := filepath.Join(s.projectRoot, ".env")
	if _, err := os.Stat(envFile); err != nil {
		return false // No .env file
	}

	// Проверяем git ls-files
	cmd := fmt.Sprintf("cd %s && git ls-files .env 2>/dev/null", s.projectRoot)
	output, err := execCommand("bash", "-c", cmd)
	if err != nil {
		return false
	}

	return strings.TrimSpace(output) != ""
}

// checkGitignore проверяет .gitignore
func (s *DeveloperSkill) checkGitignore() bool {
	gitignorePath := filepath.Join(s.projectRoot, ".gitignore")

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return false
	}

	content := string(data)

	// Проверяем обязательные записи
	required := []string{".env", "*.db", "*.log"}
	for _, r := range required {
		if !strings.Contains(content, r) {
			return false
		}
	}

	return true
}

// checkLogsForSecrets проверяет логи на наличие секретов
func (s *DeveloperSkill) checkLogsForSecrets() []string {
	var issues []string

	// Ищем log файлы
	logFiles := []string{
		filepath.Join(s.projectRoot, "*.log"),
		filepath.Join(s.projectRoot, "logs/*.log"),
	}

	patterns := []string{"token", "secret", "password", "QWEN_CLAW_TELEGRAM"}

	for _, pattern := range logFiles {
		matches, _ := filepath.Glob(pattern)
		for _, file := range matches {
			data, _ := os.ReadFile(file)
			content := strings.ToLower(string(data))

			for _, p := range patterns {
				if strings.Contains(content, p) {
					issues = append(issues, fmt.Sprintf("❌ Possible secret '%s' in %s", p, file))
				}
			}
		}
	}

	return issues
}

// initProject инициализирует проект
func (s *DeveloperSkill) initProject() (*skills.Response, error) {
	// Создаём .env если нет
	envFile := filepath.Join(s.projectRoot, ".env")
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		content := `# Qwen-Claw Configuration
QWEN_CLAW_TELEGRAM_TOKEN=""
QWEN_CLAW_TELEGRAM_ALLOWED_USERS=""
QWEN_CLAW_MODEL=""
QWEN_CLAW_APPROVAL_MODE="auto-edit"
QWEN_CLAW_DEBUG="false"
`
		if err := os.WriteFile(envFile, []byte(content), 0600); err != nil {
			return nil, fmt.Errorf("failed to create .env: %w", err)
		}
	}

	// Создаём .gitignore если нет
	gitignoreFile := filepath.Join(s.projectRoot, ".gitignore")
	if _, err := os.Stat(gitignoreFile); os.IsNotExist(err) {
		content := `# Secrets
.env
*.pem
*.key

# Databases
*.db
*.sqlite

# Logs
*.log
logs/

# Build
qwen-claw
*.bin

# OS
.DS_Store
Thumbs.db
`
		if err := os.WriteFile(gitignoreFile, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to create .gitignore: %w", err)
		}
	}

	return &skills.Response{
		Output: "✅ Project initialized!\n\nCreated:\n- .env (permissions 0600)\n- .gitignore\n\n⚠️  Edit .env and add your tokens.",
	}, nil
}

// execCommand выполняет команду
func execCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// Register регистрирует скил
func Register() {
	skills.RegisterSkill(NewDeveloperSkill())
}
