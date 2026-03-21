package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	t.Run("Init with info level", func(t *testing.T) {
		err := Init("info")
		assert.NoError(t, err)
		assert.NotNil(t, log)
	})

	t.Run("Init with debug level", func(t *testing.T) {
		err := Init("debug")
		assert.NoError(t, err)
	})

	t.Run("Init with warn level", func(t *testing.T) {
		err := Init("warn")
		assert.NoError(t, err)
	})

	t.Run("Init with error level", func(t *testing.T) {
		err := Init("error")
		assert.NoError(t, err)
	})

	t.Run("Init with invalid level defaults to info", func(t *testing.T) {
		err := Init("invalid")
		assert.NoError(t, err)
	})
}

func TestLogging(t *testing.T) {
	err := Init("debug")
	assert.NoError(t, err)

	t.Run("Debug logging", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Debug("test debug")
			Debugf("test %s", "debug")
		})
	})

	t.Run("Info logging", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Info("test info")
			Infof("test %s", "info")
		})
	})

	t.Run("Warn logging", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Warn("test warn")
			Warnf("test %s", "warn")
		})
	})

	t.Run("Error logging", func(t *testing.T) {
		assert.NotPanics(t, func() {
			Error("test error")
			Errorf("test %s", "error")
		})
	})

	t.Run("With fields", func(t *testing.T) {
		assert.NotPanics(t, func() {
			logger := With("key", "value", "count", 42)
			logger.Info("message with fields")
		})
	})

	t.Run("Sync", func(t *testing.T) {
		err := Sync()
		assert.NoError(t, err)
	})

	t.Run("GetLogger", func(t *testing.T) {
		l := GetLogger()
		assert.NotNil(t, l)
	})
}
