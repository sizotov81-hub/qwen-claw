package voice

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Whisper распознавание через whisper.cpp
type Whisper struct {
	// modelPath путь к модели
	modelPath string
	
	// whisperPath путь к бинарнику whisper
	whisperPath string
	
	// language язык распознавания
	language string
}

// NewWhisper создаёт распознаватель
func NewWhisper() *Whisper {
	return &Whisper{
		modelPath:   "/home/ss/qwen-claw/.qwen/voice/models/ggml-base.bin",
		whisperPath: "/home/ss/qwen-claw/.qwen/voice/whisper.cpp/main",
		language:    "ru",
	}
}

// IsInstalled проверяет установку whisper
func (w *Whisper) IsInstalled() bool {
	if _, err := os.Stat(w.whisperPath); err != nil {
		return false
	}
	if _, err := os.Stat(w.modelPath); err != nil {
		return false
	}
	return true
}

// Transcribe распознаёт голосовое сообщение
func (w *Whisper) Transcribe(voiceData []byte) (string, error) {
	// Сохраняем голосовое сообщение во временный файл
	tempDir := os.TempDir()
	oggFile := filepath.Join(tempDir, "voice_input.ogg")
	wavFile := filepath.Join(tempDir, "voice_input.wav")
	
	if err := os.WriteFile(oggFile, voiceData, 0644); err != nil {
		return "", fmt.Errorf("failed to save voice data: %w", err)
	}
	defer os.Remove(oggFile)
	
	// Конвертируем OGG в WAV (ffmpeg)
	if err := w.convertOGGtoWAV(oggFile, wavFile); err != nil {
		return "", fmt.Errorf("failed to convert OGG to WAV: %w", err)
	}
	defer os.Remove(wavFile)

	// Распознаём через whisper
	return w.runWhisper(wavFile)
}

// convertOGGtoWAV конвертирует OGG в WAV
func (w *Whisper) convertOGGtoWAV(oggFile, wavFile string) error {
	// Проверяем ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found")
	}
	
	cmd := exec.Command("ffmpeg",
		"-i", oggFile,
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		wavFile,
		"-y", // overwrite
		"-loglevel", "quiet",
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	
	return nil
}

// runWhisper запускает whisper.cpp
func (w *Whisper) runWhisper(wavFile string) (string, error) {
	cmd := exec.Command(w.whisperPath,
		"-m", w.modelPath,
		"-f", wavFile,
		"-l", w.language,
		"--no-timestamps",
	)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	
	return strings.TrimSpace(stdout.String()), nil
}

// InstallInstallationPlan возвращает план установки whisper
func InstallInstallationPlan() []string {
	return []string{
		"# Установить зависимости",
		"apt update && apt install -y build-essential cmake ffmpeg git",
		
		"# Клонировать whisper.cpp",
		"cd /tmp && git clone https://github.com/ggerganov/whisper.cpp.git",
		
		"# Скомпилировать",
		"cd whisper.cpp && make",
		
		"# Скачать модель",
		"cd whisper.cpp && ./models/download-ggml-model.sh base",
		
		"# Переместить в qwen-claw",
		"mkdir -p /home/ss/qwen-claw/.qwen/voice/whisper.cpp",
		"mv /tmp/whisper.cpp/main /home/ss/qwen-claw/.qwen/voice/whisper.cpp/",
		"mv /tmp/whisper.cpp/models/ggml-base.bin /home/ss/qwen-claw/.qwen/voice/models/",
	}
}

// GetInstallationCommands возвращает команды для установки
func GetInstallationCommands() []string {
	return []string{
		"apt update && apt install -y build-essential cmake ffmpeg git",
		"cd /tmp && git clone https://github.com/ggerganov/whisper.cpp.git",
		"cd /tmp/whisper.cpp && make",
		"cd /tmp/whisper.cpp && ./models/download-ggml-model.sh base",
		"mkdir -p /home/ss/qwen-claw/.qwen/voice/whisper.cpp",
		"mv /tmp/whisper.cpp/main /home/ss/qwen-claw/.qwen/voice/whisper.cpp/",
		"mkdir -p /home/ss/qwen-claw/.qwen/voice/models",
		"mv /tmp/whisper.cpp/models/ggml-base.bin /home/ss/qwen-claw/.qwen/voice/models/",
	}
}

// TranscribeSimple простая версия для тестов (через онлайн API)
func TranscribeSimple(voiceData []byte) (string, error) {
	// Временная заглушка - в будущем можно использовать онлайн API
	// Например, Google Speech-to-Text или Yandex SpeechKit
	
	return "", fmt.Errorf("online transcription not implemented yet")
}

// DownloadVoiceFile загружает файл голосового сообщения из Telegram
func DownloadVoiceFile(fileURL string) ([]byte, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	return io.ReadAll(resp.Body)
}
