package excel

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/a-perez/finance-app/internal/app/ports"
)

// ParserFactory identifies the correct parser based on the filename.
type ParserFactory struct {
	configRoot string
}

// NewParserFactory creates a new instance of ParserFactory.
func NewParserFactory(configRoot string) *ParserFactory {
	return &ParserFactory{configRoot: configRoot}
}

// GetParser returns a BankParser implementation matched by filename keyword.
func (f *ParserFactory) GetParser(filePath string) (ports.BankParser, error) {
	fileName := strings.ToLower(filepath.Base(filePath))
	mappingsPath := filepath.Join(f.configRoot, "mappings.json")

	switch {
	case strings.Contains(fileName, "openbank"):
		return NewOpenBankParser(mappingsPath), nil
	case strings.Contains(fileName, "imaginbank"):
		return NewImaginBankParser(mappingsPath), nil
	default:
		return nil, fmt.Errorf("no parser found for file: %s", fileName)
	}
}
