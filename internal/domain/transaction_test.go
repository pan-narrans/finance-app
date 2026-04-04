package domain

import (
	"testing"
	"time"
)

// TODO: review testing in godot

func TestTransaction_Format_ShouldReturnValidLedgerString_WhenValidInputProvided(t *testing.T) {
	// Arrange
	const expected = `2026/01/15 * Día
    Expenses:Compra                                     60.74 EUR
    Assets:Checking:OpenBank
`
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	amount := 60.74
	transaction := Transaction{
		Date:        date,
		Status:      StatusCleared,
		Description: "Día",
		Postings: []Posting{
			{
				Account:  "Expenses:Compra",
				Amount:   &amount,
				Currency: "EUR",
			},
			{
				Account: "Assets:Checking:OpenBank",
				Amount:  nil, // Implicit balancing
			},
		},
	}

	// Act
	got := transaction.Format()

	// Assert
	if got != expected {
		t.Errorf("Format() = \n%q, want \n%q", got, expected)
	}
}

func TestTransaction_Validate_ShouldReturnCorrectResults_ForVariousInputs(t *testing.T) {
	// Arrange
	date := time.Now()
	val := 10.0

	tests := []struct {
		name    string
		trans   Transaction
		wantErr bool
	}{
		{
			name: "Should Return No Error When Input Is Valid With Implicit Balance And Currency",
			trans: Transaction{
				Date:        date,
				Description: "Valid",
				Postings: []Posting{
					{Account: "A", Amount: &val, Currency: "USD"},
					{Account: "B", Amount: nil},
				},
			},
			wantErr: false,
		},
		{
			name: "Should Return Error When Amount Is Provided Without Currency",
			trans: Transaction{
				Date:        date,
				Description: "Missing Currency",
				Postings: []Posting{
					{Account: "A", Amount: &val, Currency: ""},
					{Account: "B", Amount: nil},
				},
			},
			wantErr: true,
		},
		{
			name: "Should Return Error When Multiple Implicit Balances Are Provided",
			trans: Transaction{
				Date:        date,
				Description: "Too many nils",
				Postings: []Posting{
					{Account: "A", Amount: nil},
					{Account: "B", Amount: nil},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Act
				err := tt.trans.Validate()

				// Assert
				if (err != nil) != tt.wantErr {
					t.Errorf("%s: Validate() error = %v, wantErr %v", tt.name, err, tt.wantErr)
				}
			},
		)
	}
}
