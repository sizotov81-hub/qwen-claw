package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, "info", cfg.Level)
	assert.Equal(t, "json", cfg.Format)
	assert.Equal(t, "stdout", cfg.Output)
	assert.True(t, cfg.AddCaller)
	assert.False(t, cfg.AddStacktrace)
}

func TestInit(t *testing.T) {
	cfg := &Config{
		Level:   "debug",
		Format:  "json",
		Output:  "stdout",
		AddCaller: true,
	}

	err := Init(cfg)
	assert.NoError(t, err)

	logger := GetLogger()
	require.NotNil(t, logger)
}

func TestInit_NilConfig(t *testing.T) {
	err := Init(nil)
	assert.NoError(t, err)

	logger := GetLogger()
	require.NotNil(t, logger)
}

func TestLogging(t *testing.T) {
	cfg := &Config{
		Level:   "debug",
		Format:  "json",
		Output:  "stdout",
		AddCaller: true,
	}

	err := Init(cfg)
	assert.NoError(t, err)

	t.Run("Debug logging", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Debug("test debug")
			Debugf("test %s", "debug")
			Debugw("test debug", "key", "value")
		})
	})

	t.Run("Info logging", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Info("test info")
			Infof("test %s", "info")
			Infow("test info", "key", "value")
		})
	})

	t.Run("Warn logging", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Warn("test warn")
			Warnf("test %s", "warn")
			Warnw("test warn", "key", "value")
		})
	})

	t.Run("Error logging", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Error("test error")
			Errorf("test %s", "error")
			Errorw("test error", "key", "value")
		})
	})

	t.Run("With fields", func(t *testing.T) {
		assert.NotPanics(t, func() {
			l := With("key", "value", "count", 42)
			l.Info("message with fields")
		})
	})

	t.Run("WithContext", func(t *testing.T) {
		assert.NotPanics(t, func() {
			ctx := context.Background()
			l := WithContext(ctx, "ctx_key", "ctx_value")
			l.Info("message with context")
		})
	})

	t.Run("Sync", func(t *testing.T) {
		// Sync может возвращать ошибку для некоторых типов вывода
		assert.NotPanics(t, func() {
			_ = Sync()
		})
	})
}

func TestSetLevel(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			SetLevel(level)
			assert.Equal(t, level, GetLevel())
		})
	}
}

func TestGetLogger(t *testing.T) {
	logger := GetLogger()
	require.NotNil(t, logger)
	assert.IsType(t, &zap.SugaredLogger{}, logger)
}

func TestWith(t *testing.T) {
	logger := With("key1", "value1", "key2", 42)
	require.NotNil(t, logger)

	// Проверяем что logger не nil и может логировать
	assert.NotPanics(t, func() {
		logger.Info("test")
	})
}

func TestWithContext(t *testing.T) {
	ctx := context.Background()
	logger := WithContext(ctx, "request_id", "123")
	require.NotNil(t, logger)

	assert.NotPanics(t, func() {
		logger.Info("test")
	})
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-123"

	logger := WithRequestID(ctx, requestID)
	require.NotNil(t, logger)

	assert.NotPanics(t, func() {
		logger.Info("test")
	})
}

func TestGetRequestID(t *testing.T) {
	t.Run("empty context", func(t *testing.T) {
		ctx := context.Background()
		rid := GetRequestID(ctx)
		assert.Equal(t, "", rid)
	})

	t.Run("with request ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RequestIDKey, "test-123")
		rid := GetRequestID(ctx)
		assert.Equal(t, "test-123", rid)
	})

	t.Run("nil context", func(t *testing.T) {
		rid := GetRequestID(nil)
		assert.Equal(t, "", rid)
	})
}

func TestNewRequestID(t *testing.T) {
	id1 := NewRequestID()
	id2 := NewRequestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "req-")
}

func TestFatalAndPanic(t *testing.T) {
	cfg := &Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}

	err := Init(cfg)
	assert.NoError(t, err)

	// Panic должен паниковать
	t.Run("Panic panics", func(t *testing.T) {
		assert.Panics(t, func() {
			Panic("test panic")
		})
	})

	// Fatal в тестах не вызываем - он завершает процесс
	t.Run("Fatal exists", func(t *testing.T) {
		// Проверяем что функция существует и не nil
		assert.NotNil(t, Fatal)
	})
}

func TestConsoleFormat(t *testing.T) {
	cfg := &Config{
		Level:   "debug",
		Format:  "console",
		Output:  "stdout",
		AddCaller: true,
	}

	err := Init(cfg)
	assert.NoError(t, err)

	assert.NotPanics(t, func() {
		Info("console format test")
	})
}

func TestConcurrentLogging(t *testing.T) {
	cfg := &Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}

	err := Init(cfg)
	assert.NoError(t, err)

	done := make(chan bool, 20)
	ctx := context.Background()

	for i := 0; i < 20; i++ {
		go func(n int) {
			Infof("concurrent message %d", n)
			Debugw("concurrent debug", "n", n)
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		select {
		case <-done:
			// OK
		case <-ctx.Done():
			t.Fatal("Timeout")
		}
	}
}
