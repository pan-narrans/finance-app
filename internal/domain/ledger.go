package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

var (
	// entryStartRegex finds ALL entry starts (Transactions OR Prices)
	entryStartRegex = regexp.MustCompile(`(?m)^(P\s+)?(\d{4}[\/-]\d{2}[\/-]\d{2})`)
	// sepRegex finds stylized month headers for stripping
	sepRegex = regexp.MustCompile(`(?m);-+\r?\n;- [A-Z ]+ -\r?\n;-+\r?\n*`)
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
	slices.SortStableFunc(
		sortable, func(a, b LedgerEntry) int {
			if a.Date.Before(b.Date) {
				return -1
			}
			if a.Date.After(b.Date) {
				return 1
			}
			return 0
		},
	)

	// 3. Reassemble
	l.Entries = append(prologue, sortable...)
}

/*
ParseLedger converts the raw text content of a ledger file into a Ledger struct.
It splits the content into discrete blocks (Transactions, Prices, Directives).
*/
func ParseLedger(content string) Ledger {
	matches := entryStartRegex.FindAllStringSubmatchIndex(content, -1)

	if len(matches) == 0 {
		return Ledger{
			Entries: []LedgerEntry{{Type: EntryTypeDirective, RawText: content}},
		}
	}

	var ledger Ledger

	// 1. Capture Prologue (everything before first date)
	prologue := content[:matches[0][0]]
	if prologue != "" {
		ledger.Entries = append(
			ledger.Entries, LedgerEntry{
				Type:    EntryTypeDirective,
				RawText: strings.TrimRight(prologue, "\n \t"),
			},
		)
	}

	// 2. Capture Blocks
	for i, match := range matches {
		isPrice := match[2] != -1 && match[3] != -1
		dateStr := content[match[4]:match[5]]
		dateStr = strings.ReplaceAll(dateStr, "-", "/")
		date, _ := time.Parse("2006/01/02", dateStr)

		start := match[0]
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}

		raw := content[start:end]
		// Strip separators from the block to prevent duplication
		raw = sepRegex.ReplaceAllString(raw, "")
		raw = strings.TrimRight(raw, "\n \t")

		entry := LedgerEntry{
			Date:    date,
			RawText: raw,
		}
		if isPrice {
			entry.Type = EntryTypePrice
		} else {
			entry.Type = EntryTypeTransaction
		}

		if entry.RawText != "" {
			ledger.Entries = append(ledger.Entries, entry)
		}
	}

	return ledger
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
