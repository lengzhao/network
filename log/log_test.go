package log

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLogManagerCreation(t *testing.T) {
	// Test with default config
	logManager := NewLogManager(nil)
	if logManager == nil {
		t.Error("Expected log manager to be created")
	}

	// Test with custom config
	config := &LogConfig{
		Level:      INFO,
		OutputFile: "",
		Format:     TEXT,
		Modules:    make(map[string]ModuleConfig),
		Enabled:    true,
	}

	logManager = NewLogManager(config)
	if logManager == nil {
		t.Error("Expected log manager to be created with custom config")
	}
}

func TestLoggerCreation(t *testing.T) {
	logManager := NewLogManager(nil)

	// Test creating a logger with module attribute using With method
	logger := logManager.With("module", "test")
	if logger == nil {
		t.Error("Expected logger to be created")
	}

	// Test creating another logger with different module
	logger2 := logManager.With("module", "test2")
	if logger2 == nil {
		t.Error("Expected second logger to be created")
	}
}

func TestLoggerWithMethod(t *testing.T) {
	logManager := NewLogManager(nil)

	// Test creating a logger with module attribute using With method
	logger := logManager.With("module", "test")
	if logger == nil {
		t.Error("Expected logger to be created with With method")
	}

	// Test chaining With method
	logger2 := logger.With("component", "handler")
	if logger2 == nil {
		t.Error("Expected logger to be created with chained With method")
	}
}

func TestLogLevel(t *testing.T) {
	logManager := NewLogManager(nil)
	logger := logManager.With("module", "test")

	// Test default level
	if logger.GetLevel() != INFO {
		t.Errorf("Expected default level to be INFO, got %v", logger.GetLevel())
	}

	// Test setting level
	logger.SetLevel(DEBUG)
	// Note: We can't directly verify the level change in the underlying slog logger
	// but we can check our stored level
	if logger.GetLevel() != DEBUG {
		t.Errorf("Expected level to be DEBUG, got %v", logger.GetLevel())
	}
}

func TestGlobalLogLevel(t *testing.T) {
	logManager := NewLogManager(nil)

	// Test default global level
	if logManager.GetGlobalLevel() != INFO {
		t.Errorf("Expected default global level to be INFO, got %v", logManager.GetGlobalLevel())
	}

	// Test setting global level
	logManager.SetGlobalLevel(WARN)
	if logManager.GetGlobalLevel() != WARN {
		t.Errorf("Expected global level to be WARN, got %v", logManager.GetGlobalLevel())
	}
}

func TestLoggingOutput(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a logger that writes to our buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := &loggerImpl{
		logger: slog.New(handler),
		level:  DEBUG,
	}

	// Test logging
	logger.Info("test message", "key", "value")

	// Check that output contains expected content
	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Expected output to contain 'key=value', got: %s", output)
	}
}

func TestModuleLoggerWithConfig(t *testing.T) {
	// Create config with module-specific settings
	config := &LogConfig{
		Level:      INFO,
		OutputFile: "",
		Format:     TEXT,
		Modules: map[string]ModuleConfig{
			"test": {
				Level:  DEBUG,
				Prefix: "TEST",
			},
		},
		Enabled: true,
	}

	logManager := NewLogManager(config)
	// Note: Module-specific configuration is not used with the new approach
	// but we can still test that the logger is created correctly
	logger := logManager.With("module", "test")

	if logger == nil {
		t.Error("Expected logger to be created")
	}
}

func TestLogManagerConfigure(t *testing.T) {
	logManager := NewLogManager(nil)

	// Test reconfiguration
	newConfig := &LogConfig{
		Level:      WARN,
		OutputFile: "",
		Format:     JSON,
		Modules: map[string]ModuleConfig{
			"test": {
				Level:  ERROR,
				Prefix: "TEST",
			},
		},
		Enabled: true,
	}

	err := logManager.Configure(newConfig)
	if err != nil {
		t.Errorf("Expected Configure to succeed, got error: %v", err)
	}

	// Check that global level was updated
	if logManager.GetGlobalLevel() != WARN {
		t.Errorf("Expected global level to be WARN after reconfiguration, got %v", logManager.GetGlobalLevel())
	}
}

func TestLoggerWithMethodChaining(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a logger that writes to our buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	baseLogger := &loggerImpl{
		logger: slog.New(handler),
		level:  DEBUG,
	}

	// Test With method
	logger := baseLogger.With("component", "test")
	if logger == nil {
		t.Error("Expected logger to be created with With method")
	}

	// Test logging with the new logger
	logger.Info("test message with component")

	// Check that output contains expected content
	output := buf.String()
	if !strings.Contains(output, "test message with component") {
		t.Errorf("Expected output to contain 'test message with component', got: %s", output)
	}
	if !strings.Contains(output, "component=test") {
		t.Errorf("Expected output to contain 'component=test', got: %s", output)
	}
}

func TestDisabledLogging(t *testing.T) {
	// Create config with logging disabled
	config := &LogConfig{
		Level:      INFO,
		OutputFile: "",
		Format:     TEXT,
		Modules:    make(map[string]ModuleConfig),
		Enabled:    false,
	}

	logManager := NewLogManager(config)
	logger := logManager.With("module", "test")

	// Test that logging doesn't panic even when disabled
	logger.Info("test message")
	logger.Error("error message")
	logger.Debug("debug message")

	// All calls should succeed without panic
}

func TestDifferentLogLevels(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a logger that writes to our buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := &loggerImpl{
		logger: slog.New(handler),
		level:  DEBUG,
	}

	// Test different log levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
	logger.Critical("critical message")

	// Check that all messages were logged
	output := buf.String()

	// Count occurrences of each message
	debugCount := strings.Count(output, "debug message")
	infoCount := strings.Count(output, "info message")
	warnCount := strings.Count(output, "warn message")
	errorCount := strings.Count(output, "error message")
	criticalCount := strings.Count(output, "critical message")

	if debugCount != 1 {
		t.Errorf("Expected 1 debug message, got %d", debugCount)
	}
	if infoCount != 1 {
		t.Errorf("Expected 1 info message, got %d", infoCount)
	}
	if warnCount != 1 {
		t.Errorf("Expected 1 warn message, got %d", warnCount)
	}
	if errorCount != 1 {
		t.Errorf("Expected 1 error message, got %d", errorCount)
	}
	if criticalCount != 1 {
		t.Errorf("Expected 1 critical message, got %d", criticalCount)
	}
}
