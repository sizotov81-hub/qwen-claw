// Package grpc предоставляет gRPC сервер для Session Memory Service
package grpc

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	sm "github.com/user/qwen-claw/internal/session"
	"github.com/user/qwen-claw/internal/server"
	"github.com/user/qwen-claw/internal/session/service"
)

// SessionMemoryServer gRPC сервер для Session Memory Service
type SessionMemoryServer struct {
	server          *server.GRPCServer
	sessionService  *service.SessionService
	memoryService   *service.MemoryService
	logger          *zap.SugaredLogger
	healthCheckFunc func() bool
}

// SessionMemoryServerConfig конфигурация сервера
type SessionMemoryServerConfig struct {
	GRPCPort      int
	HTTPPort      int
	Host          string
	SessionSvc    *service.SessionService
	MemorySvc     *service.MemoryService
	Logger        *zap.SugaredLogger
	HealthCheckFn func() bool
}

// NewSessionMemoryServer создаёт новый gRPC сервер
func NewSessionMemoryServer(cfg SessionMemoryServerConfig) *SessionMemoryServer {
	// Создаём базовый gRPC сервер
	grpcServer := server.NewGRPCServer(
		cfg.GRPCPort,
		cfg.HTTPPort,
		cfg.Host,
		server.WithLogger(cfg.Logger),
		server.WithHealthCheck(cfg.HealthCheckFn),
	)

	s := &SessionMemoryServer{
		server:          grpcServer,
		sessionService:  cfg.SessionSvc,
		memoryService:   cfg.MemorySvc,
		logger:          cfg.Logger,
		healthCheckFunc: cfg.HealthCheckFn,
	}

	// Регистрируем сервис (заглушка - в продакшене будет регистрация из proto)
	// pb.RegisterSessionMemoryServer(grpcServer.GetGRPCServer(), s)

	return s
}

// Start запускает сервер
func (s *SessionMemoryServer) Start() error {
	s.logger.Info("Starting Session Memory gRPC server...")
	
	if err := s.server.Start(); err != nil {
		return fmt.Errorf("start grpc server: %w", err)
	}

	s.logger.Infof("Session Memory server listening on gRPC port")
	return nil
}

// Stop останавливает сервер
func (s *SessionMemoryServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Session Memory gRPC server...")
	
	if err := s.server.Stop(ctx); err != nil {
		return fmt.Errorf("stop grpc server: %w", err)
	}

	return nil
}

// GetGRPCPort возвращает gRPC порт
func (s *SessionMemoryServer) GetGRPCPort() int {
	return s.server.GetGRPCPort()
}

// GetHTTPPort возвращает HTTP порт
func (s *SessionMemoryServer) GetHTTPPort() int {
	return s.server.GetHTTPPort()
}

// GetSessionService возвращает сервис сессий
func (s *SessionMemoryServer) GetSessionService() *service.SessionService {
	return s.sessionService
}

// GetMemoryService возвращает сервис памяти
func (s *SessionMemoryServer) GetMemoryService() *service.MemoryService {
	return s.memoryService
}

// HealthCheck проверяет здоровье сервиса
func (s *SessionMemoryServer) HealthCheck() bool {
	if s.healthCheckFunc != nil {
		return s.healthCheckFunc()
	}
	return true
}

// RegisterService регистрирует gRPC сервис
// В продакшене будет использоваться для регистрации сгенерированного сервиса
func (s *SessionMemoryServer) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.server.RegisterService(desc, impl)
}

// GetListener возвращает listener для gRPC сервера
func (s *SessionMemoryServer) GetListener(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

// Session Memory Service методы (заглушки для методов из proto)

// CreateSession создаёт новую сессию
// В продакшене: func (s *SessionMemoryServer) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.Session, error)
func (s *SessionMemoryServer) CreateSession(ctx context.Context, userID, title string) (sessionID string, err error) {
	sess, err := s.sessionService.CreateSession(ctx, userID, title)
	if err != nil {
		return "", err
	}
	return sess.ID, nil
}

// GetSession получает сессию
// В продакшене: func (s *SessionMemoryServer) GetSession(ctx context.Context, req *pb.SessionID) (*pb.Session, error)
func (s *SessionMemoryServer) GetSession(ctx context.Context, sessionID string) (userID, title string, err error) {
	sess, err := s.sessionService.GetSession(ctx, sessionID)
	if err != nil {
		return "", "", err
	}
	return sess.UserID, sess.Title, nil
}

// AddMessage добавляет сообщение в сессию
// В продакшене: func (s *SessionMemoryServer) AddMessage(ctx context.Context, req *pb.AddMessageRequest) (*pb.Message, error)
func (s *SessionMemoryServer) AddMessage(ctx context.Context, sessionID, role, content string, tokens int32) (messageID string, err error) {
	msg, err := s.sessionService.AddMessage(ctx, sessionID, role, content, tokens)
	if err != nil {
		return "", err
	}
	return msg.ID, nil
}

// AddMemory добавляет запись в память
// В продакшене: func (s *SessionMemoryServer) AddMemory(ctx context.Context, req *pb.MemoryEntry) (*pb.MemoryEntry, error)
func (s *SessionMemoryServer) AddMemory(ctx context.Context, userID, sessionID, content string, memType string) (entryID string, err error) {
	// В продакшене будет преобразование из pb.MemoryEntry
	entry := &sm.MemoryEntry{
		UserID:    userID,
		SessionID: sessionID,
		Content:   content,
		Type:      sm.MemoryType(memType),
	}
	
	result, err := s.memoryService.AddMemory(ctx, entry)
	if err != nil {
		return "", err
	}
	return result.ID, nil
}

// SearchMemory ищет записи в памяти
// В продакшене: func (s *SessionMemoryServer) SearchMemory(ctx context.Context, req *pb.SearchMemoryRequest) (*pb.MemoryList, error)
func (s *SessionMemoryServer) SearchMemory(ctx context.Context, userID, query string, limit int32) (results []string, err error) {
	entries, err := s.memoryService.SearchMemory(ctx, userID, query, limit)
	if err != nil {
		return nil, err
	}
	
	results = make([]string, len(entries))
	for i, e := range entries {
		results[i] = e.Content
	}
	return results, nil
}
