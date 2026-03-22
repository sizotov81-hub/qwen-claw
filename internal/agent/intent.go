package agent

import (
	"regexp"
	"strings"
)

// IntentType тип намерения
type IntentType string

const (
	IntentUnknown       IntentType = "unknown"
	IntentMemoryAdd     IntentType = "memory_add"
	IntentMemorySearch  IntentType = "memory_search"
	IntentTaskSchedule  IntentType = "task_schedule"
	IntentTaskRemove    IntentType = "task_remove"
	IntentTaskRun       IntentType = "task_run"
	IntentFileRead      IntentType = "file_read"
	IntentFileWrite     IntentType = "file_write"
	IntentShellExec     IntentType = "shell_exec"
	IntentSearch        IntentType = "search"
	IntentWebFetch      IntentType = "web_fetch"
	IntentNotify        IntentType = "notify"
)

// Intent намерение пользователя
type Intent struct {
	// Type тип намерения
	Type IntentType `json:"type"`

	// Data данные для выполнения
	Data string `json:"data"`

	// Args дополнительные аргументы
	Args []string `json:"args"`

	// Confidence уверенность распознавания (0-1)
	Confidence float64 `json:"confidence"`
}

// IntentDetector детектор намерений
type IntentDetector struct {
	// patterns шаблоны для каждого типа намерения
	patterns map[IntentType][]*regexp.Regexp
}

// NewIntentDetector создаёт детектор намерений
func NewIntentDetector() *IntentDetector {
	detector := &IntentDetector{
		patterns: make(map[IntentType][]*regexp.Regexp),
	}

	// Инициализируем шаблоны
	detector.initPatterns()

	return detector
}

