package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func init() {
	// Загружаем .env файл из директории проекта
	loadEnvFile()
}

// loadEnvFile загружает .env файл из директории проекта
func loadEnvFile() {
	// Определяем директорию проекта
	execPath, err := os.Executable()
	if err == nil {
		// Пытаемся загрузить .env из той же директории, где лежит бинарник
		binDir := filepath.Dir(execPath)
		_ = godotenv.Load(filepath.Join(binDir, ".env"))
	}
	
	// Также пробуем загрузить из текущей директории (на случай запуска из ~/bin)
	_ = godotenv.Load(".env")
	
	// И из домашней директории
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		_ = godotenv.Load(filepath.Join(homeDir, "qwen-claw", ".env"))
		_ = godotenv.Load(filepath.Join(homeDir, ".env"))
	}
}

// Config основная конфигурация приложения
type Config struct {
	// BaseDir базовая директория проекта
	BaseDir string `mapstructure:"base_dir"`
	
	// QwenDir директория для данных Qwen (.qwen)
	QwenDir string `mapstructure:"qwen_dir"`
	
	// SkillsDir директория для навыков
	SkillsDir string `mapstructure:"skills_dir"`
	
	// MemoryDir директория для памяти
	MemoryDir string `mapstructure:"memory_dir"`
	
	// Server настройки сервера
	Server ServerConfig `mapstructure:"server"`
	
	// Telegram настройки Telegram бота
	Telegram TelegramConfig `mapstructure:"telegram"`
	
	// LLM настройки LLM провайдера
	LLM LLMConfig `mapstructure:"llm"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

// TelegramConfig конфигурация Telegram бота
type TelegramConfig struct {
	Enabled      bool    `mapstructure:"enabled"`
	Token        string  `mapstructure:"token"`
	AllowedUsers []int64 `mapstructure:"allowed_users"`
}

// LLMConfig конфигурация LLM
type LLMConfig struct {
	// Model модель для Qwen Code CLI
	Model string `mapstructure:"model"`
	
	// ApprovalMode режим подтверждения (plan/default/auto-edit/yolo)
	ApprovalMode string `mapstructure:"approval_mode"`
	
	// Debug режим отладки
	Debug bool `mapstructure:"debug"`
}

// Default возвращает конфигурацию по умолчанию
func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, "qwen-claw")
	
	return &Config{
		BaseDir:   baseDir,
		QwenDir:   filepath.Join(baseDir, ".qwen"),
		SkillsDir: filepath.Join(baseDir, "skills"),
		MemoryDir: filepath.Join(baseDir, ".qwen", "memory"),
		Server: ServerConfig{
			Enabled: true,
			Host:    "127.0.0.1",
			Port:    18789,
		},
		Telegram: TelegramConfig{
			Enabled: false,
			Token:   "",
		},
		LLM: LLMConfig{
			Model:        "", // используем модель по умолчанию из qwen cli
			ApprovalMode: "auto-edit",
			Debug:        false,
		},
	}
}

// Load загружает конфигурацию из файла
func Load(configPath string) (*Config, error) {
	cfg := Default()
	
	// Если файл конфигурации существует, загружаем его
	if configPath != "" {
		viper.SetConfigFile(configPath)
		viper.SetConfigType("yaml")
		
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}
	
	// Переопределяем из переменных окружения
	viper.AutomaticEnv()
	viper.SetEnvPrefix("QWEN_CLAW")
	
	// Явно читаем переменные окружения для Telegram
	if token := os.Getenv("QWEN_CLAW_TELEGRAM_TOKEN"); token != "" {
		cfg.Telegram.Token = token
		cfg.Telegram.Enabled = true
	}
	
	// Привязываем к структуре
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Обрабатываем список разрешённых пользователей из env
	if allowedUsers := os.Getenv("QWEN_CLAW_TELEGRAM_ALLOWED_USERS"); allowedUsers != "" {
		cfg.Telegram.AllowedUsers = parseAllowedUsers(allowedUsers)
	}
	
	return cfg, nil
}

// parseAllowedUsers парсит список ID пользователей из строки
func parseAllowedUsers(s string) []int64 {
	if s == "" {
		return nil
	}
	
	parts := strings.Split(s, ",")
	ids := make([]int64, 0, len(parts))
	
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		
		var id int64
		_, err := fmt.Sscanf(p, "%d", &id)
		if err == nil {
			ids = append(ids, id)
		}
	}
	
	return ids
}

// EnsureDirs создаёт необходимые директории
func (c *Config) EnsureDirs() error {
	dirs := []string{
		c.BaseDir,
		c.QwenDir,
		c.SkillsDir,
		c.MemoryDir,
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	return nil
}

// Save сохраняет конфигурацию в файл
func (c *Config) Save(path string) error {
	viper.Set("base_dir", c.BaseDir)
	viper.Set("qwen_dir", c.QwenDir)
	viper.Set("skills_dir", c.SkillsDir)
	viper.Set("memory_dir", c.MemoryDir)
	viper.Set("server", c.Server)
	viper.Set("telegram", c.Telegram)
	viper.Set("llm", c.LLM)
	
	return viper.WriteConfigAs(path)
}
