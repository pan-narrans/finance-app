package app

import (
	"fmt"

	"github.com/a-perez/finance-app/internal/app/ports"
)

// ImportService orchestrates the process of parsing and persisting transactions from external files.
type ImportService struct {
	transactionUseCase ports.TransactionUseCase
}

// NewImportService creates a new instance of ImportService.
func NewImportService(transactionUseCase ports.TransactionUseCase) *ImportService {
	return &ImportService{
		transactionUseCase: transactionUseCase,
	}
}

// Import parses a file using the provided parser and saves the resulting transactions.
func (importService *ImportService) Import(parser ports.BankParser, filePath string) error {
	transactions, err := parser.Parse(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	for _, transaction := range transactions {
		if transaction.Code == "" {
			transaction.Code = transaction.GenerateCode()
		}

		// TODO I do not like this erorr management
		existing, err := importService.transactionUseCase.GetByCode(transaction.Code)
		if err != nil {
			return fmt.Errorf("failed to check existing transaction: %w", err)
		}

		if existing != nil {
			if err := importService.transactionUseCase.Update(transaction); err != nil {
				return fmt.Errorf("failed to update existing transaction: %w", err)
			}
		} else {
			if err := importService.transactionUseCase.Add(transaction); err != nil {
				return fmt.Errorf("failed to add new transaction: %w", err)
			}
		}
	}

	return nil
}
