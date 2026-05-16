package excel

import (
	"os"
	"strings"
	"time"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Ensure OpenBankParser implements ports.BankParser at compile time.
var _ ports.BankParser = (*OpenBankParser)(nil)

// OpenBankParser handles OpenBank-specific HTML-based XLS format.
type OpenBankParser struct {
	*BaseParser
}

// NewOpenBankParser creates a new instance of OpenBankParser.
func NewOpenBankParser(mappingProvider ports.MappingProvider, settings config.Config) *OpenBankParser {
	return &OpenBankParser{
		BaseParser: NewBaseParser(mappingProvider, settings),
	}
}

// Parse reads the HTML table and converts rows to domain transactions.
func (p *OpenBankParser) Parse(filePath string) ([]domain.Transaction, error) {
	rows, err := p.loadRows(filePath)
	if err != nil {
		return nil, err
	}

	var transactions []domain.Transaction
	for _, row := range rows {
		if transaction, err := p.rowToTransaction(row); err == nil {
			transactions = append(transactions, *transaction)
		}
	}

	return transactions, nil
}

func (p *OpenBankParser) loadRows(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	utf8Reader := transform.NewReader(file, charmap.ISO8859_1.NewDecoder())
	htmlTree, err := html.Parse(utf8Reader)
	if err != nil {
		return nil, err
	}

	return p.extractRows(htmlTree), nil
}

func (p *OpenBankParser) extractRows(node *html.Node) [][]string {
	var rows [][]string
	var traverse func(*html.Node)

	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			row := p.extractCells(node)
			if len(row) > 0 {
				rows = append(rows, row)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	traverse(node)

	return rows
}

func (p *OpenBankParser) extractCells(node *html.Node) []string {
	var row []string

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "td" {
			row = append(row, getInnerText(child))
		}
	}

	return row
}

func (p *OpenBankParser) rowToTransaction(row []string) (*domain.Transaction, error) {
	if len(row) < 10 {
		return nil, domain.NewValidationErrors("Parser", "Row", "row too short")
	}

	date, err := time.Parse("02/01/2006", strings.TrimSpace(row[3]))
	if err != nil {
		return nil, err
	}

	amount, err := ParseSpanishAmount(strings.TrimSpace(row[7]))
	if err != nil {
		return nil, err
	}

	fullDescription := strings.TrimSpace(row[5])
	description := strings.TrimSpace(strings.Split(fullDescription, ",")[0])
	cleanDescription := p.mappingProvider.CleanDescription(description)

	metadata := domain.Metadata{
		Origin: "Openbank",
	}

	if balance := strings.TrimSpace(row[9]); balance != "" {
		metadata.ID = p.HashID(balance)
	}

	if payedBy := p.mappingProvider.ResolvePayer(fullDescription); payedBy != "" {
		metadata.PayedBy = payedBy
	}

	targetAccount := p.resolveAccount(cleanDescription, amount)

	return &domain.Transaction{
		Date:        date,
		Status:      domain.StatusPending,
		Description: cleanDescription,
		Metadata:    metadata,
		Postings: []domain.Posting{
			{Account: p.settings.OpenBankAccount, Amount: &amount, Currency: p.settings.DefaultCurrency},
			{Account: targetAccount},
		},
	}, nil
}

// TODO why is this here? before we had extracted this to mappingService.ResolveAccount
func (p *OpenBankParser) resolveAccount(description string, amount float64) string {
	if account, found := p.mappingProvider.ResolveAccount(description); found {
		return account
	}

	if amount > 0 {
		return p.settings.DefaultIncomeAccount
	}

	return p.settings.DefaultExpenseAccount
}

func getInnerText(node *html.Node) string {
	var sb strings.Builder
	var collectText func(*html.Node)

	collectText = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			collectText(c)
		}
	}
	collectText(node)

	return sb.String()
}
