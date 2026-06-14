package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type EntryType int

const (
	EntryTypeTransaction EntryType = iota
	EntryTypePrice
	EntryTypeDirective // commodity, account, tag, etc.
	EntryTypeComment
)

/*
LedgerEntry represents a discrete block in the ledger file.
It can be a transaction, a price update, or a raw comment/directive.
*/
type LedgerEntry struct {
	Type    EntryType
	Date    time.Time // Zero value for non-dated entries
	RawText string
}

/*
Ledger represents a complete collection of ledger entries.
*/
type Ledger struct {
	Entries []LedgerEntry
}

/*
Sort reorders the dated entries (Transactions and Prices) chronologically.
It uses a stable sort to preserve the relative order of entries with the same date.
Non-dated entries (Global Directives/Comments) at the top of the file are preserved
as a "prologue".
*/
func (l *Ledger) Sort() {
	if len(l.Entries) == 0 {
		return
	}

	// 1. Identify the prologue (all non-dated entries at the start)
	prologueEnd := 0
	for i, entry := range l.Entries {
		if !entry.Date.IsZero() {
			prologueEnd = i
			break
		}
	}

	prologue := l.Entries[:prologueEnd]
	sortable := l.Entries[prologueEnd:]

	// 2. Stable sort dated entries
	slices.SortStableFunc(sortable, func(a, b LedgerEntry) int {
		if a.Date.Before(b.Date) {
			return -1
		}
		if a.Date.After(b.Date) {
			return 1
		}
		return 0
	})

	// 3. Reassemble
	l.Entries = append(prologue, sortable...)
}

func (l *Ledger) Format() string {
	var sb strings.Builder
	var lastMonth time.Month
	var lastYear int

	for i, entry := range l.Entries {
		// Detect month boundary for dated entries
		if !entry.Date.IsZero() {
			month := entry.Date.Month()
			year := entry.Date.Year()

			if month != lastMonth || year != lastYear {
				monthName := strings.ToUpper(month.String())
				
				// Spacing before header
				if sb.Len() > 0 {
					// Ensure we have exactly two newlines before a new month header
					current := sb.String()
					if !strings.HasSuffix(current, "\n\n") {
						if strings.HasSuffix(current, "\n") {
							sb.WriteString("\n")
						} else {
							sb.WriteString("\n\n")
						}
					}
				}
				
				sb.WriteString(";--------\n")
				sb.WriteString(fmt.Sprintf(";- %s -\n", monthName))
				sb.WriteString(";--------\n\n")

				lastMonth = month
				lastYear = year
			}
		}

		sb.WriteString(entry.RawText)
		
		// Ensure entry ends with newline
		if !strings.HasSuffix(entry.RawText, "\n") {
			sb.WriteString("\n")
		}

		// Add spacing between entries
		if i < len(l.Entries)-1 {
			// Double newline after transaction/price blocks
			if entry.Type == EntryTypeTransaction || entry.Type == EntryTypePrice {
				sb.WriteString("\n")
			}
		}
	}

	result := sb.String()
	// Clean up any accidental leading/trailing whitespace
	result = strings.TrimSpace(result)
	if result != "" {
		result += "\n"
	}
	return result
}


