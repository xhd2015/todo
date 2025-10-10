package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

var (
	infoLogger  *slog.Logger
	errorLogger *slog.Logger
)

// Init initializes the info and error loggers with separate files
func Init() error {
	// Create or open info.log file
	infoFile, err := os.OpenFile("info.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open info.log: %w", err)
	}

	// Create or open error.log file
	errorFile, err := os.OpenFile("error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open error.log: %w", err)
	}

	// Create structured loggers with plain text format
	infoLogger = slog.New(slog.NewTextHandler(infoFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	errorLogger = slog.New(slog.NewTextHandler(errorFile, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	return nil
}

// Infof logs an info-level message with format string and arguments
func Infof(ctx context.Context, format string, args ...interface{}) {
	if infoLogger == nil {
		return
	}
	message := fmt.Sprintf(format, args...)
	infoLogger.InfoContext(ctx, message)
}

// Errorf logs an error-level message with format string and arguments
func Errorf(ctx context.Context, format string, args ...interface{}) {
	if errorLogger == nil {
		return
	}
	message := fmt.Sprintf(format, args...)
	errorLogger.ErrorContext(ctx, message)
}

// Info logs an info-level message
func Info(ctx context.Context, msg string, args ...any) {
	if infoLogger == nil {
		return
	}
	infoLogger.InfoContext(ctx, msg, args...)
}

// Error logs an error-level message
func Error(ctx context.Context, msg string, args ...any) {
	if errorLogger == nil {
		return
	}
	errorLogger.ErrorContext(ctx, msg, args...)
}

// SetInfoOutput sets a custom writer for info logs (useful for testing)
func SetInfoOutput(w io.Writer) {
	infoLogger = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// SetErrorOutput sets a custom writer for error logs (useful for testing)
func SetErrorOutput(w io.Writer) {
	errorLogger = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}
