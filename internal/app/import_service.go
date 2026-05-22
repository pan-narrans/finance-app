package app

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

// Ensure ImportService implements ports.ImportUseCase at compile time.
var _ ports.ImportUseCase = (*ImportService)(nil)

// ImportService orchestrates the process of parsing and persisting transactions from external files.
type ImportService struct {
	transactionUseCase ports.TransactionUseCase
	parserProvider     ports.FileParserProvider
}

// NewImportService creates a new instance of ImportService.
func NewImportService(transactionUseCase ports.TransactionUseCase, parserProvider ports.FileParserProvider) *ImportService {
	return &ImportService{
		transactionUseCase: transactionUseCase,
		parserProvider:     parserProvider,
	}
}

/*
Import finds correct parser for file, parses it, and saves resulting transactions.
*/
func (importService *ImportService) Import(filePath string) (*ports.ImportSummary, error) {
	parser, err := importService.parserProvider.GetParser(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get parser: %w", err)
	}

	transactions, err := parser.Parse(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Sort transactions chronologically (oldest first).
	// Use Stable sort to preserve original order for transactions on the same day.
	slices.SortStableFunc(
		transactions, func(a, b domain.Transaction) int {
			return cmp.Compare(a.Date.Unix(), b.Date.Unix())
		},
	)

	summary := &ports.ImportSummary{
		Total:   len(transactions),
		Errors:  make(map[int]error),
		Pending: []domain.Transaction{},
	}

	for index, transaction := range transactions {
		if transaction.HasUnknownAccount() {
			summary.Pending = append(summary.Pending, transaction)
			continue
		}

		if err = importService.processTransaction(transaction, summary); err != nil {
			summary.Failed++
			summary.Errors[index] = err
		}
	}

	return summary, nil
}

func (importService *ImportService) processTransaction(transaction domain.Transaction, summary *ports.ImportSummary) error {
	if transaction.Code == "" {
		transaction.Code = transaction.GenerateCode()
	}
	existing, err := importService.transactionUseCase.GetByCode(transaction.Code)

	if err != nil {
		return fmt.Errorf("lookup failed: %w", err)
	}

	if existing != nil {
		if err := importService.transactionUseCase.Update(transaction); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		summary.Updated++
	} else {
		if err := importService.transactionUseCase.Add(transaction); err != nil {
			return fmt.Errorf("add failed: %w", err)
		}
		summary.Added++
	}

	return nil
}
