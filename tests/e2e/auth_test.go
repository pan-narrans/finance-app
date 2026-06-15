package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/telebot.v3"
)

func TestE2E_Auth_ShouldUpdateAuthorizedUsers_WhenConfigReloads(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)
	newUserID := int64(67890)
	configPath := filepath.Join(env.tmpDir, "config.json")

	// 1. Verify new user is initially unauthorized
	env.adapter.Bot().ProcessUpdate(telebot.Update{
		Message: &telebot.Message{
			ID:     1,
			Text:   "10 Coffee",
			Sender: &telebot.User{ID: newUserID},
			Chat:   &telebot.Chat{ID: newUserID, Type: telebot.ChatPrivate},
		},
	})

	time.Sleep(200 * time.Millisecond)
	_, ok := env.adapter.SessionManager().Get(newUserID)
	assert.False(t, ok, "Session should NOT be created for unauthorized user")

	// 2. Update config file to include the new user
	configJSON := fmt.Sprintf(`{
		"telegram_user_ids": [12345, %d],
		"default_currency": "EUR"
	}`, newUserID)

	_ = os.WriteFile(configPath, []byte(configJSON), 0644)

	// Trigger reload (simulating fsnotify which might be slow in tests, so we call it explicitly to be sure)
	err := env.configManager.Reload()
	assert.NoError(t, err)

	// 3. Try again with the same user
	env.adapter.Bot().ProcessUpdate(telebot.Update{
		Message: &telebot.Message{
			ID:     2,
			Text:   "20 Lunch",
			Sender: &telebot.User{ID: newUserID},
			Chat:   &telebot.Chat{ID: newUserID, Type: telebot.ChatPrivate},
		},
	})

	// Assert
	assert.Eventually(t, func() bool {
		s, ok := env.adapter.SessionManager().Get(newUserID)
		return ok && s.Draft.Description == "Lunch"
	}, 5*time.Second, 100*time.Millisecond, "Session SHOULD be created after hot-reload")
}
