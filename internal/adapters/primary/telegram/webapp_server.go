package telegram

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a-perez/finance-app/internal/adapters/primary/telegram/webapp"
	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

type contextKey string

const (
	userContextKey contextKey = "user"
)

type WebAppUser struct {
	ID int64 `json:"id"`
}

/*
MessageRefresher defines the contract for updating Telegram chat messages.
This allows the WebAppServer to trigger UI updates without being tightly coupled to the bot.
*/
type MessageRefresher interface {
	RefreshDraftMessage(userID int64) error
	StartImportReview(userID int64, pending []domain.Transaction) error
}

/*
WebAppServer handles HTTP requests from the Telegram Mini App.
It serves static assets and provides the necessary API endpoints.
*/
type WebAppServer struct {
	port           int
	botToken       string
	configUseCase  ports.ConfigurationUseCase
	transactionUC  ports.TransactionUseCase
	reportUC       ports.ReportUseCase
	importUC       ports.ImportUseCase
	sessionManager *SessionManager
	refresher      MessageRefresher
}

/*
NewWebAppServer creates a new instance of WebAppServer.
*/
func NewWebAppServer(
	port int,
	token string,
	configUC ports.ConfigurationUseCase,
	transactionUC ports.TransactionUseCase,
	reportUC ports.ReportUseCase,
	importUC ports.ImportUseCase,
	sessionManager *SessionManager,
	refresher MessageRefresher,
) *WebAppServer {
	return &WebAppServer{
		port:           port,
		botToken:       token,
		configUseCase:  configUC,
		transactionUC:  transactionUC,
		reportUC:       reportUC,
		importUC:       importUC,
		sessionManager: sessionManager,
		refresher:      refresher,
	}
}

/*
Start launches the HTTP server in a blocking manner.
*/
func (s *WebAppServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("WebApp server listening on %s (embedded assets)", addr)
	return http.ListenAndServe(addr, s.Router())
}

/*
Router creates and returns the http.Handler for the WebApp.
*/
func (s *WebAppServer) Router() http.Handler {
	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("/api/accounts", s.handleGetAccounts)
	mux.HandleFunc("/api/select", s.handleSelectAccount)
	mux.HandleFunc("/api/transaction", s.handleTransaction)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/reports", s.handleReports)
	mux.HandleFunc("/api/import", s.handleImport)

	// Static Assets (Embedded)
	staticFS, err := fs.Sub(webapp.Assets, "dist")
	if err != nil {
		log.Printf("Warning: failed to access embedded assets: %v", err)
	}
	fsServer := http.FileServer(http.FS(staticFS))

	// Middleware stack
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Printf("[HTTP] %s %s", r.Method, r.URL.Path)

			// Allow framing by Telegram
			w.Header().Set("Content-Security-Policy", "frame-ancestors https://web.telegram.org https://t.me;")
			// Remove X-Frame-Options if any, or set to allow
			w.Header().Del("X-Frame-Options")

			// Prevent WebView caching of index.html so updates are instant
			if r.URL.Path == "/" || r.URL.Path == "/index.html" {
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
			} else if strings.HasPrefix(r.URL.Path, "/assets/") {
				// Assets contain content hashes, so they are safe to cache aggressively
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}

			if strings.HasPrefix(r.URL.Path, "/api/") {
				s.authMiddleware(mux).ServeHTTP(w, r)
			} else {
				mux.ServeHTTP(w, r)
			}
		},
	)

	mux.Handle("/", fsServer)
	return handler
}

