package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Sandbox Docker sandbox
type Sandbox struct {
	mu sync.RWMutex
	
	// ID идентификатор sandbox
	ID string
	
	// Config конфигурация
	Config *Config
	
	// Policy политика безопасности
	Policy *SecurityPolicy
	
	// ContainerID ID контейнера
	ContainerID string
	
	// WorkDir рабочая директория
	WorkDir string
	
	// Created время создания
	Created time.Time
	
	// LastUsed время последнего использования
	LastUsed time.Time
	
	// ExecCount количество выполнений
	ExecCount int
	
	// isRunning запущен ли контейнер
	isRunning bool
}

// Manager менеджер sandbox
type Manager struct {
	mu        sync.RWMutex
	config    *Config
	policy    *SecurityPolicy
	sandboxes map[string]*Sandbox
	dataDir   string
}

// SetPolicy устанавливает политику безопасности
func (m *Manager) SetPolicy(policy *SecurityPolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policy = policy
}

// NewManager создаёт новый менеджер sandbox
func NewManager(config *Config, dataDir string) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}
	
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}
	
	return &Manager{
		config:    config,
		policy:    DefaultSecurityPolicy(),
		sandboxes: make(map[string]*Sandbox),
		dataDir:   dataDir,
	}, nil
}

// CreateSandbox создаёт новый sandbox
func (m *Manager) CreateSandbox(id string) (*Sandbox, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Проверяем, существует ли уже
	if sb, ok := m.sandboxes[id]; ok {
		return sb, nil
	}
	
	// Создаём рабочую директорию
	workDir := filepath.Join(m.dataDir, id)
	if err := os.MkdirAll(workDir, 0700); err != nil {
		return nil, err
	}
	
	sandbox := &Sandbox{
		ID:        id,
		Config:    m.config,
		Policy:    m.policy,
		WorkDir:   workDir,
		Created:   time.Now(),
		LastUsed:  time.Now(),
		ExecCount: 0,
	}
	
	m.sandboxes[id] = sandbox
	
	return sandbox, nil
}

// GetSandbox получает sandbox
func (m *Manager) GetSandbox(id string) (*Sandbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	sb, ok := m.sandboxes[id]
	if !ok {
		return nil, ErrSandboxNotFound{ID: id}
	}
	
	return sb, nil
}

// Execute выполняет команду в sandbox
func (m *Manager) Execute(ctx context.Context, id string, command string) (*ExecutionResult, error) {
	sb, err := m.GetSandbox(id)
	if err != nil {
		return nil, err
	}
	
	return sb.Execute(ctx, command)
}

// RemoveSandbox удаляет sandbox
func (m *Manager) RemoveSandbox(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	sb, ok := m.sandboxes[id]
	if !ok {
		return ErrSandboxNotFound{ID: id}
	}
	
	// Останавливаем контейнер если запущен
	if sb.isRunning {
		sb.Stop()
	}
	
	// Удаляем рабочую директорию
	if err := os.RemoveAll(sb.WorkDir); err != nil {
		return err
	}
	
	delete(m.sandboxes, id)
	return nil
}

// ListSandboxes возвращает список sandbox
func (m *Manager) ListSandboxes() []*Sandbox {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	list := make([]*Sandbox, 0, len(m.sandboxes))
	for _, sb := range m.sandboxes {
		list = append(list, sb)
	}
	
	return list
}

// Cleanup очищает старые sandbox
func (m *Manager) Cleanup(maxAge time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	for id, sb := range m.sandboxes {
		if now.Sub(sb.LastUsed) > maxAge {
			if sb.isRunning {
				sb.Stop()
			}
			os.RemoveAll(sb.WorkDir)
			delete(m.sandboxes, id)
		}
	}
	
	return nil
}

