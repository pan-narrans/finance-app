package excel

import (
	"encoding/csv"
	"io"
	"os"
	"strings"
	"time"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

// Ensure ImaginBankParser implements ports.BankParser at compile time.
var _ ports.BankParser = (*ImaginBankParser)(nil)

// ImaginBankParser handles ImaginBank-specific CSV file format.
type ImaginBankParser struct {
	*BaseParser
}

// NewImaginBankParser creates a new instance of ImaginBankParser.
func NewImaginBankParser(mappingProvider ports.MappingProvider, settings domain.Settings) *ImaginBankParser {
	return &ImaginBankParser{
		BaseParser: NewBaseParser(mappingProvider, settings),
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
		return nil, domain.NewDomainError("Parser", "Row", "row too short")
	}

	fullDescription := strings.TrimSpace(row[0])
	dateStr := strings.TrimSpace(row[1])
	amountStr := strings.TrimSpace(row[2])
	balanceStr := ""
	if len(row) > 3 {
		balanceStr = strings.TrimSpace(row[3])
	}

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

	cleanDescription := p.mappingProvider.CleanDescription(fullDescription)
	account, found := p.mappingProvider.ResolveAccount(cleanDescription)
	if !found {
		if amount > 0 {
			account = p.settings.DefaultIncomeAccount
		} else {
			account = p.settings.DefaultExpenseAccount
		}
	}

	metadata := domain.Metadata{
		Origin: "Imaginbank",
	}

	if balanceStr != "" {
		metadata.ID = p.HashID(balanceStr)
	}

	absAmount := amount
	if absAmount < 0 {
		absAmount = -absAmount
	}

	// Convention: Postings[0] is Target (Debit), Postings[1] is Source (Credit)
	var postings []domain.Posting
	bankAccount := p.settings.ImaginBankAccount

	if amount >= 0 {
		// Influx: Assets (Target) increase, Income (Source) remains credit balance
		postings = []domain.Posting{
			{Account: bankAccount, Amount: &absAmount, Currency: p.settings.DefaultCurrency},
			{Account: account},
		}
	} else {
		// Outflux: Expense (Target) increases, Assets (Source) decrease
		postings = []domain.Posting{
			{Account: account, Amount: &absAmount, Currency: p.settings.DefaultCurrency},
			{Account: bankAccount},
		}
	}

	tx := domain.Transaction{
		Date:        date,
		Status:      domain.StatusPending,
		Description: cleanDescription,
		Metadata:    metadata,
		Postings:    postings,
	}
	tx.Code = tx.GenerateCode()

	return &tx, nil
}

