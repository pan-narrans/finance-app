// Package domain contains the core business entities and rules for the Finance App.
// It is strictly independent of any external frameworks or input/output adapters.
package domain

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"time"
)

/*
TransactionStatus represents the clearing status of a transaction.

Values:
  - StatusNone: No status marker (default).
  - StatusCleared: Reconciled [Transaction] (*).
  - StatusPending: [Transaction] initiated but not cleared (!).
*/
type TransactionStatus int

const (
	StatusNone    TransactionStatus = iota // No status marker in the ledger (default).
	StatusCleared                          // [Transaction] has been reconciled (*).
	StatusPending                          // [Transaction] is initiated but not yet cleared (!).
)

/*
String implements the fmt.Stringer interface.
Returns the single-character marker used by Ledger CLI.
*/
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

/*
Transaction represents a single financial entry in a Ledger file.

Fields:
  - Date: The date of the transaction (YYYY/MM/DD).
  - Status: The clearing status (* for cleared, ! for pending, or none).
  - Code: Optional unique identifier or reference number in parentheses.
  - Description: Human-readable description, usually storing the payee.
  - Metadata: Key-value pairs stored as comments (e.g., "PayedBy": "Alex").
  - Postings: Detailed line items (at least two required).
*/
type Transaction struct {
	Date        time.Time
	Status      TransactionStatus
	Code        string
	Description string
	Metadata    map[string]string
	Postings    []Posting
}

/*
Posting represents a single line item within a transaction.

Fields:
  - Account: Full hierarchical account path (e.g., "Expenses:Food").
  - Amount: Numerical value. If nil, Ledger calculates the balancing amount.
  - Currency: Mandatory if Amount is provided (e.g., "EUR", "$").
*/
type Posting struct {
	Account  string
	Amount   *float64
	Currency string
}

/*
GenerateCode creates a deterministic unique identifier for the transaction
based on its date, description, and postings.

It uses SHA-256 and pipe delimiters to prevent field-boundary collisions.
Status is excluded as it is mutable. Specific stable metadata "ID" is
included to differentiate otherwise identical transactions.

Since the account is dependent on the description, it is excluded from code generation.
This ensures that changes to the account mappings file do not alter existing transaction codes.
*/
func (transaction *Transaction) GenerateCode() string {
	hasher := sha256.New()
	hash := func(data string) {
		hasher.Write([]byte(data))
		hasher.Write([]byte("|"))
	}

	hash(transaction.Date.Format("2006-01-02"))
	hash(transaction.Description)

	if val, ok := transaction.Metadata["ID"]; ok {
		hash(val)
	}

	for _, posting := range transaction.Postings {
		if posting.Amount != nil {
			hash(fmt.Sprintf("%.2f", *posting.Amount))
			hash(posting.Currency)
		}
	}

	fullHash := fmt.Sprintf("%x", hasher.Sum(nil))
	return fullHash[:16]
}

/*
Format returns a multi-line string compatible with Ledger CLI.

It applies the following formatting rules:
  - Dates use YYYY/MM/DD format.
  - Payees and descriptions are appended to the header.
  - Metadata is stored as indented comments below the header.
  - Account names are indented by four spaces.
  - Amounts are right-aligned to a standard column (default 52).
  - 1-character currencies (e.g. $) prefix the amount; others suffix it (e.g. EUR).
*/
//TODO Make alignment value (now 52) configurable.
func (transaction *Transaction) Format() string {
	var sb strings.Builder

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

	// Write metadata as comments in alphabetical order
	keys := make([]string, 0, len(transaction.Metadata))
	for k := range transaction.Metadata {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, k := range keys {
		write("    ; %s: %s\n", k, transaction.Metadata[k])
	}

	for _, posting := range transaction.Postings {
		write("    %s", posting.Account)

		if posting.Amount != nil {
			padding := 52 - len(posting.Account)
			if padding < 2 {
				padding = 2
			}
			sb.WriteString(strings.Repeat(" ", padding))

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

/*
Validate enforces domain business rules for a transaction.

Validation checks:
  - Required fields (Date, Description, Account).
  - Minimum of two postings.
  - Mandatory currency for numeric amounts.
  - Maximum of one implicit (nil) amount.

It returns a structured DomainError if any validation failures occur.
*/
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
		if posting.Account == "" {
			field := fmt.Sprintf("Postings[%d].Account", i)
			validationErrors.Add(entity, field, "account name is required")
		}

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
