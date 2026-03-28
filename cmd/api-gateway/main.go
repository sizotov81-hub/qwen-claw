// Package main предоставляет точку входа для API Gateway Service
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/user/qwen-claw-common/logging"
	"github.com/user/qwen-claw-common/metrics"
	"github.com/user/qwen-claw-common/grpcutil"
)

var (
	grpcPort    = flag.Int("grpc-port", 55050, "gRPC port")
	httpPort    = flag.Int("http-port", 58080, "HTTP port")
	host        = flag.String("host", "0.0.0.0", "Host to listen on")
	logLevel    = flag.String("log-level", "info", "Log level")
	logFormat   = flag.String("log-format", "json", "Log format (json/console)")
	metricsPort = flag.Int("metrics-port", 59090, "Metrics port")
)

func main() {
	flag.Parse()

	// Инициализация логгера
	logCfg := &logging.Config{
		Level:       *logLevel,
		Format:      *logFormat,
		Output:      "stdout",
		AddCaller:   true,
		ServiceName: "api-gateway",
	}
	if err := logging.Init(logCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	logging.Info("Starting API Gateway Service",
		"grpc_port", *grpcPort,
		"http_port", *httpPort,
		"host", *host,
	)

	// Инициализация метрик
	metricsCfg := &metrics.Config{
		ServiceName: "api-gateway",
		Port:        *metricsPort,
		Path:        "/metrics",
		Enabled:     true,
		Logger:      logging.GetLogger(),
	}
	m, err := metrics.NewMetrics(metricsCfg)
	if err != nil {
		logging.Errorf("Failed to init metrics: %v", err)
		os.Exit(1)
	}

	// Создание gRPC сервера
	grpcServer := grpcutil.NewGRPCServer(
		*grpcPort,
		*httpPort,
		*host,
		grpcutil.WithLogger(logging.GetLogger()),
		grpcutil.WithHealthCheck(func() bool { return true }),
	)

	// HTTP handler для health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"api-gateway"}`)
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready"}`)
	})

	// Chat endpoint - интеграция с Web UI
	// Запускаем отдельный HTTP сервер для /api/chat на порту 8085
	go StartChatServer(8085, "http://localhost:50052", 5*time.Minute)

	// Запуск сервера
	if err := grpcServer.Start(); err != nil {
		logging.Errorf("Failed to start server: %v", err)
		os.Exit(1)
	}

	logging.Info("API Gateway Service started successfully")

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logging.Info("Shutting down API Gateway Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := grpcServer.Stop(ctx); err != nil {
		logging.Errorf("Failed to stop server: %v", err)
	}

	if err := m.Close(); err != nil {
		logging.Errorf("Failed to close metrics: %v", err)
	}

	logging.Info("API Gateway Service stopped")
}
