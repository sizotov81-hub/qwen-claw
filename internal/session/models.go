// Package session содержит модели данных для Session & Memory Service
package session

import (
	"time"
)

// Session представляет сессию пользователя
type Session struct {
	ID             string            `json:"id"`
	UserID         string            `json:"user_id"`
	Title          string            `json:"title"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	LastAccessedAt time.Time         `json:"last_accessed_at"`
	Metadata       map[string]string `json:"metadata"`
	MessageCount   int32             `json:"message_count"`
	TotalTokens    int32             `json:"total_tokens"`
}

// Message представляет сообщение в сессии
type Message struct {
	ID        string            `json:"id"`
	SessionID string            `json:"session_id"`
	Role      string            `json:"role"` // user/assistant/system
	Content   string            `json:"content"`
	Tokens    int32             `json:"tokens"`
	CreatedAt time.Time         `json:"created_at"`
	Metadata  map[string]string `json:"metadata"`
}

// MemoryType тип записи памяти
type MemoryType string

const (
	MemoryTypeWorking   MemoryType = "working"
	MemoryTypeEpisodic  MemoryType = "episodic"
	MemoryTypeSemantic  MemoryType = "semantic"
	MemoryTypeProcedural MemoryType = "procedural"
)

// MemoryEntry представляет запись в памяти
type MemoryEntry struct {
	ID             string            `json:"id"`
	SessionID      string            `json:"session_id"`
	UserID         string            `json:"user_id"`
	Type           MemoryType        `json:"type"`
	Category       string            `json:"category"`
	Content        string            `json:"content"`
	Embedding      []float32         `json:"embedding"` // векторное представление
	Importance     float32           `json:"importance"`
	Retention      float32           `json:"retention"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	LastAccessedAt time.Time         `json:"last_accessed_at"`
	AccessCount    int32             `json:"access_count"`
	Metadata       map[string]string `json:"metadata"`
}

// NewSession создаёт новую сессию
func NewSession(userID, title string) *Session {
	now := time.Now()
	return &Session{
		ID:             generateID(),
		UserID:         userID,
		Title:          title,
		CreatedAt:      now,
		UpdatedAt:      now,
		LastAccessedAt: now,
		Metadata:       make(map[string]string),
		MessageCount:   0,
		TotalTokens:    0,
	}
}

// NewMessage создаёт новое сообщение
func NewMessage(sessionID, role, content string, tokens int32) *Message {
	return &Message{
		ID:        generateID(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Tokens:    tokens,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// NewMemoryEntry создаёт новую запись памяти
func NewMemoryEntry(userID, sessionID, content string, memType MemoryType) *MemoryEntry {
	now := time.Now()
	return &MemoryEntry{
		ID:             generateID(),
		UserID:         userID,
		SessionID:      sessionID,
		Type:           memType,
		Content:        content,
		Importance:     0.5,
		Retention:      1.0,
		CreatedAt:      now,
		UpdatedAt:      now,
		LastAccessedAt: now,
		AccessCount:    0,
		Metadata:       make(map[string]string),
	}
}

// AddToken добавляет токен к сессии
func (s *Session) AddToken(tokens int32) {
	s.TotalTokens += tokens
	s.MessageCount++
	s.UpdatedAt = time.Now()
}

// Touch обновляет время последнего доступа
func (s *Session) Touch() {
	s.LastAccessedAt = time.Now()
}

// Access обновляет время доступа и счётчик для памяти
func (m *MemoryEntry) Access() {
	m.LastAccessedAt = time.Now()
	m.AccessCount++
}
