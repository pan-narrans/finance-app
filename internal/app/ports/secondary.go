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
	GetAccounts() ([]string, error)
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
FileParserProvider defines the contract for obtaining the correct parser for a file.
*/
type FileParserProvider interface {
	GetParser(filePath string, parserType string) (BankParser, error)
	GetAvailableParsers() []string
}


/*
MappingProvider defines the contract for description cleaning and account resolution.
*/
type MappingProvider interface {
	CleanDescription(description string) string
	ResolveAccount(description string) (string, bool)
	IsIncomeAccount(account string) bool
	ResolvePayer(fullDescription string) string
	ResolveSource(keyword string) (string, bool)
	SearchAccounts(query string, limit int) []string
	GetAllAccounts() []string
	GetMappingData() domain.MappingData
}

/*
MappingServiceConstructor is a function type that creates a MappingProvider.
*/
type MappingServiceConstructor func(data domain.MappingData, discoveredAccounts []string) MappingProvider

/*
TransactionFormatter defines the contract for converting a transaction into a tool-specific string.
*/
type TransactionFormatter interface {
	FormatTransaction(tx domain.Transaction, alignment int) string
}

/*
ConfigProvider defines a narrow interface for adapters to access necessary application settings.
This helps break circular dependencies with the full ConfigurationUseCase.
*/
type ConfigProvider interface {
	GetLedgerAlignment() int
}

/*
ReportProvider defines the contract for generating financial reports.
*/
type ReportProvider interface {
	GetBalanceReport(period string, filter string) (string, error)
}
