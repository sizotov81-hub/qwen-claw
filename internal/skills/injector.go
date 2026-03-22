package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SkillInjector инжектор навыков для селективной загрузки
type SkillInjector struct {
	// skillsDir директория с навыками
	skillsDir string

	// loadedSkills загруженные навыки
	loadedSkills map[string]*Skill

	// relevanceThreshold порог релевантности
	relevanceThreshold float64

	// maxSkillsPerQuery максимум навыков на запрос
	maxSkillsPerQuery int
}

// InjectorConfig конфигурация инжектора
type InjectorConfig struct {
	// SkillsDir директория с навыками
	SkillsDir string

	// RelevanceThreshold порог релевантности (0.0-1.0)
	RelevanceThreshold float64

	// MaxSkillsPerQuery максимум навыков на запрос
	MaxSkillsPerQuery int
}

// DefaultInjectorConfig возвращает конфигурацию по умолчанию
func DefaultInjectorConfig() *InjectorConfig {
	return &InjectorConfig{
		SkillsDir:          "~/.qwen/skills",
		RelevanceThreshold: 0.3,
		MaxSkillsPerQuery:  5,
	}
}

// NewSkillInjector создаёт новый инжектор навыков
func NewSkillInjector(config *InjectorConfig) *SkillInjector {
	if config == nil {
		config = DefaultInjectorConfig()
	}

	// Раскрываем ~
	skillsDir := config.SkillsDir
	if strings.HasPrefix(skillsDir, "~") {
		homeDir, _ := os.UserHomeDir()
		skillsDir = filepath.Join(homeDir, skillsDir[2:])
	}

	return &SkillInjector{
		skillsDir:          skillsDir,
		loadedSkills:       make(map[string]*Skill),
		relevanceThreshold: config.RelevanceThreshold,
		maxSkillsPerQuery:  config.MaxSkillsPerQuery,
	}
}

// InjectSkills возвращает релевантные навыки для запроса
func (i *SkillInjector) InjectSkills(query string) ([]*Skill, error) {
	// Загружаем все доступные навыки
	skills, err := i.loadAllSkills()
	if err != nil {
		return nil, err
	}

	// Вычисляем релевантность
	scoredSkills := make([]struct {
		skill *Skill
		score float64
	}, 0)

	for _, skill := range skills {
		score := i.calculateRelevance(skill, query)
		if score >= i.relevanceThreshold {
			scoredSkills = append(scoredSkills, struct {
				skill *Skill
				score float64
			}{skill, score})
		}
	}

	// Сортируем по релевантности
	sortByScore(scoredSkills)

	// Берём топ-N
	result := make([]*Skill, 0)
	for idx, ss := range scoredSkills {
		if idx >= i.maxSkillsPerQuery {
			break
		}
		result = append(result, ss.skill)
	}

	return result, nil
}

// GetSkillPrompt возвращает промпт для навыка
func (i *SkillInjector) GetSkillPrompt(skill *Skill) (string, error) {
	skillDir := filepath.Join(i.skillsDir, skill.Name)

	// Пытаемся загрузить SKILL.md
	skillPromptPath := filepath.Join(skillDir, "SKILL.md")
	if data, err := os.ReadFile(skillPromptPath); err == nil {
		return string(data), nil
	}

	// Пытаемся загрузить PROMPT.md
	promptPath := filepath.Join(skillDir, "PROMPT.md")
	if data, err := os.ReadFile(promptPath); err == nil {
		return string(data), nil
	}

	// Пытаемся загрузить из skill.json
	if skill.Description != "" {
		return fmt.Sprintf("# %s\n\n%s", skill.Name, skill.Description), nil
	}

	return "", fmt.Errorf("skill prompt not found for %s", skill.Name)
}

// FormatSkillsForPrompt форматирует навыки для вставки в промпт
func (i *SkillInjector) FormatSkillsForPrompt(skills []*Skill) (string, error) {
	if len(skills) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("## Available Skills\n\n")

	for _, skill := range skills {
		prompt, err := i.GetSkillPrompt(skill)
		if err != nil {
			// Используем описание как fallback
			prompt = fmt.Sprintf("# %s\n\n%s", skill.Name, skill.Description)
		}

		sb.WriteString(prompt)
		sb.WriteString("\n\n---\n\n")
	}

	return sb.String(), nil
}

