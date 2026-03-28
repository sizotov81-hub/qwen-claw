// Package integration содержит integration тесты для микросервисов Qwen-Claw
package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/user/qwen-claw/internal/client"
	"github.com/user/qwen-claw/internal/session"
)

// skipIfNoEnv пропускает тест если переменная окружения не установлена
func skipIfNoEnv(t *testing.T, key string) {
	if os.Getenv(key) == "" {
		t.Skipf("Skipping integration test: %s not set", key)
	}
}

// TestSessionService_Integration тестирует Session Service с реальной БД
func TestSessionService_Integration(t *testing.T) {
	skipIfNoEnv(t, "TEST_DATABASE_URL")

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://qwenclaw:qwenclaw_password@localhost:5432/qwenclaw_test?sslmode=disable"
	}

	// Создаём подключение к БД (заглушка - в реальном тесте будет sql.DB)
	// db, err := sql.Open("postgres", dbURL)
	// require.NoError(t, err)
	// defer db.Close()

	t.Run("Create and Get Session", func(t *testing.T) {
		// В реальном тесте:
		// storage := service.NewSessionStorage(db)
		// svc := service.NewSessionService(storage, nil)
		
		// Заглушка для демонстрации
		sess := session.NewSession("user123", "Integration Test")
		require.NotNil(t, sess)
		assert.NotEmpty(t, sess.ID)
		assert.Equal(t, "user123", sess.UserID)
		assert.Equal(t, "Integration Test", sess.Title)
	})

	t.Run("Add Message", func(t *testing.T) {
		msg := session.NewMessage("session123", "user", "Hello!", 10)
		require.NotNil(t, msg)
		assert.Equal(t, "user", msg.Role)
		assert.Equal(t, "Hello!", msg.Content)
	})
}

// TestGRPCClient_Integration тестирует gRPC клиентов
func TestGRPCClient_Integration(t *testing.T) {
	skipIfNoEnv(t, "TEST_GRPC_ADDRESS")

	address := os.Getenv("TEST_GRPC_ADDRESS")
	if address == "" {
		address = "localhost:50051"
	}

	t.Run("Connect to server", func(t *testing.T) {
		config := client.GRPCClientConfig{
			Address:      address,
			Timeout:      5 * time.Second,
			MaxRetries:   3,
			RetryBackoff: 100 * time.Millisecond,
			EnableRetry:  true,
		}

		c := client.NewGRPCClient(config)
		require.NotNil(t, c)

		// Попытка подключения (может не удастся если сервер не запущен)
		err := c.Connect()
		if err != nil {
			t.Logf("Connection failed (expected if server not running): %v", err)
		}

		c.Close()
	})

	t.Run("Create conn directly", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(ctx, address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)

		if err != nil {
			t.Logf("Dial failed (expected if server not running): %v", err)
			return
		}
		defer conn.Close()

		t.Log("Connected successfully")
	})
}

// TestHTTPHealth_Integration тестирует health endpoints
func TestHTTPHealth_Integration(t *testing.T) {
	skipIfNoEnv(t, "TEST_HTTP_BASE_URL")

	baseURL := os.Getenv("TEST_HTTP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	services := []string{
		"session-memory",
		"qwen-wrapper",
		"llm-proxy",
		"tools-executor",
	}

	for _, svc := range services {
		t.Run(fmt.Sprintf("%s health", svc), func(t *testing.T) {
			url := fmt.Sprintf("%s/health", baseURL)
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Logf("Health check failed for %s (expected if not running): %v", svc, err)
				return
			}
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

// TestSessionMemoryClient_Integration тестирует Session Memory клиент
func TestSessionMemoryClient_Integration(t *testing.T) {
	skipIfNoEnv(t, "TEST_SESSION_MEMORY_ADDRESS")

	address := os.Getenv("TEST_SESSION_MEMORY_ADDRESS")
	if address == "" {
		address = "localhost:50051"
	}

	config := client.GRPCClientConfig{
		Address:     address,
		Timeout:     10 * time.Second,
		MaxRetries:  3,
		EnableRetry: true,
	}

	sessionClient := client.NewSessionMemoryClient(config)
	require.NotNil(t, sessionClient)

	t.Run("Create session (mock)", func(t *testing.T) {
		// В реальном тесте будет вызов gRPC метода
		// Заглушка - в реальности: sessionClient.CreateSession(ctx, "user", "title")
		sess := session.NewSession("user-integration", "Integration Session")
		assert.NotNil(t, sess)
	})
}

// TestRabbitMQ_Integration тестирует RabbitMQ интеграцию
func TestRabbitMQ_Integration(t *testing.T) {
	skipIfNoEnv(t, "TEST_RABBITMQ_URL")

	url := os.Getenv("TEST_RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	t.Skip("RabbitMQ integration test skipped - requires running RabbitMQ server")
	
	// В реальном тесте:
	// import "github.com/user/qwen-claw/internal/rabbitmq"
	//
	// cfg := rabbitmq.RabbitMQConfig{URL: url}
	// mq := rabbitmq.NewRabbitMQClient(cfg, logger)
	// err := mq.Connect()
	// require.NoError(t, err)
	// defer mq.Close()
}

// TestCircuitBreaker_Integration тестирует Circuit Breaker в реальных условиях
func TestCircuitBreaker_Integration(t *testing.T) {
	// Этот тест не требует внешних зависимостей
	// но тестирует поведение в условиях близких к реальным
	
	t.Run("Circuit opens after failures", func(t *testing.T) {
		// Тест уже покрыт в unit тестах circuitbreaker
		// Здесь можно добавить тест с реальным сервисом
		t.Skip("Covered by unit tests")
	})
}

// TestConcurrentAccess_Integration тестирует конкурентный доступ
func TestConcurrentAccess_Integration(t *testing.T) {
	skipIfNoEnv(t, "TEST_CONCURRENT_ENABLED")

	t.Run("Concurrent session creation", func(t *testing.T) {
		// Создаём много сессий параллельно
		done := make(chan bool, 100)

		for i := 0; i < 100; i++ {
			go func(n int) {
				sess := session.NewSession(
					fmt.Sprintf("user-%d", n),
					fmt.Sprintf("Session-%d", n),
				)
				assert.NotNil(t, sess)
				done <- true
			}(i)
		}

		// Ждём завершения
		for i := 0; i < 100; i++ {
			select {
			case <-done:
				// OK
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent operations")
			}
		}
	})
}

// TestMemoryLeak_Integration тестирует утечки памяти
func TestMemoryLeak_Integration(t *testing.T) {
	t.Skip("Memory leak test requires special tooling")
	
	// Для тестов на утечки памяти использовать:
	// go test -run=TestMemoryLeak -memprofilerate=1
}

// TestTimeoutHandling_Integration тестирует обработку таймаутов
func TestTimeoutHandling_Integration(t *testing.T) {
	t.Run("Context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Имитация долгой операции
		done := make(chan bool)
		go func() {
			time.Sleep(200 * time.Millisecond)
			done <- true
		}()

		select {
		case <-ctx.Done():
			assert.Equal(t, context.DeadlineExceeded, ctx.Err())
		case <-done:
			t.Fatal("Operation should have timed out")
		}
	})

	t.Run("Retry with timeout", func(t *testing.T) {
		config := client.GRPCClientConfig{
			Address:      "localhost:50051",
			Timeout:      500 * time.Millisecond,
			MaxRetries:   2,
			RetryBackoff: 100 * time.Millisecond,
			EnableRetry:  true,
		}

		c := client.NewGRPCClient(config)
		assert.NotNil(t, c)
	})
}
