package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// StreamCallback функция обратного вызова для потокового вывода
type StreamCallback func(chunk string)

// StreamConfig конфигурация для потокового выполнения
type StreamConfig struct {
	// OnChunk вызывается для каждого чанка вывода
	OnChunk StreamCallback `json:"-"`

	// OnConfirm вызывается когда требуется подтверждение
	OnConfirm func(action *PendingAction) bool `json:"-"`

	// OnError вызывается при ошибке
	OnError func(err error) `json:"-"`

	// OnComplete вызывается при завершении
	OnComplete func(output string) `json:"-"`

	// ChunkSize размер чанка для чтения
	ChunkSize int `json:"chunk_size"`

	// FlushInterval интервал сброса буфера
	FlushInterval time.Duration `json:"flush_interval"`
}

// DefaultStreamConfig конфигурация по умолчанию
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		ChunkSize:     1,
		FlushInterval: 10 * time.Millisecond,
	}
}

// StreamExecutor выполняет команду с потоковым выводом
type StreamExecutor struct {
	config StreamConfig
}

// NewStreamExecutor создаёт новый StreamExecutor
func NewStreamExecutor(config StreamConfig) *StreamExecutor {
	if config.ChunkSize == 0 {
		config.ChunkSize = 1
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = 10 * time.Millisecond
	}
	return &StreamExecutor{
		config: config,
	}
}

// Execute выполняет команду с потоковым выводом
func (e *StreamExecutor) Execute(ctx context.Context, command string, args []string) (string, error) {
	qwenPath := "qwen"
	if path, err := exec.LookPath("qwen"); err == nil {
		qwenPath = path
	}

	cmdArgs := []string{}
	cmdArgs = append(cmdArgs, command)
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, qwenPath, cmdArgs...)

	// Получаем stdout и stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Читаем вывод потоково
	var output strings.Builder
	done := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(stdout)
		for {
			chunk := make([]byte, e.config.ChunkSize)
			n, err := reader.Read(chunk)
			if n > 0 {
				text := string(chunk[:n])
				output.WriteString(text)

				// Вызываем callback
				if e.config.OnChunk != nil {
					e.config.OnChunk(text)
				}

				// Проверяем на подтверждение
				if e.config.OnConfirm != nil && strings.Contains(text, "Подтверждение") {
					// Парсим ID действия из вывода
					actionID := e.parseActionID(output.String())
					if actionID != "" {
						action := &PendingAction{ID: actionID}
						if !e.config.OnConfirm(action) {
							break
						}
					}
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	// Читаем stderr
	go func() {
		reader := bufio.NewReader(stderr)
		for {
			chunk := make([]byte, e.config.ChunkSize)
			n, err := reader.Read(chunk)
			if n > 0 {
				text := string(chunk[:n])
				output.WriteString(text)

				if e.config.OnChunk != nil {
					e.config.OnChunk(text)
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				if e.config.OnError != nil {
					e.config.OnError(err)
				}
				return
			}
		}
	}()

	// Ждём завершения
	select {
	case err := <-done:
		if err != nil {
			return output.String(), err
		}
	case <-ctx.Done():
		cmd.Process.Kill()
		return output.String(), ctx.Err()
	}

	if err := cmd.Wait(); err != nil {
		return output.String(), err
	}

	if e.config.OnComplete != nil {
		e.config.OnComplete(output.String())
	}

	return output.String(), nil
}

// parseActionID извлекает ID действия из вывода
func (e *StreamExecutor) parseActionID(output string) string {
	// Ищем паттерн "ПОДТВЕРЖДАЮ action_..."
	parts := strings.Fields(output)
	for i, part := range parts {
		if strings.ToUpper(part) == "ПОДТВЕРЖДАЮ" && i+1 < len(parts) {
			return parts[i+1]
		}
		if strings.ToUpper(part) == "CONFIRM" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// AgentStream потоковый агент для вывода с эффектом печатной машинки
type AgentStream struct {
	agent *Agent
}

// NewAgentStream создаёт новый потоковый агент
func NewAgentStream(agentInstance *Agent) *AgentStream {
	return &AgentStream{
		agent: agentInstance,
	}
}

// Run выполняет запрос с потоковым выводом
func (s *AgentStream) Run(ctx context.Context, query string, callback StreamCallback) (string, error) {
	// Распознаём намерение
	intent := s.agent.GetIntentDetector().Detect(query)

	// Проверяем подтверждение
	if s.agent.GetConfirmationManager().RequiresConfirmation(intent) {
		action := s.agent.GetConfirmationManager().CreatePending(intent, query)

		// Формируем сообщение о подтверждении
		confirmMsg := fmt.Sprintf("⚠️ Требуется подтверждение для: %s\n\n",
			s.agent.GetIntentDetector().GetIntentDescription(intent))
		confirmMsg += fmt.Sprintf("Действие: %s\n", intent.Type)
		confirmMsg += fmt.Sprintf("Запрос: %s\n\n", query)
		confirmMsg += fmt.Sprintf("Напишите: ПОДТВЕРЖДАЮ %s", action.ID)

		// Выводим по буквам для эффекта
		for _, ch := range confirmMsg {
			if callback != nil {
				callback(string(ch))
			}
			time.Sleep(5 * time.Millisecond)
		}

		return confirmMsg, nil
	}

	// Выполняем намерение
	result, err := s.agent.executeIntent(ctx, intent)
	if err != nil {
		return "", err
	}

	// Выводим результат по буквам
	for _, ch := range result {
		if callback != nil {
			callback(string(ch))
		}
		time.Sleep(2 * time.Millisecond)
	}

	return result, nil
}

// TypewriterEffect выводит текст с эффектом печатной машинки
func TypewriterEffect(text string, callback StreamCallback, speed time.Duration) {
	if speed == 0 {
		speed = 10 * time.Millisecond
	}

	for _, ch := range text {
		if callback != nil {
			callback(string(ch))
		}
		time.Sleep(speed)
	}
}

// CLITypewriter выводит текст в CLI с эффектом печатной машинки
func CLITypewriter(text string, speed time.Duration) {
	if speed == 0 {
		speed = 15 * time.Millisecond
	}

	writer := bufio.NewWriter(os.Stdout)
	for _, ch := range text {
		writer.WriteRune(ch)
		writer.Flush()
		time.Sleep(speed)
	}
	fmt.Println()
}
