package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	require.NotNil(t, cfg)

	// Server
	assert.Equal(t, 50051, cfg.Server.GRPCPort)
	assert.Equal(t, 8080, cfg.Server.HTTPPort)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)

	// Logger
	assert.Equal(t, "info", cfg.Logger.Level)
	assert.Equal(t, "json", cfg.Logger.Format)

	// Tracing
	assert.True(t, cfg.Tracing.Enabled)
	assert.Equal(t, "qwen-claw", cfg.Tracing.ServiceName)
	assert.Equal(t, 1.0, cfg.Tracing.SamplingRatio)

	// Database
	assert.Contains(t, cfg.Database.URL, "postgres://")
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)

	// Circuit Breaker
	assert.Equal(t, 5, cfg.CircuitBreaker.FailureThreshold)
	assert.Equal(t, 30*time.Second, cfg.CircuitBreaker.Timeout)

	// Rate Limit
	assert.True(t, cfg.RateLimit.Enabled)
	assert.Equal(t, 100, cfg.RateLimit.RequestsPerSecond)
}

func TestLoad(t *testing.T) {
	// Создаём временный файл с конфигурацией
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  grpc_port: 50099
  http_port: 8099
  host: "127.0.0.1"

logger:
  level: "debug"
  format: "console"

database:
  url: "postgres://test:test@localhost:5432/testdb"
  max_open_conns: 10
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Загружаем конфигурацию
	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Проверяем что значения из файла загрузились
	assert.Equal(t, 50099, cfg.Server.GRPCPort)
	assert.Equal(t, 8099, cfg.Server.HTTPPort)
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, "console", cfg.Logger.Format)
	assert.Contains(t, cfg.Database.URL, "testdb")
	assert.Equal(t, 10, cfg.Database.MaxOpenConns)

	// Проверяем что значения по умолчанию тоже применились
	assert.Equal(t, "127.0.0.1", cfg.Server.Host) // из файла
	// Tracing.ServiceName берётся из defaults если не задан в файле
}

func TestLoadNonExistent(t *testing.T) {
	cfg, err := Load("/nonexistent/config.yaml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoadFromEnv(t *testing.T) {
	// Устанавливаем переменные окружения
	os.Setenv("QWEN_CLAW_GRPC_PORT", "50077")
	os.Setenv("QWEN_CLAW_HTTP_PORT", "8077")
	os.Setenv("QWEN_CLAW_LOG_LEVEL", "debug")
	os.Setenv("QWEN_CLAW_DATABASE_URL", "postgres://env:env@localhost:5432/envdb")
	os.Setenv("QWEN_CLAW_REDIS_URL", "redis://env:6379")
	os.Setenv("QWEN_CLAW_JAEGER_ENDPOINT", "env-jaeger:4317")

	defer func() {
		os.Unsetenv("QWEN_CLAW_GRPC_PORT")
		os.Unsetenv("QWEN_CLAW_HTTP_PORT")
		os.Unsetenv("QWEN_CLAW_LOG_LEVEL")
		os.Unsetenv("QWEN_CLAW_DATABASE_URL")
		os.Unsetenv("QWEN_CLAW_REDIS_URL")
		os.Unsetenv("QWEN_CLAW_JAEGER_ENDPOINT")
	}()

	cfg, err := LoadFromEnv("test-service")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Проверяем что переменные окружения применились
	assert.Equal(t, 50077, cfg.Server.GRPCPort)
	assert.Equal(t, 8077, cfg.Server.HTTPPort)
	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Contains(t, cfg.Database.URL, "envdb")
	assert.Contains(t, cfg.Redis.URL, "env:6379")
	assert.Equal(t, "env-jaeger:4317", cfg.Tracing.JaegerEndpoint)
	assert.Equal(t, "test-service", cfg.Tracing.ServiceName)
}

func TestMergeWithDefaults(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			GRPCPort: 50000, // задано явно
			// HTTPPort не задан - должен взяться из defaults
		},
	}

	defaults := Default()
	mergeWithDefaults(cfg, defaults)

	assert.Equal(t, 50000, cfg.Server.GRPCPort)   // осталось как было
	assert.Equal(t, 8080, cfg.Server.HTTPPort)    // взято из defaults
	assert.Equal(t, "info", cfg.Logger.Level)     // взято из defaults
}

