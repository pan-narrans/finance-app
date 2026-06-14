package excel

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/a-perez/finance-app/internal/app/ports"
)

// Ensure ParserFactory implements ports.FileParserProvider at compile time.
var _ ports.FileParserProvider = (*ParserFactory)(nil)

// ParserFactory identifies the correct parser based on the filename.
type ParserFactory struct {
	configUseCase ports.ConfigurationUseCase
}

// NewParserFactory creates a new instance of ParserFactory.
func NewParserFactory(configUseCase ports.ConfigurationUseCase) *ParserFactory {
	return &ParserFactory{
		configUseCase: configUseCase,
	}
}

// GetParser returns a BankParser implementation matched by filename keyword or explicit type.
func (f *ParserFactory) GetParser(filePath string, parserType string) (ports.BankParser, error) {
	appConfig := f.configUseCase.Get()
	target := parserType
	if target == "" {
		target = strings.ToLower(filepath.Base(filePath))
	} else {
		target = strings.ToLower(target)
	}

	switch {
	case strings.Contains(target, "openbank") || strings.Contains(target, "extractdocument"):
		return NewOpenBankParser(appConfig.Mappings, appConfig.Settings), nil
	case strings.Contains(target, "imagin"):
		return NewImaginBankParser(appConfig.Mappings, appConfig.Settings), nil
	default:
		return nil, fmt.Errorf("no parser found for target: %s", target)
	}
}

// GetAvailableParsers returns the list of supported bank parser keys.
func (f *ParserFactory) GetAvailableParsers() []string {
	return []string{"imagin", "openbank"}
}

