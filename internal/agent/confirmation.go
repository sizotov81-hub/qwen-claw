package agent

import (
	"fmt"
	"sync"
	"time"
)

// ConfirmationMode режим подтверждения
type ConfirmationMode string

const (
	// ModePlan - всегда спрашивать подтверждение
	ModePlan ConfirmationMode = "plan"

	// ModeAutoEdit - спрашивать для опасных действий
	ModeAutoEdit ConfirmationMode = "auto-edit"

	// ModeYolo - никогда не спрашивать
	ModeYolo ConfirmationMode = "yolo"
)

// PendingAction ожидающее подтверждение действие
type PendingAction struct {
	// ID уникальный идентификатор
	ID string `json:"id"`

	// Intent намерение
	Intent *Intent `json:"intent"`

	// Query исходный запрос
	Query string `json:"query"`

	// Created время создания
	Created time.Time `json:"created"`

	// Expires время истечения
	Expires time.Time `json:"expires"`

	// Confirmed флаг подтверждения
	Confirmed bool `json:"confirmed"`

	// Rejected флаг отклонения
	Rejected bool `json:"rejected"`
}

// IsExpired проверяет, истекло ли действие
func (a *PendingAction) IsExpired() bool {
	return time.Now().After(a.Expires)
}

// IsPending проверяет, ожидает ли действие подтверждения
func (a *PendingAction) IsPending() bool {
	return !a.Confirmed && !a.Rejected && !a.IsExpired()
}

// ConfirmationManager менеджер подтверждений
type ConfirmationManager struct {
	// pending ожидающие действия
	pending map[string]*PendingAction

	// mode режим подтверждения
	mode ConfirmationMode

	// expirationTime время жизни подтверждения
	expirationTime time.Duration

	// mu мьютекс
	mu sync.RWMutex
}

// ConfirmationManagerConfig конфигурация менеджера подтверждений
type ConfirmationManagerConfig struct {
	// Mode режим подтверждения
	Mode ConfirmationMode `json:"mode"`

	// ExpirationTime время жизни подтверждения
	ExpirationTime time.Duration `json:"expiration_time"`
}

// DefaultConfirmationManagerConfig конфигурация по умолчанию
func DefaultConfirmationManagerConfig() ConfirmationManagerConfig {
	return ConfirmationManagerConfig{
		Mode:           ModeAutoEdit,
		ExpirationTime: 5 * time.Minute,
	}
}

// NewConfirmationManager создаёт новый менеджер подтверждений
func NewConfirmationManager(config ConfirmationManagerConfig) *ConfirmationManager {
	if config.ExpirationTime == 0 {
		config.ExpirationTime = 5 * time.Minute
	}

	return &ConfirmationManager{
		pending:        make(map[string]*PendingAction),
		mode:           config.Mode,
		expirationTime: config.ExpirationTime,
	}
}

// CreatePending создаёт ожидающее действие
func (m *ConfirmationManager) CreatePending(intent *Intent, query string) *PendingAction {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := generateActionID()
	action := &PendingAction{
		ID:        id,
		Intent:    intent,
		Query:     query,
		Created:   time.Now(),
		Expires:   time.Now().Add(m.expirationTime),
		Confirmed: false,
		Rejected:  false,
	}

	m.pending[id] = action

	// Очищаем истёкшие действия
	m.cleanupExpired()

	return action
}

// GetPending получает ожидающее действие по ID
func (m *ConfirmationManager) GetPending(id string) *PendingAction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	action, ok := m.pending[id]
	if !ok {
		return nil
	}

	return action
}

// Confirm подтверждает действие
func (m *ConfirmationManager) Confirm(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	action, ok := m.pending[id]
	if !ok {
		return false
	}

	if action.IsExpired() {
		return false
	}

	action.Confirmed = true
	return true
}

// Reject отклоняет действие
func (m *ConfirmationManager) Reject(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	action, ok := m.pending[id]
	if !ok {
		return false
	}

	if action.IsExpired() {
		return false
	}

	action.Rejected = true
	return true
}

