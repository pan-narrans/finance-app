# Skill: SDQA Expert (Go)

Expert in Software Development Quality Assurance for Go. Ensures code is robust, testable, and adheres to high architectural standards (Hexagonal/DDD).

## Core Mandates

1.  **Determinism Above All**: Tests must be 100% reproducible. No `time.Sleep()`, no unmocked `time.Now()`, and no real filesystem/network calls.
2.  **Testing Pyramid**: Prioritize fast, isolated unit tests. Use Interfaces (Ports) to mock external dependencies.
3.  **Smart Mocking**: Use Go Interfaces to isolate business logic. Never mock simple structs (Entities/DTOs); instantiate them as real fixtures.
4.  **Coverage Criterion**: Aim for "Edge Case" coverage (nil pointers, empty slices, error propagation) over simple percentage targets.

## Style Rules (Go Specific)

-   **Naming Convention**:
    -   File names: `source_file_test.go`.
    -   Test functions: `Test<Subject>_<Method>_Should<ExpectedBehavior>_When<Condition>`.
    -   Example: `TestTransaction_Format_ShouldAlignAmount_WhenCurrencyIsEUR`.
-   **AAA Structure**: Every test (including sub-tests in loops) MUST be visually divided with comments: `// Arrange`, `// Act`, `// Assert`.
-   **Table-Driven Tests**: Highly recommended for logic with multiple inputs, provided the AAA structure is maintained inside the `t.Run` block.
-   **Constants**: Use `const` for dummy codes, expected strings, and fixed test data to improve readability.

## Workflow

1.  **Analyze**: Look for logical flaws, nil-pointer risks, and boundary conditions.
2.  **Strategy**: Define the Happy Path and at least 3 Edge Cases (error states, empty inputs).
3.  **Implement**: Follow the AAA pattern strictly.

## Perfect Unit Test Example (Go)

```go
package domain

import (
	"testing"
	"time"
)

func TestTransaction_Format_ShouldReturnValidLedgerString_WhenValidInputProvided(t *testing.T) {
	// Arrange
	const expected = "2026/01/15 * Día\n    Expenses:Compra                                     60.74 EUR\n    Assets:Checking:OpenBank\n"
	
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	amount := 60.74
	
	trans := Transaction{
		Date:        date,
		Status:      StatusCleared,
		Description: "Día",
		Postings: []Posting{
			{Account: "Expenses:Compra", Amount: &amount, Currency: "EUR"},
			{Account: "Assets:Checking:OpenBank", Amount: nil},
		},
	}

	// Act
	result := trans.Format()

	// Assert
	if result != expected {
		t.Errorf("Format() failed.\nGot:\n%q\nWant:\n%q", result, expected)
	}
}

func TestTransaction_Validate_ShouldReturnError_WhenConditionsAreMet(t *testing.T) {
	// Arrange
	date := time.Now()
	val := 10.0

	tests := []struct {
		name    string
		trans   Transaction
		wantErr bool
	}{
		{
			name: "Should Return Error When Currency Is Missing But Amount Exists",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := tt.trans.Validate()

			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: Validate() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}
```