// loadAllSkills загружает все доступные навыки
func (i *SkillInjector) loadAllSkills() ([]*Skill, error) {
	skills := make([]*Skill, 0)

	// Сканируем директорию с навыками
	entries, err := os.ReadDir(i.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return skills, nil // Директория не существует — возвращаем пустой список
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(i.skillsDir, entry.Name())
		skill, err := i.loadSkill(skillPath)
		if err != nil {
			continue // Пропускаем некорректные навыки
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// loadSkill загружает один навык
func (i *SkillInjector) loadSkill(skillDir string) (*Skill, error) {
	skillJSONPath := filepath.Join(skillDir, "skill.json")

	data, err := os.ReadFile(skillJSONPath)
	if err != nil {
		return nil, err
	}

	var skill Skill
	if err := json.Unmarshal(data, &skill); err != nil {
		return nil, err
	}

	// Проверяем, включён ли навык
	if !skill.Enabled {
		return nil, fmt.Errorf("skill is disabled")
	}

	return &skill, nil
}

// calculateRelevance вычисляет релевантность навыка для запроса
func (i *SkillInjector) calculateRelevance(skill *Skill, query string) float64 {
	query = strings.ToLower(query)
	score := 0.0

	// Проверяем совпадение с командами
	for _, cmd := range skill.Commands {
		if strings.Contains(query, strings.ToLower(cmd)) {
			score += 0.5
		}
	}

	// Проверяем совпадение с описанием
	descLower := strings.ToLower(skill.Description)
	if strings.Contains(query, descLower) {
		score += 0.3
	}

	// Проверяем совпадение с именем
	if strings.Contains(query, strings.ToLower(skill.Name)) {
		score += 0.4
	}

	// Проверяем совпадение ключевых слов из описания
	keywords := extractKeywords(descLower)
	for _, keyword := range keywords {
		if strings.Contains(query, keyword) {
			score += 0.1
		}
	}

	// Нормализуем score (максимум 1.0)
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// extractKeywords извлекает ключевые слова из текста
func extractKeywords(text string) []string {
	// Простая реализация: разбиваем по пробелам и убираем стоп-слова
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"что": true, "это": true, "и": true, "или": true, "но": true,
		"в": true, "на": true, "с": true, "к": true, "по": true,
		"для": true, "от": true, "из": true, "за": true, "под": true,
		"я": true, "ты": true, "он": true, "она": true, "оно": true, "мы": true,
	}

	words := strings.Fields(text)
	keywords := make([]string, 0)
	seen := make(map[string]bool)

	for _, word := range words {
		// Убираем знаки препинания
		word = strings.Trim(word, ".,!?;:\"'()-")

		// Пропускаем стоп-слова и короткие слова
		if len(word) < 3 || stopWords[word] {
			continue
		}

		// Пропускаем дубликаты
		if seen[word] {
			continue
		}

		seen[word] = true
		keywords = append(keywords, word)
	}

	return keywords
}

// sortByScore сортирует навыки по релевантности
func sortByScore(scoredSkills []struct {
	skill *Skill
	score float64
}) {
	// Простая сортировка пузырьком
	n := len(scoredSkills)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if scoredSkills[j].score < scoredSkills[j+1].score {
				scoredSkills[j], scoredSkills[j+1] = scoredSkills[j+1], scoredSkills[j]
			}
		}
	}
}

// GetSkillInfo возвращает информацию о навыке
func (i *SkillInjector) GetSkillInfo(name string) (*Skill, error) {
	skillDir := filepath.Join(i.skillsDir, name)
	return i.loadSkill(skillDir)
}

// ListSkills возвращает список всех доступных навыков
func (i *SkillInjector) ListSkills() ([]*Skill, error) {
	return i.loadAllSkills()
}

// EnableSkill включает навык
func (i *SkillInjector) EnableSkill(name string) error {
	skillDir := filepath.Join(i.skillsDir, name)
	skillJSONPath := filepath.Join(skillDir, "skill.json")

	// Загружаем skill.json
	data, err := os.ReadFile(skillJSONPath)
	if err != nil {
		return err
	}

	var skill Skill
	if err := json.Unmarshal(data, &skill); err != nil {
		return err
	}

	// Включаем навык
	skill.Enabled = true

	// Сохраняем обратно
	newData, err := json.MarshalIndent(skill, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(skillJSONPath, newData, 0600)
}

// DisableSkill отключает навык
func (i *SkillInjector) DisableSkill(name string) error {
	skillDir := filepath.Join(i.skillsDir, name)
	skillJSONPath := filepath.Join(skillDir, "skill.json")

	// Загружаем skill.json
	data, err := os.ReadFile(skillJSONPath)
	if err != nil {
		return err
	}

	var skill Skill
	if err := json.Unmarshal(data, &skill); err != nil {
		return err
	}

	// Отключаем навык
	skill.Enabled = false

	// Сохраняем обратно
	newData, err := json.MarshalIndent(skill, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(skillJSONPath, newData, 0600)
}

// InstallSkill устанавливает навык из реестра
func (i *SkillInjector) InstallSkill(name string, registryClient *RegistryClient) error {
	skillDir := filepath.Join(i.skillsDir, name)

	// Проверяем, существует ли уже навык
	if _, err := os.Stat(skillDir); err == nil {
		return fmt.Errorf("skill %s already installed", name)
	}

	// Скачиваем навык
	if err := registryClient.Download(name, i.skillsDir); err != nil {
		return err
	}

	return nil
}

// UninstallSkill удаляет навык
func (i *SkillInjector) UninstallSkill(name string) error {
	skillDir := filepath.Join(i.skillsDir, name)

	// Проверяем существование
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill %s not found", name)
	}

	// Удаляем директорию навыка
	return os.RemoveAll(skillDir)
}

// GetSkillLastUpdated возвращает время последнего обновления навыка
func (i *SkillInjector) GetSkillLastUpdated(name string) (time.Time, error) {
	skillDir := filepath.Join(i.skillsDir, name)

	info, err := os.Stat(skillDir)
	if err != nil {
		return time.Time{}, err
	}

	return info.ModTime(), nil
}

// CheckForUpdates проверяет наличие обновлений для установленных навыков
func (i *SkillInjector) CheckForUpdates(registryClient *RegistryClient) ([]string, error) {
	updates := make([]string, 0)

	// Загружаем все установленные навыки
	installedSkills, err := i.loadAllSkills()
	if err != nil {
		return nil, err
	}

	// Проверяем каждый навык
	for _, skill := range installedSkills {
		registrySkill, err := registryClient.GetSkill(skill.Name)
		if err != nil {
			continue // Навык не найден в реестре
		}

		// Получаем время последнего обновления
		lastUpdated, err := i.GetSkillLastUpdated(skill.Name)
		if err != nil {
			continue
		}

		// Сравниваем с датой обновления в реестре
		if registrySkill.Updated.After(lastUpdated) {
			updates = append(updates, skill.Name)
		}
	}

	return updates, nil
}
