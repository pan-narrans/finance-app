// Package domain contains the core business entities and rules for the Finance App.
// It is strictly independent of any external frameworks or input/output adapters.
package domain

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
Metadata represents key-value pairs stored as comments in a Ledger transaction.
Supports specific fields for identification and origin tracking, plus arbitrary extras.
*/
type Metadata struct {
	ID      string
	Origin  string
	PayedBy string
	Extras  map[string]string
}

/*
Transaction represents a single financial entry in a Ledger file.

Fields:
  - Date: The date of the transaction (YYYY/MM/DD).
  - Status: The clearing status (* for cleared, ! for pending, or none).
  - Code: Optional unique identifier or reference number in parentheses.
  - Description: Human-readable description, usually storing the payee.
  - Metadata: Specific attributes stored as comments (e.g., ID, Origin, PayedBy).
  - Postings: Detailed line items (at least two required).
*/
type Transaction struct {
	Date        time.Time
	Status      TransactionStatus
	Code        string
	Description string
	Metadata    Metadata
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
func (t *Transaction) GenerateCode() string {
	hasher := sha256.New()
	hash := func(data string) {
		hasher.Write([]byte(data))
		hasher.Write([]byte("|"))
	}

	hash(t.Date.Format("2006-01-02"))
	hash(t.Description)

	if t.Metadata.ID != "" {
		hash(t.Metadata.ID)
	}

	for _, posting := range t.Postings {
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
  - Amounts are right-aligned to a standard column.
  - 1-character currencies (e.g. $) prefix the amount; others suffix it (e.g. EUR).
*/
func (t *Transaction) Format(alignment int) string {
	var sb strings.Builder

	t.writeLine(&sb, "%s", t.Date.Format("2006/01/02"))
	t.writeLine(&sb, " %s", t.Status.String())
	t.writeLine(&sb, " (%s)", t.Code)
	t.writeLine(&sb, " %s", t.Description)
	sb.WriteByte('\n')

	t.addMetadata(&sb)
	t.addPostings(&sb, alignment)

	return sb.String()
}

/*
writeLine prints to the buffer unless any provided string argument is empty.

This allows handling optional segments (status, code, metadata) concisely by
skipping the entire line if a required component is missing, ensuring that
related parts stay together or die together.
*/
func (t *Transaction) writeLine(builder *strings.Builder, format string, args ...any) {
	for _, arg := range args {
		if s, ok := arg.(string); ok && s == "" {
			return
		}
	}
	_, _ = fmt.Fprintf(builder, format, args...)
}

/*
addMetadata appends the transaction's metadata as comments to the builder.

It processes ID, Origin, and PayedBy fields first, followed by any
alphabetically sorted extra fields. Empty fields are omitted.
*/
func (t *Transaction) addMetadata(builder *strings.Builder) {
	const metadataFormat = "    ; %s: %s\n"

	t.writeLine(builder, metadataFormat, "ID", t.Metadata.ID)
	t.writeLine(builder, metadataFormat, "Origin", t.Metadata.Origin)
	t.writeLine(builder, metadataFormat, "PayedBy", t.Metadata.PayedBy)

	// Write arbitrary metadata in alphabetical order for stability
	if len(t.Metadata.Extras) > 0 {
		keys := make([]string, 0, len(t.Metadata.Extras))
		for k := range t.Metadata.Extras {
			keys = append(keys, k)
		}

		slices.Sort(keys)
		for _, k := range keys {
			t.writeLine(builder, metadataFormat, k, t.Metadata.Extras[k])
		}
	}
}

/*
addPostings appends all transaction postings to the builder.

It ensures account names are correctly indented and amounts are right-aligned
according to the provided alignment column. Numeric formatting follows
Ledger standards (prefix for 1-char symbols, suffix for others).
*/
func (t *Transaction) addPostings(sb *strings.Builder, alignment int) {
	for _, posting := range t.Postings {
		t.writeLine(sb, "    %s", posting.Account)

		if posting.Amount != nil {
			padding := alignment - len(posting.Account)
			padding = max(padding, 2)
			sb.WriteString(strings.Repeat(" ", padding))

			if len(posting.Currency) == 1 {
				t.writeLine(sb, "%s%.2f", posting.Currency, *posting.Amount)
			} else {
				t.writeLine(sb, "%.2f %s", *posting.Amount, posting.Currency)
			}
		}

		sb.WriteByte('\n')
	}
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
func (t *Transaction) Validate() error {
	domainError := &DomainError{}
	entity := "Transaction"

	if t.Date.IsZero() {
		domainError.Add(entity, "Date", "transaction date is required")
	}
	if t.Description == "" {
		domainError.Add(entity, "Description", "transaction description is required")
	}
	if len(t.Postings) < 2 {
		domainError.Add(entity, "Postings", "transaction must have at least two postings to balance")
	}

	nilCount := 0
	for i, posting := range t.Postings {
		if posting.Account == "" {
			field := fmt.Sprintf("Postings[%d].Account", i)
			domainError.Add(entity, field, "account name is required")
		}

		if posting.Amount != nil && posting.Currency == "" {
			field := fmt.Sprintf("Postings[%d].Currency", i)
			domainError.Add(entity, field, fmt.Sprintf("currency is mandatory for posting to account %q", posting.Account))
		}
		if posting.Amount == nil {
			nilCount++
		}
	}

	if nilCount > 1 {
		domainError.Add(entity, "Postings", "at most one posting can have an implicit amount")
	}

	if len(domainError.Errors) > 0 {
		return domainError
	}

	return nil
}

/*
FormatAccountPath ensures all segments of an account path are Title Cased.
(e.g., "expenses:food:dining" -> "Expenses:Food:Dining")
*/
func FormatAccountPath(path string) string {
	segments := strings.Split(path, ":")
	caser := cases.Title(language.Und)
	for i, seg := range segments {
		segments[i] = caser.String(strings.ToLower(strings.TrimSpace(seg)))
	}
	return strings.Join(segments, ":")
}
