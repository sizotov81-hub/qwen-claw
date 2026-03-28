// Package main предоставляет endpoint /api/chat для API Gateway
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ChatHandler обработчик chat запросов
type ChatHandler struct {
	QwenWrapperURL string
	Timeout        time.Duration
}

// NewChatHandler создаёт обработчик
func NewChatHandler(qwenWrapperURL string, timeout time.Duration) *ChatHandler {
	if qwenWrapperURL == "" {
		qwenWrapperURL = "http://localhost:50052"
	}
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	return &ChatHandler{
		QwenWrapperURL: qwenWrapperURL,
		Timeout:        timeout,
	}
}

// ChatRequest запрос к чату
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Model     string `json:"model"`
}

// ChatResponse ответ чата
type ChatResponse struct {
	Response string `json:"response"`
}

// ServeHTTP обрабатывает HTTP запрос
func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим запрос
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	fmt.Printf("[CHAT] Request: session=%s, content=%q, model=%s\n", req.SessionID, req.Content, req.Model)

	// Отправляем запрос в Qwen Wrapper
	response, err := h.callQwenWrapper(req)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"response": fmt.Sprintf("❌ Error: %v", err),
		})
		return
	}

	fmt.Printf("[CHAT] Response: %q\n", response)

	json.NewEncoder(w).Encode(ChatResponse{Response: response})
}

// callQwenWrapper отправляет запрос в Qwen Wrapper Service
func (h *ChatHandler) callQwenWrapper(req ChatRequest) (string, error) {
	// Создаём запрос к Qwen Wrapper через gRPC (заглушка)
	// В реальной реализации нужно использовать gRPC клиент
	
	// Проверяем переменную окружения для тестового режима
	if os.Getenv("QWEN_WRAPPER_MOCK") == "true" {
		return h.mockResponse(req.Content), nil
	}

	// HTTP запрос к Qwen Wrapper (если есть HTTP endpoint)
	ctx, cancel := context.WithTimeout(context.Background(), h.Timeout)
	defer cancel()

	payload := map[string]interface{}{
		"query":   req.Content,
		"model":   req.Model,
		"session": req.SessionID,
	}

	jsonPayload, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", h.QwenWrapperURL+"/api/execute", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return h.mockResponse(req.Content), nil
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: h.Timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return h.mockResponse(req.Content), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

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

// mockResponse генерирует тестовый ответ
func (h *ChatHandler) mockResponse(content string) string {
	// Простые ответы на частые вопросы
	switch {
	case contains(content, "привет", "здравствуй", "hello"):
		return "👋 Здравствуйте! Я Qwen-Claw, ваш AI-помощник. Чем могу помочь?"
	
	case contains(content, "что умеешь", "возможности", "функци"):
		return `🤖 **Я умею:**

• Выполнять shell команды
• Работать с файлами (чтение, запись, поиск)
• Искать по коду
• Выполнять Git операции
• Делать HTTP запросы
• Запоминать информацию
• Выполнять задачи по расписанию

**Примеры запросов:**
- "создай файл test.txt"
- "покажи содержимое директории"
- "найди все .go файлы"
- "запомни что мой ник Alex"`
	
	case contains(content, "как дела", "как жизнь"):
		return "✨ У меня всё отлично! Я готов помочь вам с задачами. Что бы вы хотели сделать?"
	
	case contains(content, "кто ты", "что ты"):
		return "🤖 Я **Qwen-Claw** - AI-помощник с микросервисной архитектурой. Я работаю на базе Qwen Code CLI и предоставляю удобный интерфейс для взаимодействия с кодом и системой."
	
	default:
		return fmt.Sprintf("📨 Я получил ваш запрос: %q\n\nДля выполнения команды убедитесь, что Qwen Wrapper Service запущен и подключен.", content)
	}
}

// contains проверяет наличие подстрок в строке
func contains(s string, substrs ...string) bool {
	s = strings.ToLower(s)
	for _, substr := range substrs {
		if strings.Contains(s, strings.ToLower(substr)) {
			return true
		}
	}
	return false
}
