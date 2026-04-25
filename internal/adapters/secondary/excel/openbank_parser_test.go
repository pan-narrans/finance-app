package excel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenBankParser_NewOpenBankParser_ShouldLoadMappings_WhenValidFileProvided(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	mappingPath := filepath.Join(tempDir, "mappings.json")
	mappings := mappingsData{
		Accounts: map[string]string{"DIA": "Expenses:Supermarket", "ALEJANDRO": "Income:Alex"},
		Cards:    make(map[string]string),
	}
	mappingData, _ := json.Marshal(mappings)
	_ = os.WriteFile(mappingPath, mappingData, 0644)

	// Act
	parser := NewOpenBankParser(mappingPath)

	// Assert
	assert.Equal(t, "Expenses:Supermarket", parser.accountMappings["DIA"])
}

func TestOpenBankParser_NewOpenBankParser_ShouldHandleErrors(t *testing.T) {
	// Act & Assert
	t.Run("Should handle missing file", func(t *testing.T) {
		parser := NewOpenBankParser("non-existent.json")
		assert.NotNil(t, parser)
		assert.Empty(t, parser.accountMappings)
	})

	t.Run("Should handle invalid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		mappingPath := filepath.Join(tempDir, "invalid.json")
		_ = os.WriteFile(mappingPath, []byte("invalid-json"), 0644)
		parser := NewOpenBankParser(mappingPath)
		assert.Empty(t, parser.accountMappings)
	})
}

func TestOpenBankParser_Parse_ShouldReturnTransactions_WhenValidHtmlProvided(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	htmlPath := filepath.Join(tempDir, "test.xls")
	htmlContent := `<html><body><table>
		<tr>
			<td>Valid</td><td>16/04/2026</td><td></td><td>17/04/2026</td><td></td><td>COMPRA EN DIA</td><td></td><td>-10,50</td><td></td><td>200,00</td>
		</tr>
		<tr>
			<td>Invalid (Too Short)</td><td>01/01/2026</td>
		</tr>
		<tr>
			<td>Valid</td><td>18/04/2026</td><td></td><td>19/04/2026</td><td></td><td>ALEJANDRO PEREZ</td><td></td><td>50,00</td><td></td><td>250,00</td>
		</tr>
	</table></body></html>`
	_ = os.WriteFile(htmlPath, []byte(htmlContent), 0644)

	mappingPath := filepath.Join(tempDir, "mappings.json")
	mappings := mappingsData{
		Accounts: map[string]string{"DIA": "Expenses:Supermarket", "ALEJANDRO": "Income:Alex"},
		Cards:    make(map[string]string),
	}
	mappingData, _ := json.Marshal(mappings)
	_ = os.WriteFile(mappingPath, mappingData, 0644)

	parser := NewOpenBankParser(mappingPath)

	// Act
	transactions, err := parser.Parse(htmlPath)

	// Assert
	require.NoError(t, err)
	require.Len(t, transactions, 2)

	assert.Equal(t, "2026-04-17", transactions[0].Date.Format("2006-01-02"))
	assert.Equal(t, "COMPRA EN DIA", transactions[0].Description)
	assert.Equal(t, -10.50, *transactions[0].Postings[0].Amount)
	assert.Equal(t, "Expenses:Supermarket", transactions[0].Postings[1].Account)

	assert.Equal(t, "2026-04-19", transactions[1].Date.Format("2006-01-02"))
	assert.Equal(t, 50.0, *transactions[1].Postings[0].Amount)
	assert.Equal(t, "Income:Alex", transactions[1].Postings[1].Account)
}

func TestOpenBankParser_Parse_ShouldHandleIso8859Chars_WhenEncodedProperly(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	htmlPath := filepath.Join(tempDir, "test_iso.xls")
	// 0xF1 is 'ñ' in ISO-8859-1
	htmlContent := []byte("<html><body><table><tr><td></td><td>16/04/2026</td><td></td><td>17/04/2026</td><td></td><td>ESPA\xF1A</td><td></td><td>-10,50</td><td></td><td>200,00</td></tr></table></body></html>")
	_ = os.WriteFile(htmlPath, htmlContent, 0644)

	parser := NewOpenBankParser("")

	// Act
	transactions, err := parser.Parse(htmlPath)

	// Assert
	require.NoError(t, err)
	require.Len(t, transactions, 1)
	assert.Equal(t, "ESPAñA", transactions[0].Description)
}

func TestOpenBankParser_Parse_ShouldReturnError_WhenFileNotFound(t *testing.T) {
	// Arrange
	parser := NewOpenBankParser("")

	// Act
	transactions, err := parser.Parse("non-existent.xls")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, transactions)
}

func TestOpenBankParser_ResolveAccount_ShouldReturnUnknown_WhenNoMatchFound(t *testing.T) {
	// Arrange
	parser := NewOpenBankParser("")

	// Act & Assert
	assert.Equal(t, "Expenses:Unknown", parser.resolveAccount("Some unknown expense", -10.0))
	assert.Equal(t, "Income:Unknown", parser.resolveAccount("Some unknown income", 10.0))
}

func TestOpenBankParser_RowToTransaction_ShouldSkipRow_WhenDataIsInvalid(t *testing.T) {
	// Arrange
	parser := NewOpenBankParser("")

	// Act & Assert
	t.Run("Should fail when row is too short", func(t *testing.T) {
		tx, err := parser.rowToTransaction([]string{"too", "short"})
		assert.Error(t, err)
		assert.Nil(t, tx)
	})

	t.Run("Should fail when date is invalid", func(t *testing.T) {
		row := make([]string, 10)
		row[3] = "invalid-date"
		tx, err := parser.rowToTransaction(row)
		assert.Error(t, err)
		assert.Nil(t, tx)
	})

	t.Run("Should fail when amount is invalid", func(t *testing.T) {
		row := make([]string, 10)
		row[3] = "16/04/2026"
		row[7] = "invalid-amount"
		tx, err := parser.rowToTransaction(row)
		assert.Error(t, err)
		assert.Nil(t, tx)
	})
}

func TestParseSpanishAmount_ShouldHandleThousandsSeparator(t *testing.T) {
	// Act
	amount, err := parseSpanishAmount("1.234,56")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1234.56, amount)
}
