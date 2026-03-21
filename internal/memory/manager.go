package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MemoryEntry запись памяти
type MemoryEntry struct {
	// ID уникальный идентификатор
	ID string `json:"id"`
	
	// Type тип записи (fact, task, context, conversation)
	Type string `json:"type"`
	
	// Content содержимое
	Content string `json:"content"`
	
	// Metadata дополнительные метаданные
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	
	// Created время создания
	Created time.Time `json:"created"`
	
	// Updated время обновления
	Updated time.Time `json:"updated"`
	
	// Expires время истечения (опционально)
	Expires *time.Time `json:"expires,omitempty"`
}

// Conversation разговор
type Conversation struct {
	// ID идентификатор разговора
	ID string `json:"id"`
	
	// Messages сообщения
	Messages []Message `json:"messages"`
	
	// Created время создания
	Created time.Time `json:"created"`
	
	// Updated время последнего обновления
	Updated time.Time `json:"updated"`
}

// Message сообщение в разговоре
type Message struct {
	// Role роль (user, assistant, system)
	Role string `json:"role"`
	
	// Content содержимое
	Content string `json:"content"`
	
	// Timestamp время отправки
	Timestamp time.Time `json:"timestamp"`
}

// Manager менеджер памяти
type Manager struct {
	// memoryDir директория для хранения памяти
	memoryDir string
	
	// entries загруженные записи
	entries map[string]*MemoryEntry
	
	// conversations активные разговоры
	conversations map[string]*Conversation
	
	// mu мьютекс для потокобезопасности
	mu sync.RWMutex
	
	// currentConversation текущий разговор
	currentConversationID string
}

// NewManager создаёт новый менеджер памяти
func NewManager(memoryDir string) *Manager {
	return &Manager{
		memoryDir:     memoryDir,
		entries:       make(map[string]*MemoryEntry),
		conversations: make(map[string]*Conversation),
	}
}

// Init инициализирует менеджер памяти
func (m *Manager) Init() error {
	// Создаём директорию
	if err := os.MkdirAll(m.memoryDir, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}
	
	// Загружаем существующие данные
	return m.load()
}

// load загружает данные из файлов
func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Загружаем записи памяти
	entriesFile := filepath.Join(m.memoryDir, "entries.json")
	if data, err := os.ReadFile(entriesFile); err == nil {
		var entries []*MemoryEntry
		if err := json.Unmarshal(data, &entries); err == nil {
			for _, entry := range entries {
				m.entries[entry.ID] = entry
			}
		}
	}
	
	// Загружаем разговоры
	conversationsDir := filepath.Join(m.memoryDir, "conversations")
	if entries, err := os.ReadDir(conversationsDir); err == nil {
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			
			data, err := os.ReadFile(filepath.Join(conversationsDir, entry.Name()))
			if err != nil {
				continue
			}
			
			var conv Conversation
			if err := json.Unmarshal(data, &conv); err != nil {
				continue
			}
			
			m.conversations[conv.ID] = &conv
		}
	}
	
	return nil
}

// Save сохраняет данные на диск
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.saveEntriesLocked()
}

// saveEntriesLocked сохраняет записи (должен вызываться с захваченным мьютексом)
func (m *Manager) saveEntriesLocked() error {
	// Сохраняем записи
	entries := make([]*MemoryEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		entries = append(entries, entry)
	}

	entriesData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}

	if err := os.WriteFile(filepath.Join(m.memoryDir, "entries.json"), entriesData, 0644); err != nil {
		return fmt.Errorf("failed to write entries: %w", err)
	}

	// Сохраняем разговоры
	conversationsDir := filepath.Join(m.memoryDir, "conversations")
	if err := os.MkdirAll(conversationsDir, 0755); err != nil {
		return err
	}

	for id, conv := range m.conversations {
		data, err := json.MarshalIndent(conv, "", "  ")
		if err != nil {
			continue
		}

		if err := os.WriteFile(filepath.Join(conversationsDir, id+".json"), data, 0644); err != nil {
			continue
		}
	}

	return nil
}

