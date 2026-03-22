package gateway

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/user/qwen-claw/internal/logger"
)

// MessageType тип сообщения
type MessageType string

const (
	// Клиент → Сервер
	MessageTypeAuth       MessageType = "auth"
	MessageTypeChat       MessageType = "chat"
	MessageTypeCommand    MessageType = "command"
	MessageTypeSubscribe  MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"

	// Сервер → Клиент
	MessageTypeAuthResponse MessageType = "auth_response"
	MessageTypeChatResponse MessageType = "chat_response"
	MessageTypeCommandResponse MessageType = "command_response"
	MessageTypeError        MessageType = "error"
	MessageTypeEvent        MessageType = "event"
	MessageTypePresence     MessageType = "presence"
)

// Message сообщение Gateway
type Message struct {
	// Type тип сообщения
	Type MessageType `json:"type"`

	// ID уникальный идентификатор сообщения
	ID string `json:"id"`

	// SessionID ID сессии
	SessionID string `json:"session_id,omitempty"`

	// Payload полезные данные
	Payload json.RawMessage `json:"payload,omitempty"`

	// Timestamp временная метка
	Timestamp time.Time `json:"timestamp"`
}

// AuthMessage сообщение аутентификации
type AuthMessage struct {
	Token    string `json:"token,omitempty"`
	Password string `json:"password,omitempty"`
	ClientID string `json:"client_id"`
	ClientType string `json:"client_type"` // cli/web/mobile/bot
}

// AuthResponse ответ аутентификации
type AuthResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"session_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ChatMessage сообщение чата
type ChatMessage struct {
	Content string `json:"content"`
	Model   string `json:"model,omitempty"`
}

// CommandMessage команда
type CommandMessage struct {
	Command string                 `json:"command"`
	Args    map[string]interface{} `json:"args,omitempty"`
}

// ErrorResponse ответ с ошибкой
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// EventMessage событие
type EventMessage struct {
	Event     string                 `json:"event"`
	SessionID string                 `json:"session_id"`
	Data      map[string]interface{} `json:"data"`
}

// Client клиент Gateway
type Client struct {
	mu        sync.RWMutex
	ID        string
	Type      string // cli/web/mobile/bot
	Conn      *websocket.Conn
	SessionID string
	Send      chan []byte
	Subscriptions map[string]bool // каналы подписки
	LastActivity time.Time
}

// Gateway WebSocket сервер
type Gateway struct {
	mu            sync.RWMutex
	config        *Config
	server        *http.Server
	upgrader      websocket.Upgrader
	clients       map[string]*Client
	sessionManager *SessionManager
	hub           chan *Message
	ctx           context.Context
	cancel        context.CancelFunc
	authToken     string
}

// NewGateway создаёт новый Gateway
func NewGateway(config *Config) (*Gateway, error) {
	ctx, cancel := context.WithCancel(context.Background())

	g := &Gateway{
		config:  config,
		clients: make(map[string]*Client),
		hub:     make(chan *Message, 256),
		ctx:     ctx,
		cancel:  cancel,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Разрешаем все origins для localhost
				return true
			},
		},
	}

	// Создаём менеджер сессий
	sessionDir := "/home/ss/qwen-claw/.qwen/sessions"
	sm, err := NewSessionManager(sessionDir)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}
	g.sessionManager = sm

	// Генерируем токен аутентификации если не указан
	if config.Auth.Token == "" {
		g.authToken = generateToken()
		logger.Infof("Generated Gateway auth token: %s", g.authToken)
	} else {
		g.authToken = config.Auth.Token
	}

	return g, nil
}

// generateToken генерирует безопасный токен
func generateToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// Start запускает Gateway сервер
func (g *Gateway) Start() error {
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", g.handleWebSocket)

	// Health check
	mux.HandleFunc("/health", g.handleHealth)

	// API endpoints
	mux.HandleFunc("/api/v1/sessions", g.handleSessions)
	mux.HandleFunc("/api/v1/clients", g.handleClients)
	mux.HandleFunc("/api/v1/broadcast", g.handleBroadcast)

	g.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", g.config.Bind, g.config.Port),
		Handler:      g.withAuth(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Infof("🔌 Gateway starting on %s:%d", g.config.Bind, g.config.Port)
	logger.Infof("🔑 Auth token: %s", g.authToken)

	// Запускаем обработчик сообщений
	go g.messageHandler()

	// Запускаем HTTP сервер
	if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Stop останавливает Gateway
func (g *Gateway) Stop() error {
	g.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return g.server.Shutdown(ctx)
}

// GetAuthToken возвращает токен аутентификации
func (g *Gateway) GetAuthToken() string {
	return g.authToken
}

// GetSessionManager возвращает менеджер сессий
func (g *Gateway) GetSessionManager() *SessionManager {
	return g.sessionManager
}

// GetClientCount возвращает количество подключённых клиентов
func (g *Gateway) GetClientCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.clients)
}

