package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/user/qwen-claw/internal/memory"
)

func TestStreamExecutor_Execute(t *testing.T) {
	var output strings.Builder

	config := DefaultStreamConfig()
	config.OnChunk = func(chunk string) {
		output.WriteString(chunk)
	}

	executor := NewStreamExecutor(config)

	ctx := context.Background()
	_, err := executor.Execute(ctx, "echo", []string{"hello"})

	if err != nil {
		t.Logf("Execute() error = %v (это нормально если qwen не установлен)", err)
	}
}

func TestAgentStream_Run(t *testing.T) {
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "echo",
		Model:        "",
		ApprovalMode: "yolo",
		Timeout:      5 * time.Second,
		Debug:        false,
	}, memoryManager)

	stream := NewAgentStream(agentInstance)

	var output strings.Builder
	config := DefaultStreamConfig()
	config.OnChunk = func(chunk string) {
		output.WriteString(chunk)
	}

	ctx := context.Background()
	_, err := stream.Run(ctx, "запомни тестовый факт", config.OnChunk)

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Проверяем, что вывод не пустой
	if output.Len() == 0 {
		t.Error("Run() expected non-empty output")
	}
}

func TestTypewriterEffect(t *testing.T) {
	var output strings.Builder

	TypewriterEffect("test", func(chunk string) {
		output.WriteString(chunk)
	}, 1*time.Millisecond)

	if output.String() != "test" {
		t.Errorf("TypewriterEffect() = %v, want 'test'", output.String())
	}
}

func TestAgentStream_RequiresConfirmation(t *testing.T) {
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "echo",
		Model:        "",
		ApprovalMode: "plan", // Всегда требовать подтверждение
		Timeout:      5 * time.Second,
		Debug:        false,
	}, memoryManager)

	stream := NewAgentStream(agentInstance)

	var output strings.Builder
	config := DefaultStreamConfig()
	config.OnChunk = func(chunk string) {
		output.WriteString(chunk)
	}

	ctx := context.Background()
	result, err := stream.Run(ctx, "выполни команду ls", config.OnChunk)

	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	t.Logf("Result: %s", result)
	t.Logf("Output: %s", output.String())

	// Должно запросить подтверждение
	if !strings.Contains(result, "Подтверждение") && !strings.Contains(result, "confirmation") && !strings.Contains(result, "Требуется") {
		t.Error("Run() should request confirmation in ModePlan")
	}
}

func TestStreamExecutor_ParseActionID(t *testing.T) {
	executor := NewStreamExecutor(DefaultStreamConfig())

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"русский подтверждение", "Напишите: ПОДТВЕРЖДАЮ action_123", "action_123"},
		{"английский confirm", "Type: CONFIRM action_456", "action_456"},
		{"без действия", "Просто текст", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.parseActionID(tt.output)
			if got != tt.want {
				t.Errorf("parseActionID(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestDefaultStreamConfig(t *testing.T) {
	config := DefaultStreamConfig()

	if config.ChunkSize != 1 {
		t.Errorf("DefaultStreamConfig().ChunkSize = %v, want 1", config.ChunkSize)
	}

	if config.FlushInterval != 10*time.Millisecond {
		t.Errorf("DefaultStreamConfig().FlushInterval = %v, want 10ms", config.FlushInterval)
	}
}

func TestNewStreamExecutor(t *testing.T) {
	config := StreamConfig{
		ChunkSize:     5,
		FlushInterval: 50 * time.Millisecond,
	}

	executor := NewStreamExecutor(config)

	if executor.config.ChunkSize != 5 {
		t.Errorf("NewStreamExecutor().ChunkSize = %v, want 5", executor.config.ChunkSize)
	}

	if executor.config.FlushInterval != 50*time.Millisecond {
		t.Errorf("NewStreamExecutor().FlushInterval = %v, want 50ms", executor.config.FlushInterval)
	}
}

func TestNewStreamExecutor_Defaults(t *testing.T) {
	config := StreamConfig{}
	executor := NewStreamExecutor(config)

	if executor.config.ChunkSize != 1 {
		t.Errorf("NewStreamExecutor() default ChunkSize = %v, want 1", executor.config.ChunkSize)
	}

	if executor.config.FlushInterval != 10*time.Millisecond {
		t.Errorf("NewStreamExecutor() default FlushInterval = %v, want 10ms", executor.config.FlushInterval)
	}
}
