package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/user/qwen-claw/internal/memory"
	"github.com/user/qwen-claw/internal/scheduler"
)

func TestAgentExecutor_Execute(t *testing.T) {
	// Создаём менеджер памяти
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	// Создаём планировщик с nil executor (будет заменён)
	sched := scheduler.NewScheduler(tempDir, nil)

	// Создаём агента с scheduler
	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "echo",
		Model:        "",
		ApprovalMode: "yolo",
		Timeout:      5 * time.Second,
		Debug:        false,
	}, memoryManager)

	// Устанавливаем планировщик в агент
	agentInstance.SetScheduler(sched)

	// Создаём executor через агент
	executor := NewAgentExecutor(agentInstance)

	// Настраиваем executor в планировщике
	sched.SetExecutor(executor)

	// Проверяем выполнение команды
	ctx := context.Background()
	output, err := executor.Execute(ctx, "test command")

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if output == "" {
		t.Errorf("Execute() expected non-empty output")
	}
}

func TestScheduler_WithAgentExecutor(t *testing.T) {
	// Создаём временную директорию для тестов
	tempDir := t.TempDir()

	// Создаём менеджер памяти
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	// Создаём планировщик с nil executor
	sched := scheduler.NewScheduler(tempDir, nil)
	if err := sched.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Создаём агента
	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "echo",
		Model:        "",
		ApprovalMode: "yolo",
		Timeout:      5 * time.Second,
		Debug:        false,
	}, memoryManager)

	// Устанавливаем планировщик в агент
	agentInstance.SetScheduler(sched)

	// Создаём executor для агента
	agentExecutor := NewAgentExecutor(agentInstance)

	// Настраиваем executor в планировщике
	sched.SetExecutor(agentExecutor)

	// Запускаем планировщик
	sched.Start()
	defer sched.Stop()

	// Добавляем задачу
	task, err := sched.AddTask("test_task", "Test task", "hello world", "@every 1m")
	if err != nil {
		t.Fatalf("AddTask() error = %v", err)
	}

	// Проверяем, что задача добавлена
	if task.Name != "test_task" {
		t.Errorf("Task name = %v, want 'test_task'", task.Name)
	}

	// Выполняем задачу вручную
	result, err := sched.RunTaskNow(task.ID)
	if err != nil {
		t.Errorf("RunTaskNow() error = %v", err)
	}

	if result == nil {
		t.Error("RunTaskNow() expected non-nil result")
	}
}

func TestAgentExecutor_ContextTimeout(t *testing.T) {
	// Создаём менеджер памяти
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	// Создаём планировщик с nil executor
	sched := scheduler.NewScheduler(tempDir, nil)

	// Создаём агента с коротким таймаутом
	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "sleep",
		Model:        "",
		ApprovalMode: "yolo",
		Timeout:      100 * time.Millisecond,
		Debug:        false,
	}, memoryManager)

	// Устанавливаем планировщик в агент
	agentInstance.SetScheduler(sched)

	// Создаём executor
	executor := NewAgentExecutor(agentInstance)

	// Настраиваем executor в планировщике
	sched.SetExecutor(executor)

	// Проверяем таймаут
	ctx := context.Background()
	_, err := executor.Execute(ctx, "1") // sleep 1 second

	if err == nil {
		t.Error("Execute() expected timeout error")
	}
}

func TestAgent_ScheduleTask(t *testing.T) {
	// Создаём менеджер памяти
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	// Создаём планировщик
	sched := scheduler.NewScheduler(tempDir, nil)
	if err := sched.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Создаём агента
	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "echo",
		Model:        "",
		ApprovalMode: "yolo",
		Timeout:      5 * time.Second,
		Debug:        false,
	}, memoryManager)

	// Устанавливаем планировщик в агент
	agentInstance.SetScheduler(sched)

	// Планируем задачу
	task, err := agentInstance.ScheduleTask("test_task", "Test description", "hello", "@every 1h")
	if err != nil {
		t.Fatalf("ScheduleTask() error = %v", err)
	}

	if task.Name != "test_task" {
		t.Errorf("ScheduleTask() name = %v, want 'test_task'", task.Name)
	}
}

func TestAgent_GetTasks(t *testing.T) {
	// Создаём менеджер памяти
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	// Создаём планировщик
	sched := scheduler.NewScheduler(tempDir, nil)
	if err := sched.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Создаём агента
	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "echo",
		Model:        "",
		ApprovalMode: "yolo",
		Timeout:      5 * time.Second,
		Debug:        false,
	}, memoryManager)

	// Устанавливаем планировщик в агент
	agentInstance.SetScheduler(sched)

	// Добавляем задачу через планировщик
	_, err := sched.AddTask("test_task", "Test", "hello", "@every 1h")
	if err != nil {
		t.Fatalf("AddTask() error = %v", err)
	}

	// Получаем задачи через агента
	tasks := agentInstance.GetTasks()
	if len(tasks) != 1 {
		t.Errorf("GetTasks() returned %d tasks, want 1", len(tasks))
	}
}

