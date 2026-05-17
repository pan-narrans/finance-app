package config

import (
	"encoding/json"
	"log"
	"os"

	"github.com/a-perez/finance-app/internal/domain"
)

// fileConfig is the JSON representation of the configuration file.
type fileConfig struct {
	DefaultCurrency       string   `json:"default_currency"`
	DefaultAssetAccount   string   `json:"default_asset_account"`
	DefaultIncomeAccount  string   `json:"default_income_account"`
	DefaultExpenseAccount string   `json:"default_expense_account"`
	LedgerAlignment       int      `json:"ledger_alignment"`
	ImaginBankAccount     string   `json:"imaginbank_account"`
	OpenBankAccount       string   `json:"openbank_account"`
	RootAccounts          []string `json:"root_accounts"`
}

/*
LoadConfig loads the config from a JSON file and returns [domain.Settings].

If the file is missing or invalid, it returns default values and logs a warning.
*/
func LoadConfig(path string) (domain.Settings, error) {
	settings := domain.DefaultSettings()

	fileData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Warning: Config file not found at %s. Using defaults.", path)
			return settings, nil
		}
		return settings, err
	}

	var fc fileConfig
	if err := json.Unmarshal(fileData, &fc); err != nil {
		log.Printf("Warning: Config file at %s is invalid. Using defaults. Error: %v", path, err)
		return settings, nil
	}

	// Override defaults with file values if provided
	applyIfNonZero(&settings.DefaultCurrency, fc.DefaultCurrency)
	applyIfNonZero(&settings.DefaultAssetAccount, fc.DefaultAssetAccount)
	applyIfNonZero(&settings.DefaultIncomeAccount, fc.DefaultIncomeAccount)
	applyIfNonZero(&settings.DefaultExpenseAccount, fc.DefaultExpenseAccount)
	applyIfNonZero(&settings.LedgerAlignment, fc.LedgerAlignment)
	applyIfNonZero(&settings.ImaginBankAccount, fc.ImaginBankAccount)
	applyIfNonZero(&settings.OpenBankAccount, fc.OpenBankAccount)

	if len(fc.RootAccounts) > 0 {
		settings.RootAccounts = fc.RootAccounts
	}

	return settings, nil
}

/*
applyIfNonZero updates the target pointer with the value only if the value is not the zero-value for its type.
*/
func applyIfNonZero[T comparable](target *T, value T) {
	var zero T
	if value != zero {
		*target = value
	}
}
