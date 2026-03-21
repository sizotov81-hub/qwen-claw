package selfimprovement

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine("/tmp/test_selfimp")
	assert.NotNil(t, engine)
	assert.Equal(t, "/tmp/test_selfimp", engine.projectRoot)
	assert.NotEmpty(t, engine.backupDir)
	assert.NotNil(t, engine.config)
}

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()
	assert.NotNil(t, config)

	assert.Contains(t, config.ConfirmationRequired, "system_packages")
	assert.Contains(t, config.ConfirmationRequired, "code_changes")
	assert.Contains(t, config.ConfirmationRequired, "restart")

	assert.Contains(t, config.AutoAllowed, "go_packages")
	assert.Contains(t, config.AutoAllowed, "go_test")
	assert.Contains(t, config.AutoAllowed, "go_build")

	assert.Contains(t, config.Forbidden, "rm_rf_root")
	assert.Contains(t, config.Forbidden, "chmod_777")

	assert.True(t, config.RollbackEnabled)
	assert.True(t, config.RequireBackup)
}

func TestRequestChange(t *testing.T) {
	engine := NewEngine("/tmp/test_request")

	t.Run("requires confirmation", func(t *testing.T) {
		change := engine.RequestChange(
			"code_changes",
			"Add new feature",
			"Testing",
			[]string{"echo test"},
			nil,
		)

		assert.NotNil(t, change)
		assert.Equal(t, "code_changes", change.Type)
		assert.True(t, change.RequiresConfirmation)
		assert.Equal(t, "pending", change.Status)
	})

	t.Run("auto allowed", func(t *testing.T) {
		change := engine.RequestChange(
			"go_packages",
			"Install package",
			"Testing",
			[]string{"go get test"},
			nil,
		)

		assert.NotNil(t, change)
		assert.False(t, change.RequiresConfirmation)
	})

	t.Run("forbidden action", func(t *testing.T) {
		change := engine.RequestChange(
			"dangerous",
			"Delete all",
			"Testing",
			[]string{"rm -rf /"},
			nil,
		)

		assert.NotNil(t, change)
		assert.Equal(t, "rejected", change.Status)
	})
}

func TestApprove(t *testing.T) {
	engine := NewEngine("/tmp/test_approve")

	change := engine.RequestChange(
		"go_packages",
		"Install package",
		"Testing",
		[]string{"echo install"},
		nil,
	)

	err := engine.Approve(change.ID)
	assert.NoError(t, err)
}

func TestReject(t *testing.T) {
	engine := NewEngine("/tmp/test_reject")

	change := engine.RequestChange(
		"code_changes",
		"Change code",
		"Testing",
		[]string{"echo change"},
		nil,
	)

	err := engine.Reject(change.ID)
	assert.NoError(t, err)
	assert.Equal(t, "rejected", change.Status)
}

func TestGetPendingChanges(t *testing.T) {
	engine := NewEngine("/tmp/test_pending")

	_ = engine.RequestChange("code_changes", "Change 1", "Test", []string{"echo 1"}, nil)
	_ = engine.RequestChange("code_changes", "Change 2", "Test", []string{"echo 2"}, nil)

	pending := engine.GetPendingChanges()
	assert.Len(t, pending, 2)
}

func TestFindChange(t *testing.T) {
	engine := NewEngine("/tmp/test_find_change")

	change := engine.RequestChange("go_packages", "Test", "Test", []string{"echo"}, nil)

	found := engine.findChange(change.ID)
	assert.NotNil(t, found)
	assert.Equal(t, change.ID, found.ID)

	notFound := engine.findChange("nonexistent")
	assert.Nil(t, notFound)
}

func TestRemovePending(t *testing.T) {
	engine := NewEngine("/tmp/test_remove_pending")

	change := engine.RequestChange("go_packages", "Test", "Test", []string{"echo"}, nil)
	engine.removePending(change.ID)

	pending := engine.GetPendingChanges()
	assert.Empty(t, pending)
}

func TestExecuteCommand(t *testing.T) {
	engine := NewEngine("/tmp/test_exec_cmd")

	t.Run("successful command", func(t *testing.T) {
		err := engine.executeCommand("echo test")
		assert.NoError(t, err)
	})

	t.Run("failing command", func(t *testing.T) {
		err := engine.executeCommand("false")
		assert.Error(t, err)
	})

	t.Run("invalid command", func(t *testing.T) {
		err := engine.executeCommand("nonexistent_command_xyz")
		assert.Error(t, err)
	})
}

func TestApplyFileChange(t *testing.T) {
	engine := NewEngine("/tmp/test_file_change")
	tmpFile := filepath.Join("/tmp", "test_change.txt")

	t.Run("create file", func(t *testing.T) {
		change := FileChange{
			Path:       tmpFile,
			Action:     "create",
			NewContent: "test content",
		}

		err := engine.applyFileChange(change)
		assert.NoError(t, err)

		_, err = os.Stat(tmpFile)
		assert.NoError(t, err)
	})

	t.Run("modify file", func(t *testing.T) {
		change := FileChange{
			Path:       tmpFile,
			Action:     "modify",
			OldContent: "test content",
			NewContent: "modified content",
		}

		err := engine.applyFileChange(change)
		assert.NoError(t, err)

		data, _ := os.ReadFile(tmpFile)
		assert.Contains(t, string(data), "modified")
	})

	t.Run("delete file", func(t *testing.T) {
		change := FileChange{
			Path:   tmpFile,
			Action: "delete",
		}

		err := engine.applyFileChange(change)
		assert.NoError(t, err)

		_, err = os.Stat(tmpFile)
		assert.Error(t, err)
	})

	t.Run("unknown action", func(t *testing.T) {
		change := FileChange{
			Path:   "/tmp/test.txt",
			Action: "unknown",
		}

		err := engine.applyFileChange(change)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action")
	})
}

