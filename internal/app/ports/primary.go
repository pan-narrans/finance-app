package ports

import "github.com/a-perez/finance-app/internal/domain"

/*
TransactionUseCase defines the core business logic for managing transactions.

Methods:
  - Add: Validates and records a new transaction.
  - Update: Validates and modifies an existing transaction.
  - Delete: Removes a transaction by its code.
*/
type TransactionUseCase interface {
	Add(transaction domain.Transaction) error
	Update(transaction domain.Transaction) error
	Delete(code string) error
	GetByCode(code string) (*domain.Transaction, error)
}

// ImportSummary tracks the outcome of an import process.
type ImportSummary struct {
	Total   int
	Added   int
	Updated int
	Failed  int
	Errors  map[int]error
}

// ImportUseCase defines the orchestrator for bank file imports.
type ImportUseCase interface {
	Import(parser BankParser, filePath string) (*ImportSummary, error)
	RollbackLastImport() error
}
