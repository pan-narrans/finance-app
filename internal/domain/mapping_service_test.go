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
	svc := NewMappingService(data, nil)

	// Act
	account, found := svc.ResolveAccount("AMAZON MARKETPLACE LUX")

	// Assert
	assert.True(t, found)
	assert.Equal(t, "Expenses:Shopping", account, "Should match longest keyword first")
}

func TestMappingService_ResolveAccount_ShouldReturnFalse_WhenNoMatchFound(t *testing.T) {
	// Arrange
	svc := NewMappingService(MappingData{}, nil)

	// Act
	account, found := svc.ResolveAccount("Some unknown")

	// Assert
	assert.False(t, found)
	assert.Empty(t, account)
}

func TestMappingService_IsIncomeAccount(t *testing.T) {
	svc := NewMappingService(MappingData{}, nil)

	assert.True(t, svc.IsIncomeAccount("Income:Salary"))
	assert.True(t, svc.IsIncomeAccount("Equity:Income:Bonus"))
	assert.False(t, svc.IsIncomeAccount("Expenses:Food"))
	assert.False(t, svc.IsIncomeAccount("Assets:Cash"))
}

func TestMappingService_CleanDescription_ShouldStripPrefixes(t *testing.T) {
	// Arrange
	data := MappingData{
		Prefixes: []string{"Apple pay:", "Tarjeta:"},
	}
	svc := NewMappingService(data, nil)

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
	svc := NewMappingService(data, nil)

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
	svc := NewMappingService(data, nil)

	// Act & Assert
	assert.Equal(t, "Alex", svc.ResolvePayer("Purchase with card *1234"))
	assert.Equal(t, "Maria", svc.ResolvePayer("Transfer to *5678"))
	assert.Equal(t, "", svc.ResolvePayer("No card info here"))
}

func TestMappingService_ResolveSource_ShouldReturnCorrectAccount(t *testing.T) {
	// Arrange
	data := MappingData{
		Accounts: map[string]string{
			"ALEX":     "Income:Alex",
			"PILAR":    "Income:Pilar",
			"EFECTIVO": "Assets:Cash",
		},
	}
	svc := NewMappingService(data, nil)

	// Act & Assert
	acc, found := svc.ResolveSource("Alex")
	assert.True(t, found)
	assert.Equal(t, "Income:Alex", acc)

	acc, found = svc.ResolveSource("efectivo")
	assert.True(t, found)
	assert.Equal(t, "Assets:Cash", acc)

	acc, found = svc.ResolveSource("unknown")
	assert.False(t, found)
	assert.Equal(t, "", acc)

	acc, found = svc.ResolveSource("")
	assert.False(t, found)
	assert.Equal(t, "", acc)
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

func TestMappingService_GetAllAccounts_ShouldReturnDeduplicatedAndSortedList(t *testing.T) {
	// Arrange
	data := MappingData{
		Accounts: map[string]string{
			"key1": "Expenses:Food",
			"key2": "Expenses:Food",
			"key3": "Expenses:Transport",
			"key4": "Assets:Cash",
		},
	}
	svc := NewMappingService(data, nil)

	// Act
	results := svc.accounts

	// Assert
	expected := []string{"Assets:Cash", "Expenses:Food", "Expenses:Transport"}
	assert.Equal(t, expected, results, "Should deduplicate and sort accounts")
}

func TestMappingService_SearchAccounts_ShouldReturnRankedResults(t *testing.T) {
	// Arrange
	data := MappingData{
		Accounts: map[string]string{
			"key1": "Expenses:Food:Restaurante",
			"key2": "Expenses:Food:Supermercado",
			"key3": "Expenses:Transporte:Combustible",
			"key4": "Expenses:Transporte:Parking",
			"key5": "Income:Salary",
		},
	}
	svc := NewMappingService(data, nil)

	tests := []struct {
		name     string
		query    string
		limit    int
		expected []string
	}{
		{
			name:     "Exact match",
			query:    "Income:Salary",
			limit:    5,
			expected: []string{"Income:Salary"},
		},
		{
			name:     "Substring match",
			query:    "Food",
			limit:    5,
			expected: []string{"Expenses:Food:Restaurante", "Expenses:Food:Supermercado"},
		},
		{
			name:     "Case insensitive",
			query:    "transport",
			limit:    5,
			expected: []string{"Expenses:Transporte:Combustible", "Expenses:Transporte:Parking"},
		},
		{
			name:     "Tokenized match",
			query:    "Exp Park",
			limit:    5,
			expected: []string{"Expenses:Transporte:Parking"},
		},
		{
			name:     "Ranking: prefix first",
			query:    "Exp",
			limit:    2,
			expected: []string{"Expenses:Food:Restaurante", "Expenses:Food:Supermercado"},
		},
		{
			name:     "Limit results",
			query:    "Expenses",
			limit:    1,
			expected: []string{"Expenses:Food:Restaurante"},
		},
		{
			name:     "No match",
			query:    "UnknownAccount",
			limit:    5,
			expected: []string{},
		},
		{
			name:     "Match via mapping key",
			query:    "Salary",
			limit:    5,
			expected: []string{"Income:Salary"},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				results := svc.SearchAccounts(tt.query, tt.limit)
				assert.Equal(t, tt.expected, results)
			},
		)
	}
}
