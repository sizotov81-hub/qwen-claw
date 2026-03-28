// Package server предоставляет базовый gRPC сервер для микросервисов Qwen-Claw
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// GRPCServer базовый gRPC сервер
type GRPCServer struct {
	mu           sync.RWMutex
	server       *grpc.Server
	httpServer   *http.Server
	grpcPort     int
	httpPort     int
	host         string
	logger       *zap.SugaredLogger
	healthCheck  func() bool
	startedAt    time.Time
}

// ServerOption опция сервера
type ServerOption func(*GRPCServer)

// WithLogger устанавливает логгер
func WithLogger(logger *zap.SugaredLogger) ServerOption {
	return func(s *GRPCServer) {
		s.logger = logger
	}
}

// WithHealthCheck устанавливает функцию health check
func WithHealthCheck(fn func() bool) ServerOption {
	return func(s *GRPCServer) {
		s.healthCheck = fn
	}
}

// NewGRPCServer создаёт новый gRPC сервер
func NewGRPCServer(grpcPort, httpPort int, host string, opts ...ServerOption) *GRPCServer {
	s := &GRPCServer{
		grpcPort:  grpcPort,
		httpPort:  httpPort,
		host:      host,
		startedAt: time.Now(),
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.logger == nil {
		// Создаём дефолтный логгер
		s.logger = zap.NewNop().Sugar()
	}

	// Создаём gRPC сервер
	s.server = grpc.NewServer()

	// Создаём HTTP сервер для health/metrics
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, httpPort),
		Handler: mux,
	}

	return s
}

// RegisterService регистрирует gRPC сервис
func (s *GRPCServer) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.server.RegisterService(desc, impl)
}

// Start запускает сервер
func (s *GRPCServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Запускаем HTTP сервер для health/metrics
	httpAddr := fmt.Sprintf("%s:%d", s.host, s.httpPort)
	httpListener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", httpAddr, err)
	}

	go func() {
		s.logger.Infof("HTTP server listening on %s", httpAddr)
		if err := s.httpServer.Serve(httpListener); err != http.ErrServerClosed {
			s.logger.Errorf("HTTP server error: %v", err)
		}
	}()

	// Запускаем gRPC сервер
	grpcAddr := fmt.Sprintf("%s:%d", s.host, s.grpcPort)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", grpcAddr, err)
	}

	s.logger.Infof("gRPC server listening on %s", grpcAddr)
	go func() {
		if err := s.server.Serve(grpcListener); err != nil {
			s.logger.Errorf("gRPC server error: %v", err)
		}
	}()

	return nil
}

// Stop останавливает сервер
func (s *GRPCServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Graceful shutdown для gRPC
	go func() {
		s.server.GracefulStop()
	}()

	// Shutdown HTTP сервера
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("http server shutdown error: %w", err)
	}

	s.logger.Info("Server stopped")
	return nil
}

// handleHealth обрабатывает health check запросы
func (s *GRPCServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	code := http.StatusOK

	if s.healthCheck != nil && !s.healthCheck() {
		status = "unhealthy"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"status":"%s","uptime":"%s"}`, status, time.Since(s.startedAt).Round(time.Second))
}

// handleReady обрабатывает ready check запросы
func (s *GRPCServer) handleReady(w http.ResponseWriter, r *http.Request) {
	status := "ready"
	code := http.StatusOK

	if s.healthCheck != nil && !s.healthCheck() {
		status = "not ready"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"status":"%s"}`, status)
}

// GetGRPCPort возвращает gRPC порт
func (s *GRPCServer) GetGRPCPort() int {
	return s.grpcPort
}

// GetHTTPPort возвращает HTTP порт
func (s *GRPCServer) GetHTTPPort() int {
	return s.httpPort
}

// GetUptime возвращает время работы сервера
func (s *GRPCServer) GetUptime() time.Duration {
	return time.Since(s.startedAt)
}
