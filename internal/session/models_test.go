package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSession(t *testing.T) {
	sess := NewSession("user123", "Test Session")

	require.NotNil(t, sess)
	assert.NotEmpty(t, sess.ID)
	assert.Equal(t, "user123", sess.UserID)
	assert.Equal(t, "Test Session", sess.Title)
	assert.Equal(t, int32(0), sess.MessageCount)
	assert.Equal(t, int32(0), sess.TotalTokens)
	// Metadata может быть пустой map
	assert.NotNil(t, sess.Metadata)
}

func TestSession_AddToken(t *testing.T) {
	sess := NewSession("user123", "Test")

	sess.AddToken(100)
	assert.Equal(t, int32(1), sess.MessageCount)
	assert.Equal(t, int32(100), sess.TotalTokens)

	sess.AddToken(50)
	assert.Equal(t, int32(2), sess.MessageCount)
	assert.Equal(t, int32(150), sess.TotalTokens)
}

func TestSession_Touch(t *testing.T) {
	sess := NewSession("user123", "Test")
	initialTime := sess.LastAccessedAt

	time.Sleep(10 * time.Millisecond)
	sess.Touch()

	assert.True(t, sess.LastAccessedAt.After(initialTime))
}

func TestNewMessage(t *testing.T) {
	msg := NewMessage("session123", "user", "Hello!", 10)

	require.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "session123", msg.SessionID)
	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello!", msg.Content)
	assert.Equal(t, int32(10), msg.Tokens)
}

func TestNewMemoryEntry(t *testing.T) {
	entry := NewMemoryEntry("user123", "session123", "Test content", MemoryTypeWorking)

	require.NotNil(t, entry)
	assert.NotEmpty(t, entry.ID)
	assert.Equal(t, "user123", entry.UserID)
	assert.Equal(t, "session123", entry.SessionID)
	assert.Equal(t, MemoryTypeWorking, entry.Type)
	assert.Equal(t, "Test content", entry.Content)
	assert.Equal(t, float32(0.5), entry.Importance)
	assert.Equal(t, float32(1.0), entry.Retention)
}

func TestMemoryEntry_Access(t *testing.T) {
	entry := NewMemoryEntry("user123", "session123", "Test", MemoryTypeWorking)
	initialTime := entry.LastAccessedAt
	initialCount := entry.AccessCount

	time.Sleep(10 * time.Millisecond)
	entry.Access()

	assert.True(t, entry.LastAccessedAt.After(initialTime))
	assert.Equal(t, int32(1), entry.AccessCount)
	assert.Greater(t, entry.AccessCount, initialCount)
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		expected int32
	}{
		{"", 0},
		{"Hi", 0},
		{"Hello", 1},
		{"Hello World", 2},
		{"This is a longer text with more words", 9}, // 37 символов / 4 = 9
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := EstimateTokens(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateMessage(t *testing.T) {
	tests := []struct {
		text     string
		maxLen   int
		expected string
	}{
		{"Short", 10, "Short"},
		{"This is a longer text", 10, "This is a ..."},
		{"", 5, ""},
		{"Exact", 5, "Exact"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := TruncateMessage(tt.text, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeRole(t *testing.T) {
	tests := []struct {
		role     string
		expected string
	}{
		{"user", "user"},
		{"assistant", "assistant"},
		{"system", "system"},
		{"invalid", "user"},
		{"", "user"},
		{"USER", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := SanitizeRole(tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSession(t *testing.T) {
	t.Run("valid session", func(t *testing.T) {
		sess := NewSession("user123", "Test")
		err := ValidateSession(sess)
		assert.NoError(t, err)
	})

	t.Run("nil session", func(t *testing.T) {
		err := ValidateSession(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session is nil")
	})

	t.Run("empty ID", func(t *testing.T) {
		sess := &Session{UserID: "user123"}
		err := ValidateSession(sess)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID is empty")
	})

	t.Run("empty UserID", func(t *testing.T) {
		sess := NewSession("", "Test")
		sess.ID = "123"
		err := ValidateSession(sess)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user ID is empty")
	})
}

func TestValidateMessage(t *testing.T) {
	t.Run("valid message", func(t *testing.T) {
		msg := NewMessage("session123", "user", "Hello", 10)
		err := ValidateMessage(msg)
		assert.NoError(t, err)
	})

	t.Run("nil message", func(t *testing.T) {
		err := ValidateMessage(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message is nil")
	})

	t.Run("empty session ID", func(t *testing.T) {
		msg := &Message{Role: "user", Content: "Hello"}
		err := ValidateMessage(msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID is empty")
	})

	t.Run("empty content", func(t *testing.T) {
		msg := &Message{SessionID: "123", Role: "user"}
		err := ValidateMessage(msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message content is empty")
	})

	t.Run("empty role", func(t *testing.T) {
		msg := &Message{SessionID: "123", Content: "Hello"}
		err := ValidateMessage(msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message role is empty")
	})
}

func TestValidateMemoryEntry(t *testing.T) {
	t.Run("valid entry", func(t *testing.T) {
		entry := NewMemoryEntry("user123", "session123", "Test", MemoryTypeWorking)
		err := ValidateMemoryEntry(entry)
		assert.NoError(t, err)
	})

	t.Run("nil entry", func(t *testing.T) {
		err := ValidateMemoryEntry(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory entry is nil")
	})

	t.Run("empty user ID", func(t *testing.T) {
		entry := &MemoryEntry{Content: "Test", Type: MemoryTypeWorking}
		err := ValidateMemoryEntry(entry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user ID is empty")
	})

	t.Run("empty content", func(t *testing.T) {
		entry := &MemoryEntry{UserID: "user123", Type: MemoryTypeWorking}
		err := ValidateMemoryEntry(entry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory content is empty")
	})

	t.Run("empty type", func(t *testing.T) {
		entry := &MemoryEntry{UserID: "user123", Content: "Test"}
		err := ValidateMemoryEntry(entry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory type is empty")
	})
}
