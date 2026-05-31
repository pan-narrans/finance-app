package app

import (
	"fmt"
	"strings"

	"github.com/a-perez/finance-app/internal/app/ports"
)

// Ensure ReportService implements ports.ReportUseCase at compile time.
var _ ports.ReportUseCase = (*ReportService)(nil)

/*
ReportService handles the generation of financial reports.
It implements the ports.ReportUseCase interface.
*/
type ReportService struct {
	reportProvider ports.ReportProvider
}

/*
NewReportService creates a new ReportService.
*/
func NewReportService(reportProvider ports.ReportProvider) *ReportService {
	return &ReportService{
		reportProvider: reportProvider,
	}
}

/*
GetMonthlyReport returns a formatted balance report for the current month.
*/
func (s *ReportService) GetMonthlyReport() (string, error) {
	report, err := s.reportProvider.GetBalanceReport("this month")
	if err != nil {
		return "", fmt.Errorf("failed to get monthly report: %w", err)
	}

	if strings.TrimSpace(report) == "" {
		return "No data for this month.", nil
	}

	return report, nil
}
