package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config конфигурация Gateway
type Config struct {
	// Enabled включён ли Gateway
	Enabled bool `json:"enabled"`

	// Port порт для прослушивания
	Port int `json:"port"`

	// Bind адрес привязки (127.0.0.1 или 0.0.0.0)
	Bind string `json:"bind"`

	// Auth конфигурация аутентификации
	Auth AuthConfig `json:"auth"`

	// Tailscale конфигурация
	Tailscale TailscaleConfig `json:"tailscale"`

	// MaxClients максимальное количество клиентов
	MaxClients int `json:"max_clients"`

	// Debug режим отладки
	Debug bool `json:"debug"`
}

// AuthConfig конфигурация аутентификации
type AuthConfig struct {
	// Mode режим аутентификации (none/password/token)
	Mode string `json:"mode"`

	// Password пароль для аутентификации
	Password string `json:"password,omitempty"`

	// Token токен для аутентификации
	Token string `json:"token,omitempty"`

	// AllowList список разрешённых пользователей
	AllowList []string `json:"allow_list,omitempty"`
}

// TailscaleConfig конфигурация Tailscale
type TailscaleConfig struct {
	// Mode режим (none/serve/funnel)
	Mode string `json:"mode"`

	// Hostname имя хоста Tailscale
	Hostname string `json:"hostname"`
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Enabled:    true,
		Port:       18789,
		Bind:       "127.0.0.1",
		MaxClients: 100,
		Debug:      false,
		Auth: AuthConfig{
			Mode:      "password",
			Password:  "", // будет сгенерирован при первом запуске
			AllowList: []string{},
		},
		Tailscale: TailscaleConfig{
			Mode: "none",
		},
	}
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Пытаемся загрузить из файла
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует — сохраняем дефолтную конфигурацию
			if err := SaveConfig(configPath, cfg); err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// SaveConfig сохраняет конфигурацию в файл
func SaveConfig(configPath string, cfg *Config) error {
	// Создаём директорию если не существует
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// EnsureConfig гарантирует существование конфигурационного файла
func EnsureConfig(configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, "gateway.json")
	return LoadConfig(configPath)
}
