// Package main предоставляет точку входа для Tools Executor Service
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/user/qwen-claw-common/logging"
	"github.com/user/qwen-claw-common/metrics"
	"github.com/user/qwen-claw-common/grpcutil"
)

var (
	grpcPort     = flag.Int("grpc-port", 55054, "gRPC port")
	httpPort     = flag.Int("http-port", 58084, "HTTP port")
	host         = flag.String("host", "0.0.0.0", "Host to listen on")
	logLevel     = flag.String("log-level", "info", "Log level")
	logFormat    = flag.String("log-format", "json", "Log format (json/console)")
	metricsPort  = flag.Int("metrics-port", 59094, "Metrics port")
	sandboxMode  = flag.String("sandbox-mode", "docker", "Sandbox mode (none/docker)")
)

func main() {
	flag.Parse()

	// Инициализация логгера
	logCfg := &logging.Config{
		Level:       *logLevel,
		Format:      *logFormat,
		Output:      "stdout",
		AddCaller:   true,
		ServiceName: "tools-executor",
	}
	if err := logging.Init(logCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	logging.Info("Starting Tools Executor Service",
		"grpc_port", *grpcPort,
		"sandbox_mode", *sandboxMode,
	)

	// Инициализация метрик
	metricsCfg := &metrics.Config{
		ServiceName: "tools-executor",
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

	// TODO: Инициализация Tools Executor сервиса
	// executor := tools.NewExecutor(*sandboxMode)
	// pb.RegisterToolsExecutorServer(grpcServer.GetGRPCServer(), executor)

	// Запуск сервера
	if err := grpcServer.Start(); err != nil {
		logging.Errorf("Failed to start server: %v", err)
		os.Exit(1)
	}

	logging.Info("Tools Executor Service started successfully")

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logging.Info("Shutting down Tools Executor Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := grpcServer.Stop(ctx); err != nil {
		logging.Errorf("Failed to stop server: %v", err)
	}

	if err := m.Close(); err != nil {
		logging.Errorf("Failed to close metrics: %v", err)
	}

	logging.Info("Tools Executor Service stopped")
}
