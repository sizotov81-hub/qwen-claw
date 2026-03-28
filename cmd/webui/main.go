// Package main предоставляет веб-сервер для нового веб-интерфейса Qwen-Claw
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	host      = flag.String("host", "127.0.0.1", "Host to listen on")
	port      = flag.Int("port", 64656, "Port to listen on")
	staticDir = flag.String("dir", "./web-new", "Static files directory")
	apiGateway = flag.String("api-gateway", "http://localhost:8080", "API Gateway URL")
)

// Session представляет сессию
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Messages  []Message `json:"messages,omitempty"`
}

// Message представляет сообщение
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatRequest запрос чата
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Model     string `json:"model"`
}

// ChatResponse ответ чата
type ChatResponse struct {
	Response string `json:"response"`
}

// SessionStore хранилище сессий
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

func (s *SessionStore) Create(userID, title string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	id := fmt.Sprintf("session-%d", time.Now().UnixNano())
	session := &Session{
		ID:        id,
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now(),
		Messages:  make([]Message, 0),
	}
	s.sessions[id] = session
	return session
}

func (s *SessionStore) Get(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}

func (s *SessionStore) List() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

func (s *SessionStore) AddMessage(sessionID, role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if session, ok := s.sessions[sessionID]; ok {
		session.Messages = append(session.Messages, Message{
			ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
			Role:      role,
			Content:   content,
			Timestamp: time.Now(),
		})
	}
}

var sessionStore = NewSessionStore()

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
	fmt.Printf("Static dir: %s\n", *staticDir)
	fmt.Printf("API Gateway: %s\n\n", *apiGateway)
	fmt.Printf("Press Ctrl+C to stop\n\n")

	// Создаём mux
	mux := http.NewServeMux()

	// Сначала API endpoints (до файлового сервера!)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/sessions", handleSessions)
	mux.HandleFunc("/api/sessions/", handleSessionDetail)
	mux.HandleFunc("/api/chat", handleChat)

	// Потом статические файлы
	mux.Handle("/", http.FileServer(http.Dir(*staticDir)))

	// Запускаем сервер
	addr := fmt.Sprintf("%s:%d", *host, *port)
	fmt.Printf("Starting web server on %s...\n", addr)
	
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to start server: %v\n", err)
		os.Exit(1)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("[%s] %s %s %v\n", r.RemoteAddr, r.Method, r.URL.Path, time.Since(start))
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"service": "web-ui",
		"timestamp": time.Now().Unix(),
	})
}

func handleSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.Method {
	case http.MethodGet:
		sessions := sessionStore.List()
		json.NewEncoder(w).Encode(sessions)
		
	case http.MethodPost:
		var req struct {
			UserID string `json:"user_id"`
			Title  string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			req.UserID = "default"
			req.Title = "New Chat"
		}
		session := sessionStore.Create(req.UserID, req.Title)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(session)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Извлекаем ID из пути
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/")
	sessionID := parts[0]
	
	switch r.Method {
	case http.MethodGet:
		if len(parts) > 1 && parts[1] == "messages" {
			session := sessionStore.Get(sessionID)
			if session == nil {
				http.Error(w, "Session not found", http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(session.Messages)
		} else {
			session := sessionStore.Get(sessionID)
			if session == nil {
				http.Error(w, "Session not found", http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(session)
		}
		
	case http.MethodDelete:
		// В реальной реализации нужно удалить сессию
		w.WriteHeader(http.StatusNoContent)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	// Создаём сессию если не указана
	if req.SessionID == "" {
		session := sessionStore.Create("user", "New Chat")
		req.SessionID = session.ID
	}
	
	// Добавляем сообщение пользователя в историю
	sessionStore.AddMessage(req.SessionID, "user", req.Content)
	
	// Отправляем запрос в API Gateway
	response, err := callAPIGateway(req)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"response": fmt.Sprintf("❌ Error: %v", err),
		})
		return
	}
	
	// Добавляем ответ ассистента в историю
	sessionStore.AddMessage(req.SessionID, "assistant", response)
	
	json.NewEncoder(w).Encode(ChatResponse{Response: response})
}

func callAPIGateway(req ChatRequest) (string, error) {
	// Используем API Gateway на порту 8085 для /api/chat
	apiGatewayURL := "http://localhost:8085"
	
	// Создаём запрос к API Gateway
	payload := map[string]interface{}{
		"query": req.Content,
		"model": req.Model,
	}

	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(apiGatewayURL+"/api/chat", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		// Если API Gateway недоступен, возвращаем заглушку
		return generateMockResponse(req.Content), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Если API Gateway вернул 404, используем заглушку
	if resp.StatusCode == 404 {
		return generateMockResponse(req.Content), nil
	}

	// Парсим ответ
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), nil
	}

	if response, ok := result["response"].(string); ok {
		return response, nil
	}
	if content, ok := result["content"].(string); ok {
		return content, nil
	}

	return string(body), nil
}

func generateMockResponse(content string) string {
	responses := []string{
		fmt.Sprintf("🤖 Я получил ваш запрос: %q\n\nПримечание: API Gateway не имеет endpoint /api/chat. Для полноценной работы нужно реализовать интеграцию.", content),
		fmt.Sprintf("👋 Здравствуйте! Ваш запрос: %q\n\nЯ работаю в демо-режиме.", content),
		fmt.Sprintf("✅ Запрос принят: %q\n\nДля выполнения команды необходим запущенный Qwen Wrapper Service.", content),
	}
	return responses[len(content)%len(responses)]
}
