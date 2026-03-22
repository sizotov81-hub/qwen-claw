package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventType тип события
type EventType string

const (
	EventUserMessage    EventType = "user_message"
	EventAssistantReply EventType = "assistant_reply"
	EventSystemMessage  EventType = "system_message"
	EventToolCall       EventType = "tool_call"
	EventToolResult     EventType = "tool_result"
	EventError          EventType = "error"
	EventSummary        EventType = "summary"
)

// Event событие сессии
type Event struct {
	// ID уникальный идентификатор события
	ID string `json:"id"`

	// Type тип события
	Type EventType `json:"type"`

	// Content содержимое события
	Content string `json:"content"`

	// Metadata дополнительные данные
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Tokens количество токенов
	Tokens int `json:"tokens,omitempty"`

	// Timestamp временная метка
	Timestamp time.Time `json:"timestamp"`
}

// SessionEventStore хранилище событий сессии
type SessionEventStore struct {
	mu sync.RWMutex

	// SessionID ID сессии
	SessionID string `json:"session_id"`

	// Events список событий
	Events []Event `json:"events"`

	// TotalTokens общее количество токенов
	TotalTokens int `json:"total_tokens"`

	// Summary суммаризация старых событий
	Summary string `json:"summary,omitempty"`

	// SummaryFrom индекс, с которого начинается суммаризация
	SummaryFrom int `json:"summary_from"`

	// Created время создания
	Created time.Time `json:"created"`

	// LastAccess время последнего доступа
	LastAccess time.Time `json:"last_access"`

	// filePath путь к файлу
	filePath string
}

// SessionStoreManager менеджер хранилищ событий
type SessionStoreManager struct {
	mu      sync.RWMutex
	stores  map[string]*SessionEventStore
	dataDir string
}

// NewSessionStoreManager создаёт менеджер хранилищ событий
func NewSessionStoreManager(dataDir string) (*SessionStoreManager, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}

	sm := &SessionStoreManager{
		stores:  make(map[string]*SessionEventStore),
		dataDir: dataDir,
	}

	// Загружаем существующие хранилища
	if err := sm.loadStores(); err != nil {
		return nil, err
	}

	return sm, nil
}

// GetStore получает хранилище сессии
func (m *SessionStoreManager) GetStore(sessionID string) (*SessionEventStore, error) {
	m.mu.RLock()
	if store, ok := m.stores[sessionID]; ok {
		m.mu.RUnlock()
		return store, nil
	}
	m.mu.RUnlock()

	// Загружаем с диска
	filePath := filepath.Join(m.dataDir, sessionID+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Создаём новое хранилище
		return m.CreateStore(sessionID)
	}

	store := &SessionEventStore{}
	if err := json.Unmarshal(data, store); err != nil {
		return m.CreateStore(sessionID)
	}

	store.filePath = filePath
	store.LastAccess = time.Now()

	m.mu.Lock()
	m.stores[sessionID] = store
	m.mu.Unlock()

	return store, nil
}

// CreateStore создаёт новое хранилище
func (m *SessionStoreManager) CreateStore(sessionID string) (*SessionEventStore, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	store := &SessionEventStore{
		SessionID:   sessionID,
		Events:      make([]Event, 0),
		TotalTokens: 0,
		SummaryFrom: 0,
		Created:     time.Now(),
		LastAccess:  time.Now(),
		filePath:    filepath.Join(m.dataDir, sessionID+".json"),
	}

	m.stores[sessionID] = store

	// Сохраняем на диск
	if err := store.save(); err != nil {
		return nil, err
	}

	return store, nil
}

// AddEvent добавляет событие
func (s *SessionEventStore) AddEvent(eventType EventType, content string, metadata map[string]interface{}, tokens int) Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := Event{
		ID:        generateEventID(),
		Type:      eventType,
		Content:   content,
		Metadata:  metadata,
		Tokens:    tokens,
		Timestamp: time.Now(),
	}

	s.Events = append(s.Events, event)
	s.TotalTokens += tokens
	s.LastAccess = time.Now()

	// Сохраняем каждые 10 событий
	if len(s.Events)%10 == 0 {
		s.save()
	}

	return event
}

// GetEvents возвращает все события
func (s *SessionEventStore) GetEvents() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]Event, len(s.Events))
	copy(events, s.Events)
	return events
}

// GetRecentEvents возвращает последние N событий
func (s *SessionEventStore) GetRecentEvents(n int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Events) <= n {
		events := make([]Event, len(s.Events))
		copy(events, s.Events)
		return events
	}

	events := make([]Event, n)
	copy(events, s.Events[len(s.Events)-n:])
	return events
}

// GetEventsWithSummary возвращает события с суммаризацией
func (s *SessionEventStore) GetEventsWithSummary() (string, []Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Summary == "" {
		return "", s.Events
	}

	// Возвращаем суммаризацию + последние события
	recentEvents := s.Events[s.SummaryFrom:]
	return s.Summary, recentEvents
}

// Clear очищает хранилище
func (s *SessionEventStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Events = make([]Event, 0)
	s.TotalTokens = 0
	s.Summary = ""
	s.SummaryFrom = 0
	s.LastAccess = time.Now()
	s.save()
}

// save сохраняет хранилище на диск
func (s *SessionEventStore) save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0600)
}

// Compact выполняет компaction (суммаризацию) старых событий
func (s *SessionEventStore) Compact(summary string, keepLastN int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Events) <= keepLastN {
		return nil // Нечего компактить
	}

	// Сохраняем суммаризацию
	s.Summary = summary
	s.SummaryFrom = len(s.Events) - keepLastN

	// Удаляем старые события (оставляем только последние N)
	s.Events = s.Events[s.SummaryFrom:]

	return s.save()
}

// GetTokenCount возвращает количество токенов
func (s *SessionEventStore) GetTokenCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TotalTokens
}

// GetEventCount возвращает количество событий
func (s *SessionEventStore) GetEventCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Events)
}

// loadStores загружает все хранилища с диска
func (m *SessionStoreManager) loadStores() error {
	entries, err := os.ReadDir(m.dataDir)
	if err != nil {
		return nil // Директория может не существовать
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

		store := &SessionEventStore{}
		if err := json.Unmarshal(data, store); err != nil {
			continue
		}

		store.filePath = filePath
		m.stores[store.SessionID] = store
	}

	return nil
}

// RemoveStore удаляет хранилище
func (m *SessionStoreManager) RemoveStore(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if store, ok := m.stores[sessionID]; ok {
		os.Remove(store.filePath)
		delete(m.stores, sessionID)
	}

	return nil
}

// ListStores возвращает список всех хранилищ
func (m *SessionStoreManager) ListStores() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.stores))
	for id := range m.stores {
		ids = append(ids, id)
	}

	return ids
}

// Cleanup удаляет старые хранилища
func (m *SessionStoreManager) Cleanup(maxAge time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, store := range m.stores {
		if now.Sub(store.LastAccess) > maxAge {
			os.Remove(store.filePath)
			delete(m.stores, id)
		}
	}

	return nil
}

// generateEventID генерирует уникальный ID события
func generateEventID() string {
	return time.Now().Format("20060102150405.000")
}
