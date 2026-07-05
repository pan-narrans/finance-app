package domain

// Settings represents application-wide configuration parameters.
type Settings struct {
	DefaultCurrency       string
	DefaultAssetAccount   string
	DefaultIncomeAccount  string
	DefaultExpenseAccount string
	LedgerAlignment       int
	ImaginAssetAccount    string
	OpenBankAssetAccount  string
	RootAccounts          []string
	TelegramUserIDs       []int64
}

/*
DefaultSettings returns the standard fallback configuration for the application.
*/
func DefaultSettings() Settings {
	return Settings{
		DefaultCurrency:       "EUR",
		DefaultAssetAccount:   "Assets:Cash",
		DefaultIncomeAccount:  "Income:Unknown",
		DefaultExpenseAccount: "Expenses:Unknown",
		LedgerAlignment:       52,
		ImaginAssetAccount:    "Assets:Checking:ImaginBank",
		OpenBankAssetAccount:  "Assets:Checking:OpenBank",
		RootAccounts:          []string{"Expenses", "Assets", "Liabilities", "Income"},
	}
}
