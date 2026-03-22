package agent

import (
	"testing"
	"time"
)

func TestConfirmationManager_CreatePending(t *testing.T) {
	mgr := NewConfirmationManager(DefaultConfirmationManagerConfig())

	intent := &Intent{
		Type: IntentShellExec,
		Data: "ls -la",
	}

	action := mgr.CreatePending(intent, "выполни команду ls -la")

	if action == nil {
		t.Fatal("CreatePending() returned nil")
	}

	if action.ID == "" {
		t.Error("CreatePending() action ID is empty")
	}

	if action.Intent != intent {
		t.Error("CreatePending() intent not set correctly")
	}

	if action.Confirmed {
		t.Error("CreatePending() action should not be confirmed initially")
	}

	if action.Rejected {
		t.Error("CreatePending() action should not be rejected initially")
	}
}

func TestConfirmationManager_Confirm(t *testing.T) {
	mgr := NewConfirmationManager(DefaultConfirmationManagerConfig())

	intent := &Intent{Type: IntentShellExec, Data: "ls -la"}
	action := mgr.CreatePending(intent, "test")

	// Подтверждаем
	if !mgr.Confirm(action.ID) {
		t.Error("Confirm() should return true for valid action")
	}

	// Проверяем, что действие подтверждено
	updated := mgr.GetPending(action.ID)
	if updated == nil || !updated.Confirmed {
		t.Error("Action should be confirmed")
	}
}

func TestConfirmationManager_Reject(t *testing.T) {
	mgr := NewConfirmationManager(DefaultConfirmationManagerConfig())

	intent := &Intent{Type: IntentShellExec, Data: "ls -la"}
	action := mgr.CreatePending(intent, "test")

	// Отклоняем
	if !mgr.Reject(action.ID) {
		t.Error("Reject() should return true for valid action")
	}

	// Проверяем, что действие отклонено
	updated := mgr.GetPending(action.ID)
	if updated == nil || !updated.Rejected {
		t.Error("Action should be rejected")
	}
}

func TestConfirmationManager_RequiresConfirmation(t *testing.T) {
	tests := []struct {
		name string
		mode ConfirmationMode
		intent *Intent
		want   bool
	}{
		// ModeYolo - никогда не спрашиваем
		{"yolo - shell", ModeYolo, &Intent{Type: IntentShellExec}, false},
		{"yolo - file_write", ModeYolo, &Intent{Type: IntentFileWrite}, false},

		// ModePlan - всегда спрашиваем
		{"plan - shell", ModePlan, &Intent{Type: IntentShellExec}, true},
		{"plan - search", ModePlan, &Intent{Type: IntentSearch}, true},

		// ModeAutoEdit - спрашиваем для опасных
		{"auto-edit - shell с rm", ModeAutoEdit, &Intent{Type: IntentShellExec, Data: "rm -rf /tmp"}, true},
		{"auto-edit - file_write", ModeAutoEdit, &Intent{Type: IntentFileWrite}, true},
		{"auto-edit - search", ModeAutoEdit, &Intent{Type: IntentSearch}, false},
		{"auto-edit - memory_add", ModeAutoEdit, &Intent{Type: IntentMemoryAdd}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewConfirmationManager(ConfirmationManagerConfig{
				Mode: tt.mode,
			})

			got := mgr.RequiresConfirmation(tt.intent)
			if got != tt.want {
				t.Errorf("RequiresConfirmation(%v, %v) = %v, want %v", tt.mode, tt.intent.Type, got, tt.want)
			}
		})
	}
}

func TestConfirmationManager_IsExpired(t *testing.T) {
	mgr := NewConfirmationManager(ConfirmationManagerConfig{
		ExpirationTime: 100 * time.Millisecond,
	})

	intent := &Intent{Type: IntentShellExec, Data: "ls"}
	action := mgr.CreatePending(intent, "test")

	// Сразу не истекло
	if action.IsExpired() {
		t.Error("Action should not be expired immediately")
	}

	// Ждём истечения
	time.Sleep(150 * time.Millisecond)

	if !action.IsExpired() {
		t.Error("Action should be expired after timeout")
	}
}

