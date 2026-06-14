package e2e

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/a-perez/finance-app/internal/adapters/primary/telegram"
	"github.com/stretchr/testify/assert"
)

func TestE2E_SingleTransaction_ShouldUpdateLedger_WhenHappyPath(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	// Act
	t.Logf("Sending text from User ID: %d", env.userID)
	env.sendText("12.50 Lunch")
	
	// Wait for session to be created (async handler)
	var sess telegram.UserSession
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		if ok {
			sess = s
		}
		return ok
	}, 2*time.Second, 100*time.Millisecond, "Session should be created")

	assert.Equal(t, "Lunch", sess.Draft.Description)

	env.sendCallback(telegram.CallbackConfirm)

	// Wait for ledger update
	var content []byte
	assert.Eventually(t, func() bool {
		var err error
		content, err = os.ReadFile(env.ledgerPath)
		return err == nil && len(content) > 0
	}, 2*time.Second, 100*time.Millisecond, "Ledger file should be created and populated")

	assert.Contains(t, string(content), "12.50")
	assert.Contains(t, string(content), "Lunch")
}

func TestE2E_SingleTransaction_ShouldReturnError_WhenMalformedInput(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	// Act
	env.sendText("just-description-no-amount")

	// Assert
	content, _ := os.ReadFile(env.ledgerPath)
	assert.Empty(t, strings.TrimSpace(string(content)), "Ledger should be empty after malformed input")
}
