// Простой HTTP сервер для /api/chat endpoint
package main

import (
	"fmt"
	"net/http"
	"time"
)

// StartChatServer запускает отдельный HTTP сервер для chat endpoint
func StartChatServer(port int, qwenWrapperURL string, timeout time.Duration) {
	chatHandler := NewChatHandler(qwenWrapperURL, timeout)
	
	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat", chatHandler.ServeHTTP)
	
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Chat server starting on %s\n", addr)
	
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Printf("Chat server error: %v\n", err)
		}
	}()
}
