package selfimprovement

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Engine система самосовершенствования
type Engine struct {
	// projectRoot корень проекта
	projectRoot string
	
	// backupDir директория для бэкапов
	backupDir string
	
	// logFile файл журнала
	logFile string
	
	// config конфигурация безопасности
	config *SecurityConfig
	
	// pendingChanges ожидающие подтверждения изменения
	pendingChanges []*ChangeRequest
}

// SecurityConfig конфигурация безопасности
type SecurityConfig struct {
	// ConfirmationRequired действия требующие подтверждения
	ConfirmationRequired []string `json:"confirmation_required"`
	
	// AutoAllowed действия разрешённые автоматически
	AutoAllowed []string `json:"auto_allowed"`
	
	// Forbidden запрещённые действия
	Forbidden []string `json:"forbidden"`
	
	// RollbackEnabled включён ли откат
	RollbackEnabled bool `json:"rollback_enabled"`
	
	// HealthCheckTimeout таймаут health check
	HealthCheckTimeout time.Duration `json:"health_check_timeout"`
	
	// RequireBackup требовать бэкап перед изменениями
	RequireBackup bool `json:"require_backup"`
}

// DefaultSecurityConfig конфигурация по умолчанию (максимально безопасная)
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		ConfirmationRequired: []string{
			"system_packages",    // apt install, yum install
			"code_changes",       // изменение .go файлов
			"restart",            // перезапуск процесса
			"env_changes",        // изменение .env
			"config_changes",     // изменение config.yaml
			"delete_files",       // удаление файлов
			"chmod_changes",      // изменение прав
		},
		AutoAllowed: []string{
			"go_packages",        // go get
			"go_test",            // go test
			"go_build",           // go build
			"git_commit",         // git commit
			"read_files",         // чтение файлов
		},
		Forbidden: []string{
			"rm_rf_root",         // rm -rf /
			"chmod_777",          // chmod 777
			"sudo_rm",            // sudo rm
			"curl_pipe_bash",     // curl | bash
		},
		RollbackEnabled:      true,
		HealthCheckTimeout:   30 * time.Second,
		RequireBackup:        true,
	}
}

// ChangeRequest запрос на изменение
type ChangeRequest struct {
	// ID уникальный идентификатор
	ID string `json:"id"`
	
	// Type тип изменения
	Type string `json:"type"`
	
	// Description описание
	Description string `json:"description"`
	
	// Reason причина изменения
	Reason string `json:"reason"`
	
	// Commands команды для выполнения
	Commands []string `json:"commands"`
	
	// Files файлы для изменения
	Files []FileChange `json:"files"`
	
	// RequiresConfirmation требует подтверждения
	RequiresConfirmation bool `json:"requires_confirmation"`
	
	// Created время создания
	Created time.Time `json:"created"`
	
	// Status статус
	Status string `json:"status"` // pending|approved|rejected|completed|rolled_back
	
	// BackupID ID бэкапа
	BackupID string `json:"backup_id"`
}

// FileChange изменение файла
type FileChange struct {
	// Path путь к файлу
	Path string `json:"path"`
	
	// Action действие
	Action string `json:"action"` // create|modify|delete
	
	// OldContent старое содержимое
	OldContent string `json:"old_content,omitempty"`
	
	// NewContent новое содержимое
	NewContent string `json:"new_content,omitempty"`
}

// NewEngine создаёт систему самосовершенствования
func NewEngine(projectRoot string) *Engine {
	backupDir := filepath.Join(projectRoot, ".qwen", "backups")
	logFile := filepath.Join(projectRoot, ".qwen", "self-improvement.log")
	
	os.MkdirAll(backupDir, 0755)
	
	return &Engine{
		projectRoot: projectRoot,
		backupDir:   backupDir,
		logFile:     logFile,
		config:      DefaultSecurityConfig(),
		pendingChanges: make([]*ChangeRequest, 0),
	}
}

// RequestChange создаёт запрос на изменение
func (e *Engine) RequestChange(changeType, description, reason string, commands []string, files []FileChange) *ChangeRequest {
	change := &ChangeRequest{
		ID:          generateID(),
		Type:        changeType,
		Description: description,
		Reason:      reason,
		Commands:    commands,
		Files:       files,
		Created:     time.Now(),
		Status:      "pending",
	}
	
	// Проверяем, требует ли подтверждения
	change.RequiresConfirmation = e.requiresConfirmation(changeType)
	
	// Проверяем на запрещённые действия
	if e.isForbidden(change) {
		change.Status = "rejected"
		e.logChange(change, "REJECTED (forbidden)")
		return change
	}
	
	// Добавляем в ожидающие
	if change.RequiresConfirmation {
		e.pendingChanges = append(e.pendingChanges, change)
	}
	
	e.logChange(change, "REQUESTED")
	
	return change
}

