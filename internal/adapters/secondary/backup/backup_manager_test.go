package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackupManager_CreateAndRestore_ShouldRecoverOriginalContent(t *testing.T) {
	// Arrange
	tmpDir, err := os.MkdirTemp("", "backup_test_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	ledgerPath := filepath.Join(tmpDir, "book.ledger")
	content := "2026/01/01 Shopping\n    Expenses:Food    10.00 EUR\n    Assets:Cash\n"
	err = os.WriteFile(ledgerPath, []byte(content), 0644)
	assert.NoError(t, err)

	manager := NewBackupManager(backupDir)

	// Act - Backup
	sessionID, err := manager.CreateBackup(ledgerPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, sessionID)

	// Modify ledger
	err = os.WriteFile(ledgerPath, []byte("corrupted content"), 0644)
	assert.NoError(t, err)

	// Act - Restore
	err = manager.RestoreLast(ledgerPath)
	assert.NoError(t, err)

	// Assert
	restoredContent, err := os.ReadFile(ledgerPath)
	assert.NoError(t, err)
	assert.Equal(t, content, string(restoredContent))
}

func TestBackupManager_SaveDiff_ShouldGenerateHumanReadablePatch(t *testing.T) {
	// Arrange
	tmpDir, err := os.MkdirTemp("", "diff_test_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	backupDir := filepath.Join(tmpDir, "backups")
	ledgerPath := filepath.Join(tmpDir, "book.ledger")

	oldContent := "Line 1\nLine 2\n"
	newContent := "Line 1\nLine 2 changed\nLine 3\n"

	err = os.WriteFile(ledgerPath, []byte(oldContent), 0644)
	assert.NoError(t, err)

	manager := NewBackupManager(backupDir)
	sessionID, _ := manager.CreateBackup(ledgerPath)

	err = os.WriteFile(ledgerPath, []byte(newContent), 0644)
	assert.NoError(t, err)

	// Act
	err = manager.SaveDiff(sessionID, ledgerPath)
	assert.NoError(t, err)

	// Assert
	patchFile := filepath.Join(backupDir, "import-"+sessionID+".patch")
	patchContent, err := os.ReadFile(patchFile)
	assert.NoError(t, err)

	patchStr := string(patchContent)
	assert.Contains(t, patchStr, "Line 2 changed")
	assert.Contains(t, patchStr, "Line 3")
	assert.Contains(t, patchStr, "--- Before Import")
	assert.Contains(t, patchStr, "+++ After Import")
}

func TestBackupManager_RestoreLast_ShouldFail_WhenNoBackupsExist(t *testing.T) {
	// Arrange
	tmpDir, _ := os.MkdirTemp("", "fail_test_*")
	defer os.RemoveAll(tmpDir)
	manager := NewBackupManager(filepath.Join(tmpDir, "empty"))

	// Act
	err := manager.RestoreLast(filepath.Join(tmpDir, "anything"))

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no backups found")
}

func TestBackupManager_ListBackups_ShouldReturnInChronologicalOrder(t *testing.T) {
	// Arrange
	tmpDir, _ := os.MkdirTemp("", "sort_test_*")
	defer os.RemoveAll(tmpDir)
	backupDir := filepath.Join(tmpDir, "backups")
	_ = os.MkdirAll(backupDir, 0755)

	// Create dummy backup files with specific timestamps
	files := []string{
		"import-20260101-120000.bak.gz",
		"import-20260101-100000.bak.gz",
		"import-20260101-110000.bak.gz",
	}
	for _, f := range files {
		_ = os.WriteFile(filepath.Join(backupDir, f), []byte("dummy"), 0644)
	}

	manager := NewBackupManager(backupDir)

	// Act
	backups, err := manager.listBackups()
	assert.NoError(t, err)

	// Assert - should be sorted alphabetically (which is chronological for our format)
	assert.Len(t, backups, 3)
	assert.True(t, strings.Contains(backups[0], "100000"))
	assert.True(t, strings.Contains(backups[1], "110000"))
	assert.True(t, strings.Contains(backups[2], "120000"))
}
