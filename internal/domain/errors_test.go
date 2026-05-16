package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainError_Error_ShouldReturnDetailedSummary_WhenMultipleErrorsExist(t *testing.T) {
	// Arrange
	errs := NewDomainError("Transaction", "Date", "required")
	errs.Add("Transaction", "Description", "too short")

	// Act
	got := errs.Error()

	// Assert
	assert.Equal(t, "domain failure: 2 error(s) in Transaction: [Date: required], [Description: too short]", got)
}

func TestDomainError_Error_ShouldReturnSingleSummary_WhenOneErrorExists(t *testing.T) {
	// Arrange
	errs := NewDomainError("User", "Email", "invalid format")

	// Act
	got := errs.Error()

	// Assert
	assert.Equal(t, "domain failure: 1 error(s) in User: [Email: invalid format]", got)
}

func TestDomainError_Error_ShouldReturnDefaultMessage_WhenEmpty(t *testing.T) {
	// Arrange
	errs := &DomainError{}

	// Act
	got := errs.Error()

	// Assert
	assert.Equal(t, "no domain errors occurred", got)
}

func TestNewDomainError_ShouldInitializeWithOneError(t *testing.T) {
	// Act
	errs := NewDomainError("Entity", "Field", "Message")

	// Assert
	assert.Len(t, errs.Errors, 1)
	assert.Equal(t, "Entity", errs.Errors[0].Entity)
}
