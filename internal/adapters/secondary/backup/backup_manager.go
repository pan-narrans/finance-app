package backup

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/pmezard/go-difflib/difflib"
)

var _ ports.BackupService = (*BackupManager)(nil)

/*
BackupManager handles file-based backups and unified diff generation.
It stores backups in a specified directory using gzip compression.
*/
type BackupManager struct {
	BackupDir string
}

// NewBackupManager creates a new instance of BackupManager.
func NewBackupManager(backupDir string) *BackupManager {
	return &BackupManager{
		BackupDir: backupDir,
	}
}

// CreateBackup creates a compressed backup of the specified file.
func (backupManager *BackupManager) CreateBackup(filePath string) (string, error) {
	if err := os.MkdirAll(backupManager.BackupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	sessionID := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(backupManager.BackupDir, fmt.Sprintf("import-%s.bak.gz", sessionID))

	source, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	destination, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer destination.Close()

	gw := gzip.NewWriter(destination)
	defer gw.Close()

	if _, err := io.Copy(gw, source); err != nil {
		return "", fmt.Errorf("failed to compress file: %w", err)
	}

	return sessionID, nil
}

// RestoreLast restores the most recent backup to the target path.
func (backupManager *BackupManager) RestoreLast(targetPath string) error {
	backups, err := backupManager.listBackups()
	if err != nil {
		return err
	}

	if len(backups) == 0 {
		return fmt.Errorf("no backups found in %s", backupManager.BackupDir)
	}

	// Backups are named by timestamp, so the last one is the most recent.
	lastBackup := backups[len(backups)-1]
	backupPath := filepath.Join(backupManager.BackupDir, lastBackup)

	return backupManager.restoreFile(backupPath, targetPath)
}

// SaveDiff generates and saves a unified diff between the backup and the current file.
func (backupManager *BackupManager) SaveDiff(sessionID string, currentPath string) error {
	backupPath := filepath.Join(backupManager.BackupDir, fmt.Sprintf("import-%s.bak.gz", sessionID))
	patchPath := filepath.Join(backupManager.BackupDir, fmt.Sprintf("import-%s.patch", sessionID))

	oldContent, err := backupManager.readGzipFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup for diff: %w", err)
	}

	newContent, err := os.ReadFile(currentPath)
	if err != nil {
		return fmt.Errorf("failed to read current file for diff: %w", err)
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(oldContent)),
		B:        difflib.SplitLines(string(newContent)),
		FromFile: "Before Import",
		ToFile:   "After Import",
		Context:  3,
	}

	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return fmt.Errorf("failed to generate diff string: %w", err)
	}

	if err := os.WriteFile(patchPath, []byte(text), 0644); err != nil {
		return fmt.Errorf("failed to save patch file: %w", err)
	}

	return nil
}

func (backupManager *BackupManager) listBackups() ([]string, error) {
	files, err := os.ReadDir(backupManager.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var backups []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".bak.gz") {
			backups = append(backups, file.Name())
		}
	}

	sort.Strings(backups)
	return backups, nil
}

func (backupManager *BackupManager) restoreFile(backupPath, targetPath string) error {
	source, err := os.Open(backupPath)
	if err != nil {
		return err
	}
	defer source.Close()

	gr, err := gzip.NewReader(source)
	if err != nil {
		return err
	}
	defer gr.Close()

	destination, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, gr); err != nil {
		return err
	}

	return nil
}

func (backupManager *BackupManager) readGzipFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	return io.ReadAll(gr)
}
