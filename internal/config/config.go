package config

import (
	"encoding/json"
	"log"
	"os"
)

// Environment represents application configuration.
type Config struct {
	DefaultCurrency       string `json:"default_currency"`
	DefaultAssetAccount   string `json:"default_asset_account"`
	DefaultIncomeAccount  string `json:"default_income_account"`
	DefaultExpenseAccount string `json:"default_expense_account"`
	LedgerAlignment       int    `json:"ledger_alignment"`
}

/*
LoadConfig loads the config from a JSON file directly into [Config].

If the file is missing, it returns default values and logs a warning.
*/
func LoadConfig(path string) (Config, error) {
	config := Config{
		DefaultCurrency:       "EUR",
		DefaultAssetAccount:   "Assets:Cash",
		DefaultIncomeAccount:  "Income:Unknown",
		DefaultExpenseAccount: "Expenses:Unknown",
		LedgerAlignment:       52,
	}

	fileData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Warning: Config file not found at %s. Using defaults.", path)
			return config, nil
		}
		return config, err
	}

	if err := json.Unmarshal(fileData, &config); err != nil {
		log.Printf("Warning: Config file at %s is invalid. Using defaults. Error: %v", path, err)
		return config, nil // Return defaults even if JSON is malformed
	}

	return config, nil
}
