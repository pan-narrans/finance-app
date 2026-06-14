package e2e

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_ReportGeneration_ShouldReturnReportSections_WhenHappyPath(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	// Add some data to ledger to have a report
	env.sendText("100 Income:Salary")
	assert.Eventually(t, func() bool {
		_, ok := env.adapter.SessionManager().Get(env.userID)
		return ok
	}, 5*time.Second, 100*time.Millisecond, "Session should be created for Salary")
	env.sendCallback("confirm")

	// Wait for first transaction to be written to ledger
	assert.Eventually(t, func() bool {
		content, err := os.ReadFile(env.ledgerPath)
		if err != nil {
			return false
		}
		return strings.Contains(string(content), "Income:Salary")
	}, 5*time.Second, 100*time.Millisecond, "First transaction should be written to ledger")

	env.sendText("50 Expenses:Food")
	assert.Eventually(t, func() bool {
		_, ok := env.adapter.SessionManager().Get(env.userID)
		return ok
	}, 5*time.Second, 100*time.Millisecond, "Session should be created for Food")
	env.sendCallback("confirm")

	// Wait for ledger to have 2 transactions - check for actual transaction markers
	assert.Eventually(t, func() bool {
		content, err := os.ReadFile(env.ledgerPath)
		if err != nil {
			return false
		}
		ledgerText := string(content)
		// Check for both transaction accounts instead of just year
		return strings.Contains(ledgerText, "Income:Salary") &&
			strings.Contains(ledgerText, "Expenses:Food")
	}, 5*time.Second, 100*time.Millisecond, "Ledger should have 2 transactions")

	// Act
	env.sendCommand("report")

	// Assert
	// We can check the ReportService directly to ensure it generates data from the real ledger
	sections, err := env.reportService.GetMonthlyReport("this month")
	require.NoError(t, err)
	assert.NotEmpty(t, sections)
}

func TestE2E_ReportGeneration_ShouldHandleEmptyLedger(t *testing.T) {
	// Arrange
	env := setupE2EEnv(t)

	// Act
	sections, err := env.reportService.GetMonthlyReport("this month")

	// Assert
	require.NoError(t, err)
	assert.Empty(t, sections)
}
