package ledger

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

// Ensure TransactionFileRepository implements ports.TransactionRepository at compile time.
var _ ports.TransactionRepository = (*TransactionFileRepository)(nil)

type TransactionFileRepository struct {
	FilePath       string
	configProvider ports.ConfigProvider
	formatter      ports.TransactionFormatter
	mu             sync.Mutex
}

func NewTransactionFileRepository(
	filePath string,
	configProvider ports.ConfigProvider,
	formatter ports.TransactionFormatter,
) (*TransactionFileRepository, error) {
	// Verify external dependency
	if _, err := exec.LookPath("ledger"); err != nil {
		return nil, fmt.Errorf("ledger CLI not found in PATH: %w", err)
	}

	return &TransactionFileRepository{
		FilePath:       filePath,
		configProvider: configProvider,
		formatter:      formatter,
	}, nil
}

func (fileRepository *TransactionFileRepository) Create(transaction domain.Transaction) error {
	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	ledger, err := fileRepository.readLedger()
	if err != nil {
		return err
	}

	codeMarker := fmt.Sprintf("(%s)", transaction.Code)
	for _, entry := range ledger.Entries {
		if entry.Type == domain.EntryTypeTransaction && strings.Contains(entry.RawText, codeMarker) {
			return domain.NewDomainError("Transaction", "Code", "transaction already exists")
		}
	}

	alignment := fileRepository.configProvider.GetLedgerAlignment()
	newRaw := fileRepository.formatter.FormatTransaction(transaction, alignment)
	ledger.Entries = append(
		ledger.Entries, domain.LedgerEntry{
			Date:    transaction.Date,
			RawText: newRaw,
			Type:    domain.EntryTypeTransaction,
		},
	)

	return fileRepository.writeLedger(ledger)
}

func (fileRepository *TransactionFileRepository) FindByCode(code string) (*domain.Transaction, error) {
	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	data, err := os.ReadFile(fileRepository.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	regex := fileRepository.transactionRegex(code)
	if regex.Match(data) {
		return &domain.Transaction{Code: code}, nil
	}

	return nil, nil
}

func (fileRepository *TransactionFileRepository) Update(transaction domain.Transaction) error {
	if transaction.Code == "" {
		return domain.NewDomainError("Transaction", "Code", "transaction must have a code to be updated")
	}

	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	ledger, err := fileRepository.readLedger()
	if err != nil {
		return err
	}

	found := false
	alignment := fileRepository.configProvider.GetLedgerAlignment()
	codeMarker := fmt.Sprintf("(%s)", transaction.Code)

	for i := range ledger.Entries {
		entry := &ledger.Entries[i]
		if entry.Type == domain.EntryTypeTransaction && strings.Contains(entry.RawText, codeMarker) {
			// PRESERVE COMMENTS: We replace only the transaction part, keeping any attached comments.
			// A transaction part ends after the last indented line.
			oldRaw := entry.RawText
			lines := strings.Split(oldRaw, "\n")
			lastTxLine := 0
			for j, line := range lines {
				if j == 0 || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
					lastTxLine = j
				} else {
					break
				}
			}

			newTx := fileRepository.formatter.FormatTransaction(transaction, alignment)
			comments := ""
			if lastTxLine < len(lines)-1 {
				comments = "\n" + strings.Join(lines[lastTxLine+1:], "\n")
			}

			entry.Date = transaction.Date
			entry.RawText = strings.TrimRight(newTx, "\n") + comments
			found = true
			break
		}
	}

	if !found {
		return domain.NewDomainError("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", transaction.Code))
	}

	return fileRepository.writeLedger(ledger)
}

func (fileRepository *TransactionFileRepository) Delete(code string) error {
	if code == "" {
		return domain.NewDomainError("Transaction", "Code", "code must be provided to delete a transaction")
	}

	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	ledger, err := fileRepository.readLedger()
	if err != nil {
		if os.IsNotExist(err) {
			return domain.NewDomainError("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", code))
		}
		return err
	}

	newEntries := make([]domain.LedgerEntry, 0, len(ledger.Entries))
	codeMarker := fmt.Sprintf("(%s)", code)
	found := false

	for _, entry := range ledger.Entries {
		if entry.Type == domain.EntryTypeTransaction && strings.Contains(entry.RawText, codeMarker) {
			// PRESERVE COMMENTS: If a deleted transaction has trailing comments,
			// we should ideally re-attach them to the entry above or keep them as a raw entry.
			// For simplicity, if it has comments, we turn it into a comment-only entry.
			lines := strings.Split(entry.RawText, "\n")
			lastTxLine := 0
			for j, line := range lines {
				if j == 0 || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
					lastTxLine = j
				} else {
					break
				}
			}

			if lastTxLine < len(lines)-1 {
				comments := strings.Join(lines[lastTxLine+1:], "\n")
				newEntries = append(
					newEntries, domain.LedgerEntry{
						Type:    domain.EntryTypeComment,
						RawText: comments,
					},
				)
			}

			found = true
			continue
		}
		newEntries = append(newEntries, entry)
	}

	if !found {
		return domain.NewDomainError("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", code))
	}

	ledger.Entries = newEntries
	return fileRepository.writeLedger(ledger)
}

func (fileRepository *TransactionFileRepository) GetAccounts() ([]string, error) {
	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	if _, err := os.Stat(fileRepository.FilePath); os.IsNotExist(err) {
		return nil, nil
	}

	cmd := exec.Command("ledger", "-f", fileRepository.FilePath, "accounts")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ledger accounts failed: %w (output: %q)", err, string(output))
	}

	lines := strings.Split(string(output), "\n")
	var accounts []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			accounts = append(accounts, trimmed)
		}
	}

	return accounts, nil
}

func (fileRepository *TransactionFileRepository) GetBalanceReport(period string, filter string) (string, error) {
	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	if _, err := os.Stat(fileRepository.FilePath); os.IsNotExist(err) {
		return "", nil
	}

	args := []string{"-f", fileRepository.FilePath, "bal"}
	if period != "" {
		args = append(args, "--period", period)
	}
	if filter != "" {
		args = append(args, filter)
	}

	cmd := exec.Command("ledger", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) == 0 {
			return "", nil
		}
		return "", fmt.Errorf("ledger balance failed: %w (output: %q)", err, string(output))
	}

	return string(output), nil
}

func (fileRepository *TransactionFileRepository) readLedger() (domain.Ledger, error) {
	data, err := os.ReadFile(fileRepository.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.Ledger{}, nil
		}
		return domain.Ledger{}, err
	}

	return domain.ParseLedger(string(data)), nil
}

func (fileRepository *TransactionFileRepository) writeLedger(ledger domain.Ledger) error {
	ledger.Sort()
	content := ledger.Format()
	return fileRepository.atomicWrite([]byte(content))
}

func (fileRepository *TransactionFileRepository) atomicWrite(data []byte) error {
	tmpPath := fileRepository.FilePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, fileRepository.FilePath)
}

func (fileRepository *TransactionFileRepository) transactionRegex(code string) *regexp.Regexp {
	pattern := fmt.Sprintf(`(?m)^\d{4}[\/-]\d{2}[\/-]\d{2}.*\(%s\)(?:.*\n)*?(\r?\n|$)`, regexp.QuoteMeta(code))
	return regexp.MustCompile(pattern)
}
