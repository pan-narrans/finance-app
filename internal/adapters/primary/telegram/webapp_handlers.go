package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/a-perez/finance-app/internal/domain"
)

/*
handleAPIGetAccounts returns the list of all known accounts and root options for the search UI.
*/
func (a *TelegramAdapter) handleAPIGetAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mappings := a.configUseCase.Get().Mappings
	response := struct {
		Accounts []string `json:"accounts"`
		Roots    []string `json:"roots"`
	}{
		Accounts: mappings.GetAllAccounts(),
		Roots:    []string{"Expenses", "Assets", "Income", "Liabilities"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

/*
handleAPISelectAccount receives the selection from the TMA.
It validates the Telegram initData and updates the user's session.
*/
func (a *TelegramAdapter) handleAPISelectAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		InitData string `json:"initData"`
		Account  string `json:"account"`
		Type     string `json:"type"` // "source" or "target"
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// 1. Validate InitData
	initData, err := url.ParseQuery(payload.InitData)
	if err != nil || !a.validateInitData(payload.InitData) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Extract User ID
	userJSON := initData.Get("user")
	var user struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		http.Error(w, "Invalid user data", http.StatusBadRequest)
		return
	}

	// 3. Update Session
	_, exists := a.sessionManager.Get(user.ID)
	if !exists {
		http.Error(w, "Session expired", http.StatusNotFound)
		return
	}

	log.Printf("Mini App selection: User %d selected %s as %s", user.ID, payload.Account, payload.Type)

	formattedAccount := domain.FormatAccountPath(payload.Account)

	a.sessionManager.Update(user.ID, func(s *UserSession) {
		if payload.Type == "source" {
			s.Draft.Postings[1].Account = formattedAccount
			s.SourceOverridden = true
		} else {
			s.Draft.Postings[0].Account = formattedAccount
			s.TargetOverridden = true
		}
	})

	// 4. Update the bot message asynchronously
	if err := a.refreshDraftMessage(user.ID); err != nil {
		log.Printf("Failed to refresh message for user %d: %v", user.ID, err)
		// We don't return error to user here as the selection itself succeeded
	}

	w.WriteHeader(http.StatusOK)
}

/*
validateInitData verifies that the data received from the Mini App is authentic.
Implements the HMAC-SHA256 signature check required by Telegram.
*/
func (a *TelegramAdapter) validateInitData(initDataRaw string) bool {
	params, _ := url.ParseQuery(initDataRaw)
	hash := params.Get("hash")
	params.Del("hash")

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var dataCheckString strings.Builder
	for i, k := range keys {
		if i > 0 {
			dataCheckString.WriteString("\n")
		}
		dataCheckString.WriteString(fmt.Sprintf("%s=%s", k, params.Get(k)))
	}

	// 1. Create secret key: HMAC-SHA256("WebAppData", botToken)
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(a.botToken))
	secretKey := mac.Sum(nil)

	// 2. Create signature: HMAC-SHA256(dataCheckString, secretKey)
	mac = hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString.String()))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	return expectedHash == hash
}
