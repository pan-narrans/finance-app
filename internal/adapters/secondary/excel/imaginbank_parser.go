package excel

import (
	"encoding/csv"
	"io"
	"os"
	"strings"
	"time"

	"github.com/a-perez/finance-app/internal/domain"
)

// ImaginBankParser handles ImaginBank-specific CSV file format.
type ImaginBankParser struct {
	*BaseParser
}

// NewImaginBankParser creates a new instance of ImaginBankParser.
func NewImaginBankParser(mappingsPath string) *ImaginBankParser {
	return &ImaginBankParser{
		BaseParser: NewBaseParser(mappingsPath),
	}
}

// Parse implements ports.BankParser.
func (p *ImaginBankParser) Parse(filePath string) ([]domain.Transaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.LazyQuotes = true

	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	var transactions []domain.Transaction
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			continue
		}

		if transaction, err := p.rowToTransaction(row); err == nil {
			transactions = append(transactions, *transaction)
		}
	}

	return transactions, nil
}

func (p *ImaginBankParser) rowToTransaction(row []string) (*domain.Transaction, error) {
	if len(row) < 3 {
		return nil, domain.NewValidationErrors("Parser", "Row", "row too short")
	}

	fullDescription := strings.TrimSpace(row[0])
	dateStr := strings.TrimSpace(row[1])
	amountStr := strings.TrimSpace(row[2])
	balanceStr := strings.TrimSpace(row[3])

	date, err := time.Parse("02/01/2006", dateStr)
	if err != nil {
		return nil, err
	}

	// Clean amount: remove "EUR" suffix and parse Spanish format
	amountStr = strings.TrimSuffix(amountStr, "EUR")
	amount, err := ParseSpanishAmount(amountStr)
	if err != nil {
		return nil, err
	}

	cleanDescription := p.CleanDescription(fullDescription)
	targetAccount := p.ResolveAccount(cleanDescription, amount)

	metadata := make(map[string]string)
	metadata["Origin"] = "Imaginbank"

	if balanceStr != "" {
		metadata["ID"] = p.HashID(balanceStr)
	}

	return &domain.Transaction{
		Date:        date,
		Status:      domain.StatusPending,
		Description: cleanDescription,
		Metadata:    metadata,
		Postings: []domain.Posting{
			{Account: "Assets:Checking:ImaginBank", Amount: &amount, Currency: "EUR"},
			{Account: targetAccount},
		},
	}, nil
}
