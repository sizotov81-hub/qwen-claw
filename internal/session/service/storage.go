// Package storage предоставляет хранилище данных для Session & Memory Service
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sm "github.com/user/qwen-claw/internal/session"
)

// SessionStorage хранилище сессий
type SessionStorage struct {
	db *sql.DB
}

// NewSessionStorage создаёт новое хранилище сессий
func NewSessionStorage(db *sql.DB) *SessionStorage {
	return &SessionStorage{db: db}
}

// Create создаёт новую сессию
func (s *SessionStorage) Create(ctx context.Context, sess *sm.Session) error {
	metadata, _ := json.Marshal(sess.Metadata)

	query := `
		INSERT INTO sessions (id, user_id, title, created_at, updated_at, last_accessed_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, query,
		sess.ID,
		sess.UserID,
		sess.Title,
		sess.CreatedAt,
		sess.UpdatedAt,
		sess.LastAccessedAt,
		metadata,
	)

	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

// GetByID получает сессию по ID
func (s *SessionStorage) GetByID(ctx context.Context, id string) (*sm.Session, error) {
	query := `
		SELECT id, user_id, title, created_at, updated_at, last_accessed_at, metadata
		FROM sessions
		WHERE id = $1
	`

	var sess sm.Session
	var metadata []byte

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&sess.ID,
		&sess.UserID,
		&sess.Title,
		&sess.CreatedAt,
		&sess.UpdatedAt,
		&sess.LastAccessedAt,
		&metadata,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if err := json.Unmarshal(metadata, &sess.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	// Получаем количество сообщений и токенов
	countQuery := `
		SELECT COUNT(*), COALESCE(SUM(tokens), 0)
		FROM session_messages
		WHERE session_id = $1
	`
	err = s.db.QueryRowContext(ctx, countQuery, id).Scan(&sess.MessageCount, &sess.TotalTokens)
	if err != nil {
		return nil, fmt.Errorf("get message count: %w", err)
	}

	return &sess, nil
}

// Update обновляет сессию
func (s *SessionStorage) Update(ctx context.Context, sess *sm.Session) error {
	metadata, _ := json.Marshal(sess.Metadata)

	query := `
		UPDATE sessions
		SET title = $2, updated_at = $3, last_accessed_at = $4, metadata = $5
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		sess.ID,
		sess.Title,
		sess.UpdatedAt,
		sess.LastAccessedAt,
		metadata,
	)

	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	return nil
}

