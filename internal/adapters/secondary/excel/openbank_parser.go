package excel

import (
	"github.com/a-perez/finance-app/internal/domain"
)

// OpenBankParser handles OpenBank-specific file formats.
type OpenBankParser struct{}

// NewOpenBankParser creates a new instance of OpenBankParser.
func NewOpenBankParser() *OpenBankParser {
	return &OpenBankParser{}
}

// Parse implements ports.BankParser.
func (p *OpenBankParser) Parse(filePath string) ([]domain.Transaction, error) {
	// (TODO) Implement actual parsing logic.
	return []domain.Transaction{}, nil
}
