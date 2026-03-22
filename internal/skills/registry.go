package skills

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RegistrySkill информация о навыке в реестре
type RegistrySkill struct {
	// Name имя навыка
	Name string `json:"name"`

	// Description описание
	Description string `json:"description"`

	// Version текущая версия
	Version string `json:"version"`

	// Author автор
	Author string `json:"author"`

	// Category категория
	Category string `json:"category"`

	// Tags теги
	Tags []string `json:"tags"`

	// Repository URL репозитория
	Repository string `json:"repository"`

	// DownloadURL URL для скачивания
	DownloadURL string `json:"download_url"`

	// Homepage домашняя страница
	Homepage string `json:"homepage"`

	// License лицензия
	License string `json:"license"`

	// Downloads количество загрузок
	Downloads int `json:"downloads"`

	// Created дата создания
	Created time.Time `json:"created"`

	// Updated дата обновления
	Updated time.Time `json:"updated"`
}

// RegistryResponse ответ от реестра
type RegistryResponse struct {
	// Skills список навыков
	Skills []RegistrySkill `json:"skills"`

	// Total общее количество
	Total int `json:"total"`

	// Page текущая страница
	Page int `json:"page"`

	// PageSize размер страницы
	PageSize int `json:"page_size"`
}

// RegistryClient клиент для работы с реестром навыков
type RegistryClient struct {
	// BaseURL базовый URL реестра
	BaseURL string

	// Timeout таймаут запросов
	Timeout time.Duration

	// CacheDir директория кэша
	CacheDir string

	// CacheTTL время жизни кэша
	CacheTTL time.Duration
}

// RegistryConfig конфигурация клиента реестра
type RegistryConfig struct {
	// BaseURL базовый URL реестра
	BaseURL string

	// Timeout таймаут запросов (по умолчанию 30 секунд)
	Timeout time.Duration

	// CacheDir директория кэша (по умолчанию ~/.qwen/registry-cache)
	CacheDir string

	// CacheTTL время жизни кэша (по умолчанию 1 час)
	CacheTTL time.Duration
}

// DefaultRegistryConfig возвращает конфигурацию по умолчанию
func DefaultRegistryConfig() *RegistryConfig {
	homeDir, _ := os.UserHomeDir()
	return &RegistryConfig{
		BaseURL:  "https://registry.qwen-claw.dev", // Заглушка, можно заменить на реальный реестр
		Timeout:  30 * time.Second,
		CacheDir: filepath.Join(homeDir, ".qwen", "registry-cache"),
		CacheTTL: 1 * time.Hour,
	}
}

// NewRegistryClient создаёт новый клиент реестра
func NewRegistryClient(config *RegistryConfig) *RegistryClient {
	if config == nil {
		config = DefaultRegistryConfig()
	}

	return &RegistryClient{
		BaseURL:  config.BaseURL,
		Timeout:  config.Timeout,
		CacheDir: config.CacheDir,
		CacheTTL: config.CacheTTL,
	}
}

// Search ищет навыки в реестре
func (c *RegistryClient) Search(query string, category string, tags []string) ([]RegistrySkill, error) {
	// Пытаемся получить из кэша
	cacheKey := c.getCacheKey("search", query, category, strings.Join(tags, ","))
	if cached, err := c.getFromCache(cacheKey); err == nil {
		return cached, nil
	}

	// Формируем URL
	url := fmt.Sprintf("%s/api/v1/skills/search", c.BaseURL)
	params := []string{}
	if query != "" {
		params = append(params, "q="+query)
	}
	if category != "" {
		params = append(params, "category="+category)
	}
	for _, tag := range tags {
		params = append(params, "tag="+tag)
	}
	if len(params) > 0 {
		url += "?" + joinStrings(params, "&")
	}

	// Выполняем запрос
	client := &http.Client{Timeout: c.Timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to search registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var registryResp RegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registryResp); err != nil {
		return nil, err
	}

	// Сохраняем в кэш
	c.saveToCache(cacheKey, registryResp.Skills)

	return registryResp.Skills, nil
}

