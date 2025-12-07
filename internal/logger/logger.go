package logger

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

// InitLogger initializes the global logger
// Following Open/Closed Principle: open for extension (can add new writers), closed for modification
func InitLogger(level, format string) {
	// Set log level
	logLevel := parseLogLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	// Set output format
	var output io.Writer = os.Stdout
	if format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
			NoColor:    false,
			FormatLevel: func(i interface{}) string {
				level := strings.ToUpper(fmt.Sprintf("%s", i))
				switch level {
				case "INFO":
					return "\033[32mINFO\033[0m"  // Green
				case "WARN":
					return "\033[33mWARN\033[0m"  // Yellow
				case "ERROR":
					return "\033[31mERROR\033[0m" // Red
				case "FATAL":
					return "\033[35mFATAL\033[0m" // Magenta
				case "DEBUG":
					return "\033[36mDEBUG\033[0m" // Cyan
				default:
					return level
				}
			},
			FormatMessage: func(i interface{}) string {
				return fmt.Sprintf("| %s", i)
			},
			FormatFieldName: func(i interface{}) string {
				return fmt.Sprintf("\033[36m%s\033[0m=", i)
			},
			FormatFieldValue: func(i interface{}) string {
				return fmt.Sprintf("\033[33m%s\033[0m", i)
			},
		}
	}

	logger = zerolog.New(output).With().
		Timestamp().
		Caller().
		Logger()

	// Set as global logger
	log.Logger = logger
}

// GetLogger returns the configured logger instance
func GetLogger() *zerolog.Logger {
	return &logger
}

// parseLogLevel converts string log level to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Helper functions for structured logging

// Info logs an info message
func Info() *zerolog.Event {
	return logger.Info()
}

// Debug logs a debug message
func Debug() *zerolog.Event {
	return logger.Debug()
}

// Warn logs a warning message
func Warn() *zerolog.Event {
	return logger.Warn()
}

// Error logs an error message
func Error() *zerolog.Event {
	return logger.Error()
}

// Fatal logs a fatal message and exits
func Fatal() *zerolog.Event {
	return logger.Fatal()
}

// WithContext creates a new logger with additional context
func WithContext(key string, value interface{}) zerolog.Logger {
	return logger.With().Interface(key, value).Logger()
}
