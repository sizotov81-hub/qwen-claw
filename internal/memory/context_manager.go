package memory

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ContextStatus статус использования контекста
type ContextStatus string

const (
	ContextStatusNormal     ContextStatus = "normal"      // < 50%
	ContextStatusWarning    ContextStatus = "warning"     // 50-75%
	ContextStatusCritical   ContextStatus = "critical"    // 75-90%
	ContextStatusOverflow   ContextStatus = "overflow"    // > 90%
)

// ContextConfig конфигурация управления контекстом
type ContextConfig struct {
	// MaxTokens максимальное количество токенов
	MaxTokens int `json:"max_tokens"`

	// WarningThreshold порог предупреждения (0-1)
	WarningThreshold float64 `json:"warning_threshold"` // 0.5

	// CriticalThreshold критический порог (0-1)
	CriticalThreshold float64 `json:"critical_threshold"` // 0.75

	// OverflowThreshold порог переполнения (0-1)
	OverflowThreshold float64 `json:"overflow_threshold"` // 0.9

	// AutoCompress автоматически сжимать при критическом уровне
	AutoCompress bool `json:"auto_compress"`

	// AutoNewSession автоматически создавать новую сессию при переполнении
	AutoNewSession bool `json:"auto_new_session"`

	// PreserveIncomplete сохранять незавершённые задачи при смене сессии
	PreserveIncomplete bool `json:"preserve_incomplete"`
}

// DefaultContextConfig конфигурация по умолчанию
func DefaultContextConfig() *ContextConfig {
	return &ContextConfig{
		MaxTokens:          8000, // Типичный лимит для LLM
		WarningThreshold:   0.5,
		CriticalThreshold:  0.75,
		OverflowThreshold:  0.9,
		AutoCompress:       true,
		AutoNewSession:     true,
		PreserveIncomplete: true,
	}
}

// ContextToken токен контекста
type ContextToken struct {
	// Content содержимое
	Content string `json:"content"`

	// Type тип (user/assistant/system)
	Type string `json:"type"`

	// Tokens количество токенов
	Tokens int `json:"tokens"`

	// Timestamp время создания
	Timestamp time.Time `json:"timestamp"`

	// Priority приоритет (0-10, 10 = самый важный)
	Priority int `json:"priority"`

	// TaskID ID связанной задачи
	TaskID string `json:"task_id"`

	// IsIncomplete задача не завершена
	IsIncomplete bool `json:"is_incomplete"`
}

// ContextManager менеджер контекста
type ContextManager struct {
	mu sync.RWMutex

	// Токены контекста
	tokens []ContextToken

	// Текущее количество токенов
	currentTokens int

	// Конфигурация
	config *ContextConfig

	// Callbacks
	onWarning    func(status ContextStatus, usage float64)
	onCompress   func() error
	onNewSession func() error
}

// NewContextManager создаёт новый менеджер контекста
func NewContextManager(config *ContextConfig) *ContextManager {
	if config == nil {
		config = DefaultContextConfig()
	}

	return &ContextManager{
		tokens:        make([]ContextToken, 0),
		currentTokens: 0,
		config:        config,
	}
}

// AddToken добавляет токен в контекст
func (m *ContextManager) AddToken(token ContextToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Оцениваем количество токенов (примерно: 1 токен = 4 символа)
	if token.Tokens == 0 {
		token.Tokens = len(token.Content) / 4
	}

	// Проверяем не переполнится ли контекст
	if m.currentTokens+token.Tokens > m.config.MaxTokens {
		// Пытаемся освободить место
		if err := m.makeRoom(token.Tokens); err != nil {
			return fmt.Errorf("context overflow: %w", err)
		}
	}

	m.tokens = append(m.tokens, token)
	m.currentTokens += token.Tokens

	// Проверяем статус
	status := m.GetStatus()
	if status != ContextStatusNormal && m.onWarning != nil {
		usage := float64(m.currentTokens) / float64(m.config.MaxTokens)
		m.onWarning(status, usage)
	}

	return nil
}

