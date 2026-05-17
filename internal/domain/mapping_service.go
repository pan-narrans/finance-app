package domain

import (
	"cmp"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// MappingData holds the raw configuration for transaction mappings.
type MappingData struct {
	Accounts     map[string]string `json:"accounts"`
	Descriptions map[string]string `json:"descriptions"`
	Sources      map[string]string `json:"sources"`
	Cards        map[string]string `json:"cards"`
	Prefixes     []string          `json:"prefixes"`
}

/*
MappingService provides logic for cleaning descriptions and resolving financial entities.

It acts as a translation layer between raw input data (e.g., bank statements)
and domain-specific values (accounts, payers, sources) using configurable rules.
*/
type MappingService struct {
	data                      MappingData
	accountMappings           map[string]string
	descriptionMappings       map[string]string
	sourceMappings            map[string]string
	sortedAccountKeywords     []string
	sortedDescriptionKeywords []string
	cardMappings              map[string]string
	prefixRegexes             []*regexp.Regexp
	accounts                  []string
}

/*
NewMappingService creates and initializes a MappingService.

It pre-processes mapping data by:
  - Sorting keywords by length (descending) to ensure longest-match priority.
  - Compiling case-insensitive prefix regular expressions.
  - Extracting a unique, sorted list of known account names.
*/
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

	uniqueAccounts := slices.Sorted(maps.Values(data.Accounts))
	uniqueAccounts = slices.Compact(uniqueAccounts)

	return &MappingService{
		data:                      data,
		accountMappings:           data.Accounts,
		descriptionMappings:       data.Descriptions,
		sourceMappings:            data.Sources,
		sortedAccountKeywords:     sortedAccountKeywords,
		sortedDescriptionKeywords: sortedDescriptionKeywords,
		cardMappings:              data.Cards,
		prefixRegexes:             prefixRegexes,
		accounts:                  uniqueAccounts,
	}
}

/*
GetData returns the raw mapping data.
*/
// TODO Rename to GetMappingData
func (s *MappingService) GetData() MappingData {
	return s.data
}

/*
CleanDescription strips configured prefixes and applies description mappings.

The process follows these steps:
 1. Trim whitespace.
 2. Remove all matching configured prefixes (e.g., "PURCHASE ").
 3. Re-trim whitespace.
 4. Replace the remaining string if it matches a description keyword.
*/
func (s *MappingService) CleanDescription(description string) string {
	clean := strings.TrimSpace(description)
	for _, regex := range s.prefixRegexes {
		clean = regex.ReplaceAllString(clean, "")
	}
	clean = strings.TrimSpace(clean)

	result := clean
	if match, ok := s.findMatch(clean, s.sortedDescriptionKeywords, s.descriptionMappings); ok {
		result = match
	}

	return result
}

/*
ResolveAccount matches description against keywords to determine the target account.

Resolution logic:
  - Return mapped account if description contains a known keyword.
  - Fallback to defaultIncome if amount is positive.
  - Fallback to defaultExpense if amount is negative or zero.
*/
func (s *MappingService) ResolveAccount(description string, amount float64, defaultIncome, defaultExpense string) string {
	account := ""

	if match, ok := s.findMatch(description, s.sortedAccountKeywords, s.accountMappings); ok {
		account = match
	} else if amount > 0 {
		account = defaultIncome
	} else {
		account = defaultExpense
	}

	return account
}

/*
ResolvePayer matches card numbers in the full description to their owners.

It iterates through card mappings and returns the owner's name if the
card number is found as a substring within the full description.
*/
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

/*
ResolveSource returns the account associated with a keyword (e.g., 'alex' -> 'Assets:Cash:Alex').

It returns the mapped account name and true if found; otherwise, returns
an empty string and false.
*/
func (s *MappingService) ResolveSource(keyword string) (string, bool) {
	if keyword == "" {
		return "", false
	}

	account, exists := s.sourceMappings[strings.ToLower(keyword)]
	return account, exists
}

/*
SearchAccounts returns ranked account matches for a query.

The scoring algorithm prioritizes:
  - Exact substring matches of the full query (highest).
  - Prefix matches.
  - Partial token matches.
  - Direct account name matches over mapping keyword matches.

It returns a slice of account names ranked by relevance. If limit is greater than zero,
the result is truncated to that size. If limit is zero or negative, all matches
are returned.
*/
func (s *MappingService) SearchAccounts(query string, limit int) []string {
	if query == "" {
		return nil
	}

	queryUpper := strings.ToUpper(query)
	tokens := strings.Fields(queryUpper)

	accountScores := s.scoreAccounts(queryUpper, tokens)
	mappingScores := s.scoreMappingKeys(queryUpper, tokens)

	// Merge scores, keeping the highest score for each account
	for account, score := range mappingScores {
		if score > accountScores[account] {
			accountScores[account] = score
		}
	}

	return sortAndLimitResults(accountScores, limit)
}

/*
findMatch searches for the first keyword contained within the text.
It returns the mapped value and true if found; otherwise, empty string and false.
Matches are case-insensitive.
*/
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

// scoredAccount represents an account name paired with its search relevance score.
type scoredAccount struct {
	name  string
	score int
}

// scoreAccounts returns a map of scores based on matches against direct account names.
func (s *MappingService) scoreAccounts(queryUpper string, tokens []string) map[string]int {
	scores := make(map[string]int)
	for _, account := range s.accounts {
		if score, ok := calculateScore(account, queryUpper, tokens); ok {
			scores[account] = score
		}
	}
	return scores
}

/*
scoreMappingKeys returns a map of scores based on matches against mapping keywords.
Mapping matches are slightly penalized to prioritize direct account name matches.
*/
func (s *MappingService) scoreMappingKeys(queryUpper string, tokens []string) map[string]int {
	scores := make(map[string]int)
	for key, account := range s.accountMappings {
		if score, ok := calculateScore(key, queryUpper, tokens); ok {
			// Mapping key matches are slightly penalized vs direct name matches
			// to prioritize names if both match.
			mappingScore := score - 1
			if mappingScore > scores[account] {
				scores[account] = mappingScore
			}
		}
	}
	return scores
}

/*
calculateScore computes a match score for a given text against a query and its tokens.

It returns the calculated score and true if all tokens are present in the text.
If any token is missing, it returns 0 and false.

Scoring Rules:
  - All tokens must be present in the text (case-insensitive).
  - Score increases by the length of matching tokens.
  - +100 points for an exact substring match of the full query.
  - +50 points for a prefix match of the full query.
*/
func calculateScore(text, queryUpper string, tokens []string) (int, bool) {
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

/*
sortAndLimitResults converts scores to a ranked list of account names.
Results are sorted by score (descending) then by name (ascending).
*/
func sortAndLimitResults(scores map[string]int, limit int) []string {
	scored := make([]scoredAccount, 0, len(scores))
	for account, score := range scores {
		scored = append(scored, scoredAccount{account, score})
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
	if limit > 0 {
		resultCount = min(limit, resultCount)
	}

	results := make([]string, resultCount)
	for i := 0; i < resultCount; i++ {
		results[i] = scored[i].name
	}

	return results
}

/*
sortKeywords prepares keywords for matching by sorting them by length (descending).
This ensures that the longest possible match is attempted first.
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
