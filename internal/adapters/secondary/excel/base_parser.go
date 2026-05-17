package excel

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

// BaseParser encapsulates shared utility logic for all Excel-based parsers.
type BaseParser struct {
	mappingProvider ports.MappingProvider
	settings        domain.Settings
}

// NewBaseParser creates a new BaseParser instance.
func NewBaseParser(mappingProvider ports.MappingProvider, settings domain.Settings) *BaseParser {
	return &BaseParser{
		mappingProvider: mappingProvider,
		settings:        settings,
	}
}

/*
HashID returns an 8-character MD5 hash of the provided string.
Used for bank-provided balances to create stable external IDs.
*/
func (b *BaseParser) HashID(data string) string {
	if data == "" {
		return ""
	}

	hasher := md5.New()
	hasher.Write([]byte(data))

	return fmt.Sprintf("%x", hasher.Sum(nil))[:8]
}

/*
ParseSpanishAmount converts a Spanish-formatted currency string (e.g., "1.234,56")
to a float64. It removes thousands separators (dots) and replaces the
decimal comma with a dot before parsing.
*/
func ParseSpanishAmount(s string) (float64, error) {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}
