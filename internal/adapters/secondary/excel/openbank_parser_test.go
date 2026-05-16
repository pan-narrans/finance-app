package excel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/a-perez/finance-app/internal/config"
	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenBankParser_Parse_ShouldReturnTransactions_WhenValidHtmlProvided(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	htmlPath := filepath.Join(tempDir, "test.xls")
	htmlContent := `<html><body><table>
                <tr>
                        <td>Valid</td><td>18/04/2026</td><td></td><td>19/04/2026</td><td></td><td>ALEJANDRO PEREZ *1234</td><td></td><td>50,00</td><td></td><td>250,00</td>
                </tr>
                <tr>
                        <td>Invalid (Too Short)</td><td>01/01/2026</td>
                </tr>
                <tr>
                        <td>Valid</td><td>16/04/2026</td><td></td><td>17/04/2026</td><td></td><td>Apple pay: COMPRA EN DIA, MADRID</td><td></td><td>-10,50</td><td></td><td>200,00</td>
                </tr>
        </table></body></html>`
	_ = os.WriteFile(htmlPath, []byte(htmlContent), 0644)

	mappingData := config.MappingData{
		Accounts: map[string]string{"DIA": "Expenses:Supermarket", "ALEJANDRO": "Income:Alex"},
		Cards:    map[string]string{"1234": "Alex"},
		Prefixes: []string{"Apple pay:"},
	}
	mappingService := domain.NewMappingService(mappingData, config.Config{})
	parser := NewOpenBankParser(mappingService)

	// Act
	transactions, err := parser.Parse(htmlPath)

	// Assert
	require.NoError(t, err)
	require.Len(t, transactions, 2)

	// First Transaction (18/04/2026 - Newest)
	assert.Equal(t, "2026-04-19", transactions[0].Date.Format("2006-01-02"))
	assert.Equal(t, 50.0, *transactions[0].Postings[0].Amount)
	assert.Equal(t, "Income:Alex", transactions[0].Postings[1].Account)
	assert.Equal(t, "Alex", transactions[0].Metadata.PayedBy)

	// Second Transaction (16/04/2026 - Oldest)
	assert.Equal(t, "2026-04-17", transactions[1].Date.Format("2006-01-02"))
	assert.Equal(t, "COMPRA EN DIA", transactions[1].Description)
	assert.Equal(t, -10.50, *transactions[1].Postings[0].Amount)
	assert.Equal(t, "Expenses:Supermarket", transactions[1].Postings[1].Account)
	assert.Equal(t, "Openbank", transactions[1].Metadata.Origin)
	assert.Equal(t, "72c8aa47", transactions[1].Metadata.ID)
}

func TestOpenBankParser_Parse_ShouldHandleIso8859Chars_WhenEncodedProperly(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	htmlPath := filepath.Join(tempDir, "test_iso.xls")
	// 0xF1 is 'ñ' in ISO-8859-1
	htmlContent := []byte("<html><body><table><tr><td></td><td>16/04/2026</td><td></td><td>17/04/2026</td><td></td><td>ESPA\xF1A</td><td></td><td>-10,50</td><td></td><td>200,00</td></tr></table></body></html>")
	_ = os.WriteFile(htmlPath, htmlContent, 0644)

	parser := NewOpenBankParser(domain.NewMappingService(config.MappingData{}, config.Config{}))

	// Act
	transactions, err := parser.Parse(htmlPath)

	// Assert
	require.NoError(t, err)
	require.Len(t, transactions, 1)
	assert.Equal(t, "ESPAñA", transactions[0].Description)
}

func TestOpenBankParser_Parse_ShouldReturnError_WhenFileNotFound(t *testing.T) {
	// Arrange
	parser := NewOpenBankParser(domain.NewMappingService(config.MappingData{}, config.Config{}))

	// Act
	transactions, err := parser.Parse("non-existent.xls")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, transactions)
}

func TestOpenBankParser_RowToTransaction_ShouldSkipRow_WhenDataIsInvalid(t *testing.T) {
	// Arrange
	parser := NewOpenBankParser(domain.NewMappingService(config.MappingData{}, config.Config{}))

	// Act & Assert
	t.Run(
		"Should fail when row is too short", func(t *testing.T) {
			tx, err := parser.rowToTransaction([]string{"too", "short"})
			assert.Error(t, err)
			assert.Nil(t, tx)
		},
	)

	t.Run(
		"Should fail when date is invalid", func(t *testing.T) {
			row := make([]string, 10)
			row[3] = "invalid-date"
			tx, err := parser.rowToTransaction(row)
			assert.Error(t, err)
			assert.Nil(t, tx)
		},
	)

	t.Run(
		"Should fail when amount is invalid", func(t *testing.T) {
			row := make([]string, 10)
			row[3] = "16/04/2026"
			row[7] = "invalid-amount"
			tx, err := parser.rowToTransaction(row)
			assert.Error(t, err)
			assert.Nil(t, tx)
		},
	)
}

func TestOpenBankParser_RowToTransaction_ShouldStripPrefixes(t *testing.T) {
	// Arrange
	mappingData := config.MappingData{
		Prefixes: []string{"Apple pay:", "Tarjeta:"},
	}
	mappingService := domain.NewMappingService(mappingData, config.Config{})
	parser := NewOpenBankParser(mappingService)

	tests := []struct {
		input    string
		expected string
	}{
		{"Apple pay: Starbucks", "Starbucks"},
		{"APPLE PAY: McDonald's", "McDonald's"},
		{"Tarjeta: Amazon", "Amazon"},
		{"Regular Purchase", "Regular Purchase"},
	}

	for _, tt := range tests {
		row := []string{"", "", "", "01/01/2026", "", tt.input, "", "10,00", "", "100,00"}
		tx, err := parser.rowToTransaction(row)
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, tx.Description)
	}
}

func TestOpenBankParser_RowToTransaction_ShouldApplyDescriptionMappings(t *testing.T) {
	// Arrange
	mappingData := config.MappingData{
		Descriptions: map[string]string{
			"SQ *BEN AND JERRY": "Ben & Jerry's",
			"AMZN MKTP":         "Amazon",
		},
	}
	mappingService := domain.NewMappingService(mappingData, config.Config{})
	parser := NewOpenBankParser(mappingService)

	tests := []struct {
		input    string
		expected string
	}{
		{"SQ *BEN AND JERRY MADRID", "Ben & Jerry's"},
		{"AMZN MKTP LUXEMBOURG", "Amazon"},
		{"Regular Purchase", "Regular Purchase"},
	}

	for _, tt := range tests {
		row := []string{"", "", "", "01/01/2026", "", tt.input, "", "10,00", "", "100,00"}
		tx, err := parser.rowToTransaction(row)
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, tx.Description)
	}
}

func TestParseSpanishAmount_ShouldHandleThousandsSeparator(t *testing.T) {
	// Act
	amount, err := ParseSpanishAmount("1.234,56")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1234.56, amount)
}