func TestRabbitMQConfig(t *testing.T) {
	cfg := Default()

	// Проверяем что RabbitMQ конфигурация не пустая
	assert.NotEmpty(t, cfg.RabbitMQ.URL)

	// Добавляем exchange и queue
	cfg.RabbitMQ.Exchanges = []ExchangeConfig{
		{Name: "qwenclaw.events", Type: "topic"},
	}
	cfg.RabbitMQ.Queues = []QueueConfig{
		{Name: "llm-requests", Durable: true},
	}

	require.Len(t, cfg.RabbitMQ.Exchanges, 1)
	assert.Equal(t, "qwenclaw.events", cfg.RabbitMQ.Exchanges[0].Name)
	assert.Equal(t, "topic", cfg.RabbitMQ.Exchanges[0].Type)

	require.Len(t, cfg.RabbitMQ.Queues, 1)
	assert.Equal(t, "llm-requests", cfg.RabbitMQ.Queues[0].Name)
	assert.True(t, cfg.RabbitMQ.Queues[0].Durable)
}

func TestGRPCClientsConfig(t *testing.T) {
	cfg := Default()

	// Инициализируем gRPC клиентов если они пустые
	if cfg.GRPCClients.SessionMemory.Address == "" {
		cfg.GRPCClients.SessionMemory = GRPCClientConfig{
			Address: "session-memory:50051",
			Timeout: 30 * time.Second,
		}
	}
	if cfg.GRPCClients.QwenWrapper.Address == "" {
		cfg.GRPCClients.QwenWrapper = GRPCClientConfig{
			Address: "qwen-wrapper:50052",
			Timeout: 300 * time.Second,
		}
	}
	if cfg.GRPCClients.LLMProxy.Address == "" {
		cfg.GRPCClients.LLMProxy = GRPCClientConfig{
			Address: "llm-proxy:50053",
			Timeout: 120 * time.Second,
		}
	}
	if cfg.GRPCClients.ToolsExecutor.Address == "" {
		cfg.GRPCClients.ToolsExecutor = GRPCClientConfig{
			Address: "tools-executor:50054",
			Timeout: 60 * time.Second,
		}
	}

	// Проверяем что gRPC клиенты сконфигурированы
	assert.NotEmpty(t, cfg.GRPCClients.SessionMemory.Address)
	assert.NotEmpty(t, cfg.GRPCClients.QwenWrapper.Address)
	assert.NotEmpty(t, cfg.GRPCClients.LLMProxy.Address)
	assert.NotEmpty(t, cfg.GRPCClients.ToolsExecutor.Address)

	// Проверяем таймауты
	assert.Greater(t, cfg.GRPCClients.SessionMemory.Timeout, time.Duration(0))
	assert.Greater(t, cfg.GRPCClients.QwenWrapper.Timeout, time.Duration(0))
}

func TestSecurityConfig(t *testing.T) {
	cfg := Default()

	// Инициализируем Security если он пустой
	if cfg.Security.JWTSecret == "" {
		cfg.Security.JWTSecret = "change-me-in-production"
	}
	if cfg.Security.JWTTTL == 0 {
		cfg.Security.JWTTTL = 24 * time.Hour
	}

	// Проверяем что JWT секрет задан
	assert.NotEmpty(t, cfg.Security.JWTSecret)
	assert.NotEqual(t, "", cfg.Security.JWTSecret)

	// Проверяем TTL
	assert.Greater(t, cfg.Security.JWTTTL, time.Duration(0))
}
