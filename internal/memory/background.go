package memory

import (
	"encoding/json"
	"strings"
	"time"
)

// findNodesByQuery находит узлы по запросу
func (m *Manager) findNodesByQuery(query string) []*Node {
	sqlQuery := `
		SELECT id, type, name
		FROM nodes
		WHERE name LIKE ?
		LIMIT 10
	`
	
	rows, _ := m.assocDB.Query(sqlQuery, "%"+query+"%")
	defer rows.Close()
	
	nodes := make([]*Node, 0)
	
	for rows.Next() {
		var node Node
		rows.Scan(&node.ID, &node.Type, &node.Name)
		nodes = append(nodes, &node)
	}
	
	return nodes
}

// getEntriesByNode получает записи связанные с узлом
func (m *Manager) getEntriesByNode(nodeID string) ([]*Entry, error) {
	query := `
		SELECT id, type, category, content, metadata, importance,
		       created_at, updated_at
		FROM memory_entries
		WHERE category = ?
		LIMIT 10
	`
	
	rows, err := m.db.Query(query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	return parseEntries(rows), nil
}

// getNeighbors получает соседние узлы
func (m *Manager) getNeighbors(nodeID string, minWeight float64) ([]*Node, error) {
	query := `
		SELECT n.id, n.type, n.name
		FROM nodes n
		JOIN edges e ON n.id = e.target_id
		WHERE e.source_id = ? AND e.weight > ?
		LIMIT 10
	`
	
	rows, err := m.assocDB.Query(query, nodeID, minWeight)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	nodes := make([]*Node, 0)
	
	for rows.Next() {
		var node Node
		rows.Scan(&node.ID, &node.Type, &node.Name)
		nodes = append(nodes, &node)
	}
	
	return nodes, nil
}

// createNode создаёт узел
func (m *Manager) createNode(name, nodeType string) (*Node, error) {
	node := &Node{
		ID:   generateID(),
		Type: nodeType,
		Name: name,
	}
	
	query := `INSERT INTO nodes (id, type, name, last_activated) VALUES (?, ?, ?, ?)`
	_, err := m.assocDB.Exec(query, node.ID, node.Type, node.Name, time.Now().Unix())
	
	return node, err
}

// createEdge создаёт связь
func (m *Manager) createEdge(sourceID, targetID, edgeType string, weight float64) error {
	query := `
		INSERT OR REPLACE INTO edges (source_id, target_id, weight, type, co_activation_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, COALESCE((SELECT co_activation_count FROM edges WHERE source_id = ? AND target_id = ?), 0) + 1, ?, ?)
	`
	
	now := time.Now().Unix()
	_, err := m.assocDB.Exec(query, sourceID, targetID, weight, edgeType, sourceID, targetID, now, now)
	
	return err
}

// strengthenAssociation усиливает связь (Hebbian learning)
func (m *Manager) strengthenAssociation(sourceID, targetID string) error {
	query := `
		UPDATE edges
		SET weight = MIN(weight + 0.1, 1.0),
		    co_activation_count = co_activation_count + 1,
		    updated_at = ?
		WHERE source_id = ? AND target_id = ?
	`
	
	_, err := m.assocDB.Exec(query, time.Now().Unix(), sourceID, targetID)
	return err
}

// consolidationLoop фоновый процесс консолидации
func (m *Manager) consolidationLoop() {
	ticker := time.NewTicker(m.config.ConsolidationTime)
	defer ticker.Stop()
	
	for range ticker.C {
		m.consolidateWorkingMemory()
	}
}

// consolidateWorkingMemory переносит важное из working в долговременную
func (m *Manager) consolidateWorkingMemory() {
	m.working.mu.RLock()
	defer m.working.mu.RUnlock()
	
	for _, chunk := range m.working.Chunks {
		// Проверяем важность чанка
		if chunk.Priority > 0.7 {
			// Консолидируем в semantic
			for _, entry := range chunk.Entries {
				entry.Type = "semantic"
				entry.HalfLife = 7 * 24 * time.Hour // 7 дней
				m.saveEntry(entry)
			}
		}
	}
}

// forgettingLoop фоновый процесс забывания
func (m *Manager) forgettingLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		m.forgetOldMemories()
	}
}

