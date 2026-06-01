package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/a-perez/finance-app/internal/adapters/primary/telegram/webapp"
	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

/*
MessageRefresher defines the contract for updating Telegram chat messages.
This allows the WebAppServer to trigger UI updates without being tightly coupled to the bot.
*/
type MessageRefresher interface {
	RefreshDraftMessage(userID int64) error
}

/*
WebAppServer handles HTTP requests from the Telegram Mini App.
It serves static assets and provides the necessary API endpoints.
*/
type WebAppServer struct {
	port           int
	botToken       string
	configUseCase  ports.ConfigurationUseCase
	sessionManager *SessionManager
	refresher      MessageRefresher
}

/*
NewWebAppServer creates a new instance of WebAppServer.
*/
func NewWebAppServer(port int, token string, configUC ports.ConfigurationUseCase, sessionManager *SessionManager, refresher MessageRefresher) *WebAppServer {
	return &WebAppServer{
		port:           port,
		botToken:       token,
		configUseCase:  configUC,
		sessionManager: sessionManager,
		refresher:      refresher,
	}
}

/*
Start launches the HTTP server in a blocking manner.
*/
func (s *WebAppServer) Start() error {
	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("/api/accounts", s.handleGetAccounts)
	mux.HandleFunc("/api/select", s.handleSelectAccount)

	// Static Assets (Embedded)
	staticFS, err := fs.Sub(webapp.Assets, "dist")
	if err != nil {
		return fmt.Errorf("failed to access embedded assets: %w", err)
	}
	fsServer := http.FileServer(http.FS(staticFS))

	// Middleware for logging
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HTTP] %s %s", r.Method, r.URL.Path)
		mux.ServeHTTP(w, r)
	})

	mux.Handle("/", fsServer)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("WebApp server listening on %s (embedded assets)", addr)
	return http.ListenAndServe(addr, handler)
}

func (s *WebAppServer) handleGetAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	appConfig := s.configUseCase.Get()
	response := struct {
		Accounts []string `json:"accounts"`
		Roots    []string `json:"roots"`
	}{
		Accounts: appConfig.Mappings.GetAllAccounts(),
		Roots:    appConfig.Settings.RootAccounts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *WebAppServer) handleSelectAccount(w http.ResponseWriter, r *http.Request) {
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
	if err != nil || !s.validateInitData(payload.InitData) {
		if err != nil {
			log.Printf("InitData parse error: %v", err)
		}
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
	_, exists := s.sessionManager.Get(user.ID)
	if !exists {
		http.Error(w, "Session expired", http.StatusNotFound)
		return
	}

	formattedAccount := domain.FormatAccountPath(payload.Account)

	s.sessionManager.Update(user.ID, func(sess *UserSession) {
		if payload.Type == "source" {
			sess.Draft.Postings[1].Account = formattedAccount
			sess.SourceOverridden = true
		} else {
			sess.Draft.Postings[0].Account = formattedAccount
			sess.TargetOverridden = true
		}
	})

	// 4. Update the bot message asynchronously
	if err := s.refresher.RefreshDraftMessage(user.ID); err != nil {
		log.Printf("Failed to refresh message for user %d: %v", user.ID, err)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *WebAppServer) validateInitData(initDataRaw string) bool {
	params, err := url.ParseQuery(initDataRaw)
	if err != nil {
		return false
	}
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
	mac.Write([]byte(s.botToken))
	secretKey := mac.Sum(nil)

	// 2. Create signature: HMAC-SHA256(dataCheckString, secretKey)
	mac = hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString.String()))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	return expectedHash == hash
}
