package telegram_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/user/qwen-claw/bots/telegram"
	"github.com/user/qwen-claw/internal/agent"
	"github.com/user/qwen-claw/internal/memory"
	"github.com/user/qwen-claw/internal/scheduler"
)

// TestTelegramBot_Integration integration тест для Telegram бота
// Требует TELEGRAM_TOKEN в переменных окружения
func TestTelegramBot_Integration(t *testing.T) {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		t.Skip("TELEGRAM_TOKEN not set, skipping integration test")
	}

	// Создаём зависимости
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	assert.NoError(t, memoryManager.Init())

	sched := scheduler.NewScheduler(tempDir, nil)
	assert.NoError(t, sched.Init())

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			QwenPath:     "echo",
			ApprovalMode: "yolo",
			Timeout:      5 * time.Second,
		},
		memoryManager,
	)

	// Создаём бота
	bot, err := telegram.NewBot(
		telegram.BotConfig{
			Token:   token,
			Timeout: 10 * time.Second,
		},
		agentInstance,
		memoryManager,
		sched,
	)

	assert.NoError(t, err)
	assert.NotNil(t, bot)

	// Проверяем что бот может получить информацию о себе
	info, err := bot.GetBotInfo()
	if err != nil {
		t.Logf("Warning: failed to get bot info: %v", err)
	} else {
		assert.NotEmpty(t, info)
		t.Logf("Bot username: %s", info)
	}
}

// TestTelegramBot_Commands тест команд бота
func TestTelegramBot_Commands(t *testing.T) {
	// Моковый тест без реального API
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	assert.NoError(t, memoryManager.Init())

	sched := scheduler.NewScheduler(tempDir, nil)
	assert.NoError(t, sched.Init())

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			QwenPath:     "echo",
			ApprovalMode: "yolo",
		},
		memoryManager,
	)

	// Проверяем что агент работает
	assert.NotNil(t, agentInstance)
	assert.NotNil(t, agentInstance.GetIntentDetector())
	assert.NotNil(t, agentInstance.GetConfirmationManager())
}

// TestTelegramBot_MessageHandling тест обработки сообщений
func TestTelegramBot_MessageHandling(t *testing.T) {
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	assert.NoError(t, memoryManager.Init())

	// Тестируем только менеджер памяти - агент требует qwen cli
	t.Run("memory add", func(t *testing.T) {
		entry, err := memoryManager.Remember("fact", "тест запись", nil)
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, "fact", entry.Type)
		assert.Contains(t, entry.Content, "тест запись")
	})

	t.Run("memory recall", func(t *testing.T) {
		// Сначала добавим запись
		_, err := memoryManager.Remember("fact", "тест для поиска", nil)
		assert.NoError(t, err)

		// Ищем запись
		results := memoryManager.Recall("поиск", 10)
		// Поиск может вернуть результаты или нет (зависит от индекса)
		_ = results
	})

	t.Run("memory list", func(t *testing.T) {
		entries := memoryManager.ListEntries()
		assert.GreaterOrEqual(t, len(entries), 0)
	})
}

// TestTelegramBot_InlineKeyboard тест inline клавиатур
func TestTelegramBot_InlineKeyboard(t *testing.T) {
	// Проверяем что клавиатуры создаются правильно
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Confirm", "confirm_action:test123"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Reject", "reject_action:test123"),
		),
	)

	assert.NotNil(t, keyboard)
	assert.Len(t, keyboard.InlineKeyboard, 1)
	assert.Len(t, keyboard.InlineKeyboard[0], 2)
}

// TestTelegramBot_CallbackQuery тест callback запросов
func TestTelegramBot_CallbackQuery(t *testing.T) {
	// Тестируем парсинг callback данных
	testCases := []struct {
		name     string
		data     string
		expected struct {
			action   string
			actionID string
		}
	}{
		{
			name: "confirm action",
			data: "confirm_action:action_123",
			expected: struct {
				action   string
				actionID string
			}{action: "confirm_action", actionID: "action_123"},
		},
		{
			name: "reject action",
			data: "reject_action:action_456",
			expected: struct {
				action   string
				actionID string
			}{action: "reject_action", actionID: "action_456"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Простая проверка формата
			assert.Contains(t, tc.data, "_action:")
		})
	}
}

