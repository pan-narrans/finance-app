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
func (importService *ImportService) Import(filePath string, parserType string) (*ports.ImportSummary, error) {
	parser, err := importService.parserProvider.GetParser(filePath, parserType)
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
		if transaction.Code == "" {
			transaction.Code = transaction.GenerateCode()
		}

		existing, err := importService.transactionUseCase.GetByCode(transaction.Code)
		if err != nil {
			summary.Failed++
			summary.Errors[index] = fmt.Errorf("lookup failed: %w", err)
			continue
		}

		if existing != nil {
			// If already in ledger, we skip if the new parse is still Unknown
			// (to avoid overwriting user corrections with "Unknown" again).
			if transaction.HasUnknownAccount() {
				summary.Updated++
				continue
			}

			// If it's fully known now (maybe user added mapping), we update it.
			if err = importService.transactionUseCase.Update(transaction); err != nil {
				summary.Failed++
				summary.Errors[index] = fmt.Errorf("update failed: %w", err)
			} else {
				summary.Updated++
			}
			continue
		}

		// New transaction: check for unknown accounts
		if transaction.HasUnknownAccount() {
			summary.Pending = append(summary.Pending, transaction)
			continue
		}

		// Fully known and new: Add it
		if err = importService.transactionUseCase.Add(transaction); err != nil {
			summary.Failed++
			summary.Errors[index] = fmt.Errorf("add failed: %w", err)
		} else {
			summary.Added++
		}
	}

	return summary, nil
}

// GetAvailableBanks returns the list of supported bank identifiers.
func (importService *ImportService) GetAvailableBanks() []string {
	return importService.parserProvider.GetAvailableParsers()
}
