package excel

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
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
	cardMappings    map[string]string
	sortedKeywords  []string
	prefixRegexes   []*regexp.Regexp
}

type mappingsData struct {
	Accounts map[string]string `json:"accounts"`
	Cards    map[string]string `json:"cards"`
	Prefixes []string          `json:"prefixes"`
}

// NewOpenBankParser creates a new instance of OpenBankParser with optional mappings.
func NewOpenBankParser(mappingsPath string) *OpenBankParser {
	// TODO this should be generic for other parsers, use composition
	data := mappingsData{
		Accounts: make(map[string]string),
		Cards:    make(map[string]string),
		Prefixes: make([]string, 0),
	}

	if mappingsPath != "" {
		fileData, err := os.ReadFile(mappingsPath)
		if err == nil {
			if err := json.Unmarshal(fileData, &data); err != nil {
				log.Printf("Error unmarshaling mappings: %v", err)
			}
		}
	}

	// Pre-sort keywords by length descending for deterministic matching (longest first wins)
	keywords := make([]string, 0, len(data.Accounts))
	for k := range data.Accounts {
		keywords = append(keywords, k)
	}
	sort.Slice(
		keywords, func(i, j int) bool {
			if len(keywords[i]) == len(keywords[j]) {
				return keywords[i] < keywords[j]
			}
			return len(keywords[i]) > len(keywords[j])
		},
	)

	// Compile prefixes into case-insensitive regexes anchored to start
	prefixRegexes := make([]*regexp.Regexp, 0, len(data.Prefixes))
	for _, prefix := range data.Prefixes {
		pattern := "(?i)^" + regexp.QuoteMeta(prefix) + `\s*`
		if regex, err := regexp.Compile(pattern); err == nil {
			prefixRegexes = append(prefixRegexes, regex)
		}
	}

	return &OpenBankParser{
		accountMappings: data.Accounts,
		cardMappings:    data.Cards,
		sortedKeywords:  keywords,
		prefixRegexes:   prefixRegexes,
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

	amount, err := parseSpanishAmount(strings.TrimSpace(row[7]))
	if err != nil {
		return nil, err
	}

	fullDescription := strings.TrimSpace(row[5])
	cleanDescription := strings.TrimSpace(strings.Split(fullDescription, ",")[0])

	for _, re := range p.prefixRegexes {
		cleanDescription = re.ReplaceAllString(cleanDescription, "")
	}

	metadata := make(map[string]string)
	metadata["Origin"] = "Openbank"

	if balance := strings.TrimSpace(row[9]); balance != "" {
		hasher := md5.New()
		hasher.Write([]byte(balance))
		metadata["ID"] = fmt.Sprintf("%x", hasher.Sum(nil))[:8]
	}

	if payedBy := p.resolvePayer(fullDescription); payedBy != "" {
		metadata["PayedBy"] = payedBy
	}

	targetAccount := p.resolveAccount(cleanDescription, amount)

	return &domain.Transaction{
		Date:        date,
		Status:      domain.StatusPending,
		Description: cleanDescription,
		Metadata:    metadata,
		Postings: []domain.Posting{
			{Account: "Assets:Checking:OpenBank", Amount: &amount, Currency: "EUR"},
			{Account: targetAccount},
		},
	}, nil
}

func (p *OpenBankParser) resolveAccount(description string, amount float64) string {
	account := ""

	if amount > 0 {
		account = "Income:Unknown"
	} else {
		account = "Expenses:Unknown"
	}

	for _, keyword := range p.sortedKeywords {
		if strings.Contains(strings.ToUpper(description), strings.ToUpper(keyword)) {
			account = p.accountMappings[keyword]
			break
		}
	}

	return account
}

func (p *OpenBankParser) resolvePayer(fullDescription string) string {
	payer := ""

	for cardNumber, owner := range p.cardMappings {
		if strings.Contains(fullDescription, cardNumber) {
			payer = owner
			break
		}
	}

	return payer
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
