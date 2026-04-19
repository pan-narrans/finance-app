package excel

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/a-perez/finance-app/internal/domain"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// OpenBankParser handles OpenBank-specific HTML-based XLS format.
type OpenBankParser struct {
	accountMappings map[string]string
}

// NewOpenBankParser creates a new instance of OpenBankParser with optional mappings.
func NewOpenBankParser(mappingsPath string) *OpenBankParser {
	mappings := make(map[string]string)
	if mappingsPath != "" {
		data, err := os.ReadFile(mappingsPath)
		if err == nil {
			if err := json.Unmarshal(data, &mappings); err != nil {
				log.Printf("Error unmarshaling mappings: %v", err)
			}
		}
	}
	return &OpenBankParser{accountMappings: mappings}
}

// Parse reads the HTML table and converts rows to domain transactions.
func (p *OpenBankParser) Parse(filePath string) ([]domain.Transaction, error) {
	rows, err := p.loadRows(filePath)
	if err != nil {
		return nil, err
	}

	var transactions []domain.Transaction
	for _, row := range rows {
		transaction, err := p.rowToTransaction(row)
		if err != nil {
			continue // Skip non-data or invalid rows
		}
		transactions = append(transactions, *transaction)
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

	// 3: Fecha Valor, 5: Concepto, 7: Importe
	date, err := time.Parse("02/01/2006", strings.TrimSpace(row[3]))
	if err != nil {
		return nil, err
	}

	amount, err := parseSpanishAmount(strings.TrimSpace(row[7]))
	if err != nil {
		return nil, err
	}

	description := strings.TrimSpace(row[5])
	targetAccount := p.resolveAccount(description, amount)

	return &domain.Transaction{
		Date:        date,
		Description: description,
		Postings: []domain.Posting{
			{Account: "Assets:Checking:OpenBank", Amount: &amount, Currency: "EUR"},
			{Account: targetAccount},
		},
	}, nil
}

func (p *OpenBankParser) resolveAccount(description string, amount float64) string {
	for keyword, account := range p.accountMappings {
		if strings.Contains(strings.ToUpper(description), strings.ToUpper(keyword)) {
			return account
		}
	}

	if amount > 0 {
		return "Income:Unknown"
	}

	return "Expenses:Unknown"
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

func parseSpanishAmount(s string) (float64, error) {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}
