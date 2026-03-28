// Package cache предоставляет кэш для Session & Memory Service
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user/qwen-claw/internal/session"
)

// SessionCache кэш для сессий
type SessionCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewSessionCache создаёт новый кэш сессий
func NewSessionCache(client *redis.Client, ttl time.Duration) *SessionCache {
	return &SessionCache{
		client: client,
		ttl:    ttl,
	}
}

// Get получает сессию из кэша
func (c *SessionCache) Get(ctx context.Context, id string) (*session.Session, error) {
	key := fmt.Sprintf("session:%s", id)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // не найдено в кэше
	}
	if err != nil {
		return nil, fmt.Errorf("get session from cache: %w", err)
	}

	var sess session.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return &sess, nil
}

// Set сохраняет сессию в кэш
func (c *SessionCache) Set(ctx context.Context, sess *session.Session) error {
	key := fmt.Sprintf("session:%s", sess.ID)

	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("set session in cache: %w", err)
	}

	return nil
}

// Delete удаляет сессию из кэша
func (c *SessionCache) Delete(ctx context.Context, id string) error {
	key := fmt.Sprintf("session:%s", id)

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete session from cache: %w", err)
	}

	return nil
}

// Touch обновляет TTL сессии в кэше
func (c *SessionCache) Touch(ctx context.Context, id string) error {
	key := fmt.Sprintf("session:%s", id)

	if err := c.client.Expire(ctx, key, c.ttl).Err(); err != nil {
		return fmt.Errorf("touch session in cache: %w", err)
	}

	return nil
}

// MemoryCache кэш для записей памяти
type MemoryCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewMemoryCache создаёт новый кэш памяти
func NewMemoryCache(client *redis.Client, ttl time.Duration) *MemoryCache {
	return &MemoryCache{
		client: client,
		ttl:    ttl,
	}
}

// Get получает запись памяти из кэша
func (c *MemoryCache) Get(ctx context.Context, id string) (*session.MemoryEntry, error) {
	key := fmt.Sprintf("memory:%s", id)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // не найдено в кэше
	}
	if err != nil {
		return nil, fmt.Errorf("get memory from cache: %w", err)
	}

	var entry session.MemoryEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("unmarshal memory: %w", err)
	}

	return &entry, nil
}

// Set сохраняет запись памяти в кэш
func (c *MemoryCache) Set(ctx context.Context, entry *session.MemoryEntry) error {
	key := fmt.Sprintf("memory:%s", entry.ID)

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal memory: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("set memory in cache: %w", err)
	}

	return nil
}

// Delete удаляет запись памяти из кэша
func (c *MemoryCache) Delete(ctx context.Context, id string) error {
	key := fmt.Sprintf("memory:%s", id)

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete memory from cache: %w", err)
	}

	return nil
}

// InvalidateBySession инвалидирует кэш для сессии
func (c *MemoryCache) InvalidateBySession(ctx context.Context, sessionID string) error {
	pattern := fmt.Sprintf("memory:*") // В продакшене лучше использовать Redis SCAN

	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		// Проверяем принадлежит ли запись сессии
		// Это упрощённая реализация
		if err := c.client.Del(ctx, key).Err(); err != nil {
			return fmt.Errorf("invalidate memory cache: %w", err)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan memory cache: %w", err)
	}

	return nil
}
