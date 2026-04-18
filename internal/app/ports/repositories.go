package ports

import "github.com/a-perez/finance-app/internal/domain"

/*
TransactionRepository defines the contract for persisting transactions.

Methods:
  - Create: Persists a new transaction to the storage.
  - FindByCode: Retrieves a transaction by its unique reference code.
  - Update: Replaces an existing transaction with new data.
  - Delete: Removes a transaction from storage using its unique code.
*/
type TransactionRepository interface {
	Create(transaction domain.Transaction) error
	FindByCode(code string) (*domain.Transaction, error)
	Update(transaction domain.Transaction) error
	Delete(code string) error
}
