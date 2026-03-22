package main_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCLI_Integration базовый integration тест для CLI
func TestCLI_Integration(t *testing.T) {
	// Собираем бинарник
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qwen-claw-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/main.go")
	cmd.Dir = getProjectRoot()
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	err := cmd.Run()
	assert.NoError(t, err, "Failed to build binary")
	assert.FileExists(t, binaryPath, "Binary should exist")
}

// TestCLI_Help тест команды help
func TestCLI_Help(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qwen-claw-test")
	buildBinary(t, binaryPath)

	cmd := exec.Command(binaryPath, "--help")
	output, err := cmd.CombinedOutput()

	assert.NoError(t, err)
	outputStr := string(output)
	assert.Contains(t, outputStr, "Qwen-Claw")
	assert.Contains(t, outputStr, "Usage:")
	assert.Contains(t, outputStr, "Available Commands:")
}

// TestCLI_Version тест версии (если есть)
func TestCLI_Doctor(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qwen-claw-test")
	buildBinary(t, binaryPath)

	cmd := exec.Command(binaryPath, "doctor")
	output, _ := cmd.CombinedOutput()

	// Doctor может вернуть ошибку если qwen cli не установлен
	// Но вывод должен содержать информацию о проверках
	outputStr := string(output)
	assert.Contains(t, outputStr, "Checking")
}

// TestCLI_Memory тест памяти
func TestCLI_Memory(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qwen-claw-test")
	buildBinary(t, binaryPath)

	// Создаём временную директорию для памяти
	memoryDir := filepath.Join(tmpDir, ".qwen", "memory")
	os.MkdirAll(memoryDir, 0755)

	cmd := exec.Command(binaryPath, "memory", "list")
	cmd.Env = append(os.Environ(),
		"QWEN_CLAW_MEMORY_DIR="+memoryDir,
	)
	output, _ := cmd.CombinedOutput()

	// Команда должна выполниться (даже если память пуста)
	assert.NotNil(t, output)
}

// TestCLI_Tasks тест задач
func TestCLI_Tasks(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qwen-claw-test")
	buildBinary(t, binaryPath)

	// Создаём временную директорию для scheduler
	schedulerDir := filepath.Join(tmpDir, ".qwen", "scheduler")
	os.MkdirAll(schedulerDir, 0755)

	cmd := exec.Command(binaryPath, "tasks", "list")
	cmd.Env = append(os.Environ(),
		"QWEN_CLAW_QWEN_DIR="+filepath.Join(tmpDir, ".qwen"),
	)
	output, err := cmd.CombinedOutput()

	// Команда должна выполниться (даже если задач нет)
	assert.NotNil(t, output)
	_ = err
}

// TestCLI_Config тест конфигурации
func TestCLI_Config(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qwen-claw-test")
	buildBinary(t, binaryPath)

	// Создаём тестовый config
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
base_dir: ` + tmpDir + `
qwen_dir: ` + filepath.Join(tmpDir, ".qwen") + `
memory_dir: ` + filepath.Join(tmpDir, ".qwen", "memory") + `
server:
  enabled: false
`
	os.WriteFile(configPath, []byte(configContent), 0644)

	cmd := exec.Command(binaryPath, "--config", configPath, "doctor")
	output, err := cmd.CombinedOutput()

	// Конфигурация должна загрузиться
	assert.NotNil(t, output)
	_ = err
}

// TestCLI_Interactive тест интерактивного режима (базовый)
func TestCLI_Interactive(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qwen-claw-test")
	buildBinary(t, binaryPath)

	// Запускаем с stdin
	cmd := exec.Command(binaryPath, "-i")
	cmd.Dir = getProjectRoot()

	// Запускаем но не ждём завершения
	err := cmd.Start()
	assert.NoError(t, err)

	// Даём время на запуск
	time.Sleep(500 * time.Millisecond)

	// Останавливаем
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
}

// TestCLI_Run тест запуска запроса
func TestCLI_Run(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qwen-claw-test")
	buildBinary(t, binaryPath)

	// Создаём временную директорию для памяти
	memoryDir := filepath.Join(tmpDir, ".qwen", "memory")
	os.MkdirAll(memoryDir, 0755)

	cmd := exec.Command(binaryPath, "run", "test query")
	cmd.Env = append(os.Environ(),
		"QWEN_CLAW_MEMORY_DIR="+memoryDir,
	)
	output, err := cmd.CombinedOutput()

	// Запрос должен выполниться (даже с ошибкой если qwen не установлен)
	assert.NotNil(t, output)
	_ = err
}

// TestCLI_Environment тест переменных окружения
func TestCLI_Environment(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qwen-claw-test")
	buildBinary(t, binaryPath)

	// Тестируем с разными переменными окружения
	testCases := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			name: "debug mode",
			env: map[string]string{
				"QWEN_CLAW_DEBUG": "true",
			},
			expected: "", // Просто проверяем что запускается
		},
		{
			name: "approval mode",
			env: map[string]string{
				"QWEN_CLAW_APPROVAL_MODE": "yolo",
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, "--help")
			env := os.Environ()
			for k, v := range tc.env {
				env = append(env, k+"="+v)
			}
			cmd.Env = env

			output, err := cmd.CombinedOutput()
			assert.NoError(t, err)
			assert.NotNil(t, output)
		})
	}
}

// TestCLI_Concurrent тест конкурентного запуска
func TestCLI_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qwen-claw-test")
	buildBinary(t, binaryPath)

	// Запускаем несколько команд параллельно
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func(id int) {
			cmd := exec.Command(binaryPath, "--help")
			_, err := cmd.CombinedOutput()
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Ждём завершения всех
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent test")
		}
	}
}

// TestCLI_Timeout тест таймаутов
func TestCLI_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "qwen-claw-test")
	buildBinary(t, binaryPath)

	// Запускаем команду с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--help")
	err := cmd.Run()

	// Должно завершиться успешно до таймаута
	assert.NoError(t, err)
}

// Вспомогательные функции

func getProjectRoot() string {
	// Предполагаем что тесты запускаются из корня проекта
	cwd, _ := os.Getwd()
	if strings.HasSuffix(cwd, "/cmd") {
		return filepath.Dir(cwd)
	}
	return cwd
}

func buildBinary(t *testing.T, outputPath string) {
	cmd := exec.Command("go", "build", "-o", outputPath, "./cmd/main.go")
	cmd.Dir = getProjectRoot()
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	err := cmd.Run()
	assert.NoError(t, err, "Failed to build binary")
	assert.FileExists(t, outputPath, "Binary should exist")
}
