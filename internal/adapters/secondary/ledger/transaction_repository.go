package ledger

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

// Ensure TransactionFileRepository implements ports.TransactionRepository at compile time.
var _ ports.TransactionRepository = (*TransactionFileRepository)(nil)

type entryType int

const (
	typeTransaction entryType = iota
	typePrice
)

type ledgerEntry struct {
	Date    time.Time
	RawText string
	Type    entryType
}

type ledgerFile struct {
	Prologue string
	Entries  []ledgerEntry
	Epilogue string
}

/*
TransactionFileRepository implements ports.TransactionRepository using a plain-text file.

It uses regex-based parsing to interact with transactions in the Ledger CLI format.

Methods:
  - Create: Appends a new transaction to the end of the ledger file.
  - FindByCode: Searches the file for a transaction with the given unique code.
  - Update: Replaces an existing transaction block in the file with a new formatted version.
  - Delete: Removes a transaction block from the file by its unique code.
*/
type TransactionFileRepository struct {
	FilePath      string
	configUseCase ports.ConfigurationUseCase
	formatter     ports.TransactionFormatter
	mu            sync.Mutex
}

// NewTransactionFileRepository creates a new instance of TransactionFileRepository.
func NewTransactionFileRepository(
	filePath string,
	configUC ports.ConfigurationUseCase,
	formatter ports.TransactionFormatter,
) *TransactionFileRepository {
	return &TransactionFileRepository{
		FilePath:      filePath,
		configUseCase: configUC,
		formatter:     formatter,
	}
}

// Create writes a transaction to the end of the ledger file.
func (fileRepository *TransactionFileRepository) Create(transaction domain.Transaction) error {
	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	lf, err := fileRepository.readLedgerFile()
	if err != nil {
		return err
	}

	// Check for duplicates
	codeMarker := fmt.Sprintf("(%s)", transaction.Code)
	for _, entry := range lf.Entries {
		if entry.Type == typeTransaction && strings.Contains(entry.RawText, codeMarker) {
			return domain.NewDomainError("Transaction", "Code", "transaction already exists")
		}
	}

	alignment := fileRepository.configUseCase.Get().Settings.LedgerAlignment
	newRaw := fileRepository.formatter.FormatTransaction(transaction, alignment)
	lf.Entries = append(lf.Entries, ledgerEntry{Date: transaction.Date, RawText: newRaw, Type: typeTransaction})

	return fileRepository.writeLedgerFile(lf)
}

// FindByCode searches the file using a regex to find a transaction with the given code.
func (fileRepository *TransactionFileRepository) FindByCode(code string) (*domain.Transaction, error) {
	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	data, err := os.ReadFile(fileRepository.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	regex := fileRepository.transactionRegex(code)

	if regex.Match(data) {
		return &domain.Transaction{Code: code}, nil
	}

	return nil, nil
}

/*
Update replaces an existing transaction block with a new formatted version.
It returns a domain.DomainError if the transaction code is not found in the file.
*/
func (fileRepository *TransactionFileRepository) Update(transaction domain.Transaction) error {
	if transaction.Code == "" {
		return domain.NewDomainError("Transaction", "Code", "transaction must have a code to be updated")
	}

	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	lf, err := fileRepository.readLedgerFile()
	if err != nil {
		return err
	}

	found := false
	alignment := fileRepository.configUseCase.Get().Settings.LedgerAlignment
	codeMarker := fmt.Sprintf("(%s)", transaction.Code)

	for i, entry := range lf.Entries {
		if entry.Type == typeTransaction && strings.Contains(entry.RawText, codeMarker) {
			lf.Entries[i].Date = transaction.Date
			lf.Entries[i].RawText = fileRepository.formatter.FormatTransaction(transaction, alignment)
			found = true
			break
		}
	}

	if !found {
		return domain.NewDomainError("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", transaction.Code))
	}

	return fileRepository.writeLedgerFile(lf)
}

