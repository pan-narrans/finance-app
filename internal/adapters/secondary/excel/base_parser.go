package excel

import (
	"cmp"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type mappingsData struct {
	Accounts     map[string]string `json:"accounts"`
	Descriptions map[string]string `json:"descriptions"`
	Cards        map[string]string `json:"cards"`
	Prefixes     []string          `json:"prefixes"`
}

// BaseParser encapsulates shared logic for all Excel-based parsers.
type BaseParser struct {
	accountMappings           map[string]string
	descriptionMappings       map[string]string
	sortedAccountKeywords     []string
	sortedDescriptionKeywords []string
	cardMappings              map[string]string
	prefixRegexes             []*regexp.Regexp
}

// NewBaseParser creates and initializes a BaseParser from a mappings file.
func NewBaseParser(mappingsPath string) *BaseParser {
	data := mappingsData{
		Accounts:     make(map[string]string),
		Descriptions: make(map[string]string),
		Cards:        make(map[string]string),
		Prefixes:     make([]string, 0),
	}

	if mappingsPath != "" {
		fileData, err := os.ReadFile(mappingsPath)
		if err == nil {
			if err := json.Unmarshal(fileData, &data); err != nil {
				log.Printf("Error unmarshaling mappings: %v", err)
			}
		}
	}

	sortedAccountKeywords := sortKeywords(data.Accounts)
	sortedDescriptionKeywords := sortKeywords(data.Descriptions)

	// Compile prefixes into case-insensitive regexes anchored to start
	prefixRegexes := make([]*regexp.Regexp, 0, len(data.Prefixes))
	for _, prefix := range data.Prefixes {
		pattern := "(?i)^" + regexp.QuoteMeta(prefix) + `\s*`
		if regex, err := regexp.Compile(pattern); err == nil {
			prefixRegexes = append(prefixRegexes, regex)
		}
	}

	return &BaseParser{
		accountMappings:           data.Accounts,
		descriptionMappings:       data.Descriptions,
		sortedAccountKeywords:     sortedAccountKeywords,
		sortedDescriptionKeywords: sortedDescriptionKeywords,
		cardMappings:              data.Cards,
		prefixRegexes:             prefixRegexes,
	}
}

// CleanDescription strips configured prefixes and applies description mappings.
func (b *BaseParser) CleanDescription(description string) string {
	clean := strings.TrimSpace(description)
	for _, re := range b.prefixRegexes {
		clean = re.ReplaceAllString(clean, "")
	}
	clean = strings.TrimSpace(clean)

	if match, ok := b.findMatch(clean, b.sortedDescriptionKeywords, b.descriptionMappings); ok {
		return match
	}

	return clean
}

// ResolveAccount matches description against keywords to determine the target account.
func (b *BaseParser) ResolveAccount(description string, amount float64) string {
	account := ""

	if match, ok := b.findMatch(description, b.sortedAccountKeywords, b.accountMappings); ok {
		account = match
	} else if amount > 0 {
		account = "Income:Unknown"
	} else {
		account = "Expenses:Unknown"
	}

	return account
}

/*
HashID returns an 8-character MD5 hash of the provided string.
Used for bank-provided balances to create stable external IDs.
*/
func (b *BaseParser) HashID(data string) string {
	if data == "" {
		return ""
	}
	hasher := md5.New()
	hasher.Write([]byte(data))
	return fmt.Sprintf("%x", hasher.Sum(nil))[:8]
}

/*
ParseSpanishAmount converts a Spanish-formatted currency string (e.g., "1.234,56")
to a float64. It removes thousands separators (dots) and replaces the
decimal comma with a dot before parsing.
*/
func ParseSpanishAmount(s string) (float64, error) {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}

/*
findMatch performs a case-insensitive keyword search within the provided text.
It iterates through the sorted keywords and returns the mapped value for the
first match found.

Returns (value, true) if found, or ("", false) otherwise.
*/
func (b *BaseParser) findMatch(text string, keywords []string, mappings map[string]string) (string, bool) {
	upperText := strings.ToUpper(text)
	for _, keyword := range keywords {
		if strings.Contains(upperText, strings.ToUpper(keyword)) {
			return mappings[keyword], true
		}
	}
	return "", false
}

/*
sortKeywords returns a slice of keys from the provided map, sorted by length
in descending order (longest first) to ensure deterministic keyword matching.
Keys of equal length are sorted alphabetically.
*/
func sortKeywords(m map[string]string) []string {
	keywords := make([]string, 0, len(m))
	for k := range m {
		keywords = append(keywords, k)
	}
	slices.SortFunc(
		keywords, func(a, b string) int {
			if len(a) != len(b) {
				return cmp.Compare(len(b), len(a))
			}
			return cmp.Compare(a, b)
		},
	)
	return keywords
}
