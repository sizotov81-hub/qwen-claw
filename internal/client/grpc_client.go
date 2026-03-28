// Package client предоставляет gRPC клиенты для микросервисов Qwen-Claw
package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCClientConfig конфигурация gRPC клиента
type GRPCClientConfig struct {
	Address       string
	Timeout       time.Duration
	MaxRetries    int
	RetryBackoff  time.Duration
	EnableRetry   bool
}

// GRPCClient базовый gRPC клиент
type GRPCClient struct {
	mu       sync.RWMutex
	conn     *grpc.ClientConn
	config   GRPCClientConfig
	cancel   context.CancelFunc
	ctx      context.Context
}

// NewGRPCClient создаёт новый gRPC клиент
func NewGRPCClient(config GRPCClientConfig) *GRPCClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &GRPCClient{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Connect подключается к серверу
func (c *GRPCClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil // Уже подключен
	}

	opts := []grpc.DialOption{
		grpc.WithInsecure(), // В продакшене использовать grpc.WithTransportCredentials()
		grpc.WithBlock(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, c.config.Address, opts...)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.conn = conn
	return nil
}

// Close закрывает соединение
func (c *GRPCClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("close failed: %w", err)
		}
		c.conn = nil
	}

	return nil
}

// GetConn возвращает соединение
func (c *GRPCClient) GetConn() *grpc.ClientConn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// ExecuteWithRetry выполняет запрос с retry логикой
func (c *GRPCClient) ExecuteWithRetry(ctx context.Context, fn func(ctx context.Context) error) error {
	if !c.config.EnableRetry {
		return fn(ctx)
	}

	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Создаём контекст с таймаутом
		reqCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
		defer cancel()

		lastErr = fn(reqCtx)
		if lastErr == nil {
			return nil // Успех
		}

		// Проверяем тип ошибки
		st, ok := status.FromError(lastErr)
		if ok {
			// Не retry'им ошибки клиента
			if st.Code() == codes.InvalidArgument ||
				st.Code() == codes.NotFound ||
				st.Code() == codes.PermissionDenied ||
				st.Code() == codes.Unauthenticated {
				return lastErr
			}
		}

		// Ждём перед retry
		if attempt < c.config.MaxRetries {
			select {
			case <-time.After(c.config.RetryBackoff * time.Duration(attempt+1)):
				// Продолжаем
			case <-ctx.Done():
				return ctx.Err()
			case <-c.ctx.Done():
				return c.ctx.Err()
			}
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// ExecuteWithTimeout выполняет запрос с таймаутом
func (c *GRPCClient) ExecuteWithTimeout(ctx context.Context, fn func(ctx context.Context) error) error {
	reqCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	return fn(reqCtx)
}

// IsConnected проверяет подключение
func (c *GRPCClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil
}

// SessionMemoryClient клиент для Session Memory Service
type SessionMemoryClient struct {
	*GRPCClient
}

// NewSessionMemoryClient создаёт клиент для Session Memory Service
func NewSessionMemoryClient(config GRPCClientConfig) *SessionMemoryClient {
	return &SessionMemoryClient{
		GRPCClient: NewGRPCClient(config),
	}
}

// CreateSession создаёт сессию
func (c *SessionMemoryClient) CreateSession(ctx context.Context, userID, title string) (string, error) {
	var sessionID string
	err := c.ExecuteWithRetry(ctx, func(reqCtx context.Context) error {
		return c.ExecuteWithTimeout(reqCtx, func(ctx context.Context) error {
			// В продакшене: pb.NewSessionMemoryClient(c.conn).CreateSession(ctx, &pb.CreateSessionRequest{...})
			// Заглушка для демонстрации
			sessionID = "session-" + userID
			return nil
		})
	})
	return sessionID, err
}

// GetSession получает сессию
func (c *SessionMemoryClient) GetSession(ctx context.Context, sessionID string) (string, string, error) {
	var userID, title string
	err := c.ExecuteWithRetry(ctx, func(reqCtx context.Context) error {
		return c.ExecuteWithTimeout(reqCtx, func(ctx context.Context) error {
			// В продакшене: pb.NewSessionMemoryClient(c.conn).GetSession(ctx, &pb.SessionID{Id: sessionID})
			userID = "user-" + sessionID
			title = "Session " + sessionID
			return nil
		})
	})
	return userID, title, err
}

// LLMProxyClient клиент для LLM Proxy Service
type LLMProxyClient struct {
	*GRPCClient
}

// NewLLMProxyClient создаёт клиент для LLM Proxy Service
func NewLLMProxyClient(config GRPCClientConfig) *LLMProxyClient {
	return &LLMProxyClient{
		GRPCClient: NewGRPCClient(config),
	}
}

// ChatCompletion отправляет запрос к LLM
func (c *LLMProxyClient) ChatCompletion(ctx context.Context, model, message string) (string, error) {
	var response string
	err := c.ExecuteWithRetry(ctx, func(reqCtx context.Context) error {
		return c.ExecuteWithTimeout(reqCtx, func(ctx context.Context) error {
			// В продакшене: pb.NewLLMProxyClient(c.conn).ChatCompletion(ctx, &pb.ChatRequest{...})
			response = "Response from " + model + ": " + message
			return nil
		})
	})
	return response, err
}

// ToolsExecutorClient клиент для Tools Executor Service
type ToolsExecutorClient struct {
	*GRPCClient
}

// NewToolsExecutorClient создаёт клиент для Tools Executor Service
func NewToolsExecutorClient(config GRPCClientConfig) *ToolsExecutorClient {
	return &ToolsExecutorClient{
		GRPCClient: NewGRPCClient(config),
	}
}

// Execute выполняет инструмент
func (c *ToolsExecutorClient) Execute(ctx context.Context, toolName string, args map[string]string) (string, error) {
	var output string
	err := c.ExecuteWithRetry(ctx, func(reqCtx context.Context) error {
		return c.ExecuteWithTimeout(reqCtx, func(ctx context.Context) error {
			// В продакшене: pb.NewToolsExecutorClient(c.conn).Execute(ctx, &pb.ExecuteRequest{...})
			output = "Executed " + toolName
			return nil
		})
	})
	return output, err
}

// QwenWrapperClient клиент для Qwen Wrapper Service
type QwenWrapperClient struct {
	*GRPCClient
}

// NewQwenWrapperClient создаёт клиент для Qwen Wrapper Service
func NewQwenWrapperClient(config GRPCClientConfig) *QwenWrapperClient {
	return &QwenWrapperClient{
		GRPCClient: NewGRPCClient(config),
	}
}

// ExecuteQuery выполняет запрос к Qwen
func (c *QwenWrapperClient) ExecuteQuery(ctx context.Context, query string) (string, error) {
	var response string
	err := c.ExecuteWithRetry(ctx, func(reqCtx context.Context) error {
		return c.ExecuteWithTimeout(reqCtx, func(ctx context.Context) error {
			// В продакшене: pb.NewQwenWrapperClient(c.conn).ExecuteQuery(ctx, &pb.QueryRequest{...})
			response = "Qwen response: " + query
			return nil
		})
	})
	return response, err
}