// requiresConfirmation проверяет, требует ли действие подтверждения
func (e *Engine) requiresConfirmation(actionType string) bool {
	for _, t := range e.config.ConfirmationRequired {
		if t == actionType {
			return true
		}
	}
	return false
}

// isForbidden проверяет, запрещено ли действие
func (e *Engine) isForbidden(change *ChangeRequest) bool {
	for _, forbidden := range e.config.Forbidden {
		// Проверяем команды
		for _, cmd := range change.Commands {
			if strings.Contains(cmd, forbidden) {
				return true
			}
		}
		
		// Проверяем файлы
		for _, file := range change.Files {
			if strings.Contains(file.Path, forbidden) {
				return true
			}
		}
	}
	
	return false
}

// Approve подтверждает изменение
func (e *Engine) Approve(changeID string) error {
	change := e.findChange(changeID)
	if change == nil {
		return fmt.Errorf("change %s not found", changeID)
	}
	
	if change.Status != "pending" {
		return fmt.Errorf("change %s is not pending (status: %s)", changeID, change.Status)
	}
	
	// Создаём бэкап если требуется
	if e.config.RequireBackup {
		backupID, err := e.createBackup(change)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		change.BackupID = backupID
	}
	
	// Выполняем изменение
	change.Status = "approved"
	e.logChange(change, "APPROVED")
	
	return e.executeChange(change)
}

// Reject отклоняет изменение
func (e *Engine) Reject(changeID string) error {
	change := e.findChange(changeID)
	if change == nil {
		return fmt.Errorf("change %s not found", changeID)
	}
	
	change.Status = "rejected"
	e.logChange(change, "REJECTED")
	
	// Удаляем из ожидающих
	e.removePending(changeID)
	
	return nil
}

// executeChange выполняет изменение
func (e *Engine) executeChange(change *ChangeRequest) error {
	e.logChange(change, "EXECUTING")
	
	// Выполняем команды
	for _, cmd := range change.Commands {
		if err := e.executeCommand(cmd); err != nil {
			e.logChange(change, fmt.Sprintf("COMMAND FAILED: %s - %v", cmd, err))
			
			// Откат если требуется
			if e.config.RollbackEnabled && change.BackupID != "" {
				e.rollback(change.BackupID)
				change.Status = "rolled_back"
				e.logChange(change, "ROLLED BACK")
			} else {
				change.Status = "failed"
			}
			
			return err
		}
	}
	
	// Применяем изменения файлов
	for _, file := range change.Files {
		if err := e.applyFileChange(file); err != nil {
			e.logChange(change, fmt.Sprintf("FILE CHANGE FAILED: %s - %v", file.Path, err))
			
			if e.config.RollbackEnabled && change.BackupID != "" {
				e.rollback(change.BackupID)
				change.Status = "rolled_back"
				e.logChange(change, "ROLLED BACK")
			} else {
				change.Status = "failed"
			}
			
			return err
		}
	}
	
	change.Status = "completed"
	e.logChange(change, "COMPLETED")
	
	// Удаляем из ожидающих
	e.removePending(change.ID)
	
	return nil
}

// executeCommand выполняет команду
func (e *Engine) executeCommand(command string) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = e.projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	
	return nil
}

// applyFileChange применяет изменение файла
func (e *Engine) applyFileChange(change FileChange) error {
	switch change.Action {
	case "create", "modify":
		return os.WriteFile(change.Path, []byte(change.NewContent), 0644)
		
	case "delete":
		return os.Remove(change.Path)
		
	default:
		return fmt.Errorf("unknown action: %s", change.Action)
	}
}

// createBackup создаёт бэкап
func (e *Engine) createBackup(change *ChangeRequest) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	backupID := fmt.Sprintf("backup_%s_%s", timestamp, change.ID[:8])
	backupPath := filepath.Join(e.backupDir, backupID)
	
	os.MkdirAll(backupPath, 0755)
	
	// Бэкапим файлы
	for _, file := range change.Files {
		if file.Action == "modify" || file.Action == "delete" {
			src := file.Path
			dst := filepath.Join(backupPath, filepath.Base(file.Path)+".bak")
			
			data, err := os.ReadFile(src)
			if err != nil {
				return "", err
			}
			
			if err := os.WriteFile(dst, data, 0644); err != nil {
				return "", err
			}
		}
	}
	
	// Сохраняем метаданные
	meta := map[string]interface{}{
		"change_id":   change.ID,
		"timestamp":   timestamp,
		"description": change.Description,
	}
	
	metaPath := filepath.Join(backupPath, "metadata.json")
	data, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(metaPath, data, 0644)
	
	e.logChange(change, fmt.Sprintf("BACKUP CREATED: %s", backupID))
	
	return backupID, nil
}

