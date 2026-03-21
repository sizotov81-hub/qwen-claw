package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SearchResult результат поиска
type SearchResult struct {
	Query       string        `json:"query"`
	Found       bool          `json:"found"`
	Entries     []*Entry      `json:"entries"`
	Source      string        `json:"source"` // working|semantic|associative|web
	Latency     time.Duration `json:"latency"`
	Suggestion  string        `json:"suggestion,omitempty"`
	WebContent  string        `json:"web_content,omitempty"`
}

// SearchContext контекст поиска
type SearchContext struct {
	UserID      string    `json:"user_id"`
	SessionID   string    `json:"session_id"`
	CurrentTask string    `json:"current_task"`
	Timestamp   time.Time `json:"timestamp"`
}

// Find главный метод поиска (всегда вызывается первым)
func (m *Manager) Find(query string, context *SearchContext) *SearchResult {
	startTime := time.Now()
	
	if context == nil {
		context = &SearchContext{
			Timestamp: time.Now(),
		}
	}
	
	result := &SearchResult{
		Query: query,
		Entries: make([]*Entry, 0),
	}
	
	// УРОВЕНЬ 1: Working Memory (~1мс)
	if workingResult := m.searchWorking(query); workingResult.Found {
		result.Found = true
		result.Source = "working"
		result.Entries = workingResult.Entries
		result.Latency = time.Since(startTime)
		m.logAccess(result.Entries)
		return result
	}
	
	// УРОВЕНЬ 2: Full-Text Search (~10мс)
	if ftsResult := m.searchFTS(query); ftsResult.Found {
		result.Found = true
		result.Source = "semantic"
		result.Entries = ftsResult.Entries
		result.Latency = time.Since(startTime)
		m.logAccess(result.Entries)
		return result
	}
	
	// УРОВЕНЬ 3: Association Graph (~50мс)
	if assocResult := m.searchAssociations(query, context); assocResult.Found {
		result.Found = true
		result.Source = "associative"
		result.Entries = assocResult.Entries
		result.Latency = time.Since(startTime)
		m.logAccess(result.Entries)
		return result
	}
	
	// УРОВЕНЬ 4: Web Search (fallback, ~100-500мс)
	if m.config.EnableWebSearch {
		if webResult := m.searchWeb(query, context); webResult.Found {
			result.Found = true
			result.Source = "web"
			result.WebContent = webResult.WebContent
			result.Latency = time.Since(startTime)
			
			// Сохраняем найденное в память
			m.saveFromWeb(query, webResult.WebContent)
			
			return result
		}
	}
	
	// НИЧЕГО НЕ НАЙДЕНО
	result.Latency = time.Since(startTime)
	result.Suggestion = m.generateSuggestion(query, context)
	m.logMissedQuery(query, context)
	
	return result
}

// searchWorking поиск в working memory
func (m *Manager) searchWorking(query string) *SearchResult {
	m.working.mu.RLock()
	defer m.working.mu.RUnlock()
	
	tokens := tokenize(query)
	
	// Ищем в активных чанках
	for _, chunk := range m.working.Chunks {
		// Проверка названия чанка
		if containsAny(chunk.Name, tokens) {
			m.touchChunk(chunk.ID)
			return &SearchResult{
				Found: true,
				Entries: chunk.Entries,
				Source: "working",
			}
		}
		
		// Проверка содержимого
		for _, entry := range chunk.Entries {
			if containsAny(entry.Content, tokens) {
				m.touchEntry(entry.ID)
				return &SearchResult{
					Found: true,
					Entries: []*Entry{entry},
					Source: "working",
				}
			}
		}
	}
	
	return &SearchResult{Found: false}
}

// searchFTS full-text поиск
func (m *Manager) searchFTS(query string) *SearchResult {
	expandedQuery := expandQuery(query)
	
	sql := `
		SELECT e.id, e.type, e.category, e.content, e.metadata,
		       e.importance, e.retention, e.created_at, e.updated_at,
		       e.last_accessed, e.access_count, e.repetition_count, e.half_life
		FROM memory_entries e
		JOIN memory_fts fts ON e.rowid = fts.rowid
		WHERE memory_fts MATCH ?
		ORDER BY bm25(memory_fts)
		LIMIT 10
	`
	
	rows, err := m.db.Query(sql, expandedQuery)
	if err != nil {
		return &SearchResult{Found: false}
	}
	defer rows.Close()
	
	entries := parseEntries(rows)
	
	if len(entries) == 0 {
		return &SearchResult{Found: false}
	}
	
	return &SearchResult{
		Found: true,
		Entries: entries,
		Source: "semantic",
	}
}

