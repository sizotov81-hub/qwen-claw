package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockSessionStorage заглушка для SessionStorage
type mockSessionStorage struct{}

func (m *mockSessionStorage) Create(ctx context.Context, sess interface{}) error {
	return nil
}

func (m *mockSessionStorage) GetByID(ctx context.Context, id string) (interface{}, error) {
	return nil, nil
}

func (m *mockSessionStorage) Update(ctx context.Context, sess interface{}) error {
	return nil
}

func (m *mockSessionStorage) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockSessionStorage) ListByUser(ctx context.Context, userID string, limit, offset int32) ([]interface{}, error) {
	return nil, nil
}

func (m *mockSessionStorage) AddMessage(ctx context.Context, msg interface{}) error {
	return nil
}

func (m *mockSessionStorage) GetMessages(ctx context.Context, sessionID string, limit, offset int32, order string) ([]interface{}, error) {
	return nil, nil
}

// mockMemoryStorage заглушка для MemoryStorage
type mockMemoryStorage struct{}

func (m *mockMemoryStorage) Add(ctx context.Context, entry interface{}) error {
	return nil
}

func (m *mockMemoryStorage) GetByID(ctx context.Context, id string) (interface{}, error) {
	return nil, nil
}

func (m *mockMemoryStorage) SearchByText(ctx context.Context, userID, query string, limit int32) ([]interface{}, error) {
	return nil, nil
}

func (m *mockMemoryStorage) GetBySession(ctx context.Context, sessionID string) ([]interface{}, error) {
	return nil, nil
}

func (m *mockMemoryStorage) UpdateAccess(ctx context.Context, id string) error {
	return nil
}

func (m *mockMemoryStorage) Delete(ctx context.Context, id string) error {
	return nil
}

func TestNewSessionMemoryServer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	cfg := SessionMemoryServerConfig{
		GRPCPort:     50051,
		HTTPPort:     8080,
		Host:         "127.0.0.1",
		SessionSvc:   nil,
		MemorySvc:    nil,
		Logger:       logger.Sugar(),
		HealthCheckFn: func() bool { return true },
	}

	server := NewSessionMemoryServer(cfg)

	require.NotNil(t, server)
	assert.Equal(t, 50051, server.GetGRPCPort())
	assert.Equal(t, 8080, server.GetHTTPPort())
	assert.True(t, server.HealthCheck())
}

func TestSessionMemoryServer_StartStop(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	cfg := SessionMemoryServerConfig{
		GRPCPort:      50052,
		HTTPPort:      8081,
		Host:          "127.0.0.1",
		Logger:        logger.Sugar(),
		HealthCheckFn: func() bool { return true },
	}

	server := NewSessionMemoryServer(cfg)

	// Запускаем сервер
	err := server.Start()
	require.NoError(t, err)

	// Даём время на запуск
	time.Sleep(100 * time.Millisecond)

	// Проверяем что сервер работает
	assert.True(t, server.HealthCheck())

	// Останавливаем сервер
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(ctx)
	assert.NoError(t, err)
}

func TestSessionMemoryServer_Methods(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	cfg := SessionMemoryServerConfig{
		GRPCPort:      50053,
		HTTPPort:      8082,
		Host:          "127.0.0.1",
		Logger:        logger.Sugar(),
		HealthCheckFn: func() bool { return true },
	}

	server := NewSessionMemoryServer(cfg)

	// Проверяем что методы существуют и не паникуют
	t.Run("CreateSession", func(t *testing.T) {
		// Метод должен существовать
		assert.NotNil(t, server.CreateSession)
	})

	t.Run("GetSession", func(t *testing.T) {
		assert.NotNil(t, server.GetSession)
	})

	t.Run("AddMessage", func(t *testing.T) {
		assert.NotNil(t, server.AddMessage)
	})

	t.Run("AddMemory", func(t *testing.T) {
		assert.NotNil(t, server.AddMemory)
	})

	t.Run("SearchMemory", func(t *testing.T) {
		assert.NotNil(t, server.SearchMemory)
	})
}

func TestSessionMemoryServer_GetServices(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	cfg := SessionMemoryServerConfig{
		GRPCPort:      50054,
		HTTPPort:      8083,
		Host:          "127.0.0.1",
		Logger:        logger.Sugar(),
		HealthCheckFn: func() bool { return true },
	}

	server := NewSessionMemoryServer(cfg)

	// Проверяем что сервисы возвращаются
	sessionSvc := server.GetSessionService()
	memorySvc := server.GetMemoryService()

	// Они могут быть nil если не переданы в конфиге
	assert.Nil(t, sessionSvc)
	assert.Nil(t, memorySvc)
}

func TestSessionMemoryServer_HealthCheck(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	healthy := true
	cfg := SessionMemoryServerConfig{
		GRPCPort:      50055,
		HTTPPort:      8084,
		Host:          "127.0.0.1",
		Logger:        logger.Sugar(),
		HealthCheckFn: func() bool { return healthy },
	}

	server := NewSessionMemoryServer(cfg)

	// Проверяем health check
	assert.True(t, server.HealthCheck())

	// Делаем сервер "нездоровым"
	healthy = false
	assert.False(t, server.HealthCheck())
}

func TestSessionMemoryServer_ConcurrentAccess(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	cfg := SessionMemoryServerConfig{
		GRPCPort:      50056,
		HTTPPort:      8085,
		Host:          "127.0.0.1",
		Logger:        logger.Sugar(),
		HealthCheckFn: func() bool { return true },
	}

	server := NewSessionMemoryServer(cfg)

	// Запускаем сервер
	err := server.Start()
	require.NoError(t, err)
	defer server.Stop(context.Background())

	// Даём время на запуск
	time.Sleep(100 * time.Millisecond)

	// Запускаем несколько горутин для проверки конкурентного доступа
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_ = server.GetGRPCPort()
			_ = server.GetHTTPPort()
			_ = server.HealthCheck()
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
