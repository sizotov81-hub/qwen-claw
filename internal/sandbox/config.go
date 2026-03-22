package sandbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// SandboxMode режим sandbox
type SandboxMode string

const (
	// SandboxModeOff sandbox отключён
	SandboxModeOff SandboxMode = "off"
	
	// SandboxModePerSession sandbox для каждой сессии
	SandboxModePerSession SandboxMode = "per-session"
	
	// SandboxModePerAgent sandbox для каждого агента
	SandboxModePerAgent SandboxMode = "per-agent"
	
	// SandboxModeAlways всегда использовать sandbox
	SandboxModeAlways SandboxMode = "always"
)

// Config конфигурация sandbox
type Config struct {
	// Enabled включён ли sandbox
	Enabled bool `json:"enabled"`
	
	// Mode режим работы
	Mode SandboxMode `json:"mode"`
	
	// DockerImage Docker образ для sandbox
	DockerImage string `json:"docker_image"`
	
	// NetworkEnabled разрешена ли сеть
	NetworkEnabled bool `json:"network_enabled"`
	
	// Mounts точки монтирования
	Mounts []Mount `json:"mounts"`
	
	// Resources лимиты ресурсов
	Resources ResourceLimits `json:"resources"`
	
	// AllowedTools разрешённые инструменты
	AllowedTools []string `json:"allowed_tools"`
	
	// DeniedTools запрещённые инструменты
	DeniedTools []string `json:"denied_tools"`
	
	// Timeout таймаут выполнения команд
	Timeout time.Duration `json:"timeout"`
	
	// MaxOutputSize максимальный размер вывода (байты)
	MaxOutputSize int64 `json:"max_output_size"`
}

// Mount точка монтирования
type Mount struct {
	// Source источник (хост)
	Source string `json:"source"`
	
	// Destination назначение (контейнер)
	Destination string `json:"destination"`
	
	// ReadOnly только для чтения
	ReadOnly bool `json:"read_only"`
}

// ResourceLimits лимиты ресурсов
type ResourceLimits struct {
	// Memory лимит памяти (например, "512m")
	Memory string `json:"memory"`
	
	// CPU лимит CPU (например, "0.5")
	CPU string `json:"cpu"`
	
	// MaxProcesses максимальное количество процессов
	MaxProcesses int `json:"max_processes"`
	
	// MaxFiles максимальное количество открытых файлов
	MaxFiles int `json:"max_files"`
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Enabled:       true,
		Mode:          SandboxModePerSession,
		DockerImage:   "alpine:latest",
		NetworkEnabled: false,
		Mounts: []Mount{
			{
				Source:      "/tmp/qwen-claw-sandbox",
				Destination: "/workspace",
				ReadOnly:    false,
			},
		},
		Resources: ResourceLimits{
			Memory:       "512m",
			CPU:          "0.5",
			MaxProcesses: 10,
			MaxFiles:     100,
		},
		AllowedTools: []string{
			"bash",
			"sh",
			"read",
			"write",
			"edit",
			"process",
		},
		DeniedTools: []string{
			"browser",
			"nodes",
			"sudo",
		},
		Timeout:       5 * time.Minute,
		MaxOutputSize: 10 * 1024 * 1024, // 10MB
	}
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Сохраняем дефолтную конфигурацию
			return config, SaveConfig(configPath, config)
		}
		return nil, err
	}
	
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}
	
	return config, nil
}

// SaveConfig сохраняет конфигурацию в файл
func SaveConfig(configPath string, config *Config) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0600)
}

// IsToolAllowed проверяет, разрешён ли инструмент
func (c *Config) IsToolAllowed(tool string) bool {
	// Если есть denied tools — проверяем сначала
	for _, denied := range c.DeniedTools {
		if tool == denied {
			return false
		}
	}
	
	// Если allowed tools пуст — все разрешены кроме denied
	if len(c.AllowedTools) == 0 {
		return true
	}
	
	// Проверяем allowed
	for _, allowed := range c.AllowedTools {
		if tool == allowed {
			return true
		}
	}
	
	return false
}

// Validate проверяает конфигурацию
func (c *Config) Validate() error {
	if c.Enabled && c.Mode == SandboxModeOff {
		return nil // Sandbox отключён
	}
	
	if c.DockerImage == "" {
		return ErrInvalidConfig{Field: "docker_image", Message: "must be set"}
	}
	
	if c.Resources.Memory == "" {
		return ErrInvalidConfig{Field: "resources.memory", Message: "must be set"}
	}
	
	if c.Timeout <= 0 {
		return ErrInvalidConfig{Field: "timeout", Message: "must be positive"}
	}
	
	return nil
}

// ErrInvalidConfig ошибка неверной конфигурации
type ErrInvalidConfig struct {
	Field   string
	Message string
}

func (e ErrInvalidConfig) Error() string {
	return "invalid config: " + e.Field + " " + e.Message
}
