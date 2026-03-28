// Package session содержит модели данных для Session & Memory Service
package session

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// generateID генерирует уникальный ID
func generateID() string {
	return uuid.New().String()
}

// FormatTimestamp форматирует время для вывода
func FormatTimestamp(t time.Time) string {
	return t.Format(time.RFC3339)
}

// EstimateTokens оценивает количество токенов в тексте
// Приблизительно: 1 токен = 4 символа для английского текста
func EstimateTokens(text string) int32 {
	if len(text) == 0 {
		return 0
	}
	return int32(len(text) / 4)
}

// TruncateMessage обрезает сообщение до максимальной длины
func TruncateMessage(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// SanitizeRole проверяет и нормализует роль сообщения
func SanitizeRole(role string) string {
	switch role {
	case "user", "assistant", "system":
		return role
	default:
		return "user"
	}
}

// ValidateSession проверяет валидность сессии
func ValidateSession(s *Session) error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	if s.ID == "" {
		return fmt.Errorf("session ID is empty")
	}
	if s.UserID == "" {
		return fmt.Errorf("user ID is empty")
	}
	return nil
}

// ValidateMessage проверяет валидность сообщения
func ValidateMessage(m *Message) error {
	if m == nil {
		return fmt.Errorf("message is nil")
	}
	if m.SessionID == "" {
		return fmt.Errorf("session ID is empty")
	}
	if m.Content == "" {
		return fmt.Errorf("message content is empty")
	}
	if m.Role == "" {
		return fmt.Errorf("message role is empty")
	}
	return nil
}

// ValidateMemoryEntry проверяет валидность записи памяти
func ValidateMemoryEntry(m *MemoryEntry) error {
	if m == nil {
		return fmt.Errorf("memory entry is nil")
	}
	if m.UserID == "" {
		return fmt.Errorf("user ID is empty")
	}
	if m.Content == "" {
		return fmt.Errorf("memory content is empty")
	}
	if m.Type == "" {
		return fmt.Errorf("memory type is empty")
	}
	return nil
}