func TestConfirmationManager_Cleanup(t *testing.T) {
	mgr := NewConfirmationManager(ConfirmationManagerConfig{
		ExpirationTime: 50 * time.Millisecond,
	})

	// Создаём несколько действий
	intent := &Intent{Type: IntentShellExec, Data: "ls"}
	action1 := mgr.CreatePending(intent, "test1")
	mgr.CreatePending(intent, "test2")

	// Подтверждаем одно
	mgr.Confirm(action1.ID)

	// Ждём истечения
	time.Sleep(100 * time.Millisecond)

	// Очищаем
	count := mgr.Cleanup()

	// Должно быть очищено至少 2 действия
	if count < 2 {
		t.Errorf("Cleanup() removed %d actions, want >= 2", count)
	}
}

func TestConfirmationManager_GetPendingActions(t *testing.T) {
	mgr := NewConfirmationManager(DefaultConfirmationManagerConfig())

	intent := &Intent{Type: IntentShellExec, Data: "ls"}
	action1 := mgr.CreatePending(intent, "test1")
	action2 := mgr.CreatePending(intent, "test2")

	// Одно подтверждаем
	mgr.Confirm(action1.ID)

	// Получаем ожидающие
	pending := mgr.GetPendingActions()

	// Должно остаться только одно ожидающее
	if len(pending) != 1 {
		t.Errorf("GetPendingActions() returned %d actions, want 1", len(pending))
	}

	if pending[0].ID != action2.ID {
		t.Error("GetPendingActions() should return only pending actions")
	}
}

func TestConfirmationManager_GetStats(t *testing.T) {
	mgr := NewConfirmationManager(DefaultConfirmationManagerConfig())

	intent := &Intent{Type: IntentShellExec, Data: "ls"}
	action1 := mgr.CreatePending(intent, "test1")
	action2 := mgr.CreatePending(intent, "test2")
	mgr.CreatePending(intent, "test3")

	// Подтверждаем одно, отклоняем другое
	mgr.Confirm(action1.ID)
	mgr.Reject(action2.ID)

	stats := mgr.GetStats()

	if stats.Confirmed != 1 {
		t.Errorf("GetStats().Confirmed = %d, want 1", stats.Confirmed)
	}

	if stats.Rejected != 1 {
		t.Errorf("GetStats().Rejected = %d, want 1", stats.Rejected)
	}

	if stats.Pending != 1 {
		t.Errorf("GetStats().Pending = %d, want 1", stats.Pending)
	}
}

func TestConfirmationManager_GetSetMode(t *testing.T) {
	mgr := NewConfirmationManager(DefaultConfirmationManagerConfig())

	if mgr.GetMode() != ModeAutoEdit {
		t.Errorf("GetMode() = %v, want %v", mgr.GetMode(), ModeAutoEdit)
	}

	mgr.SetMode(ModeYolo)

	if mgr.GetMode() != ModeYolo {
		t.Errorf("GetMode() after Set = %v, want %v", mgr.GetMode(), ModeYolo)
	}
}

func TestConfirmationManager_containsDangerousCommand(t *testing.T) {
	mgr := NewConfirmationManager(DefaultConfirmationManagerConfig())

	tests := []struct {
		name string
		data string
		want bool
	}{
		{"rm -rf", "rm -rf /tmp", true},
		{"rm -r", "rm -r directory", true},
		{"rm file", "rm file.txt", true},
		{"delete", "delete database", true},
		{"drop", "drop table users", true},
		{"chmod 777", "chmod 777 file", true},
		{"sudo", "sudo apt update", true},
		{"redirect >", "echo test > file.txt", true},
		{"redirect >>", "echo test >> file.txt", true},
		{"safe", "ls -la", false},
		{"cat", "cat file.txt", false},
		{"find", "find . -name test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgr.containsDangerousCommand(tt.data)
			if got != tt.want {
				t.Errorf("containsDangerousCommand(%q) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

func TestConfirmationManager_isDangerousIntent(t *testing.T) {
	mgr := NewConfirmationManager(DefaultConfirmationManagerConfig())

	tests := []struct {
		name   string
		intent *Intent
		want   bool
	}{
		{"file_write", &Intent{Type: IntentFileWrite, Data: "test.txt"}, true},
		{"shell_exec с rm", &Intent{Type: IntentShellExec, Data: "rm file"}, true},
		{"task_remove", &Intent{Type: IntentTaskRemove, Data: "task1"}, true},
		{"search", &Intent{Type: IntentSearch, Data: "query"}, false},
		{"memory_add", &Intent{Type: IntentMemoryAdd, Data: "fact"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgr.isDangerousIntent(tt.intent)
			if got != tt.want {
				t.Errorf("isDangerousIntent(%v) = %v, want %v", tt.intent.Type, got, tt.want)
			}
		})
	}
}