// GetSkill получает информацию о навыке
func (c *RegistryClient) GetSkill(name string) (*RegistrySkill, error) {
	// Пытаемся получить из кэша
	cacheKey := c.getCacheKey("skill", name)
	if cached, err := c.getFromCache(cacheKey); err == nil && len(cached) > 0 {
		return &cached[0], nil
	}

	// Выполняем запрос
	url := fmt.Sprintf("%s/api/v1/skills/%s", c.BaseURL, name)
	client := &http.Client{Timeout: c.Timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	var skill RegistrySkill
	if err := json.NewDecoder(resp.Body).Decode(&skill); err != nil {
		return nil, err
	}

	// Сохраняем в кэш
	c.saveToCache(cacheKey, []RegistrySkill{skill})

	return &skill, nil
}

// Download скачивает навык
func (c *RegistryClient) Download(name string, destDir string) error {
	// Получаем информацию о навыке
	skill, err := c.GetSkill(name)
	if err != nil {
		return err
	}

	// Создаём временную директорию
	tmpDir, err := os.MkdirTemp("", "qwen-claw-skill-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Скачиваем файл
	zipPath := filepath.Join(tmpDir, name+".zip")
	if err := c.downloadFile(skill.DownloadURL, zipPath); err != nil {
		return fmt.Errorf("failed to download skill: %w", err)
	}

	// Распаковываем
	if err := c.extractZip(zipPath, destDir); err != nil {
		return fmt.Errorf("failed to extract skill: %w", err)
	}

	return nil
}

// ListPopular возвращает популярные навыки
func (c *RegistryClient) ListPopular(limit int) ([]RegistrySkill, error) {
	// Пытаемся получить из кэша
	cacheKey := c.getCacheKey("popular", fmt.Sprintf("%d", limit))
	if cached, err := c.getFromCache(cacheKey); err == nil {
		if len(cached) > limit {
			return cached[:limit], nil
		}
		return cached, nil
	}

	// Выполняем запрос
	url := fmt.Sprintf("%s/api/v1/skills/popular?limit=%d", c.BaseURL, limit)
	client := &http.Client{Timeout: c.Timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular skills: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var registryResp RegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registryResp); err != nil {
		return nil, err
	}

	// Сохраняем в кэш
	c.saveToCache(cacheKey, registryResp.Skills)

	return registryResp.Skills, nil
}

// ListCategories возвращает список категорий
func (c *RegistryClient) ListCategories() ([]string, error) {
	// Пытаемся получить из кэша
	cacheKey := c.getCacheKey("categories")
	if cached, err := c.getFromCache(cacheKey); err == nil {
		categories := make([]string, len(cached))
		for i, c := range cached {
			categories[i] = c.Name
		}
		return categories, nil
	}

	// Выполняем запрос
	url := fmt.Sprintf("%s/api/v1/skills/categories", c.BaseURL)
	client := &http.Client{Timeout: c.Timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var categories []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
		return nil, err
	}

	categoryNames := make([]string, len(categories))
	for i, cat := range categories {
		categoryNames[i] = cat.Name
	}

	// Сохраняем в кэш
	cacheData, _ := json.Marshal(categories)
	c.saveRawToCache(cacheKey, cacheData)

	return categoryNames, nil
}

// getCacheKey генерирует ключ кэша
func (c *RegistryClient) getCacheKey(prefix string, parts ...string) string {
	key := prefix
	for _, part := range parts {
		key += "_" + part
	}
	return key + ".json"
}

// getFromCache получает данные из кэша
func (c *RegistryClient) getFromCache(cacheKey string) ([]RegistrySkill, error) {
	cachePath := filepath.Join(c.CacheDir, cacheKey)

	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, err
	}

	// Проверяем возраст кэша
	if time.Since(info.ModTime()) > c.CacheTTL {
		os.Remove(cachePath)
		return nil, fmt.Errorf("cache expired")
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var skills []RegistrySkill
	if err := json.Unmarshal(data, &skills); err != nil {
		return nil, err
	}

	return skills, nil
}

// saveToCache сохраняет данные в кэш
func (c *RegistryClient) saveToCache(cacheKey string, skills []RegistrySkill) {
	cachePath := filepath.Join(c.CacheDir, cacheKey)

	// Создаём директорию кэша
	if err := os.MkdirAll(c.CacheDir, 0700); err != nil {
		return
	}

	data, _ := json.Marshal(skills)
	os.WriteFile(cachePath, data, 0600)
}

// saveRawToCache сохраняет сырые данные в кэш
func (c *RegistryClient) saveRawToCache(cacheKey string, data []byte) {
	cachePath := filepath.Join(c.CacheDir, cacheKey)

	if err := os.MkdirAll(c.CacheDir, 0700); err != nil {
		return
	}

	os.WriteFile(cachePath, data, 0600)
}

// downloadFile скачивает файл по URL
func (c *RegistryClient) downloadFile(url string, destPath string) error {
	client := &http.Client{Timeout: c.Timeout}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractZip распаковывает ZIP архив
func (c *RegistryClient) extractZip(zipPath string, destDir string) error {
	// Используем unzip или встроенную распаковку
	cmd := exec.Command("unzip", "-o", zipPath, "-d", destDir)
	if err := cmd.Run(); err != nil {
		// Если unzip не доступен, пробуем встроенную распаковку
		return c.extractZipNative(zipPath, destDir)
	}
	return nil
}

// extractZipNative распаковывает ZIP без внешней команды
func (c *RegistryClient) extractZipNative(zipPath string, destDir string) error {
	// TODO: Реализовать нативную распаковку ZIP
	// Для простоты пока возвращаем ошибку
	return fmt.Errorf("unzip command not available, please install it")
}

// joinStrings объединяет строки с разделителем
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
