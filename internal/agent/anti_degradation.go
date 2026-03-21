package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TaskMetric метрики задачи
type TaskMetric struct {
	Query       string        `json:"query"`
	Duration    time.Duration `json:"duration"`
	Attempts    int           `json:"attempts"`
	Errors      []string      `json:"errors"`
	Success     bool          `json:"success"`
	Timestamp   time.Time     `json:"timestamp"`
	PatternType string        `json:"pattern_type,omitempty"`
}

// DegradationPattern тип паттерна деградации
type DegradationPattern string

const (
	PatternCyclicError     DegradationPattern = "cyclic_error"
	PatternSlowExecution   DegradationPattern = "slow_execution"
	PatternMultipleFailures DegradationPattern = "multiple_failures"
	PatternVerboseResponse DegradationPattern = "verbose_response"
	PatternContextLoss     DegradationPattern = "context_loss"
)

// AntiDegradationSystem система антидеградации
type AntiDegradationSystem struct {
	mu                sync.RWMutex
	metrics           []*TaskMetric
	degradationLog    []string
	lessonsLearned    map[string]bool
	dataDir           string
	degradationThresholds DegradationThresholds
}

// DegradationThresholds пороги для обнаружения деградации
type DegradationThresholds struct {
	MaxDuration       time.Duration // Максимальное время выполнения
	MaxAttempts       int           // Максимальное количество попыток
	MaxErrorRepeat    int           // Максимальное повторение ошибки
	MaxResponseLength int           // Максимальная длина ответа
}

// DefaultThresholds пороги по умолчанию
func DefaultThresholds() DegradationThresholds {
	return DegradationThresholds{
		MaxDuration:       30 * time.Second,
		MaxAttempts:       3,
		MaxErrorRepeat:    2,
		MaxResponseLength: 500,
	}
}

// NewAntiDegradationSystem создаёт систему антидеградации
func NewAntiDegradationSystem(dataDir string) *AntiDegradationSystem {
	ads := &AntiDegradationSystem{
		metrics:                make([]*TaskMetric, 0),
		degradationLog:         make([]string, 0),
		lessonsLearned:         make(map[string]bool),
		dataDir:                dataDir,
		degradationThresholds:  DefaultThresholds(),
	}
	ads.load()
	return ads
}

// RecordTask записывает метрики задачи
func (ads *AntiDegradationSystem) RecordTask(query string, duration time.Duration, attempts int, errors []string, success bool) {
	ads.mu.Lock()
	defer ads.mu.Unlock()

	metric := &TaskMetric{
		Query:     query,
		Duration:  duration,
		Attempts:  attempts,
		Errors:    errors,
		Success:   success,
		Timestamp: time.Now(),
	}

	ads.metrics = append(ads.metrics, metric)

	// Проверяем на паттерны деградации
	if pattern := ads.detectPattern(metric); pattern != "" {
		metric.PatternType = string(pattern)
		ads.degradationLog = append(ads.degradationLog, string(pattern))
		ads.save()
	}

	// Сохраняем каждые 10 записей
	if len(ads.metrics)%10 == 0 {
		ads.save()
	}
}

// detectPattern обнаруживает паттерн деградации
func (ads *AntiDegradationSystem) detectPattern(metric *TaskMetric) DegradationPattern {
	// Проверка на медленное выполнение
	if metric.Duration > ads.degradationThresholds.MaxDuration {
		return PatternSlowExecution
	}

	// Проверка на множественные неудачи
	if metric.Attempts > ads.degradationThresholds.MaxAttempts && !metric.Success {
		return PatternMultipleFailures
	}

	// Проверка на циклические ошибки
	if len(metric.Errors) > 0 {
		errorCount := 0
		for _, m := range ads.metrics {
			for _, e := range m.Errors {
				if e == metric.Errors[0] {
					errorCount++
				}
			}
		}
		if errorCount > ads.degradationThresholds.MaxErrorRepeat {
			return PatternCyclicError
		}
	}

	return ""
}

