package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/user/qwen-claw/internal/agent"
	"github.com/user/qwen-claw/internal/memory"
	"github.com/user/qwen-claw/internal/scheduler"
)

// ServerConfig конфигурация веб-сервера
type ServerConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	SecretPhrase string `json:"secret_phrase"`
	JWTSecret   string `json:"jwt_secret"`
}

// AuthClaims JWT claims
type AuthClaims struct {
	Authenticated bool   `json:"auth"`
	IP            string `json:"ip"`
	jwt.RegisteredClaims
}

// Server веб-сервер
type Server struct {
	config    ServerConfig
	server    *http.Server
	agent     *agent.Agent
	memory    *memory.Manager
	scheduler *scheduler.Scheduler
	wsClients map[*websocket.Conn]bool
	wsMu      sync.RWMutex
	authData  string // Для хранения secret phrase
}

// upgrader для WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// NewServer создаёт новый веб-сервер
func NewServer(
	config ServerConfig,
	agentInstance *agent.Agent,
	memoryManager *memory.Manager,
	schedulerInstance *scheduler.Scheduler,
) *Server {
	return &Server{
		config:    config,
		agent:     agentInstance,
		memory:    memoryManager,
		scheduler: schedulerInstance,
		wsClients: make(map[*websocket.Conn]bool),
		authData:  config.SecretPhrase,
	}
}

// InitSecurity инициализирует безопасность (генерирует секреты если не заданы)
func InitSecurity(configPath string, config *ServerConfig) error {
	// Проверяем, существует ли файл с секретами
	secretsFile := filepath.Join(filepath.Dir(configPath), ".web-secrets.json")

	if data, err := os.ReadFile(secretsFile); err == nil {
		// Загружаем существующие секреты
		var saved map[string]string
		if err := json.Unmarshal(data, &saved); err == nil {
			if config.SecretPhrase == "" {
				config.SecretPhrase = saved["secret_phrase"]
			}
			if config.JWTSecret == "" {
				config.JWTSecret = saved["jwt_secret"]
			}
			return nil
		}
	}

	// Генерируем новые секреты
	if config.SecretPhrase == "" {
		config.SecretPhrase = generateSecureToken(32)
	}
	if config.JWTSecret == "" {
		config.JWTSecret = generateSecureToken(64)
	}

	// Сохраняем секреты
	secrets := map[string]string{
		"secret_phrase": config.SecretPhrase,
		"jwt_secret":    config.JWTSecret,
	}
	data, _ := json.MarshalIndent(secrets, "", "  ")
	os.WriteFile(secretsFile, data, 0600)

	fmt.Printf("🔐 Web UI Security Initialized\n")
	fmt.Printf("   Secret Phrase: %s\n", config.SecretPhrase)
	fmt.Printf("   (saved to %s)\n\n", secretsFile)

	return nil
}

// generateSecureToken генерирует криптографически безопасный токен
func generateSecureToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Start запускает веб-сервер
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Статические файлы (только для аутентифицированных)
	mux.HandleFunc("/", s.staticHandler)

	// API endpoints
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/auth", s.handleAuth)
	mux.HandleFunc("/api/chat", s.authMiddleware(s.handleChat))
	mux.HandleFunc("/api/memory", s.authMiddleware(s.handleMemory))
	mux.HandleFunc("/api/tasks", s.authMiddleware(s.handleTasks))
	mux.HandleFunc("/api/skills", s.authMiddleware(s.handleSkills))
	mux.HandleFunc("/api/status", s.authMiddleware(s.handleStatus))
	mux.HandleFunc("/api/confirmations", s.authMiddleware(s.handleConfirmations))
	mux.HandleFunc("/api/confirm", s.authMiddleware(s.handleConfirm))

	// WebSocket
	mux.HandleFunc("/ws", s.wsAuthMiddleware(s.handleWebSocket))

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler:      s.corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return s.server.ListenAndServe()
}

