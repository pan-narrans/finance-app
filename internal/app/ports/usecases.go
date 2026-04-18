package ports

import "github.com/a-perez/finance-app/internal/domain"

// TransactionUseCase defines the contract for transaction business logic.
type TransactionUseCase interface {
	Add(transaction domain.Transaction) error
	Update(transaction domain.Transaction) error
	Delete(code string) error
}