// searchAssociations поиск по графу ассоциаций
func (m *Manager) searchAssociations(query string, context *SearchContext) *SearchResult {
	// Находим начальные узлы
	nodes := m.findNodesByQuery(query)
	
	if len(nodes) == 0 {
		return &SearchResult{Found: false}
	}
	
	// BFS обход графа
	visited := make(map[string]bool)
	queue := NewNodeQueue(nodes...)
	results := make([]*Entry, 0)
	
	for !queue.Empty() {
		node := queue.Dequeue()
		
		if visited[node.ID] {
			continue
		}
		visited[node.ID] = true
		
		// Получаем связанные записи
		entries, err := m.getEntriesByNode(node.ID)
		if err == nil && len(entries) > 0 {
			results = append(results, entries...)
		}
		
		// Добавляем соседей с весом > 0.3
		neighbors, err := m.getNeighbors(node.ID, 0.3)
		if err == nil {
			queue.Enqueue(neighbors...)
		}
	}
	
	if len(results) == 0 {
		return &SearchResult{Found: false}
	}
	
	return &SearchResult{
		Found: true,
		Entries: results,
		Source: "associative",
	}
}

// searchWeb поиск в интернете (fallback)
func (m *Manager) searchWeb(query string, context *SearchContext) *SearchResult {
	// Используем DuckDuckGo HTML scraping (без API ключа)
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(searchURL)
	if err != nil {
		return &SearchResult{Found: false}
	}
	defer resp.Body.Close()
	
	// Парсим результаты (упрощённо)
	content, err := extractMainContent(resp.Body)
	if err != nil || len(content) == 0 {
		return &SearchResult{Found: false}
	}
	
	return &SearchResult{
		Found: true,
		WebContent: content,
		Source: "web",
	}
}

// saveFromWeb сохраняет найденное из веба
func (m *Manager) saveFromWeb(query string, content string) error {
	// Сохраняем как semantic запись
	entry := &Entry{
		ID: generateID(),
		Type: "semantic",
		Category: inferCategory(query),
		Content: content,
		Importance: 0.5,
		Created: time.Now(),
		Updated: time.Now(),
		HalfLife: 7 * 24 * time.Hour, // 7 дней для веб-контента
	}
	
	return m.saveEntry(entry)
}

// generateSuggestion генерирует подсказку при отсутствии результата
func (m *Manager) generateSuggestion(query string, context *SearchContext) string {
	// Предлагаем категории
	category := inferCategory(query)
	
	// Получаем последние записи
	recent, _ := m.GetRecent(3)
	
	if len(recent) > 0 {
		return fmt.Sprintf(
			"🔍 Не нашёл '%s' в памяти. Попробую найти в интернете...\n\n"+
			"Пока вот недавнее из категории '%s':\n%s",
			query, category, formatEntries(recent),
		)
	}
	
	return fmt.Sprintf(
		"🔍 Не нашёл '%s' в памяти. Ищу в интернете...",
		query,
	)
}

// logMissedQuery логирует пропущенные запросы
func (m *Manager) logMissedQuery(query string, context *SearchContext) {
	// Сохраняем для анализа паттернов
	logFile := filepath.Join(m.dataDir, "missed_queries.log")
	
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	
	timestamp := time.Now().Format(time.RFC3339)
	f.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, query))
}

// logAccess логирует доступ к записям
func (m *Manager) logAccess(entries []*Entry) {
	for _, entry := range entries {
		m.touchEntry(entry.ID)
	}
}

// touchEntry обновляет статистику доступа
func (m *Manager) touchEntry(id string) {
	query := `
		UPDATE memory_entries
		SET last_accessed = ?, access_count = access_count + 1
		WHERE id = ?
	`
	m.db.Exec(query, time.Now().Unix(), id)
}

