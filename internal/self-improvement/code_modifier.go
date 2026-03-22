package selfimprovement

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/qwen-claw/internal/voice"
)

var (
	ErrFileNotFound = errors.New("file not found")
	ErrContentMismatch = errors.New("file content has changed")
	ErrGitNotEnabled = errors.New("git not enabled")
)

// CodeModifier модификатор кода
type CodeModifier struct {
	projectRoot string
	gitEnabled  bool
}

// NewCodeModifier создаёт модификатор
func NewCodeModifier(projectRoot string) *CodeModifier {
	return &CodeModifier{
		projectRoot: projectRoot,
		gitEnabled:  isGitRepo(projectRoot),
	}
}

// ModifyFile изменяет файл
func (cm *CodeModifier) ModifyFile(filePath, oldContent, newContent string) error {
	fullPath := filepath.Join(cm.projectRoot, filePath)
	
	// Проверяем существование
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("modify file: %w", ErrFileNotFound)
	}
	
	// Читаем текущее содержимое
	current, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	
	// Проверяем, что oldContent совпадает
	if string(current) != oldContent {
		return fmt.Errorf("modify file: %w", ErrContentMismatch)
	}
	
	// Записываем новое
	return os.WriteFile(fullPath, []byte(newContent), 0644)
}

// CreateFile создаёт файл
func (cm *CodeModifier) CreateFile(filePath, content string) error {
	fullPath := filepath.Join(cm.projectRoot, filePath)
	
	// Создаём директорию
	dir := filepath.Dir(fullPath)
	os.MkdirAll(dir, 0755)
	
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// DeleteFile удаляет файл
func (cm *CodeModifier) DeleteFile(filePath string) error {
	fullPath := filepath.Join(cm.projectRoot, filePath)
	return os.Remove(fullPath)
}

// RunTests запускает тесты
func (cm *CodeModifier) RunTests() (bool, string, error) {
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = cm.projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, string(output), err
	}
	
	return true, string(output), nil
}

// Build собирает проект
func (cm *CodeModifier) Build() (bool, string, error) {
	cmd := exec.Command("go", "build", "-o", "qwen-claw", "./cmd/main.go")
	cmd.Dir = cm.projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, string(output), err
	}
	
	return true, string(output), nil
}

// Commit коммитит изменения
func (cm *CodeModifier) Commit(message string) error {
	if !cm.gitEnabled {
		return fmt.Errorf("commit changes: %w", ErrGitNotEnabled)
	}
	
	commands := []string{
		"git add -A",
		fmt.Sprintf("git commit -m '%s'", message),
	}
	
	for _, cmdStr := range commands {
		cmd := exec.Command("bash", "-c", cmdStr)
		cmd.Dir = cm.projectRoot
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, string(output))
		}
	}
	
	return nil
}

// Restart перезапускает бота
func (cm *CodeModifier) Restart(serviceName string) error {
	// Проверяем, запущен ли как systemd сервис
	cmd := exec.Command("systemctl", "is-active", serviceName)
	cmd.Dir = cm.projectRoot
	
	output, _ := cmd.CombinedOutput()
	if strings.TrimSpace(string(output)) == "active" {
		// Перезапускаем через systemd
		cmd = exec.Command("systemctl", "restart", serviceName)
		return cmd.Run()
	}
	
	// Иначе просто убиваем старый процесс и запускаем новый
	cmd = exec.Command("pkill", "-f", "qwen-claw")
	cmd.Run()
	
	time.Sleep(2 * time.Second)
	
	// Запускаем новый
	newCmd := exec.Command("./qwen-claw", "telegram")
	newCmd.Dir = cm.projectRoot
	newCmd.Start()
	
	return nil
}

// HealthCheck проверяет здоровье после рестарта
func (cm *CodeModifier) HealthCheck(timeout time.Duration) bool {
	done := make(chan bool)
	
	go func() {
		time.Sleep(1 * time.Second)
		
		// Проверяем, запущен ли процесс
		cmd := exec.Command("pgrep", "-f", "qwen-claw")
		output, err := cmd.CombinedOutput()
		
		done <- err == nil && len(output) > 0
	}()
	
	select {
	case result := <-done:
		return result
	case <-time.After(timeout):
		return false
	}
}

// isGitRepo проверяет, git репозиторий ли это
func isGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

// GenerateChangePlan генерирует план изменений для распознавания голосовых
func GenerateChangePlan() *ChangeRequest {
	description := "Добавить распознавание голосовых сообщений из Telegram"
	reason := "Пользователь отправляет голосовые сообщения, нужно их распознавать"

	commands := voice.GetInstallationCommands()
	
	files := []FileChange{
		{
			Path:   "bots/telegram/bot.go",
			Action: "modify",
			// NewContent будет сгенерирован динамически
		},
	}
	
	return &ChangeRequest{
		Type:        "voice_recognition",
		Description: description,
		Reason:      reason,
		Commands:    commands,
		Files:       files,
	}
}
