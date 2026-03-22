package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SessionType тип сессии
type SessionType string

const (
	SessionTypeMain   SessionType = "main"
	SessionTypeDM     SessionType = "dm"
	SessionTypeGroup  SessionType = "group"
	SessionTypeCLI    SessionType = "cli"
	SessionTypeWeb    SessionType = "web"
)

// SessionStatus статус сессии
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusIdle      SessionStatus = "idle"
	SessionStatusSuspended SessionStatus = "suspended"
)

// GatewaySession сессия Gateway
type GatewaySession struct {
	mu sync.RWMutex

	// ID сессии
	ID string `json:"id"`

	// Type тип сессии
	Type SessionType `json:"type"`

	// Status статус сессии
	Status SessionStatus `json:"status"`

	// ClientID ID подключённого клиента
	ClientID string `json:"client_id"`

	// Channel канал связи (telegram/whatsapp/cli/web)
	Channel string `json:"channel"`

	// Metadata дополнительные метаданные
	Metadata map[string]interface{} `json:"metadata"`

	// Created время создания
	Created time.Time `json:"created"`

	// LastAccess время последнего доступа
	LastAccess time.Time `json:"last_access"`

	// LastActivity время последней активности
	LastActivity time.Time `json:"last_activity"`

	// MessageCount количество сообщений
	MessageCount int `json:"message_count"`

	// filePath путь к файлу сессии
	filePath string
}

// SessionManager менеджер сессий Gateway
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*GatewaySession
	dataDir  string
}

// NewSessionManager создаёт новый менеджер сессий
func NewSessionManager(dataDir string) (*SessionManager, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}

	sm := &SessionManager{
		sessions: make(map[string]*GatewaySession),
		dataDir:  dataDir,
	}

	// Загружаем существующие сессии
	if err := sm.loadSessions(); err != nil {
		return nil, err
	}

	return sm, nil
}

// CreateSession создаёт новую сессию
func (m *SessionManager) CreateSession(id string, sessionType SessionType, channel string) (*GatewaySession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &GatewaySession{
		ID:           id,
		Type:         sessionType,
		Status:       SessionStatusActive,
		Channel:      channel,
		Metadata:     make(map[string]interface{}),
		Created:      time.Now(),
		LastAccess:   time.Now(),
		LastActivity: time.Now(),
		MessageCount: 0,
		filePath:     filepath.Join(m.dataDir, id+".json"),
	}

	m.sessions[id] = session

	// Сохраняем на диск
	if err := session.save(); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession получает сессию по ID
func (m *SessionManager) GetSession(id string) (*GatewaySession, error) {
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
		// Создаём новую сессию
		return m.CreateSession(id, SessionTypeMain, "unknown")
	}

	session := &GatewaySession{}
	if err := json.Unmarshal(data, session); err != nil {
		return m.CreateSession(id, SessionTypeMain, "unknown")
	}

	session.filePath = filePath
	session.LastAccess = time.Now()

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	return session, nil
}

// GetSessionByClientID получает сессию по ID клиента
func (m *SessionManager) GetSessionByClientID(clientID string) *GatewaySession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, session := range m.sessions {
		if session.ClientID == clientID {
			return session
		}
	}

	return nil
}

// ListSessions возвращает список всех сессий
func (m *SessionManager) ListSessions() []*GatewaySession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*GatewaySession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// ListSessionsByChannel возвращает сессии по каналу
func (m *SessionManager) ListSessionsByChannel(channel string) []*GatewaySession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*GatewaySession, 0)
	for _, session := range m.sessions {
		if session.Channel == channel {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// RemoveSession удаляет сессию
func (m *SessionManager) RemoveSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[id]; ok {
		os.Remove(session.filePath)
		delete(m.sessions, id)
	}

	return nil
}

// loadSessions загружает все сессии с диска
func (m *SessionManager) loadSessions() error {
	entries, err := os.ReadDir(m.dataDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(m.dataDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		session := &GatewaySession{}
		if err := json.Unmarshal(data, session); err != nil {
			continue
		}

		session.filePath = filePath
		m.sessions[session.ID] = session
	}

	return nil
}

// save сохраняет сессию на диск
func (s *GatewaySession) save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0600)
}

// UpdateActivity обновляет время активности
func (s *GatewaySession) UpdateActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastActivity = time.Now()
	s.MessageCount++
	s.save()
}

// SetClientID устанавливает ID клиента
func (s *GatewaySession) SetClientID(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ClientID = clientID
	s.LastAccess = time.Now()
	s.save()
}

// SetStatus устанавливает статус сессии
func (s *GatewaySession) SetStatus(status SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = status
	s.LastAccess = time.Now()
	s.save()
}

// GetMetadata получает значение из метаданных
func (s *GatewaySession) GetMetadata(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.Metadata[key]
	return value, ok
}

// SetMetadata устанавливает значение в метаданных
func (s *GatewaySession) SetMetadata(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Metadata[key] = value
	s.LastAccess = time.Now()
	s.save()
}

// GetActiveSessions возвращает активные сессии
func (m *SessionManager) GetActiveSessions() []*GatewaySession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*GatewaySession, 0)
	for _, session := range m.sessions {
		if session.Status == SessionStatusActive {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// Cleanup удаляет неактивные сессии
func (m *SessionManager) Cleanup(maxAge time.Duration) error {
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