// GetRecentMetrics возвращает последние метрики
func (ads *AntiDegradationSystem) GetRecentMetrics(limit int) []*TaskMetric {
	ads.mu.RLock()
	defer ads.mu.RUnlock()

	if limit > len(ads.metrics) {
		return ads.metrics
	}
	return ads.metrics[len(ads.metrics)-limit:]
}

// GetDegradationReport возвращает отчёт о деградации
func (ads *AntiDegradationSystem) GetDegradationReport() DegradationReport {
	ads.mu.RLock()
	defer ads.mu.RUnlock()

	report := DegradationReport{
		TotalTasks:           len(ads.metrics),
		DegradationCount:     len(ads.degradationLog),
		LessonsCount:         len(ads.lessonsLearned),
		Patterns:             make(map[string]int),
		RecentDegradations:   []string{},
	}

	// Считаем паттерны
	for _, pattern := range ads.degradationLog {
		report.Patterns[pattern]++
	}

	// Берём последние 10 (без паники если меньше)
	if len(ads.degradationLog) > 10 {
		report.RecentDegradations = ads.degradationLog[len(ads.degradationLog)-10:]
	} else {
		report.RecentDegradations = ads.degradationLog
	}

	// Вычисляем эффективность
	if report.TotalTasks > 0 {
		report.Efficiency = float64(report.TotalTasks-report.DegradationCount) / float64(report.TotalTasks) * 100
	}

	return report
}

// DegradationReport отчёт о деградации
type DegradationReport struct {
	TotalTasks           int            `json:"total_tasks"`
	DegradationCount     int            `json:"degradation_count"`
	LessonsCount         int            `json:"lessons_count"`
	Patterns             map[string]int `json:"patterns"`
	RecentDegradations   []string       `json:"recent_degradations"`
	Efficiency           float64        `json:"efficiency"`
}

// MarkLessonLearned отмечает урок как усвоенный
func (ads *AntiDegradationSystem) MarkLessonLearned(lessonKey string) {
	ads.mu.Lock()
	defer ads.mu.Unlock()
	ads.lessonsLearned[lessonKey] = true
	ads.save()
}

// IsLessonLearned проверяет, усвоен ли урок
func (ads *AntiDegradationSystem) IsLessonLearned(lessonKey string) bool {
	ads.mu.RLock()
	defer ads.mu.RUnlock()
	return ads.lessonsLearned[lessonKey]
}

// save сохраняет данные
func (ads *AntiDegradationSystem) save() {
	os.MkdirAll(ads.dataDir, 0755)

	// Сохраняем метрики
	if data, err := json.MarshalIndent(ads.metrics, "", "  "); err == nil {
		os.WriteFile(filepath.Join(ads.dataDir, "metrics.json"), data, 0644)
	}

	// Сохраняем уроки
	if data, err := json.MarshalIndent(ads.lessonsLearned, "", "  "); err == nil {
		os.WriteFile(filepath.Join(ads.dataDir, "lessons.json"), data, 0644)
	}
}

// load загружает данные
func (ads *AntiDegradationSystem) load() {
	// Загружаем метрики
	if data, err := os.ReadFile(filepath.Join(ads.dataDir, "metrics.json")); err == nil {
		json.Unmarshal(data, &ads.metrics)
	}

	// Загружаем уроки
	if data, err := os.ReadFile(filepath.Join(ads.dataDir, "lessons.json")); err == nil {
		json.Unmarshal(data, &ads.lessonsLearned)
	}
}

// GetStats возвращает статистику для CLI
func (ads *AntiDegradationSystem) GetStats() string {
	report := ads.GetDegradationReport()

	output := "📊 Anti-Degradation Report\n"
	output += "=========================\n\n"
	output += fmt.Sprintf("✅ Total tasks: %d\n", report.TotalTasks)
	output += fmt.Sprintf("⚠️  Degradations: %d\n", report.DegradationCount)
	output += fmt.Sprintf("📚 Lessons learned: %d\n", report.LessonsCount)
	output += fmt.Sprintf("📈 Efficiency: %.1f%%\n\n", report.Efficiency)

	if len(report.Patterns) > 0 {
		output += "Patterns detected:\n"
		for pattern, count := range report.Patterns {
			output += fmt.Sprintf("  - %s: %d\n", pattern, count)
		}
	}

	return output
}