// initPatterns инициализирует шаблоны распознавания
func (d *IntentDetector) initPatterns() {
	// Запомнить в память
	d.patterns[IntentMemoryAdd] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^запомни\s+(.*)`),
		regexp.MustCompile(`(?i)^сохрани\s+(.*)`),
		regexp.MustCompile(`(?i)^добавь\s+в\s+память\s+(.*)`),
		regexp.MustCompile(`(?i)^remember\s+(.*)`),
		regexp.MustCompile(`(?i)^save\s+(.*)`),
	}

	// Поиск в памяти
	d.patterns[IntentMemorySearch] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^найди\s+в\s+памяти\s+(.*)`),
		regexp.MustCompile(`(?i)^покажи\s+из\s+памяти\s+(.*)`),
		regexp.MustCompile(`(?i)^что\s+ты\s+помнишь\s+о\s+(.*)`),
		regexp.MustCompile(`(?i)^recall\s+(.*)`),
		regexp.MustCompile(`(?i)^search\s+memory\s+(.*)`),
	}

	// Запланировать задачу
	d.patterns[IntentTaskSchedule] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^запланируй\s+(.*)`),
		regexp.MustCompile(`(?i)^создай\s+задачу\s+(.*)`),
		regexp.MustCompile(`(?i)^добавь\s+задачу\s+(.*)`),
		regexp.MustCompile(`(?i)^schedule\s+(.*)`),
		regexp.MustCompile(`(?i)^create\s+task\s+(.*)`),
	}

	// Удалить задачу
	d.patterns[IntentTaskRemove] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^удали\s+задачу\s+(.*)`),
		regexp.MustCompile(`(?i)^remove\s+task\s+(.*)`),
		regexp.MustCompile(`(?i)^delete\s+task\s+(.*)`),
	}

	// Выполнить задачу
	d.patterns[IntentTaskRun] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^выполни\s+задачу\s+(.*)`),
		regexp.MustCompile(`(?i)^запусти\s+задачу\s+(.*)`),
		regexp.MustCompile(`(?i)^run\s+task\s+(.*)`),
	}

	// Прочитать файл
	d.patterns[IntentFileRead] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^прочитай\s+файл\s+(.*)`),
		regexp.MustCompile(`(?i)^покажи\s+файл\s+(.*)`),
		regexp.MustCompile(`(?i)^read\s+file\s+(.*)`),
		regexp.MustCompile(`(?i)^cat\s+(.*)`),
	}

	// Записать файл
	d.patterns[IntentFileWrite] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^запиши\s+в\s+файл\s+(.*)`),
		regexp.MustCompile(`(?i)^создай\s+файл\s+(.*)`),
		regexp.MustCompile(`(?i)^write\s+to\s+file\s+(.*)`),
		regexp.MustCompile(`(?i)^create\s+file\s+(.*)`),
	}

	// Выполнить команду
	d.patterns[IntentShellExec] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^выполни\s+команду\s+(.*)`),
		regexp.MustCompile(`(?i)^запусти\s+(.*)`),
		regexp.MustCompile(`(?i)^exec\s+(.*)`),
		regexp.MustCompile(`(?i)^run\s+(.*)`),
		regexp.MustCompile(`(?i)^execute\s+(.*)`),
		regexp.MustCompile(`(?i)^выполни\s+(.*)`),
	}

	// Поиск
	d.patterns[IntentSearch] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^найди\s+(.*)`),
		regexp.MustCompile(`(?i)^поиск\s+(.*)`),
		regexp.MustCompile(`(?i)^search\s+(.*)`),
		regexp.MustCompile(`(?i)^find\s+(.*)`),
	}

	// Получить из интернета
	d.patterns[IntentWebFetch] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^получи\s+из\s+интернета\s+(.*)`),
		regexp.MustCompile(`(?i)^fetch\s+(.*)`),
		regexp.MustCompile(`(?i)^download\s+(.*)`),
	}

	// Уведомить
	d.patterns[IntentNotify] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^уведоми\s+меня\s+(.*)`),
		regexp.MustCompile(`(?i)^notify\s+me\s+(.*)`),
		regexp.MustCompile(`(?i)^отправь\s+уведомление\s+(.*)`),
	}
}

// Detect распознаёт намерение из запроса
func (d *IntentDetector) Detect(query string) *Intent {
	query = strings.TrimSpace(query)

	// Проверяем каждый тип намерения
	for intentType, patterns := range d.patterns {
		for _, pattern := range patterns {
			matches := pattern.FindStringSubmatch(query)
			if len(matches) >= 2 {
				return &Intent{
					Type:       intentType,
					Data:       matches[1],
					Args:       matches[2:],
					Confidence: 0.9, // Высокая уверенность при совпадении паттерна
				}
			}
		}
	}

	// Не нашли совпадений - неизвестное намерение
	return &Intent{
		Type:       IntentUnknown,
		Data:       query,
		Confidence: 0.5,
	}
}

// RequiresSkillExecution проверяет, требует ли намерение выполнения навыка
func (d *IntentDetector) RequiresSkillExecution(intent *Intent) bool {
	switch intent.Type {
	case IntentMemoryAdd, IntentMemorySearch,
		IntentTaskSchedule, IntentTaskRemove, IntentTaskRun,
		IntentFileRead, IntentFileWrite,
		IntentShellExec, IntentSearch, IntentWebFetch, IntentNotify:
		return true
	default:
		return false
	}
}

// GetIntentDescription возвращает описание намерения
func (d *IntentDetector) GetIntentDescription(intent *Intent) string {
	descriptions := map[IntentType]string{
		IntentUnknown:      "неизвестное действие",
		IntentMemoryAdd:    "запомнить информацию",
		IntentMemorySearch: "поиск в памяти",
		IntentTaskSchedule: "запланировать задачу",
		IntentTaskRemove:   "удалить задачу",
		IntentTaskRun:      "выполнить задачу",
		IntentFileRead:     "прочитать файл",
		IntentFileWrite:    "записать файл",
		IntentShellExec:    "выполнить команду",
		IntentSearch:       "поиск",
		IntentWebFetch:     "получить из интернета",
		IntentNotify:       "отправить уведомление",
	}

	if desc, ok := descriptions[intent.Type]; ok {
		return desc
	}
	return descriptions[IntentUnknown]
}
