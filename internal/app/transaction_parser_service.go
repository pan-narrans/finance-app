package app

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Ensure TransactionParserService implements ports.TransactionParserUseCase at compile time.
var _ ports.TransactionParserUseCase = (*TransactionParserService)(nil)

var (
	entryRegex  = regexp.MustCompile(`^(?:([a-zA-Z]+)\s+)?(\d+([.,]\d+)?)\s+(.+)$`)
	alphaRegex  = regexp.MustCompile(`^[a-zA-Z]+$`)
	amountRegex = regexp.MustCompile(`^\d`)
)

/*
TransactionParserService handles the conversion of raw text input into domain transactions.
It implements the ports.TransactionParserUseCase interface.
*/
type TransactionParserService struct {
	configUseCase ports.ConfigurationUseCase
}

/*
NewTransactionParserService creates a new TransactionParserService.
*/
func NewTransactionParserService(configUseCase ports.ConfigurationUseCase) *TransactionParserService {
	return &TransactionParserService{
		configUseCase: configUseCase,
	}
}

/*
ParseText converts a raw string (e.g., "cash 12.50 dinner") into a domain.Transaction.

Format: "[source] amount description"
- source: Optional keyword for the origin account (e.g., "cash").
- amount: Numeric value (supports dot or comma as decimal separator).
- description: Textual description of the transaction.

It uses MappingService to clean the description and resolve accounts.
*/
func (s *TransactionParserService) ParseText(text, origin string) (domain.Transaction, error) {
	matches := entryRegex.FindStringSubmatch(text)
	if len(matches) < 5 {
		return domain.Transaction{}, fmt.Errorf("format not recognized; use: '[source] amount description'")
	}

	appConfig := s.configUseCase.Get()
	sourceKeyword := matches[1]
	amountStr := matches[2]
	description := matches[4]

	// Strict Source Validation: Only use sourceKeyword if it's a known mapping.
	if sourceKeyword != "" {
		if _, found := appConfig.Mappings.ResolveSource(sourceKeyword); !found {
			description = sourceKeyword + " " + description
			sourceKeyword = ""
		}
	}

	amount, err := s.parseAmount(amountStr)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("invalid amount format: %w", err)
	}

	cleanDescription := appConfig.Mappings.CleanDescription(description)
	targetAccount := s.resolveTargetAccount(appConfig, cleanDescription)
	sourceAccount := s.resolveSourceAccount(appConfig, sourceKeyword)

	// Convention: Postings[0] is Target (Debit), Postings[1] is Source (Credit)
	var postings []domain.Posting
	isIncome := appConfig.Mappings.IsIncomeAccount(targetAccount)

	if isIncome {
		// Income: Assets (Target) increase, Income (Source) remains credit balance
		postings = []domain.Posting{
			{Account: sourceAccount, Amount: &amount, Currency: appConfig.Settings.DefaultCurrency},
			{Account: targetAccount, Amount: nil},
		}
	} else {
		// Expense/Transfer: Expense (Target) increases, Assets (Source) decrease
		postings = []domain.Posting{
			{Account: targetAccount, Amount: &amount, Currency: appConfig.Settings.DefaultCurrency},
			{Account: sourceAccount, Amount: nil},
		}
	}

	// Add Metadata
	metadata := domain.Metadata{
		Origin: origin,
		ID:     s.hashID(fmt.Sprintf("%d", time.Now().UnixNano())),
	}

	// Create transaction
	tx := domain.Transaction{
		Date:        time.Now(),
		Status:      domain.StatusPending,
		Description: cleanDescription,
		Metadata:    metadata,
		Postings:    postings,
	}
	tx.Code = tx.GenerateCode()

	return tx, nil
}

/*
parseAmount handles numeric conversion from raw input strings.
It supports international formats (e.g., 1,234.56 or 1.234,56) by
identifying thousands separators versus decimal points.
*/
func (s *TransactionParserService) parseAmount(amountStr string) (float64, error) {
	normalized := strings.TrimSpace(amountStr)

	lastComma := strings.LastIndex(normalized, ",")
	lastDot := strings.LastIndex(normalized, ".")

	if lastComma != -1 && lastDot != -1 {
		// Both present: the last one is the decimal separator
		if lastComma > lastDot {
			normalized = strings.ReplaceAll(normalized, ".", "")
			normalized = strings.Replace(normalized, ",", ".", 1)
		} else {
			normalized = strings.ReplaceAll(normalized, ",", "")
		}
	} else if lastComma != -1 {
		// Only commas: more than one means they are thousands separators
		if strings.Count(normalized, ",") > 1 {
			normalized = strings.ReplaceAll(normalized, ",", "")
		} else {
			// Single comma: assume decimal unless it looks like a thousands separator (e.g., "1,000")
			// but we prefer decimal for simplicity in common mobile inputs.
			normalized = strings.Replace(normalized, ",", ".", 1)
		}
	} else if lastDot != -1 {
		// Only dots: more than one means they are thousands separators
		if strings.Count(normalized, ".") > 1 {
			normalized = strings.ReplaceAll(normalized, ".", "")
		}
	}

	return strconv.ParseFloat(normalized, 64)
}

/*
resolveTargetAccount determines the expense/income account for the transaction.
It uses mapping keywords first, and if the result is unknown, it attempts to
find the best ranked match as a suggestion.
*/
func (s *TransactionParserService) resolveTargetAccount(appConfig *ports.AppConfig, cleanDescription string) string {
	account, found := appConfig.Mappings.ResolveAccount(cleanDescription)
	if !found {
		account = appConfig.Settings.DefaultExpenseAccount
	}

	// Auto-pick if Unknown
	if strings.HasSuffix(account, ":Unknown") {
		suggestions := appConfig.Mappings.SearchAccounts(cleanDescription, 1)
		if len(suggestions) > 0 {
			account = suggestions[0]
		}
	}

	return account
}

/*
resolveSourceAccount determines the asset/origin account for the transaction.
If the keyword matches a source mapping, it uses that account.
If no mapping exists but a keyword is provided, it falls back to Income:[Keyword].
Otherwise, it returns the default asset account.
*/
func (s *TransactionParserService) resolveSourceAccount(appConfig *ports.AppConfig, sourceKeyword string) string {
	if sourceKeyword == "" {
		return appConfig.Settings.DefaultAssetAccount
	}

	if account, found := appConfig.Mappings.ResolveSource(sourceKeyword); found {
		return account
	}

	// Fallback: if source name provided but no mapping, use Income:[Source]
	titleCase := cases.Title(language.Und)
	return fmt.Sprintf("Income:%s", titleCase.String(strings.ToLower(sourceKeyword)))
}

/*
GuessSource attempts to identify a potential source keyword from the input text.
It uses a heuristic: if the first word is alphabetic and followed by a number,
it is likely intended as the source account keyword.
*/
func (s *TransactionParserService) GuessSource(text string) string {
	words := strings.Fields(text)
	if len(words) < 2 {
		return ""
	}

	// Heuristic: first word alphabetic, second word starts with digit (amount)
	isAlpha := alphaRegex.MatchString(words[0])
	isAmount := amountRegex.MatchString(words[1])

	if isAlpha && isAmount {
		return strings.ToLower(words[0])
	}

	return ""
}

/*
hashID returns an 8-character MD5 hash of the provided string.
...
Used for generating stable external IDs for bot transactions.
*/
func (s *TransactionParserService) hashID(data string) string {
	if data == "" {
		return ""
	}
	hasher := md5.New()
	hasher.Write([]byte(data))
	return fmt.Sprintf("%x", hasher.Sum(nil))[:8]
}
