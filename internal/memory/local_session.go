package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LocalSession локальная сессия (приватная, не синхронизируется)
type LocalSession struct {
	mu sync.RWMutex

	// ID сессии
	ID string `json:"id"`

	// История диалога
	ConversationHistory []string `json:"conversation_history"`

	// Контекст
	Context map[string]interface{} `json:"context"`

	// Время создания
	Created time.Time `json:"created"`

	// Время последнего доступа
	LastAccess time.Time `json:"last_access"`

	// Путь к файлу
	filePath string
}

// LocalSessionManager менеджер локальных сессий
type LocalSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*LocalSession
	dataDir  string
}

// NewLocalSessionManager создаёт менеджер локальных сессий
func NewLocalSessionManager(dataDir string) (*LocalSessionManager, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}

	return &LocalSessionManager{
		sessions: make(map[string]*LocalSession),
		dataDir:  dataDir,
	}, nil
}

// CreateSession создаёт новую сессию
func (m *LocalSessionManager) CreateSession(id string) (*LocalSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &LocalSession{
		ID:                  id,
		ConversationHistory: make([]string, 0),
		Context:             make(map[string]interface{}),
		Created:             time.Now(),
		LastAccess:          time.Now(),
		filePath:            filepath.Join(m.dataDir, id+".json"),
	}

	m.sessions[id] = session

	// Сохраняем на диск
	if err := session.save(); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession получает сессию
func (m *LocalSessionManager) GetSession(id string) (*LocalSession, error) {
	m.mu.RLock()
	if session, ok := m.sessions[id]; ok {
		m.mu.RUnlock()
		return session, nil
	}
	m.mu.RUnlock()

	// Загружаем с диска
	filePath := filepath.Join(m.dataDir, id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return m.CreateSession(id)
	}

	session := &LocalSession{}
	if err := json.Unmarshal(data, session); err != nil {
		return m.CreateSession(id)
	}

	session.filePath = filePath
	session.LastAccess = time.Now()

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	return session, nil
}

// AddToHistory добавляет сообщение в историю сессии
func (s *LocalSession) AddToHistory(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ConversationHistory = append(s.ConversationHistory, message)
	s.LastAccess = time.Now()

	// Сохраняем каждые 10 сообщений
	if len(s.ConversationHistory)%10 == 0 {
		s.save()
	}
}

// GetHistory возвращает историю сессии
func (s *LocalSession) GetHistory() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := make([]string, len(s.ConversationHistory))
	copy(history, s.ConversationHistory)
	return history
}

// GetRecentHistory возвращает последние N сообщений
func (s *LocalSession) GetRecentHistory(n int) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.ConversationHistory) <= n {
		return s.ConversationHistory
	}

	return s.ConversationHistory[len(s.ConversationHistory)-n:]
}

// SetContext устанавливает значение в контексте
func (s *LocalSession) SetContext(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Context[key] = value
	s.LastAccess = time.Now()
}

// GetContext получает значение из контекста
func (s *LocalSession) GetContext(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.Context[key]
	return value, ok
}

// Clear очищает сессию
func (s *LocalSession) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ConversationHistory = make([]string, 0)
	s.Context = make(map[string]interface{})
	s.LastAccess = time.Now()
	s.save()
}

// save сохраняет сессию на диск
func (s *LocalSession) save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	// Сохраняем с безопасными правами (только владелец)
	return os.WriteFile(s.filePath, data, 0600)
}

// Delete удаляет сессию
func (m *LocalSessionManager) DeleteSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[id]; ok {
		os.Remove(session.filePath)
		delete(m.sessions, id)
	}

	return nil
}

// ListSessions возвращает список всех сессий
func (m *LocalSessionManager) ListSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}

	return ids
}

// Cleanup удаляет старые сессии
func (m *LocalSessionManager) Cleanup(maxAge time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, session := range m.sessions {
		if now.Sub(session.LastAccess) > maxAge {
			os.Remove(session.filePath)
			delete(m.sessions, id)
		}
	}

	return nil
}
