package domain

import (
	"fmt"
	"strings"
)

/*
DomainFieldError represents a single, field-specific issue within a domain entity.

Fields:
  - Entity: The name of the domain entity (e.g., "Transaction").
  - Field: The specific field that failed validation (e.g., "Date").
  - Message: A human-readable error message.
*/
type DomainFieldError struct {
	Entity  string `json:"entity"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

/*
DomainError is a structured collection of one or more domain-level issues.
It implements the standard error interface.
*/
type DomainError struct {
	Errors []DomainFieldError `json:"errors"`
}

// NewDomainError creates a DomainError containing a single DomainFieldError.
func NewDomainError(entity, field, message string) *DomainError {
	return &DomainError{
		Errors: []DomainFieldError{
			{Entity: entity, Field: field, Message: message},
		},
	}
}

// Add appends a new field-specific error to the collection.
func (domainError *DomainError) Add(entity, field, message string) {
	domainError.Errors = append(
		domainError.Errors, DomainFieldError{
			Entity:  entity,
			Field:   field,
			Message: message,
		},
	)
}

/*
Error implements the standard error interface.
Returns a detailed summary of all domain failures.
*/
func (domainError *DomainError) Error() string {
	if len(domainError.Errors) == 0 {
		return "no domain errors occurred"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "domain failure: %d error(s) in %s: ", len(domainError.Errors), domainError.Errors[0].Entity)

	for i, fieldError := range domainError.Errors {
		if i > 0 {
			builder.WriteString(", ")
		}
		fmt.Fprintf(&builder, "[%s: %s]", fieldError.Field, fieldError.Message)
	}

	return builder.String()
}
