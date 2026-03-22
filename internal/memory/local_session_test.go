package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalSessionManager_CreateSession(t *testing.T) {
	tempDir := t.TempDir()

	manager, err := NewLocalSessionManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	session, err := manager.CreateSession("test-session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.ID != "test-session" {
		t.Errorf("Session ID = %v, want 'test-session'", session.ID)
	}

	if session.ConversationHistory == nil {
		t.Error("ConversationHistory should not be nil")
	}

	if session.Context == nil {
		t.Error("Context should not be nil")
	}
}

func TestLocalSessionManager_GetSession(t *testing.T) {
	tempDir := t.TempDir()

	manager, err := NewLocalSessionManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Создаём сессию
	_, err = manager.CreateSession("test-session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Получаем сессию
	session, err := manager.GetSession("test-session")
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if session.ID != "test-session" {
		t.Errorf("Session ID = %v, want 'test-session'", session.ID)
	}
}

func TestLocalSession_AddToHistory(t *testing.T) {
	tempDir := t.TempDir()

	manager, err := NewLocalSessionManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	session, err := manager.CreateSession("test-session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Добавляем сообщения
	session.AddToHistory("User: Hello")
	session.AddToHistory("Assistant: Hi there")

	history := session.GetHistory()
	if len(history) != 2 {
		t.Errorf("GetHistory() returned %d items, want 2", len(history))
	}

	if history[0] != "User: Hello" {
		t.Errorf("First message = %v, want 'User: Hello'", history[0])
	}
}

func TestLocalSession_GetRecentHistory(t *testing.T) {
	tempDir := t.TempDir()

	manager, err := NewLocalSessionManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	session, err := manager.CreateSession("test-session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Добавляем 15 сообщений
	for i := 0; i < 15; i++ {
		session.AddToHistory(string(rune('A'+i)))
	}

	// Получаем последние 5
	recent := session.GetRecentHistory(5)
	if len(recent) != 5 {
		t.Errorf("GetRecentHistory(5) returned %d items, want 5", len(recent))
	}

	// Последние 5 должны быть K, L, M, N, O
	if recent[0] != "K" {
		t.Errorf("First recent = %v, want 'K'", recent[0])
	}
}

func TestLocalSession_Context(t *testing.T) {
	tempDir := t.TempDir()

	manager, err := NewLocalSessionManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	session, err := manager.CreateSession("test-session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Устанавливаем контекст
	session.SetContext("user_name", "Alice")
	session.SetContext("preference", "dark_mode")

	// Получаем контекст
	value, ok := session.GetContext("user_name")
	if !ok {
		t.Error("GetContext('user_name') should return true")
	}
	if value != "Alice" {
		t.Errorf("GetContext('user_name') = %v, want 'Alice'", value)
	}

	// Несуществующий ключ
	_, ok = session.GetContext("nonexistent")
	if ok {
		t.Error("GetContext('nonexistent') should return false")
	}
}

func TestLocalSession_Clear(t *testing.T) {
	tempDir := t.TempDir()

	manager, err := NewLocalSessionManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	session, err := manager.CreateSession("test-session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Добавляем данные
	session.AddToHistory("test")
	session.SetContext("key", "value")

	// Очищаем
	session.Clear()

	history := session.GetHistory()
	if len(history) != 0 {
		t.Errorf("After Clear(), history has %d items, want 0", len(history))
	}

	_, ok := session.GetContext("key")
	if ok {
		t.Error("After Clear(), context should be empty")
	}
}

func TestLocalSessionManager_DeleteSession(t *testing.T) {
	tempDir := t.TempDir()

	manager, err := NewLocalSessionManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Создаём сессию
	_, err = manager.CreateSession("test-session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Удаляем
	err = manager.DeleteSession("test-session")
	if err != nil {
		t.Errorf("DeleteSession() error = %v", err)
	}

	// Проверяем что файл удалён
	filePath := filepath.Join(tempDir, "test-session.json")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Session file should be deleted")
	}
}

func TestLocalSessionManager_ListSessions(t *testing.T) {
	tempDir := t.TempDir()

	manager, err := NewLocalSessionManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Создаём 3 сессии
	manager.CreateSession("session1")
	manager.CreateSession("session2")
	manager.CreateSession("session3")

	sessions := manager.ListSessions()
	if len(sessions) != 3 {
		t.Errorf("ListSessions() returned %d sessions, want 3", len(sessions))
	}
}

func TestLocalSessionManager_Cleanup(t *testing.T) {
	tempDir := t.TempDir()

	manager, err := NewLocalSessionManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Создаём сессию
	manager.CreateSession("old-session")

	// Ждём немного
	time.Sleep(100 * time.Millisecond)

	// Очищаем старые (старше 50ms)
	err = manager.Cleanup(50 * time.Millisecond)
	if err != nil {
		t.Errorf("Cleanup() error = %v", err)
	}

	sessions := manager.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("After cleanup, %d sessions remain, want 0", len(sessions))
	}
}

func TestLocalSession_SaveLoad(t *testing.T) {
	tempDir := t.TempDir()

	manager, err := NewLocalSessionManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	session, err := manager.CreateSession("persist-test")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Добавляем данные
	session.AddToHistory("Message 1")
	session.AddToHistory("Message 2")
	session.SetContext("key", "value")

	// Сохраняем
	session.save()

	// Создаём новый менеджер и загружаем сессию
	manager2, _ := NewLocalSessionManager(tempDir)
	loadedSession, err := manager2.GetSession("persist-test")
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	history := loadedSession.GetHistory()
	if len(history) != 2 {
		t.Errorf("Loaded history has %d items, want 2", len(history))
	}

	value, ok := loadedSession.GetContext("key")
	if !ok || value != "value" {
		t.Error("Loaded context is incorrect")
	}
}
