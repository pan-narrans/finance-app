package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransaction_Format_ShouldReturnValidLedgerString_WhenValidInputProvided(t *testing.T) {
	// Arrange
	const expected = "2026/01/15 * Día\n    Expenses:Shopping                                   60.74 EUR\n    Assets:Checking:OpenBank\n"
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	transaction := Transaction{
		Date:        date,
		Status:      StatusCleared,
		Description: "Día",
		Postings: []Posting{
			{Account: "Expenses:Shopping", Amount: new(60.74), Currency: "EUR"},
			{Account: "Assets:Checking:OpenBank", Amount: nil},
		},
	}

	// Act
	got := transaction.Format()

	// Assert
	assert.Equal(t, expected, got)
}

func TestTransaction_Validate_ShouldReturnNoErrors_WhenInputIsValid(t *testing.T) {
	// Arrange
	transaction := Transaction{
		Date:        time.Now(),
		Description: "Valid",
		Postings: []Posting{
			{Account: "A", Amount: new(10.0), Currency: "USD"},
			{Account: "B", Amount: nil},
		},
	}

	// Act
	err := transaction.Validate()

	// Assert
	assert.NoError(t, err)
}

func TestTransaction_Validate_ShouldReturnStructuredErrors_WhenInputIsInvalid(t *testing.T) {
	// Arrange
	date := time.Now()
	val := 10.0

	tests := []struct {
		name           string
		transaction    Transaction
		expectedErrors []ValidationError
	}{
		{
			name: "Should Detect Missing Date And Description",
			transaction: Transaction{
				Postings: []Posting{
					{Account: "A", Amount: &val, Currency: "USD"},
					{Account: "B", Amount: nil},
				},
			},
			expectedErrors: []ValidationError{
				{Entity: "Transaction", Field: "Date", Message: "transaction date is required"},
				{Entity: "Transaction", Field: "Description", Message: "transaction description is required"},
			},
		},
		{
			name: "Should Detect Missing Currency For Numerical Amount",
			transaction: Transaction{
				Date:        date,
				Description: "Missing Currency",
				Postings: []Posting{
					{Account: "A", Amount: &val, Currency: ""},
					{Account: "B", Amount: nil},
				},
			},
			expectedErrors: []ValidationError{
				{Entity: "Transaction", Field: "Postings[0].Currency", Message: "currency is mandatory for posting to account \"A\""},
			},
		},
		{
			name: "Should Detect Multiple Implicit Balances",
			transaction: Transaction{
				Date:        date,
				Description: "Too many nils",
				Postings: []Posting{
					{Account: "A", Amount: nil},
					{Account: "B", Amount: nil},
				},
			},
			expectedErrors: []ValidationError{
				{Entity: "Transaction", Field: "Postings", Message: "at most one posting can have an implicit amount"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Act
				err := tt.transaction.Validate()

				// Assert
				var validationErrors *ValidationErrors
				ok := errors.As(err, &validationErrors)
				require.Error(t, err)
				require.True(t, ok, "Error should be of type *ValidationErrors")
				assert.Equal(t, tt.expectedErrors, validationErrors.Errors)
			},
		)
	}
}

func TestTransaction_Samples_ShouldBeHandledCorrectly(t *testing.T) {
	// Arrange
	date := time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC)
	val100 := 100.0
	val50 := 50.0

	tests := []struct {
		name        string
		transaction Transaction
		isValid     bool
	}{
		{
			name: "Valid: Simple balanced transaction",
			transaction: Transaction{
				Date: date, Description: "Grocery",
				Postings: []Posting{
					{Account: "Expenses:Food", Amount: &val50, Currency: "USD"},
					{Account: "Assets:Checking", Amount: nil},
				},
			},
			isValid: true,
		},
		{
			name: "Valid: Multi-posting split",
			transaction: Transaction{
				Date: date, Description: "Split Bill",
				Postings: []Posting{
					{Account: "Expenses:Rent", Amount: &val100, Currency: "EUR"},
					{Account: "Expenses:Internet", Amount: &val50, Currency: "EUR"},
					{Account: "Assets:Checking", Amount: nil},
				},
			},
			isValid: true,
		},
		{
			name: "Invalid: No postings",
			transaction: Transaction{
				Date: date, Description: "Empty",
				Postings: []Posting{},
			},
			isValid: false,
		},
		{
			name: "Invalid: All implicit amounts",
			transaction: Transaction{
				Date: date, Description: "Guess work",
				Postings: []Posting{
					{Account: "Assets:Checking", Amount: nil},
					{Account: "Expenses:Unknown", Amount: nil},
				},
			},
			isValid: false,
		},
		{
			name: "Invalid: Missing account name",
			transaction: Transaction{
				Date: date, Description: "No account",
				Postings: []Posting{
					{Account: "", Amount: &val50, Currency: "USD"},
					{Account: "Assets:Cash", Amount: nil},
				},
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := tt.transaction.Validate()

			// Assert
			if tt.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
