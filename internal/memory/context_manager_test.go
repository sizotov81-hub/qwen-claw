package memory

import (
	"testing"
	"time"
)

func TestContextManager_AddToken(t *testing.T) {
	config := DefaultContextConfig()
	config.MaxTokens = 1000

	manager := NewContextManager(config)

	// Добавляем токен
	err := manager.AddToken(ContextToken{
		Content: "Hello world",
		Type:    "user",
	})

	if err != nil {
		t.Errorf("AddToken() error = %v", err)
	}

	usage := manager.GetUsage()
	if usage.CurrentTokens == 0 {
		t.Error("AddToken() should increase token count")
	}
}

func TestContextManager_GetStatus(t *testing.T) {
	config := DefaultContextConfig()
	config.MaxTokens = 100

	manager := NewContextManager(config)

	// Нормальный статус (< 50%)
	status := manager.GetStatus()
	if status != ContextStatusNormal {
		t.Errorf("GetStatus() = %v, want %v", status, ContextStatusNormal)
	}

	// Добавляем токены до предупреждения (50-75%)
	for i := 0; i < 6; i++ {
		manager.AddToken(ContextToken{
			Content: "test",
			Tokens:  10,
		})
	}

	status = manager.GetStatus()
	if status != ContextStatusWarning {
		t.Errorf("GetStatus() at 60%% = %v, want %v", status, ContextStatusWarning)
	}

	// Добавляем до критического (75-90%)
	for i := 0; i < 3; i++ {
		manager.AddToken(ContextToken{
			Content: "test",
			Tokens:  10,
		})
	}

	status = manager.GetStatus()
	if status != ContextStatusCritical {
		t.Errorf("GetStatus() at 90%% = %v, want %v", status, ContextStatusCritical)
	}
}

func TestContextManager_GetUsage(t *testing.T) {
	config := DefaultContextConfig()
	config.MaxTokens = 1000

	manager := NewContextManager(config)

	// Добавляем токены
	for i := 0; i < 5; i++ {
		manager.AddToken(ContextToken{
			Content: "test",
			Tokens:  100,
		})
	}

	usage := manager.GetUsage()

	if usage.CurrentTokens != 500 {
		t.Errorf("GetUsage().CurrentTokens = %d, want 500", usage.CurrentTokens)
	}

	if usage.MaxTokens != 1000 {
		t.Errorf("GetUsage().MaxTokens = %d, want 1000", usage.MaxTokens)
	}

	if usage.UsagePercent != 50.0 {
		t.Errorf("GetUsage().UsagePercent = %.2f, want 50.0", usage.UsagePercent)
	}

	if usage.AvailableTokens != 500 {
		t.Errorf("GetUsage().AvailableTokens = %d, want 500", usage.AvailableTokens)
	}
}

func TestContextManager_Compress(t *testing.T) {
	config := DefaultContextConfig()
	config.MaxTokens = 1000

	manager := NewContextManager(config)

	// Добавляем токены с разным приоритетом
	manager.AddToken(ContextToken{
		Content:  "Important",
		Priority: 10,
		Tokens:   100,
	})

	manager.AddToken(ContextToken{
		Content:  "Low priority old",
		Priority: 3,
		Tokens:   100,
		Timestamp: time.Now().Add(-15 * time.Minute),
	})

	manager.AddToken(ContextToken{
		Content:  "Medium priority",
		Priority: 5,
		Tokens:   100,
	})

	initialUsage := manager.GetUsage()

	// Сжимаем
	err := manager.Compress()
	if err != nil {
		t.Errorf("Compress() error = %v", err)
	}

	finalUsage := manager.GetUsage()

	// Должно удалиться меньше токенов
	if finalUsage.CurrentTokens >= initialUsage.CurrentTokens {
		t.Error("Compress() should reduce token count")
	}
}

func TestContextManager_GetIncompleteTasks(t *testing.T) {
	config := DefaultContextConfig()
	manager := NewContextManager(config)

	// Добавляем завершённые и незавершённые задачи
	manager.AddToken(ContextToken{
		Content:      "Task 1",
		IsIncomplete: false,
	})

	manager.AddToken(ContextToken{
		Content:      "Task 2",
		IsIncomplete: true,
	})

	manager.AddToken(ContextToken{
		Content:      "Task 3",
		IsIncomplete: true,
	})

	incomplete := manager.GetIncompleteTasks()

	if len(incomplete) != 2 {
		t.Errorf("GetIncompleteTasks() returned %d tasks, want 2", len(incomplete))
	}
}