func (s *WebAppServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			initDataRaw := r.Header.Get("X-TMA-Init-Data")
			if initDataRaw == "" {
				http.Error(w, "Unauthorized: Missing initData", http.StatusUnauthorized)
				return
			}

			if !s.validateInitData(initDataRaw) {
				http.Error(w, "Unauthorized: Invalid initData", http.StatusUnauthorized)
				return
			}

			params, _ := url.ParseQuery(initDataRaw)
			userJSON := params.Get("user")
			var user WebAppUser
			if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
				http.Error(w, "Bad Request: Invalid user data", http.StatusBadRequest)
				return
			}

			// Strict Authorization Check: Validate user ID against allowed IDs
			allowedIDs := s.configUseCase.Get().Settings.TelegramUserIDs
			isAllowed := false
			for _, id := range allowedIDs {
				if id == user.ID {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				log.Printf("[AUTH-WEBAPP] Unauthorized API access attempt from User ID: %d", user.ID)
				http.Error(w, "Forbidden: User is not authorized to use this application", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}

func (s *WebAppServer) handleGetAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	appConfig := s.configUseCase.Get()
	accounts := appConfig.Mappings.GetAllAccounts()
	if accounts == nil {
		accounts = []string{}
	}
	roots := appConfig.Settings.RootAccounts
	if roots == nil {
		roots = []string{}
	}

	response := struct {
		Accounts []string `json:"accounts"`
		Roots    []string `json:"roots"`
	}{
		Accounts: accounts,
		Roots:    roots,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *WebAppServer) handleTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Date        string  `json:"date"`
		Description string  `json:"description"`
		Amount      float64 `json:"amount"`
		Source      string  `json:"source"`
		Target      string  `json:"target"`
		Currency    string  `json:"currency"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", payload.Date)
	if err != nil {
		http.Error(w, "Invalid date format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	tx := domain.Transaction{
		Date:        date,
		Description: payload.Description,
		Metadata: domain.Metadata{
			Origin: domain.OriginTelegram,
		},
		Postings: []domain.Posting{
			{
				Account:  domain.FormatAccountPath(payload.Target),
				Amount:   &payload.Amount,
				Currency: payload.Currency,
			},
			{
				Account: domain.FormatAccountPath(payload.Source),
			},
		},
	}

	if err := s.transactionUC.Add(tx); err != nil {
		log.Printf("Failed to add transaction: %v", err)
		http.Error(w, fmt.Sprintf("Failed to add transaction: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *WebAppServer) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}

	transactions, err := s.transactionUC.List(limit)
	if err != nil {
		log.Printf("Failed to get history: %v", err)
		http.Error(w, "Failed to get history", http.StatusInternalServerError)
		return
	}
	if transactions == nil {
		transactions = []domain.Transaction{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}

func (s *WebAppServer) handleReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	period := r.URL.Query().Get("period")
	sections, err := s.reportUC.GetMonthlyReport(period)
	if err != nil {
		log.Printf("Failed to get reports: %v", err)
		http.Error(w, "Failed to get reports", http.StatusInternalServerError)
		return
	}
	if sections == nil {
		sections = []ports.ReportSection{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sections)
}

func (s *WebAppServer) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit to 10MB
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save to temp file (preserve original filename keywords so parser factory can match bank type)
	safeName := strings.ReplaceAll(handler.Filename, "/", "_")
	tempFile, err := os.CreateTemp("", "finance-import-*"+safeName)
	if err != nil {
		http.Error(w, "Error creating temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	bankType := r.FormValue("bank_type")
	summary, err := s.importUC.Import(tempFile.Name(), bankType)
	if err != nil {
		log.Printf("Import failed: %v", err)
		http.Error(w, fmt.Sprintf("Import failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (s *WebAppServer) handleSelectAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Account string `json:"account"`
		Type    string `json:"type"` // "source" or "target"
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	user, ok := r.Context().Value(userContextKey).(*WebAppUser)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 3. Update Session
	_, exists := s.sessionManager.Get(user.ID)
	if !exists {
		http.Error(w, "Session expired", http.StatusNotFound)
		return
	}

	formattedAccount := domain.FormatAccountPath(payload.Account)

	s.sessionManager.Update(
		user.ID, func(sess *UserSession) {
			targetIndex, sourceIndex := 0, 1
			if sess.Draft.IsIncome() {
				targetIndex, sourceIndex = 1, 0
			}

			if payload.Type == "source" {
				sess.Draft.Postings[sourceIndex].Account = formattedAccount
				sess.SourceOverridden = true
			} else {
				sess.Draft.Postings[targetIndex].Account = formattedAccount
				sess.TargetOverridden = true
			}
		},
	)

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

// trigger-air-rebuild-v10
