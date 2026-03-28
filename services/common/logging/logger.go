// Package logging предоставляет расширенное логирование для микросервисов Qwen-Claw
package logging

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log     *zap.SugaredLogger
	once    sync.Once
	levelEn zapcore.Level
)

// Config конфигурация логгера
type Config struct {
	Level         string `json:"level"`
	Format        string `json:"format"` // json/console
	Output        string `json:"output"` // stdout/stderr/file
	FilePath      string `json:"file_path"`
	AddCaller     bool   `json:"add_caller"`
	AddStacktrace bool   `json:"add_stacktrace"`
	ServiceName   string `json:"service_name"`
}

// DefaultConfig конфигурация по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Level:         "info",
		Format:        "json",
		Output:        "stdout",
		AddCaller:     true,
		AddStacktrace: false,
		ServiceName:   os.Getenv("SERVICE_NAME"),
	}
}

// Init инициализирует логгер
func Init(cfg *Config) error {
	var err error
	once.Do(func() {
		if cfg == nil {
			cfg = DefaultConfig()
		}

		// Устанавливаем уровень логирования
		switch cfg.Level {
		case "debug":
			levelEn = zapcore.DebugLevel
		case "info":
			levelEn = zapcore.InfoLevel
		case "warn":
			levelEn = zapcore.WarnLevel
		case "error":
			levelEn = zapcore.ErrorLevel
		default:
			levelEn = zapcore.InfoLevel
		}

		// Создаём конфиг zap
		zapConfig := zap.Config{
			Level:            zap.NewAtomicLevelAt(levelEn),
			Development:      false,
			Encoding:         cfg.Format,
			EncoderConfig:    newEncoderConfig(cfg.Format),
			OutputPaths:      []string{cfg.Output},
			ErrorOutputPaths: []string{"stderr"},
		}

		// Добавляем caller
		if cfg.AddCaller {
			zapConfig.InitialFields = map[string]interface{}{
				"service": cfg.ServiceName,
			}
			zapConfig.EncoderConfig.CallerKey = "caller"
		}

		// Добавляем stacktrace для ошибок
		if cfg.AddStacktrace {
			zapConfig.EncoderConfig.StacktraceKey = "stacktrace"
		}

		var logger *zap.Logger
		logger, err = zapConfig.Build()
		if err == nil {
			log = logger.Sugar()
		}
	})

	return err
}

// newEncoderConfig создаёт конфигурацию энкодера
func newEncoderConfig(format string) zapcore.EncoderConfig {
	if format == "console" {
		return zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
	}

	// JSON формат
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// Debug логирует debug сообщение
func Debug(args ...interface{}) {
	log.Debug(args...)
}

// Debugf логирует debug сообщение с форматом
func Debugf(template string, args ...interface{}) {
	log.Debugf(template, args...)
}

// Debugw логирует debug сообщение с полями
func Debugw(msg string, keysAndValues ...interface{}) {
	log.Debugw(msg, keysAndValues...)
}

// Info логирует info сообщение
func Info(args ...interface{}) {
	log.Info(args...)
}

// Infof логирует info сообщение с форматом
func Infof(template string, args ...interface{}) {
	log.Infof(template, args...)
}

// Infow логирует info сообщение с полями
func Infow(msg string, keysAndValues ...interface{}) {
	log.Infow(msg, keysAndValues...)
}

// Warn логирует warn сообщение
func Warn(args ...interface{}) {
	log.Warn(args...)
}

// Warnf логирует warn сообщение с форматом
func Warnf(template string, args ...interface{}) {
	log.Warnf(template, args...)
}

// Warnw логирует warn сообщение с полями
func Warnw(msg string, keysAndValues ...interface{}) {
	log.Warnw(msg, keysAndValues...)
}

// Error логирует error сообщение
func Error(args ...interface{}) {
	log.Error(args...)
}

// Errorf логирует error сообщение с форматом
func Errorf(template string, args ...interface{}) {
	log.Errorf(template, args...)
}

// Errorw логирует error сообщение с полями
func Errorw(msg string, keysAndValues ...interface{}) {
	log.Errorw(msg, keysAndValues...)
}

// Fatal логирует fatal сообщение и завершает программу
func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

// Fatalf логирует fatal сообщение с форматом и завершает программу
func Fatalf(template string, args ...interface{}) {
	log.Fatalf(template, args...)
}

// Panic логирует panic сообщение и паникует
func Panic(args ...interface{}) {
	log.Panic(args...)
}

// Panicf логирует panic сообщение с форматом и паникует
func Panicf(template string, args ...interface{}) {
	log.Panicf(template, args...)
}

// Sync синхронизирует буферы
func Sync() error {
	return log.Sync()
}

// GetLogger возвращает sugared logger
func GetLogger() *zap.SugaredLogger {
	return log
}

// WithContext создаёт logger с контекстом
func WithContext(ctx context.Context, keysAndValues ...interface{}) *zap.SugaredLogger {
	if log == nil {
		return zap.NewNop().Sugar()
	}
	return log.With(keysAndValues...)
}

// With создаёт logger с полями
func With(keysAndValues ...interface{}) *zap.SugaredLogger {
	if log == nil {
		return zap.NewNop().Sugar()
	}
	return log.With(keysAndValues...)
}

// SetLevel устанавливает уровень логирования
func SetLevel(level string) {
	switch level {
	case "debug":
		levelEn = zapcore.DebugLevel
	case "info":
		levelEn = zapcore.InfoLevel
	case "warn":
		levelEn = zapcore.WarnLevel
	case "error":
		levelEn = zapcore.ErrorLevel
	}
}

// GetLevel возвращает текущий уровень логирования
func GetLevel() string {
	return levelEn.String()
}

// RequestID ключ для request ID в контексте
type RequestID string

const RequestIDKey RequestID = "request_id"

// WithRequestID добавляет request ID в logger
func WithRequestID(ctx context.Context, requestID string) *zap.SugaredLogger {
	return WithContext(ctx, string(RequestIDKey), requestID)
}

// GetRequestID извлекает request ID из контекста
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if rid, ok := ctx.Value(RequestIDKey).(string); ok {
		return rid
	}
	return ""
}

// NewRequestID генерирует новый request ID
func NewRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

func init() {
	// Инициализируем дефолтным логгером если Init не вызван
	once.Do(func() {
		zapConfig := zap.NewProductionConfig()
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		logger, _ := zapConfig.Build()
		log = logger.Sugar()
	})
}
