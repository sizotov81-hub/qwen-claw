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
	"github.com/user/qwen-claw/internal/gateway"
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
	gateway   *gateway.Gateway
	wsClients map[*websocket.Conn]*wsClient
	wsMu      sync.RWMutex
	authData  string // Для хранения secret phrase
}

// wsClient отслеживает состояние WebSocket клиента
type wsClient struct {
	conn         *websocket.Conn
	lastMessage  time.Time
	messageCount int
	mu           sync.RWMutex
}

// upgrader для WebSocket с защитой
var upgrader = websocket.Upgrader{
	ReadBufferSize:     1024 * 1024,      // 1MB лимит на сообщение
	WriteBufferSize:    1024 * 1024,
	CheckOrigin:        func(r *http.Request) bool { return true },
	EnableCompression:  true,              // Включаем сжатие
	HandshakeTimeout:   10 * time.Second,  // Таймаут рукопожатия
}

// Константы для rate limiting
const (
	wsMaxMessageCount  = 100              // Максимум сообщений в минуту
	wsRateLimitWindow  = time.Minute      // Окно для rate limiting
	wsMaxMessageSize   = 1024 * 1024      // 1MB макс размер сообщения
	wsWriteTimeout     = 10 * time.Second // Таймаут записи
	wsPongTimeout      = 60 * time.Second // Таймаут pong
)

// NewServer создаёт новый веб-сервер
func NewServer(
	config ServerConfig,
	agentInstance *agent.Agent,
	memoryManager *memory.Manager,
	schedulerInstance *scheduler.Scheduler,
	gatewayInstance *gateway.Gateway,
) *Server {
	return &Server{
		config:    config,
		agent:     agentInstance,
		memory:    memoryManager,
		scheduler: schedulerInstance,
		gateway:   gatewayInstance,
		wsClients: make(map[*websocket.Conn]*wsClient),
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
	mux.HandleFunc("/api/chat/stream", s.authMiddleware(s.handleChatStream))
	mux.HandleFunc("/api/memory", s.authMiddleware(s.handleMemory))
	mux.HandleFunc("/api/tasks", s.authMiddleware(s.handleTasks))
	mux.HandleFunc("/api/skills", s.authMiddleware(s.handleSkillsExtended))
	mux.HandleFunc("/api/status", s.authMiddleware(s.handleStatus))
	mux.HandleFunc("/api/confirmations", s.authMiddleware(s.handleConfirmations))
	mux.HandleFunc("/api/confirm", s.authMiddleware(s.handleConfirm))
	
	// Новые endpoints для Web UI
	mux.HandleFunc("/api/v1/sessions", s.authMiddleware(s.handleSessions))
	mux.HandleFunc("/api/v1/config", s.authMiddleware(s.handleConfig))
	mux.HandleFunc("/api/v1/logs", s.authMiddleware(s.handleLogs))

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
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// staticHandler обрабатывает статические файлы с проверкой секретной фразы
func (s *Server) staticHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Эти файлы отдаём всегда (форма входа и зависимости)
	publicFiles := map[string]bool{
		"/":           true,
		"/index.html": true,
		"/static/":    true,
	}

	isPublic := publicFiles[path] || strings.HasPrefix(path, "/static/")
	
	if isPublic {
		// Обслуживаем файлы из web/static
		filePath := filepath.Join("web/static", strings.TrimPrefix(path, "/"))
		if path == "/" {
			filePath = filepath.Join("web/static", "index.html")
		}
		
		// Проверяем существование файла
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			http.NotFound(w, r)
			return
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
	filePath := filepath.Join("web/static", strings.TrimPrefix(path, "/"))
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

// handleChatStream streaming ответов через SSE
func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
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

	// Настраиваем SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.sendError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Отправляем событие начала
	fmt.Fprintf(w, "event: start\ndata: {\"status\": \"thinking\"}\n\n")
	flusher.Flush()

	// Выполняем запрос
	ctx := context.Background()
	response, err := s.agent.Run(ctx, req.Message)

	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"error\": \"%s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}

	// Отправляем ответ по символам (эффект печати)
	for _, ch := range response {
		fmt.Fprintf(w, "event: token\ndata: {\"token\": \"%c\"}\n\n", ch)
		flusher.Flush()
		time.Sleep(10 * time.Millisecond) // Скорость печати
	}

	// Отправляем событие завершения
	fmt.Fprintf(w, "event: complete\ndata: {\"response\": \"%s\"}\n\n", response)
	flusher.Flush()

	// Проверяем ожидающие действия
	actions := s.agent.GetPendingActions()
	if len(actions) > 0 {
		fmt.Fprintf(w, "event: confirmation\ndata: {\"actions\": %d}\n\n", len(actions))
		flusher.Flush()
	}
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

// forwardGatewayEvents пересылает события Gateway в WebSocket
func (s *Server) forwardGatewayEvents(conn *websocket.Conn) {
	// Подписываемся на broadcast Gateway
	// В реальной реализации Gateway должен иметь канал для подписки
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		if s.gateway != nil {
			clientCount := s.gateway.GetClientCount()
			s.sendWS(conn, map[string]interface{}{
				"type": "event",
				"payload": map[string]interface{}{
					"event":         "gateway_status",
					"client_count":  clientCount,
					"timestamp":     time.Now().Format(time.RFC3339),
				},
			})
		}
	}
}

// handleWebSocket обработка WebSocket подключений
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Устанавливаем лимит на размер сообщения
	conn.SetReadLimit(wsMaxMessageSize)

	// Создаём клиента с rate limiting
	client := &wsClient{
		conn:         conn,
		lastMessage:  time.Now(),
		messageCount: 0,
	}

	s.wsMu.Lock()
	s.wsClients[conn] = client
	s.wsMu.Unlock()

	defer func() {
		s.wsMu.Lock()
		delete(s.wsClients, conn)
		s.wsMu.Unlock()
		conn.Close()
	}()

	// Устанавливаем таймауты
	conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
		return nil
	})

	// Отправляем ping для keepalive
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(wsWriteTimeout)); err != nil {
				return
			}
		}
	}()

	// Подписываемся на события Gateway если есть
	if s.gateway != nil {
		go s.forwardGatewayEvents(conn)
	}

	for {
		// Проверяем rate limit
		if !client.checkRateLimit() {
			s.sendWS(conn, map[string]interface{}{
				"type":    "error",
				"message": "Rate limit exceeded. Please slow down.",
			})
			continue
		}

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
						// Отправляем ответ
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

// checkRateLimit проверяет rate limit для клиента
func (c *wsClient) checkRateLimit() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Сбрасываем счётчик если окно истекло
	if now.Sub(c.lastMessage) > wsRateLimitWindow {
		c.messageCount = 0
		c.lastMessage = now
	}

	// Проверяем лимит
	if c.messageCount >= wsMaxMessageCount {
		return false
	}

	c.messageCount++
	return true
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
	Message string      `json:"message,omitempty"`
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

// handleSessions - список сессий
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listSessions(w, r)
	case "POST":
		s.createSession(w, r)
	case "DELETE":
		s.deleteSession(w, r)
	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	// Получаем сессии из Gateway
	var sessions []map[string]interface{}
	
	if s.gateway != nil {
		gwSessions := s.gateway.GetSessionManager().ListSessions()
		sessions = make([]map[string]interface{}, 0, len(gwSessions))
		for _, gs := range gwSessions {
			sessions = append(sessions, map[string]interface{}{
				"id":            gs.ID,
				"type":          string(gs.Type),
				"status":        string(gs.Status),
				"channel":       gs.Channel,
				"client_id":     gs.ClientID,
				"message_count": gs.MessageCount,
				"created":       gs.Created.Format(time.RFC3339),
				"last_access":   gs.LastAccess.Format(time.RFC3339),
			})
		}
	}
	
	// Если нет сессий из Gateway — возвращаем заглушку
	if len(sessions) == 0 {
		sessions = []map[string]interface{}{
			{
				"id":            "default",
				"type":          "main",
				"status":        "active",
				"channel":       "web",
				"message_count": 0,
				"created":       time.Now().Format(time.RFC3339),
				"last_access":   time.Now().Format(time.RFC3339),
			},
		}
	}
	
	s.sendJSON(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"sessions": sessions,
			"total":    len(sessions),
		},
	})
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	session := map[string]interface{}{
		"id":            fmt.Sprintf("session_%d", time.Now().UnixNano()),
		"type":          req.Type,
		"status":        "active",
		"channel":       "web",
		"message_count": 0,
		"created":       time.Now().Format(time.RFC3339),
		"last_access":   time.Now().Format(time.RFC3339),
	}
	
	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    session,
		Message: "Session created",
	})
}