func (fileRepository *TransactionFileRepository) Delete(code string) error {
	if code == "" {
		return domain.NewDomainError("Transaction", "Code", "code must be provided to delete a transaction")
	}

	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	lf, err := fileRepository.readLedgerFile()
	if err != nil {
		if os.IsNotExist(err) {
			return domain.NewDomainError("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", code))
		}
		return err
	}

	newEntries := make([]ledgerEntry, 0, len(lf.Entries))
	codeMarker := fmt.Sprintf("(%s)", code)
	found := false

	for _, entry := range lf.Entries {
		if entry.Type == typeTransaction && strings.Contains(entry.RawText, codeMarker) {
			found = true
			continue
		}
		newEntries = append(newEntries, entry)
	}

	if !found {
		return domain.NewDomainError("Transaction", "Code", fmt.Sprintf("transaction with code %q not found", code))
	}

	lf.Entries = newEntries
	return fileRepository.writeLedgerFile(lf)
}

/*
atomicWrite writes data to a temporary file and renames it to the target file path.
This ensures the write is atomic and prevents file corruption on crash/failure.
*/
func (fileRepository *TransactionFileRepository) atomicWrite(data []byte) error {
	tmpPath := fileRepository.FilePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, fileRepository.FilePath)
}

// GetAccounts retrieves the list of accounts from the ledger file using the ledger CLI.
func (fileRepository *TransactionFileRepository) GetAccounts() ([]string, error) {
	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	// Check if file exists first to avoid unnecessary CLI errors
	if _, err := os.Stat(fileRepository.FilePath); os.IsNotExist(err) {
		return nil, nil
	}

	cmd := exec.Command("ledger", "-f", fileRepository.FilePath, "accounts")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ledger accounts failed: %w (output: %q)", err, string(output))
	}

	lines := strings.Split(string(output), "\n")
	var accounts []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			accounts = append(accounts, trimmed)
		}
	}

	return accounts, nil
}

// GetBalanceReport executes the ledger balance command for the given period and filter.
func (fileRepository *TransactionFileRepository) GetBalanceReport(period string, filter string) (string, error) {
	fileRepository.mu.Lock()
	defer fileRepository.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(fileRepository.FilePath); os.IsNotExist(err) {
		return "", nil
	}

	args := []string{"-f", fileRepository.FilePath, "bal"}
	if period != "" {
		args = append(args, "--period", period)
	}
	if filter != "" {
		args = append(args, filter)
	}

	cmd := exec.Command("ledger", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Ledger returns error 1 if no matches found for filter
		if len(output) == 0 {
			return "", nil
		}
		return "", fmt.Errorf("ledger balance failed: %w (output: %q)", err, string(output))
	}

	return string(output), nil
}

/*
transactionRegex compiles a regular expression to match a transaction block
by its unique code. It looks for the DATE followed by the (CODE) marker.
*/
func (fileRepository *TransactionFileRepository) transactionRegex(code string) *regexp.Regexp {
	// (?m) enables multi-line mode.
	// We match from the date line to the next blank line or end of file.
	pattern := fmt.Sprintf(`(?m)^\d{4}[\/-]\d{2}[\/-]\d{2}.*\(%s\)(?:.*\n)*?(\r?\n|$)`, regexp.QuoteMeta(code))
	return regexp.MustCompile(pattern)
}

