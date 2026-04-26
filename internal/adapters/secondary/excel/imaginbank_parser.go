package excel

import (
	"github.com/a-perez/finance-app/internal/domain"
)

// ImaginBankParser handles ImaginBank-specific file formats.
type ImaginBankParser struct {
	*BaseParser
}

// NewImaginBankParser creates a new instance of ImaginBankParser.
func NewImaginBankParser(mappingsPath string) *ImaginBankParser {
	return &ImaginBankParser{
		BaseParser: NewBaseParser(mappingsPath),
	}
}

// Parse implements ports.BankParser.
func (p *ImaginBankParser) Parse(filePath string) ([]domain.Transaction, error) {
	// (TODO) Implement actual parsing logic.
	return []domain.Transaction{}, nil
}
