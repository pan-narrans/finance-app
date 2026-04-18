package ports

import "github.com/a-perez/finance-app/internal/domain"

// TransactionRepository defines the contract for persisting transactions.
type TransactionRepository interface {
	Create(transaction domain.Transaction) error
	FindByCode(code string) (*domain.Transaction, error)
	Update(transaction domain.Transaction) error
	Delete(code string) error
}
