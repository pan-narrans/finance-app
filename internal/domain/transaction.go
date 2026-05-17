// Package domain contains the core business entities and rules for the Finance App.
// It is strictly independent of any external frameworks or input/output adapters.
package domain

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

/*
TransactionStatus represents the clearing status of a transaction.
Values map to Ledger CLI's standard indicators (* for cleared, ! for pending).
*/
type TransactionStatus int

const (
	StatusUncleared TransactionStatus = iota
	StatusCleared
	StatusPending
)

/*
Posting represents a single line item within a transaction.
It includes the account, amount, and currency.
*/
type Posting struct {
	Account  string
	Amount   *float64 // Pointer to handle implicit (nil) amounts
	Currency string
}

/*
Metadata stores non-accounting information about a transaction.
Used for tracking origin, external IDs, and attribution.
*/
type Metadata struct {
	Origin  string
	ID      string
	PayedBy string
	Extras  map[string]string
}

/*
Transaction represents a single financial entry in the system.
It is the primary domain entity, aggregating all relevant accounting data.
*/
type Transaction struct {
	Date        time.Time
	Status      TransactionStatus
	Description string
	Code        string
	Metadata    Metadata
	Postings    []Posting
}

/*
GenerateCode creates a unique 16-character identifier for the transaction.
The code is derived from the transaction's stable properties (Date, Description, Postings).
*/
func (t *Transaction) GenerateCode() string {
	hasher := sha256.New()
	hash := func(s string) { hasher.Write([]byte(s)) }

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
