package app

import (
	"io"
	"log/slog"
	"os"
)

/*
SetupLogger initializes a structured logger using slog.
It writes to both the provided writer (e.g., a file) and stdout.
*/
func SetupLogger(logFile io.Writer) *slog.Logger {
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return slog.New(handler)
}
