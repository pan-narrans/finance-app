package app

import (
	"fmt"
	"strings"
	"time"

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
	configUseCase  ports.ConfigurationUseCase
}

/*
NewReportService creates a new ReportService.
*/
func NewReportService(reportProvider ports.ReportProvider, configUseCase ports.ConfigurationUseCase) *ReportService {
	return &ReportService{
		reportProvider: reportProvider,
		configUseCase:  configUseCase,
	}
}

/*
GetMonthlyReport returns segmented balance reports for the given period.
It iterates through root accounts defined in configuration.
*/
func (s *ReportService) GetMonthlyReport(period string) ([]ports.ReportSection, error) {
	if period == "" {
		period = "this month"
	}

	dateRange := s.calculateDateRange(period)
	rootAccounts := s.configUseCase.Get().Settings.RootAccounts
	var sections []ports.ReportSection

	for _, root := range rootAccounts {
		report, err := s.reportProvider.GetBalanceReport(period, root)
		if err != nil {
			return nil, fmt.Errorf("failed to get report for %s: %w", root, err)
		}

		if strings.TrimSpace(report) != "" {
			sections = append(
				sections, ports.ReportSection{
					Title:     root,
					DateRange: dateRange,
					Content:   report,
				},
			)
		}
	}

	return sections, nil
}

func (s *ReportService) calculateDateRange(period string) string {
	now := time.Now()
	var start, end time.Time

	if period == "last month" {
		// First day of last month
		start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		// Last day of last month
		end = start.AddDate(0, 1, -1)
	} else {
		// First day of this month
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		// Today
		end = now
	}

	return fmt.Sprintf("%s - %s", start.Format("02/01/2006"), end.Format("02/01/2006"))
}
