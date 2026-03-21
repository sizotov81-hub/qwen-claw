package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.BaseDir)
	assert.NotEmpty(t, cfg.QwenDir)
	assert.NotEmpty(t, cfg.MemoryDir)
}

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := Default()
	cfg.BaseDir = tmpDir
	err := cfg.Save(configPath)
	assert.NoError(t, err)

	loaded, err := Load(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Equal(t, tmpDir, loaded.BaseDir)
}

func TestLoadNonExistent(t *testing.T) {
	cfg, err := Load("/nonexistent/config.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestEnsureDirs(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Default()
	cfg.BaseDir = filepath.Join(tmpDir, "test_base")
	cfg.QwenDir = filepath.Join(tmpDir, "test_qwen")
	cfg.MemoryDir = filepath.Join(tmpDir, "test_memory")

	err := cfg.EnsureDirs()
	assert.NoError(t, err)

	_, err = os.Stat(cfg.BaseDir)
	assert.NoError(t, err)
	_, err = os.Stat(cfg.QwenDir)
	assert.NoError(t, err)
	_, err = os.Stat(cfg.MemoryDir)
	assert.NoError(t, err)
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := Default()
	cfg.BaseDir = tmpDir
	err := cfg.Save(configPath)
	assert.NoError(t, err)

	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}

func TestParseAllowedUsers(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		ids := parseAllowedUsers("")
		assert.Nil(t, ids)
	})

	t.Run("single user", func(t *testing.T) {
		ids := parseAllowedUsers("123456")
		assert.Len(t, ids, 1)
		assert.Equal(t, int64(123456), ids[0])
	})

	t.Run("multiple users", func(t *testing.T) {
		ids := parseAllowedUsers("123,456,789")
		assert.Len(t, ids, 3)
		assert.Equal(t, int64(123), ids[0])
		assert.Equal(t, int64(456), ids[1])
		assert.Equal(t, int64(789), ids[2])
	})

	t.Run("with spaces", func(t *testing.T) {
		ids := parseAllowedUsers("123, 456 , 789")
		assert.Len(t, ids, 3)
	})

	t.Run("invalid numbers skipped", func(t *testing.T) {
		ids := parseAllowedUsers("123,abc,456")
		assert.Len(t, ids, 2)
	})
}

func TestLoadEnv(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	content := `QWEN_CLAW_TELEGRAM_TOKEN="test_token"
QWEN_CLAW_TELEGRAM_ALLOWED_USERS="123,456"
QWEN_CLAW_MODEL="test_model"
QWEN_CLAW_APPROVAL_MODE="plan"
QWEN_CLAW_DEBUG="true"
QWEN_CLAW_SERVER_PORT="9999"
`
	err := os.WriteFile(envPath, []byte(content), 0644)
	assert.NoError(t, err)

	os.Setenv("QWEN_CLAW_TELEGRAM_TOKEN", "test_token")
	os.Setenv("QWEN_CLAW_TELEGRAM_ALLOWED_USERS", "123,456")

	cfg, err := Load("")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	os.Unsetenv("QWEN_CLAW_TELEGRAM_TOKEN")
	os.Unsetenv("QWEN_CLAW_TELEGRAM_ALLOWED_USERS")
}

func TestServerConfig(t *testing.T) {
	cfg := Default()
	assert.NotNil(t, cfg.Server)
	assert.True(t, cfg.Server.Enabled)
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 18789, cfg.Server.Port)
}

func TestTelegramConfig(t *testing.T) {
	cfg := Default()
	assert.NotNil(t, cfg.Telegram)
	assert.False(t, cfg.Telegram.Enabled)
	assert.Empty(t, cfg.Telegram.Token)
}

func TestLLMConfig(t *testing.T) {
	cfg := Default()
	assert.NotNil(t, cfg.LLM)
	assert.Empty(t, cfg.LLM.Model)
	assert.Equal(t, "auto-edit", cfg.LLM.ApprovalMode)
	assert.False(t, cfg.LLM.Debug)
}
