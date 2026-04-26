package excel

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type mappingsData struct {
	Accounts map[string]string `json:"accounts"`
	Cards    map[string]string `json:"cards"`
	Prefixes []string          `json:"prefixes"`
}

// BaseParser encapsulates shared logic for all Excel-based parsers.
type BaseParser struct {
	accountMappings map[string]string
	cardMappings    map[string]string
	sortedKeywords  []string
	prefixRegexes   []*regexp.Regexp
}

// NewBaseParser creates and initializes a BaseParser from a mappings file.
func NewBaseParser(mappingsPath string) *BaseParser {
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

	return &BaseParser{
		accountMappings: data.Accounts,
		cardMappings:    data.Cards,
		sortedKeywords:  keywords,
		prefixRegexes:   prefixRegexes,
	}
}

// CleanDescription strips configured prefixes from the description.
func (b *BaseParser) CleanDescription(description string) string {
	clean := strings.TrimSpace(description)
	for _, re := range b.prefixRegexes {
		clean = re.ReplaceAllString(clean, "")
	}
	return strings.TrimSpace(clean)
}

// ResolveAccount matches description against keywords to determine the target account.
func (b *BaseParser) ResolveAccount(description string, amount float64) string {
	account := ""

	if amount > 0 {
		account = "Income:Unknown"
	} else {
		account = "Expenses:Unknown"
	}

	upperDesc := strings.ToUpper(description)
	for _, keyword := range b.sortedKeywords {
		if strings.Contains(upperDesc, strings.ToUpper(keyword)) {
			account = b.accountMappings[keyword]
			break
		}
	}

	return account
}

func ParseSpanishAmount(s string) (float64, error) {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}
