package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGRPCClient(t *testing.T) {
	config := GRPCClientConfig{
		Address:      "localhost:50051",
		Timeout:      30 * time.Second,
		MaxRetries:   3,
		RetryBackoff: 100 * time.Millisecond,
		EnableRetry:  true,
	}

	client := NewGRPCClient(config)

	require.NotNil(t, client)
	assert.Equal(t, config.Address, client.config.Address)
	assert.Equal(t, config.Timeout, client.config.Timeout)
	assert.Equal(t, config.MaxRetries, client.config.MaxRetries)
	assert.False(t, client.IsConnected())
}

func TestGRPCClient_ConnectClose(t *testing.T) {
	config := GRPCClientConfig{
		Address:      "localhost:50052",
		Timeout:      5 * time.Second,
		MaxRetries:   1,
		RetryBackoff: 50 * time.Millisecond,
		EnableRetry:  false,
	}

	client := NewGRPCClient(config)

	// Connect должен вернуть ошибку (сервер не запущен)
	err := client.Connect()
	assert.Error(t, err)

	// Close не должен паниковать
	err = client.Close()
	assert.NoError(t, err)
}

func TestGRPCClient_ExecuteWithTimeout(t *testing.T) {
	config := GRPCClientConfig{
		Address:     "localhost:50053",
		Timeout:     100 * time.Millisecond,
		MaxRetries:  0,
		EnableRetry: false,
	}

	client := NewGRPCClient(config)

	// Тест с быстрым выполнением
	err := client.ExecuteWithTimeout(context.Background(), func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// Тест с таймаутом - ожидаем ошибку контекста
	err = client.ExecuteWithTimeout(context.Background(), func(ctx context.Context) error {
		<-ctx.Done() // Ждём пока контекст истечёт
		return ctx.Err()
	})
	assert.Error(t, err)
}

func TestGRPCClient_ExecuteWithRetry(t *testing.T) {
	config := GRPCClientConfig{
		Address:      "localhost:50054",
		Timeout:      1 * time.Second,
		MaxRetries:   3,
		RetryBackoff: 10 * time.Millisecond,
		EnableRetry:  true,
	}

	client := NewGRPCClient(config)

	// Тест с успешным выполнением с первой попытки
	attempts := 0
	err := client.ExecuteWithRetry(context.Background(), func(ctx context.Context) error {
		attempts++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)

	// Тест с успешным выполнением после retry
	attempts = 0
	err = client.ExecuteWithRetry(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return context.DeadlineExceeded // Retry'аемая ошибка
		}
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)

	// Тест с неудачным выполнением (все retry исчерпаны)
	attempts = 0
	err = client.ExecuteWithRetry(context.Background(), func(ctx context.Context) error {
		attempts++
		return context.DeadlineExceeded
	})
	assert.Error(t, err)
	assert.Equal(t, 4, attempts) // 1 + 3 retry
}

func TestSessionMemoryClient(t *testing.T) {
	config := GRPCClientConfig{
		Address:     "localhost:50055",
		Timeout:     5 * time.Second,
		MaxRetries:  1,
		EnableRetry: false,
	}

	client := NewSessionMemoryClient(config)
	require.NotNil(t, client)

	// Тест CreateSession (заглушка)
	ctx := context.Background()
	sessionID, err := client.CreateSession(ctx, "user123", "Test Session")
	assert.NoError(t, err)
	assert.Contains(t, sessionID, "session-user123")

	// Тест GetSession (заглушка)
	userID, title, err := client.GetSession(ctx, "test-id")
	assert.NoError(t, err)
	assert.Contains(t, userID, "user-test-id")
	assert.Contains(t, title, "Session test-id")
}

func TestLLMProxyClient(t *testing.T) {
	config := GRPCClientConfig{
		Address:     "localhost:50056",
		Timeout:     30 * time.Second,
		MaxRetries:  2,
		EnableRetry: true,
	}

	client := NewLLMProxyClient(config)
	require.NotNil(t, client)

	// Тест ChatCompletion (заглушка)
	ctx := context.Background()
	response, err := client.ChatCompletion(ctx, "gpt-4", "Hello!")
	assert.NoError(t, err)
	assert.Contains(t, response, "Response from gpt-4")
}

func TestToolsExecutorClient(t *testing.T) {
	config := GRPCClientConfig{
		Address:     "localhost:50057",
		Timeout:     60 * time.Second,
		MaxRetries:  1,
		EnableRetry: false,
	}

	client := NewToolsExecutorClient(config)
	require.NotNil(t, client)

	// Тест Execute (заглушка)
	ctx := context.Background()
	output, err := client.Execute(ctx, "bash", map[string]string{"cmd": "ls -la"})
	assert.NoError(t, err)
	assert.Contains(t, output, "Executed bash")
}

func TestQwenWrapperClient(t *testing.T) {
	config := GRPCClientConfig{
		Address:     "localhost:50058",
		Timeout:     300 * time.Second,
		MaxRetries:  2,
		EnableRetry: true,
	}

	client := NewQwenWrapperClient(config)
	require.NotNil(t, client)

	// Тест ExecuteQuery (заглушка)
	ctx := context.Background()
	response, err := client.ExecuteQuery(ctx, "What is Go?")
	assert.NoError(t, err)
	assert.Contains(t, response, "Qwen response")
}

func TestGRPCClient_ConcurrentAccess(t *testing.T) {
	config := GRPCClientConfig{
		Address:     "localhost:50059",
		Timeout:     5 * time.Second,
		MaxRetries:  0,
		EnableRetry: false,
	}

	client := NewGRPCClient(config)

	// Запускаем несколько горутин
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = client.IsConnected()
			_ = client.GetConn()
			done <- true
		}()
	}

	// Ждём завершения
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// OK
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for goroutines")
		}
	}
}
