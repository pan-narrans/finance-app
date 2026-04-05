// Package domain contains the core business entities and rules for the Finance App.
// It is strictly independent of any external frameworks or input/output adapters.
package domain

import (
	"fmt"
	"strings"
	"time"
)

// TransactionStatus represents the Cleared (*) or Pending (!) status of a transaction,
// as defined by the Ledger CLI format.
type TransactionStatus int

const (
	// StatusNone indicates no status marker in the ledger (default).
	StatusNone TransactionStatus = iota
	// StatusCleared indicates the transaction has been reconciled (*).
	StatusCleared
	// StatusPending indicates the transaction is initiated but not yet cleared (!).
	StatusPending
)

// String implements the fmt.Stringer interface.
// It returns the single-character marker used by Ledger CLI: "*" for cleared, "!" for pending.
func (transactionStatus TransactionStatus) String() string {
	switch transactionStatus {
	case StatusCleared:
		return "*"
	case StatusPending:
		return "!"
	default:
		return ""
	}
}

// Transaction represents a single financial entry in a Ledger file.
// It follows the canonical Ledger CLI format: DATE [STATUS] [(CODE)] DESC.
// A valid transaction must have at least two balanced postings.
type Transaction struct {
	Date        time.Time         // The date of the transaction (YYYY/MM/DD).
	Status      TransactionStatus // The clearing status (*, !, or none).
	Code        string            // Optional unique identifier or reference number in parentheses.
	Description string            // Human-readable description of the transaction, usually used to store the payee.
	Postings    []Posting         // Detailed line items (at least two required).
}

// Posting represents a single line item within a transaction.
// It consists of an account name and an optional amount with currency.
// In Ledger, at most one posting per transaction can omit the amount for automatic balancing.
type Posting struct {
	Account  string   // Full hierarchical account path (e.g., "Expenses:Food").
	Amount   *float64 // Numerical value. If nil, Ledger calculates the balancing amount.
	Currency string   // Mandatory if Amount is provided (e.g., "EUR", "$").
}

// Format returns the transaction formatted as a multi-line string compatible with Ledger CLI.
// It uses a standard alignment (column 52) for amounts to ensure human-readability.
func (transaction *Transaction) Format() string {
	var sb strings.Builder

	// Helper function to simplify formatted writes to the builder.
	// We ignore the error because strings.Builder.Write never returns one.
	write := func(format string, args ...any) {
		_, _ = fmt.Fprintf(&sb, format, args...)
	}

	sb.WriteString(transaction.Date.Format("2006/01/02"))

	if statusStr := transaction.Status.String(); statusStr != "" {
		write(" %s", statusStr)
	}

	if transaction.Code != "" {
		write(" (%s)", transaction.Code)
	}

	write(" %s\n", transaction.Description)

	for _, posting := range transaction.Postings {
		write("    %s", posting.Account)

		if posting.Amount != nil {
			// Calculate padding to align amounts (column 52)
			// TODO: Make this value configurable
			padding := 52 - len(posting.Account)
			if padding < 2 {
				padding = 2
			}
			sb.WriteString(strings.Repeat(" ", padding))

			// Format amount and currency
			// Heuristic: 1-char symbols prefix (e.g., "$"), others suffix (e.g., "EUR")
			if len(posting.Currency) == 1 {
				write("%s%.2f", posting.Currency, *posting.Amount)
			} else {
				write("%.2f %s", *posting.Amount, posting.Currency)
			}
		}

		sb.WriteByte('\n')
	}

	return sb.String()
}

// Validate performs business rule validation on the transaction.
// It ensures required fields are present and that our internal rules
// (like mandatory currency for numeric amounts) are respected.
// It returns a structured DomainError if any validation failures occur.
func (transaction *Transaction) Validate() error {
	validationErrors := &ValidationErrors{}
	entity := "Transaction"

	if transaction.Date.IsZero() {
		validationErrors.Add(entity, "Date", "transaction date is required")
	}
	if transaction.Description == "" {
		validationErrors.Add(entity, "Description", "transaction description is required")
	}
	if len(transaction.Postings) < 2 {
		validationErrors.Add(entity, "Postings", "transaction must have at least two postings to balance")
	}

	nilCount := 0
	for i, posting := range transaction.Postings {
		// Business Rule: If an amount is present, currency is mandatory.
		if posting.Amount != nil && posting.Currency == "" {
			field := fmt.Sprintf("Postings[%d].Currency", i)
			validationErrors.Add(entity, field, fmt.Sprintf("currency is mandatory for posting to account %q", posting.Account))
		}
		if posting.Amount == nil {
			nilCount++
		}
	}

	if nilCount > 1 {
		validationErrors.Add(entity, "Postings", "at most one posting can have an implicit amount")
	}

	if len(validationErrors.Errors) > 0 {
		return validationErrors
	}

	return nil
}
