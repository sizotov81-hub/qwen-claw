// Package main предоставляет точку входа для LLM Proxy Service
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
	grpcPort      = flag.Int("grpc-port", 55053, "gRPC port")
	httpPort      = flag.Int("http-port", 58083, "HTTP port")
	host          = flag.String("host", "0.0.0.0", "Host to listen on")
	logLevel      = flag.String("log-level", "info", "Log level")
	logFormat     = flag.String("log-format", "json", "Log format (json/console)")
	metricsPort   = flag.Int("metrics-port", 59093, "Metrics port")
	openaiKey     = flag.String("openai-key", "", "OpenAI API key")
	anthropicKey  = flag.String("anthropic-key", "", "Anthropic API key")
	dashscopeKey  = flag.String("dashscope-key", "", "DashScope API key")
)

func main() {
	flag.Parse()

	// Инициализация логгера
	logCfg := &logging.Config{
		Level:       *logLevel,
		Format:      *logFormat,
		Output:      "stdout",
		AddCaller:   true,
		ServiceName: "llm-proxy",
	}
	if err := logging.Init(logCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	logging.Info("Starting LLM Proxy Service",
		"grpc_port", *grpcPort,
	)

	// Инициализация метрик
	metricsCfg := &metrics.Config{
		ServiceName: "llm-proxy",
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

	// TODO: Инициализация LLM Proxy сервиса
	// proxy := llm.NewProxy(*openaiKey, *anthropicKey, *dashscopeKey)
	// pb.RegisterLLMProxyServer(grpcServer.GetGRPCServer(), proxy)

	// Запуск сервера
	if err := grpcServer.Start(); err != nil {
		logging.Errorf("Failed to start server: %v", err)
		os.Exit(1)
	}

	logging.Info("LLM Proxy Service started successfully")

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logging.Info("Shutting down LLM Proxy Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := grpcServer.Stop(ctx); err != nil {
		logging.Errorf("Failed to stop server: %v", err)
	}

	if err := m.Close(); err != nil {
		logging.Errorf("Failed to close metrics: %v", err)
	}

	logging.Info("LLM Proxy Service stopped")
}
