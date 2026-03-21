package skills

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine("/tmp/test_skills")
	assert.NotNil(t, engine)
	assert.Equal(t, "/tmp/test_skills", engine.skillsDir)
}

func TestNewAgentSkillEngine(t *testing.T) {
	engine := NewAgentSkillEngine()
	assert.NotNil(t, engine)
	assert.NotEmpty(t, engine.skills)
}

func TestLoadBuiltinSkills(t *testing.T) {
	engine := NewEngine("/tmp/test_builtin")
	engine.loadBuiltinSkills()

	assert.GreaterOrEqual(t, len(engine.skills), 4)

	expectedSkills := []string{"shell", "file", "search", "memory"}
	for _, name := range expectedSkills {
		assert.Contains(t, engine.skills, name)
	}
}

func TestFindSkillByCommand(t *testing.T) {
	engine := NewEngine("/tmp/test_find")
	engine.loadBuiltinSkills()

	t.Run("find by name", func(t *testing.T) {
		skill := engine.FindSkillByCommand("shell")
		assert.NotNil(t, skill)
		assert.Equal(t, "shell", skill.Name)
	})

	t.Run("find by alias", func(t *testing.T) {
		skill := engine.FindSkillByCommand("exec")
		assert.NotNil(t, skill)
		assert.Equal(t, "shell", skill.Name)
	})

	t.Run("not found", func(t *testing.T) {
		skill := engine.FindSkillByCommand("nonexistent")
		assert.Nil(t, skill)
	})
}

func TestExecute(t *testing.T) {
	engine := NewEngine("/tmp/test_execute")
	engine.loadBuiltinSkills()

	t.Run("execute shell skill", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.Execute(ctx, &SkillRequest{
			Command: "shell",
			Args:    []string{"echo", "hello"},
		})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "hello")
	})

	t.Run("execute file read", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.Execute(ctx, &SkillRequest{
			Command: "file",
			Args:    []string{"read", "/etc/hostname"},
		})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotEmpty(t, result.Output)
	})

	t.Run("execute search", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.Execute(ctx, &SkillRequest{
			Command: "search",
			Args:    []string{"package", "."},
		})

		assert.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("execute memory", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.Execute(ctx, &SkillRequest{
			Command: "memory",
			Args:    []string{"add", "test fact"},
		})

		assert.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("skill not found", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.Execute(ctx, &SkillRequest{
			Command: "nonexistent",
		})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "no skill found")
	})
}

func TestExecShell(t *testing.T) {
	engine := NewEngine("/tmp/test_shell")

	t.Run("empty args", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execShell(ctx, &SkillRequest{})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "no command specified")
	})

	t.Run("successful execution", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execShell(ctx, &SkillRequest{
			Args: []string{"echo", "test"},
		})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "test")
	})

	t.Run("command error", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execShell(ctx, &SkillRequest{
			Args: []string{"false"},
		})

		assert.NoError(t, err)
		assert.False(t, result.Success)
	})
}

func TestExecFile(t *testing.T) {
	engine := NewEngine("/tmp/test_file")

	t.Run("empty args", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execFile(ctx, &SkillRequest{})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "usage:")
	})

	t.Run("read nonexistent file", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execFile(ctx, &SkillRequest{
			Args: []string{"read", "/nonexistent/file.txt"},
		})

		assert.NoError(t, err)
		assert.False(t, result.Success)
	})

	t.Run("write and read", func(t *testing.T) {
		ctx := context.Background()

		writeResult, err := engine.execFile(ctx, &SkillRequest{
			Args: []string{"write", "/tmp/test_write.txt", "test content"},
		})

		assert.NoError(t, err)
		assert.True(t, writeResult.Success)

		readResult, err := engine.execFile(ctx, &SkillRequest{
			Args: []string{"read", "/tmp/test_write.txt"},
		})

		assert.NoError(t, err)
		assert.True(t, readResult.Success)
		assert.Contains(t, readResult.Output, "test content")
	})
}

func TestExecSearch(t *testing.T) {
	engine := NewEngine("/tmp/test_search")

	t.Run("empty pattern", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execSearch(ctx, &SkillRequest{})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "no search pattern specified")
	})

	t.Run("search in current dir", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execSearch(ctx, &SkillRequest{
			Args: []string{"package", "."},
		})

		assert.NoError(t, err)
		assert.True(t, result.Success)
	})
}

func TestExecGit(t *testing.T) {
	engine := NewEngine("/tmp/test_git")

	t.Run("empty args", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execGit(ctx, &SkillRequest{})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "usage:")
	})

	t.Run("git version", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execGit(ctx, &SkillRequest{
			Args: []string{"version"},
		})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "git version")
	})
}

func TestExecHTTP(t *testing.T) {
	engine := NewEngine("/tmp/test_http")

	t.Run("empty args", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execHTTP(ctx, &SkillRequest{})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "usage:")
	})

	t.Run("get request", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execHTTP(ctx, &SkillRequest{
			Args: []string{"get", "https://httpbin.org/get"},
		})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "httpbin.org")
	})
}

func TestExecNotify(t *testing.T) {
	engine := NewEngine("/tmp/test_notify")

	t.Run("empty args", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execNotify(ctx, &SkillRequest{})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "usage:")
	})

	t.Run("send notification", func(t *testing.T) {
		ctx := context.Background()
		result, err := engine.execNotify(ctx, &SkillRequest{
			Args: []string{"Test notification"},
		})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "Notification")
	})
}

func TestList(t *testing.T) {
	engine := NewEngine("/tmp/test_list")
	engine.loadBuiltinSkills()

	skills := engine.List()
	assert.NotEmpty(t, skills)
	assert.GreaterOrEqual(t, len(skills), 4)
}

func TestGet(t *testing.T) {
	engine := NewEngine("/tmp/test_get")
	engine.loadBuiltinSkills()

	skill := engine.Get("shell")
	assert.NotNil(t, skill)
	assert.Equal(t, "shell", skill.Name)

	nonexistent := engine.Get("nonexistent")
	assert.Nil(t, nonexistent)
}

func TestSetMemoryManager(t *testing.T) {
	engine := NewEngine("/tmp/test_setmem")
	assert.Nil(t, engine.memoryManager)

	mockManager := &mockMemoryManager{}
	engine.SetMemoryManager(mockManager)
	assert.NotNil(t, engine.memoryManager)
}

type mockMemoryManager struct{}

func (m *mockMemoryManager) Remember(string, string, map[string]interface{}) (*interface{}, error) {
	return nil, nil
}

func (m *mockMemoryManager) Recall(string, int) []*interface{} {
	return nil
}

func (m *mockMemoryManager) ListEntries() []*interface{} {
	return nil
}

func (m *mockMemoryManager) Clear() error {
	return nil
}
