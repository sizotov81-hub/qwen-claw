package memory

import (
	"fmt"
	"strings"
	"time"
)

// CompactionConfig конфигурация компaction
type CompactionConfig struct {
	// MaxEvents максимальное количество событий перед компaction
	MaxEvents int

	// KeepLastN сколько последних событий сохранять
	KeepLastN int

	// AutoCompact автоматически выполнять компaction
	AutoCompact bool

	// SummaryPrefix префикс для суммаризации
	SummaryPrefix string
}

// DefaultCompactionConfig возвращает конфигурацию по умолчанию
func DefaultCompactionConfig() *CompactionConfig {
	return &CompactionConfig{
		MaxEvents:     100,
		KeepLastN:     20,
		AutoCompact:   true,
		SummaryPrefix: "📝 Summary:",
	}
}

// Compactor компaction сессий
type Compactor struct {
	config *CompactionConfig
}

// NewCompactor создаёт новый компactor
func NewCompactor(config *CompactionConfig) *Compactor {
	if config == nil {
		config = DefaultCompactionConfig()
	}
	return &Compactor{
		config: config,
	}
}

// ShouldCompact проверяет, нужна ли компaction
func (c *Compactor) ShouldCompact(store *SessionEventStore) bool {
	if !c.config.AutoCompact {
		return false
	}
	return store.GetEventCount() >= c.config.MaxEvents
}

// Compact выполняет компaction хранилища
func (c *Compactor) Compact(store *SessionEventStore, summaryGenerator func([]Event) string) error {
	if !c.ShouldCompact(store) {
		return nil
	}

	events := store.GetEvents()
	if len(events) <= c.config.KeepLastN {
		return nil
	}

	// Генерируем суммаризацию старых событий
	oldEvents := events[:len(events)-c.config.KeepLastN]
	summary := summaryGenerator(oldEvents)

	// Выполняем компaction
	if err := store.Compact(summary, c.config.KeepLastN); err != nil {
		return err
	}

	return nil
}

// SimpleSummaryGenerator простой генератор суммаризации
func SimpleSummaryGenerator(events []Event) string {
	if len(events) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📝 Summary of %d events from %s:\n\n",
		len(events),
		events[0].Timestamp.Format("2006-01-02 15:04"),
	))

	// Группируем события по типам
	userMessages := 0
	assistantReplies := 0
	toolCalls := 0

	for _, event := range events {
		switch event.Type {
		case EventUserMessage:
			userMessages++
		case EventAssistantReply:
			assistantReplies++
		case EventToolCall:
			toolCalls++
		}
	}

	if userMessages > 0 {
		sb.WriteString(fmt.Sprintf("- User messages: %d\n", userMessages))
	}
	if assistantReplies > 0 {
		sb.WriteString(fmt.Sprintf("- Assistant replies: %d\n", assistantReplies))
	}
	if toolCalls > 0 {
		sb.WriteString(fmt.Sprintf("- Tool calls: %d\n", toolCalls))
	}

	sb.WriteString(fmt.Sprintf("\nTotal tokens: ~%d\n", estimateTokens(events)))
	sb.WriteString(fmt.Sprintf("Time range: %s - %s\n",
		events[0].Timestamp.Format("15:04"),
		events[len(events)-1].Timestamp.Format("15:04"),
	))

	return sb.String()
}

// estimateTokens оценивает количество токенов
func estimateTokens(events []Event) int {
	total := 0
	for _, event := range events {
		total += event.Tokens
	}
	return total
}

// GetCompactionStatus возвращает статус компaction
type CompactionStatus struct {
	NeedsCompaction bool
	EventCount      int
	MaxEvents       int
	UsagePercent    float64
}

// GetCompactionStatus получает статус компaction
func (c *Compactor) GetCompactionStatus(store *SessionEventStore) CompactionStatus {
	eventCount := store.GetEventCount()
	usagePercent := float64(eventCount) / float64(c.config.MaxEvents)

	return CompactionStatus{
		NeedsCompaction: eventCount >= c.config.MaxEvents,
		EventCount:      eventCount,
		MaxEvents:       c.config.MaxEvents,
		UsagePercent:    usagePercent * 100,
	}
}

// FormatStatus форматирует статус для вывода
func (s CompactionStatus) FormatStatus() string {
	status := "🟢 Normal"
	if s.UsagePercent >= 50 {
		status = "🟡 Warning"
	}
	if s.UsagePercent >= 75 {
		status = "🟠 Critical"
	}
	if s.UsagePercent >= 90 {
		status = "🔴 Overflow"
	}

	return fmt.Sprintf("%s (%d/%d events, %.1f%%)",
		status,
		s.EventCount,
		s.MaxEvents,
		s.UsagePercent,
	)
}

// AutoCompactWithTimer автоматически выполняет компaction по таймеру
func (c *Compactor) AutoCompactWithTimer(store *SessionEventStore, summaryGenerator func([]Event) string, interval time.Duration) *time.Ticker {
	ticker := time.NewTicker(interval)

	go func() {
		for range ticker.C {
			if err := c.Compact(store, summaryGenerator); err != nil {
				// Логируем ошибку, но не останавливаем таймер
				fmt.Printf("Compaction error: %v\n", err)
			}
		}
	}()

	return ticker
}
