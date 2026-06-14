package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/a-perez/finance-app/internal/adapters/primary/telegram"
	"github.com/stretchr/testify/assert"
	"gopkg.in/telebot.v3"
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
	}, 5*time.Second, 100*time.Millisecond, "Session should be created")

	assert.Equal(t, "Lunch", sess.Draft.Description)

	env.sendCallback(telegram.CallbackConfirm)

	// Wait for ledger update
	var content []byte
	assert.Eventually(t, func() bool {
		var err error
		content, err = os.ReadFile(env.ledgerPath)
		return err == nil && len(content) > 0
	}, 5*time.Second, 100*time.Millisecond, "Ledger file should be created and populated")

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

func TestE2E_Transaction_ShouldOverwritePendingSession_WhenNewTextArrives(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	// Act 1: Send first transaction
	env.sendText("10 Coffee")
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.Draft.Description == "Coffee"
	}, 5*time.Second, 100*time.Millisecond, "Session should be created for Coffee")

	// Act 2: Send second transaction without confirming first
	env.sendText("20 Lunch")
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.Draft.Description == "Lunch"
	}, 5*time.Second, 100*time.Millisecond, "Session should be overwritten by Lunch")

	env.sendCallback(telegram.CallbackConfirm)

	// Assert
	var content []byte
	assert.Eventually(t, func() bool {
		var err error
		content, err = os.ReadFile(env.ledgerPath)
		return err == nil && strings.Contains(string(content), "Lunch") && !strings.Contains(string(content), "Coffee")
	}, 5*time.Second, 100*time.Millisecond, "Only the last transaction should be in ledger")
}

func TestE2E_Transaction_ShouldHandleEmojiAndSpecialChars(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	// Act
	env.sendText("15 🍕 Dinner @Home & \"Friends\"")
	assert.Eventually(t, func() bool {
		_, ok := env.adapter.SessionManager().Get(env.userID)
		return ok
	}, 5*time.Second, 100*time.Millisecond, "Session should be created")

	env.sendCallback(telegram.CallbackConfirm)

	// Assert
	var content []byte
	assert.Eventually(t, func() bool {
		var err error
		content, err = os.ReadFile(env.ledgerPath)
		return err == nil && len(content) > 0
	}, 5*time.Second, 100*time.Millisecond, "Ledger should be populated")

	ledgerText := string(content)
	assert.Contains(t, ledgerText, "🍕 Dinner @Home & \"Friends\"")
}

func TestE2E_Transaction_ShouldLearnMapping_WhenManualAccountOverride(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	// 1. Send transaction that results in Unknown account
	env.sendText("50 UnusualExpense")
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && strings.HasSuffix(s.Draft.Postings[0].Account, ":Unknown")
	}, 5*time.Second, 100*time.Millisecond, "Should have Unknown account initially")

	// 2. Override account (simulate selection from search)
	env.sendCallback(telegram.CallbackSelectAcc, "Expenses:Leisure")
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.Draft.Postings[0].Account == "Expenses:Leisure" && s.TargetOverridden
	}, 5*time.Second, 100*time.Millisecond, "Account should be overridden to Leisure")

	env.sendCallback(telegram.CallbackConfirm)

	// 3. Wait for confirmation and ledger update
	assert.Eventually(t, func() bool {
		content, _ := os.ReadFile(env.ledgerPath)
		return strings.Contains(string(content), "UnusualExpense")
	}, 5*time.Second, 100*time.Millisecond, "First transaction should be saved")

	// 4. Send another transaction with same description
	env.sendText("30 UnusualExpense")
	
	// Assert: It should now suggest Expenses:Leisure
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && s.Draft.Postings[0].Account == "Expenses:Leisure"
	}, 5*time.Second, 100*time.Millisecond, "Second transaction should automatically pick learned mapping")
}

func TestE2E_Transaction_ShouldIgnoreUnauthorizedUser(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)
	unauthorizedUserID := int64(99999)

	// Act
	env.adapter.Bot().ProcessUpdate(telebot.Update{
		Message: &telebot.Message{
			ID:     1,
			Text:   "10 Coffee",
			Sender: &telebot.User{ID: unauthorizedUserID},
			Chat:   &telebot.Chat{ID: unauthorizedUserID, Type: telebot.ChatPrivate},
		},
	})

	// Assert
	time.Sleep(200 * time.Millisecond) // Wait for middleware
	_, ok := env.adapter.SessionManager().Get(unauthorizedUserID)
	assert.False(t, ok, "Session should NOT be created for unauthorized user")
}