// AddMessage добавляет сообщение в контекст
func (m *ContextManager) AddMessage(content, msgType string, taskID string, isIncomplete bool) error {
	token := ContextToken{
		Content:      content,
		Type:         msgType,
		Timestamp:    time.Now(),
		Priority:     5, // Средний приоритет
		TaskID:       taskID,
		IsIncomplete: isIncomplete,
	}

	return m.AddToken(token)
}

// GetStatus возвращает текущий статус контекста
func (m *ContextManager) GetStatus() ContextStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	usage := float64(m.currentTokens) / float64(m.config.MaxTokens)

	if usage >= m.config.OverflowThreshold {
		return ContextStatusOverflow
	}
	if usage >= m.config.CriticalThreshold {
		return ContextStatusCritical
	}
	if usage >= m.config.WarningThreshold {
		return ContextStatusWarning
	}
	return ContextStatusNormal
}

// GetUsage возвращает информацию об использовании контекста
func (m *ContextManager) GetUsage() ContextUsage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	usage := float64(m.currentTokens) / float64(m.config.MaxTokens)

	return ContextUsage{
		CurrentTokens:   m.currentTokens,
		MaxTokens:       m.config.MaxTokens,
		UsagePercent:    usage * 100,
		Status:          m.GetStatus(),
		TokenCount:      len(m.tokens),
		AvailableTokens: m.config.MaxTokens - m.currentTokens,
	}
}

// ContextUsage информация об использовании контекста
type ContextUsage struct {
	CurrentTokens   int           `json:"current_tokens"`
	MaxTokens       int           `json:"max_tokens"`
	UsagePercent    float64       `json:"usage_percent"`
	Status          ContextStatus `json:"status"`
	TokenCount      int           `json:"token_count"`
	AvailableTokens int           `json:"available_tokens"`
}

// Compress сжимает контекст
func (m *ContextManager) Compress() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Стратегия сжатия:
	// 1. Удаляем старые токены с низким приоритетом
	// 2. Объединяем последовательные сообщения одного типа
	// 3. Сохраняем токены с IsIncomplete=true

	newTokens := make([]ContextToken, 0)
	preservedTokens := 0

	for _, token := range m.tokens {
		// Сохраняем важные токены
		if token.Priority >= 8 || token.IsIncomplete {
			newTokens = append(newTokens, token)
			preservedTokens++
			continue
		}

		// Удаляем старые токены с низким приоритетом
		if token.Priority < 5 && time.Since(token.Timestamp) > 10*time.Minute {
			continue
		}

		newTokens = append(newTokens, token)
	}

	// Пересчитываем токены
	m.tokens = newTokens
	m.currentTokens = 0
	for _, t := range m.tokens {
		m.currentTokens += t.Tokens
	}

	if m.onCompress != nil {
		return m.onCompress()
	}

	return nil
}

// makeRoom освобождает место для новых токенов
func (m *ContextManager) makeRoom(needed int) error {
	// Проверяем статус
	status := m.GetStatus()

	switch status {
	case ContextStatusOverflow:
		// Критическое переполнение — создаём новую сессию
		if m.config.AutoNewSession && m.onNewSession != nil {
			return m.onNewSession()
		}
		// Пытаемся сжать
		if m.config.AutoCompress {
			return m.Compress()
		}
		return fmt.Errorf("context overflow, cannot make room")

	case ContextStatusCritical:
		// Критический уровень — сжимаем
		if m.config.AutoCompress {
			return m.Compress()
		}
		// Удаляем старые токены с низким приоритетом
		return m.removeOldLowPriority(needed)

	case ContextStatusWarning:
		// Предупреждение — удаляем только очень старые
		return m.removeOldLowPriority(needed)

	default:
		// Нормальный уровень — просто удаляем самые старые
		return m.removeOldest(needed)
	}
}

