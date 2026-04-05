package domain

import (
	"fmt"
	"strings"
)

// ValidationError represents a single, field-specific issue within a domain entity.
// It is used by [MultiError].
type ValidationError struct {
	Entity  string `json:"entity"`  // The name of the domain entity (e.g., "Transaction").
	Field   string `json:"field"`   // The specific field that failed validation (e.g., "Date").
	Message string `json:"message"` // A human-readable error message.
}

// MultiError is a structured collection of one or more validation errors.
// It implements the standard error interface for basic error reporting.
type MultiError struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the standard error interface.
// It provides a detailed summary of all validation failures, including entity, field, and message.
func (e *MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no validation errors occurred"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("validation failed: %d error(s) in %s: ", len(e.Errors), e.Errors[0].Entity))

	for i, err := range e.Errors {
		if i > 0 {
			builder.WriteString(", ")
		}
		fmt.Fprintf(&builder, "[%s: %s]", err.Field, err.Message)
	}

	return builder.String()
}
