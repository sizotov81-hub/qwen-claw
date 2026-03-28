// Package service предоставляет бизнес-логику для Session & Memory Service
package service

import (
	"context"
	"fmt"

	"github.com/user/qwen-claw/internal/cache"
	sm "github.com/user/qwen-claw/internal/session"
)

// SessionService сервис для управления сессиями
type SessionService struct {
	storage *SessionStorage
	cache   *cache.SessionCache
}

// NewSessionService создаёт новый сервис сессий
func NewSessionService(storage *SessionStorage, cache *cache.SessionCache) *SessionService {
	return &SessionService{
		storage: storage,
		cache:   cache,
	}
}

// CreateSession создаёт новую сессию
func (s *SessionService) CreateSession(ctx context.Context, userID, title string) (*sm.Session, error) {
	sess := sm.NewSession(userID, title)

	// Сохраняем в хранилище
	if err := s.storage.Create(ctx, sess); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Сохраняем в кэш
	if s.cache != nil {
		if err := s.cache.Set(ctx, sess); err != nil {
			// Логгируем ошибку кэша, но не прерываем операцию
		}
	}

	return sess, nil
}

// GetSession получает сессию по ID
func (s *SessionService) GetSession(ctx context.Context, id string) (*sm.Session, error) {
	// Пробуем получить из кэша
	if s.cache != nil {
		sess, err := s.cache.Get(ctx, id)
		if err == nil && sess != nil {
			return sess, nil
		}
	}

	// Получаем из хранилища
	sess, err := s.storage.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Сохраняем в кэш
	if s.cache != nil {
		if err := s.cache.Set(ctx, sess); err != nil {
			// Логгируем ошибку кэша
		}
	}

	return sess, nil
}

// UpdateSession обновляет сессию
func (s *SessionService) UpdateSession(ctx context.Context, id, title string, metadata map[string]string) (*sm.Session, error) {
	sess, err := s.storage.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	sess.Title = title
	if metadata != nil {
		sess.Metadata = metadata
	}
	sess.Touch()

	if err := s.storage.Update(ctx, sess); err != nil {
		return nil, err
	}

	// Обновляем кэш
	if s.cache != nil {
		if err := s.cache.Set(ctx, sess); err != nil {
			// Логгируем ошибку кэша
		}
	}

	return sess, nil
}

// DeleteSession удаляет сессию
func (s *SessionService) DeleteSession(ctx context.Context, id string) error {
	// Удаляем из хранилища
	if err := s.storage.Delete(ctx, id); err != nil {
		return err
	}

	// Удаляем из кэша
	if s.cache != nil {
		if err := s.cache.Delete(ctx, id); err != nil {
			// Логгируем ошибку кэша
		}
	}

	return nil
}

// ListSessions получает список сессий пользователя
func (s *SessionService) ListSessions(ctx context.Context, userID string, limit, offset int32) ([]*sm.Session, error) {
	return s.storage.ListByUser(ctx, userID, limit, offset)
}

// AddMessage добавляет сообщение в сессию
func (s *SessionService) AddMessage(ctx context.Context, sessionID, role, content string, tokens int32) (*sm.Message, error) {
	// Проверяем существование сессии
	_, err := s.storage.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	msg := sm.NewMessage(sessionID, role, content, tokens)

	if err := s.storage.AddMessage(ctx, msg); err != nil {
		return nil, err
	}

	// Инвалидируем кэш сессии
	if s.cache != nil {
		if err := s.cache.Delete(ctx, sessionID); err != nil {
			// Логгируем ошибку кэша
		}
	}

	return msg, nil
}

// GetMessages получает сообщения сессии
func (s *SessionService) GetMessages(ctx context.Context, sessionID string, limit, offset int32, order string) ([]*sm.Message, error) {
	return s.storage.GetMessages(ctx, sessionID, limit, offset, order)
}

// MemoryService сервис для управления памятью
type MemoryService struct {
	storage *MemoryStorage
	cache   *cache.MemoryCache
}

// NewMemoryService создаёт новый сервис памяти
func NewMemoryService(storage *MemoryStorage, cache *cache.MemoryCache) *MemoryService {
	return &MemoryService{
		storage: storage,
		cache:   cache,
	}
}

// AddMemory добавляет запись в память
func (m *MemoryService) AddMemory(ctx context.Context, entry *sm.MemoryEntry) (*sm.MemoryEntry, error) {
	if err := sm.ValidateMemoryEntry(entry); err != nil {
		return nil, fmt.Errorf("validate memory entry: %w", err)
	}

	if err := m.storage.Add(ctx, entry); err != nil {
		return nil, err
	}

	// Сохраняем в кэш
	if m.cache != nil {
		if err := m.cache.Set(ctx, entry); err != nil {
			// Логгируем ошибку кэша
		}
	}

	return entry, nil
}

// GetMemory получает запись памяти по ID
func (m *MemoryService) GetMemory(ctx context.Context, id string) (*sm.MemoryEntry, error) {
	// Пробуем получить из кэша
	if m.cache != nil {
		entry, err := m.cache.Get(ctx, id)
		if err == nil && entry != nil {
			entry.Access()
			return entry, nil
		}
	}

	// Получаем из хранилища
	entry, err := m.storage.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Обновляем счётчик доступа
	entry.Access()

	// Сохраняем в кэш
	if m.cache != nil {
		if err := m.cache.Set(ctx, entry); err != nil {
			// Логгируем ошибку кэша
		}
	}

	return entry, nil
}

// SearchMemory ищет записи в памяти по тексту
func (m *MemoryService) SearchMemory(ctx context.Context, userID, query string, limit int32) ([]*sm.MemoryEntry, error) {
	return m.storage.SearchByText(ctx, userID, query, limit)
}

// GetSessionMemory получает записи памяти сессии
func (m *MemoryService) GetSessionMemory(ctx context.Context, sessionID string) ([]*sm.MemoryEntry, error) {
	return m.storage.GetBySession(ctx, sessionID)
}

// DeleteMemory удаляет запись из памяти
func (m *MemoryService) DeleteMemory(ctx context.Context, id string) error {
	if err := m.storage.Delete(ctx, id); err != nil {
		return err
	}

	// Удаляем из кэша
	if m.cache != nil {
		if err := m.cache.Delete(ctx, id); err != nil {
			// Логгируем ошибку кэша
		}
	}

	return nil
}
