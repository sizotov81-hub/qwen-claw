package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TaskStatus статус задачи
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusDisabled  TaskStatus = "disabled"
)

// Task задача планировщика
type Task struct {
	// ID уникальный идентификатор
	ID string `json:"id"`

	// Name имя задачи
	Name string `json:"name"`

	// Description описание задачи
	Description string `json:"description,omitempty"`

	// Command команда для выполнения (query для Qwen)
	Command string `json:"command"`

	// Schedule cron-выражение (например: "0 9 * * *" - каждый день в 9:00)
	// Поддерживаемые форматы:
	// - "@daily", "@hourly", "@weekly", "@monthly"
	// - "*/5 * * * *" - каждые 5 минут
	// - "0 9 * * *" - каждый день в 9:00
	// - "0 0 * * 0" - каждое воскресенье в полночь
	Schedule string `json:"schedule"`

	// Enabled включена ли задача
	Enabled bool `json:"enabled"`

	// LastRun время последнего запуска
	LastRun *time.Time `json:"last_run,omitempty"`

	// NextRun время следующего запуска
	NextRun *time.Time `json:"next_run,omitempty"`

	// RunCount количество запусков
	RunCount int `json:"run_count"`

	// Created время создания
	Created time.Time `json:"created"`

	// Updated время обновления
	Updated time.Time `json:"updated"`
}

// TaskResult результат выполнения задачи
type TaskResult struct {
	// TaskID ID задачи
	TaskID string `json:"task_id"`

	// Started время начала выполнения
	Started time.Time `json:"started"`

	// Completed время завершения
	Completed time.Time `json:"completed"`

	// Success успешно ли выполнено
	Success bool `json:"success"`

	// Output выходные данные
	Output string `json:"output,omitempty"`

	// Error ошибка
	Error string `json:"error,omitempty"`
}

// Executor интерфейс для выполнения задач
type Executor interface {
	Execute(ctx context.Context, command string) (string, error)
}

// Scheduler планировщик задач
type Scheduler struct {
	// dataDir директория для хранения данных
	dataDir string

	// tasks задачи
	tasks map[string]*Task

	// results результаты выполнения
	results []*TaskResult

	// executor исполнитель задач
	executor Executor

	// mu мьютекс
	mu sync.RWMutex

	// stopChan канал для остановки
	stopChan chan struct{}

	// running флаг работы
	running bool
}

// NewScheduler создаёт новый планировщик
func NewScheduler(dataDir string, executor Executor) *Scheduler {
	return &Scheduler{
		dataDir:  dataDir,
		tasks:    make(map[string]*Task),
		results:  make([]*TaskResult, 0),
		executor: executor,
		stopChan: make(chan struct{}),
	}
}

// Init инициализирует планировщик
func (s *Scheduler) Init() error {
	// Создаём директорию
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create scheduler directory: %w", err)
	}

	// Загружаем задачи
	return s.loadTasks()
}

// loadTasks загружает задачи из файла
func (s *Scheduler) loadTasks() error {
	tasksFile := filepath.Join(s.dataDir, "tasks.json")
	data, err := os.ReadFile(tasksFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var tasks []*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("failed to parse tasks: %w", err)
	}

	for _, task := range tasks {
		s.tasks[task.ID] = task
	}

	return nil
}

// saveTasks сохраняет задачи в файл
func (s *Scheduler) saveTasks() error {
	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	tasksFile := filepath.Join(s.dataDir, "tasks.json")
	return os.WriteFile(tasksFile, data, 0644)
}

// Start запускает планировщик
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.runLoop()
}

// Stop останавливает планировщик
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopChan)
	s.stopChan = make(chan struct{})
}

// runLoop основной цикл планировщика
func (s *Scheduler) runLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Проверяем сразу при запуске
	s.checkAndRunTasks()

	for {
		select {
		case <-ticker.C:
			s.checkAndRunTasks()
		case <-s.stopChan:
			return
		}
	}
}

// checkAndRunTasks проверяет и запускает задачи
func (s *Scheduler) checkAndRunTasks() {
	s.mu.RLock()
	now := time.Now()

	var toRun []*Task
	for _, task := range s.tasks {
		if !task.Enabled {
			continue
		}

		nextRun := s.calculateNextRun(task.Schedule, now)
		if nextRun == nil {
			continue
		}

		// Проверяем, пора ли запускать
		if nextRun.Before(now) || nextRun.Equal(now) {
			// Проверяем, не была ли задача уже запущена в это время
			if task.LastRun == nil || nextRun.After(*task.LastRun) {
				toRun = append(toRun, task)
			}
		}
	}
	s.mu.RUnlock()

	// Запускаем задачи
	for _, task := range toRun {
		go s.executeTask(task)
	}
}

// calculateNextRun вычисляет время следующего запуска
func (s *Scheduler) calculateNextRun(schedule string, now time.Time) *time.Time {
	// Обрабатываем специальные cron-выражения
	switch schedule {
	case "@hourly":
		next := now.Add(1 * time.Hour).Truncate(1 * time.Hour)
		return &next

	case "@daily", "@midnight":
		next := now.AddDate(0, 0, 1).Truncate(24 * time.Hour)
		next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
		return &next

	case "@weekly":
		daysUntilSunday := 7 - int(now.Weekday())
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		next := now.AddDate(0, 0, daysUntilSunday).Truncate(24 * time.Hour)
		next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
		return &next

	case "@monthly":
		next := now.AddDate(0, 1, 0).Truncate(24 * time.Hour)
		next = time.Date(next.Year(), next.Month(), 1, 0, 0, 0, 0, next.Location())
		return &next

	case "@yearly", "@annually":
		next := now.AddDate(1, 0, 0).Truncate(24 * time.Hour)
		next = time.Date(next.Year(), 1, 1, 0, 0, 0, 0, next.Location())
		return &next
	}

	// Парсим cron-выражение
	return s.parseCronExpression(schedule, now)
}

