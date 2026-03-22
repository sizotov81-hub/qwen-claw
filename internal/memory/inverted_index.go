package memory

import (
	"strings"
	"sync"
)

// InvertedIndex инвертированный индекс для быстрого поиска
type InvertedIndex struct {
	mu       sync.RWMutex
	index    map[string]map[string]bool // слово -> ID записей
	entries  map[string]*Entry          // ID -> запись
}

// NewInvertedIndex создаёт новый инвертированный индекс
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		index:   make(map[string]map[string]bool),
		entries: make(map[string]*Entry),
	}
}

// Add добавляет запись в индекс
func (idx *InvertedIndex) Add(entry *Entry) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.entries[entry.ID] = entry

	// Индексируем содержимое
	words := idx.tokenize(entry.Content)
	for _, word := range words {
		if idx.index[word] == nil {
			idx.index[word] = make(map[string]bool)
		}
		idx.index[word][entry.ID] = true
	}

	// Индексируем категорию
	if entry.Category != "" {
		word := strings.ToLower(entry.Category)
		if idx.index[word] == nil {
			idx.index[word] = make(map[string]bool)
		}
		idx.index[word][entry.ID] = true
	}

	// Индексируем тип
	word := strings.ToLower(entry.Type)
	if idx.index[word] == nil {
		idx.index[word] = make(map[string]bool)
	}
	idx.index[word][entry.ID] = true
}

// Remove удаляет запись из индекса
func (idx *InvertedIndex) Remove(entryID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	entry, ok := idx.entries[entryID]
	if !ok {
		return
	}

	// Удаляем из entries
	delete(idx.entries, entryID)

	// Удаляем из index
	words := idx.tokenize(entry.Content)
	for _, word := range words {
		if postings, ok := idx.index[word]; ok {
			delete(postings, entryID)
			if len(postings) == 0 {
				delete(idx.index, word)
			}
		}
	}
}

// Search ищет записи по запросу
func (idx *InvertedIndex) Search(query string) []*Entry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	words := idx.tokenize(query)
	if len(words) == 0 {
		return nil
	}

	// Находим пересечение всех слов
	resultIDs := make(map[string]bool)
	first := true

	for _, word := range words {
		if postings, ok := idx.index[word]; ok {
			if first {
				// Инициализируем первым набором
				for id := range postings {
					resultIDs[id] = true
				}
				first = false
			} else {
				// Пересекаем с текущим набором
				newIDs := make(map[string]bool)
				for id := range resultIDs {
					if postings[id] {
						newIDs[id] = true
					}
				}
				resultIDs = newIDs
			}
		} else {
			// Слово не найдено — пустой результат
			return nil
		}
	}

	// Собираем результаты
	results := make([]*Entry, 0, len(resultIDs))
	for id := range resultIDs {
		if entry, ok := idx.entries[id]; ok {
			results = append(results, entry)
		}
	}

	return results
}

// SearchPartial ищет частичные совпадения (медленнее но полнее)
func (idx *InvertedIndex) SearchPartial(query string) []*Entry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	query = strings.ToLower(query)
	results := make([]*Entry, 0)

	// Ищем по частичному совпадению слов
	for word, postings := range idx.index {
		if strings.Contains(word, query) || strings.Contains(query, word) {
			for id := range postings {
				if entry, ok := idx.entries[id]; ok {
					// Проверяем что ещё не добавили
					found := false
					for _, r := range results {
						if r.ID == id {
							found = true
							break
						}
					}
					if !found {
						results = append(results, entry)
					}
				}
			}
		}
	}

	return results
}

// Clear очищает индекс
func (idx *InvertedIndex) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.index = make(map[string]map[string]bool)
	idx.entries = make(map[string]*Entry)
}

// Size возвращает размер индекса
func (idx *InvertedIndex) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.entries)
}

// tokenize разбивает текст на слова
func (idx *InvertedIndex) tokenize(text string) []string {
	// Приводим к нижнему регистру
	text = strings.ToLower(text)

	// Разбиваем по не-алфавитным символам
	words := make([]string, 0)
	current := strings.Builder{}

	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= 'а' && r <= 'я') {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				word := current.String()
				if len(word) >= 2 { // Игнорируем слишком короткие слова
					words = append(words, word)
				}
				current.Reset()
			}
		}
	}

	// Добавляем последнее слово
	if current.Len() > 0 {
		word := current.String()
		if len(word) >= 2 {
			words = append(words, word)
		}
	}

	return words
}

// Rebuild перестраивает индекс из всех записей
func (idx *InvertedIndex) Rebuild(entries []*Entry) {
	idx.Clear()
	for _, entry := range entries {
		idx.Add(entry)
	}
}