// Stop останавливает веб-сервер
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// staticHandler обрабатывает статические файлы с проверкой секретной фразы
func (s *Server) staticHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Эти файлы отдаём всегда (форма входа и зависимости)
	publicFiles := map[string]bool{
		"/":          true,
		"/index.html": true,
		"/styles.css": true,
		"/app.js":     true,
	}

	if publicFiles[path] {
		filePath := filepath.Join("internal/web/static", strings.TrimPrefix(path, "/"))
		if path == "/" {
			filePath = filepath.Join("internal/web/static", "index.html")
		}
		http.ServeFile(w, r, filePath)
		return
	}

	// Остальные файлы требуют секретную фразу или JWT
	secret := r.Header.Get("X-Secret-Phrase")
	if secret == "" {
		secret = r.URL.Query().Get("secret")
	}

	if secret != s.authData {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
		return
	}

	// Отдаём статический файл
	filePath := filepath.Join("internal/web/static", strings.TrimPrefix(path, "/"))
	http.ServeFile(w, r, filePath)
}

// authMiddleware проверяет JWT токен
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		}

		if tokenString == "" {
			// Проверяем секретную фразу как fallback
			secret := r.Header.Get("X-Secret-Phrase")
			if secret == s.authData {
				next.ServeHTTP(w, r)
				return
			}
			s.sendError(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		claims := &AuthClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.config.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			s.sendError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// wsAuthMiddleware для WebSocket
func (s *Server) wsAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret := r.URL.Query().Get("secret")
		token := r.URL.Query().Get("token")

		// Проверяем либо секрет, либо токен
		if secret == s.authData {
			next.ServeHTTP(w, r)
			return
		}

		if token != "" {
			claims := &AuthClaims{}
			jwtToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(s.config.JWTSecret), nil
			})

			if err == nil && jwtToken.Valid {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Отклоняем подключение без ответа (скрытие от сканирования)
		w.WriteHeader(http.StatusNotFound)
	}
}

