package domain

import (
	"cmp"
	"regexp"
	"slices"
	"strings"
)

// MappingData holds the raw configuration for transaction mappings.
type MappingData struct {
	Accounts     map[string]string `json:"accounts"`
	Descriptions map[string]string `json:"descriptions"`
	Cards        map[string]string `json:"cards"`
	Prefixes     []string          `json:"prefixes"`
}

// MappingService provides logic for cleaning descriptions and resolving accounts/payers.
type MappingService struct {
	accountMappings           map[string]string
	descriptionMappings       map[string]string
	sortedAccountKeywords     []string
	sortedDescriptionKeywords []string
	cardMappings              map[string]string
	prefixRegexes             []*regexp.Regexp
}

// NewMappingService creates and initializes a MappingService.
func NewMappingService(data MappingData) *MappingService {
	sortedAccountKeywords := sortKeywords(data.Accounts)
	sortedDescriptionKeywords := sortKeywords(data.Descriptions)

	prefixRegexes := make([]*regexp.Regexp, 0, len(data.Prefixes))
	for _, prefix := range data.Prefixes {
		pattern := "(?i)^" + regexp.QuoteMeta(prefix) + `\s*`
		if regex, err := regexp.Compile(pattern); err == nil {
			prefixRegexes = append(prefixRegexes, regex)
		}
	}

	return &MappingService{
		accountMappings:           data.Accounts,
		descriptionMappings:       data.Descriptions,
		sortedAccountKeywords:     sortedAccountKeywords,
		sortedDescriptionKeywords: sortedDescriptionKeywords,
		cardMappings:              data.Cards,
		prefixRegexes:             prefixRegexes,
	}
}

// CleanDescription strips configured prefixes and applies description mappings.
func (s *MappingService) CleanDescription(description string) string {
	clean := strings.TrimSpace(description)
	for _, re := range s.prefixRegexes {
		clean = re.ReplaceAllString(clean, "")
	}
	clean = strings.TrimSpace(clean)

	result := clean
	if match, ok := s.findMatch(clean, s.sortedDescriptionKeywords, s.descriptionMappings); ok {
		result = match
	}

	return result
}

// ResolveAccount matches description against keywords to determine the target account.
func (s *MappingService) ResolveAccount(description string, amount float64) string {
	account := ""

	if match, ok := s.findMatch(description, s.sortedAccountKeywords, s.accountMappings); ok {
		account = match
	} else if amount > 0 {
		account = "Income:Unknown"
	} else {
		account = "Expenses:Unknown"
	}

	return account
}

// ResolvePayer matches card numbers in the full description to their owners.
func (s *MappingService) ResolvePayer(fullDescription string) string {
	payer := ""

	for cardNumber, owner := range s.cardMappings {
		if strings.Contains(fullDescription, cardNumber) {
			payer = owner
			break
		}
	}

	return payer
}

func (s *MappingService) findMatch(text string, keywords []string, mappings map[string]string) (string, bool) {
	upperText := strings.ToUpper(text)
	result := ""
	found := false

	for _, keyword := range keywords {
		if strings.Contains(upperText, strings.ToUpper(keyword)) {
			result = mappings[keyword]
			found = true
			break
		}
	}

	return result, found
}

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
