package logger

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log *zap.SugaredLogger
	once sync.Once
)

// Init инициализирует логгер
func Init(level string) error {
	var err error
	once.Do(func() {
		config := zap.NewProductionConfig()

		// Устанавливаем уровень логирования
		switch level {
		case "debug":
			config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		case "info":
			config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		case "warn":
			config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
		case "error":
			config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
		default:
			config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		}

		// Формат вывода
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.LevelKey = "level"
		config.EncoderConfig.NameKey = "logger"
		config.EncoderConfig.CallerKey = "caller"
		config.EncoderConfig.MessageKey = "msg"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

		var logger *zap.Logger
		logger, err = config.Build()
		if err == nil {
			log = logger.Sugar()
		}
	})
	return err
}

// Debug логирует debug сообщение
func Debug(args ...interface{}) {
	log.Debug(args...)
}

// Debugf логирует debug сообщение с форматом
func Debugf(template string, args ...interface{}) {
	log.Debugf(template, args...)
}

// Info логирует info сообщение
func Info(args ...interface{}) {
	log.Info(args...)
}

// Infof логирует info сообщение с форматом
func Infof(template string, args ...interface{}) {
	log.Infof(template, args...)
}

// Warn логирует warn сообщение
func Warn(args ...interface{}) {
	log.Warn(args...)
}

// Warnf логирует warn сообщение с форматом
func Warnf(template string, args ...interface{}) {
	log.Warnf(template, args...)
}

// Error логирует error сообщение
func Error(args ...interface{}) {
	log.Error(args...)
}

// Errorf логирует error сообщение с форматом
func Errorf(template string, args ...interface{}) {
	log.Errorf(template, args...)
}

// Fatal логирует fatal сообщение и завершает программу
func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

// Fatalf логирует fatal сообщение с форматом и завершает программу
func Fatalf(template string, args ...interface{}) {
	log.Fatalf(template, args...)
}

// With создаёт логгер с полями
func With(keysAndValues ...interface{}) *zap.SugaredLogger {
	return log.With(keysAndValues...)
}

// Sync синхронизирует буферы
func Sync() error {
	return log.Sync()
}

// GetLogger возвращает sugared logger
func GetLogger() *zap.SugaredLogger {
	return log
}

func init() {
	// Инициализируем дефолтным логгером если Init не вызван
	once.Do(func() {
		logger, _ := zap.NewProduction()
		log = logger.Sugar()
	})
}
