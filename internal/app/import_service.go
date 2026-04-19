package app

import (
	"fmt"

	"github.com/a-perez/finance-app/internal/app/ports"
)

// ImportService orchestrates the process of parsing and persisting transactions from external files.
type ImportService struct {
	transactionRepository ports.TransactionRepository
}

// NewImportService creates a new instance of ImportService.
func NewImportService(transactionRepository ports.TransactionRepository) *ImportService {
	return &ImportService{
		transactionRepository: transactionRepository,
	}
}

// Import parses a file using the provided parser and saves the resulting transactions to the repository.
func (importService *ImportService) Import(parser ports.BankParser, filePath string) error {
	transactions, err := parser.Parse(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	for _, transaction := range transactions {
		if err := transaction.Validate(); err != nil {
			return fmt.Errorf("invalid transaction from file: %w", err)
		}

		if err := importService.transactionRepository.Create(transaction); err != nil {
			return fmt.Errorf("failed to save transaction: %w", err)
		}
	}

	return nil
}
