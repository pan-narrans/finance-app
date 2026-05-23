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
	Pending []domain.Transaction
}

// ImportUseCase defines the orchestrator for bank file imports.
type ImportUseCase interface {
	Import(filePath string) (*ImportSummary, error)
}

/*
AppConfig combines application settings and the derived mapping service.
It represents a single, consistent snapshot of the application configuration.
*/
type AppConfig struct {
	Settings domain.Settings
	Mappings MappingProvider
}

/*
ConfigurationUseCase defines the contract for accessing and updating application configuration.
*/
type ConfigurationUseCase interface {
	Get() *AppConfig
	SaveMappings(data domain.MappingData) error
	UpdateMapping(fn func(data *domain.MappingData)) error
	LearnMapping(transaction domain.Transaction, targetOverride bool, sourceOverride bool, originalSource string) error
}

/*
TransactionParserUseCase defines the logic for converting raw input strings into domain transactions.
*/
type TransactionParserUseCase interface {
	ParseText(text, origin string) (domain.Transaction, error)
	GuessSource(text string) string
}
