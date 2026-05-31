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
GetMonthlyReport returns segmented balance reports for the current month.
It iterates through root accounts defined in configuration.
*/
func (s *ReportService) GetMonthlyReport() ([]ports.ReportSection, error) {
	rootAccounts := s.configUseCase.Get().Settings.RootAccounts
	var sections []ports.ReportSection

	for _, root := range rootAccounts {
		report, err := s.reportProvider.GetBalanceReport("this month", root)
		if err != nil {
			return nil, fmt.Errorf("failed to get report for %s: %w", root, err)
		}

		if strings.TrimSpace(report) != "" {
			sections = append(sections, ports.ReportSection{
				Title:   root,
				Content: report,
			})
		}
	}

	return sections, nil
}
