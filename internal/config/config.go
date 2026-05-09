package config

import (
	"encoding/json"
	"os"
)

// Environment represents application configuration.
type Config struct {
	DefaultCurrency       string `json:"default_currency"`
	DefaultBotAccount     string `json:"default_bot_account"`
	DefaultIncomeAccount  string `json:"default_income_account"`
	DefaultExpenseAccount string `json:"default_expense_account"`
}

// LoadConfig loads the config from a JSON file directly into [Config].
func LoadConfig(path string) (Config, error) {
	config := Config{
		DefaultCurrency:       "",
		DefaultBotAccount:     "",
		DefaultIncomeAccount:  "",
		DefaultExpenseAccount: "",
	}

	fileData, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}

	if err := json.Unmarshal(fileData, &config); err != nil {
		return config, err
	}

	return config, nil
}