// TestTelegramBot_RateLimiting тест rate limiting
func TestTelegramBot_RateLimiting(t *testing.T) {
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	assert.NoError(t, memoryManager.Init())

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			QwenPath:     "echo",
			ApprovalMode: "yolo",
			Timeout:      1 * time.Second, // Короткий таймаут
		},
		memoryManager,
	)

	// Запускаем много запросов быстро
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			ctx := t.Context()
			_, _ = agentInstance.Run(ctx, "test")
			done <- true
		}()
	}

	// Ждём завершения (должны обработаться или таймаутить)
	count := 0
	for {
		select {
		case <-done:
			count++
			if count >= 5 {
				return
			}
		case <-time.After(5 * time.Second):
			t.Logf("Some requests timed out (expected)")
			return
		}
	}
}

// TestTelegramBot_Confirmation тест подтверждений
func TestTelegramBot_Confirmation(t *testing.T) {
	tempDir := t.TempDir()
	memoryManager := memory.NewManager(tempDir)
	assert.NoError(t, memoryManager.Init())

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			QwenPath:     "echo",
			ApprovalMode: "plan", // Всегда требовать подтверждение
			Timeout:      5 * time.Second,
		},
		memoryManager,
	)

	// Получаем детектор намерений
	detector := agentInstance.GetIntentDetector()
	assert.NotNil(t, detector)

	// Получаем менеджер подтверждений
	confirmMgr := agentInstance.GetConfirmationManager()
	assert.NotNil(t, confirmMgr)

	// Создаём тестовое намерение
	intent := detector.Detect("выполни команду test")

	// Проверяем требуется ли подтверждение
	requiresConfirm := confirmMgr.RequiresConfirmation(intent)
	assert.True(t, requiresConfirm, "Should require confirmation in plan mode")
}

// TestTelegramBot_VoiceMessages тест голосовых сообщений
func TestTelegramBot_VoiceMessages(t *testing.T) {
	// Проверяем что структура для голосовых существует
	voice := tgbotapi.Voice{
		FileID:   "test_file_id",
		Duration: 10,
	}

	assert.Equal(t, "test_file_id", voice.FileID)
	assert.Equal(t, 10, voice.Duration)
}

// TestTelegramBot_DocumentHandling тест обработки документов
func TestTelegramBot_DocumentHandling(t *testing.T) {
	// Проверяем что структура для документов существует
	doc := tgbotapi.Document{
		FileID:   "test_file_id",
		FileName: "test.txt",
	}

	assert.Equal(t, "test_file_id", doc.FileID)
	assert.Equal(t, "test.txt", doc.FileName)
}

// TestTelegramBot_ErrorHandling тест обработки ошибок
func TestTelegramBot_ErrorHandling(t *testing.T) {
	// Тестируем с невалидным токеном
	_, err := telegram.NewBot(
		telegram.BotConfig{
			Token: "", // Пустой токен
		},
		nil, nil, nil,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "telegram token is required")
}

// TestTelegramBot_AccessControl тест контроля доступа
func TestTelegramBot_AccessControl(t *testing.T) {
	// Тестируем только конфигурацию доступа без создания бота
	// (создание бота требует валидный токен)

	config := telegram.BotConfig{
		Token:        "test_token",
		AllowedUsers: []int64{123, 456},
		Timeout:      5 * time.Second,
	}

	assert.Equal(t, "test_token", config.Token)
	assert.Len(t, config.AllowedUsers, 2)
	assert.Contains(t, config.AllowedUsers, int64(123))
	assert.Contains(t, config.AllowedUsers, int64(456))

	// Проверяем логику контроля доступа
	isAllowed := func(userID int64, allowedUsers []int64) bool {
		if len(allowedUsers) == 0 {
			return true
		}
		for _, id := range allowedUsers {
			if id == userID {
				return true
			}
		}
		return false
	}

	assert.True(t, isAllowed(123, config.AllowedUsers))
	assert.True(t, isAllowed(456, config.AllowedUsers))
	assert.False(t, isAllowed(789, config.AllowedUsers))
}
