package memory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("/tmp/test_memory")
	assert.NotNil(t, manager)
	assert.Equal(t, "/tmp/test_memory", manager.dataDir)
}

func TestInit(t *testing.T) {
	manager := NewManager("/tmp/test_memory_init")
	err := manager.Init()
	assert.NoError(t, err)
}

func TestAddEntry(t *testing.T) {
	manager := NewManager("/tmp/test_memory_add")
	err := manager.Init()
	assert.NoError(t, err)

	entry, err := manager.AddEntry("fact", "test", "Test content", nil)
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, "fact", entry.Type)
	assert.Equal(t, "test", entry.Category)
	assert.Equal(t, "Test content", entry.Content)
}

func TestGetRecent(t *testing.T) {
	manager := NewManager("/tmp/test_memory_recent")
	err := manager.Init()
	assert.NoError(t, err)

	_, _ = manager.AddEntry("fact", "cat1", "Content 1", nil)
	_, _ = manager.AddEntry("fact", "cat2", "Content 2", nil)

	entries, err := manager.GetRecent(5)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestListEntries(t *testing.T) {
	manager := NewManager("/tmp/test_memory_list")
	err := manager.Init()
	assert.NoError(t, err)

	_, _ = manager.AddEntry("fact", "test", "Test 1", nil)
	_, _ = manager.AddEntry("fact", "test", "Test 2", nil)

	entries := manager.ListEntries()
	assert.GreaterOrEqual(t, len(entries), 2)
}

func TestClear(t *testing.T) {
	manager := NewManager("/tmp/test_memory_clear")
	err := manager.Init()
	assert.NoError(t, err)

	_, _ = manager.AddEntry("fact", "test", "To be cleared", nil)

	err = manager.Clear()
	assert.NoError(t, err)

	entries := manager.ListEntries()
	assert.Empty(t, entries)
}

func TestRemember(t *testing.T) {
	manager := NewManager("/tmp/test_memory_remember")
	err := manager.Init()
	assert.NoError(t, err)

	entry, err := manager.Remember("fact", "Remember this", nil)
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, "Remember this", entry.Content)
}

func TestRecall(t *testing.T) {
	manager := NewManager("/tmp/test_memory_recall")
	err := manager.Init()
	assert.NoError(t, err)

	_, _ = manager.Remember("fact", "Go programming", nil)
	_, _ = manager.Remember("fact", "Python programming", nil)

	results := manager.Recall("Go", 10)
	assert.NotEmpty(t, results)
}

func TestAddMessage(t *testing.T) {
	manager := NewManager("/tmp/test_memory_message")
	err := manager.Init()
	assert.NoError(t, err)

	err = manager.AddMessage("user", "Hello")
	assert.NoError(t, err)

	err = manager.AddMessage("assistant", "Hi there")
	assert.NoError(t, err)
}

func TestGetContext(t *testing.T) {
	manager := NewManager("/tmp/test_memory_context")
	err := manager.Init()
	assert.NoError(t, err)

	_ = manager.AddMessage("user", "Test message")
	context := manager.GetContext()
	assert.NotEmpty(t, context)
}

func TestWorkingMemory(t *testing.T) {
	manager := NewManager("/tmp/test_memory_working")
	err := manager.Init()
	assert.NoError(t, err)

	entry := &Entry{
		ID:       "test_entry",
		Type:     "working",
		Category: "test",
		Content:  "Test working memory",
		Created:  time.Now(),
	}

	manager.addToWorking(entry)

	assert.NotEmpty(t, manager.working.Chunks)
	assert.Contains(t, manager.working.Chunks, "test")
}

func TestTouchEntry(t *testing.T) {
	manager := NewManager("/tmp/test_memory_touch")
	err := manager.Init()
	assert.NoError(t, err)

	entry, _ := manager.AddEntry("fact", "test", "Touch test", nil)

	manager.touchEntry(entry.ID)

	updated, _ := manager.GetRecent(1)
	assert.Greater(t, updated[0].AccessCount, 0)
}

func TestForgettingCurve(t *testing.T) {
	manager := NewManager("/tmp/test_memory_forget")
	err := manager.Init()
	assert.NoError(t, err)

	entry := &Entry{
		ID:         "forget_test",
		Type:       "semantic",
		Category:   "test",
		Content:    "Will be forgotten",
		Retention:  0.05,
		HalfLife:   24 * time.Hour,
		LastAccess: time.Now().Add(-48 * time.Hour),
	}

	_ = manager.saveEntry(entry)

	manager.forgetOldMemories()

	entries := manager.ListEntries()
	assert.NotContains(t, entries, entry)
}

func TestConsolidation(t *testing.T) {
	manager := NewManager("/tmp/test_memory_consolidate")
	err := manager.Init()
	assert.NoError(t, err)

	chunk := &Chunk{
		ID:       "test_chunk",
		Name:     "test",
		Priority: 0.9,
		Entries: []*Entry{
			{ID: "e1", Type: "working", Content: "Consolidate me"},
		},
	}

	manager.working.Chunks["test_chunk"] = chunk

	manager.consolidateWorkingMemory()

	entries := manager.ListEntries()
	assert.NotEmpty(t, entries)
}

func TestRestructure(t *testing.T) {
	manager := NewManager("/tmp/test_memory_restruct")
	err := manager.Init()
	assert.NoError(t, err)

	_, _ = manager.AddEntry("fact", "test", "Test restructure", nil)

	manager.restructure()

	entries := manager.ListEntries()
	assert.NotEmpty(t, entries)
}

func TestCreateAssociations(t *testing.T) {
	manager := NewManager("/tmp/test_memory_assoc")
	err := manager.Init()
	assert.NoError(t, err)

	entry := &Entry{
		ID:       "assoc_test",
		Type:     "semantic",
		Category: "test_category",
		Content:  "Test associations",
	}

	manager.createAssociations(entry)

	nodes := manager.findNodesByQuery("test_category")
	assert.NotEmpty(t, nodes)
}

func TestSaveConfig(t *testing.T) {
	manager := NewManager("/tmp/test_memory_config")
	err := manager.Init()
	assert.NoError(t, err)

	err = manager.saveConfig()
	assert.NoError(t, err)
}

func TestLoadConfig(t *testing.T) {
	manager := NewManager("/tmp/test_memory_loadconfig")
	err := manager.Init()
	assert.NoError(t, err)

	manager.loadConfig()
	assert.NotNil(t, manager.config)
}
