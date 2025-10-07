package log

// Logger is the interface for logging operations
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, args ...any)

	// Info logs an info message
	Info(msg string, args ...any)

	// Warn logs a warning message
	Warn(msg string, args ...any)

	// Error logs an error message
	Error(msg string, args ...any)

	// Critical logs a critical message
	Critical(msg string, args ...any)

	// With returns a new logger with additional attributes
	With(args ...any) Logger

	// SetLevel sets the logging level
	SetLevel(level LogLevel)

	// GetLevel returns the current logging level
	GetLevel() LogLevel
}

// LogManager is the interface for log management operations
type LogManager interface {
	// With returns a new logger with additional attributes
	With(args ...any) Logger

	// SetGlobalLevel sets the global logging level
	SetGlobalLevel(level LogLevel)

	// GetGlobalLevel returns the global logging level
	GetGlobalLevel() LogLevel

	// Configure reconfigures the log system
	Configure(config *LogConfig) error
}
