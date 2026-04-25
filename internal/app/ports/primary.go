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
