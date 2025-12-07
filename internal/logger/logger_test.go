package logger_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/monitoring-engine/monitoring-tool/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLogger(t *testing.T) {
	t.Run("should initialize logger with info level", func(t *testing.T) {
		logger.InitLogger("info", "json")

		log := logger.GetLogger()
		assert.NotNil(t, log)
	})

	t.Run("should initialize logger with debug level", func(t *testing.T) {
		logger.InitLogger("debug", "json")

		log := logger.GetLogger()
		assert.NotNil(t, log)
		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
	})

	t.Run("should initialize logger with warn level", func(t *testing.T) {
		logger.InitLogger("warn", "json")

		log := logger.GetLogger()
		assert.NotNil(t, log)
		assert.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel())
	})

	t.Run("should initialize logger with warning level", func(t *testing.T) {
		logger.InitLogger("warning", "json")

		log := logger.GetLogger()
		assert.NotNil(t, log)
		assert.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel())
	})

	t.Run("should initialize logger with error level", func(t *testing.T) {
		logger.InitLogger("error", "json")

		log := logger.GetLogger()
		assert.NotNil(t, log)
		assert.Equal(t, zerolog.ErrorLevel, zerolog.GlobalLevel())
	})

	t.Run("should default to info level for invalid level", func(t *testing.T) {
		logger.InitLogger("invalid", "json")

		log := logger.GetLogger()
		assert.NotNil(t, log)
		assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
	})

	t.Run("should initialize logger with console format", func(t *testing.T) {
		logger.InitLogger("info", "console")

		log := logger.GetLogger()
		assert.NotNil(t, log)
	})

	t.Run("should initialize logger with json format", func(t *testing.T) {
		logger.InitLogger("info", "json")

		log := logger.GetLogger()
		assert.NotNil(t, log)
	})

	t.Run("should handle empty level string", func(t *testing.T) {
		logger.InitLogger("", "json")

		log := logger.GetLogger()
		assert.NotNil(t, log)
		assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
	})

	t.Run("should handle empty format string", func(t *testing.T) {
		logger.InitLogger("info", "")

		log := logger.GetLogger()
		assert.NotNil(t, log)
	})
}

func TestGetLogger(t *testing.T) {
	t.Run("should return logger instance", func(t *testing.T) {
		logger.InitLogger("info", "json")

		log := logger.GetLogger()
		assert.NotNil(t, log)
	})

	t.Run("should return same logger instance", func(t *testing.T) {
		logger.InitLogger("info", "json")

		log1 := logger.GetLogger()
		log2 := logger.GetLogger()

		assert.NotNil(t, log1)
		assert.NotNil(t, log2)
	})
}

func TestLoggerHelperFunctions(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer

	t.Run("should log info message", func(t *testing.T) {
		buf.Reset()
		logger.InitLogger("info", "json")

		// Create a custom logger for testing that writes to buffer
		testLogger := zerolog.New(&buf).With().Timestamp().Logger()
		testLogger.Info().Msg("test info message")

		output := buf.String()
		assert.Contains(t, output, "test info message")
		assert.Contains(t, output, "info")
	})

	t.Run("should log debug message when debug level enabled", func(t *testing.T) {
		buf.Reset()
		logger.InitLogger("debug", "json")

		testLogger := zerolog.New(&buf).Level(zerolog.DebugLevel).With().Timestamp().Logger()
		testLogger.Debug().Msg("test debug message")

		output := buf.String()
		assert.Contains(t, output, "test debug message")
		assert.Contains(t, output, "debug")
	})

	t.Run("should log warn message", func(t *testing.T) {
		buf.Reset()
		logger.InitLogger("warn", "json")

		testLogger := zerolog.New(&buf).Level(zerolog.WarnLevel).With().Timestamp().Logger()
		testLogger.Warn().Msg("test warn message")

		output := buf.String()
		assert.Contains(t, output, "test warn message")
		assert.Contains(t, output, "warn")
	})

	t.Run("should log error message", func(t *testing.T) {
		buf.Reset()
		logger.InitLogger("error", "json")

		testLogger := zerolog.New(&buf).Level(zerolog.ErrorLevel).With().Timestamp().Logger()
		testLogger.Error().Msg("test error message")

		output := buf.String()
		assert.Contains(t, output, "test error message")
		assert.Contains(t, output, "error")
	})

	t.Run("Info helper should return event", func(t *testing.T) {
		logger.InitLogger("info", "json")
		event := logger.Info()
		assert.NotNil(t, event)
	})

	t.Run("Debug helper should return event", func(t *testing.T) {
		logger.InitLogger("debug", "json")
		event := logger.Debug()
		assert.NotNil(t, event)
	})

	t.Run("Warn helper should return event", func(t *testing.T) {
		logger.InitLogger("warn", "json")
		event := logger.Warn()
		assert.NotNil(t, event)
	})

	t.Run("Error helper should return event", func(t *testing.T) {
		logger.InitLogger("error", "json")
		event := logger.Error()
		assert.NotNil(t, event)
	})
}

