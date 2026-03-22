package agent

import (
	"context"
	"testing"
	"time"
)

func TestQwenExecutor_Execute(t *testing.T) {
	tests := []struct {
		name        string
		config      QwenExecutorConfig
		command     string
		args        []string
		wantErr     bool
		description string
	}{
		{
			name: "базовое выполнение",
			config: QwenExecutorConfig{
				QwenPath:     "echo", // используем echo для теста
				ApprovalMode: "yolo",
				Timeout:      1 * time.Second,
			},
			command:     "test command",
			args:        []string{"arg1", "arg2"},
			wantErr:     false,
			description: "базовое выполнение команды",
		},
		{
			name: "с моделью",
			config: QwenExecutorConfig{
				QwenPath:     "echo",
				Model:        "test-model",
				ApprovalMode: "yolo",
				Timeout:      1 * time.Second,
			},
			command:     "test",
			args:        []string{},
			wantErr:     false,
			description: "выполнение с указанием модели",
		},
		{
			name: "с debug режимом",
			config: QwenExecutorConfig{
				QwenPath:     "echo",
				Debug:        true,
				ApprovalMode: "yolo",
				Timeout:      1 * time.Second,
			},
			command:     "test",
			args:        []string{},
			wantErr:     false,
			description: "выполнение в debug режиме",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewQwenExecutor(tt.config)

			ctx := context.Background()
			output, err := executor.Execute(ctx, tt.command, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("QwenExecutor.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && output == "" {
				t.Errorf("QwenExecutor.Execute() expected non-empty output")
			}
		})
	}
}

func TestQwenExecutor_RequiresConfirmation(t *testing.T) {
	tests := []struct {
		name         string
		approvalMode string
		action       string
		wantConfirm  bool
		description  string
	}{
		// Режим yolo - никогда не спрашиваем
		{
			name:         "yolo режим - безопасное действие",
			approvalMode: "yolo",
			action:       "создай файл",
			wantConfirm:  false,
			description:  "в режиме yolo подтверждения не требуются",
		},
		{
			name:         "yolo режим - опасное действие",
			approvalMode: "yolo",
			action:       "удали файл",
			wantConfirm:  false,
			description:  "в режиме yolo даже опасные действия не требуют подтверждения",
		},

		// Режим plan - всегда спрашиваем
		{
			name:         "plan режим - безопасное действие",
			approvalMode: "plan",
			action:       "покажи файл",
			wantConfirm:  true,
			description:  "в режиме plan всегда требуется подтверждение",
		},
		{
			name:         "plan режим - опасное действие",
			approvalMode: "plan",
			action:       "удали файл",
			wantConfirm:  true,
			description:  "в режиме plan всегда требуется подтверждение",
		},

		// Режим auto-edit - спрашиваем для опасных действий
		{
			name:         "auto-edit режим - безопасное действие",
			approvalMode: "auto-edit",
			action:       "покажи содержимое",
			wantConfirm:  false,
			description:  "в режиме auto-edit безопасные действия не требуют подтверждения",
		},
		{
			name:         "auto-edit режим - delete действие",
			approvalMode: "auto-edit",
			action:       "delete file test.txt",
			wantConfirm:  true,
			description:  "delete требует подтверждения",
		},
		{
			name:         "auto-edit режим - remove действие",
			approvalMode: "auto-edit",
			action:       "remove directory",
			wantConfirm:  true,
			description:  "remove требует подтверждения",
		},
		{
			name:         "auto-edit режим - rm команда",
			approvalMode: "auto-edit",
			action:       "rm -rf /tmp/test",
			wantConfirm:  true,
			description:  "rm требует подтверждения",
		},
		{
			name:         "auto-edit режим - chmod действие",
			approvalMode: "auto-edit",
			action:       "chmod 777 file",
			wantConfirm:  true,
			description:  "chmod требует подтверждения",
		},
		{
			name:         "auto-edit режим - sudo действие",
			approvalMode: "auto-edit",
			action:       "sudo apt update",
			wantConfirm:  true,
			description:  "sudo требует подтверждения",
		},
		{
			name:         "auto-edit режим - перенаправление вывода",
			approvalMode: "auto-edit",
			action:       "echo 'test' > file.txt",
			wantConfirm:  true,
			description:  "перенаправление вывода требует подтверждения",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewQwenExecutor(QwenExecutorConfig{
				ApprovalMode: tt.approvalMode,
			})

			got := executor.RequiresConfirmation(tt.action)
			if got != tt.wantConfirm {
				t.Errorf("QwenExecutor.RequiresConfirmation() = %v, want %v для действия '%s'", got, tt.wantConfirm, tt.action)
			}
		})
	}
}

