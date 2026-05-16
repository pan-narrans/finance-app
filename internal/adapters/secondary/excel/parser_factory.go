package excel

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/config"
)

// Ensure ParserFactory implements ports.FileParserProvider at compile time.
var _ ports.FileParserProvider = (*ParserFactory)(nil)

// ParserFactory identifies the correct parser based on the filename.
type ParserFactory struct {
	configManager *config.Manager
}

// NewParserFactory creates a new instance of ParserFactory.
func NewParserFactory(configManager *config.Manager) *ParserFactory {
	return &ParserFactory{
		configManager: configManager,
	}
}

// GetParser returns a BankParser implementation matched by filename keyword.
func (f *ParserFactory) GetParser(filePath string) (ports.BankParser, error) {
	fileName := strings.ToLower(filepath.Base(filePath))
	appConfig := f.configManager.Get()

	switch {
	case strings.Contains(fileName, "openbank"):
		return NewOpenBankParser(appConfig.Mappings, appConfig.Settings), nil
	case strings.Contains(fileName, "imaginbank"):
		return NewImaginBankParser(appConfig.Mappings, appConfig.Settings), nil
	default:
		return nil, fmt.Errorf("no parser found for file: %s", fileName)
	}
}
