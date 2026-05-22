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

var entryRegex = regexp.MustCompile(`^(?:([a-zA-Z]+)\s+)?(\d+([.,]\d+)?)\s+(.+)$`)

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

	amount, err := s.parseAmount(matches[2])
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("invalid amount format: %w", err)
	}

	appConfig := s.configUseCase.Get()
	cleanDescription := appConfig.Mappings.CleanDescription(matches[4])
	targetAccount := s.resolveTargetAccount(appConfig, cleanDescription, amount)
	sourceAccount := s.resolveSourceAccount(appConfig, matches[1])

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
		Postings: []domain.Posting{
			{Account: targetAccount, Amount: &amount, Currency: appConfig.Settings.DefaultCurrency},
			{Account: sourceAccount, Amount: nil},
		},
	}
	tx.Code = tx.GenerateCode()

	return tx, nil
}

/*
parseAmount handles numeric conversion from raw input strings.
It supports both dot and comma as decimal separators.
*/
func (s *TransactionParserService) parseAmount(amountStr string) (float64, error) {
	normalized := strings.Replace(amountStr, ",", ".", 1)
	return strconv.ParseFloat(normalized, 64)
}

/*
resolveTargetAccount determines the expense/income account for the transaction.
It uses mapping keywords first, and if the result is unknown, it attempts to
find the best ranked match as a suggestion.
*/
func (s *TransactionParserService) resolveTargetAccount(appConfig *ports.AppConfig, cleanDescription string, amount float64) string {
	account := appConfig.Mappings.ResolveAccount(cleanDescription, amount, appConfig.Settings.DefaultIncomeAccount, appConfig.Settings.DefaultExpenseAccount)

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
	isAlpha := regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(words[0])
	isAmount := regexp.MustCompile(`^\d`).MatchString(words[1])

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
