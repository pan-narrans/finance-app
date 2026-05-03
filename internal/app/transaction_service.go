package app

import (
	"fmt"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

/*
TransactionService coordinates transaction-related business logic.

It validates transactions using domain rules before delegating persistence
to the underlying TransactionRepository.

Methods:
  - Add: Validates and saves a new transaction to the ledger.
  - Update: Validates and replaces an existing transaction in the ledger.
  - Delete: Removes a transaction from the ledger by its unique code.
*/
type TransactionService struct {
	repository ports.TransactionRepository
}

// NewTransactionService creates a new instance of TransactionService.
func NewTransactionService(ledgerRepository ports.TransactionRepository) *TransactionService {
	return &TransactionService{
		repository: ledgerRepository,
	}
}

// Add validates and saves a new transaction.
func (transactionService *TransactionService) Add(transaction domain.Transaction) error {
	// 1. Domain Validation
	if err := transaction.Validate(); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	// 2. Persistence
	if err := transactionService.repository.Create(transaction); err != nil {
		return fmt.Errorf("failed to save transaction: %w", err)
	}

	return nil
}

// Update validates and replaces an existing transaction.
func (transactionService *TransactionService) Update(transaction domain.Transaction) error {
	// 1. Domain Validation
	if err := transaction.Validate(); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	// 2. Persistence
	if err := transactionService.repository.Update(transaction); err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

// Delete removes a transaction by its code.
func (transactionService *TransactionService) Delete(code string) error {
	if code == "" {
		return fmt.Errorf("transaction code must be provided")
	}

	if err := transactionService.repository.Delete(code); err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	return nil
}

// GetByCode Finds a transaction by its code.
func (transactionService *TransactionService) GetByCode(code string) (*domain.Transaction, error) {
	return transactionService.repository.FindByCode(code)
}