func TestAgent_ProcessIntent(t *testing.T) {
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	sched := scheduler.NewScheduler(tempDir, nil)
	if err := sched.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "echo",
		Model:        "",
		ApprovalMode: "yolo",
		Timeout:      5 * time.Second,
		Debug:        false,
	}, memoryManager)

	agentInstance.SetScheduler(sched)

	ctx := context.Background()

	// Тест: запомнить (без подтверждения в режиме yolo)
	result, err := agentInstance.ProcessIntent(ctx, "запомни тестовый факт")
	if err != nil {
		t.Errorf("ProcessIntent() error = %v", err)
	}
	if !strings.Contains(result, "Запомнил") {
		t.Errorf("ProcessIntent() result = %v, want to contain 'Запомнил'", result)
	}

	// Тест: выполнить команду (требует подтверждения в режиме plan)
	agentInstance.SetConfirmationMode(ModePlan)
	result, err = agentInstance.ProcessIntent(ctx, "выполни команду ls -la")
	if err != nil {
		t.Errorf("ProcessIntent() error = %v", err)
	}
	// В режиме plan должна запросить подтверждение
	if !strings.Contains(result, "Требуется подтверждение") && !strings.Contains(result, "confirmation") {
		t.Logf("ProcessIntent() result = %v", result)
		t.Error("ProcessIntent() should request confirmation in ModePlan")
	}
}

func TestAgent_ConfirmAction(t *testing.T) {
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

	// Создаём намерение
	intent := agentInstance.GetIntentDetector().Detect("запомни: важный факт")

	// Создаём ожидающее действие
	action := agentInstance.GetConfirmationManager().CreatePending(intent, "запомни: важный факт")

	// Подтверждаем действие
	result, err := agentInstance.ConfirmAction(action.ID)
	if err != nil {
		t.Errorf("ConfirmAction() error = %v", err)
	}
	if !strings.Contains(result, "выполнено") {
		t.Errorf("ConfirmAction() result = %v, want to contain 'выполнено'", result)
	}
}

func TestAgent_RejectAction(t *testing.T) {
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "echo",
		Model:        "",
		ApprovalMode: "plan",
		Timeout:      5 * time.Second,
		Debug:        false,
	}, memoryManager)

	intent := agentInstance.GetIntentDetector().Detect("запомни: факт")
	action := agentInstance.GetConfirmationManager().CreatePending(intent, "test")

	// Отклоняем действие
	err := agentInstance.RejectAction(action.ID)
	if err != nil {
		t.Errorf("RejectAction() error = %v", err)
	}

	// Проверяем, что действие отклонено
	updated := agentInstance.GetConfirmationManager().GetPending(action.ID)
	if updated == nil || !updated.Rejected {
		t.Error("Action should be rejected")
	}
}

func TestAgent_GetPendingActions(t *testing.T) {
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "echo",
		Model:        "",
		ApprovalMode: "plan",
		Timeout:      5 * time.Second,
		Debug:        false,
	}, memoryManager)

	// Создаём несколько действий
	intent := agentInstance.GetIntentDetector().Detect("запомни: факт")
	agentInstance.GetConfirmationManager().CreatePending(intent, "test1")
	agentInstance.GetConfirmationManager().CreatePending(intent, "test2")

	actions := agentInstance.GetPendingActions()
	if len(actions) != 2 {
		t.Errorf("GetPendingActions() returned %d actions, want 2", len(actions))
	}
}

func TestAgent_SetConfirmationMode(t *testing.T) {
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "echo",
		Model:        "",
		ApprovalMode: "auto-edit",
		Timeout:      5 * time.Second,
		Debug:        false,
	}, memoryManager)

	// Проверяем режим по умолчанию
	if agentInstance.GetConfirmationMode() != ModeAutoEdit {
		t.Errorf("GetConfirmationMode() = %v, want %v", agentInstance.GetConfirmationMode(), ModeAutoEdit)
	}

	// Меняем режим
	agentInstance.SetConfirmationMode(ModeYolo)

	if agentInstance.GetConfirmationMode() != ModeYolo {
		t.Errorf("GetConfirmationMode() after Set = %v, want %v", agentInstance.GetConfirmationMode(), ModeYolo)
	}
}

func TestAgent_ExecuteCommand(t *testing.T) {
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

	ctx := context.Background()
	output, err := agentInstance.ExecuteCommand(ctx, "test", nil)
	if err != nil {
		t.Errorf("ExecuteCommand() error = %v", err)
	}
	if output == "" {
		t.Error("ExecuteCommand() expected non-empty output")
	}
}

func TestAgent_SetApprovalMode(t *testing.T) {
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	if err := memoryManager.Init(); err != nil {
		t.Fatalf("Failed to init memory: %v", err)
	}

	agentInstance := NewAgent(AgentConfig{
		QwenPath:     "echo",
		Model:        "",
		ApprovalMode: "auto-edit",
		Timeout:      5 * time.Second,
		Debug:        false,
	}, memoryManager)

	if agentInstance.GetApprovalMode() != "auto-edit" {
		t.Errorf("GetApprovalMode() = %v, want 'auto-edit'", agentInstance.GetApprovalMode())
	}

	agentInstance.SetApprovalMode("yolo")

	if agentInstance.GetApprovalMode() != "yolo" {
		t.Errorf("GetApprovalMode() after Set = %v, want 'yolo'", agentInstance.GetApprovalMode())
	}
}