// RequiresConfirmation проверяет, требует ли действие подтверждения
func (m *ConfirmationManager) RequiresConfirmation(intent *Intent) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch m.mode {
	case ModeYolo:
		// В режиме yolo никогда не спрашиваем
		return false

	case ModePlan:
		// В режиме plan всегда спрашиваем
		return true

	case ModeAutoEdit:
		// В режиме auto-edit спрашиваем для опасных действий
		return m.isDangerousIntent(intent)

	default:
		return false
	}
}

// isDangerousIntent проверяет, является ли намерение опасным
func (m *ConfirmationManager) isDangerousIntent(intent *Intent) bool {
	// Опасные типы намерений
	dangerousTypes := []IntentType{
		IntentFileWrite,
		IntentShellExec,
		IntentTaskRemove,
	}

	for _, t := range dangerousTypes {
		if intent.Type == t {
			return true
		}
	}

	// Проверяем данные на опасные команды
	if m.containsDangerousCommand(intent.Data) {
		return true
	}

	return false
}

// containsDangerousCommand проверяет, содержит ли команда опасные действия
func (m *ConfirmationManager) containsDangerousCommand(data string) bool {
	dangerousPatterns := []string{
		"rm -rf",
		"rm -r",
		"rm ",
		"del ",
		"delete ",
		"drop ",
		"chmod 777",
		"chmod -R",
		"chown ",
		"sudo",
		">>",
		"> ",
		"|",
	}

	for _, pattern := range dangerousPatterns {
		if containsIgnoreCase(data, pattern) {
			return true
		}
	}

	return false
}

// GetPendingActions получает все ожидающие действия
func (m *ConfirmationManager) GetPendingActions() []*PendingAction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	actions := make([]*PendingAction, 0)
	for _, action := range m.pending {
		if action.IsPending() {
			actions = append(actions, action)
		}
	}

	return actions
}

// Cleanup очищает истёкшие и обработанные действия
func (m *ConfirmationManager) Cleanup() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, action := range m.pending {
		if action.IsExpired() || action.Confirmed || action.Rejected {
			delete(m.pending, id)
			count++
		}
	}

	return count
}

// cleanupExpired очищает истёкшие действия (должен вызываться с захваченным мьютексом)
func (m *ConfirmationManager) cleanupExpired() {
	for id, action := range m.pending {
		if action.IsExpired() {
			delete(m.pending, id)
		}
	}
}

// GetMode возвращает текущий режим
func (m *ConfirmationManager) GetMode() ConfirmationMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mode
}

// SetMode устанавливает режим
func (m *ConfirmationManager) SetMode(mode ConfirmationMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mode = mode
}

// GetStats возвращает статистику
func (m *ConfirmationManager) GetStats() ConfirmationStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pending, confirmed, rejected, expired int
	for _, action := range m.pending {
		if action.IsExpired() {
			expired++
		} else if action.Confirmed {
			confirmed++
		} else if action.Rejected {
			rejected++
		} else {
			pending++
		}
	}

	return ConfirmationStats{
		Pending:   pending,
		Confirmed: confirmed,
		Rejected:  rejected,
		Expired:   expired,
		Total:     len(m.pending),
	}
}

// ConfirmationStats статистика подтверждений
type ConfirmationStats struct {
	Pending   int `json:"pending"`
	Confirmed int `json:"confirmed"`
	Rejected  int `json:"rejected"`
	Expired   int `json:"expired"`
	Total     int `json:"total"`
}

// generateActionID генерирует уникальный ID для действия
func generateActionID() string {
	return fmt.Sprintf("action_%d", time.Now().UnixNano())
}

// containsIgnoreCase проверяет, содержит ли строка подстроку без учёта регистра
func containsIgnoreCase(s, substr string) bool {
	return contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if toLower(s[i:i+len(substr)]) == toLower(substr) {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