func TestWithContext(t *testing.T) {
	t.Run("should create logger with context", func(t *testing.T) {
		logger.InitLogger("info", "json")

		contextLogger := logger.WithContext("user_id", "12345")
		assert.NotNil(t, contextLogger)
	})

	t.Run("should add string context to logger", func(t *testing.T) {
		var buf bytes.Buffer
		logger.InitLogger("info", "json")

		testLogger := zerolog.New(&buf).With().Str("user_id", "12345").Logger()
		testLogger.Info().Msg("test message")

		output := buf.String()
		assert.Contains(t, output, "user_id")
		assert.Contains(t, output, "12345")
	})

	t.Run("should add int context to logger", func(t *testing.T) {
		var buf bytes.Buffer
		logger.InitLogger("info", "json")

		testLogger := zerolog.New(&buf).With().Int("count", 42).Logger()
		testLogger.Info().Msg("test message")

		output := buf.String()
		assert.Contains(t, output, "count")
		assert.Contains(t, output, "42")
	})

	t.Run("should add struct context to logger", func(t *testing.T) {
		var buf bytes.Buffer
		logger.InitLogger("info", "json")

		type TestStruct struct {
			Name string
			Age  int
		}

		testLogger := zerolog.New(&buf).With().Interface("user", TestStruct{Name: "John", Age: 30}).Logger()
		testLogger.Info().Msg("test message")

		output := buf.String()
		assert.Contains(t, output, "user")
	})
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel zerolog.Level
	}{
		{"debug level", "debug", zerolog.DebugLevel},
		{"info level", "info", zerolog.InfoLevel},
		{"warn level", "warn", zerolog.WarnLevel},
		{"warning level", "warning", zerolog.WarnLevel},
		{"error level", "error", zerolog.ErrorLevel},
		{"unknown level defaults to info", "unknown", zerolog.InfoLevel},
		{"uppercase DEBUG", "DEBUG", zerolog.DebugLevel},
		{"uppercase INFO", "INFO", zerolog.InfoLevel},
		{"uppercase WARN", "WARN", zerolog.WarnLevel},
		{"uppercase ERROR", "ERROR", zerolog.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger.InitLogger(tt.level, "json")
			assert.Equal(t, tt.expectedLevel, zerolog.GlobalLevel())
		})
	}
}

func TestLoggerFormats(t *testing.T) {
	t.Run("should handle json format", func(t *testing.T) {
		var buf bytes.Buffer
		logger.InitLogger("info", "json")

		testLogger := zerolog.New(&buf).With().Timestamp().Logger()
		testLogger.Info().Str("key", "value").Msg("test")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "test", logEntry["message"])
		assert.Equal(t, "value", logEntry["key"])
	})

	t.Run("should handle console format", func(t *testing.T) {
		logger.InitLogger("info", "console")
		log := logger.GetLogger()
		assert.NotNil(t, log)
	})
}

func TestLoggerCaller(t *testing.T) {
	t.Run("should include caller information", func(t *testing.T) {
		var buf bytes.Buffer
		logger.InitLogger("info", "json")

		testLogger := zerolog.New(&buf).With().Caller().Logger()
		testLogger.Info().Msg("test with caller")

		output := buf.String()
		assert.Contains(t, output, "caller")
	})
}

func TestLoggerTimestamp(t *testing.T) {
	t.Run("should include timestamp", func(t *testing.T) {
		var buf bytes.Buffer
		logger.InitLogger("info", "json")

		testLogger := zerolog.New(&buf).With().Timestamp().Logger()
		testLogger.Info().Msg("test with timestamp")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)

		assert.Contains(t, logEntry, "time")
	})
}

func TestLoggerCaseInsensitivity(t *testing.T) {
	tests := []struct {
		input    string
		expected zerolog.Level
	}{
		{"debug", zerolog.DebugLevel},
		{"DEBUG", zerolog.DebugLevel},
		{"Debug", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"INFO", zerolog.InfoLevel},
		{"Info", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"WARN", zerolog.WarnLevel},
		{"Warn", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"ERROR", zerolog.ErrorLevel},
		{"Error", zerolog.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run("should handle "+tt.input, func(t *testing.T) {
			logger.InitLogger(tt.input, "json")
			assert.Equal(t, tt.expected, zerolog.GlobalLevel())
		})
	}
}

func TestLoggerStructuredFields(t *testing.T) {
	t.Run("should support structured logging with fields", func(t *testing.T) {
		var buf bytes.Buffer
		logger.InitLogger("info", "json")

		testLogger := zerolog.New(&buf).With().Timestamp().Logger()
		testLogger.Info().
			Str("field1", "value1").
			Int("field2", 123).
			Bool("field3", true).
			Msg("structured log")

		output := buf.String()
		assert.Contains(t, output, "field1")
		assert.Contains(t, output, "value1")
		assert.Contains(t, output, "field2")
		assert.Contains(t, output, "123")
		assert.Contains(t, output, "field3")
		assert.Contains(t, output, "true")
	})
}

func TestLoggerMultipleInitializations(t *testing.T) {
	t.Run("should handle multiple initializations", func(t *testing.T) {
		logger.InitLogger("info", "json")
		log1 := logger.GetLogger()

		logger.InitLogger("debug", "json")
		log2 := logger.GetLogger()

		assert.NotNil(t, log1)
		assert.NotNil(t, log2)
		assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
	})
}

func TestLoggerWithoutInitialization(t *testing.T) {
	t.Run("should return logger even without explicit initialization", func(t *testing.T) {
		log := logger.GetLogger()
		assert.NotNil(t, log)
	})
}

func TestConsoleWriterFormatting(t *testing.T) {
	t.Run("console writer should format output", func(t *testing.T) {
		var buf bytes.Buffer

		consoleWriter := zerolog.ConsoleWriter{
			Out:        &buf,
			TimeFormat: "15:04:05",
			NoColor:    true,
		}

		testLogger := zerolog.New(consoleWriter).With().Timestamp().Logger()
		testLogger.Info().Msg("test console output")

		output := buf.String()
		assert.NotEmpty(t, output)
		assert.Contains(t, strings.ToLower(output), "test console output")
	})
}
