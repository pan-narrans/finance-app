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

	upperClean := strings.ToUpper(clean)
	for _, keyword := range b.sortedDescriptionKeywords {
		if strings.Contains(upperClean, strings.ToUpper(keyword)) {
			return b.descriptionMappings[keyword]
		}
	}

	return clean
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
	for _, keyword := range b.sortedAccountKeywords {
		if strings.Contains(upperDesc, strings.ToUpper(keyword)) {
			account = b.accountMappings[keyword]
			break
		}
	}

	return account
}

// HashID returns an 8-character MD5 hash of the provided string.
// Used for bank-provided balances to create stable external IDs.
func (b *BaseParser) HashID(data string) string {
	if data == "" {
		return ""
	}
	hasher := md5.New()
	hasher.Write([]byte(data))
	return fmt.Sprintf("%x", hasher.Sum(nil))[:8]
}

func ParseSpanishAmount(s string) (float64, error) {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}

func sortKeywords(m map[string]string) []string {
	keywords := make([]string, 0, len(m))
	for k := range m {
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
	return keywords
}