// touchChunk обновляет чанк
func (m *Manager) touchChunk(id string) {
	if chunk, ok := m.working.Chunks[id]; ok {
		chunk.LastUsed = time.Now()
	}
}

// Вспомогательные функции

func tokenize(query string) []string {
	return strings.Fields(strings.ToLower(query))
}

func containsAny(text string, tokens []string) bool {
	text = strings.ToLower(text)
	for _, token := range tokens {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func expandQuery(query string) string {
	tokens := strings.Fields(query)
	return strings.Join(tokens, " OR ")
}

func parseEntries(rows *sql.Rows) []*Entry {
	entries := make([]*Entry, 0)
	
	for rows.Next() {
		var entry Entry
		var createdAt, updatedAt, lastAccessed int64
		var metadataJSON string
		
		rows.Scan(
			&entry.ID, &entry.Type, &entry.Category, &entry.Content,
			&metadataJSON, &entry.Importance, &entry.Retention,
			&createdAt, &updatedAt, &lastAccessed, &entry.AccessCount,
			&entry.RepetitionCount, &entry.HalfLife,
		)
		
		entry.Created = time.Unix(createdAt, 0)
		entry.Updated = time.Unix(updatedAt, 0)
		entry.LastAccessed = time.Unix(lastAccessed, 0)
		
		json.Unmarshal([]byte(metadataJSON), &entry.Metadata)
		
		entries = append(entries, &entry)
	}
	
	return entries
}

func generateID() string {
	return fmt.Sprintf("mem_%d", time.Now().UnixNano())
}

func inferCategory(query string) string {
	// Простая эвристика
	query = strings.ToLower(query)
	
	if strings.Contains(query, "git") {
		return "git"
	}
	if strings.Contains(query, "npm") || strings.Contains(query, "node") {
		return "npm"
	}
	if strings.Contains(query, "docker") {
		return "docker"
	}
	if strings.Contains(query, "file") || strings.Contains(query, "read") || strings.Contains(query, "write") {
		return "files"
	}
	
	return "general"
}

func formatEntries(entries []*Entry) string {
	var sb strings.Builder
	for i, e := range entries {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("• %s", truncate(e.Content, 60)))
	}
	return sb.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// extractMainContent извлекает основное содержание из HTML
func extractMainContent(body io.Reader) (string, error) {
	// Упрощённый парсинг (в продакшене использовать goquery)
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	
	html := string(data)
	
	// Ищем сниппеты результатов
	var results []string
	
	// DuckDuckGo HTML формат: <a class="result__a" href="...">Title</a>
	// Упрощённо: ищем текст между тегами
	parts := strings.Split(html, "<a class=\"result__a\"")
	
	for i := 1; i < len(parts) && i <= 5; i++ {
		// Извлекаем заголовок
		titleStart := strings.Index(parts[i], ">")
		titleEnd := strings.Index(parts[i], "</a>")
		
		if titleStart != -1 && titleEnd != -1 && titleEnd > titleStart {
			title := parts[i][titleStart+1 : titleEnd]
			title = stripHTML(title)
			results = append(results, title)
		}
	}
	
	if len(results) == 0 {
		return "", fmt.Errorf("no results found")
	}
	
	return strings.Join(results, "\n\n"), nil
}

// stripHTML удаляет HTML теги
func stripHTML(html string) string {
	// Удаляем теги
	result := strings.Map(func(r rune) rune {
		if r == '<' || r == '>' {
			return -1
		}
		return r
	}, html)
	
	return strings.TrimSpace(result)
}

// NodeQueue очередь для BFS
type NodeQueue struct {
	nodes []*Node
}

func NewNodeQueue(nodes ...*Node) *NodeQueue {
	return &NodeQueue{nodes: nodes}
}

func (q *NodeQueue) Empty() bool {
	return len(q.nodes) == 0
}

func (q *NodeQueue) Enqueue(nodes ...*Node) {
	q.nodes = append(q.nodes, nodes...)
}

func (q *NodeQueue) Dequeue() *Node {
	if q.Empty() {
		return nil
	}
	node := q.nodes[0]
	q.nodes = q.nodes[1:]
	return node
}

// Node узел графа ассоциаций
type Node struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}
