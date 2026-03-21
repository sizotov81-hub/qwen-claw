package voice

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWhisper(t *testing.T) {
	whisper := NewWhisper()
	assert.NotNil(t, whisper)
	assert.Equal(t, "ru", whisper.language)
	assert.NotEmpty(t, whisper.modelPath)
	assert.NotEmpty(t, whisper.whisperPath)
}

func TestIsInstalled(t *testing.T) {
	whisper := NewWhisper()

	installed := whisper.IsInstalled()
	assert.False(t, installed, "whisper should not be installed by default")
}

func TestGetInstallationCommands(t *testing.T) {
	commands := GetInstallationCommands()
	assert.NotEmpty(t, commands)
	assert.GreaterOrEqual(t, len(commands), 5)

	assert.Contains(t, commands[0], "apt update")
	assert.Contains(t, commands[1], "git clone")
	assert.Contains(t, commands[2], "make")
}

func TestInstallInstallationPlan(t *testing.T) {
	plan := InstallInstallationPlan()
	assert.NotEmpty(t, plan)
	assert.GreaterOrEqual(t, len(plan), 5)
}

func TestConvertOGGtoWAV(t *testing.T) {
	whisper := NewWhisper()

	t.Run("ffmpeg not found", func(t *testing.T) {
		err := whisper.convertOGGtoWAV("/tmp/test.ogg", "/tmp/test.wav")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ffmpeg not found")
	})
}

func TestRunWhisper(t *testing.T) {
	whisper := NewWhisper()

	t.Run("whisper not installed", func(t *testing.T) {
		_, err := whisper.runWhisper("/tmp/test.wav")
		assert.Error(t, err)
	})
}

func TestTranscribe(t *testing.T) {
	whisper := NewWhisper()

	t.Run("whisper not installed", func(t *testing.T) {
		_, err := whisper.Transcribe([]byte("test voice data"))
		assert.Error(t, err)
	})
}

func TestDownloadVoiceFile(t *testing.T) {
	t.Run("invalid url", func(t *testing.T) {
		_, err := DownloadVoiceFile("invalid://url")
		assert.Error(t, err)
	})

	t.Run("nonexistent url", func(t *testing.T) {
		_, err := DownloadVoiceFile("http://nonexistent.invalid/file.ogg")
		assert.Error(t, err)
	})
}

func TestWhisperPaths(t *testing.T) {
	whisper := NewWhisper()

	assert.Contains(t, whisper.modelPath, ".qwen")
	assert.Contains(t, whisper.modelPath, "models")
	assert.Contains(t, whisper.modelPath, "ggml-base.bin")

	assert.Contains(t, whisper.whisperPath, ".qwen")
	assert.Contains(t, whisper.whisperPath, "whisper.cpp")
	assert.Contains(t, whisper.whisperPath, "main")
}

func TestTranscribeSimple(t *testing.T) {
	_, err := TranscribeSimple([]byte("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestFileOperations(t *testing.T) {
	tmpFile := "/tmp/test_voice.ogg"

	t.Run("create voice file", func(t *testing.T) {
		data := []byte("fake ogg data")
		err := os.WriteFile(tmpFile, data, 0644)
		assert.NoError(t, err)
	})

	t.Run("read voice file", func(t *testing.T) {
		data, err := os.ReadFile(tmpFile)
		assert.NoError(t, err)
		assert.Equal(t, []byte("fake ogg data"), data)
	})

	t.Run("cleanup", func(t *testing.T) {
		os.Remove(tmpFile)
		_, err := os.Stat(tmpFile)
		assert.Error(t, err)
	})
}