// Start запускает контейнер
func (s *Sandbox) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.isRunning {
		return nil
	}
	
	// Проверяем, доступен ли Docker
	if !isDockerAvailable() {
		return ErrDockerNotAvailable{}
	}
	
	// Формируем команду docker run
	args := []string{
		"run",
		"-d",
		"--rm",
		"--name", fmt.Sprintf("qwen-claw-sandbox-%s", s.ID),
	}
	
	// Добавляем лимиты ресурсов
	if s.Config.Resources.Memory != "" {
		args = append(args, "--memory", s.Config.Resources.Memory)
	}
	if s.Config.Resources.CPU != "" {
		args = append(args, "--cpus", s.Config.Resources.CPU)
	}
	
	// Отключаем сеть если не разрешена
	if !s.Config.NetworkEnabled {
		args = append(args, "--network", "none")
	}
	
	// Добавляем точки монтирования
	for _, mount := range s.Config.Mounts {
		mountStr := fmt.Sprintf("%s:%s", mount.Source, mount.Destination)
		if mount.ReadOnly {
			mountStr += ":ro"
		}
		args = append(args, "-v", mountStr)
	}
	
	// Добавляем лимиты процессов и файлов
	if s.Config.Resources.MaxProcesses > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", s.Config.Resources.MaxProcesses))
	}
	if s.Config.Resources.MaxFiles > 0 {
		args = append(args, "--ulimit", fmt.Sprintf("nofile=%d:%d", s.Config.Resources.MaxFiles, s.Config.Resources.MaxFiles))
	}
	
	// Добавляем образ
	args = append(args, s.Config.DockerImage)
	args = append(args, "tail", "-f", "/dev/null") // Держим контейнер запущенным
	
	// Запускаем
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container: %w, output: %s", err, string(output))
	}
	
	s.ContainerID = strings.TrimSpace(string(output))
	s.isRunning = true
	
	return nil
}

// Stop останавливает контейнер
func (s *Sandbox) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isRunning || s.ContainerID == "" {
		return nil
	}
	
	// Останавливаем контейнер
	cmd := exec.Command("docker", "stop", "-t", "5", s.ContainerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop container: %w, output: %s", err, string(output))
	}
	
	s.isRunning = false
	s.ContainerID = ""
	
	return nil
}

// ExecutionResult результат выполнения
type ExecutionResult struct {
	// ExitCode код выхода
	ExitCode int
	
	// Stdout стандартный вывод
	Stdout string
	
	// Stderr стандартный поток ошибок
	Stderr string
	
	// Duration длительность выполнения
	Duration time.Duration
	
	// TimedOut превышен ли таймаут
	TimedOut bool
}

// Execute выполняет команду в sandbox
func (s *Sandbox) Execute(ctx context.Context, command string) (*ExecutionResult, error) {
	s.mu.Lock()
	s.LastUsed = time.Now()
	s.ExecCount++
	s.mu.Unlock()
	
	// Проверяем команду на безопасность
	if err := s.Policy.CheckCommand(command); err != nil {
		return nil, err
	}
	
	startTime := time.Now()
	
	// Если sandbox не запущен и включён — запускаем
	if s.Config.Enabled && !s.isRunning {
		if err := s.Start(); err != nil {
			return nil, err
		}
	}
	
	var stdout, stderr bytes.Buffer
	
	var cmd *exec.Cmd
	if s.isRunning {
		// Выполняем в контейнере
		args := []string{
			"exec",
			s.ContainerID,
			"sh", "-c", command,
		}
		cmd = exec.CommandContext(ctx, "docker", args...)
	} else {
		// Выполняем локально с ограничениями
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = s.WorkDir
	}
	
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Запускаем
	err := cmd.Run()
	duration := time.Since(startTime)
	
	result := &ExecutionResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}
	
	// Проверяем таймаут
	if ctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
	}
	
	// Получаем код выхода
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else if err == nil {
		result.ExitCode = 0
	} else {
		result.ExitCode = -1
	}
	
	// Проверяем размер вывода
	if int64(len(result.Stdout)+len(result.Stderr)) > s.Config.MaxOutputSize {
		return nil, ErrOutputTooLarge{
			Size: int64(len(result.Stdout) + len(result.Stderr)),
			Limit: s.Config.MaxOutputSize,
		}
	}
	
	return result, nil
}

// GetStats возвращает статистику sandbox
func (s *Sandbox) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"id":          s.ID,
		"container":   s.ContainerID,
		"running":     s.isRunning,
		"work_dir":    s.WorkDir,
		"created":     s.Created.Format(time.RFC3339),
		"last_used":   s.LastUsed.Format(time.RFC3339),
		"exec_count":  s.ExecCount,
		"mode":        s.Config.Mode,
		"enabled":     s.Config.Enabled,
	}
}

// isDockerAvailable проверяет доступность Docker
func isDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

// ErrSandboxNotFound ошибка: sandbox не найден
type ErrSandboxNotFound struct {
	ID string
}

func (e ErrSandboxNotFound) Error() string {
	return fmt.Sprintf("sandbox not found: %s", e.ID)
}

// ErrDockerNotAvailable ошибка: Docker недоступен
type ErrDockerNotAvailable struct{}

func (e ErrDockerNotAvailable) Error() string {
	return "docker is not available or not installed"
}

// ErrOutputTooLarge ошибка: вывод слишком большой
type ErrOutputTooLarge struct {
	Size  int64
	Limit int64
}

func (e ErrOutputTooLarge) Error() string {
	return fmt.Sprintf("output too large: %d bytes (limit: %d)", e.Size, e.Limit)
}