// Broadcast рассылает сообщение всем клиентам
func (g *Gateway) Broadcast(event string, data map[string]interface{}) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	msg := EventMessage{
		Event: event,
		Data:  data,
	}

	payload, _ := json.Marshal(msg)

	for _, client := range g.clients {
		select {
		case client.Send <- payload:
		default:
			// Клиент не готов — закрываем
			close(client.Send)
		}
	}
}

// handleWebSocket обрабатывает WebSocket подключения
func (g *Gateway) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		ID:            generateClientID(),
		Type:          "unknown",
		Conn:          conn,
		Send:          make(chan []byte, 256),
		Subscriptions: make(map[string]bool),
		LastActivity:  time.Now(),
	}

	// Запускаем обработчики клиента
	go g.writePump(client)
	go g.readPump(client)
}

// readPump читает сообщения от клиента
func (g *Gateway) readPump(client *Client) {
	defer func() {
		g.removeClient(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(512 * 1024) // 512KB лимит
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.mu.Lock()
		client.LastActivity = time.Now()
		client.mu.Unlock()
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Errorf("WebSocket error: %v", err)
			}
			break
		}

		client.mu.Lock()
		client.LastActivity = time.Now()
		client.mu.Unlock()

		// Парсим сообщение
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.Errorf("Failed to parse message: %v", err)
			continue
		}

		// Обрабатываем сообщение
		g.handleMessage(client, &msg)
	}
}

// writePump пишет сообщения клиенту
func (g *Gateway) writePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		g.removeClient(client)
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage обрабатывает сообщение от клиента
func (g *Gateway) handleMessage(client *Client, msg *Message) {
	switch msg.Type {
	case MessageTypeAuth:
		g.handleAuth(client, msg)
	case MessageTypeChat:
		g.handleChat(client, msg)
	case MessageTypeCommand:
		g.handleCommand(client, msg)
	case MessageTypeSubscribe:
		g.handleSubscribe(client, msg)
	case MessageTypeUnsubscribe:
		g.handleUnsubscribe(client, msg)
	default:
		g.sendError(client, 400, "Unknown message type", "")
	}
}

// handleAuth обрабатывает аутентификацию
func (g *Gateway) handleAuth(client *Client, msg *Message) {
	var auth AuthMessage
	if err := json.Unmarshal(msg.Payload, &auth); err != nil {
		g.sendError(client, 400, "Invalid auth payload", "")
		return
	}

	// Проверяем токен/пароль
	valid := false
	if g.config.Auth.Mode == "token" && auth.Token == g.authToken {
		valid = true
	} else if g.config.Auth.Mode == "password" && auth.Password == g.config.Auth.Password {
		valid = true
	} else if g.config.Auth.Mode == "none" {
		valid = true
	}

	if !valid {
		g.sendAuthResponse(client, false, "", "Invalid credentials")
		return
	}

	// Создаём/получаем сессию
	sessionType := SessionTypeCLI
	if auth.ClientType == "web" {
		sessionType = SessionTypeWeb
	} else if auth.ClientType == "bot" {
		sessionType = SessionTypeDM
	}

	session, err := g.sessionManager.GetSession(auth.ClientID)
	if err != nil {
		session, _ = g.sessionManager.CreateSession(auth.ClientID, sessionType, auth.ClientType)
	}

	session.SetClientID(client.ID)

	client.ID = auth.ClientID
	client.Type = auth.ClientType
	client.SessionID = session.ID

	g.addClient(client)

	g.sendAuthResponse(client, true, session.ID, "")
	logger.Infof("Client %s authenticated, session: %s", client.ID, session.ID)
}

// handleChat обрабатывает сообщение чата
func (g *Gateway) handleChat(client *Client, msg *Message) {
	if client.SessionID == "" {
		g.sendError(client, 401, "Not authenticated", "")
		return
	}

	var chat ChatMessage
	if err := json.Unmarshal(msg.Payload, &chat); err != nil {
		g.sendError(client, 400, "Invalid chat payload", "")
		return
	}

	// Обновляем активность сессии
	session, _ := g.sessionManager.GetSession(client.SessionID)
	session.UpdateActivity()

	// Рассылаем событие
	g.Broadcast("chat_message", map[string]interface{}{
		"session_id": client.SessionID,
		"client_id":  client.ID,
		"content":    chat.Content,
		"timestamp":  time.Now().Unix(),
	})
}

// handleCommand обрабатывает команду
func (g *Gateway) handleCommand(client *Client, msg *Message) {
	if client.SessionID == "" {
		g.sendError(client, 401, "Not authenticated", "")
		return
	}

	var cmd CommandMessage
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		g.sendError(client, 400, "Invalid command payload", "")
		return
	}

	logger.Debugf("Command received: %s", cmd.Command)

	// Рассылаем событие
	g.Broadcast("command", map[string]interface{}{
		"session_id": client.SessionID,
		"command":    cmd.Command,
		"args":       cmd.Args,
	})
}