func TestQwenExecutor_isDangerousAction(t *testing.T) {
	tests := []struct {
		name   string
		action string
		want   bool
	}{
		{"delete файл", "delete file.txt", true},
		{"remove директория", "remove directory", true},
		{"rm команда", "rm -rf /tmp/test", true},
		{"drop таблица", "drop table users", true},
		{"destroy сервер", "destroy server", true},
		{"format диск", "format /dev/sda", true},
		{"chmod права", "chmod 777 file", true},
		{"chown владелец", "chown root file", true},
		{"sudo команда", "sudo apt update", true},
		{"kill процесс", "kill 1234", true},
		{"terminate процесс", "terminate process", true},
		{"overwrite файл", "overwrite config", true},
		{"перенаправление >", "echo test > file.txt", true},
		{"перенаправление >>", "echo test >> file.txt", true},
		{"безопасное действие", "покажи файл", false},
		{"чтение файла", "cat file.txt", false},
		{"создание файла", "создай файл test.txt", false},
		{"поиск", "find . -name test", false},
	}

	executor := NewQwenExecutor(QwenExecutorConfig{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.isDangerousAction(tt.action)
			if got != tt.want {
				t.Errorf("QwenExecutor.isDangerousAction(%q) = %v, want %v", tt.action, got, tt.want)
			}
		})
	}
}

func TestQwenExecutor_Confirm(t *testing.T) {
	tests := []struct {
		name         string
		approvalMode string
		action       string
		want         bool
	}{
		{"yolo - подтверждение", "yolo", "любое действие", true},
		{"plan - подтверждение", "plan", "любое действие", false},
		{"auto-edit безопасное", "auto-edit", "покажи файл", true},
		{"auto-edit опасное", "auto-edit", "delete file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewQwenExecutor(QwenExecutorConfig{
				ApprovalMode: tt.approvalMode,
			})

			got := executor.Confirm(tt.action)
			if got != tt.want {
				t.Errorf("QwenExecutor.Confirm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQwenExecutor_GetSetApprovalMode(t *testing.T) {
	executor := NewQwenExecutor(QwenExecutorConfig{
		ApprovalMode: "auto-edit",
	})

	// Проверяем начальное значение
	if mode := executor.GetApprovalMode(); mode != "auto-edit" {
		t.Errorf("GetApprovalMode() = %v, want 'auto-edit'", mode)
	}

	// Меняем режим
	executor.SetApprovalMode("yolo")

	// Проверяем новое значение
	if mode := executor.GetApprovalMode(); mode != "yolo" {
		t.Errorf("GetApprovalMode() after Set = %v, want 'yolo'", mode)
	}
}

func TestQwenExecutor_GetSetTimeout(t *testing.T) {
	executor := NewQwenExecutor(QwenExecutorConfig{
		Timeout: 5 * time.Minute,
	})

	// Проверяем начальное значение
	if timeout := executor.GetTimeout(); timeout != 5*time.Minute {
		t.Errorf("GetTimeout() = %v, want 5m", timeout)
	}

	// Меняем таймаут
	executor.SetTimeout(10 * time.Second)

	// Проверяем новое значение
	if timeout := executor.GetTimeout(); timeout != 10*time.Second {
		t.Errorf("GetTimeout() after Set = %v, want 10s", timeout)
	}
}

func TestQwenExecutor_buildCommand(t *testing.T) {
	tests := []struct {
		name   string
		config QwenExecutorConfig
		command string
		args    []string
		want    []string
	}{
		{
			name: "базовая команда",
			config: QwenExecutorConfig{
				QwenPath:     "qwen",
				ApprovalMode: "auto-edit",
			},
			command: "test",
			args:    []string{"arg1", "arg2"},
			want:    []string{"--approval-mode", "auto-edit", "test", "arg1", "arg2"},
		},
		{
			name: "с моделью",
			config: QwenExecutorConfig{
				QwenPath:     "qwen",
				Model:        "gpt-4",
				ApprovalMode: "yolo",
			},
			command: "test",
			args:    []string{},
			want:    []string{"-m", "gpt-4", "--approval-mode", "yolo", "test"},
		},
		{
			name: "с debug",
			config: QwenExecutorConfig{
				QwenPath:     "qwen",
				Debug:        true,
				ApprovalMode: "plan",
			},
			command: "test",
			args:    []string{},
			want:    []string{"--approval-mode", "plan", "--debug", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewQwenExecutor(tt.config)
			got := executor.buildCommand(tt.command, tt.args)

			if len(got) != len(tt.want) {
				t.Errorf("buildCommand() length = %v, want %v", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("buildCommand()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestQwenExecutor_getCleanEnv(t *testing.T) {
	executor := NewQwenExecutor(QwenExecutorConfig{})

	env := executor.getCleanEnv()

	// Проверяем, что окружение не пустое
	if len(env) == 0 {
		t.Error("getCleanEnv() returned empty environment")
	}

	// Проверяем, что заблокированные переменные исключены
	blockedVars := []string{
		"QWEN_CLAW_TELEGRAM_TOKEN",
		"QWEN_CLAW_TELEGRAM_ALLOWED_USERS",
		"TELEGRAM_TOKEN",
		"API_KEY",
		"SECRET",
		"PASSWORD",
		"TOKEN",
	}

	for _, blocked := range blockedVars {
		for _, e := range env {
			if len(e) > len(blocked) && e[:len(blocked)+1] == blocked+"=" {
				t.Errorf("getCleanEnv() should not contain %s, but found: %s", blocked, e)
			}
		}
	}
}
