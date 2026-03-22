package config

import (
	"sync"
	"time"
)

// CacheEntry запись кэша конфигурации
type CacheEntry struct {
	config    *Config
	timestamp time.Time
}

// ConfigCache LRU кэш для конфигурации
type ConfigCache struct {
	mu       sync.RWMutex
	cache    map[string]*CacheEntry
	maxAge   time.Duration
	maxSize  int
 accesses []string // Для LRU
}

// DefaultConfigAge время жизни кэша по умолчанию
const DefaultConfigAge = 5 * time.Minute

// DefaultCacheSize максимальный размер кэша
const DefaultCacheSize = 10

// globalCache глобальный кэш конфигураций
var globalCache = NewConfigCache(DefaultConfigAge, DefaultCacheSize)

// NewConfigCache создаёт новый кэш конфигураций
func NewConfigCache(maxAge time.Duration, maxSize int) *ConfigCache {
	return &ConfigCache{
		cache:    make(map[string]*CacheEntry),
		maxAge:   maxAge,
		maxSize:  maxSize,
		accesses: make([]string, 0),
	}
}

// Get получает конфигурацию из кэша
func (c *ConfigCache) Get(path string) (*Config, bool) {
	c.mu.RLock()
	entry, ok := c.cache[path]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Проверяем не истёк ли кэш
	if time.Since(entry.timestamp) > c.maxAge {
		c.Delete(path)
		return nil, false
	}

	// Обновляем порядок доступа (LRU)
	c.mu.Lock()
	c.updateAccess(path)
	c.mu.Unlock()

	return entry.config, true
}

// Set сохраняет конфигурацию в кэш
func (c *ConfigCache) Set(path string, config *Config) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Если кэш уже есть, обновляем
	if _, ok := c.cache[path]; ok {
		c.cache[path] = &CacheEntry{
			config:    config,
			timestamp: time.Now(),
		}
		c.updateAccess(path)
		return
	}

	// Проверяем размер кэша
	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}

	c.cache[path] = &CacheEntry{
		config:    config,
		timestamp: time.Now(),
	}
	c.accesses = append(c.accesses, path)
}

// Delete удаляет конфигурацию из кэша
func (c *ConfigCache) Delete(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, path)

	// Удаляем из списка доступов
	for i, p := range c.accesses {
		if p == path {
			c.accesses = append(c.accesses[:i], c.accesses[i+1:]...)
			break
		}
	}
}

// Clear очищает весь кэш
func (c *ConfigCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CacheEntry)
	c.accesses = make([]string, 0)
}

// updateAccess обновляет порядок доступа (LRU)
func (c *ConfigCache) updateAccess(path string) {
	// Удаляем из текущего положения
	for i, p := range c.accesses {
		if p == path {
			c.accesses = append(c.accesses[:i], c.accesses[i+1:]...)
			break
		}
	}

	// Добавляем в конец (самый свежий)
	c.accesses = append(c.accesses, path)
}

// evictOldest удаляет самую старую запись
func (c *ConfigCache) evictOldest() {
	if len(c.accesses) == 0 {
		return
	}

	oldest := c.accesses[0]
	delete(c.cache, oldest)
	c.accesses = c.accesses[1:]
}

// GetStats возвращает статистику кэша
func (c *ConfigCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Size:      len(c.cache),
		MaxSize:   c.maxSize,
		MaxAge:    c.maxAge,
		Accesses:  len(c.accesses),
	}
}

// CacheStats статистика кэша
type CacheStats struct {
	Size      int           `json:"size"`
	MaxSize   int           `json:"max_size"`
	MaxAge    time.Duration `json:"max_age"`
	Accesses  int           `json:"accesses"`
}

// GetConfig получает конфигурацию из глобального кэша
func GetConfig(path string) (*Config, bool) {
	return globalCache.Get(path)
}

// SetConfig сохраняет конфигурацию в глобальный кэш
func SetConfig(path string, config *Config) {
	globalCache.Set(path, config)
}

// ClearConfigCache очищает глобальный кэш
func ClearConfigCache() {
	globalCache.Clear()
}

// GetCacheStats возвращает статистику глобального кэша
func GetCacheStats() CacheStats {
	return globalCache.GetStats()
}
