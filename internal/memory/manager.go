package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Manager менеджер памяти с нейробиологической архитектурой
type Manager struct {
	mu sync.RWMutex
	
	// Базы данных
	db         *sql.DB
	assocDB    *sql.DB
	
	// Working memory (in-memory)
	working *WorkingMemory
	
	// Пути
	dataDir string
	
	// Настройки
	config *Config
}

// Config конфигурация памяти
type Config struct {
	// Working memory
	MaxChunks       int           `json:"max_chunks"`        // 7±2
	ChunkDecayTime  time.Duration `json:"chunk_decay_time"`  // 5 мин
	
	// Консолидация
	ConsolidationTime time.Duration `json:"consolidation_time"` // 5 мин базово
	
	// Забывание
	ForgettingHalfLife time.Duration `json:"forgetting_half_life"` // 1 день
	ForgettingThreshold float64      `json:"forgetting_threshold"` // 0.1 (10%)
	
	// Реструктуризация
	MaxEntries         int     `json:"max_entries"`          // 1000
	FragmentationLimit float64 `json:"fragmentation_limit"`  // 0.3 (30%)
	
	// Поиск в интернете
	EnableWebSearch    bool   `json:"enable_web_search"`     // true
	WebSearchThreshold int    `json:"web_search_threshold"`  // 3 (после 3 неудач)
}

// DefaultConfig конфигурация по умолчанию
func DefaultConfig() *Config {
	return &Config{
		MaxChunks:            9,
		ChunkDecayTime:       5 * time.Minute,
		ConsolidationTime:    5 * time.Minute,
		ForgettingHalfLife:   24 * time.Hour,
		ForgettingThreshold:  0.1,
		MaxEntries:           1000,
		FragmentationLimit:   0.3,
		EnableWebSearch:      true,
		WebSearchThreshold:   3,
	}
}

// WorkingMemory оперативная память (7±2 чанка)
type WorkingMemory struct {
	mu        sync.RWMutex
	Chunks    map[string]*Chunk
	AccessOrder []string
	MaxChunks int
	DecayTime time.Duration
}

// Chunk группа связанных записей
type Chunk struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"` // "git", "npm", "deployment"
	Entries  []*Entry      `json:"entries"`
	Created  time.Time     `json:"created"`
	LastUsed time.Time     `json:"last_used"`
	Priority float64       `json:"priority"` // для вытеснения
}

// Entry запись памяти
type Entry struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"` // working|episodic|semantic|procedural
	Category       string                 `json:"category"`
	Content        string                 `json:"content"`
	Metadata       map[string]interface{} `json:"metadata"`
	Importance     float64                `json:"importance"` // 0-1
	Retention      float64                `json:"retention"`  // 0-1
	Created        time.Time              `json:"created"`
	Updated        time.Time              `json:"updated"`
	LastAccessed   time.Time              `json:"last_accessed"`
	AccessCount    int                    `json:"access_count"`
	RepetitionCount int                   `json:"repetition_count"`
	HalfLife       time.Duration          `json:"half_life"`
	ChunkID        string                 `json:"chunk_id"`
}

// NewManager создаёт новый менеджер памяти
func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir: dataDir,
		config:  DefaultConfig(),
		working: &WorkingMemory{
			Chunks:    make(map[string]*Chunk),
			AccessOrder: make([]string, 0),
			MaxChunks: 9,
			DecayTime: 5 * time.Minute,
		},
	}
}

// Init инициализирует память
func (m *Manager) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Создаём директорию
	if err := os.MkdirAll(m.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}
	
	// Загружаем конфигурацию если есть
	m.loadConfig()
	
	// Инициализируем БД
	if err := m.initDatabase(); err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}
	
	if err := m.initAssociationDB(); err != nil {
		return fmt.Errorf("failed to init association database: %w", err)
	}
	
	// Загружаем working memory
	if err := m.loadWorkingMemory(); err != nil {
		return fmt.Errorf("failed to load working memory: %w", err)
	}
	
	// Запускаем фоновые процессы
	go m.consolidationLoop()
	go m.forgettingLoop()
	go m.restructuringLoop()
	
	return nil
}

// loadConfig загружает конфигурацию
func (m *Manager) loadConfig() {
	configPath := filepath.Join(m.dataDir, "memory_config.json")
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return // Используем дефолтную
	}
	
	json.Unmarshal(data, m.config)
}

// saveConfig сохраняет конфигурацию
func (m *Manager) saveConfig() error {
	configPath := filepath.Join(m.dataDir, "memory_config.json")
	
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}

