package telegram

import (
	"testing"
	"time"

	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestUI_BuildDraftMessage_ShouldReturnFormattedTextAndMarkup(t *testing.T) {
	// Arrange
	ui := NewUI(52)
	tx := domain.Transaction{
		Date:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Description: "Test",
		Postings: []domain.Posting{
			{Account: "Expenses:Food", Amount: new(10.0), Currency: "EUR"},
			{Account: "Assets:Checking", Amount: nil},
		},
	}
	mappingProvider := domain.NewMappingService(domain.MappingData{})

	// Act
	msg, selector := ui.BuildDraftMessage(tx, mappingProvider)

	// Assert
	assert.Contains(t, msg, "Draft Transaction:")
	assert.Contains(t, msg, "2026/01/01")
	assert.Contains(t, msg, "Expenses:Food")
	assert.NotNil(t, selector)
	assert.Len(t, selector.InlineKeyboard, 3) // Confirm, Edit row, Discard
}

func TestUI_BuildDraftMessage_ShouldIncludeSuggestions_WhenAccountIsUnknown(t *testing.T) {
	// Arrange
	ui := NewUI(52)
	tx := domain.Transaction{
		Description: "Starbucks",
		Postings: []domain.Posting{
			{Account: "Expenses:Unknown", Amount: new(5.0), Currency: "EUR"},
		},
	}
	data := domain.MappingData{
		Accounts: map[string]string{"STARBUCKS": "Expenses:Food:Coffee"},
	}
	mappingProvider := domain.NewMappingService(data)

	// Act
	msg, selector := ui.BuildDraftMessage(tx, mappingProvider)

	// Assert
	assert.Contains(t, msg, "Unknown account. Suggestions:")
	assert.NotNil(t, selector)
	// 3 standard rows + 1 suggestion row
	assert.Len(t, selector.InlineKeyboard, 4)
}

func TestUI_BuildEditPrompt_ShouldReturnCorrectType(t *testing.T) {
	// Arrange
	ui := NewUI(52)

	// Act & Assert (Target)
	msg, _ := ui.BuildEditPrompt(false)
	assert.Contains(t, msg, "target account")

	// Act & Assert (Source)
	msg, _ = ui.BuildEditPrompt(true)
	assert.Contains(t, msg, "source account")
}

func TestUI_BuildSearchResults_ShouldIncludeAllOptions(t *testing.T) {
	// Arrange
	ui := NewUI(52)
	results := []string{"Acc1", "Acc2"}

	// Act
	msg, selector := ui.BuildSearchResults("query", results)

	// Assert
	assert.Contains(t, msg, "Search results for 'query':")
	assert.NotNil(t, selector)
	// 2 results + 1 create + 1 cancel = 4 rows
	assert.Len(t, selector.InlineKeyboard, 4)
}
