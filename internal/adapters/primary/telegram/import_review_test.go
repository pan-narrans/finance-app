package telegram

import (
	"testing"

	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestRefreshPendingMappingsUpdatesOnlyChangedRoleMatches(t *testing.T) {
	before := domain.NewMappingService(domain.MappingData{
		Accounts: map[string]string{"PAYEE": "Expenses:Old"},
	}, nil)
	after := domain.NewMappingService(domain.MappingData{
		AccountRoles: map[string]domain.AccountRoles{
			"PAYEE": {Target: "Expenses:New", Source: "Expenses:Old"},
		},
	}, nil)

	var queue []domain.Transaction
	for _, code := range []string{"first", "second", "third"} {
		queue = append(queue, domain.Transaction{
			Code:        code,
			MappingKeys: [2]string{"PAYEE Card", ""},
			Postings:    []domain.Posting{{Account: "Expenses:Old"}, {Account: "Assets:OpenBank"}},
			Metadata:    domain.Metadata{ID: code},
		})
	}
	queue = append(queue, domain.Transaction{
		Code:        "inflow",
		MappingKeys: [2]string{"", "PAYEE"},
		Postings:    []domain.Posting{{Account: "Assets:OpenBank"}, {Account: "Expenses:Old"}},
	})
	updated := refreshPendingMappings(queue, before, after)

	for i, original := range queue[:3] {
		assert.Equal(t, "Expenses:New", updated[i].Postings[0].Account)
		assert.Equal(t, "Expenses:Old", original.Postings[0].Account)
		assert.Equal(t, original.Postings[1], updated[i].Postings[1])
		assert.Equal(t, original.Code, updated[i].Code)
		assert.Equal(t, original.Metadata, updated[i].Metadata)
	}
	assert.Equal(t, queue[3], updated[3])
}
