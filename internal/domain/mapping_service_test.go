package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMappingService_ResolveAccount_ShouldPreferLongestMatch(t *testing.T) {
	// Arrange
	data := MappingData{
		Accounts: map[string]string{
			"AMAZON":             "Expenses:General",
			"AMAZON MARKETPLACE": "Expenses:Shopping",
		},
	}
	svc := NewMappingService(data)

	// Act
	account := svc.ResolveAccount("AMAZON MARKETPLACE LUX", -50.0)

	// Assert
	assert.Equal(t, "Expenses:Shopping", account, "Should match longest keyword first")
}

func TestMappingService_ResolveAccount_ShouldReturnUnknown_WhenNoMatchFound(t *testing.T) {
	// Arrange
	svc := NewMappingService(MappingData{})

	// Act & Assert
	assert.Equal(t, "Expenses:Unknown", svc.ResolveAccount("Some unknown expense", -10.0))
	assert.Equal(t, "Income:Unknown", svc.ResolveAccount("Some unknown income", 10.0))
}

func TestMappingService_CleanDescription_ShouldStripPrefixes(t *testing.T) {
	// Arrange
	data := MappingData{
		Prefixes: []string{"Apple pay:", "Tarjeta:"},
	}
	svc := NewMappingService(data)

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
		result := svc.CleanDescription(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMappingService_CleanDescription_ShouldApplyDescriptionMappings(t *testing.T) {
	// Arrange
	data := MappingData{
		Descriptions: map[string]string{
			"SQ *BEN AND JERRY": "Ben & Jerry's",
			"AMZN MKTP":         "Amazon",
		},
	}
	svc := NewMappingService(data)

	tests := []struct {
		input    string
		expected string
	}{
		{"SQ *BEN AND JERRY MADRID", "Ben & Jerry's"},
		{"AMZN MKTP LUXEMBOURG", "Amazon"},
		{"Regular Purchase", "Regular Purchase"},
	}

	for _, tt := range tests {
		result := svc.CleanDescription(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMappingService_ResolvePayer_ShouldReturnCorrectOwner(t *testing.T) {
	// Arrange
	data := MappingData{
		Cards: map[string]string{"*1234": "Alex", "*5678": "Maria"},
	}
	svc := NewMappingService(data)

	// Act & Assert
	assert.Equal(t, "Alex", svc.ResolvePayer("Purchase with card *1234"))
	assert.Equal(t, "Maria", svc.ResolvePayer("Transfer to *5678"))
	assert.Equal(t, "", svc.ResolvePayer("No card info here"))
}

func TestSortKeywords_ShouldSortByLengthDescending(t *testing.T) {
	// Arrange
	m := map[string]string{
		"A":   "val",
		"ABC": "val",
		"AB":  "val",
	}

	// Act
	result := sortKeywords(m)

	// Assert
	assert.Equal(t, []string{"ABC", "AB", "A"}, result)
}