// removeOldLowPriority удаляет старые токены с низким приоритетом
func (m *ContextManager) removeOldLowPriority(needed int) error {
	removed := 0
	newTokens := make([]ContextToken, 0)

	// Сортируем по приоритету и времени
	for i := len(m.tokens) - 1; i >= 0; i-- {
		token := m.tokens[i]
		if removed >= needed {
			newTokens = append([]ContextToken{token}, newTokens...)
			continue
		}

		// Сохраняем важные токены
		if token.Priority >= 7 || token.IsIncomplete {
			newTokens = append([]ContextToken{token}, newTokens...)
			continue
		}

		// Удаляем старые с низким приоритетом
		if token.Priority < 5 && time.Since(token.Timestamp) > 5*time.Minute {
			removed += token.Tokens
			continue
		}

		newTokens = append([]ContextToken{token}, newTokens...)
	}

	m.tokens = newTokens
	m.currentTokens -= removed
	return nil
}

// removeOldest удаляет самые старые токены
func (m *ContextManager) removeOldest(needed int) error {
	removed := 0
	newTokens := make([]ContextToken, 0)

	for i := len(m.tokens) - 1; i >= 0; i-- {
		token := m.tokens[i]
		if removed >= needed {
			newTokens = append([]ContextToken{token}, newTokens...)
			continue
		}

		// Сохраняем важные токены
		if token.Priority >= 8 || token.IsIncomplete {
			newTokens = append([]ContextToken{token}, newTokens...)
			continue
		}

		removed += token.Tokens
	}

	m.tokens = newTokens
	m.currentTokens -= removed
	return nil
}

// GetIncompleteTasks возвращает незавершённые задачи в контексте
func (m *ContextManager) GetIncompleteTasks() []ContextToken {
	m.mu.RLock()
	defer m.mu.RUnlock()

	incomplete := make([]ContextToken, 0)
	for _, token := range m.tokens {
		if token.IsIncomplete {
			incomplete = append(incomplete, token)
		}
	}
	return incomplete
}

// GetTaskContext возвращает контекст для конкретной задачи
func (m *ContextManager) GetTaskContext(taskID string) []ContextToken {
	m.mu.RLock()
	defer m.mu.RUnlock()

	taskTokens := make([]ContextToken, 0)
	for _, token := range m.tokens {
		if token.TaskID == taskID {
			taskTokens = append(taskTokens, token)
		}
	}
	return taskTokens
}

// ClearTaskContext очищает контекст задачи
func (m *ContextManager) ClearTaskContext(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newTokens := make([]ContextToken, 0)
	removed := 0

	for _, token := range m.tokens {
		if token.TaskID == taskID && !token.IsIncomplete {
			removed += token.Tokens
			continue
		}
		newTokens = append(newTokens, token)
	}

	m.tokens = newTokens
	m.currentTokens -= removed
}

// SetOnWarning устанавливает callback для предупреждений
func (m *ContextManager) SetOnWarning(fn func(status ContextStatus, usage float64)) {
	m.onWarning = fn
}

// SetOnCompress устанавливает callback для сжатия
func (m *ContextManager) SetOnCompress(fn func() error) {
	m.onCompress = fn
}

// SetOnNewSession устанавливает callback для новой сессии
func (m *ContextManager) SetOnNewSession(fn func() error) {
	m.onNewSession = fn
}

// GetContextForQuery возвращает отформатированный контекст для запроса
func (m *ContextManager) GetContextForQuery() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.tokens) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("📋 Контекст диалога:\n\n")

	for _, token := range m.tokens {
		prefix := "💬"
		if token.Type == "user" {
			prefix = "👤"
		} else if token.Type == "assistant" {
			prefix = "🤖"
		} else if token.Type == "system" {
			prefix = "⚙️"
		}

		sb.WriteString(fmt.Sprintf("%s %s\n", prefix, token.Content))
	}

	return sb.String()
}

// Reset сбрасывает контекст
func (m *ContextManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokens = make([]ContextToken, 0)
	m.currentTokens = 0
}

// EstimateTokens оценивает количество токенов в тексте
func EstimateTokens(text string) int {
	// Приблизительная оценка: 1 токен = 4 символа
	return len(text) / 4
}
