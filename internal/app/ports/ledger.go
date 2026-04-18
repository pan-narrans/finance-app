package ports

import "github.com/a-perez/finance-app/internal/domain"

// LedgerRepository defines the contract for persisting transactions to a ledger file.
type LedgerRepository interface {
	Append(transaction domain.Transaction) error
	FindByCode(code string) (*domain.Transaction, error)
	Update(transaction domain.Transaction) error
	Delete(code string) error
}