// Delete удаляет сессию
func (s *SessionStorage) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = $1`

	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

// ListByUser получает список сессий пользователя
func (s *SessionStorage) ListByUser(ctx context.Context, userID string, limit, offset int32) ([]*sm.Session, error) {
	query := `
		SELECT id, user_id, title, created_at, updated_at, last_accessed_at, metadata
		FROM sessions
		WHERE user_id = $1
		ORDER BY last_accessed_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*sm.Session

	for rows.Next() {
		var sess sm.Session
		var metadata []byte

		err := rows.Scan(
			&sess.ID,
			&sess.UserID,
			&sess.Title,
			&sess.CreatedAt,
			&sess.UpdatedAt,
			&sess.LastAccessedAt,
			&metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		if err := json.Unmarshal(metadata, &sess.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}

		sessions = append(sessions, &sess)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return sessions, nil
}

// AddMessage добавляет сообщение в сессию
func (s *SessionStorage) AddMessage(ctx context.Context, msg *sm.Message) error {
	metadata, _ := json.Marshal(msg.Metadata)

	query := `
		INSERT INTO session_messages (id, session_id, role, content, tokens, created_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, query,
		msg.ID,
		msg.SessionID,
		msg.Role,
		msg.Content,
		msg.Tokens,
		msg.CreatedAt,
		metadata,
	)

	if err != nil {
		return fmt.Errorf("add message: %w", err)
	}

	return nil
}

// GetMessages получает сообщения сессии
func (s *SessionStorage) GetMessages(ctx context.Context, sessionID string, limit, offset int32, order string) ([]*sm.Message, error) {
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	query := fmt.Sprintf(`
		SELECT id, session_id, role, content, tokens, created_at, metadata
		FROM session_messages
		WHERE session_id = $1
		ORDER BY created_at %s
		LIMIT $2 OFFSET $3
	`, order)

	rows, err := s.db.QueryContext(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var messages []*sm.Message

	for rows.Next() {
		var msg sm.Message
		var metadata []byte

		err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.Role,
			&msg.Content,
			&msg.Tokens,
			&msg.CreatedAt,
			&metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		if err := json.Unmarshal(metadata, &msg.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}

		messages = append(messages, &msg)
	}

	return messages, nil
}

// MemoryStorage хранилище памяти
type MemoryStorage struct {
	db *sql.DB
}

// NewMemoryStorage создаёт новое хранилище памяти
func NewMemoryStorage(db *sql.DB) *MemoryStorage {
	return &MemoryStorage{db: db}
}

// Add добавляет запись памяти
func (m *MemoryStorage) Add(ctx context.Context, entry *sm.MemoryEntry) error {
	metadata, _ := json.Marshal(entry.Metadata)

	query := `
		INSERT INTO memory_entries (
			id, session_id, user_id, type, category, content,
			importance, retention, created_at, updated_at, last_accessed_at, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := m.db.ExecContext(ctx, query,
		entry.ID,
		entry.SessionID,
		entry.UserID,
		string(entry.Type),
		entry.Category,
		entry.Content,
		entry.Importance,
		entry.Retention,
		entry.CreatedAt,
		entry.UpdatedAt,
		entry.LastAccessedAt,
		metadata,
	)

	if err != nil {
		return fmt.Errorf("add memory: %w", err)
	}

	return nil
}

// GetByID получает запись памяти по ID
func (m *MemoryStorage) GetByID(ctx context.Context, id string) (*sm.MemoryEntry, error) {
	query := `
		SELECT id, session_id, user_id, type, category, content,
		       importance, retention, created_at, updated_at, last_accessed_at,
		       access_count, metadata
		FROM memory_entries
		WHERE id = $1
	`

	var entry sm.MemoryEntry
	var metadata []byte
	var memType string

	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID,
		&entry.SessionID,
		&entry.UserID,
		&memType,
		&entry.Category,
		&entry.Content,
		&entry.Importance,
		&entry.Retention,
		&entry.CreatedAt,
		&entry.UpdatedAt,
		&entry.LastAccessedAt,
		&entry.AccessCount,
		&metadata,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("memory entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}

	entry.Type = sm.MemoryType(memType)

	if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &entry, nil
}

// SearchByText ищет записи памяти по тексту
func (m *MemoryStorage) SearchByText(ctx context.Context, userID, query string, limit int32) ([]*sm.MemoryEntry, error) {
	searchQuery := "%" + query + "%"

	sqlQuery := `
		SELECT id, session_id, user_id, type, category, content,
		       importance, retention, created_at, updated_at, last_accessed_at,
		       access_count, metadata
		FROM memory_entries
		WHERE user_id = $1 AND content ILIKE $2
		ORDER BY last_accessed_at DESC
		LIMIT $3
	`

	rows, err := m.db.QueryContext(ctx, sqlQuery, userID, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search memory: %w", err)
	}
	defer rows.Close()

	var entries []*sm.MemoryEntry

	for rows.Next() {
		var entry sm.MemoryEntry
		var metadata []byte
		var memType string

		err := rows.Scan(
			&entry.ID,
			&entry.SessionID,
			&entry.UserID,
			&memType,
			&entry.Category,
			&entry.Content,
			&entry.Importance,
			&entry.Retention,
			&entry.CreatedAt,
			&entry.UpdatedAt,
			&entry.LastAccessedAt,
			&entry.AccessCount,
			&metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}

		entry.Type = sm.MemoryType(memType)

		if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}

// GetBySession получает записи памяти по сессии
func (m *MemoryStorage) GetBySession(ctx context.Context, sessionID string) ([]*sm.MemoryEntry, error) {
	query := `
		SELECT id, session_id, user_id, type, category, content,
		       importance, retention, created_at, updated_at, last_accessed_at,
		       access_count, metadata
		FROM memory_entries
		WHERE session_id = $1
		ORDER BY created_at DESC
	`

	rows, err := m.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session memory: %w", err)
	}
	defer rows.Close()

	var entries []*sm.MemoryEntry

	for rows.Next() {
		var entry sm.MemoryEntry
		var metadata []byte
		var memType string

		err := rows.Scan(
			&entry.ID,
			&entry.SessionID,
			&entry.UserID,
			&memType,
			&entry.Category,
			&entry.Content,
			&entry.Importance,
			&entry.Retention,
			&entry.CreatedAt,
			&entry.UpdatedAt,
			&entry.LastAccessedAt,
			&entry.AccessCount,
			&metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}

		entry.Type = sm.MemoryType(memType)

		if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}

// UpdateAccess обновляет время доступа и счётчик
func (m *MemoryStorage) UpdateAccess(ctx context.Context, id string) error {
	query := `
		UPDATE memory_entries
		SET last_accessed_at = $2
		WHERE id = $1
	`

	_, err := m.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("update access: %w", err)
	}

	return nil
}

// Delete удаляет запись памяти
func (m *MemoryStorage) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM memory_entries WHERE id = $1`

	_, err := m.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}

	return nil
}