// Remember добавляет запись в память
func (m *Manager) Remember(entryType, content string, metadata map[string]interface{}) (*MemoryEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := generateID()
	now := time.Now()

	entry := &MemoryEntry{
		ID:        id,
		Type:      entryType,
		Content:   content,
		Metadata:  metadata,
		Created:   now,
		Updated:   now,
	}

	m.entries[id] = entry
	
	// Сохраняем на диск
	_ = m.saveEntriesLocked()
	
	return entry, nil
}

// Recall ищет записи в памяти
func (m *Manager) Recall(query string, limit int) []*MemoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var results []*MemoryEntry
	
	for _, entry := range m.entries {
		// Простой поиск по содержимому
		if containsIgnoreCase(entry.Content, query) {
			results = append(results, entry)
		}
		
		// Поиск по типу
		if containsIgnoreCase(entry.Type, query) {
			results = append(results, entry)
		}
		
		if len(results) >= limit {
			break
		}
	}
	
	return results
}

// GetEntry получает запись по ID
func (m *Manager) GetEntry(id string) *MemoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.entries[id]
}

// DeleteEntry удаляет запись
func (m *Manager) DeleteEntry(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.entries, id)
	return nil
}

// StartConversation начинает новый разговор
func (m *Manager) StartConversation() *Conversation {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	id := generateID()
	now := time.Now()
	
	conv := &Conversation{
		ID:        id,
		Messages:  make([]Message, 0),
		Created:   now,
		Updated:   now,
	}
	
	m.conversations[id] = conv
	m.currentConversationID = id
	
	return conv
}

// GetCurrentConversation получает текущий разговор
func (m *Manager) GetCurrentConversation() *Conversation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.currentConversationID == "" {
		return nil
	}
	
	return m.conversations[m.currentConversationID]
}

// AddMessage добавляет сообщение в разговор
func (m *Manager) AddMessage(role, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.currentConversationID == "" {
		// Начинаем новый разговор
		id := generateID()
		now := time.Now()
		
		m.conversations[id] = &Conversation{
			ID:        id,
			Messages:  make([]Message, 0),
			Created:   now,
			Updated:   now,
		}
		m.currentConversationID = id
	}
	
	conv := m.conversations[m.currentConversationID]
	conv.Messages = append(conv.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	conv.Updated = time.Now()
	
	return nil
}

// GetContext получает контекст для LLM
func (m *Manager) GetContext() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Получаем последние сообщения из текущего разговора
	if m.currentConversationID != "" {
		if conv, ok := m.conversations[m.currentConversationID]; ok {
			if len(conv.Messages) > 0 {
				// Берем последние 10 сообщений
				start := 0
				if len(conv.Messages) > 10 {
					start = len(conv.Messages) - 10
				}
				
				context := "Recent conversation:\n"
				for _, msg := range conv.Messages[start:] {
					context += fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content)
				}
				return context
			}
		}
	}
	
	// Получаем важные факты
	var facts []string
	for _, entry := range m.entries {
		if entry.Type == "fact" {
			facts = append(facts, entry.Content)
		}
	}
	
	if len(facts) > 0 {
		return "Known facts:\n" + joinStrings(facts, "\n")
	}
	
	return ""
}

// ListEntries возвращает все записи
func (m *Manager) ListEntries() []*MemoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make([]*MemoryEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		result = append(result, entry)
	}
	return result
}

// Clear очищает всю память
func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = make(map[string]*MemoryEntry)
	m.conversations = make(map[string]*Conversation)
	m.currentConversationID = ""

	return m.saveEntriesLocked()
}

// Вспомогательные функции

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func containsIgnoreCase(s, substr string) bool {
	return contains(s, substr) || contains(lower(s), lower(substr))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func lower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + 32
		}
		result[i] = c
	}
	return string(result)
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	
	totalLen := len(strs) - 1 + len(sep)*(len(strs)-1)
	for _, s := range strs {
		totalLen += len(s)
	}
	
	result := make([]byte, 0, totalLen)
	for i, s := range strs {
		if i > 0 {
			result = append(result, sep...)
		}
		result = append(result, s...)
	}
	
	return string(result)
}
