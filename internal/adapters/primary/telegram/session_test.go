package telegram

import (
	"sync"
	"testing"

	"github.com/a-perez/finance-app/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestSessionManager_CRUD_ShouldWorkAsExpected(t *testing.T) {
	// Arrange
	manager := NewSessionManager()
	userID := int64(123)
	session := &UserSession{
		Draft: domain.Transaction{Description: "Test"},
		State: StateNone,
	}

	// Act: Set & Get
	manager.Set(userID, session)
	got, ok := manager.Get(userID)

	// Assert
	assert.True(t, ok)
	assert.Equal(t, *session, got)

	// Act: Update
	manager.Update(
		userID, func(s *UserSession) {
			s.State = StateAwaitingQuery
		},
	)
	updated, _ := manager.Get(userID)

	// Assert
	assert.Equal(t, StateAwaitingQuery, updated.State)

	// Act: Delete
	manager.Delete(userID)
	_, ok = manager.Get(userID)

	// Assert
	assert.False(t, ok)
}

func TestSessionManager_Update_ShouldDoNothing_WhenSessionDoesNotExist(t *testing.T) {
	// Arrange
	manager := NewSessionManager()
	called := false

	// Act
	manager.Update(
		999, func(s *UserSession) {
			called = true
		},
	)

	// Assert
	assert.False(t, called)
}

func TestSessionManager_Concurrency_ShouldNotPanic(t *testing.T) {
	// Arrange
	manager := NewSessionManager()
	var wg sync.WaitGroup
	count := 100

	// Act
	wg.Add(count * 2)
	for i := 0; i < count; i++ {
		go func(id int64) {
			defer wg.Done()
			manager.Set(id, &UserSession{})
		}(int64(i))

		go func(id int64) {
			defer wg.Done()
			manager.Get(id)
		}(int64(i))
	}
	wg.Wait()

	// Assert: No panic occurred
}
