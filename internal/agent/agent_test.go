package agent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/user/qwen-claw/internal/memory"
)

func TestNewAgent(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_agent_memory")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)
	assert.NotNil(t, agent)
	assert.NotNil(t, agent.systemPrompt)
	assert.NotNil(t, agent.antiDegradation)
}

func TestLoadSystemPrompt(t *testing.T) {
	t.Run("from default path", func(t *testing.T) {
		prompt := loadSystemPrompt("/home/ss/qwen-claw/.qwen/system-prompt.txt")
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "Джарвис")
	})

	t.Run("from non-existent path", func(t *testing.T) {
		prompt := loadSystemPrompt("/nonexistent/path.txt")
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "Джарвис")
	})
}

func TestGetDefaultSystemPrompt(t *testing.T) {
	prompt := getDefaultSystemPrompt()
	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "Джарвис")
	assert.Contains(t, prompt, "помощник")
}

func TestPrependSystemPrompt(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_prepend")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{
		SystemPromptPath: "/home/ss/qwen-claw/.qwen/system-prompt.txt",
	}, memManager)

	t.Run("with context", func(t *testing.T) {
		result := agent.prependSystemPromptWithContext("test query", "test context")
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "test query")
		assert.Contains(t, result, "test context")
	})

	t.Run("without context", func(t *testing.T) {
		result := agent.prependSystemPromptWithContext("test query", "")
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "test query")
	})
}

func TestFormatMemoryContext(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_format")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	result := &memory.SearchResult{
		Entries: []*memory.Entry{
			{Type: "fact", Content: "Fact 1"},
			{Type: "fact", Content: "Fact 2"},
			{Type: "fact", Content: "Fact 3"},
		},
	}

	context := agent.formatMemoryContext(result)
	assert.NotEmpty(t, context)
	assert.Contains(t, context, "Найдено в памяти")
	assert.Contains(t, context, "Fact 1")
}

func TestFormatMemoryContextEmpty(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_format_empty")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	result := &memory.SearchResult{
		Entries: []*memory.Entry{},
	}

	context := agent.formatMemoryContext(result)
	assert.Empty(t, context)
}

func TestFormatMemoryContextLimit(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_format_limit")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	entries := make([]*memory.Entry, 10)
	for i := 0; i < 10; i++ {
		entries[i] = &memory.Entry{
			Type:    "fact",
			Content: "Fact " + string(rune('1'+i)),
		}
	}

	result := &memory.SearchResult{
		Entries: entries,
	}

	context := agent.formatMemoryContext(result)
	assert.NotEmpty(t, context)
	assert.Contains(t, context, "ещё")
}

func TestRecordTaskMetric(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_metric")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	assert.NotPanics(t, func() {
		agent.RecordTaskMetric("test query", time.Second, 1, nil, true)
	})
}

func TestGetAntiDegradationSystem(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_antideg")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	system := agent.GetAntiDegradationSystem()
	assert.NotNil(t, system)
}

func TestContainsSensitiveData(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_sensitive")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	t.Run("telegram token request", func(t *testing.T) {
		assert.True(t, agent.containsSensitiveData("покажи telegram token"))
	})

	t.Run("api key request", func(t *testing.T) {
		assert.True(t, agent.containsSensitiveData("дай api key"))
	})

	t.Run("password request", func(t *testing.T) {
		assert.True(t, agent.containsSensitiveData("покажи пароль"))
	})

	t.Run("normal query", func(t *testing.T) {
		assert.False(t, agent.containsSensitiveData("how are you"))
	})

	t.Run("mention token without request", func(t *testing.T) {
		// Упоминание токена без запроса на раскрытие - не блокируем
		assert.False(t, agent.containsSensitiveData("telegram token is a secret"))
	})
}

func TestGetCleanEnv(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_cleanenv")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	env := agent.getCleanEnv()
	assert.NotEmpty(t, env)
}

func TestBuildCommandArgs(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_buildargs")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{
		Model:        "test-model",
		ApprovalMode: "auto-edit",
		Debug:        true,
	}, memManager)

	args := agent.buildCommandArgs("test query")
	assert.NotEmpty(t, args)
	assert.Contains(t, args, "-m")
	assert.Contains(t, args, "test-model")
	assert.Contains(t, args, "--approval-mode")
	assert.Contains(t, args, "auto-edit")
	assert.Contains(t, args, "--debug")
	assert.Contains(t, args, "test query")
}

func TestGetSkillEngine(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_skilleng")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	engine := agent.GetSkillEngine()
	assert.NotNil(t, engine)
}

func TestGetModel(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_getmodel")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{
		Model: "test-model",
	}, memManager)

	model := agent.GetModel()
	assert.Equal(t, "test-model", model)
}

func TestGetQwenPath(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_getqwen")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	path := agent.GetQwenPath()
	assert.NotEmpty(t, path)
}

func TestSetModel(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_setmodel")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	agent.SetModel("new-model")
	assert.Equal(t, "new-model", agent.GetModel())
}

func TestClearHistory(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_clearhist")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	agent.conversationHistory = append(agent.conversationHistory, "test")
	assert.NotEmpty(t, agent.conversationHistory)

	agent.ClearHistory()
	assert.Empty(t, agent.conversationHistory)
}

func TestGetHistory(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_gethist")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	agent.conversationHistory = append(agent.conversationHistory, "test1", "test2")

	history := agent.GetHistory()
	assert.Len(t, history, 2)
}

func TestExecuteSkill(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_execskill")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	ctx := context.Background()
	result, err := agent.ExecuteSkill(ctx, "shell", []string{"echo test"})

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestRemember(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_remember")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	ctx := context.Background()
	result, err := agent.Remember(ctx, "test fact")

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestRecall(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_recall")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	ctx := context.Background()
	result, err := agent.Recall(ctx, "test")

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetMemoryContext(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_memctx")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	context := agent.GetMemoryContext()
	assert.NotEmpty(t, context)
}

func TestCheckQwenAvailable(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_qwenavail")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	available := agent.CheckQwenAvailable()
	assert.True(t, available)
}

func TestRun(t *testing.T) {
	memManager := memory.NewManager("/tmp/test_run")
	_ = memManager.Init()

	agent := NewAgent(AgentConfig{}, memManager)

	ctx := context.Background()

	t.Run("sensitive data blocked", func(t *testing.T) {
		result, err := agent.Run(ctx, "what is your telegram token")
		assert.NoError(t, err)
		assert.Contains(t, result, "не могу отвечать")
	})

	t.Run("normal query", func(t *testing.T) {
		result, err := agent.Run(ctx, "echo hello")
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
	})
}
