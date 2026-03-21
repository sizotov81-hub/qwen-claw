package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewScheduler(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler", nil)
	assert.NotNil(t, sched)
	assert.Equal(t, "/tmp/test_scheduler", sched.dataDir)
}

func TestInit(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_init", nil)
	err := sched.Init()
	assert.NoError(t, err)
}

func TestAddTask(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_add", nil)
	err := sched.Init()
	assert.NoError(t, err)

	task, err := sched.AddTask("test_task", "Test Task", "echo test", "@daily")
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, "test_task", task.Name)
	assert.Equal(t, "@daily", task.Schedule)
	assert.True(t, task.Enabled)
}

func TestGetTask(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_get", nil)
	err := sched.Init()
	assert.NoError(t, err)

	created, _ := sched.AddTask("get_task", "Get Task", "echo get", "@hourly")

	task, err := sched.GetTask(created.ID)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, task.ID)

	_, err = sched.GetTask("nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestRemoveTask(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_remove", nil)
	err := sched.Init()
	assert.NoError(t, err)

	created, _ := sched.AddTask("remove_task", "Remove Task", "echo remove", "@weekly")

	err = sched.RemoveTask(created.ID)
	assert.NoError(t, err)

	err = sched.RemoveTask(created.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestEnableTask(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_enable", nil)
	err := sched.Init()
	assert.NoError(t, err)

	created, _ := sched.AddTask("enable_task", "Enable Task", "echo enable", "@daily")
	created.Enabled = false

	err = sched.EnableTask(created.ID)
	assert.NoError(t, err)

	task, _ := sched.GetTask(created.ID)
	assert.True(t, task.Enabled)
}

func TestDisableTask(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_disable", nil)
	err := sched.Init()
	assert.NoError(t, err)

	created, _ := sched.AddTask("disable_task", "Disable Task", "echo disable", "@daily")

	err = sched.DisableTask(created.ID)
	assert.NoError(t, err)

	task, _ := sched.GetTask(created.ID)
	assert.False(t, task.Enabled)
}

func TestListTasks(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_list", nil)
	err := sched.Init()
	assert.NoError(t, err)

	_, _ = sched.AddTask("task1", "Task 1", "echo 1", "@daily")
	_, _ = sched.AddTask("task2", "Task 2", "echo 2", "@hourly")

	tasks := sched.ListTasks()
	assert.Len(t, tasks, 2)
}

func TestCalculateNextRun(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_next", nil)

	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)

	t.Run("@hourly", func(t *testing.T) {
		next := sched.calculateNextRun("@hourly", now)
		assert.NotNil(t, next)
		assert.Equal(t, 13, next.Hour())
	})

	t.Run("@daily", func(t *testing.T) {
		next := sched.calculateNextRun("@daily", now)
		assert.NotNil(t, next)
		assert.Equal(t, 22, next.Day())
	})

	t.Run("@weekly", func(t *testing.T) {
		next := sched.calculateNextRun("@weekly", now)
		assert.NotNil(t, next)
		assert.Equal(t, time.Sunday, next.Weekday())
	})

	t.Run("@monthly", func(t *testing.T) {
		next := sched.calculateNextRun("@monthly", now)
		assert.NotNil(t, next)
		assert.Equal(t, 4, int(next.Month()))
	})

	t.Run("invalid expression", func(t *testing.T) {
		next := sched.calculateNextRun("invalid", now)
		assert.Nil(t, next)
	})
}

func TestParseCronExpression(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_parse", nil)

	now := time.Date(2026, 3, 21, 12, 30, 0, 0, time.UTC)

	t.Run("every 5 minutes", func(t *testing.T) {
		next := sched.parseCronExpression("*/5 * * * *", now)
		assert.NotNil(t, next)
		assert.Equal(t, 35, next.Minute())
	})

	t.Run("specific time", func(t *testing.T) {
		next := sched.parseCronExpression("0 9 * * *", now)
		assert.NotNil(t, next)
		assert.Equal(t, 9, next.Hour())
		assert.Equal(t, 22, next.Day())
	})

	t.Run("invalid format", func(t *testing.T) {
		next := sched.parseCronExpression("* * *", now)
		assert.Nil(t, next)
	})
}

func TestMatchField(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_match", nil)

	t.Run("wildcard", func(t *testing.T) {
		assert.True(t, sched.matchField("*", 5, 0, 59))
	})

	t.Run("exact match", func(t *testing.T) {
		assert.True(t, sched.matchField("15", 15, 0, 59))
		assert.False(t, sched.matchField("15", 30, 0, 59))
	})

	t.Run("step", func(t *testing.T) {
		assert.True(t, sched.matchField("*/5", 15, 0, 59))
		assert.False(t, sched.matchField("*/5", 17, 0, 59))
	})

	t.Run("range", func(t *testing.T) {
		assert.True(t, sched.matchField("10-20", 15, 0, 59))
		assert.False(t, sched.matchField("10-20", 25, 0, 59))
	})

	t.Run("list", func(t *testing.T) {
		assert.True(t, sched.matchField("1,5,10", 5, 0, 59))
		assert.False(t, sched.matchField("1,5,10", 7, 0, 59))
	})
}

func TestRunTaskNow(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_run", nil)
	err := sched.Init()
	assert.NoError(t, err)

	created, _ := sched.AddTask("run_task", "Run Task", "echo hello", "@daily")

	result, err := sched.RunTaskNow(created.ID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, created.ID, result.TaskID)
}

func TestGetResults(t *testing.T) {
	sched := NewScheduler("/tmp/test_scheduler_results", nil)
	err := sched.Init()
	assert.NoError(t, err)

	created, _ := sched.AddTask("result_task", "Result Task", "echo result", "@daily")
	_, _ = sched.RunTaskNow(created.ID)

	results := sched.GetResults(10)
	assert.NotEmpty(t, results)
}
