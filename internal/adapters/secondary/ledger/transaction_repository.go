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

/*
TransactionFileRepository implements ports.TransactionRepository using a plain-text file.

It uses regex-based parsing to interact with transactions in the Ledger CLI format.

Methods:
  - Create: Appends a new transaction to the end of the ledger file.
  - FindByCode: Searches the file for a transaction with the given unique code.
  - Update: Replaces an existing transaction block in the file with a new formatted version.
  - Delete: Removes a transaction block from the file by its unique code.
*/
type TransactionFileRepository struct {
	FilePath      string
	configUseCase ports.ConfigurationUseCase
	formatter     ports.TransactionFormatter
	mu            sync.Mutex
}

// NewTransactionFileRepository creates a new instance of TransactionFileRepository.
func NewTransactionFileRepository(filePath string, configUC ports.ConfigurationUseCase, formatter ports.TransactionFormatter) *TransactionFileRepository {
	return &TransactionFileRepository{
		FilePath:      filePath,
		configUseCase: configUC,
		formatter:     formatter,
	}
}

// Create writes a transaction to the end of the ledger file.
func (fileRepository *TransactionFileRepository) Create(transaction domain.Transaction) error {
	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	alignment := fileRepository.configUseCase.Get().Settings.LedgerAlignment
	content := fileRepository.formatter.FormatTransaction(transaction, alignment)
	content += "\n"

	file, err := os.OpenFile(fileRepository.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return err
	}

	return nil
}

// FindByCode searches the file using a regex to find a transaction with the given code.
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

/*
Update replaces an existing transaction block with a new formatted version.
It returns a domain.DomainError if the transaction code is not found in the file.
*/
func (fileRepository *TransactionFileRepository) Update(transaction domain.Transaction) error {
	if transaction.Code == "" {
		return domain.NewDomainError("Transaction", "Code", "transaction must have a code to be updated")
	}

	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	data, err := os.ReadFile(fileRepository.FilePath)
	if err != nil {
		return err
	}

	regex := fileRepository.transactionRegex(transaction.Code)

	if !regex.Match(data) {
		return domain.NewDomainError("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", transaction.Code))
	}

	alignment := fileRepository.configUseCase.Get().Settings.LedgerAlignment
	newContent := fileRepository.formatter.FormatTransaction(transaction, alignment) + "\n"
	updatedData := regex.ReplaceAllString(string(data), newContent)

	return os.WriteFile(fileRepository.FilePath, []byte(updatedData), 0644)
}

func (fileRepository *TransactionFileRepository) Delete(code string) error {
	if code == "" {
		return domain.NewDomainError("Transaction", "Code", "code must be provided to delete a transaction")
	}

	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	data, err := os.ReadFile(fileRepository.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.NewDomainError("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", code))
		}
		return err
	}

	regex := fileRepository.transactionRegex(code)

	if !regex.Match(data) {
		return domain.NewDomainError("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", code))
	}

	updatedData := regex.ReplaceAllString(string(data), "")

	return os.WriteFile(fileRepository.FilePath, []byte(updatedData), 0644)
}

// GetAccounts retrieves the list of accounts from the ledger file.
// It combines accounts found via 'ledger accounts' and manual parsing of 'account' declarations.
func (fileRepository *TransactionFileRepository) GetAccounts() ([]string, error) {
	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	// 1. Get accounts with postings via ledger CLI
	cmd := exec.Command("ledger", "-f", fileRepository.FilePath, "accounts")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If file doesn't exist yet, return empty list instead of error
		if os.IsNotExist(err) {
			return nil, nil
		}
		// We still try to parse declarations even if ledger fails (e.g. syntax error in transactions)
	}

	accountMap := make(map[string]bool)

	// Add accounts from ledger output
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				accountMap[trimmed] = true
			}
		}
	}

	// 2. Manual parsing for 'account' declarations (even if unused or file has errors)
	data, readErr := os.ReadFile(fileRepository.FilePath)
	if readErr == nil {
		// Regex to match: account Expenses:Something
		re := regexp.MustCompile(`(?m)^account\s+([^\n\r;]+)`)
		matches := re.FindAllStringSubmatch(string(data), -1)
		for _, m := range matches {
			if len(m) > 1 {
				accountMap[strings.TrimSpace(m[1])] = true
			}
		}
	}

	var accounts []string
	for acc := range accountMap {
		accounts = append(accounts, acc)
	}

	// If both failed and we have no accounts, return the ledger error
	if len(accounts) == 0 && err != nil {
		return nil, fmt.Errorf("ledger accounts failed: %w (output: %q)", err, string(output))
	}

	return accounts, nil
}

/*
transactionRegex compiles a regular expression to match a transaction block
by its unique code. It looks for the DATE followed by the (CODE) marker.
*/
func (fileRepository *TransactionFileRepository) transactionRegex(code string) *regexp.Regexp {
	// (?m) enables multi-line mode.
	// We match from the date line to the next blank line or end of file.
	pattern := fmt.Sprintf(`(?m)^\d{4}[\/-]\d{2}[\/-]\d{2}.*\(%s\)(?:.*\n)*?(\r?\n|$)`, regexp.QuoteMeta(code))
	return regexp.MustCompile(pattern)
}
