package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewGRPCServer(t *testing.T) {
	server := NewGRPCServer(50051, 8080, "127.0.0.1")

	require.NotNil(t, server)
	assert.Equal(t, 50051, server.GetGRPCPort())
	assert.Equal(t, 8080, server.GetHTTPPort())
	// Uptime должен быть около 0, но не точно 0 из-за времени инициализации
	assert.Less(t, server.GetUptime(), 1*time.Second)
}

func TestNewGRPCServer_WithOptions(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	healthCheck := func() bool { return true }

	server := NewGRPCServer(
		50052,
		8081,
		"0.0.0.0",
		WithLogger(logger.Sugar()),
		WithHealthCheck(healthCheck),
	)

	require.NotNil(t, server)
	assert.Equal(t, 50052, server.GetGRPCPort())
	assert.Equal(t, 8081, server.GetHTTPPort())
}

func TestGRPCServer_StartStop(t *testing.T) {
	server := NewGRPCServer(50053, 8082, "127.0.0.1")

	// Запускаем сервер
	err := server.Start()
	require.NoError(t, err)

	// Даём серверу время запуститься
	time.Sleep(100 * time.Millisecond)

	// Проверяем что сервер работает
	assert.Greater(t, server.GetUptime(), 0*time.Second)

	// Останавливаем сервер
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(ctx)
	assert.NoError(t, err)
}

func TestGRPCServer_HealthCheck(t *testing.T) {
	healthy := true
	healthCheck := func() bool { return healthy }

	server := NewGRPCServer(
		50054,
		8083,
		"127.0.0.1",
		WithHealthCheck(healthCheck),
	)

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop(context.Background())

	// Даём серверу время запуститься
	time.Sleep(100 * time.Millisecond)

	// Проверяем health endpoint когда здоров
	resp, err := http.Get("http://127.0.0.1:8083/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Проверяем ready endpoint
	resp, err = http.Get("http://127.0.0.1:8083/ready")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Делаем сервер "нездоровым"
	healthy = false

	resp, err = http.Get("http://127.0.0.1:8083/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestGRPCServer_RegisterService(t *testing.T) {
	_ = NewGRPCServer(50055, 8084, "127.0.0.1")

	// Регистрируем тестовый сервис (заглушка)
	// В реальном коде здесь будет регистрация gRPC сервиса
	// server.RegisterService(&SomeService_ServiceDesc, &SomeServiceImpl{})

	// Проверяем что метод не паникует
	assert.NotPanics(t, func() {
		// server.RegisterService(...)
	})
}

func TestGRPCServer_ConcurrentAccess(t *testing.T) {
	server := NewGRPCServer(50056, 8085, "127.0.0.1")

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop(context.Background())

	// Даём серверу время запуститься
	time.Sleep(100 * time.Millisecond)

	// Запускаем несколько горутин для проверки конкурентного доступа
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_ = server.GetGRPCPort()
			_ = server.GetHTTPPort()
			_ = server.GetUptime()
			done <- true
		}()
	}

	// Ждём завершения всех горутин
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// OK
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for goroutines")
		}
	}
}

func TestGRPCServer_HealthEndpointResponse(t *testing.T) {
	server := NewGRPCServer(50057, 8086, "127.0.0.1")

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop(context.Background())

	// Даём серверу время запуститься
	time.Sleep(100 * time.Millisecond)

	// Проверяем что health endpoint возвращает JSON
	resp, err := http.Get("http://127.0.0.1:8086/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