// handleSubscribe обрабатывает подписку
func (g *Gateway) handleSubscribe(client *Client, msg *Message) {
	var payload struct {
		Channel string `json:"channel"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		g.sendError(client, 400, "Invalid subscribe payload", "")
		return
	}

	client.mu.Lock()
	client.Subscriptions[payload.Channel] = true
	client.mu.Unlock()

	logger.Debugf("Client %s subscribed to %s", client.ID, payload.Channel)
}

// handleUnsubscribe обрабатывает отписку
func (g *Gateway) handleUnsubscribe(client *Client, msg *Message) {
	var payload struct {
		Channel string `json:"channel"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		g.sendError(client, 400, "Invalid unsubscribe payload", "")
		return
	}

	client.mu.Lock()
	delete(client.Subscriptions, payload.Channel)
	client.mu.Unlock()

	logger.Debugf("Client %s unsubscribed from %s", client.ID, payload.Channel)
}

// messageHandler обработчик сообщений в фоне
func (g *Gateway) messageHandler() {
	for {
		select {
		case msg := <-g.hub:
			// Обработка сообщений в фоне
			logger.Debugf("Processing message: %s", msg.Type)
		case <-g.ctx.Done():
			return
		}
	}
}

// addClient добавляет клиента
func (g *Gateway) addClient(client *Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.clients[client.ID] = client

	logger.Infof("Client connected: %s (type: %s), total: %d", client.ID, client.Type, len(g.clients))

	// Отправляем событие presence
	g.Broadcast("presence", map[string]interface{}{
		"event":       "client_connected",
		"client_id":   client.ID,
		"client_type": client.Type,
		"total":       len(g.clients),
	})
}

// removeClient удаляет клиента
func (g *Gateway) removeClient(client *Client) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.clients[client.ID]; ok {
		delete(g.clients, client.ID)
		close(client.Send)

		logger.Infof("Client disconnected: %s, total: %d", client.ID, len(g.clients))

		// Отправляем событие presence
		g.Broadcast("presence", map[string]interface{}{
			"event":     "client_disconnected",
			"client_id": client.ID,
			"total":     len(g.clients),
		})
	}
}

// sendAuthResponse отправляет ответ аутентификации
func (g *Gateway) sendAuthResponse(client *Client, success bool, sessionID, errMsg string) {
	resp := AuthResponse{
		Success:   success,
		SessionID: sessionID,
		Error:     errMsg,
	}

	payload, _ := json.Marshal(resp)

	msg := Message{
		Type:      MessageTypeAuthResponse,
		Timestamp: time.Now(),
		Payload:   json.RawMessage(payload),
	}

	data, _ := json.Marshal(msg)
	client.Send <- data
}

// sendError отправляет ошибку
func (g *Gateway) sendError(client *Client, code int, message, details string) {
	resp := ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	}

	payload, _ := json.Marshal(resp)

	msg := Message{
		Type:      MessageTypeError,
		Timestamp: time.Now(),
		Payload:   json.RawMessage(payload),
	}

	data, _ := json.Marshal(msg)
	client.Send <- data
}

// withAuth middleware аутентификации
func (g *Gateway) withAuth(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Для WebSocket аутентификация происходит в сообщении
		if r.URL.Path == "/ws" {
			handler.ServeHTTP(w, r)
			return
		}

		// Health check без аутентификации
		if r.URL.Path == "/health" {
			handler.ServeHTTP(w, r)
			return
		}

		// Проверяем токен
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if token != "Bearer "+g.authToken && token != g.authToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// handleHealth обработчик health check
func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"clients":       g.GetClientCount(),
		"auth_token":    g.authToken[:8] + "...",
		"uptime":        time.Now().Unix(),
	})
}

// handleSessions обработчик сессий
func (g *Gateway) handleSessions(w http.ResponseWriter, r *http.Request) {
	sessions := g.sessionManager.ListSessions()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// handleClients обработчик клиентов
func (g *Gateway) handleClients(w http.ResponseWriter, r *http.Request) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clients := make([]map[string]interface{}, 0, len(g.clients))
	for _, client := range g.clients {
		clients = append(clients, map[string]interface{}{
			"id":       client.ID,
			"type":     client.Type,
			"session":  client.SessionID,
			"active":   time.Since(client.LastActivity).Seconds() < 60,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"clients": clients,
		"total":   len(clients),
	})
}

// handleBroadcast обработчик broadcast
func (g *Gateway) handleBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Event string                 `json:"event"`
		Data  map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	g.Broadcast(payload.Event, payload.Data)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// generateClientID генерирует ID клиента
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}
