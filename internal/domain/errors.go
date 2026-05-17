package domain

import (
	"fmt"
	"strings"
)

// TODO the naming in this file is absolute garbage. I'll need to revisit it.

/*
ValidationError represents a single, field-specific issue within a domain entity.

Fields:
  - Entity: The name of the domain entity (e.g., "Transaction").
  - Field: The specific field that failed validation (e.g., "Date").
  - Message: A human-readable error message.
*/
type ValidationError struct {
	Entity  string `json:"entity"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

/*
ValidationErrors is a structured collection of one or more validation errors.
It implements the standard error interface.
*/
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// NewValidationErrors creates a ValidationErrors containing a single ValidationError.
func NewValidationErrors(entity, field, message string) *ValidationErrors {
	return &ValidationErrors{
		Errors: []ValidationError{
			{Entity: entity, Field: field, Message: message},
		},
	}
}

// Add appends a new ValidationError to the collection.
func (validationErrors *ValidationErrors) Add(entity, field, message string) {
	validationErrors.Errors = append(
		validationErrors.Errors, ValidationError{
			Entity:  entity,
			Field:   field,
			Message: message,
		},
	)
}

/*
Error implements the standard error interface.
Returns a detailed summary of all validation failures.
*/
func (validationErrors *ValidationErrors) Error() string {
	if len(validationErrors.Errors) == 0 {
		return "no validation errors occurred"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "validation failed: %d error(s) in %s: ", len(validationErrors.Errors), validationErrors.Errors[0].Entity)

	for i, validationError := range validationErrors.Errors {
		if i > 0 {
			builder.WriteString(", ")
		}
		fmt.Fprintf(&builder, "[%s: %s]", validationError.Field, validationError.Message)
	}

	return builder.String()
}