// parseCronExpression парсит cron-выражение
func (s *Scheduler) parseCronExpression(expr string, now time.Time) *time.Time {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil
	}

	minute := parts[0]
	hour := parts[1]
	dayOfMonth := parts[2]
	month := parts[3]
	dayOfWeek := parts[4]

	// Начинаем с текущей минуты и идём вперёд
	for i := 0; i < 525600; i++ { // Максимум год вперёд
		candidate := now.Add(time.Duration(i) * time.Minute).Truncate(time.Minute)

		if s.matchField(minute, candidate.Minute(), 0, 59) &&
			s.matchField(hour, candidate.Hour(), 0, 23) &&
			s.matchField(dayOfMonth, candidate.Day(), 1, 31) &&
			s.matchField(month, int(candidate.Month()), 1, 12) &&
			s.matchField(dayOfWeek, int(candidate.Weekday()), 0, 6) {

			if candidate.After(now) {
				return &candidate
			}
		}
	}

	return nil
}

// matchField проверяет соответствие поля cron
func (s *Scheduler) matchField(field string, value, min, max int) bool {
	if field == "*" {
		return true
	}

	// Проверяем шаг (*/5)
	if strings.HasPrefix(field, "*/") {
		var step int
		if _, err := fmt.Sscanf(field, "*/%d", &step); err == nil && step > 0 {
			return value%step == 0
		}
	}

	// Проверяем конкретное значение
	var val int
	if _, err := fmt.Sscanf(field, "%d", &val); err == nil {
		return value == val
	}

	// Проверяем диапазон (1-5)
	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) == 2 {
			var start, end int
			if _, err := fmt.Sscanf(parts[0], "%d", &start); err != nil {
				return false
			}
			if _, err := fmt.Sscanf(parts[1], "%d", &end); err != nil {
				return false
			}
			return value >= start && value <= end
		}
	}

	// Проверяем список (1,3,5)
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		for _, p := range parts {
			var val int
			if _, err := fmt.Sscanf(p, "%d", &val); err == nil && value == val {
				return true
			}
		}
		return false
	}

	return false
}

// executeTask выполняет задачу
func (s *Scheduler) executeTask(task *Task) {
	// Обновляем статус задачи
	s.mu.Lock()
	task.LastRun = ptrTime(time.Now())
	task.NextRun = s.calculateNextRun(task.Schedule, time.Now())
	task.RunCount++
	task.Updated = time.Now()
	_ = s.saveTasks()
	s.mu.Unlock()

	// Выполняем задачу
	result := &TaskResult{
		TaskID:  task.ID,
		Started: time.Now(),
	}

	ctx := context.Background()
	output, err := s.executor.Execute(ctx, task.Command)

	result.Completed = time.Now()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
		result.Output = output
	}

	// Сохраняем результат
	s.mu.Lock()
	s.results = append(s.results, result)
	// Храним только последние 100 результатов
	if len(s.results) > 100 {
		s.results = s.results[len(s.results)-100:]
	}
	s.mu.Unlock()
}

// AddTask добавляет задачу
func (s *Scheduler) AddTask(name, description, command, schedule string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := generateID()
	now := time.Now()

	task := &Task{
		ID:          id,
		Name:        name,
		Description: description,
		Command:     command,
		Schedule:    schedule,
		Enabled:     true,
		Created:     now,
		Updated:     now,
		NextRun:     s.calculateNextRun(schedule, now),
	}

	s.tasks[id] = task

	if err := s.saveTasks(); err != nil {
		delete(s.tasks, id)
		return nil, err
	}

	return task, nil
}

// RemoveTask удаляет задачу
func (s *Scheduler) RemoveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	delete(s.tasks, id)
	return s.saveTasks()
}

// GetTask получает задачу по ID
func (s *Scheduler) GetTask(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	return task, nil
}

// ListTasks возвращает список всех задач
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		result = append(result, task)
	}
	return result
}

// GetResults возвращает результаты выполнения
func (s *Scheduler) GetResults(limit int) []*TaskResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.results) {
		return s.results
	}

	return s.results[len(s.results)-limit:]
}

// EnableTask включает задачу
func (s *Scheduler) EnableTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	task.Enabled = true
	task.Updated = time.Now()
	task.NextRun = s.calculateNextRun(task.Schedule, time.Now())
	return s.saveTasks()
}

// DisableTask отключает задачу
func (s *Scheduler) DisableTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	task.Enabled = false
	task.Updated = time.Now()
	return s.saveTasks()
}

// RunTaskNow выполняет задачу немедленно
func (s *Scheduler) RunTaskNow(id string) (*TaskResult, error) {
	s.mu.RLock()
	task, ok := s.tasks[id]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	result := &TaskResult{
		TaskID:  task.ID,
		Started: time.Now(),
	}

	ctx := context.Background()
	output, err := s.executor.Execute(ctx, task.Command)

	result.Completed = time.Now()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
		result.Output = output
	}

	// Сохраняем результат
	s.mu.Lock()
	s.results = append(s.results, result)
	if len(s.results) > 100 {
		s.results = s.results[len(s.results)-100:]
	}
	s.mu.Unlock()

	return result, nil
}

// Вспомогательные функции

func generateID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
