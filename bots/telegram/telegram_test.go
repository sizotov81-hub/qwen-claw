package telegram

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBotConfig(t *testing.T) {
	config := BotConfig{
		Token:        "test_token",
		AllowedUsers: []int64{123, 456},
		Timeout:      5 * time.Minute,
	}

	assert.Equal(t, "test_token", config.Token)
	assert.Len(t, config.AllowedUsers, 2)
	assert.Equal(t, int64(123), config.AllowedUsers[0])
}

func TestCreateHTTPClient(t *testing.T) {
	t.Run("no proxy", func(t *testing.T) {
		client, err := createHTTPClient()
		assert.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, 60*time.Second, client.Timeout)
	})
}

func TestBotStruct(t *testing.T) {
	bot := &Bot{
		config: BotConfig{
			Token: "test",
		},
		running: false,
	}

	assert.Equal(t, "test", bot.config.Token)
	assert.False(t, bot.running)
}

func TestRunningStr(t *testing.T) {
	bot := &Bot{running: true}
	assert.Equal(t, "🟢 Работает", bot.runningStr())

	bot.running = false
	assert.Equal(t, "🔴 Остановлен", bot.runningStr())
}

func TestSplitMessage(t *testing.T) {
	bot := &Bot{}

	t.Run("short message", func(t *testing.T) {
		parts := bot.splitMessage("short", 100)
		assert.Len(t, parts, 1)
		assert.Equal(t, "short", parts[0])
	})

	t.Run("long message", func(t *testing.T) {
		long := ""
		for i := 0; i < 500; i++ {
			long += "a"
		}

		parts := bot.splitMessage(long, 100)
		assert.Greater(t, len(parts), 1)

		for _, part := range parts {
			assert.LessOrEqual(t, len(part), 100)
		}
	})

	t.Run("message with newlines", func(t *testing.T) {
		msg := "line1\nline2\nline3\nline4\nline5"
		parts := bot.splitMessage(msg, 20)
		assert.Greater(t, len(parts), 1)
	})
}

func TestFormatResponse(t *testing.T) {
	bot := &Bot{}

	t.Run("escape markdown", func(t *testing.T) {
		input := "test_with_asterisks*and*brackets[test"
		output := bot.formatResponse(input)

		assert.Contains(t, output, "\\_")
		assert.Contains(t, output, "\\*")
		assert.Contains(t, output, "\\[")
	})

	t.Run("code blocks", func(t *testing.T) {
		input := "```go\ncode\n```"
		output := bot.formatResponse(input)
		assert.Contains(t, output, "```go\n")
	})
}

func TestAllowedUsers(t *testing.T) {
	config := BotConfig{
		Token:        "test",
		AllowedUsers: []int64{123, 456, 789},
	}

	assert.Contains(t, config.AllowedUsers, int64(123))
	assert.Contains(t, config.AllowedUsers, int64(456))
	assert.NotContains(t, config.AllowedUsers, int64(999))
}

func TestEmptyAllowedUsers(t *testing.T) {
	config := BotConfig{
		Token:        "test",
		AllowedUsers: []int64{},
	}

	assert.Empty(t, config.AllowedUsers)
	assert.Nil(t, config.AllowedUsers)
}

func TestTimeout(t *testing.T) {
	config := BotConfig{
		Token:   "test",
		Timeout: 10 * time.Minute,
	}

	assert.Equal(t, 10*time.Minute, config.Timeout)
}

func TestBotNotRunning(t *testing.T) {
	bot := &Bot{running: false}
	assert.False(t, bot.running)
}

func TestBotStop(t *testing.T) {
	bot := &Bot{running: true}
	bot.Stop()
	assert.False(t, bot.running)
}