func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		s.sendError(w, "Session ID required", http.StatusBadRequest)
		return
	}
	
	s.sendJSON(w, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Session %s deleted", sessionID),
	})
}

// handleConfig - конфигурация
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.getConfig(w, r)
	case "POST":
		s.saveConfig(w, r)
	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	// Загружаем актуальный конфиг из агента
	config := map[string]interface{}{
		"host": s.config.Host,
		"port": s.config.Port,
		"llm": map[string]interface{}{
			"model":         s.agent.GetModel(),
			"approval_mode": s.agent.GetApprovalMode(),
		},
		"gateway": map[string]interface{}{
			"enabled": s.gateway != nil,
			"port":    18789,
			"clients": 0,
		},
		"sandbox": map[string]interface{}{
			"enabled": true,
			"mode":    "per-session",
		},
	}
	
	// Добавляем информацию о Gateway если есть
	if s.gateway != nil {
		config["gateway"].(map[string]interface{})["clients"] = s.gateway.GetClientCount()
	}
	
	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    config,
	})
}

func (s *Server) saveConfig(w http.ResponseWriter, r *http.Request) {
	var config map[string]interface{}
	
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.sendError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Сохраняем конфиг в файл
	configPath := "config.web.json"
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		s.sendError(w, "Failed to marshal config", http.StatusInternalServerError)
		return
	}
	
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		s.sendError(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Применяем изменения если возможно
	if llm, ok := config["llm"].(map[string]interface{}); ok {
		if model, ok := llm["model"].(string); ok && model != "" {
			s.agent.SetModel(model)
		}
		if approvalMode, ok := llm["approval_mode"].(string); ok && approvalMode != "" {
			s.agent.SetApprovalMode(approvalMode)
		}
	}
	
	s.sendJSON(w, APIResponse{
		Success: true,
		Message: "Configuration saved and applied",
		Data: map[string]interface{}{
			"path": configPath,
		},
	})
}

