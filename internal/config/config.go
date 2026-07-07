package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/a-perez/finance-app/internal/domain"
)
// fileConfig is the JSON representation of the configuration file.
type fileConfig struct {
	DefaultCurrency       string   `json:"default_currency"`
	DefaultAssetAccount   string   `json:"default_asset_account"`
	DefaultIncomeAccount  string   `json:"default_income_account"`
	DefaultExpenseAccount string   `json:"default_expense_account"`
	LedgerAlignment       int      `json:"ledger_alignment"`
	ImaginAssetAccount    string   `json:"imagin_asset_account"`
	ImaginBankAccount     string   `json:"imaginbank_account"` // Legacy support
	OpenBankAssetAccount  string   `json:"openbank_asset_account"`
	OpenBankAccount       string   `json:"openbank_account"` // Legacy support
	RootAccounts          []string `json:"root_accounts"`
	TelegramUserIDs       []int64  `json:"telegram_user_ids"`
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
		return settings, fmt.Errorf("invalid config JSON at %s: %w", path, err)
	}

	// Override defaults with file values if provided
	applyIfNonZero(&settings.DefaultCurrency, fc.DefaultCurrency)
	applyIfNonZero(&settings.DefaultAssetAccount, fc.DefaultAssetAccount)
	applyIfNonZero(&settings.DefaultIncomeAccount, fc.DefaultIncomeAccount)
	applyIfNonZero(&settings.DefaultExpenseAccount, fc.DefaultExpenseAccount)
	applyIfNonZero(&settings.LedgerAlignment, fc.LedgerAlignment)

	// Bank accounts: Support both new and legacy keys
	if fc.ImaginAssetAccount != "" {
		settings.ImaginAssetAccount = fc.ImaginAssetAccount
	} else if fc.ImaginBankAccount != "" {
		settings.ImaginAssetAccount = fc.ImaginBankAccount
	}

	if fc.OpenBankAssetAccount != "" {
		settings.OpenBankAssetAccount = fc.OpenBankAssetAccount
	} else if fc.OpenBankAccount != "" {
		settings.OpenBankAssetAccount = fc.OpenBankAccount
	}

	if len(fc.RootAccounts) > 0 {
		settings.RootAccounts = fc.RootAccounts
	}

	if len(fc.TelegramUserIDs) > 0 {
		settings.TelegramUserIDs = fc.TelegramUserIDs
	}

	// Always fallback or override with environment TELEGRAM_USER_IDS if set
	if envIDsRaw := os.Getenv("TELEGRAM_USER_IDS"); envIDsRaw != "" {
		var envIDs []int64
		for _, part := range strings.Split(envIDsRaw, ",") {
			part = strings.TrimSpace(part)
			if val, err := strconv.ParseInt(part, 10, 64); err == nil {
				envIDs = append(envIDs, val)
			}
		}
		if len(envIDs) > 0 {
			settings.TelegramUserIDs = envIDs
		}
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
