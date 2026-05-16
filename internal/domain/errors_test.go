package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationErrors_Error_ShouldReturnDetailedSummary_WhenMultipleErrorsExist(t *testing.T) {
	// Arrange
	errs := NewValidationErrors("Transaction", "Date", "required")
	errs.Add("Transaction", "Description", "too short")

	// Act
	got := errs.Error()

	// Assert
	assert.Equal(t, "validation failed: 2 error(s) in Transaction: [Date: required], [Description: too short]", got)
}

func TestValidationErrors_Error_ShouldReturnSingleSummary_WhenOneErrorExists(t *testing.T) {
	// Arrange
	errs := NewValidationErrors("User", "Email", "invalid format")

	// Act
	got := errs.Error()

	// Assert
	assert.Equal(t, "validation failed: 1 error(s) in User: [Email: invalid format]", got)
}

func TestValidationErrors_Error_ShouldReturnDefaultMessage_WhenEmpty(t *testing.T) {
	// Arrange
	errs := &ValidationErrors{}

	// Act
	got := errs.Error()

	// Assert
	assert.Equal(t, "no validation errors occurred", got)
}

func TestNewValidationErrors_ShouldInitializeWithOneError(t *testing.T) {
	// Act
	errs := NewValidationErrors("Entity", "Field", "Message")

	// Assert
	assert.Len(t, errs.Errors, 1)
	assert.Equal(t, "Entity", errs.Errors[0].Entity)
}