func TestCreateBackup(t *testing.T) {
	engine := NewEngine("/tmp/test_backup")

	change := &ChangeRequest{
		ID:          "test_change_123",
		Description: "Test backup",
		Files: []FileChange{
			{
				Path:       "/tmp/backup_test.txt",
				Action:     "modify",
				NewContent: "test",
			},
		},
	}

	os.WriteFile(change.Files[0].Path, []byte("original"), 0644)

	backupID, err := engine.createBackup(change)
	assert.NoError(t, err)
	assert.NotEmpty(t, backupID)

	_, err = os.Stat(filepath.Join(engine.backupDir, backupID))
	assert.NoError(t, err)
}

func TestRollback(t *testing.T) {
	engine := NewEngine("/tmp/test_rollback")

	change := &ChangeRequest{
		ID:          "test_rollback_123",
		Description: "Test rollback",
		Files: []FileChange{
			{
				Path:       "/tmp/rollback_test.txt",
				Action:     "modify",
				OldContent: "original",
				NewContent: "modified",
			},
		},
	}

	os.WriteFile(change.Files[0].Path, []byte("original"), 0644)
	backupID, _ := engine.createBackup(change)

	os.WriteFile(change.Files[0].Path, []byte("changed"), 0644)

	err := engine.rollback(backupID)
	assert.NoError(t, err)

	data, _ := os.ReadFile(change.Files[0].Path)
	assert.Contains(t, string(data), "original")
}

func TestLogChange(t *testing.T) {
	engine := NewEngine("/tmp/test_log")

	change := &ChangeRequest{
		ID:          "test_log_123",
		Description: "Test logging",
		Status:      "pending",
	}

	assert.NotPanics(t, func() {
		engine.logChange(change, "TESTED")
	})

	_, err := os.Stat(engine.logFile)
	assert.NoError(t, err)
}

func TestLoadLog(t *testing.T) {
	engine := NewEngine("/tmp/test_loadlog")

	change := &ChangeRequest{
		ID:          "test_load_123",
		Description: "Test load log",
		Status:      "completed",
	}

	engine.logChange(change, "COMPLETED")

	changes := engine.loadLog()
	assert.NotEmpty(t, changes)
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}

func TestFormatChangeRequest(t *testing.T) {
	change := &ChangeRequest{
		ID:          "format_test",
		Type:        "code_changes",
		Description: "Test formatting",
		Reason:      "Testing",
		Commands:    []string{"echo test"},
		Files: []FileChange{
			{Path: "test.go", Action: "modify"},
		},
		RequiresConfirmation: true,
	}

	formatted := FormatChangeRequest(change)
	assert.NotEmpty(t, formatted)
	assert.Contains(t, formatted, "Test formatting")
	assert.Contains(t, formatted, "ПОДТВЕРЖДАЮ")
}

func TestIsForbidden(t *testing.T) {
	engine := NewEngine("/tmp/test_forbidden")

	t.Run("forbidden command", func(t *testing.T) {
		change := &ChangeRequest{
			Commands: []string{"rm -rf /"},
		}
		assert.True(t, engine.isForbidden(change))
	})

	t.Run("forbidden file", func(t *testing.T) {
		change := &ChangeRequest{
			Files: []FileChange{
				{Path: "rm_rf_root"},
			},
		}
		assert.True(t, engine.isForbidden(change))
	})

	t.Run("allowed", func(t *testing.T) {
		change := &ChangeRequest{
			Commands: []string{"echo safe"},
		}
		assert.False(t, engine.isForbidden(change))
	})
}

func TestRequiresConfirmation(t *testing.T) {
	engine := NewEngine("/tmp/test_confirm")

	assert.True(t, engine.requiresConfirmation("system_packages"))
	assert.True(t, engine.requiresConfirmation("code_changes"))
	assert.False(t, engine.requiresConfirmation("go_packages"))
}

func TestExecuteChange(t *testing.T) {
	engine := NewEngine("/tmp/test_exec_change")

	change := &ChangeRequest{
		ID:          "exec_test",
		Type:        "go_packages",
		Description: "Test execution",
		Commands:    []string{"echo executing"},
		Files:       []FileChange{},
		Status:      "approved",
	}

	err := engine.executeChange(change)
	assert.NoError(t, err)
	assert.Equal(t, "completed", change.Status)
}

func TestExecuteChangeWithRollback(t *testing.T) {
	engine := NewEngine("/tmp/test_exec_rollback")
	engine.config.RollbackEnabled = true

	tmpFile := filepath.Join("/tmp", "rollback_exec_test.txt")
	os.WriteFile(tmpFile, []byte("original"), 0644)

	change := &ChangeRequest{
		ID:          "rollback_exec_test",
		Type:        "code_changes",
		Description: "Test rollback on failure",
		Commands:    []string{"false"},
		Files: []FileChange{
			{
				Path:       tmpFile,
				Action:     "modify",
				OldContent: "original",
				NewContent: "modified",
			},
		},
		Status: "approved",
	}

	err := engine.executeChange(change)
	assert.Error(t, err)
	assert.Equal(t, "failed", change.Status)
}
