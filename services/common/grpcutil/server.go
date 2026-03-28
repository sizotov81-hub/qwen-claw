// Package grpcutil предоставляет базовый gRPC сервер для микросервисов Qwen-Claw
package grpcutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// GRPCServer базовый gRPC сервер
type GRPCServer struct {
	mu          sync.RWMutex
	server      *grpc.Server
	httpServer  *http.Server
	grpcPort    int
	httpPort    int
	host        string
	logger      *zap.SugaredLogger
	healthCheck func() bool
	startedAt   time.Time
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

	// Создаём gRPC сервер с health check
	healthServer := health.NewServer()
	s.server = grpc.NewServer(
		grpc.UnaryInterceptor(s.loggingInterceptor),
	)

	// Регистрируем health service
	healthpb.RegisterHealthServer(s.server, healthServer)

	// Создаём HTTP сервер для health/metrics
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, httpPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

// loggingInterceptor логгирует gRPC запросы
func (s *GRPCServer) loggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	// Вызываем handler
	resp, err := handler(ctx, req)

	duration := time.Since(start)

	if err != nil {
		s.logger.Errorw("gRPC request failed",
			"method", info.FullMethod,
			"duration", duration.String(),
			"error", err,
		)
	} else {
		s.logger.Debugw("gRPC request completed",
			"method", info.FullMethod,
			"duration", duration.String(),
		)
	}

	return resp, err
}

// GetGRPCServer возвращает underlying gRPC сервер
func (s *GRPCServer) GetGRPCServer() *grpc.Server {
	return s.server
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

	go func() {
		s.logger.Infof("gRPC server listening on %s", grpcAddr)
		if err := s.server.Serve(grpcListener); err != nil {
			s.logger.Errorf("gRPC server error: %v", err)
		}
	}()

	return nil
}

// handleHealth обрабатывает health check запросы
func (s *GRPCServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.healthCheck != nil && !s.healthCheck() {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"unhealthy"}`)
		return
	}

	uptime := time.Since(s.startedAt)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","uptime":"%s"}`, uptime.String())
}

// handleReady обрабатывает ready check запросы
func (s *GRPCServer) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.healthCheck != nil && !s.healthCheck() {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not_ready"}`)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ready"}`)
}

// Stop останавливает сервер
func (s *GRPCServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Graceful shutdown для HTTP
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.Errorf("HTTP server shutdown error: %v", err)
		}
	}

	// Graceful shutdown для gRPC
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("gRPC server stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("gRPC server forced to stop")
		s.server.Stop()
	}

	return nil
}

// GetUptime возвращает время работы сервера
func (s *GRPCServer) GetUptime() time.Duration {
	return time.Since(s.startedAt)
}
