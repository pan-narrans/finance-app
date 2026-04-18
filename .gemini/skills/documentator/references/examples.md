# Documentation Examples

## Structs

### Bad (Line Comments)
```go
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
```

### Good (Block Comments)
```go
/*
Transaction represents a single financial entry in a Ledger file.

Fields:
  - Date: The date of the transaction (YYYY/MM/DD).
  - Status: The clearing status (* for cleared, ! for pending, or none).
  - Code: Optional unique identifier or reference number in parentheses.
  - Description: Human-readable description, usually storing the payee.
  - Postings: Detailed line items (at least two required).
*/
type Transaction struct {
	Date        time.Time
	Status      TransactionStatus
	Code        string
	Description string
	Postings    []Posting
}
```
