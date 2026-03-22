package agent

import (
	"testing"
)

func TestIntentDetector_Detect(t *testing.T) {
	detector := NewIntentDetector()

	tests := []struct {
		name         string
		query        string
		wantType     IntentType
		wantConfidence float64
	}{
		// Запомнить
		{"сохрани", "сохрани в память: API ключ", IntentMemoryAdd, 0.9},
		{"remember", "remember this fact", IntentMemoryAdd, 0.9},

		// Поиск в памяти
		{"что помнишь", "что ты помнишь о базе данных", IntentMemorySearch, 0.9},

		// Запланировать задачу
		{"запланировать", "запланируй задачу backup на @daily", IntentTaskSchedule, 0.9},
		{"создать задачу", "создай задачу с командой echo hello", IntentTaskSchedule, 0.9},

		// Удалить задачу
		{"удалить задачу", "удали задачу task_123", IntentTaskRemove, 0.9},

		// Выполнить задачу
		{"выполнить задачу", "выполни задачу backup", IntentTaskRun, 0.9},

		// Прочитать файл
		{"прочитать файл", "прочитай файл config.yaml", IntentFileRead, 0.9},
		{"покажи файл", "покажи файл README.md", IntentFileRead, 0.9},

		// Записать файл
		{"записать файл", "запиши в файл test.txt содержимое", IntentFileWrite, 0.9},
		{"создать файл", "создай файл new.go", IntentFileWrite, 0.9},

		// Выполнить команду
		{"выполнить команду", "выполни команду ls -la", IntentShellExec, 0.9},
		{"запустить", "запусти тесты", IntentShellExec, 0.9},

		// Поиск
		{"найти", "найди информацию о Go", IntentSearch, 0.9},
		{"поиск", "поиск файлов с расширением .go", IntentSearch, 0.9},

		// Неизвестное
		{"неизвестное", "привет, как дела?", IntentUnknown, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := detector.Detect(tt.query)

			if intent.Type != tt.wantType {
				t.Errorf("Detect(%q) type = %v, want %v", tt.query, intent.Type, tt.wantType)
			}

			if intent.Confidence < tt.wantConfidence {
				t.Errorf("Detect(%q) confidence = %v, want >= %v", tt.query, intent.Confidence, tt.wantConfidence)
			}
		})
	}
}

func TestIntentDetector_RequiresSkillExecution(t *testing.T) {
	detector := NewIntentDetector()

	tests := []struct {
		name     string
		query    string
		wantExec bool
	}{
		{"сохрани", "сохрани факт", true},
		{"найти в памяти", "найди в памяти информацию", true},
		{"запланировать", "запланируй задачу", true},
		{"удалить задачу", "удали задачу task_123", true},
		{"выполнить задачу", "выполни задачу backup", true},
		{"прочитать файл", "прочитай файл config.yaml", true},
		{"создать файл", "создай файл test.txt", true},
		{"выполнить команду", "выполни команду ls -la", true},
		{"найти", "найди информацию", true},
		{"получить из интернета", "получи из интернета URL", true},
		{"уведомить", "уведоми меня о событии", true},
		{"неизвестное", "привет", false},
		{"вопрос", "как дела?", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := detector.Detect(tt.query)
			got := detector.RequiresSkillExecution(intent)

			if got != tt.wantExec {
				t.Errorf("RequiresSkillExecution(%q -> %v) = %v, want %v", tt.query, intent.Type, got, tt.wantExec)
			}
		})
	}
}

func TestIntentDetector_GetIntentDescription(t *testing.T) {
	detector := NewIntentDetector()

	tests := []struct {
		name     string
		intent   *Intent
		wantDesc string
	}{
		{"memory_add", &Intent{Type: IntentMemoryAdd}, "запомнить информацию"},
		{"task_schedule", &Intent{Type: IntentTaskSchedule}, "запланировать задачу"},
		{"unknown", &Intent{Type: IntentUnknown}, "неизвестное действие"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.GetIntentDescription(tt.intent)
			if got != tt.wantDesc {
				t.Errorf("GetIntentDescription() = %v, want %v", got, tt.wantDesc)
			}
		})
	}
}

func TestIntentDetector_DetectDataExtraction(t *testing.T) {
	detector := NewIntentDetector()

	tests := []struct {
		name     string
		query    string
		wantData string
	}{
		{"прочитать файл", "прочитай файл config.yaml", "config.yaml"},
		{"выполнить команду", "выполни команду ls -la", "ls -la"},
		{"запомни с двоеточием", "запомни: проект использует Go", "запомни: проект использует Go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := detector.Detect(tt.query)

			if intent.Data != tt.wantData {
				t.Errorf("Detect(%q).Data = %v, want %v", tt.query, intent.Data, tt.wantData)
			}
		})
	}
}
