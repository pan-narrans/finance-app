package ledger

import (
	"fmt"
	"os"
	"regexp"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

// Ensure TransactionFileRepository implements ports.TransactionRepository at compile time.
var _ ports.TransactionRepository = (*TransactionFileRepository)(nil)

// TransactionFileRepository implements ports.TransactionRepository for a plain-text file.
type TransactionFileRepository struct {
	FilePath string
}

// NewTransactionFileRepository creates a new instance of TransactionFileRepository.
func NewTransactionFileRepository(filePath string) *TransactionFileRepository {
	return &TransactionFileRepository{
		FilePath: filePath,
	}
}

// Create writes a transaction to the end of the ledger file.
func (fileRepository *TransactionFileRepository) Create(transaction domain.Transaction) error {
	content := transaction.Format()
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
	data, err := os.ReadFile(fileRepository.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Regex to match the transaction block with the given (CODE)
	pattern := fmt.Sprintf(`(?m)^\d{4}[\/-]\d{2}[\/-]\d{2}.*\(%s\)(?:.*\n)*?(\r?\n|$)`, regexp.QuoteMeta(code))
	regex := regexp.MustCompile(pattern)

	if regex.Match(data) {
		// Return a shell transaction with the found code.
		// Future: parse the full block back into a domain.Transaction.
		return &domain.Transaction{Code: code}, nil
	}

	return nil, nil
}

// Update replaces an existing transaction block with a new formatted version.
// It returns a domain.DomainError if the transaction code is not found in the file.
func (fileRepository *TransactionFileRepository) Update(transaction domain.Transaction) error {
	if transaction.Code == "" {
		return domain.NewValidationErrors("Transaction", "Code", "transaction must have a code to be updated")
	}

	data, err := os.ReadFile(fileRepository.FilePath)
	if err != nil {
		return err
	}

	// Dynamic regex to find the transaction by its CODE
	pattern := fmt.Sprintf(`(?m)^\d{4}[\/-]\d{2}[\/-]\d{2}.*\(%s\)(?:.*\n)*?(\r?\n|$)`, regexp.QuoteMeta(transaction.Code))
	regex := regexp.MustCompile(pattern)

	// Verify the transaction exists before trying to update it
	if !regex.Match(data) {
		return domain.NewValidationErrors("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", transaction.Code))
	}

	// Replace the old block with the new formatted one
	newContent := transaction.Format() + "\n"
	updatedData := regex.ReplaceAllString(string(data), newContent)

	// Write the entire file back
	return os.WriteFile(fileRepository.FilePath, []byte(updatedData), 0644)
}

// Delete removes a transaction block from the file by its code.
func (fileRepository *TransactionFileRepository) Delete(code string) error {
	if code == "" {
		return domain.NewValidationErrors("Transaction", "Code", "code must be provided to delete a transaction")
	}

	data, err := os.ReadFile(fileRepository.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.NewValidationErrors("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", code))
		}
		return err
	}

	// Dynamic regex to find the transaction by its CODE
	pattern := fmt.Sprintf(`(?m)^\d{4}[\/-]\d{2}[\/-]\d{2}.*\(%s\)(?:.*\n)*?(\r?\n|$)`, regexp.QuoteMeta(code))
	regex := regexp.MustCompile(pattern)

	// Verify the transaction exists before trying to delete it
	if !regex.Match(data) {
		return domain.NewValidationErrors("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", code))
	}

	// Remove the block
	updatedData := regex.ReplaceAllString(string(data), "")

	// Write the entire file back
	return os.WriteFile(fileRepository.FilePath, []byte(updatedData), 0644)
}