// forgetOldMemories удаляет старые записи по кривой забывания
func (m *Manager) forgetOldMemories() {
	query := `
		UPDATE memory_entries
		SET retention = retention * EXP(-1.0 * (CAST(strftime('%s', 'now') AS REAL) - last_accessed) / half_life)
		WHERE retention > 0
	`
	
	_, err := m.db.Exec(query)
	if err != nil {
		return
	}
	
	// Удаляем с retention < threshold
	deleteQuery := `
		DELETE FROM memory_entries
		WHERE retention < ? AND type != 'procedural'
	`
	
	m.db.Exec(deleteQuery, m.config.ForgettingThreshold)
}

// restructuringLoop фоновый процесс реструктуризации
func (m *Manager) restructuringLoop() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		if m.shouldRestructure() {
			m.restructure()
		}
	}
}

// shouldRestructure проверяет необходимость реструктуризации
func (m *Manager) shouldRestructure() bool {
	// Проверяем количество записей
	var count int
	m.db.QueryRow("SELECT COUNT(*) FROM memory_entries").Scan(&count)
	
	if count > m.config.MaxEntries {
		return true
	}
	
	// Проверяем фрагментацию
	var deadCount int
	m.db.QueryRow(
		"SELECT COUNT(*) FROM memory_entries WHERE retention < ?",
		m.config.ForgettingThreshold,
	).Scan(&deadCount)
	
	fragmentation := float64(deadCount) / float64(count)
	
	if fragmentation > m.config.FragmentationLimit {
		return true
	}
	
	return false
}

// restructure проводит реструктуризацию
func (m *Manager) restructure() {
	// 1. Удаляем мёртвые записи
	m.db.Exec(
		"DELETE FROM memory_entries WHERE retention < ?",
		m.config.ForgettingThreshold,
	)
	
	// 2. Находим дубликаты
	m.findAndMergeDuplicates()
	
	// 3. Находим противоречия
	m.resolveContradictions()
	
	// 4. Оптимизируем БД
	m.db.Exec("VACUUM")
	m.assocDB.Exec("VACUUM")
}

// findAndMergeDuplicates находит и объединяет дубликаты
func (m *Manager) findAndMergeDuplicates() {
	query := `
		SELECT content, GROUP_CONCAT(id), COUNT(*)
		FROM memory_entries
		GROUP BY content
		HAVING COUNT(*) > 1
	`
	
	rows, err := m.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var content, ids string
		var count int
		
		rows.Scan(&content, &ids, &count)
		
		// Оставляем только одну запись
		idList := splitString(ids, ",")
		for i := 1; i < len(idList); i++ {
			m.db.Exec("DELETE FROM memory_entries WHERE id = ?", idList[i])
		}
	}
}

// resolveContradictions разрешает противоречия
func (m *Manager) resolveContradictions() {
	// Упрощённо: ищем записи с одинаковой категорией но разным содержанием
	// В реальности нужна более сложная логика
}

