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
	}, 2*time.Second, 100*time.Millisecond, "Session should be created for Salary")
	env.sendCallback("confirm")

	env.sendText("50 Expenses:Food")
	assert.Eventually(t, func() bool {
		_, ok := env.adapter.SessionManager().Get(env.userID)
		return ok
	}, 2*time.Second, 100*time.Millisecond, "Session should be created for Food")
	env.sendCallback("confirm")

	// Wait for ledger to have 2 transactions
	assert.Eventually(t, func() bool {
		content, _ := os.ReadFile(env.ledgerPath)
		return strings.Count(string(content), "2026") >= 2 // Assuming date starts with 2026
	}, 2*time.Second, 100*time.Millisecond, "Ledger should have 2 transactions")

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
