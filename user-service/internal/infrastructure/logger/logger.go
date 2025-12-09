package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger wraps zerolog logger
type Logger struct {
	logger zerolog.Logger
}

// New creates a new logger instance
func New(serviceName string, environment string) *Logger {
	var writer io.Writer = os.Stdout

	// Pretty logging for development
	if environment == "development" {
		writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if environment == "development" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	logger := zerolog.New(writer).
		With().
		Timestamp().
		Str("service", serviceName).
		Str("environment", environment).
		Logger()

	// Set as global logger
	log.Logger = logger

	return &Logger{logger: logger}
}

// Info logs an info message
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Debug logs a debug message
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Warn logs a warning message
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Error logs an error message
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

// With creates a child logger with additional fields
func (l *Logger) With() zerolog.Context {
	return l.logger.With()
}

// GetZerolog returns the underlying zerolog logger
func (l *Logger) GetZerolog() *zerolog.Logger {
	return &l.logger
}
