package excel

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

// Ensure ParserFactory implements ports.FileParserProvider at compile time.
var _ ports.FileParserProvider = (*ParserFactory)(nil)

// BankParserConstructor is a function that creates a BankParser.
type BankParserConstructor func(mappings ports.MappingProvider, settings domain.Settings) ports.BankParser

var (
	registry = make(map[string]BankParserConstructor)
	// aliases allows matching multiple filenames to the same parser
	aliases = make(map[string]string)
)

func init() {
	RegisterParser("openbank", func(m ports.MappingProvider, s domain.Settings) ports.BankParser {
		return NewOpenBankParser(m, s)
	})
	RegisterAlias("extractdocument", "openbank")

	RegisterParser("imagin", func(m ports.MappingProvider, s domain.Settings) ports.BankParser {
		return NewImaginBankParser(m, s)
	})
}

// RegisterParser adds a new parser constructor to the global registry.
func RegisterParser(name string, constructor BankParserConstructor) {
	registry[strings.ToLower(name)] = constructor
}

// RegisterAlias links a filename keyword to an existing parser name.
func RegisterAlias(alias, parserName string) {
	aliases[strings.ToLower(alias)] = strings.ToLower(parserName)
}

// ParserFactory identifies the correct parser based on the filename or explicit type.
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
	target := strings.ToLower(parserType)
	if target == "" {
		target = strings.ToLower(filepath.Base(filePath))
	}

	// 1. Direct match in registry
	for name, constructor := range registry {
		if strings.Contains(target, name) {
			return constructor(appConfig.Mappings, appConfig.Settings), nil
		}
	}

	// 2. Alias match
	for alias, name := range aliases {
		if strings.Contains(target, alias) {
			if constructor, ok := registry[name]; ok {
				return constructor(appConfig.Mappings, appConfig.Settings), nil
			}
		}
	}

	return nil, fmt.Errorf("no parser found for target: %s (available: %v)", target, f.GetAvailableParsers())
}

// GetAvailableParsers returns the list of supported bank parser keys.
func (f *ParserFactory) GetAvailableParsers() []string {
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