func (fileRepository *TransactionFileRepository) readLedgerFile() (ledgerFile, error) {
	data, err := os.ReadFile(fileRepository.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ledgerFile{}, nil
		}
		return ledgerFile{}, err
	}

	content := string(data)
	
	// 1. Strip ALL month headers from the file to prevent duplication
	lines := strings.Split(content, "\n")
	var cleanedLines []string
	headerLineRegex := regexp.MustCompile(`^;- [A-Z ]+ -$`)
	
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if headerLineRegex.MatchString(line) {
			if i > 0 && strings.HasPrefix(strings.TrimSpace(lines[i-1]), ";---") &&
				i < len(lines)-1 && strings.HasPrefix(strings.TrimSpace(lines[i+1]), ";---") {
				if len(cleanedLines) > 0 && strings.HasPrefix(strings.TrimSpace(cleanedLines[len(cleanedLines)-1]), ";---") {
					cleanedLines = cleanedLines[:len(cleanedLines)-1]
				}
				i++ // Skip next dash line
				continue
			}
		}
		cleanedLines = append(cleanedLines, lines[i])
	}
	content = strings.Join(cleanedLines, "\n")

	// 2. Find entry boundaries (Transactions OR Prices)
	// (P\s+)? captures the price prefix if present
	entryStartRegex := regexp.MustCompile(`(?m)^(P\s+)?(\d{4}[\/-]\d{2}[\/-]\d{2})`)
	matches := entryStartRegex.FindAllStringSubmatchIndex(content, -1)

	if len(matches) == 0 {
		return ledgerFile{Prologue: content}, nil
	}

	lf := ledgerFile{
		Prologue: strings.TrimRight(content[:matches[0][0]], "\n \t") + "\n\n",
	}

	for i, match := range matches {
		isPrice := match[2] != -1 && match[3] != -1 // P\s+ group matched
		dateStr := content[match[4]:match[5]]
		dateStr = strings.ReplaceAll(dateStr, "-", "/")
		date, _ := time.Parse("2006/01/02", dateStr)

		start := match[0]
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}

		raw := content[start:end]
		entry := ledgerEntry{Date: date}

		if isPrice {
			entry.Type = typePrice
			// Price updates are single lines. 
			// We take the first line and treat the rest as trailing noise/epilogue potential
			pLines := strings.Split(raw, "\n")
			entry.RawText = strings.TrimSpace(pLines[0])
			if i == len(matches)-1 {
				lf.Epilogue = strings.Join(pLines[1:], "\n")
			}
		} else {
			entry.Type = typeTransaction
			// Transaction handling logic (last indented line)
			txLines := strings.Split(raw, "\n")
			lastTxLine := 0
			for j := len(txLines) - 1; j >= 0; j-- {
				line := txLines[j]
				if strings.TrimSpace(line) == "" {
					continue
				}
				if j == 0 || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
					lastTxLine = j
					break
				}
			}
			entry.RawText = strings.TrimSpace(strings.Join(txLines[:lastTxLine+1], "\n"))
			if i == len(matches)-1 {
				lf.Epilogue = strings.Join(txLines[lastTxLine+1:], "\n")
			}
		}

		if entry.RawText != "" {
			lf.Entries = append(lf.Entries, entry)
		}
	}

	return lf, nil
}

func (fileRepository *TransactionFileRepository) writeLedgerFile(lf ledgerFile) error {
	var sb strings.Builder
	
	if strings.TrimSpace(lf.Prologue) != "" {
		sb.WriteString(strings.TrimRight(lf.Prologue, "\n"))
		sb.WriteString("\n\n")
	}

	if len(lf.Entries) > 0 {
		// Stable sort by date. 
		// If same date, preserve relative order (StableSortFunc).
		slices.SortStableFunc(lf.Entries, func(a, b ledgerEntry) int {
			if a.Date.Before(b.Date) { return -1 }
			if a.Date.After(b.Date) { return 1 }
			return 0
		})

		var lastMonth time.Month
		var lastYear int

		for _, entry := range lf.Entries {
			month := entry.Date.Month()
			year := entry.Date.Year()

			if month != lastMonth || year != lastYear {
				monthName := strings.ToUpper(month.String())
				sb.WriteString(";--------\n")
				sb.WriteString(fmt.Sprintf(";- %s -\n", monthName))
				sb.WriteString(";--------\n\n")

				lastMonth = month
				lastYear = year
			}

			sb.WriteString(entry.RawText)
			sb.WriteString("\n\n")
		}
	}

	if strings.TrimSpace(lf.Epilogue) != "" {
		sb.WriteString(strings.TrimLeft(lf.Epilogue, "\n"))
	}

	result := sb.String()
	result = strings.TrimRight(result, "\n")
	if result != "" {
		result += "\n"
	}

	return fileRepository.atomicWrite([]byte(result))
}
