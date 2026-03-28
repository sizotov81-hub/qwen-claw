// Package main предоставляет простой веб-сервер для статических файлов
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

var (
	host       = flag.String("host", "127.0.0.1", "Host to listen on")
	port       = flag.Int("port", 64656, "Port to listen on")
	staticDir  = flag.String("dir", "./internal/web/static", "Static files directory")
)

func main() {
	flag.Parse()

	// Проверяем наличие директории
	if _, err := os.Stat(*staticDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "❌ Static directory not found: %s\n", *staticDir)
		os.Exit(1)
	}

	fmt.Printf("🌐 Qwen-Claw Web UI\n")
	fmt.Printf("==================================\n\n")
	fmt.Printf("URL: http://%s:%d\n", *host, *port)
	fmt.Printf("Static dir: %s\n\n", *staticDir)
	fmt.Printf("Press Ctrl+C to stop\n\n")

	// Создаём файловый сервер
	fs := http.FileServer(http.Dir(*staticDir))

	// Добавляем логирование
	http.Handle("/", loggingMiddleware(fs))

	// Health endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"web-ui"}`)
	})

	// Запускаем сервер
	addr := fmt.Sprintf("%s:%d", *host, *port)
	fmt.Printf("Starting web server on %s...\n", addr)
	
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to start server: %v\n", err)
		os.Exit(1)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%s] %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
