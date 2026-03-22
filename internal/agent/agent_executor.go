package agent

import (
	"context"
	"time"
)

// AgentExecutor адаптер для выполнения задач через агента
type AgentExecutor struct {
	agent *Agent
}

// NewAgentExecutor создаёт новый AgentExecutor
func NewAgentExecutor(agentInstance *Agent) *AgentExecutor {
	return &AgentExecutor{
		agent: agentInstance,
	}
}

// Execute выполняет команду через агента
func (e *AgentExecutor) Execute(ctx context.Context, command string) (string, error) {
	// Устанавливаем таймаут для выполнения задачи
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Выполняем команду через агента
	return e.agent.Run(ctx, command)
}

// GetAgent возвращает агента
func (e *AgentExecutor) GetAgent() *Agent {
	return e.agent
}