// saveEntry сохраняет запись
func (m *Manager) saveEntry(entry *Entry) error {
	metadataJSON, _ := json.Marshal(entry.Metadata)

	query := `
		INSERT OR REPLACE INTO memory_entries
		(id, type, category, content, metadata, importance, retention,
		 created_at, updated_at, last_accessed, access_count,
		 repetition_count, half_life, chunk_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := m.db.Exec(query,
		entry.ID, entry.Type, entry.Category, entry.Content,
		string(metadataJSON), entry.Importance, entry.Retention,
		entry.Created.Unix(), entry.Updated.Unix(), entry.LastAccessed.Unix(),
		entry.AccessCount, entry.RepetitionCount, int(entry.HalfLife.Seconds()),
		entry.ChunkID,
	)

	// Обновляем индекс
	if err == nil {
		m.index.Add(entry)
	}

	return err
}

// AddEntry добавляет запись (публичный API)
func (m *Manager) AddEntry(entryType, category, content string, metadata map[string]interface{}) (*Entry, error) {
	entry := &Entry{
		ID:        generateID(),
		Type:      entryType,
		Category:  category,
		Content:   content,
		Metadata:  metadata,
		Importance: 0.5,
		Retention: 1.0,
		Created:   time.Now(),
		Updated:   time.Now(),
		HalfLife:  24 * time.Hour,
	}
	
	// Добавляем в working memory
	m.addToWorking(entry)
	
	// Сохраняем в БД
	if err := m.saveEntry(entry); err != nil {
		return nil, err
	}
	
	// Создаём ассоциации
	m.createAssociations(entry)
	
	return entry, nil
}

// addToWorking добавляет в working memory
func (m *Manager) addToWorking(entry *Entry) {
	m.working.mu.Lock()
	defer m.working.mu.Unlock()
	
	// Ищем или создаём чанк
	chunkID := entry.Category
	chunk, exists := m.working.Chunks[chunkID]
	
	if !exists {
		// Проверяем лимит чанков
		if len(m.working.Chunks) >= m.working.MaxChunks {
			// Вытесняем наименее приоритетный
			m.evictLowestPriorityChunk()
		}
		
		chunk = &Chunk{
			ID:       chunkID,
			Name:     entry.Category,
			Entries:  make([]*Entry, 0),
			Created:  time.Now(),
			LastUsed: time.Now(),
		}
		
		m.working.Chunks[chunkID] = chunk
		m.working.AccessOrder = append(m.working.AccessOrder, chunkID)
	}
	
	chunk.Entries = append(chunk.Entries, entry)
	entry.ChunkID = chunkID
}

// evictLowestPriorityChunk вытесняет наименее приоритетный чанк
func (m *Manager) evictLowestPriorityChunk() {
	var lowestID string
	var lowestPriority float64 = 1.0
	
	for id, chunk := range m.working.Chunks {
		// Приоритет = важность × свежесть
		priority := chunk.Priority * time.Since(chunk.LastUsed).Minutes()
		
		if priority < lowestPriority {
			lowestPriority = priority
			lowestID = id
		}
	}
	
	if lowestID != "" {
		delete(m.working.Chunks, lowestID)
		
		// Удаляем из AccessOrder
		for i, id := range m.working.AccessOrder {
			if id == lowestID {
				m.working.AccessOrder = append(m.working.AccessOrder[:i], m.working.AccessOrder[i+1:]...)
				break
			}
		}
	}
}

// createAssociations создаёт ассоциации для записи
func (m *Manager) createAssociations(entry *Entry) {
	// Создаём узел для категории
	node, _ := m.createNode(entry.Category, "category")
	
	// Связываем с другими узлами той же категории
	relatedNodes := m.findNodesByQuery(entry.Category)
	
	for _, related := range relatedNodes {
		if related.ID != node.ID {
			m.createEdge(node.ID, related.ID, "semantic", 0.5)
		}
	}
}

// GetRecent получает последние записи
func (m *Manager) GetRecent(limit int) ([]*Entry, error) {
	query := `
		SELECT id, type, category, content, metadata, importance,
		       created_at, updated_at, last_accessed, access_count
		FROM memory_entries
		ORDER BY last_accessed DESC
		LIMIT ?
	`
	
	rows, err := m.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	return parseEntries(rows), nil
}

// ListEntries получает все записи (для CLI)
func (m *Manager) ListEntries() []*Entry {
	query := `
		SELECT id, type, category, content, metadata, importance,
		       created_at, updated_at, last_accessed, access_count,
		       repetition_count, half_life
		FROM memory_entries
		ORDER BY created_at DESC
	`
	
	rows, err := m.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()
	
	return parseEntries(rows)
}

// Clear очищает память
func (m *Manager) Clear() error {
	_, err := m.db.Exec("DELETE FROM memory_entries")
	
	m.working.mu.Lock()
	m.working.Chunks = make(map[string]*Chunk)
	m.working.AccessOrder = make([]string, 0)
	m.working.mu.Unlock()
	
	return err
}

// Remember добавляет факт в память (совместимость со старым API)
func (m *Manager) Remember(entryType, content string, metadata map[string]interface{}) (*Entry, error) {
	return m.AddEntry(entryType, "fact", content, metadata)
}

// Recall ищет в памяти (совместимость со старым API)
func (m *Manager) Recall(query string, limit int) []*Entry {
	result := m.Find(query, nil)
	
	if limit > 0 && len(result.Entries) > limit {
		return result.Entries[:limit]
	}
	
	return result.Entries
}

// AddMessage добавляет сообщение в память (совместимость)
func (m *Manager) AddMessage(role, content string) error {
	entryType := "episodic"
	if role == "user" {
		entryType = "episodic_user"
	}
	
	_, err := m.AddEntry(entryType, "conversation", content, map[string]interface{}{
		"role": role,
	})
	
	return err
}

// GetContext получает контекст (совместимость)
func (m *Manager) GetContext() string {
	entries, _ := m.GetRecent(10)
	
	if len(entries) == 0 {
		return ""
	}
	
	return formatEntries(entries)
}

// Init инициализирует (совместимость)
// Примечание: основной метод Init в manager.go

// Вспомогательные функции

func splitString(s, sep string) []string {
	return strings.Split(s, sep)
}