// initDatabase инициализирует основную БД
func (m *Manager) initDatabase() error {
	dbPath := filepath.Join(m.dataDir, "memory.db")
	
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	
	m.db = db
	
	// Создаём таблицы
	schema := `
	-- Основная таблица записей
	CREATE TABLE IF NOT EXISTS memory_entries (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		category TEXT,
		content TEXT NOT NULL,
		metadata JSON,
		importance REAL DEFAULT 0.5,
		retention REAL DEFAULT 1.0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		last_accessed INTEGER,
		access_count INTEGER DEFAULT 0,
		repetition_count INTEGER DEFAULT 0,
		half_life INTEGER DEFAULT 300,
		chunk_id TEXT,
		embeddings BLOB
	);
	
	-- Чанки для working memory
	CREATE TABLE IF NOT EXISTS chunks (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		entry_count INTEGER DEFAULT 0,
		last_used INTEGER NOT NULL
	);
	
	-- Индексы
	CREATE INDEX IF NOT EXISTS idx_type ON memory_entries(type);
	CREATE INDEX IF NOT EXISTS idx_category ON memory_entries(category);
	CREATE INDEX IF NOT EXISTS idx_chunk ON memory_entries(chunk_id);
	CREATE INDEX IF NOT EXISTS idx_retention ON memory_entries(retention);
	CREATE INDEX IF NOT EXISTS idx_last_accessed ON memory_entries(last_accessed);
	
	-- Full-text search
	CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(
		content,
		category,
		content='memory_entries',
		content_rowid='rowid'
	);
	
	-- Триггеры для FTS
	CREATE TRIGGER IF NOT EXISTS memory_ai AFTER INSERT ON memory_entries BEGIN
		INSERT INTO memory_fts(rowid, content, category) 
		VALUES (new.rowid, new.content, new.category);
	END;
	
	CREATE TRIGGER IF NOT EXISTS memory_ad AFTER DELETE ON memory_entries BEGIN
		DELETE FROM memory_fts WHERE rowid = old.rowid;
	END;
	
	CREATE TRIGGER IF NOT EXISTS memory_au AFTER UPDATE ON memory_entries BEGIN
		DELETE FROM memory_fts WHERE rowid = old.rowid;
		INSERT INTO memory_fts(rowid, content, category) 
		VALUES (new.rowid, new.content, new.category);
	END;
	`
	
	_, err = db.Exec(schema)
	return err
}

// initAssociationDB инициализирует БД ассоциаций
func (m *Manager) initAssociationDB() error {
	dbPath := filepath.Join(m.dataDir, "associations.db")
	
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	
	m.assocDB = db
	
	schema := `
	-- Узлы графа
	CREATE TABLE IF NOT EXISTS nodes (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		activation REAL DEFAULT 0.0,
		last_activated INTEGER
	);
	
	-- Связи
	CREATE TABLE IF NOT EXISTS edges (
		source_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		weight REAL DEFAULT 0.0,
		type TEXT NOT NULL,
		co_activation_count INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		PRIMARY KEY (source_id, target_id)
	);
	
	-- Индексы
	CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
	CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);
	CREATE INDEX IF NOT EXISTS idx_edges_weight ON edges(weight DESC);
	`
	
	_, err = db.Exec(schema)
	return err
}

// loadWorkingMemory загружает working memory из БД
func (m *Manager) loadWorkingMemory() error {
	// Загружаем последние активные чанки
	query := `
		SELECT id, name, created_at, last_used 
		FROM chunks 
		ORDER BY last_used DESC 
		LIMIT ?
	`
	
	rows, err := m.db.Query(query, m.config.MaxChunks)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var chunk Chunk
		var createdAt, lastUsed int64
		
		err := rows.Scan(&chunk.ID, &chunk.Name, &createdAt, &lastUsed)
		if err != nil {
			return err
		}
		
		chunk.Created = time.Unix(createdAt, 0)
		chunk.LastUsed = time.Unix(lastUsed, 0)
		chunk.Entries = make([]*Entry, 0)
		
		// Загружаем записи чанка
		entries, err := m.getChunkEntries(chunk.ID)
		if err != nil {
			return err
		}
		
		chunk.Entries = entries
		m.working.Chunks[chunk.ID] = &chunk
		m.working.AccessOrder = append(m.working.AccessOrder, chunk.ID)
	}
	
	return nil
}

// getChunkEntries получает записи чанка
func (m *Manager) getChunkEntries(chunkID string) ([]*Entry, error) {
	query := `
		SELECT id, type, category, content, metadata, importance, retention,
		       created_at, updated_at, last_accessed, access_count,
		       repetition_count, half_life
		FROM memory_entries
		WHERE chunk_id = ?
		ORDER BY last_accessed DESC
	`
	
	rows, err := m.db.Query(query, chunkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	entries := make([]*Entry, 0)
	
	for rows.Next() {
		var entry Entry
		var createdAt, updatedAt, lastAccessed int64
		var metadataJSON string
		
		err := rows.Scan(
			&entry.ID, &entry.Type, &entry.Category, &entry.Content,
			&metadataJSON, &entry.Importance, &entry.Retention,
			&createdAt, &updatedAt, &lastAccessed, &entry.AccessCount,
			&entry.RepetitionCount, &entry.HalfLife,
		)
		if err != nil {
			return nil, err
		}
		
		entry.Created = time.Unix(createdAt, 0)
		entry.Updated = time.Unix(updatedAt, 0)
		entry.LastAccessed = time.Unix(lastAccessed, 0)
		
		json.Unmarshal([]byte(metadataJSON), &entry.Metadata)
		
		entries = append(entries, &entry)
	}
	
	return entries, nil
}
