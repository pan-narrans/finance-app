package app

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/a-perez/finance-app/internal/app/ports"
	"github.com/a-perez/finance-app/internal/domain"
)

// ImportService orchestrates the process of parsing and persisting transactions from external files.
type ImportService struct {
	transactionUseCase ports.TransactionUseCase
	backupService      ports.BackupService
	logger             *slog.Logger
	ledgerFilePath     string
}

// NewImportService creates a new instance of ImportService.
func NewImportService(
	transactionUseCase ports.TransactionUseCase,
	backupService ports.BackupService,
	logger *slog.Logger,
	ledgerFilePath string,
) *ImportService {
	return &ImportService{
		transactionUseCase: transactionUseCase,
		backupService:      backupService,
		logger:             logger,
		ledgerFilePath:     ledgerFilePath,
	}
}

/*
Import parses a file using the provided parser and saves the resulting transactions.
It continues processing even if individual transactions fail to save.
*/
func (importService *ImportService) Import(parser ports.BankParser, filePath string) (*ports.ImportSummary, error) {
	// 1. Create pre-import backup
	sessionID, err := importService.backupService.CreateBackup(importService.ledgerFilePath)
	if err != nil {
		return nil, fmt.Errorf("pre-import backup failed: %w", err)
	}

	importService.logger.Info(
		"Starting import session",
		slog.String("sessionID", sessionID),
		slog.String("file", filePath),
	)

	transactions, err := parser.Parse(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Sort transactions chronologically (oldest first).
	sort.SliceStable(
		transactions, func(i, j int) bool {
			return transactions[i].Date.Before(transactions[j].Date)
		},
	)

	summary := &ports.ImportSummary{
		Total:  len(transactions),
		Errors: make(map[int]error),
	}

	for index, transaction := range transactions {
		if err = importService.processTransaction(transaction, summary); err != nil {
			summary.Failed++
			summary.Errors[index] = err
			importService.logger.Error(
				"Transaction failed",
				slog.Int("index", index),
				slog.String("desc", transaction.Description),
				slog.Any("error", err),
			)
		}
	}

	// 2. Save human-readable diff
	if err := importService.backupService.SaveDiff(sessionID, importService.ledgerFilePath); err != nil {
		importService.logger.Warn("Failed to save session diff", slog.Any("error", err))
	}

	importService.logger.Info(
		"Import session finished",
		slog.String("sessionID", sessionID),
		slog.Int("added", summary.Added),
		slog.Int("updated", summary.Updated),
		slog.Int("failed", summary.Failed),
	)

	return summary, nil
}

// RollbackLastImport restores the ledger to the state before the last successful import start.
func (importService *ImportService) RollbackLastImport() error {
	importService.logger.Info("Attempting to rollback last import")
	if err := importService.backupService.RestoreLast(importService.ledgerFilePath); err != nil {
		importService.logger.Error("Rollback failed", slog.Any("error", err))
		return fmt.Errorf("rollback failed: %w", err)
	}
	importService.logger.Info("Rollback successful")
	return nil
}

func (importService *ImportService) processTransaction(transaction domain.Transaction, summary *ports.ImportSummary) error {
	if transaction.Code == "" {
		transaction.Code = transaction.GenerateCode()
	}
	existing, err := importService.transactionUseCase.GetByCode(transaction.Code)

	if err != nil {
		return fmt.Errorf("lookup failed: %w", err)
	}

	if existing != nil {
		if err := importService.transactionUseCase.Update(transaction); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		summary.Updated++
		importService.logger.Info(
			"Transaction updated",
			slog.String("code", transaction.Code),
			slog.String("desc", transaction.Description),
		)
	} else {
		if err := importService.transactionUseCase.Add(transaction); err != nil {
			return fmt.Errorf("add failed: %w", err)
		}
		summary.Added++
		importService.logger.Info(
			"Transaction added",
			slog.String("code", transaction.Code),
			slog.String("desc", transaction.Description),
		)
	}

	return nil
}
