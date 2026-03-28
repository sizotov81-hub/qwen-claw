package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigCache_SetGet(t *testing.T) {
	cache := NewConfigCache(5*time.Minute, 10)

	config := &Config{
		BaseDir: "/test",
	}

	// Кэша нет
	_, ok := cache.Get("/test/config.yaml")
	assert.False(t, ok)

	// Сохраняем
	cache.Set("/test/config.yaml", config)

	// Получаем
	cached, ok := cache.Get("/test/config.yaml")
	assert.True(t, ok)
	assert.Equal(t, "/test", cached.BaseDir)
}

func TestConfigCache_MaxAge(t *testing.T) {
	cache := NewConfigCache(50*time.Millisecond, 10)

	config := &Config{BaseDir: "/test"}
	cache.Set("/test/config.yaml", config)

	// Сразу доступно
	_, ok := cache.Get("/test/config.yaml")
	assert.True(t, ok)

	// Ждём истечения
	time.Sleep(100 * time.Millisecond)

	// Истекло
	_, ok = cache.Get("/test/config.yaml")
	assert.False(t, ok)
}

func TestConfigCache_MaxSize(t *testing.T) {
	cache := NewConfigCache(5*time.Minute, 3)

	// Добавляем 3 записи
	cache.Set("/config1.yaml", &Config{BaseDir: "1"})
	cache.Set("/config2.yaml", &Config{BaseDir: "2"})
	cache.Set("/config3.yaml", &Config{BaseDir: "3"})

	// Добавляем 4-ю (должна вытеснить первую)
	cache.Set("/config4.yaml", &Config{BaseDir: "4"})

	// Первая должна быть вытеснена
	_, ok := cache.Get("/config1.yaml")
	assert.False(t, ok)

	// Остальные доступны
	cached, ok := cache.Get("/config2.yaml")
	assert.True(t, ok)
	assert.Equal(t, "2", cached.BaseDir)

	cached, ok = cache.Get("/config3.yaml")
	assert.True(t, ok)
	assert.Equal(t, "3", cached.BaseDir)

	cached, ok = cache.Get("/config4.yaml")
	assert.True(t, ok)
	assert.Equal(t, "4", cached.BaseDir)
}

func TestConfigCache_LRU(t *testing.T) {
	cache := NewConfigCache(5*time.Minute, 3)

	// Добавляем 3 записи
	cache.Set("/config1.yaml", &Config{BaseDir: "1"})
	cache.Set("/config2.yaml", &Config{BaseDir: "2"})
	cache.Set("/config3.yaml", &Config{BaseDir: "3"})

	// Обращаемся к первой (делает её свежей)
	cache.Get("/config1.yaml")

	// Добавляем 4-ю (должна вытеснить вторую, т.к. первая свежая)
	cache.Set("/config4.yaml", &Config{BaseDir: "4"})

	// Первая ещё доступна (была обновлена)
	_, ok := cache.Get("/config1.yaml")
	assert.True(t, ok)

	// Вторая вытеснена
	_, ok = cache.Get("/config2.yaml")
	assert.False(t, ok)
}

func TestConfigCache_Delete(t *testing.T) {
	cache := NewConfigCache(5*time.Minute, 10)

	config := &Config{BaseDir: "/test"}
	cache.Set("/test/config.yaml", config)

	// Проверяем что есть
	_, ok := cache.Get("/test/config.yaml")
	assert.True(t, ok)

	// Удаляем
	cache.Delete("/test/config.yaml")

	// Проверяем что нет
	_, ok = cache.Get("/test/config.yaml")
	assert.False(t, ok)
}

func TestConfigCache_Clear(t *testing.T) {
	cache := NewConfigCache(5*time.Minute, 10)

	// Добавляем записи
	cache.Set("/config1.yaml", &Config{BaseDir: "1"})
	cache.Set("/config2.yaml", &Config{BaseDir: "2"})

	// Очищаем
	cache.Clear()

	// Проверяем что пусто
	stats := cache.GetStats()
	assert.Equal(t, 0, stats.Size)
}

func TestConfigCache_GetStats(t *testing.T) {
	cache := NewConfigCache(5*time.Minute, 10)

	// Добавляем записи
	cache.Set("/config1.yaml", &Config{BaseDir: "1"})
	cache.Set("/config2.yaml", &Config{BaseDir: "2"})

	stats := cache.GetStats()
	assert.Equal(t, 2, stats.Size)
	assert.Equal(t, 10, stats.MaxSize)
	assert.Equal(t, 5*time.Minute, stats.MaxAge)
	assert.Equal(t, 2, stats.Accesses)
}

func TestGlobalCache(t *testing.T) {
	// Очищаем глобальный кэш перед тестом
	ClearConfigCache()

	config := &Config{BaseDir: "/global"}

	// Сохраняем
	SetConfig("/global.yaml", config)

	// Получаем
	cached, ok := GetConfig("/global.yaml")
	assert.True(t, ok)
	assert.Equal(t, "/global", cached.BaseDir)

	// Статистика
	stats := GetCacheStats()
	assert.Equal(t, 1, stats.Size)
}

func TestConfig_Load_WithCache(t *testing.T) {
	// Очищаем кэш
	ClearConfigCache()

	// Создаём временный файл с минимальной конфигурацией
	tempFile := filepath.Join(t.TempDir(), "test_config.yaml")
	err := os.WriteFile(tempFile, []byte("base_dir: /test\n"), 0644)
	assert.NoError(t, err)

	// Первый загруз — без кэша
	cfg1, err := Load(tempFile)
	assert.NoError(t, err)
	assert.NotNil(t, cfg1)

	// Второй загруз — из кэша
	cfg2, err := Load(tempFile)
	assert.NoError(t, err)
	assert.NotNil(t, cfg2)

	// Это тот же объект (из кэша)
	assert.Equal(t, cfg1, cfg2)

	// Проверяем статистику
	stats := GetCacheStats()
	assert.GreaterOrEqual(t, stats.Size, 1)
}
