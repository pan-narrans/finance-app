package excel

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
)

// Ensure ImaginBankParser implements ports.BankParser at compile time.
var _ ports.FileParserProvider = (*ParserFactory)(nil)

// ParserFactory identifies the correct parser based on the filename.
type ParserFactory struct {
	mappingService *domain.MappingService
	cfg            config.Config
}

// NewParserFactory creates a new instance of ParserFactory.
func NewParserFactory(mappingService *domain.MappingService, cfg config.Config) *ParserFactory {
	return &ParserFactory{
		mappingService: mappingService,
		cfg:            cfg,
	}
}

// GetParser returns a BankParser implementation matched by filename keyword.
func (f *ParserFactory) GetParser(filePath string) (ports.BankParser, error) {
	fileName := strings.ToLower(filepath.Base(filePath))

	switch {
	case strings.Contains(fileName, "openbank"):
		return NewOpenBankParser(f.mappingService, f.cfg), nil
	case strings.Contains(fileName, "imaginbank"):
		return NewImaginBankParser(f.mappingService, f.cfg), nil
	default:
		return nil, fmt.Errorf("no parser found for file: %s", fileName)
	}
}