func TestContextManager_GetTaskContext(t *testing.T) {
	config := DefaultContextConfig()
	manager := NewContextManager(config)

	// Добавляем токены для разных задач
	manager.AddToken(ContextToken{
		Content: "Task 1 message 1",
		TaskID:  "task-1",
	})

	manager.AddToken(ContextToken{
		Content: "Task 1 message 2",
		TaskID:  "task-1",
	})

	manager.AddToken(ContextToken{
		Content: "Task 2 message 1",
		TaskID:  "task-2",
	})

	task1Context := manager.GetTaskContext("task-1")
	if len(task1Context) != 2 {
		t.Errorf("GetTaskContext('task-1') returned %d tokens, want 2", len(task1Context))
	}

	task2Context := manager.GetTaskContext("task-2")
	if len(task2Context) != 1 {
		t.Errorf("GetTaskContext('task-2') returned %d tokens, want 1", len(task2Context))
	}
}

func TestContextManager_ClearTaskContext(t *testing.T) {
	config := DefaultContextConfig()
	manager := NewContextManager(config)

	// Добавляем токены
	manager.AddToken(ContextToken{
		Content: "Task 1",
		TaskID:  "task-1",
		Tokens:  100,
	})

	manager.AddToken(ContextToken{
		Content: "Task 2",
		TaskID:  "task-2",
		Tokens:  100,
	})

	initialUsage := manager.GetUsage()

	// Очищаем контекст task-1
	manager.ClearTaskContext("task-1")

	finalUsage := manager.GetUsage()

	// Должно удалиться 100 токенов
	if finalUsage.CurrentTokens != initialUsage.CurrentTokens-100 {
		t.Errorf("ClearTaskContext() should remove 100 tokens")
	}
}

func TestContextManager_Reset(t *testing.T) {
	config := DefaultContextConfig()
	manager := NewContextManager(config)

	// Добавляем токены
	for i := 0; i < 5; i++ {
		manager.AddToken(ContextToken{
			Content: "test",
			Tokens:  100,
		})
	}

	// Сбрасываем
	manager.Reset()

	usage := manager.GetUsage()
	if usage.CurrentTokens != 0 {
		t.Errorf("Reset() should clear all tokens, got %d", usage.CurrentTokens)
	}
}

func TestContextManager_Callbacks(t *testing.T) {
	config := DefaultContextConfig()
	config.MaxTokens = 100
	config.WarningThreshold = 0.5

	manager := NewContextManager(config)

	warningCalled := false
	manager.SetOnWarning(func(status ContextStatus, usage float64) {
		warningCalled = true
	})

	// Добавляем токены до предупреждения
	for i := 0; i < 6; i++ {
		manager.AddToken(ContextToken{
			Content: "test",
			Tokens:  10,
		})
	}

	if !warningCalled {
		t.Error("OnWarning callback should be called at warning level")
	}
}

func TestContextManager_EstimateTokens(t *testing.T) {
	text := "Hello world this is a test"
	estimated := EstimateTokens(text)

	// 28 символов / 4 = 7 токенов (примерно)
	if estimated == 0 {
		t.Error("EstimateTokens() should return non-zero value")
	}
}

func TestContextManager_MakeRoom(t *testing.T) {
	config := DefaultContextConfig()
	config.MaxTokens = 100
	config.AutoCompress = true

	manager := NewContextManager(config)

	// Добавляем токены до критического уровня
	for i := 0; i < 8; i++ {
		manager.AddToken(ContextToken{
			Content:  "test",
			Tokens:   10,
			Priority: 3, // Низкий приоритет
		})
	}

	// Пытаемся добавить ещё
	err := manager.AddToken(ContextToken{
		Content: "new token",
		Tokens:  50,
	})

	// Должно сработать сжатие или удаление старых
	if err != nil {
		t.Logf("AddToken() triggered cleanup: %v", err)
	}
}

func TestContextStatus_String(t *testing.T) {
	tests := []struct {
		status ContextStatus
		want   string
	}{
		{ContextStatusNormal, "normal"},
		{ContextStatusWarning, "warning"},
		{ContextStatusCritical, "critical"},
		{ContextStatusOverflow, "overflow"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("ContextStatus = %v, want %v", tt.status, tt.want)
			}
		})
	}
}
