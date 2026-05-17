package domain

// Settings represents application-wide configuration parameters.
type Settings struct {
	DefaultCurrency       string
	DefaultAssetAccount   string
	DefaultIncomeAccount  string
	DefaultExpenseAccount string
	LedgerAlignment       int
	ImaginBankAccount     string
	OpenBankAccount       string
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
		ImaginBankAccount:     "Assets:Checking:ImaginBank",
		OpenBankAccount:       "Assets:Checking:OpenBank",
	}
}