func TestE2E_Transaction_ShouldHandlePersistenceFailure(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)
	
	// Make ledger file read-only to simulate failure
	_ = os.WriteFile(env.ledgerPath, []byte(""), 0444)
	defer os.Chmod(env.ledgerPath, 0644) // Restore for cleanup

	// Act
	env.sendText("10 Coffee")
	assert.Eventually(t, func() bool {
		_, ok := env.adapter.SessionManager().Get(env.userID)
		return ok
	}, 5*time.Second, 100*time.Millisecond)

	env.sendCallback(telegram.CallbackConfirm)

	// Assert
	// We expect the ledger to remain empty (or unchanged)
	time.Sleep(500 * time.Millisecond)
	content, _ := os.ReadFile(env.ledgerPath)
	assert.Empty(t, strings.TrimSpace(string(content)), "Ledger should be empty due to write failure")
}

func TestE2E_Transaction_ShouldInterruptAccountCreation_WhenNewTransactionComes(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	// 1. Start manual transaction
	env.sendText("50 Unknown")
	assert.Eventually(t, func() bool {
		_, ok := env.adapter.SessionManager().Get(env.userID)
		return ok
	}, 5*time.Second, 100*time.Millisecond)

	// 2. Start account creation flow
	env.sendCallback(telegram.CallbackCreateAcc)
	assert.Eventually(t, func() bool {
		s, _ := env.adapter.SessionManager().Get(env.userID)
		return s.State == telegram.StateCreatingAccountParent
	}, 5*time.Second, 100*time.Millisecond)

	// 3. Select a parent to move to StateCreatingAccountChild
	env.sendCallback(telegram.CallbackSelectParent, "Expenses")
	assert.Eventually(t, func() bool {
		s, _ := env.adapter.SessionManager().Get(env.userID)
		return s.State == telegram.StateCreatingAccountChild
	}, 5*time.Second, 100*time.Millisecond)

	time.Sleep(500 * time.Millisecond)

	// 4. Suddenly send a new transaction text
	env.sendText("12.34 UniqueTransaction")
	
	// Assert: State should be reset to None, and draft should be UniqueTransaction
	assert.Eventually(t, func() bool {
		s, _ := env.adapter.SessionManager().Get(env.userID)
		return s.State == telegram.StateNone && s.Draft.Description == "UniqueTransaction"
	}, 5*time.Second, 100*time.Millisecond)
}

func TestE2E_Transaction_ShouldHandleBillionaireAmounts(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)
	billion := "1000000000.99"

	// Act
	env.sendText(billion + " StartupInvestment")
	assert.Eventually(t, func() bool {
		_, ok := env.adapter.SessionManager().Get(env.userID)
		return ok
	}, 5*time.Second, 100*time.Millisecond)

	env.sendCallback(telegram.CallbackConfirm)

	// Assert
	var content []byte
	assert.Eventually(t, func() bool {
		var err error
		content, err = os.ReadFile(env.ledgerPath)
		return err == nil && len(content) > 0
	}, 5*time.Second, 100*time.Millisecond)

	ledgerText := string(content)
	// Check for raw amount or formatted with commas depending on alignment implementation
	// The core requirement is that it parses and persists without overflow.
	assert.Contains(t, ledgerText, "1000000000.99") 
}

func TestE2E_Transaction_ShouldHandleRapidFireCommands(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	// Act: Send many commands in parallel/rapid sequence
	for i := 0; i < 10; i++ {
		go func(idx int) {
			env.sendText(fmt.Sprintf("%d Lunch-%d", idx+10, idx))
		}(i)
	}

	// Assert: It should eventually settle on ONE of them without crashing
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(env.userID)
		return ok && strings.HasPrefix(s.Draft.Description, "Lunch-")
	}, 5*time.Second, 100*time.Millisecond)

	env.sendCallback(telegram.CallbackConfirm)

	// Ledger should have exactly one transaction if confirm was only sent once
	assert.Eventually(t, func() bool {
		content, _ := os.ReadFile(env.ledgerPath)
		return len(content) > 0
	}, 5*time.Second, 100*time.Millisecond)
}










