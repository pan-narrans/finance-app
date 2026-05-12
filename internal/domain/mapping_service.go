package domain

import (
	"cmp"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/a-perez/finance-app/internal/config"
)

// MappingService provides logic for cleaning descriptions and resolving accounts/payers.
type MappingService struct {
	accountMappings           map[string]string
	descriptionMappings       map[string]string
	sourceMappings            map[string]string
	sortedAccountKeywords     []string
	sortedDescriptionKeywords []string
	cardMappings              map[string]string
	prefixRegexes             []*regexp.Regexp
	accounts                  []string
	cfg                       config.Config
}

// NewMappingService creates and initializes a MappingService.
func NewMappingService(data config.MappingData, cfg config.Config) *MappingService {
	sortedAccountKeywords := sortKeywords(data.Accounts)
	sortedDescriptionKeywords := sortKeywords(data.Descriptions)

	prefixRegexes := make([]*regexp.Regexp, 0, len(data.Prefixes))
	for _, prefix := range data.Prefixes {
		pattern := "(?i)^" + regexp.QuoteMeta(prefix) + `\s*`
		if regex, err := regexp.Compile(pattern); err == nil {
			prefixRegexes = append(prefixRegexes, regex)
		}
	}

	uniqueAccounts := slices.Sorted(maps.Values(data.Accounts))
	uniqueAccounts = slices.Compact(uniqueAccounts)

	return &MappingService{
		accountMappings:           data.Accounts,
		descriptionMappings:       data.Descriptions,
		sourceMappings:            data.Sources,
		sortedAccountKeywords:     sortedAccountKeywords,
		sortedDescriptionKeywords: sortedDescriptionKeywords,
		cardMappings:              data.Cards,
		prefixRegexes:             prefixRegexes,
		accounts:                  uniqueAccounts,
		cfg:                       cfg,
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
		account = s.cfg.DefaultIncomeAccount
	} else {
		account = s.cfg.DefaultExpenseAccount
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

// ResolveSource returns the account associated with a keyword (e.g., 'alex' -> 'Income:Alex').
// It returns the account and true if found, otherwise empty string and false.
func (s *MappingService) ResolveSource(keyword string) (string, bool) {
	if keyword == "" {
		return "", false
	}

	account, exists := s.sourceMappings[strings.ToLower(keyword)]
	return account, exists
}

// SearchAccounts returns ranked account matches for a query.
// TODO refactor, this method is way too big
func (s *MappingService) SearchAccounts(query string, limit int) []string {
	if query == "" {
		return nil
	}

	queryUpper := strings.ToUpper(query)
	tokens := strings.Fields(queryUpper)

	type scoredAccount struct {
		name  string
		score int
	}

	// map[accountName]maxScore
	scores := make(map[string]int)

	// Helper to calculate score for a string
	calcScore := func(text string) (int, bool) {
		textUpper := strings.ToUpper(text)
		score := 0
		matchesAll := true
		for _, token := range tokens {
			if strings.Contains(textUpper, token) {
				score += len(token)
			} else {
				matchesAll = false
			}
		}
		if !matchesAll {
			return 0, false
		}
		if strings.Contains(textUpper, queryUpper) {
			score += 100
		}
		if strings.HasPrefix(textUpper, queryUpper) {
			score += 50
		}
		return score, true
	}

	// 1. Search in unique account names
	for _, acc := range s.accounts {
		if score, ok := calcScore(acc); ok {
			scores[acc] = score
		}
	}

	// 2. Search in mapping keys
	for key, acc := range s.accountMappings {
		if score, ok := calcScore(key); ok {
			// Mapping key matches are slightly penalized vs direct name matches
			// to prioritize names if both match.
			mappingScore := score - 1
			if mappingScore > scores[acc] {
				scores[acc] = mappingScore
			}
		}
	}

	scored := make([]scoredAccount, 0, len(scores))
	for acc, score := range scores {
		scored = append(scored, scoredAccount{acc, score})
	}

	// Sort by score (desc), then name (asc)
	slices.SortFunc(
		scored, func(a, b scoredAccount) int {
			if a.score != b.score {
				return cmp.Compare(b.score, a.score)
			}
			return cmp.Compare(a.name, b.name)
		},
	)

	resultCount := len(scored)
	if limit > 0 && limit < resultCount {
		resultCount = limit
	}

	results := make([]string, resultCount)
	for i := 0; i < resultCount; i++ {
		results[i] = scored[i].name
	}

	return results
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
