// Package config предоставляет конфигурацию для микросервисов Qwen-Claw
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config полная конфигурация сервиса
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Logger        LoggerConfig        `mapstructure:"logger"`
	Tracing       TracingConfig       `mapstructure:"tracing"`
	Metrics       MetricsConfig       `mapstructure:"metrics"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Redis         RedisConfig         `mapstructure:"redis"`
	RabbitMQ      RabbitMQConfig      `mapstructure:"rabbitmq"`
	GRPCClients   GRPCClientsConfig   `mapstructure:"grpc_clients"`
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
	RateLimit     RateLimitConfig     `mapstructure:"rate_limit"`
	Security      SecurityConfig      `mapstructure:"security"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	GRPCPort int    `mapstructure:"grpc_port"`
	HTTPPort int    `mapstructure:"http_port"`
	Host     string `mapstructure:"host"`
}

// LoggerConfig конфигурация логирования
type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// TracingConfig конфигурация трассировки
type TracingConfig struct {
	Enabled        bool    `mapstructure:"enabled"`
	ServiceName    string  `mapstructure:"service_name"`
	JaegerEndpoint string  `mapstructure:"jaeger_endpoint"`
	SamplingRatio  float64 `mapstructure:"sampling_ratio"`
}

// MetricsConfig конфигурация метрик
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

// DatabaseConfig конфигурация БД
type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig конфигурация Redis
type RedisConfig struct {
	URL        string `mapstructure:"url"`
	MaxRetries int    `mapstructure:"max_retries"`
	PoolSize   int    `mapstructure:"pool_size"`
}

// RabbitMQConfig конфигурация RabbitMQ
type RabbitMQConfig struct {
	URL       string           `mapstructure:"url"`
	Exchanges []ExchangeConfig `mapstructure:"exchanges"`
	Queues    []QueueConfig    `mapstructure:"queues"`
}

// ExchangeConfig конфигурация exchange
type ExchangeConfig struct {
	Name string `mapstructure:"name"`
	Type string `mapstructure:"type"`
}

// QueueConfig конфигурация очереди
type QueueConfig struct {
	Name   string `mapstructure:"name"`
	Durable bool  `mapstructure:"durable"`
}

// GRPCClientsConfig конфигурация gRPC клиентов
type GRPCClientsConfig struct {
	SessionMemory GRPCClientConfig `mapstructure:"session_memory"`
	QwenWrapper   GRPCClientConfig `mapstructure:"qwen_wrapper"`
	LLMProxy      GRPCClientConfig `mapstructure:"llm_proxy"`
	ToolsExecutor GRPCClientConfig `mapstructure:"tools_executor"`
}

// GRPCClientConfig конфигурация gRPC клиента
type GRPCClientConfig struct {
	Address string        `mapstructure:"address"`
	Timeout time.Duration `mapstructure:"timeout"`
	Retry   RetryConfig   `mapstructure:"retry"`
}

// RetryConfig конфигурация retry
type RetryConfig struct {
	MaxAttempts int           `mapstructure:"max_attempts"`
	Backoff     time.Duration `mapstructure:"backoff"`
}

// CircuitBreakerConfig конфигурация circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold int           `mapstructure:"failure_threshold"`
	SuccessThreshold int           `mapstructure:"success_threshold"`
	Timeout          time.Duration `mapstructure:"timeout"`
	HalfOpenMaxCalls int           `mapstructure:"half_open_max_calls"`
}

// RateLimitConfig конфигурация rate limiting
type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerSecond int  `mapstructure:"requests_per_second"`
	BurstSize         int  `mapstructure:"burst_size"`
}

// SecurityConfig конфигурация безопасности
type SecurityConfig struct {
	JWTSecret string        `mapstructure:"jwt_secret"`
	JWTTTL    time.Duration `mapstructure:"jwt_ttl"`
	APIKeys   []string      `mapstructure:"api_keys"`
}

// Default возвращает конфигурацию по умолчанию
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			GRPCPort: 55051,
			HTTPPort: 58080,
			Host:     "0.0.0.0",
		},
		Logger: LoggerConfig{
			Level:  "info",
			Format: "json",
		},
		Tracing: TracingConfig{
			Enabled:        true,
			ServiceName:    "qwen-claw",
			JaegerEndpoint: "jaeger:4317",
			SamplingRatio:  1.0,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Port:    59090,
			Path:    "/metrics",
		},
		Database: DatabaseConfig{
			URL:             "postgres://qwenclaw:qwenclaw_password@localhost:5432/qwenclaw?sslmode=disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Redis: RedisConfig{
			URL:        "redis://localhost:6379",
			MaxRetries: 3,
			PoolSize:   10,
		},
		RabbitMQ: RabbitMQConfig{
			URL: "amqp://qwenclaw:qwenclaw_password@localhost:5672/",
		},
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 3,
			Timeout:          30 * time.Second,
			HalfOpenMaxCalls: 3,
		},
		RateLimit: RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: 100,
			BurstSize:         200,
		},
		Security: SecurityConfig{
			JWTSecret: "change-me-in-production",
			JWTTTL:    24 * time.Hour,
		},
	}
}

// Load загружает конфигурацию из файла
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Переопределяем из переменных окружения
	viper.AutomaticEnv()
	viper.SetEnvPrefix("QWEN_CLAW")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Применяем значения по умолчанию для незаданных полей
	defaults := Default()
	mergeWithDefaults(&cfg, defaults)

	return &cfg, nil
}

// LoadFromEnv загружает конфигурацию из переменных окружения
func LoadFromEnv(serviceName string) (*Config, error) {
	cfg := Default()

	// Переопределяем service-specific настройки
	if serviceName != "" {
		cfg.Tracing.ServiceName = serviceName
	}

	// Читаем из env
	if val := os.Getenv("QWEN_CLAW_GRPC_PORT"); val != "" {
		fmt.Sscanf(val, "%d", &cfg.Server.GRPCPort)
	}
	if val := os.Getenv("QWEN_CLAW_HTTP_PORT"); val != "" {
		fmt.Sscanf(val, "%d", &cfg.Server.HTTPPort)
	}
	if val := os.Getenv("QWEN_CLAW_LOG_LEVEL"); val != "" {
		cfg.Logger.Level = val
	}
	if val := os.Getenv("QWEN_CLAW_DATABASE_URL"); val != "" {
		cfg.Database.URL = val
	}
	if val := os.Getenv("QWEN_CLAW_REDIS_URL"); val != "" {
		cfg.Redis.URL = val
	}
	if val := os.Getenv("QWEN_CLAW_JAEGER_ENDPOINT"); val != "" {
		cfg.Tracing.JaegerEndpoint = val
	}

	return cfg, nil
}

// mergeWithDefaults заполняет незаполненные поля значениями по умолчанию
func mergeWithDefaults(cfg *Config, defaults *Config) {
	if cfg.Server.GRPCPort == 0 {
		cfg.Server.GRPCPort = defaults.Server.GRPCPort
	}
	if cfg.Server.HTTPPort == 0 {
		cfg.Server.HTTPPort = defaults.Server.HTTPPort
	}
	if cfg.Logger.Level == "" {
		cfg.Logger.Level = defaults.Logger.Level
	}
	// ... можно добавить остальные поля
}
