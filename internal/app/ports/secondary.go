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

/*
BankParser defines the contract for parsing bank-specific files into domain transactions.

Methods:
  - Parse: Reads a bank-specific file and returns a list of domain transactions.
*/
type BankParser interface {
	Parse(filePath string) ([]domain.Transaction, error)
}

/*
BackupService defines the contract for creating and restoring backups of the ledger file.

Methods:
  - CreateBackup: Creates a compressed backup of the specified file. Returns a session ID.
  - RestoreLast: Restores the most recent backup to the target path.
  - SaveDiff: Generates and saves a unified diff between the backup and the current file.
*/
type BackupService interface {
	CreateBackup(filePath string) (sessionID string, err error)
	RestoreLast(targetPath string) error
	SaveDiff(sessionID string, currentPath string) error
}
