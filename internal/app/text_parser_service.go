package app

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
)

var entryRegex = regexp.MustCompile(`^(?:([a-zA-Z]+)\s+)?(\d+([.,]\d+)?)\s+(.+)$`)

/*
TextParserService handles the conversion of raw text input into domain transactions.
It implements the ports.TextParserUseCase interface.
*/
type TextParserService struct {
	mappingService *domain.MappingService
	cfg            config.Config
}

/*
NewTextParserService creates a new TextParserService.
*/
func NewTextParserService(mappingService *domain.MappingService, cfg config.Config) *TextParserService {
	return &TextParserService{
		mappingService: mappingService,
		cfg:            cfg,
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
func (s *TextParserService) ParseText(text, origin string) (domain.Transaction, error) {
	matches := entryRegex.FindStringSubmatch(text)
	if len(matches) < 5 {
		return domain.Transaction{}, fmt.Errorf("format not recognized; use: '[source] amount description'")
	}

	sourceKeyword := matches[1]
	amountStr := strings.Replace(matches[2], ",", ".", 1)
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("invalid amount format: %w", err)
	}

	description := s.mappingService.CleanDescription(matches[4])
	targetAccount := s.mappingService.ResolveAccount(description, amount)

	// Resolve income/source account
	sourceAccount := s.cfg.DefaultBotAccount
	if sourceKeyword != "" {
		if acc, found := s.mappingService.ResolveSource(sourceKeyword); found {
			sourceAccount = acc
		} else {
			// Fallback: if source name provided but no mapping, use Income:[Source]
			sourceAccount = fmt.Sprintf("Income:%s", strings.Title(strings.ToLower(sourceKeyword)))
		}
	}

	// Auto-pick if Unknown
	if strings.HasSuffix(targetAccount, ":Unknown") {
		suggestions := s.mappingService.SearchAccounts(description, 1)
		if len(suggestions) > 0 {
			targetAccount = suggestions[0]
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
		Description: description,
		Metadata:    metadata,
		Postings: []domain.Posting{
			{Account: targetAccount, Amount: &amount, Currency: s.cfg.DefaultCurrency},
			{Account: sourceAccount, Amount: nil},
		},
	}
	tx.Code = tx.GenerateCode()

	return tx, nil
}

/*
hashID returns an 8-character MD5 hash of the provided string.
Used for generating stable external IDs for bot transactions.
*/
func (s *TextParserService) hashID(data string) string {
	if data == "" {
		return ""
	}
	hasher := md5.New()
	hasher.Write([]byte(data))
	return fmt.Sprintf("%x", hasher.Sum(nil))[:8]
}
