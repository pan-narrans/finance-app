package telegram

import (
	"sync"

	"github.com/a-perez/finance-app/internal/domain"
)

/*
SearchState represents the current step in a user's multi-message interaction.
*/
type SearchState string

const (
	StateNone                  SearchState = ""                // StateNone indicates the user is not in a multistep flow.
	StateAwaitingQuery         SearchState = "awaiting_query"  // StateAwaitingQuery indicates the bot is waiting for an account search query.
	StateCreatingAccountParent SearchState = "creating_parent" // StateCreatingAccountParent indicates the user is selecting the root account.
	StateCreatingAccountChild  SearchState = "creating_child"  // StateCreatingAccountChild indicates the user is typing a subaccount name.
	StateCreatingAccountReview SearchState = "creating_review" // StateCreatingAccountReview indicates the user is reviewing the constructed path.
)

/*
UserSession stores transient data and state for a specific Telegram user.
*/
type UserSession struct {
	Draft                 domain.Transaction
	State                 SearchState
	EditingPosting        int
	NewAccountPath        string
	TargetOverridden      bool
	SourceOverridden      bool
	OriginalSourceKeyword string
	PendingQueue          []domain.Transaction
	LastMessageID         int
	LastChatID            int64
}

/*
SessionManager provides thread-safe management of active user sessions.
*/
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[int64]*UserSession
}

/*
NewSessionManager creates a new SessionManager.
*/
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[int64]*UserSession),
	}
}

/*
Get retrieves a session for a specific user.
It returns the session and true if found; otherwise, nil and false.
*/
func (m *SessionManager) Get(userID int64) (*UserSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[userID]
	return session, ok
}

/*
Set initializes or overwrites a session for a specific user.
*/
func (m *SessionManager) Set(userID int64, session *UserSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[userID] = session
}

/*
Delete removes a session for a specific user.
*/
func (m *SessionManager) Delete(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, userID)
}

/*
Update provides a way to modify a session in a thread-safe manner using a callback.
If the session does not exist, the callback is not executed.
*/
func (m *SessionManager) Update(userID int64, fn func(session *UserSession)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session, ok := m.sessions[userID]; ok {
		fn(session)
	}
}
