package ledger

import (
	"fmt"
	"slices"
	"strings"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

// Ensure LedgerFormatter implements ports.TransactionFormatter.
var _ ports.TransactionFormatter = (*LedgerFormatter)(nil)

/*
LedgerFormatter encapsulates the logic for converting domain entities into Ledger CLI strings.
It is stateless.
*/
type LedgerFormatter struct{}

/*
NewLedgerFormatter creates a new instance of LedgerFormatter.
*/
func NewLedgerFormatter() *LedgerFormatter {
	return &LedgerFormatter{}
}

/*
FormatTransaction converts a [domain.Transaction] into a plain-text block compatible with Ledger CLI.
The alignment parameter controls the start column for amounts.
*/
func (f *LedgerFormatter) FormatTransaction(t domain.Transaction, alignment int) string {
	var sb strings.Builder

	statusMark := ""
	switch t.Status {
	case domain.StatusCleared:
		statusMark = "* "
	case domain.StatusPending:
		statusMark = "! "
	}

	codeMark := ""
	if t.Code != "" {
		codeMark = fmt.Sprintf("(%s) ", t.Code)
	}

	f.writeLine(&sb, "%s", t.Date.Format("2006/01/02"))
	f.writeLine(&sb, " %s", statusMark)
	f.writeLine(&sb, "%s", codeMark)
	f.writeLine(&sb, "%s", t.Description)
	sb.WriteByte('\n')

	f.addMetadata(&sb, t.Metadata)
	f.addPostings(&sb, t.Postings, alignment)

	return sb.String()
}

func (f *LedgerFormatter) writeLine(builder *strings.Builder, format string, args ...any) {
	for _, arg := range args {
		if s, ok := arg.(string); ok && s == "" {
			return
		}
	}
	_, _ = fmt.Fprintf(builder, format, args...)
}

func (f *LedgerFormatter) addMetadata(builder *strings.Builder, m domain.Metadata) {
	const metadataFormat = "    ; %s: %s\n"

	f.writeLine(builder, metadataFormat, "ID", m.ID)
	f.writeLine(builder, metadataFormat, "Origin", m.Origin)
	f.writeLine(builder, metadataFormat, "PayedBy", m.PayedBy)

	if len(m.Extras) > 0 {
		keys := make([]string, 0, len(m.Extras))
		for k := range m.Extras {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			f.writeLine(builder, metadataFormat, k, m.Extras[k])
		}
	}
}

func (f *LedgerFormatter) addPostings(sb *strings.Builder, postings []domain.Posting, alignment int) {
	for _, posting := range postings {
		f.writeLine(sb, "    %s", posting.Account)

		if posting.Amount != nil {
			padding := alignment - len(posting.Account)
			padding = max(padding, 2)
			sb.WriteString(strings.Repeat(" ", padding))

			if len(posting.Currency) == 1 {
				f.writeLine(sb, "%s%.2f", posting.Currency, *posting.Amount)
			} else {
				f.writeLine(sb, "%.2f %s", *posting.Amount, posting.Currency)
			}
		}

		sb.WriteByte('\n')
	}
}