// handleAuth обработка аутентификации
func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Проверяем секретную фразу
	if req.Secret != s.authData {
		// Возвращаем 404 для скрытия наличия сервиса
		s.sendError(w, "Not found", http.StatusNotFound)
		return
	}

	// Генерируем JWT токен
	claims := &AuthClaims{
		Authenticated: true,
		IP:            r.RemoteAddr,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		s.sendError(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	s.sendJSON(w, map[string]interface{}{
		"success": true,
		"token":   tokenString,
		"expires": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
}

// handleHealth проверка здоровья (всегда отвечает 404 без секретной фразы)
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Проверяем секретную фразу
	secret := r.Header.Get("X-Secret-Phrase")
	if secret != s.authData {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
		return
	}

	s.sendJSON(w, APIResponse{Success: true, Data: map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}})
}

// handleChat обработка чата
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		s.sendError(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Выполняем запрос
	ctx := context.Background()
	response, err := s.agent.Run(ctx, req.Message)
	if err != nil {
		s.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ через WebSocket
	s.broadcastWS(map[string]interface{}{
		"type":      "chat_response",
		"message":   req.Message,
		"response":  response,
		"timestamp": time.Now().Format(time.RFC3339),
	})

	s.sendJSON(w, APIResponse{
		Success: true,
		Data: ChatResponse{
			Message:   response,
			Timestamp: time.Now().Format(time.RFC3339),
		},
	})
}

// handleMemory управление памятью
func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		entries := s.memory.ListEntries()
		s.sendJSON(w, APIResponse{Success: true, Data: entries})

	case "POST":
		var req struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		entry, err := s.memory.Remember(req.Type, req.Content, nil)
		if err != nil {
			s.sendError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.sendJSON(w, APIResponse{Success: true, Data: entry})

	case "DELETE":
		if err := s.memory.Clear(); err != nil {
			s.sendError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.sendJSON(w, APIResponse{Success: true, Data: map[string]string{"status": "cleared"}})

	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTasks управление задачами
func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		tasks := s.scheduler.ListTasks()
		s.sendJSON(w, APIResponse{Success: true, Data: tasks})

	case "POST":
		var req struct {
			Name     string `json:"name"`
			Schedule string `json:"schedule"`
			Command  string `json:"command"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		task, err := s.scheduler.AddTask(req.Name, req.Name, req.Command, req.Schedule)
		if err != nil {
			s.sendError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.sendJSON(w, APIResponse{Success: true, Data: task})

	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSkills список навыков
func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	skills := s.agent.GetSkillEngine().List()
	s.sendJSON(w, APIResponse{Success: true, Data: skills})
}

// handleStatus статус системы
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"qwen_available": s.agent.CheckQwenAvailable(),
		"qwen_path":      s.agent.GetQwenPath(),
		"model":          s.agent.GetModel(),
		"memory_entries": len(s.memory.ListEntries()),
		"tasks_count":    len(s.scheduler.ListTasks()),
		"skills_count":   len(s.agent.GetSkillEngine().List()),
		"ws_clients":     len(s.wsClients),
	}

	s.sendJSON(w, APIResponse{Success: true, Data: status})
}

// handleWebSocket обработка WebSocket подключений
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.wsMu.Lock()
	s.wsClients[conn] = true
	s.wsMu.Unlock()

	defer func() {
		s.wsMu.Lock()
		delete(s.wsClients, conn)
		s.wsMu.Unlock()
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Обрабатываем сообщение
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if action, ok := msg["action"].(string); ok {
			switch action {
			case "chat":
				if text, ok := msg["message"].(string); ok {
					ctx := context.Background()
					
					// Отправляем статус "thinking"
					s.sendWS(conn, map[string]interface{}{
						"type":    "thinking",
						"message": "🤔 Думаю...",
					})
					
					// Выполняем запрос
					response, err := s.agent.Run(ctx, text)
					
					if err != nil {
						s.sendWS(conn, map[string]interface{}{
							"type":    "error",
							"message": err.Error(),
						})
					} else {
						// Отправляем ответ с эффектом печати (посимвольно)
						s.sendWS(conn, map[string]interface{}{
							"type":      "chat_response",
							"message":   response,
							"timestamp": time.Now().Format(time.RFC3339),
							"streaming": true,
						})
						
						// Проверяем ожидающие действия
						actions := s.agent.GetPendingActions()
						if len(actions) > 0 {
							s.sendWS(conn, map[string]interface{}{
								"type":     "confirmation",
								"actions":  actions,
								"message":  "Требуется подтверждение действия",
							})
						}
					}
				}
			}
		}
	}
}

// broadcastWS рассылает сообщение всем WebSocket клиентам
func (s *Server) broadcastWS(data map[string]interface{}) {
	s.wsMu.RLock()
	defer s.wsMu.RUnlock()

	for conn := range s.wsClients {
		s.sendWS(conn, data)
	}
}

// sendWS отправляет сообщение через WebSocket
func (s *Server) sendWS(conn *websocket.Conn, data interface{}) {
	conn.WriteJSON(data)
}

// Вспомогательные функции
func (s *Server) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) sendError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)
	s.sendJSON(w, APIResponse{Success: false, Error: message})
}

// corsMiddleware добавляет CORS заголовки
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Secret-Phrase")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// API Response структуры
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// handleConfirmations возвращает список ожидающих подтверждений
func (s *Server) handleConfirmations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	actions := s.agent.GetPendingActions()

	// Форматируем ответ
	data := make([]map[string]interface{}, 0)
	for _, action := range actions {
		if action.IsPending() {
			data = append(data, map[string]interface{}{
				"id":        action.ID,
				"query":     action.Query,
				"type":      string(action.Intent.Type),
				"created":   action.Created.Format(time.RFC3339),
				"expires":   action.Expires.Format(time.RFC3339),
				"is_expired": action.IsExpired(),
			})
		}
	}

	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    data,
	})
}

// handleConfirm обрабатывает подтверждение действия
func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим JSON запроса
	var req struct {
		ActionID string `json:"action_id"`
		Confirm  bool   `json:"confirm"` // true = подтвердить, false = отклонить
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ActionID == "" {
		s.sendError(w, "action_id is required", http.StatusBadRequest)
		return
	}

	var response string
	var err error

	if req.Confirm {
		response, err = s.agent.ConfirmAction(req.ActionID)
	} else {
		err = s.agent.RejectAction(req.ActionID)
		if err == nil {
			response = "Action rejected"
		}
	}

	if err != nil {
		s.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.sendJSON(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"message":   response,
			"action_id": req.ActionID,
			"confirmed": req.Confirm,
		},
	})
}
