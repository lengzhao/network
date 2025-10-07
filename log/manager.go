package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// loggerImpl implements the Logger interface
type loggerImpl struct {
	logger *slog.Logger
	level  LogLevel
}

// Debug logs a debug message
func (l *loggerImpl) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info logs an info message
func (l *loggerImpl) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *loggerImpl) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error logs an error message
func (l *loggerImpl) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// Critical logs a critical message
func (l *loggerImpl) Critical(msg string, args ...any) {
	// Use error level for critical messages as slog doesn't have a critical level
	l.logger.Log(context.Background(), slog.LevelError, msg, args...)
}

// With returns a new logger with additional attributes
func (l *loggerImpl) With(args ...any) Logger {
	return &loggerImpl{
		logger: l.logger.With(args...),
		level:  l.level,
	}
}

// SetLevel sets the logging level
func (l *loggerImpl) SetLevel(level LogLevel) {
	l.level = level
	// Note: slog doesn't allow changing level of existing logger directly
	// This would require recreating the logger
}

// GetLevel returns the current logging level
func (l *loggerImpl) GetLevel() LogLevel {
	return l.level
}

// logManagerImpl implements the LogManager interface
type logManagerImpl struct {
	config     *LogConfig
	globalLevel LogLevel
	logger     *slog.Logger
	mu         sync.RWMutex
}

// NewLogManager creates a new log manager
func NewLogManager(config *LogConfig) LogManager {
	if config == nil {
		config = DefaultLogConfig()
	}
	
	// Create the base logger
	logger := createSlogLogger(config)
	
	return &logManagerImpl{
		config:     config,
		globalLevel: config.Level,
		logger:     logger,
	}
}

// With returns a new logger with additional attributes
func (lm *logManagerImpl) With(args ...any) Logger {
	return &loggerImpl{
		logger: lm.logger.With(args...),
		level:  lm.globalLevel,
	}
}

// SetGlobalLevel sets the global logging level
func (lm *logManagerImpl) SetGlobalLevel(level LogLevel) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.globalLevel = level
}

// GetGlobalLevel returns the global logging level
func (lm *logManagerImpl) GetGlobalLevel() LogLevel {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.globalLevel
}

// Configure reconfigures the log system
func (lm *logManagerImpl) Configure(config *LogConfig) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	lm.config = config
	lm.globalLevel = config.Level
	
	// Recreate the base logger
	lm.logger = createSlogLogger(config)
	
	return nil
}

// Run starts the log manager
func (lm *logManagerImpl) Run(ctx context.Context) error {
	// The log manager doesn't need to run a separate goroutine
	// but we implement this method to satisfy the interface
	<-ctx.Done()
	return ctx.Err()
}

// createSlogLogger creates a slog.Logger based on the configuration
func createSlogLogger(config *LogConfig) *slog.Logger {
	if !config.Enabled {
		// Return a logger that discards all output
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	
	// Determine output writer
	var writer io.Writer = os.Stdout
	if config.OutputFile != "" {
		file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			// Fall back to stdout if file cannot be opened
			fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v\n", config.OutputFile, err)
			writer = os.Stdout
		} else {
			writer = file
		}
	}
	
	// Determine handler options
	opts := &slog.HandlerOptions{
		Level: slog.Level(config.Level),
	}
	
	// Create handler based on format
	var handler slog.Handler
	switch config.Format {
	case JSON:
		handler = slog.NewJSONHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}
	
	return slog.New(handler)
}