// Простой HTTP сервер для /api/chat endpoint
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/user/qwen-claw-common/logging"
)

// StartChatServer запускает отдельный HTTP сервер для chat endpoint
func StartChatServer(port int, qwenWrapperURL string, timeout time.Duration) {
	chatHandler := NewChatHandler(qwenWrapperURL, timeout)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat", chatHandler.ServeHTTP)

	addr := fmt.Sprintf(":%d", port)
	logging.Infof("Chat server starting on %s", addr)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			logging.Errorf("Chat server error on %s: %v", addr, err)
		}
	}()
}