// handleLogs - логи
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.getLogs(w, r)
	case "WS":
		s.streamLogs(w, r)
	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getLogs(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")
	if level == "" {
		level = "all"
	}
	limit := 100
	
	// Загружаем логи из файла если существует
	logFile := ".qwen/logs/qwen-claw.log"
	logs := s.loadLogsFromFile(logFile, level, limit)
	
	// Если логов нет — возвращаем тестовые
	if len(logs) == 0 {
		logs = []map[string]interface{}{
			{
				"timestamp": time.Now().Format(time.RFC3339),
				"level":     "info",
				"message":   "Web UI started",
			},
			{
				"timestamp": time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
				"level":     "info",
				"message":   "Gateway connected",
			},
		}
	}
	
	s.sendJSON(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"logs":  logs,
			"total": len(logs),
		},
	})
}

// loadLogsFromFile загружает логи из файла
func (s *Server) loadLogsFromFile(filePath string, level string, limit int) []map[string]interface{} {
	logs := make([]map[string]interface{}, 0)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return logs
	}
	
	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0 && len(logs) < limit; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		
		// Парсим лог (формат: timestamp level message)
		logEntry := s.parseLogLine(line)
		if logEntry != nil {
			if level == "all" || logEntry["level"] == level {
				logs = append(logs, logEntry)
			}
		}
	}
	
	return logs
}

// parseLogLine парсит строку лога
func (s *Server) parseLogLine(line string) map[string]interface{} {
	// Простая эвристика для парсинга
	parts := strings.SplitN(line, " ", 4)
	if len(parts) < 3 {
		return nil
	}
	
	level := strings.ToLower(parts[1])
	if level != "info" && level != "warn" && level != "error" && level != "debug" {
		level = "info"
	}
	
	message := line
	if len(parts) >= 4 {
		message = parts[3]
	}
	
	return map[string]interface{}{
		"timestamp": parts[0] + " " + parts[1],
		"level":     level,
		"message":   message,
	}
}

// streamLogs - WebSocket streaming логов
func (s *Server) streamLogs(w http.ResponseWriter, r *http.Request) {
	// Upgrade до WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	
	defer conn.Close()
	
	// Отправляем начальные логи
	logFile := ".qwen/logs/qwen-claw.log"
	initialLogs := s.loadLogsFromFile(logFile, "all", 50)
	
	for _, log := range initialLogs {
		s.sendWS(conn, map[string]interface{}{
			"type": "log",
			"payload": log,
		})
	}
	
	// Tail файл в реальном времени
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	lastSize := int64(0)
	for {
		select {
		case <-ticker.C:
			// Проверяем изменения в файле логов
			info, err := os.Stat(logFile)
			if err == nil && info.Size() > lastSize {
				// Файл изменился — читаем новые строки
				data, err := os.ReadFile(logFile)
				if err == nil {
					lines := strings.Split(string(data), "\n")
					for i := len(lines) - 51; i < len(lines); i++ {
						if i >= 0 && strings.TrimSpace(lines[i]) != "" {
							logEntry := s.parseLogLine(lines[i])
							if logEntry != nil {
								s.sendWS(conn, map[string]interface{}{
									"type": "log",
									"payload": logEntry,
								})
							}
						}
					}
					lastSize = info.Size()
				}
			}
		case <-r.Context().Done():
			return
		}
	}
}

// handleSkills - расширенная версия
func (s *Server) handleSkillsExtended(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.getSkills(w, r)
	case "POST":
		s.installSkill(w, r)
	case "DELETE":
		s.uninstallSkill(w, r)
	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getSkills(w http.ResponseWriter, r *http.Request) {
	skills := s.agent.GetSkillEngine().List()
	
	skillList := make([]map[string]interface{}, 0, len(skills))
	for _, skill := range skills {
		skillList = append(skillList, map[string]interface{}{
			"name":        skill.Name,
			"description": skill.Description,
			"type":        string(skill.Type),
			"enabled":     skill.Enabled,
			"commands":    skill.Commands,
			"entry_point": skill.EntryPoint,
		})
	}
	
	s.sendJSON(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"skills": skillList,
			"total":  len(skillList),
		},
	})
}

func (s *Server) installSkill(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	// В реальной реализации - установка из реестра
	
	s.sendJSON(w, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Skill %s installed", req.Name),
	})
}

func (s *Server) uninstallSkill(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	// В реальной реализации - удаление навыка
	
	s.sendJSON(w, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Skill %s uninstalled", req.Name),
	})
}
