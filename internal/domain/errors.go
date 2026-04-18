package domain

import (
	"fmt"
	"strings"
)

// TODO the naming in this file is absolute garbage. I'll need to revisit it.

// ValidationError represents a single, field-specific issue within a domain entity.
// It is used by [DomainError].
type ValidationError struct {
	Entity  string `json:"entity"`  // The name of the domain entity (e.g., "Transaction").
	Field   string `json:"field"`   // The specific field that failed validation (e.g., "Date").
	Message string `json:"message"` // A human-readable error message.
}

// ValidationErrors DomainError is a structured collection of one or more validation errors.
// It implements the standard error interface for basic error reporting.
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// NewValidationErrors creates a DomainError containing a single ValidationError.
// It is a helper to simplify returning single validation issues.
func NewValidationErrors(entity, field, message string) *ValidationErrors {
	return &ValidationErrors{
		Errors: []ValidationError{
			{Entity: entity, Field: field, Message: message},
		},
	}
}

// Add appends a new ValidationError to the DomainError collection.
func (domainError *ValidationErrors) Add(entity, field, message string) {
	domainError.Errors = append(
		domainError.Errors, ValidationError{
			Entity:  entity,
			Field:   field,
			Message: message,
		},
	)
}

// ValidationErrors implements the standard error interface.
// It provides a detailed summary of all validation failures, including entity, field, and message.
func (domainError *ValidationErrors) Error() string {
	if len(domainError.Errors) == 0 {
		return "no validation errors occurred"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("validation failed: %d error(s) in %s: ", len(domainError.Errors), domainError.Errors[0].Entity))

	for i, validationError := range domainError.Errors {
		if i > 0 {
			builder.WriteString(", ")
		}
		fmt.Fprintf(&builder, "[%s: %s]", validationError.Field, validationError.Message)
	}

	return builder.String()
}