// rollback откатывает изменение
func (e *Engine) rollback(backupID string) error {
	backupPath := filepath.Join(e.backupDir, backupID)
	
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup %s not found", backupID)
	}
	
	// Восстанавливаем файлы
	files, _ := filepath.Glob(filepath.Join(backupPath, "*.bak"))
	
	for _, file := range files {
		originalName := strings.TrimSuffix(filepath.Base(file), ".bak")
		originalPath := filepath.Join(e.projectRoot, originalName)
		
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		
		os.WriteFile(originalPath, data, 0644)
	}
	
	return nil
}

// findChange находит изменение по ID
func (e *Engine) findChange(changeID string) *ChangeRequest {
	for _, change := range e.pendingChanges {
		if change.ID == changeID {
			return change
		}
	}
	
	// Ищем в логе
	logChanges := e.loadLog()
	for _, change := range logChanges {
		if change.ID == changeID {
			return change
		}
	}
	
	return nil
}

// removePending удаляет из ожидающих
func (e *Engine) removePending(changeID string) {
	for i, change := range e.pendingChanges {
		if change.ID == changeID {
			e.pendingChanges = append(e.pendingChanges[:i], e.pendingChanges[i+1:]...)
			return
		}
	}
}

// GetPendingChanges возвращает ожидающие изменения
func (e *Engine) GetPendingChanges() []*ChangeRequest {
	return e.pendingChanges
}

// logChange логирует изменение
func (e *Engine) logChange(change *ChangeRequest, event string) {
	f, err := os.OpenFile(e.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	
	timestamp := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf("[%s] %s [%s] %s: %s\n",
		timestamp, change.ID, change.Status, event, change.Description)
	
	f.WriteString(line)
}

// loadLog загружает журнал изменений
func (e *Engine) loadLog() []*ChangeRequest {
	changes := make([]*ChangeRequest, 0)
	
	data, err := os.ReadFile(e.logFile)
	if err != nil {
		return changes
	}
	
	lines := strings.Split(string(data), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		// Парсим строку лога
		// Формат: [timestamp] id [status] event: description
		parts := strings.SplitN(line, "]", 4)
		if len(parts) < 4 {
			continue
		}
		
		change := &ChangeRequest{}
		// Извлекаем ID
		idPart := strings.TrimPrefix(parts[1], " ")
		change.ID = strings.Split(idPart, " ")[0]
		
		// Извлекаем статус
		statusPart := strings.TrimPrefix(parts[2], " ")
		change.Status = strings.Trim(statusPart, "[] ")
		
		changes = append(changes, change)
	}
	
	return changes
}

// generateID генерирует уникальный ID
func generateID() string {
	return fmt.Sprintf("change_%d", time.Now().UnixNano())
}

// FormatChangeRequest форматирует запрос на подтверждение
func FormatChangeRequest(change *ChangeRequest) string {
	var sb strings.Builder
	
	sb.WriteString("🔧 **Запрос на изменение**\n\n")
	sb.WriteString(fmt.Sprintf("**ID:** `%s`\n", change.ID))
	sb.WriteString(fmt.Sprintf("**Тип:** %s\n", change.Type))
	sb.WriteString(fmt.Sprintf("**Описание:** %s\n", change.Description))
	sb.WriteString(fmt.Sprintf("**Причина:** %s\n\n", change.Reason))
	
	if len(change.Commands) > 0 {
		sb.WriteString("**Команды:**\n")
		for _, cmd := range change.Commands {
			sb.WriteString(fmt.Sprintf("```\n%s\n```\n", cmd))
		}
		sb.WriteString("\n")
	}
	
	if len(change.Files) > 0 {
		sb.WriteString("**Файлы:**\n")
		for _, file := range change.Files {
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", file.Path, file.Action))
		}
		sb.WriteString("\n")
	}
	
	if change.RequiresConfirmation {
		sb.WriteString("⚠️ **Требуется ваше подтверждение!**\n\n")
		sb.WriteString("Для подтверждения напишите: `ПОДТВЕРЖДАЮ <ID>`\n")
		sb.WriteString("Для отмены напишите: `ОТМЕНА <ID>`\n")
	}
	
	return sb.String()
}
